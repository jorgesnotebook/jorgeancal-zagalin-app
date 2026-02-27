package plugin

import (
	"fmt"
	"strings"
)

// hasDatasourceType returns true if any cached datasource has a type that
// contains typeKeyword. Returns true when the cache is empty so tools are not
// hidden before the cache has been populated for the first time.
func hasDatasourceType(datasources []DatasourceInfo, typeKeyword string) bool {
	if len(datasources) == 0 {
		return true // cache not yet warm — don't gate
	}
	for _, ds := range datasources {
		if strings.Contains(strings.ToLower(ds.Type), typeKeyword) {
			return true
		}
	}
	return false
}

// buildDatasourceHint returns " — available: name (uid: uid), ..." for datasources
// whose type contains the given type keyword (e.g. "prometheus", "loki", "tempo").
// Returns empty string when the cache is empty or no matching datasource exists.
func buildDatasourceHint(datasources []DatasourceInfo, typeKeyword string) string {
	var parts []string
	for _, ds := range datasources {
		if strings.Contains(strings.ToLower(ds.Type), typeKeyword) {
			parts = append(parts, fmt.Sprintf("%s (uid: %s)", ds.Name, ds.UID))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " — available: " + strings.Join(parts, ", ")
}

type Tool struct {
	Type         string        `json:"type"`
	Function     Function      `json:"function"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ToolParameters struct {
	Type       string                        `json:"type"`
	Properties map[string]PropertyDefinition `json:"properties"`
	Required   []string                      `json:"required,omitempty"`
}

type PropertyDefinition struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

var ZAGALIN_TOOLS = []Tool{
	{
		Type: "function",
		Function: Function{
			Name:        "navigate_to_dashboard",
			Description: "Navigate to a specific Grafana dashboard by UID",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"dashboardUid": {
						Type:        "string",
						Description: "The unique identifier (UID) of the dashboard",
					},
					"panelId": {
						Type:        "number",
						Description: "Optional panel ID to focus on a specific panel",
					},
				},
				Required: []string{"dashboardUid"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "create_promql_query",
			Description: "Generate a PromQL query for Prometheus metrics. CRITICAL: Only use metric names from the datasource metadata provided in the system prompt. If metric is not listed, ask user for clarification.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"metric": {
						Type:        "string",
						Description: "The metric name to query (e.g., http_requests_total)",
					},
					"filters": {
						Type:        "object",
						Description: "Label filters as key-value pairs (e.g., {job: \"api\", status: \"200\"})",
					},
					"aggregation": {
						Type:        "string",
						Description: "Aggregation function to apply",
						Enum:        []string{"sum", "avg", "min", "max", "count", "rate"},
					},
					"timeRange": {
						Type:        "string",
						Description: "Time range for rate calculations (e.g., \"5m\", \"1h\")",
					},
				},
				Required: []string{"metric"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "create_logql_query",
			Description: "Generate a LogQL query for Loki logs. CRITICAL: Only use log stream selectors and labels from the datasource metadata. If stream is not listed, ask user for clarification.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"logStream": {
						Type:        "string",
						Description: "Log stream selector (e.g., {job=\"varlogs\"})",
					},
					"filter": {
						Type:        "string",
						Description: "Log line filter expression",
					},
					"parser": {
						Type:        "string",
						Description: "Parser to use for log lines",
						Enum:        []string{"json", "logfmt", "pattern", "regexp"},
					},
				},
				Required: []string{"logStream"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "create_traceql_query",
			Description: "Generate a TraceQL query for Tempo traces. Use this to search traces by attributes, duration, or status. CRITICAL: Only use service names from the datasource metadata. If service is not listed, ask user for clarification.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"traceSelector": {
						Type:        "string",
						Description: "Trace selector expression (e.g., {service.name=\"api-gateway\"})",
					},
					"filters": {
						Type:        "object",
						Description: "Additional attribute filters (e.g., {status: \"error\", \"http.status_code\": \"500\"})",
					},
					"duration": {
						Type:        "string",
						Description: "Duration filter (e.g., \"> 1s\", \"< 100ms\")",
					},
				},
				Required: []string{"traceSelector"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "get_trace_by_id",
			Description: "Fetch and analyze a specific trace by its trace ID. Use this when the user provides a trace ID. Returns trace structure with spans, services, durations, and errors.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"traceId": {
						Type:        "string",
						Description: "The trace ID to fetch (e.g., \"abc123def456\")",
					},
					"datasource": {
						Type:        "string",
						Description: "Tempo datasource UID or name",
					},
				},
				Required: []string{"traceId", "datasource"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "get_logs",
			Description: "Fetch and analyze logs from a Loki datasource. Use this when the user asks about logs they are viewing or when you need to investigate log patterns, errors, or volume. Returns log analysis with trends, error rates, and notable messages.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"panelId": {
						Type:        "number",
						Description: "Panel ID to analyze (if user is viewing a specific log panel)",
					},
					"query": {
						Type:        "string",
						Description: "LogQL query to execute (e.g., \"{namespace=\\\"production\\\"} |= \\\"error\\\"\")",
					},
					"datasource": {
						Type:        "string",
						Description: "Loki datasource UID or name",
					},
					"maxLines": {
						Type:        "number",
						Description: "Maximum number of log lines to fetch (default: 1000, max: 5000)",
					},
				},
				Required: []string{},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "open_explore_view",
			Description: "Open Explore view with a pre-populated query",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"datasource": {
						Type:        "string",
						Description: "Datasource name or UID",
					},
					"query": {
						Type:        "string",
						Description: "The query to run",
					},
					"queryType": {
						Type:        "string",
						Description: "Type of query",
						Enum:        []string{"metrics", "logs", "traces"},
					},
				},
				Required: []string{"datasource", "query"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "execute_promql",
			Description: "Execute a PromQL query against Prometheus and return structured analytics including trends, anomalies, min/max/avg values. CRITICAL: ALWAYS call this after generating a query when investigating metrics - do not generate a query and stop. This provides actual data for analysis.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"datasourceUid": {
						Type:        "string",
						Description: "The Prometheus datasource UID",
					},
					"query": {
						Type:        "string",
						Description: "The PromQL query to execute (e.g., 'rate(http_requests_total[5m])')",
					},
					"from": {
						Type:        "string",
						Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
					},
					"to": {
						Type:        "string",
						Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
					},
					"step": {
						Type:        "string",
						Description: "Query resolution step (e.g., '15s', '1m'). Optional - Grafana calculates if not provided",
					},
				},
				Required: []string{"datasourceUid", "query"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "execute_logql",
			Description: "Execute a LogQL query against Loki and return log analysis including error rates, patterns, trends. CRITICAL: ALWAYS call this after generating a query when investigating logs - do not generate a query and stop. This provides actual log data for troubleshooting.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"datasourceUid": {
						Type:        "string",
						Description: "The Loki datasource UID",
					},
					"query": {
						Type:        "string",
						Description: "The LogQL query to execute (e.g., '{namespace=\"production\"} |= \"error\"')",
					},
					"from": {
						Type:        "string",
						Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
					},
					"to": {
						Type:        "string",
						Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of log lines to return (default: 1000, max: 5000)",
					},
				},
				Required: []string{"datasourceUid", "query"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "execute_traceql",
			Description: "Execute a TraceQL query against Tempo and return trace analytics including latency percentiles (p50, p95, p99), error rates, and service/operation breakdowns. CRITICAL: ALWAYS call this after generating a query when investigating traces - do not generate a query and stop. This provides actual trace data for distributed system troubleshooting.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"datasourceUid": {
						Type:        "string",
						Description: "The Tempo datasource UID",
					},
					"query": {
						Type:        "string",
						Description: "The TraceQL query to execute (e.g., '{service.name=\"api-gateway\"}')",
					},
					"from": {
						Type:        "string",
						Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
					},
					"to": {
						Type:        "string",
						Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of traces to return (default: 100, max: 1000)",
					},
				},
				Required: []string{"datasourceUid", "query"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "get_firing_alerts",
			Description: "Fetch currently firing alerts from Grafana's Alertmanager. Returns a count of firing alerts, details for each alert (name, severity, service, labels, start time, summary), and detects patterns when multiple alerts share the same service. Use this to understand the current alert state before investigating issues.",
			Parameters: ToolParameters{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
				Required:   []string{},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "search_dashboards",
			Description: "Search for Grafana dashboards by title or tag. Use this to discover what dashboards exist before navigating to one or recommending a dashboard to the user.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"query": {
						Type:        "string",
						Description: "Search query to filter dashboards by title (e.g., \"kubernetes\", \"latency\")",
					},
					"tag": {
						Type:        "string",
						Description: "Filter dashboards by tag (e.g., \"production\", \"team-platform\")",
					},
					"limit": {
						Type:        "number",
						Description: "Maximum number of results to return (default: 50)",
					},
				},
				Required: []string{},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "get_dashboard",
			Description: "Fetch details of a specific Grafana dashboard by UID. Returns the dashboard title, tags, folder, and a list of panels (id, title, type). Use this to understand what panels a dashboard has before fetching panel data or guiding the user.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"uid": {
						Type:        "string",
						Description: "The unique identifier (UID) of the dashboard to fetch",
					},
				},
				Required: []string{"uid"},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "get_annotations",
			Description: "Fetch Grafana annotations for a given time range and optionally for a specific dashboard. Useful for understanding deployment or incident timelines — annotations often mark deployments, incidents, or other events.",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"dashboardUID": {
						Type:        "string",
						Description: "Filter annotations to a specific dashboard UID (optional)",
					},
					"from": {
						Type:        "string",
						Description: "Start time for annotations (e.g., \"now-7d\", \"now-1h\"). Default: \"now-1h\"",
					},
					"to": {
						Type:        "string",
						Description: "End time for annotations (e.g., \"now\"). Default: \"now\"",
					},
				},
				Required: []string{},
			},
		},
	},
	{
		Type: "function",
		Function: Function{
			Name:        "list_folders",
			Description: "List all Grafana folders. Use this to discover how dashboards are organized and to identify which folders exist before searching for dashboards within a specific area.",
			Parameters: ToolParameters{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
				Required:   []string{},
			},
		},
	},
}

func GetTools(functionCallingEnabled bool, settings *Settings, datasources []DatasourceInfo) []Tool {
	if !functionCallingEnabled {
		return nil
	}

	hasPrometheus := hasDatasourceType(datasources, "prometheus")
	hasLoki := hasDatasourceType(datasources, "loki")
	hasTempo := hasDatasourceType(datasources, "tempo")

	tools := make([]Tool, 0, len(ZAGALIN_TOOLS))

	for _, tool := range ZAGALIN_TOOLS {
		name := tool.Function.Name

		// Gate datasource-specific tools based on what is actually configured.
		if (name == "create_promql_query" || name == "execute_promql") && !hasPrometheus {
			continue
		}
		if (name == "create_logql_query" || name == "execute_logql" || name == "get_logs") && !hasLoki {
			continue
		}
		if (name == "create_traceql_query" || name == "execute_traceql" || name == "get_trace_by_id") && !hasTempo {
			continue
		}

		switch name {
		case "create_promql_query":
			tools = append(tools, buildPromQLTool(settings, datasources))
		case "create_logql_query":
			tools = append(tools, buildLogQLTool(settings, datasources))
		case "create_traceql_query":
			tools = append(tools, buildTraceQLTool(settings, datasources))
		case "execute_promql":
			tools = append(tools, buildExecutePromQLTool(settings, datasources))
		case "execute_logql":
			tools = append(tools, buildExecuteLogQLTool(settings, datasources))
		case "execute_traceql":
			tools = append(tools, buildExecuteTraceQLTool(settings, datasources))
		default:
			tools = append(tools, tool)
		}
	}

	return tools
}

func buildPromQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"metric": {
			Type:        "string",
			Description: "The metric name to query (e.g., http_requests_total)",
		},
		"filters": {
			Type:        "object",
			Description: "Label filters as key-value pairs (e.g., {job: \"api\", status: \"200\"})",
		},
		"aggregation": {
			Type:        "string",
			Description: "Aggregation function to apply",
			Enum:        []string{"sum", "avg", "min", "max", "count", "rate"},
		},
		"timeRange": {
			Type:        "string",
			Description: "Time range for rate calculations (e.g., \"5m\", \"1h\")",
		},
	}

	description := "Generate a PromQL query for Prometheus metrics" + buildDatasourceHint(datasources, "prometheus")

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += ". IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "create_promql_query",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"metric"},
			},
		},
	}
}

func buildLogQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"logStream": {
			Type:        "string",
			Description: "Log stream selector (e.g., {job=\"varlogs\"})",
		},
		"filter": {
			Type:        "string",
			Description: "Log line filter expression",
		},
		"parser": {
			Type:        "string",
			Description: "Parser to use for log lines",
			Enum:        []string{"json", "logfmt", "pattern", "regexp"},
		},
	}

	description := "Generate a LogQL query for Loki logs" + buildDatasourceHint(datasources, "loki")

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += ". IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "create_logql_query",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"logStream"},
			},
		},
	}
}

func buildTraceQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"traceSelector": {
			Type:        "string",
			Description: "Trace selector expression (e.g., {service.name=\"api-gateway\"})",
		},
		"filters": {
			Type:        "object",
			Description: "Additional attribute filters (e.g., {status: \"error\", \"http.status_code\": \"500\"})",
		},
		"duration": {
			Type:        "string",
			Description: "Duration filter (e.g., \"> 1s\", \"< 100ms\")",
		},
	}

	description := "Generate a TraceQL query for Tempo traces" + buildDatasourceHint(datasources, "tempo")

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += ". IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "create_traceql_query",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"traceSelector"},
			},
		},
	}
}

func buildExecutePromQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"datasourceUid": {
			Type:        "string",
			Description: "The Prometheus datasource UID" + buildDatasourceHint(datasources, "prometheus"),
		},
		"query": {
			Type:        "string",
			Description: "The PromQL query to execute (e.g., 'rate(http_requests_total[5m])')",
		},
		"from": {
			Type:        "string",
			Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
		},
		"to": {
			Type:        "string",
			Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
		},
		"step": {
			Type:        "string",
			Description: "Query resolution step (e.g., '15s', '1m'). Optional - Grafana calculates if not provided",
		},
	}

	description := "Execute a PromQL query against Prometheus and return structured analytics including trends, anomalies, min/max/avg values. CRITICAL: ALWAYS call this after generating a query when investigating metrics - do not generate a query and stop. This provides actual data for analysis."

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += " IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "execute_promql",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"datasourceUid", "query"},
			},
		},
	}
}

func buildExecuteLogQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"datasourceUid": {
			Type:        "string",
			Description: "The Loki datasource UID" + buildDatasourceHint(datasources, "loki"),
		},
		"query": {
			Type:        "string",
			Description: "The LogQL query to execute (e.g., '{namespace=\"production\"} |= \"error\"')",
		},
		"from": {
			Type:        "string",
			Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
		},
		"to": {
			Type:        "string",
			Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
		},
		"limit": {
			Type:        "number",
			Description: "Maximum number of log lines to return (default: 1000, max: 5000)",
		},
	}

	description := "Execute a LogQL query against Loki and return log analysis including error rates, patterns, trends. CRITICAL: ALWAYS call this after generating a query when investigating logs - do not generate a query and stop. This provides actual log data for troubleshooting."

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += " IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "execute_logql",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"datasourceUid", "query"},
			},
		},
	}
}

func buildExecuteTraceQLTool(settings *Settings, datasources []DatasourceInfo) Tool {
	props := map[string]PropertyDefinition{
		"datasourceUid": {
			Type:        "string",
			Description: "The Tempo datasource UID" + buildDatasourceHint(datasources, "tempo"),
		},
		"query": {
			Type:        "string",
			Description: "The TraceQL query to execute (e.g., '{service.name=\"api-gateway\"}')",
		},
		"from": {
			Type:        "string",
			Description: "Start time for query (e.g., 'now-15m', '2024-01-01T00:00:00Z'). Default: 'now-15m'",
		},
		"to": {
			Type:        "string",
			Description: "End time for query (e.g., 'now', '2024-01-01T01:00:00Z'). Default: 'now'",
		},
		"limit": {
			Type:        "number",
			Description: "Maximum number of traces to return (default: 100, max: 1000)",
		},
	}

	description := "Execute a TraceQL query against Tempo and return trace analytics including latency percentiles (p50, p95, p99), error rates, and service/operation breakdowns. CRITICAL: ALWAYS call this after generating a query when investigating traces - do not generate a query and stop. This provides actual trace data for distributed system troubleshooting."

	if settings != nil && settings.OtelEnforcement.Enabled {
		props["serviceName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry service.name for multi-tenant scoping (e.g., \"api-gateway\", \"payment-service\"). Extract from dashboard context if available.",
		}
		props["environmentName"] = PropertyDefinition{
			Type:        "string",
			Description: "OpenTelemetry deployment.environment.name (e.g., \"production\", \"staging\", \"development\"). Extract from dashboard context if available.",
		}
		description += " IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
	}

	return Tool{
		Type: "function",
		Function: Function{
			Name:        "execute_traceql",
			Description: description,
			Parameters: ToolParameters{
				Type:       "object",
				Properties: props,
				Required:   []string{"datasourceUid", "query"},
			},
		},
	}
}
