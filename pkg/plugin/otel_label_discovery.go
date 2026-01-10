package plugin

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type OTelLabelFormat struct {
	ServiceNameLabel       string 
	EnvironmentNameLabel   string 
	Discovered             bool   
	DatasourceUID          string
	DatasourceType         DatasourceType
}

type OTelLabelRegistry struct {
	formats map[string]*OTelLabelFormat 
}

func NewOTelLabelRegistry() *OTelLabelRegistry {
	return &OTelLabelRegistry{
		formats: make(map[string]*OTelLabelFormat),
	}
}

func (r *OTelLabelRegistry) GetFormat(datasourceUID string) *OTelLabelFormat {
	if format, ok := r.formats[datasourceUID]; ok {
		return format
	}
	return nil
}

func (r *OTelLabelRegistry) SetFormat(datasourceUID string, format *OTelLabelFormat) {
	r.formats[datasourceUID] = format
}

func (a *App) DiscoverOTelLabels(ctx context.Context, datasourceUID string, dsType DatasourceType) *OTelLabelFormat {
	format := &OTelLabelFormat{
		DatasourceUID:  datasourceUID,
		DatasourceType: dsType,
		Discovered:     false,
	}

	if a.contextManager != nil {
		obsCtx := a.contextManager.GetContext()

		switch dsType {
		case DatasourcePrometheus:
			format.ServiceNameLabel, format.EnvironmentNameLabel = discoverPromQLLabels(obsCtx.Metrics.Labels)
			if format.ServiceNameLabel != "" {
				format.Discovered = true
			}

		case DatasourceLoki:
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

func discoverPromQLLabels(availableLabels []string) (serviceLabel, environmentLabel string) {
	serviceVariations := []string{
		"service_name",      
		"service.name",      
		"service",           
		"app",               
		"job",               
	}

	environmentVariations := []string{
		"deployment_environment_name", 
		"deployment.environment.name", 
		"environment",                  
		"env",                          
		"namespace",                    
		"cluster",                      
	}

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

func discoverTraceQLLabels(ctx context.Context, app *App, datasourceUID string) (serviceLabel, environmentLabel string) {


	if app.contextManager != nil {
		obsCtx := app.contextManager.GetContext()

		if len(obsCtx.Traces.Services) > 0 {

			serviceVariations := []string{
				"span.service.name",      
				"resource.service.name",  
				"service.name",           
				"service",                
			}

			environmentVariations := []string{
				"deployment.environment.name",     
				"deployment_environment_name",     
				"environment",                      
				"env",                              
			}

			serviceLabel = serviceVariations[0]
			environmentLabel = environmentVariations[0]
		}
	}

	return serviceLabel, environmentLabel
}

func getDefaultLabelNames(dsType DatasourceType) (serviceLabel, environmentLabel string) {
	switch dsType {
	case DatasourcePrometheus, DatasourceLoki:
		return "service_name", "deployment_environment_name"

	case DatasourceTempo:
		return "span.service.name", "deployment.environment.name"

	default:
		return "service_name", "environment"
	}
}

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
