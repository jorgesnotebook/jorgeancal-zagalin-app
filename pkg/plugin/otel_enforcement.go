package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type OtelScope struct {
	ServiceName     string
	EnvironmentName string
	Source          string 
}

type DatasourceType string

const (
	DatasourcePrometheus DatasourceType = "prometheus"
	DatasourceLoki       DatasourceType = "loki"
	DatasourceTempo      DatasourceType = "tempo"
	DatasourceOther      DatasourceType = "other"
)

func (a *App) extractOtelScopeFromQuery(queryReq QueryRequest, dsType DatasourceType) *OtelScope {
	scope := &OtelScope{Source: "missing"}

	for _, query := range queryReq.Queries {
		exprStr := query.Expr
		if exprStr == "" {
			exprStr = query.Query
		}
		if exprStr == "" {
			continue
		}

		switch dsType {
		case DatasourcePrometheus, DatasourceLoki:
			if strings.Contains(exprStr, "service_name=") || strings.Contains(exprStr, "service.name=") {
				scope.ServiceName = extractLabelValue(exprStr, []string{"service_name", "service.name"})
			}
			if strings.Contains(exprStr, "deployment_environment_name=") || strings.Contains(exprStr, "deployment.environment.name=") {
				scope.EnvironmentName = extractLabelValue(exprStr, []string{"deployment_environment_name", "deployment.environment.name"})
			}

		case DatasourceTempo:
			if strings.Contains(exprStr, "service.name=") || strings.Contains(exprStr, "span.service.name=") || strings.Contains(exprStr, "resource.service.name=") {
				scope.ServiceName = extractLabelValue(exprStr, []string{"span.service.name", "resource.service.name", "service.name"})
			}
			if strings.Contains(exprStr, "deployment.environment.name=") {
				scope.EnvironmentName = extractLabelValue(exprStr, []string{"deployment.environment.name"})
			}

		case DatasourceOther:
			if strings.Contains(exprStr, "service") && (strings.Contains(exprStr, "name") || strings.Contains(exprStr, "=")) {
				scope.ServiceName = "detected"
			}
			if strings.Contains(exprStr, "environment") || strings.Contains(exprStr, "env") {
				scope.EnvironmentName = "detected"
			}
		}
	}

	if scope.ServiceName != "" || scope.EnvironmentName != "" {
		scope.Source = "context"
	}

	return scope
}

func extractLabelValue(expr string, labelNames []string) string {
	for _, labelName := range labelNames {
		patterns := []string{
			labelName + `="`,
			labelName + `=~"`,
		}

		for _, pattern := range patterns {
			idx := strings.Index(expr, pattern)
			if idx != -1 {
				start := idx + len(pattern)
				end := strings.Index(expr[start:], `"`)
				if end != -1 {
					return expr[start : start+end]
				}
			}
		}
	}
	return ""
}

func (a *App) applyOtelScopeDefaults(scope *OtelScope) {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return
	}

	if scope.ServiceName == "" && a.settings.OtelEnforcement.DefaultServiceName != "" {
		scope.ServiceName = a.settings.OtelEnforcement.DefaultServiceName
		scope.Source = "defaults"
		backend.Logger.Debug("Applied default service name", "serviceName", scope.ServiceName)
	}

	if scope.EnvironmentName == "" && a.settings.OtelEnforcement.DefaultEnvironmentName != "" {
		scope.EnvironmentName = a.settings.OtelEnforcement.DefaultEnvironmentName
		scope.Source = "defaults"
		backend.Logger.Debug("Applied default environment name", "environmentName", scope.EnvironmentName)
	}
}

func (a *App) validateOtelScope(scope *OtelScope) error {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return nil 
	}

	var missing []string

	if a.settings.OtelEnforcement.RequireServiceName && scope.ServiceName == "" {
		missing = append(missing, "service.name")
	}

	if a.settings.OtelEnforcement.RequireEnvironmentName && scope.EnvironmentName == "" {
		missing = append(missing, "deployment.environment.name")
	}

	if len(missing) > 0 {
		if a.settings.OtelEnforcement.RejectIfNoScope {
			return fmt.Errorf("query rejected: missing required OTel attributes: %s", strings.Join(missing, ", "))
		}
		backend.Logger.Warn("Query missing required OTel attributes but rejection disabled",
			"missing", missing,
			"scope", scope,
		)
	}

	return nil
}

