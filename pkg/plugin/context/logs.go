package context

import (
	"context"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// fetchLogsContext fetches logs metadata from Loki datasources
func (m *Manager) fetchLogsContext(ctx context.Context, datasourceUIDs []string) (*LogsContext, error) {
	logsCtx := &LogsContext{
		Streams:     []string{},
		Labels:      []string{},
		LabelValues: make(map[string][]string),
	}

	streamSet := make(map[string]bool)
	labelSet := make(map[string]bool)
	labelValuesMap := make(map[string]map[string]bool)

	for _, dsUID := range datasourceUIDs {
		// Get datasource info
		ds, err := m.client.GetDatasource(ctx, dsUID)
		if err != nil {
			backend.Logger.Debug("Failed to get datasource", "uid", dsUID, "error", err)
			continue
		}

		// Only process Loki datasources
		if !strings.Contains(strings.ToLower(ds.Type), "loki") {
			continue
		}

		backend.Logger.Debug("Fetching logs from datasource", "name", ds.Name, "type", ds.Type)

		// Query for label names
		// Loki supports label_names() similar to Prometheus
		resp, err := m.client.QueryDatasource(ctx, dsUID, `label_names()`, "loki")
		if err != nil {
			backend.Logger.Warn("Failed to query label names", "datasource", ds.Name, "error", err)
			// Try alternative queries for log streams
			m.fetchLogStreamsAlternative(ctx, dsUID, ds.Name, streamSet, labelSet)
			continue
		}

		// Parse response for label names
		for _, result := range resp.Results {
			if result.Error != "" {
				backend.Logger.Warn("Query error", "error", result.Error)
				continue
			}

			for _, frame := range result.Frames {
				labels := extractValuesFromFrame(frame)
				for _, label := range labels {
					// Skip internal labels
					if strings.HasPrefix(label, "__") {
						continue
					}
					labelSet[label] = true
				}
			}
		}

		// For common labels, get sample values
		commonLabels := []string{"app", "namespace", "pod", "container", "job", "filename", "level", "env", "cluster", "host"}
		for _, label := range commonLabels {
			if _, exists := labelSet[label]; !exists {
				continue
			}

			// Query for label values using label_values
			query := `label_values(` + label + `)`
			resp, err := m.client.QueryDatasource(ctx, dsUID, query, "loki")
			if err != nil {
				continue
			}

			for _, result := range resp.Results {
				if result.Error != "" {
					continue
				}

				for _, frame := range result.Frames {
					values := extractValuesFromFrame(frame)
					if _, exists := labelValuesMap[label]; !exists {
						labelValuesMap[label] = make(map[string]bool)
					}
					for _, val := range values {
						labelValuesMap[label][val] = true
					}
				}
			}
		}

		// Build sample log streams from label combinations
		if len(labelSet) > 0 {
			// Create some sample stream selectors
			for label := range labelSet {
				if values, ok := labelValuesMap[label]; ok {
					for val := range values {
						stream := `{` + label + `="` + val + `"}`
						streamSet[stream] = true
						if len(streamSet) >= 50 {
							break
						}
					}
				}
				if len(streamSet) >= 50 {
					break
				}
			}
		}
	}

	// Convert sets to slices
	for stream := range streamSet {
		logsCtx.Streams = append(logsCtx.Streams, stream)
	}
	for label := range labelSet {
		logsCtx.Labels = append(logsCtx.Labels, label)
	}
	for label, valuesSet := range labelValuesMap {
		values := []string{}
		for val := range valuesSet {
			values = append(values, val)
		}
		// Limit to 10 sample values per label
		if len(values) > 10 {
			sort.Strings(values)
			values = values[:10]
		}
		logsCtx.LabelValues[label] = values
	}

	// Sort for consistency
	sort.Strings(logsCtx.Streams)
	sort.Strings(logsCtx.Labels)

	// Limit streams to avoid overwhelming the LLM
	if len(logsCtx.Streams) > 100 {
		logsCtx.Streams = logsCtx.Streams[:100]
	}

	logsCtx.SampleCount = len(logsCtx.Streams)

	backend.Logger.Info("Logs context fetched",
		"streams", len(logsCtx.Streams),
		"labels", len(logsCtx.Labels),
	)

	return logsCtx, nil
}

// fetchLogStreamsAlternative tries alternative methods to fetch log streams
func (m *Manager) fetchLogStreamsAlternative(ctx context.Context, dsUID, dsName string, streamSet, labelSet map[string]bool) {
	// Try to query with a simple log selector
	commonSelectors := []string{
		`{app=~".+"}`,
		`{namespace=~".+"}`,
		`{job=~".+"}`,
		`{}`, // All logs
	}

	for _, selector := range commonSelectors {
		query := selector + ` | json`
		resp, err := m.client.QueryDatasource(ctx, dsUID, query, "loki")
		if err != nil {
			continue
		}

		for _, result := range resp.Results {
			if result.Error != "" {
				continue
			}

			for _, frame := range result.Frames {
				// Extract labels from frame fields
				for _, field := range frame.Schema.Fields {
					if field.Labels != nil {
						for k, v := range field.Labels {
							if !strings.HasPrefix(k, "__") {
								labelSet[k] = true
								stream := `{` + k + `="` + v + `"}`
								streamSet[stream] = true
							}
						}
					}
				}
			}
		}

		// If we got some results, don't try more queries
		if len(streamSet) > 0 {
			break
		}
	}
}
