package plugin

// Tool represents an LLM function calling tool
type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

// Function represents a function definition for LLM tool use
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

// ToolParameters represents the JSON schema for function parameters
type ToolParameters struct {
	Type       string                        `json:"type"`
	Properties map[string]PropertyDefinition `json:"properties"`
	Required   []string                      `json:"required,omitempty"`
}

// PropertyDefinition represents a parameter property schema
type PropertyDefinition struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ZAGALIN_TOOLS contains all available function calling tools
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
			Description: "Generate a PromQL query for Prometheus metrics",
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
			Description: "Generate a LogQL query for Loki logs",
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
}

// GetTools returns the tools to send to the LLM
// If functionCalling is disabled, returns nil
func GetTools(functionCallingEnabled bool) []Tool {
	if !functionCallingEnabled {
		return nil
	}
	return ZAGALIN_TOOLS
}
