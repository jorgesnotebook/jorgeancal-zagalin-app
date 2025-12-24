package context

import (
	"context"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// fetchMetricsContext fetches metrics metadata from Prometheus datasources
func (m *Manager) fetchMetricsContext(ctx context.Context, datasourceUIDs []string) (*MetricsContext, error) {
	metricsCtx := &MetricsContext{
		MetricNames: []string{},
		Labels:      []string{},
		LabelValues: make(map[string][]string),
	}

	labelSet := make(map[string]bool)
	metricSet := make(map[string]bool)
	labelValuesMap := make(map[string]map[string]bool)

	for _, dsUID := range datasourceUIDs {
		// Get datasource info
		ds, err := m.client.GetDatasource(ctx, dsUID)
		if err != nil {
			backend.Logger.Debug("Failed to get datasource", "uid", dsUID, "error", err)
			continue
		}

		// Only process Prometheus datasources
		if !strings.Contains(strings.ToLower(ds.Type), "prometheus") {
			continue
		}

		backend.Logger.Debug("Fetching metrics from datasource", "name", ds.Name, "type", ds.Type)

		// Query for metric names using label_values() - gets all metric names
		metricNamesQuery := `label_values(__name__)`
		resp, err := m.client.QueryDatasource(ctx, dsUID, metricNamesQuery, "prometheus")
		if err != nil {
			backend.Logger.Warn("Failed to query metric names", "datasource", ds.Name, "error", err)
			continue
		}

		// Parse response for metric names
		for _, result := range resp.Results {
			if result.Error != "" {
				backend.Logger.Warn("Query error", "error", result.Error)
				continue
			}

			for _, frame := range result.Frames {
				// Extract metric names from frame
				names := extractValuesFromFrame(frame)
				for _, name := range names {
					metricSet[name] = true
				}
			}
		}

		// Query for label names
		labelNamesQuery := `label_names()`
		resp, err = m.client.QueryDatasource(ctx, dsUID, labelNamesQuery, "prometheus")
		if err != nil {
			backend.Logger.Warn("Failed to query label names", "datasource", ds.Name, "error", err)
			continue
		}

		// Parse response for label names
		for _, result := range resp.Results {
			if result.Error != "" {
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
		commonLabels := []string{"job", "instance", "namespace", "pod", "container", "service", "app", "env", "cluster"}
		for _, label := range commonLabels {
			if _, exists := labelSet[label]; !exists {
				continue
			}

			// Query for label values
			labelValuesQuery := `label_values(` + label + `)`
			resp, err := m.client.QueryDatasource(ctx, dsUID, labelValuesQuery, "prometheus")
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
	}

	// Convert sets to slices
	for metric := range metricSet {
		metricsCtx.MetricNames = append(metricsCtx.MetricNames, metric)
	}
	for label := range labelSet {
		metricsCtx.Labels = append(metricsCtx.Labels, label)
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
		metricsCtx.LabelValues[label] = values
	}

	// Sort for consistency
	sort.Strings(metricsCtx.MetricNames)
	sort.Strings(metricsCtx.Labels)

	// Limit metrics to avoid overwhelming the LLM
	if len(metricsCtx.MetricNames) > 200 {
		metricsCtx.MetricNames = metricsCtx.MetricNames[:200]
	}

	metricsCtx.SampleCount = len(metricsCtx.MetricNames)

	backend.Logger.Info("Metrics context fetched",
		"metrics", len(metricsCtx.MetricNames),
		"labels", len(metricsCtx.Labels),
	)

	return metricsCtx, nil
}

// extractValuesFromFrame extracts string values from a data frame
func extractValuesFromFrame(frame Frame) []string {
	values := []string{}

	// Try to extract from schema fields
	for _, field := range frame.Schema.Fields {
		if field.Name != "" && field.Name != "Time" {
			values = append(values, field.Name)
		}
	}

	// Note: Full frame data parsing would require more complex JSON unmarshaling
	// For now, we rely on field names which often contain the values we need

	return values
}
