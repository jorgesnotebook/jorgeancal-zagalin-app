package plugin

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
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
			Name:        "get_panel_data",
			Description: "Retrieve current data from a dashboard panel",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"dashboardUid": {
						Type:        "string",
						Description: "Dashboard UID",
					},
					"panelId": {
						Type:        "number",
						Description: "Panel ID",
					},
				},
				Required: []string{"dashboardUid", "panelId"},
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
			Name:        "explain_error",
			Description: "Provide a detailed explanation of an error message or status code",
			Parameters: ToolParameters{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"errorMessage": {
						Type:        "string",
						Description: "The error message or status code to explain",
					},
					"context": {
						Type:        "string",
						Description: "Additional context (e.g., which service, what operation)",
					},
				},
				Required: []string{"errorMessage"},
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
}

func GetTools(functionCallingEnabled bool, settings *Settings) []Tool {
	if !functionCallingEnabled {
		return nil
	}

	tools := make([]Tool, 0, len(ZAGALIN_TOOLS))

	for _, tool := range ZAGALIN_TOOLS {
		if tool.Function.Name == "create_promql_query" {
			tools = append(tools, buildPromQLTool(settings))
		} else if tool.Function.Name == "create_logql_query" {
			tools = append(tools, buildLogQLTool(settings))
		} else if tool.Function.Name == "create_traceql_query" {
			tools = append(tools, buildTraceQLTool(settings))
		} else if tool.Function.Name == "execute_promql" {
			tools = append(tools, buildExecutePromQLTool(settings))
		} else if tool.Function.Name == "execute_logql" {
			tools = append(tools, buildExecuteLogQLTool(settings))
		} else {
			tools = append(tools, tool)
		}
	}

	return tools
}

func buildPromQLTool(settings *Settings) Tool {
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

	description := "Generate a PromQL query for Prometheus metrics"

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

func buildLogQLTool(settings *Settings) Tool {
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

	description := "Generate a LogQL query for Loki logs"

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

func buildTraceQLTool(settings *Settings) Tool {
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

	description := "Generate a TraceQL query for Tempo traces"

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

func buildExecutePromQLTool(settings *Settings) Tool {
	props := map[string]PropertyDefinition{
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

func buildExecuteLogQLTool(settings *Settings) Tool {
	props := map[string]PropertyDefinition{
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
