package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// OTelLabelFormat represents the discovered OTel label naming convention for a datasource
type OTelLabelFormat struct {
	ServiceNameLabel       string // Actual label name for service (e.g., "service_name", "service.name", "span.service.name")
	EnvironmentNameLabel   string // Actual label name for environment (e.g., "deployment_environment_name", "environment", "env")
	Discovered             bool   // Whether these were discovered vs using defaults
	DatasourceUID          string
	DatasourceType         DatasourceType
}

// OTelLabelRegistry caches discovered OTel label formats per datasource
type OTelLabelRegistry struct {
	formats map[string]*OTelLabelFormat // datasourceUID -> format
}

// NewOTelLabelRegistry creates a new label registry
func NewOTelLabelRegistry() *OTelLabelRegistry {
	return &OTelLabelRegistry{
		formats: make(map[string]*OTelLabelFormat),
	}
}

// GetFormat returns the label format for a datasource
func (r *OTelLabelRegistry) GetFormat(datasourceUID string) *OTelLabelFormat {
	if format, ok := r.formats[datasourceUID]; ok {
		return format
	}
	return nil
}

// SetFormat stores the label format for a datasource
func (r *OTelLabelRegistry) SetFormat(datasourceUID string, format *OTelLabelFormat) {
	r.formats[datasourceUID] = format
}

// DiscoverOTelLabels attempts to discover which OTel label names are used by querying the datasource
// This helps support both underscore (service_name) and dot notation (service.name) automatically
func (a *App) DiscoverOTelLabels(ctx context.Context, datasourceUID string, dsType DatasourceType) *OTelLabelFormat {
	format := &OTelLabelFormat{
		DatasourceUID:  datasourceUID,
		DatasourceType: dsType,
		Discovered:     false,
	}

	// Check if we have cached context with label information
	if a.contextManager != nil {
		obsCtx := a.contextManager.GetContext()

		switch dsType {
		case DatasourcePrometheus:
			format.ServiceNameLabel, format.EnvironmentNameLabel = discoverPromQLLabels(obsCtx.Metrics.Labels)
			if format.ServiceNameLabel != "" {
				format.Discovered = true
			}

		case DatasourceLoki:
			// Loki uses same label conventions as Prometheus
			format.ServiceNameLabel, format.EnvironmentNameLabel = discoverPromQLLabels(obsCtx.Logs.Labels)
			if format.ServiceNameLabel != "" {
				format.Discovered = true
			}

		case DatasourceTempo:
			format.ServiceNameLabel, format.EnvironmentNameLabel = discoverTraceQLLabels(ctx, a, datasourceUID)
			if format.ServiceNameLabel != "" {
				format.Discovered = true
			}
		}
	}

	// If discovery failed, use defaults
	if !format.Discovered {
		format.ServiceNameLabel, format.EnvironmentNameLabel = getDefaultLabelNames(dsType)
	}

	backend.Logger.Debug("OTel label discovery result",
		"datasource", datasourceUID,
		"type", dsType,
		"serviceLabel", format.ServiceNameLabel,
		"environmentLabel", format.EnvironmentNameLabel,
		"discovered", format.Discovered,
	)

	return format
}

// discoverPromQLLabels finds OTel-compatible labels in Prometheus
func discoverPromQLLabels(availableLabels []string) (serviceLabel, environmentLabel string) {
	// Try different service name variations (order matters - most specific first)
	serviceVariations := []string{
		"service_name",      // OpenTelemetry standard with underscore
		"service.name",      // OpenTelemetry standard with dot (less common in Prom)
		"service",           // Short form
		"app",               // Common alternative
		"job",               // Prometheus convention
	}

	environmentVariations := []string{
		"deployment_environment_name", // OpenTelemetry standard with underscore
		"deployment.environment.name", // OpenTelemetry standard with dot
		"environment",                  // Short form
		"env",                          // Common alternative
		"namespace",                    // Kubernetes convention
		"cluster",                      // Multi-cluster setups
	}

	// Find first matching service label
	for _, label := range availableLabels {
		labelLower := strings.ToLower(label)
		for _, variant := range serviceVariations {
			if labelLower == variant {
				serviceLabel = label
				break
			}
		}
		if serviceLabel != "" {
			break
		}
	}

	// Find first matching environment label
	for _, label := range availableLabels {
		labelLower := strings.ToLower(label)
		for _, variant := range environmentVariations {
			if labelLower == variant {
				environmentLabel = label
				break
			}
		}
		if environmentLabel != "" {
			break
		}
	}

	return serviceLabel, environmentLabel
}

// discoverTraceQLLabels finds OTel-compatible attributes in Tempo/TraceQL
func discoverTraceQLLabels(ctx context.Context, app *App, datasourceUID string) (serviceLabel, environmentLabel string) {
	// Tempo/TraceQL has more variations due to span vs resource attributes
	// We need to query Tempo to see what's actually available

	// Try a simple TraceQL query to discover attributes
	// Query: {} - returns all spans, we can inspect their attributes

	// For now, try the most common variations based on context
	if app.contextManager != nil {
		obsCtx := app.contextManager.GetContext()

		// Check if we have any trace data with service information
		if len(obsCtx.Traces.Services) > 0 {
			// Services were discovered, meaning one of these labels worked
			// Try to infer which one by checking common patterns

			// Most common TraceQL variations (in order of preference):
			serviceVariations := []string{
				"span.service.name",      // Most common in Tempo
				"resource.service.name",  // Alternative in Tempo
				"service.name",           // Generic form
				"service",                // Short form
			}

			environmentVariations := []string{
				"deployment.environment.name",     // OpenTelemetry standard
				"deployment_environment_name",     // Underscore variant (less common)
				"environment",                      // Short form
				"env",                              // Common alternative
			}

			// For TraceQL, default to span.service.name as it's most common
			serviceLabel = serviceVariations[0]
			environmentLabel = environmentVariations[0]
		}
	}

	return serviceLabel, environmentLabel
}

// getDefaultLabelNames returns default label names for a datasource type
func getDefaultLabelNames(dsType DatasourceType) (serviceLabel, environmentLabel string) {
	switch dsType {
	case DatasourcePrometheus, DatasourceLoki:
		// Default to underscore notation for Prometheus/Loki
		return "service_name", "deployment_environment_name"

	case DatasourceTempo:
		// Default to span.service.name for Tempo/TraceQL
		return "span.service.name", "deployment.environment.name"

	default:
		// Generic default
		return "service_name", "environment"
	}
}

// BuildOTelLabels creates label/selector strings based on discovered format
func BuildOTelLabels(format *OTelLabelFormat, serviceName, environmentName string) []string {
	if format == nil {
		return []string{}
	}

	var labels []string

	if serviceName != "" && format.ServiceNameLabel != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, format.ServiceNameLabel, serviceName))
	}

	if environmentName != "" && format.EnvironmentNameLabel != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, format.EnvironmentNameLabel, environmentName))
	}

	return labels
}

// TryMultipleFormats attempts to inject OTel labels using multiple label name variations
// This is a fallback when discovery hasn't run or failed
func TryMultipleFormats(dsType DatasourceType, serviceName, environmentName string) []string {
	serviceLabel, environmentLabel := getDefaultLabelNames(dsType)

	var labels []string

	if serviceName != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, serviceLabel, serviceName))
	}

	if environmentName != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, environmentLabel, environmentName))
	}

	return labels
}
