package context

import (
	"context"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// fetchTracesContext fetches traces metadata from Tempo datasources
func (m *Manager) fetchTracesContext(ctx context.Context, datasourceUIDs []string) (*TracesContext, error) {
	tracesCtx := &TracesContext{
		Services:   []string{},
		Operations: []string{},
		SpanNames:  []string{},
	}

	serviceSet := make(map[string]bool)
	operationSet := make(map[string]bool)
	spanSet := make(map[string]bool)

	for _, dsUID := range datasourceUIDs {
		// Get datasource info
		ds, err := m.client.GetDatasource(ctx, dsUID)
		if err != nil {
			backend.Logger.Debug("Failed to get datasource", "uid", dsUID, "error", err)
			continue
		}

		// Only process Tempo/tracing datasources
		dsType := strings.ToLower(ds.Type)
		if !strings.Contains(dsType, "tempo") && !strings.Contains(dsType, "jaeger") && !strings.Contains(dsType, "zipkin") {
			continue
		}

		backend.Logger.Debug("Fetching traces from datasource", "name", ds.Name, "type", ds.Type)

		// For Tempo, we can use TraceQL or search API
		// Try to get service names using a simple TraceQL query
		serviceQuery := `{}`
		resp, err := m.client.QueryDatasource(ctx, dsUID, serviceQuery, "tempo")
		if err != nil {
			backend.Logger.Warn("Failed to query traces", "datasource", ds.Name, "error", err)
			continue
		}

		// Parse response for service names and spans
		for _, result := range resp.Results {
			if result.Error != "" {
				backend.Logger.Warn("Query error", "error", result.Error)
				continue
			}

			for _, frame := range result.Frames {
				// Extract service names and operations from frame
				for _, field := range frame.Schema.Fields {
					if field.Labels != nil {
						// Look for common trace labels
						if serviceName, ok := field.Labels["service.name"]; ok {
							serviceSet[serviceName] = true
						}
						if service, ok := field.Labels["service"]; ok {
							serviceSet[service] = true
						}
						if operation, ok := field.Labels["operation"]; ok {
							operationSet[operation] = true
						}
						if name, ok := field.Labels["name"]; ok {
							operationSet[name] = true
						}
						if spanName, ok := field.Labels["span.name"]; ok {
							spanSet[spanName] = true
						}
					}

					// Also check field name for service/operation info
					fieldName := field.Name
					if strings.Contains(strings.ToLower(fieldName), "service") {
						serviceSet[fieldName] = true
					}
					if strings.Contains(strings.ToLower(fieldName), "operation") {
						operationSet[fieldName] = true
					}
				}
			}
		}

		// If we didn't get much data, try alternative approach
		if len(serviceSet) == 0 {
			m.fetchTracesAlternative(ctx, dsUID, ds.Name, serviceSet, operationSet)
		}
	}

	// Convert sets to slices
	for service := range serviceSet {
		// Filter out empty or invalid entries
		if service != "" && len(service) < 100 {
			tracesCtx.Services = append(tracesCtx.Services, service)
		}
	}
	for operation := range operationSet {
		if operation != "" && len(operation) < 100 {
			tracesCtx.Operations = append(tracesCtx.Operations, operation)
		}
	}
	for span := range spanSet {
		if span != "" && len(span) < 100 {
			tracesCtx.SpanNames = append(tracesCtx.SpanNames, span)
		}
	}

	// Sort for consistency
	sort.Strings(tracesCtx.Services)
	sort.Strings(tracesCtx.Operations)
	sort.Strings(tracesCtx.SpanNames)

	// Limit to avoid overwhelming the LLM
	if len(tracesCtx.Services) > 100 {
		tracesCtx.Services = tracesCtx.Services[:100]
	}
	if len(tracesCtx.Operations) > 100 {
		tracesCtx.Operations = tracesCtx.Operations[:100]
	}
	if len(tracesCtx.SpanNames) > 100 {
		tracesCtx.SpanNames = tracesCtx.SpanNames[:100]
	}

	tracesCtx.SampleCount = len(tracesCtx.Services)

	backend.Logger.Info("Traces context fetched",
		"services", len(tracesCtx.Services),
		"operations", len(tracesCtx.Operations),
		"spans", len(tracesCtx.SpanNames),
	)

	return tracesCtx, nil
}

// fetchTracesAlternative tries alternative methods to fetch trace metadata
func (m *Manager) fetchTracesAlternative(ctx context.Context, dsUID, dsName string, serviceSet, operationSet map[string]bool) {
	// For Tempo, we might need to use the search API or tags API
	// This is a placeholder for future implementation when we have better API access

	// Common service names to look for (in case we can't query)
	// This is just a fallback and won't be accurate
	backend.Logger.Debug("Using alternative trace fetch method", "datasource", dsName)

	// We could potentially make HTTP requests to Tempo's API endpoints directly
	// For now, we'll log that we couldn't get trace data
	backend.Logger.Info("No trace data available from datasource", "datasource", dsName)
}
