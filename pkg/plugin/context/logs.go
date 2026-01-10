package context

import (
	"context"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

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
		ds, err := m.client.GetDatasource(ctx, dsUID)
		if err != nil {
			backend.Logger.Debug("Failed to get datasource", "uid", dsUID, "error", err)
			continue
		}

		if !strings.Contains(strings.ToLower(ds.Type), "loki") {
			continue
		}

		backend.Logger.Debug("Fetching logs from datasource", "name", ds.Name, "type", ds.Type)

		resp, err := m.client.QueryDatasource(ctx, dsUID, `label_names()`, "loki")
		if err != nil {
			backend.Logger.Warn("Failed to query label names", "datasource", ds.Name, "error", err)
			m.fetchLogStreamsAlternative(ctx, dsUID, ds.Name, streamSet, labelSet)
			continue
		}

		for _, result := range resp.Results {
			if result.Error != "" {
				backend.Logger.Warn("Query error", "error", result.Error)
				continue
			}

			for _, frame := range result.Frames {
				labels := extractValuesFromFrame(frame)
				for _, label := range labels {
					if strings.HasPrefix(label, "__") {
						continue
					}
					labelSet[label] = true
				}
			}
		}

		commonLabels := []string{"app", "namespace", "pod", "container", "job", "filename", "level", "env", "cluster", "host"}
		for _, label := range commonLabels {
			if _, exists := labelSet[label]; !exists {
				continue
			}

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

		if len(labelSet) > 0 {
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
		if len(values) > 10 {
			sort.Strings(values)
			values = values[:10]
		}
		logsCtx.LabelValues[label] = values
	}

	sort.Strings(logsCtx.Streams)
	sort.Strings(logsCtx.Labels)

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

func (m *Manager) fetchLogStreamsAlternative(ctx context.Context, dsUID, dsName string, streamSet, labelSet map[string]bool) {
	commonSelectors := []string{
		`{app=~".+"}`,
		`{namespace=~".+"}`,
		`{job=~".+"}`,
		`{}`, 
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

		if len(streamSet) > 0 {
			break
		}
	}
}
