package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// OtelScope represents the required OTel attributes for query scoping
type OtelScope struct {
	ServiceName     string
	EnvironmentName string
	Source          string // "context", "defaults", or "missing"
}

// extractOtelScopeFromQuery attempts to extract OTel scope from query labels
func (a *App) extractOtelScopeFromQuery(queryReq QueryRequest) *OtelScope {
	scope := &OtelScope{Source: "missing"}

	// Parse queries to find service.name and deployment.environment.name labels
	for _, query := range queryReq.Queries {
		// Expr is already a string in QueryPayload struct
		exprStr := query.Expr
		if exprStr == "" {
			// Try Query field as fallback
			exprStr = query.Query
		}
		if exprStr == "" {
			continue
		}

		// Simple pattern matching for PromQL/LogQL labels
		// Look for service_name="..." or service.name="..."
		if strings.Contains(exprStr, "service_name=") || strings.Contains(exprStr, "service.name=") {
			scope.ServiceName = extractLabelValue(exprStr, []string{"service_name", "service.name"})
		}

		// Look for deployment_environment_name="..." or deployment.environment.name="..."
		if strings.Contains(exprStr, "deployment_environment_name=") || strings.Contains(exprStr, "deployment.environment.name=") {
			scope.EnvironmentName = extractLabelValue(exprStr, []string{"deployment_environment_name", "deployment.environment.name"})
		}
	}

	if scope.ServiceName != "" || scope.EnvironmentName != "" {
		scope.Source = "context"
	}

	return scope
}

// extractLabelValue extracts the value of a label from a query expression
// Handles both label_name="value" and label_name=~"regex" formats
func extractLabelValue(expr string, labelNames []string) string {
	for _, labelName := range labelNames {
		// Look for label_name="value" or label_name=~"value"
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

// applyOtelScopeDefaults applies default OTel scope if enforcement is enabled
func (a *App) applyOtelScopeDefaults(scope *OtelScope) {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return
	}

	// Apply defaults if values are missing
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

// validateOtelScope validates that required OTel attributes are present
func (a *App) validateOtelScope(scope *OtelScope) error {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return nil // Enforcement disabled
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

// injectOtelScope injects OTel scope labels into queries
// This is a simplified implementation that handles basic PromQL/LogQL queries
func (a *App) injectOtelScope(queryReq *QueryRequest, scope *OtelScope) error {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return nil
	}

	if scope.ServiceName == "" && scope.EnvironmentName == "" {
		return nil // Nothing to inject
	}

	for i := range queryReq.Queries {
		// Expr is already a string in QueryPayload struct
		exprStr := queryReq.Queries[i].Expr
		if exprStr == "" {
			// Try Query field as fallback
			exprStr = queryReq.Queries[i].Query
		}
		if exprStr == "" {
			continue
		}

		// Build label selectors to inject
		var labels []string
		if scope.ServiceName != "" {
			labels = append(labels, fmt.Sprintf(`service_name="%s"`, scope.ServiceName))
		}
		if scope.EnvironmentName != "" {
			labels = append(labels, fmt.Sprintf(`deployment_environment_name="%s"`, scope.EnvironmentName))
		}

		// Inject labels into query
		// This is a simplified implementation - in production, use a proper PromQL/LogQL parser
		injectedExpr := injectLabelsIntoQuery(exprStr, labels)

		// Update the appropriate field
		if queryReq.Queries[i].Expr != "" {
			queryReq.Queries[i].Expr = injectedExpr
		} else {
			queryReq.Queries[i].Query = injectedExpr
		}

		backend.Logger.Debug("Injected OTel scope into query",
			"originalExpr", exprStr,
			"injectedExpr", injectedExpr,
			"scope", scope,
		)
	}

	return nil
}

// injectLabelsIntoQuery injects label selectors into a PromQL/LogQL query
// This is a simplified implementation - in production, use a proper parser
func injectLabelsIntoQuery(expr string, labels []string) string {
	if len(labels) == 0 {
		return expr
	}

	labelsStr := strings.Join(labels, ",")

	// For PromQL: metric_name{existing_labels} -> metric_name{existing_labels,injected_labels}
	// For LogQL: {existing_labels} -> {existing_labels,injected_labels}

	// Find the first { in the query
	openBrace := strings.Index(expr, "{")
	if openBrace == -1 {
		// No labels in query, add them: metric_name -> metric_name{injected_labels}
		// Find where the metric name ends (space, [, or end of string)
		end := len(expr)
		for i, ch := range expr {
			if ch == ' ' || ch == '[' || ch == '(' {
				end = i
				break
			}
		}
		return expr[:end] + "{" + labelsStr + "}" + expr[end:]
	}

	// Find the matching closing brace
	closeBrace := strings.Index(expr[openBrace:], "}")
	if closeBrace == -1 {
		return expr // Malformed query, don't modify
	}
	closeBrace += openBrace

	// Check if there are existing labels
	existingLabels := strings.TrimSpace(expr[openBrace+1 : closeBrace])
	if existingLabels == "" {
		// Empty labels: {} -> {injected_labels}
		return expr[:openBrace+1] + labelsStr + expr[closeBrace:]
	}

	// Existing labels: {label1="value1"} -> {label1="value1",injected_labels}
	return expr[:closeBrace] + "," + labelsStr + expr[closeBrace:]
}

// logOtelScopeUsage logs the OTel scope usage for audit purposes
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

// MarshalJSON implements json.Marshaler for OtelScope (for logging)
func (s *OtelScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"serviceName":     s.ServiceName,
		"environmentName": s.EnvironmentName,
		"source":          s.Source,
	})
}