func (a *App) injectOtelScope(queryReq *QueryRequest, scope *OtelScope, dsType DatasourceType) error {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return nil
	}

	if scope.ServiceName == "" && scope.EnvironmentName == "" {
		return nil 
	}

	if dsType == DatasourceOther {
		backend.Logger.Debug("Skipping injection for non-standard datasource", "datasourceType", dsType)
		return nil
	}

	for i := range queryReq.Queries {
		exprStr := queryReq.Queries[i].Expr
		if exprStr == "" {
			exprStr = queryReq.Queries[i].Query
		}
		if exprStr == "" {
			continue
		}

		var injectedExpr string

		labelFormat := a.otelRegistry.GetFormat(queryReq.Datasource)
		if labelFormat == nil {
			labelFormat = a.DiscoverOTelLabels(context.Background(), queryReq.Datasource, dsType)
			a.otelRegistry.SetFormat(queryReq.Datasource, labelFormat)

			backend.Logger.Info("Discovered OTel label format",
				"datasource", queryReq.Datasource,
				"type", dsType,
				"serviceLabel", labelFormat.ServiceNameLabel,
				"environmentLabel", labelFormat.EnvironmentNameLabel,
				"discovered", labelFormat.Discovered,
			)
		}

		switch dsType {
		case DatasourcePrometheus, DatasourceLoki:
			labels := BuildOTelLabels(labelFormat, scope.ServiceName, scope.EnvironmentName)
			injectedExpr = injectLabelsIntoQuery(exprStr, labels)

		case DatasourceTempo:
			spanSelectors := BuildOTelLabels(labelFormat, scope.ServiceName, scope.EnvironmentName)
			injectedExpr = injectTraceQLSelectors(exprStr, spanSelectors)

		default:
			continue
		}

		if queryReq.Queries[i].Expr != "" {
			queryReq.Queries[i].Expr = injectedExpr
		} else {
			queryReq.Queries[i].Query = injectedExpr
		}

		backend.Logger.Debug("Injected OTel scope into query",
			"datasourceType", dsType,
			"originalExpr", exprStr,
			"injectedExpr", injectedExpr,
			"scope", scope,
		)
	}

	return nil
}

func injectLabelsIntoQuery(expr string, labels []string) string {
	if len(labels) == 0 {
		return expr
	}

	labelsStr := strings.Join(labels, ",")


	openBrace := strings.Index(expr, "{")
	if openBrace == -1 {
		end := len(expr)
		for i, ch := range expr {
			if ch == ' ' || ch == '[' || ch == '(' {
				end = i
				break
			}
		}
		return expr[:end] + "{" + labelsStr + "}" + expr[end:]
	}

	closeBrace := strings.Index(expr[openBrace:], "}")
	if closeBrace == -1 {
		return expr 
	}
	closeBrace += openBrace

	existingLabels := strings.TrimSpace(expr[openBrace+1 : closeBrace])
	if existingLabels == "" {
		return expr[:openBrace+1] + labelsStr + expr[closeBrace:]
	}

	return expr[:closeBrace] + "," + labelsStr + expr[closeBrace:]
}

func injectTraceQLSelectors(expr string, selectors []string) string {
	if len(selectors) == 0 {
		return expr
	}

	selectorsStr := strings.Join(selectors, " && ")

	openBrace := strings.Index(expr, "{")
	if openBrace == -1 {
		return "{" + selectorsStr + "}"
	}

	closeBrace := strings.Index(expr[openBrace:], "}")
	if closeBrace == -1 {
		return expr 
	}
	closeBrace += openBrace

	existingSelectors := strings.TrimSpace(expr[openBrace+1 : closeBrace])
	if existingSelectors == "" {
		return expr[:openBrace+1] + selectorsStr + expr[closeBrace:]
	}

	return expr[:closeBrace] + " && " + selectorsStr + expr[closeBrace:]
}

func (a *App) logOtelScopeUsage(user *UserIdentity, scope *OtelScope, datasource string) {
	if scope.Source == "defaults" {
		backend.Logger.Info("OTel scope fallback applied",
			"user", user.UserLogin,
			"userId", user.UserID,
			"orgId", user.OrgID,
			"datasource", datasource,
			"serviceName", scope.ServiceName,
			"environmentName", scope.EnvironmentName,
			"source", scope.Source,
		)
	} else if scope.Source == "context" {
		backend.Logger.Debug("OTel scope from context",
			"user", user.UserLogin,
			"serviceName", scope.ServiceName,
			"environmentName", scope.EnvironmentName,
		)
	}
}

func (s *OtelScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"serviceName":     s.ServiceName,
		"environmentName": s.EnvironmentName,
		"source":          s.Source,
	})
}
