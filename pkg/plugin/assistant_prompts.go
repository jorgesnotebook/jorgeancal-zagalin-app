package plugin

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BASE_SYSTEM_PROMPT is the core identity and behavior for Zagalin
const BASE_SYSTEM_PROMPT = `You are Zagalin, a Senior Staff SRE embedded in Grafana. You've been on-call for years and have deep operational experience with observability systems.

Your Role:
- Help engineers understand their metrics, logs, and traces
- Identify reliability issues before they become incidents
- Suggest practical improvements based on SRE best practices
- Debug production issues using the dashboard context you have

How You Work:
- You have full context about the current dashboard and panels - use it directly
- Prioritize: Reliability > Performance > Features
- Explain the "why" behind recommendations (you're teaching, not dictating)
- Focus on actionable insights, not theory
- If you see potential production risks, call them out clearly

When You Don't Have Dashboard Context (Observability Workflow):
1. **Start with Metrics** - Get high-level context from available metrics (PromQL)
   - What services are affected? What's the error rate? Is latency spiking?
2. **Dive into Logs** - If metrics don't tell the full story, check logs (LogQL)
   - What are the actual error messages? What patterns do you see?
3. **Trace for Details** - Use logs to find trace IDs, then pull traces (TraceQL)
   - What's the exact failure path? Which service is the bottleneck?

This is the SRE debugging workflow - start broad, narrow down. Always follow this pattern.

Communication Style:
- Concise and practical - engineers are busy
- Use technical terms correctly - they know what they're doing
- Provide code/queries ready to use
- When suggesting changes, explain operational impact

You've seen these systems fail at 3am. Share that experience.`

// SkillPrompt represents a task-specific system prompt
type SkillPrompt struct {
	Name             string
	TaskInstructions string
}

// SkillPrompts contains all available assistant skills
var SkillPrompts = map[string]SkillPrompt{
	"explain_panel": {
		Name: "explain_panel",
		TaskInstructions: `Your task is to explain what a panel shows, how it works, and potential issues.

Follow this structure:
1. **What it shows**: Explain in plain English what the panel displays
2. **How it works**: Break down the queries and transformations
3. **Common pitfalls**: Warn about potential misinterpretations or issues
4. **Suggestions**: Offer 1-2 improvements if applicable`,
	},
	"generate_query": {
		Name: "generate_query",
		TaskInstructions: `Your task is to convert English requests into valid %s queries.

Guidelines:
- Generate syntactically correct %s
- Use best practices (avoid high cardinality, use appropriate functions)
- Explain what the query does
- Warn about potential performance issues
- Suggest time range if not specified

Format your response as:
` + "```%s" + `
<query>
` + "```" + `

**Explanation**: <what the query does>
**Performance**: <any warnings about cardinality, time range, etc>
**Usage**: <suggested time range and resolution>`,
	},
	"troubleshoot": {
		Name: "troubleshoot",
		TaskInstructions: `Your task is to provide a structured troubleshooting guide.

Format your response as:

**Problem Summary**: <restate the issue in one sentence>

**Quick Checks** (1-3 items):
- [ ] <simple check>
- [ ] <simple check>

**Diagnostic Queries** (3-5 queries to run):
1. **Check X**: ` + "`<query>`" + ` - <what to look for>
2. **Verify Y**: ` + "`<query>`" + ` - <what to look for>
3. **Investigate Z**: ` + "`<query>`" + ` - <what to look for>

**Common Causes**:
- <likely cause 1>
- <likely cause 2>

**Next Steps**: <what to do based on findings>

Keep it practical and actionable. Reference the current dashboard context when relevant.`,
	},
	"analyze_dashboard": {
		Name: "analyze_dashboard",
		TaskInstructions: `The user is asking about what they see on their screen.

Your task is to:
1. **Describe the overall purpose** - What story does this dashboard tell? What system/service is being monitored?
2. **Summarize key panels** - What are the most important visualizations? Group related panels together
3. **Identify patterns** - What should the user focus on? Are there any red flags or interesting trends?
4. **Provide context** - Why would someone use this dashboard? What decisions does it support?

Be conversational and insightful. Imagine you're sitting next to them explaining what you see.`,
	},
}

// AssistantContext represents the Grafana context for the assistant
type AssistantContext struct {
	Dashboard    *DashboardContext  `json:"dashboard,omitempty"`
	Panel        *PanelContext      `json:"panel,omitempty"`
	TimeRange    *TimeRange         `json:"timeRange,omitempty"`
	TemplateVars []TemplateVariable `json:"templateVars,omitempty"`
}

// DashboardContext contains dashboard metadata
type DashboardContext struct {
	UID    string         `json:"uid"`
	Title  string         `json:"title"`
	Tags   []string       `json:"tags,omitempty"`
	Panels []PanelContext `json:"panels,omitempty"`
}

// PanelContext contains panel configuration
type PanelContext struct {
	Title           string                 `json:"title"`
	Type            string                 `json:"type"`
	Description     string                 `json:"description,omitempty"`
	Targets         []QueryTarget          `json:"targets"`
	FieldConfig     map[string]interface{} `json:"fieldConfig,omitempty"`
	Transformations []interface{}          `json:"transformations,omitempty"`
}

// QueryTarget represents a panel query
type QueryTarget struct {
	RefID      string                 `json:"refId"`
	Expr       string                 `json:"expr,omitempty"`
	Query      string                 `json:"query,omitempty"`
	Datasource map[string]interface{} `json:"datasource,omitempty"`
}

// TemplateVariable represents a dashboard template variable
type TemplateVariable struct {
	Name    string                 `json:"name"`
	Current map[string]interface{} `json:"current"`
}

// BuildSystemPrompt constructs the full system prompt for a skill with context
func BuildSystemPrompt(skill string, context AssistantContext) string {
	skillPrompt, exists := SkillPrompts[skill]
	if !exists {
		// Default to base prompt if skill not found
		return BASE_SYSTEM_PROMPT
	}

	taskInstructions := skillPrompt.TaskInstructions

	// For generate_query skill, detect query language from context
	if skill == "generate_query" {
		queryLang := detectQueryLanguage(context)
		taskInstructions = fmt.Sprintf(taskInstructions, queryLang, queryLang, queryLang)
	}

	return fmt.Sprintf("%s\n\n---\n\n%s", BASE_SYSTEM_PROMPT, taskInstructions)
}

// BuildUserPrompt constructs the user prompt with context injection
func BuildUserPrompt(skill string, userMessage string, context AssistantContext) string {
	switch skill {
	case "explain_panel":
		return buildExplainPanelPrompt(context)
	case "generate_query":
		return buildGenerateQueryPrompt(userMessage, context)
	case "troubleshoot":
		return buildTroubleshootPrompt(userMessage, context)
	case "analyze_dashboard":
		return buildAnalyzeDashboardPrompt(context)
	default:
		// For unrecognized skills, just return the user message
		return userMessage
	}
}

func buildExplainPanelPrompt(context AssistantContext) string {
	if context.Panel == nil {
		return "No panel context available."
	}

	panel := context.Panel
	prompt := fmt.Sprintf("Explain this panel:\n\n**Panel Title**: %s\n**Panel Type**: %s\n", panel.Title, panel.Type)

	if panel.Description != "" {
		prompt += fmt.Sprintf("**Description**: %s\n", panel.Description)
	}

	prompt += "\n**Queries**:\n"
	for _, target := range panel.Targets {
		query := target.Expr
		if query == "" {
			query = target.Query
		}
		prompt += fmt.Sprintf("Query %s: %s\n", target.RefID, query)
	}

	// Add field config info if available
	if fieldConfig, ok := panel.FieldConfig["defaults"].(map[string]interface{}); ok {
		if unit, ok := fieldConfig["unit"].(string); ok && unit != "" {
			prompt += fmt.Sprintf("\n**Unit**: %s\n", unit)
		}
	}

	if len(panel.Transformations) > 0 {
		prompt += fmt.Sprintf("\n**Transformations**: %d applied\n", len(panel.Transformations))
	}

	return prompt
}

func buildGenerateQueryPrompt(userMessage string, context AssistantContext) string {
	queryLang := detectQueryLanguage(context)
	prompt := fmt.Sprintf("Generate a %s query for: \"%s\"\n", queryLang, userMessage)

	// Add template variables if available
	if len(context.TemplateVars) > 0 {
		prompt += "\nAvailable template variables:\n"
		for _, v := range context.TemplateVars {
			if currentValue, ok := v.Current["value"]; ok {
				prompt += fmt.Sprintf("- $%s = %v\n", v.Name, currentValue)
			}
		}
	}

	// Add current panel context if available
	if context.Panel != nil && len(context.Panel.Targets) > 0 {
		prompt += "\nCurrent panel context:\n"
		for _, target := range context.Panel.Targets {
			query := target.Expr
			if query == "" {
				query = target.Query
			}
			prompt += fmt.Sprintf("- %s: %s\n", target.RefID, query)
		}
	}

	return prompt
}

func buildTroubleshootPrompt(userMessage string, context AssistantContext) string {
	prompt := fmt.Sprintf("Help troubleshoot: \"%s\"\n", userMessage)

	if context.Dashboard != nil {
		prompt += fmt.Sprintf("\nCurrent Dashboard: \"%s\"\n", context.Dashboard.Title)
	}

	if context.Panel != nil {
		prompt += fmt.Sprintf("Current Panel: \"%s\" (%s)\n", context.Panel.Title, context.Panel.Type)
	}

	if context.TimeRange != nil {
		prompt += fmt.Sprintf("Time Range: %s to %s\n", context.TimeRange.From, context.TimeRange.To)
	}

	return prompt
}

func buildAnalyzeDashboardPrompt(context AssistantContext) string {
	if context.Dashboard == nil {
		return "No dashboard context available. The user might not be on a dashboard page."
	}

	dashboard := context.Dashboard
	prompt := fmt.Sprintf("# Dashboard: \"%s\"\n", dashboard.Title)

	if len(dashboard.Tags) > 0 {
		tagsJSON, _ := json.Marshal(dashboard.Tags)
		prompt += fmt.Sprintf("Tags: %s\n", string(tagsJSON))
	}

	prompt += "\n"

	// List all panels
	if len(dashboard.Panels) > 0 {
		prompt += fmt.Sprintf("## Panels on this dashboard (%d total):\n\n", len(dashboard.Panels))

		for idx, panel := range dashboard.Panels {
			prompt += fmt.Sprintf("%d. **%s** (%s)\n", idx+1, panel.Title, panel.Type)

			if panel.Description != "" {
				prompt += fmt.Sprintf("   Description: %s\n", panel.Description)
			}

			if len(panel.Targets) > 0 {
				queryCount := 0
				for _, target := range panel.Targets {
					if target.Expr != "" || target.Query != "" {
						queryCount++
					}
				}
				if queryCount > 0 {
					prompt += fmt.Sprintf("   Queries: %d query(ies)\n", queryCount)
				}
			}
		}
	}

	// Add time range context
	if context.TimeRange != nil {
		prompt += "\n## Time Range\n"
		prompt += fmt.Sprintf("Looking at data from %s to %s\n", context.TimeRange.From, context.TimeRange.To)
	}

	// Add panel focus if viewing specific panel
	if context.Panel != nil {
		prompt += "\n## Currently Focused Panel\n"
		prompt += fmt.Sprintf("The user has \"%s\" (%s) in focus.\n", context.Panel.Title, context.Panel.Type)
	}

	return prompt
}

// detectQueryLanguage determines the query language from context
func detectQueryLanguage(context AssistantContext) string {
	if context.Panel == nil || len(context.Panel.Targets) == 0 {
		return "PromQL" // Default to Prometheus
	}

	// Check datasource types
	for _, target := range context.Panel.Targets {
		if dsType, ok := target.Datasource["type"].(string); ok {
			switch dsType {
			case "loki":
				return "LogQL"
			case "tempo":
				return "TraceQL"
			case "prometheus":
				return "PromQL"
			}
		}
	}

	return "PromQL" // Default
}

// DetectSkill auto-detects which skill to use based on user input and context
func DetectSkill(userInput string, context AssistantContext) string {
	// Simple keyword-based detection (can be enhanced with more sophisticated NLP)
	input := userInput

	// Dashboard analysis triggers - check FIRST to catch broad queries
	if context.Dashboard != nil {
		keywords := []string{
			"what do i see", "what am i looking at", "describe this dashboard",
			"look at this", "in front of me", "what is this dashboard",
			"show me this", "analyze this", "understand this", "overview",
		}
		for _, keyword := range keywords {
			if contains(input, keyword) && !contains(input, "query") {
				return "analyze_dashboard"
			}
		}
		if contains(input, "this dashboard") && !contains(input, "query") {
			return "analyze_dashboard"
		}
	}

	// Explain panel triggers (more specific than dashboard analysis)
	if context.Panel != nil {
		keywords := []string{"this panel", "this graph", "this chart", "what does this show"}
		for _, keyword := range keywords {
			if contains(input, keyword) {
				return "explain_panel"
			}
		}
		if (contains(input, "explain") && (contains(input, "panel") || contains(input, "graph"))) {
			return "explain_panel"
		}
	}

	// Generate query triggers
	queryKeywords := []string{
		"query for", "promql", "logql", "create a query", "write a query",
		"rate", "sum", "avg", "count", "histogram",
	}
	for _, keyword := range queryKeywords {
		if contains(input, keyword) {
			return "generate_query"
		}
	}

	// Troubleshooting triggers
	troubleshootKeywords := []string{
		"troubleshoot", "debug", "not working", "why", "error", "failing",
	}
	for _, keyword := range troubleshootKeywords {
		if contains(input, keyword) {
			return "troubleshoot"
		}
	}

	// Default: no specific skill detected
	return ""
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// PLANNING_SYSTEM_PROMPT is used for generating structured execution plans
const PLANNING_SYSTEM_PROMPT = `You are Zagalin, a planning assistant for observability workflows.

When given a task, break it down into clear, actionable steps.

CRITICAL: Check if the user has dashboard context.

**If user is on a dashboard:**
1. FIRST: Analyze what's already visible on the dashboard (panels, queries, current values, patterns)
2. THEN: Only go deeper into raw queries if the dashboard doesn't answer the question
3. Use existing dashboard panels as starting points - don't reinvent the wheel

**If user has NO dashboard context:**
Follow the observability pyramid (Metrics → Logs → Traces):
1. Start with high-level metrics
2. Narrow to logs if needed
3. Trace specific requests if necessary

Your response MUST be valid JSON in this exact format:
{
  "goal": "One sentence describing the overall objective",
  "steps": [
    {
      "title": "Step 1: Analyze dashboard panels",
      "description": "Review error rate and latency panels on current dashboard"
    },
    {
      "title": "Step 2: Check for anomalies",
      "description": "Identify any spikes or drops in the visible metrics"
    },
    {
      "title": "Step 3: Investigate root cause",
      "description": "Query logs for errors if dashboard shows elevated error rate"
    }
  ],
  "estimatedDuration": "2-3 minutes"
}

Guidelines:
- Keep steps atomic (each step should take 30-90 seconds)
- Maximum 5 steps total
- Each step should produce a concrete artifact (query, link, finding)
- PRIORITIZE dashboard context over raw queries when available
- Steps should build on each other logically

DO NOT include any text outside the JSON. NO markdown code blocks. Just pure JSON.`

// BuildPlanningPrompt constructs the user prompt for plan generation
func BuildPlanningPrompt(userMessage string, context AssistantContext) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User request: %s\n\n", userMessage))

	// Emphasize dashboard context if available
	if context.Dashboard != nil {
		prompt.WriteString("DASHBOARD CONTEXT AVAILABLE:\n")
		prompt.WriteString(fmt.Sprintf("Dashboard: \"%s\"\n", context.Dashboard.Title))
		if len(context.Dashboard.Tags) > 0 {
			prompt.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(context.Dashboard.Tags, ", ")))
		}

		if len(context.Dashboard.Panels) > 0 {
			prompt.WriteString(fmt.Sprintf("Visible panels (%d total):\n", len(context.Dashboard.Panels)))
			limit := 10
			if len(context.Dashboard.Panels) < limit {
				limit = len(context.Dashboard.Panels)
			}
			for i := 0; i < limit; i++ {
				panel := context.Dashboard.Panels[i]
				prompt.WriteString(fmt.Sprintf("  %d. %s (%s)\n", i+1, panel.Title, panel.Type))
			}
			if len(context.Dashboard.Panels) > 10 {
				prompt.WriteString(fmt.Sprintf("  ... and %d more panels\n", len(context.Dashboard.Panels)-10))
			}
		}

		prompt.WriteString("\nIMPORTANT: The user is LOOKING at this dashboard right now. Start by analyzing what's visible before suggesting new queries.\n\n")
	} else {
		prompt.WriteString("NO DASHBOARD CONTEXT - User is not on a dashboard.\n")
		prompt.WriteString("Start with high-level queries to gather context.\n\n")
	}

	// Add panel context if available
	if context.Panel != nil {
		prompt.WriteString(fmt.Sprintf("FOCUSED PANEL: \"%s\" (%s)\n", context.Panel.Title, context.Panel.Type))
		if len(context.Panel.Targets) > 0 {
			prompt.WriteString("Panel queries:\n")
			for _, target := range context.Panel.Targets {
				if target.Expr != "" {
					prompt.WriteString(fmt.Sprintf("  - %s\n", target.Expr))
				} else if target.Query != "" {
					prompt.WriteString(fmt.Sprintf("  - %s\n", target.Query))
				}
			}
		}
		prompt.WriteString("\n")
	}

	// Add time range if available
	if context.TimeRange != nil {
		prompt.WriteString(fmt.Sprintf("Time range: %s to %s\n\n", context.TimeRange.From, context.TimeRange.To))
	}

	prompt.WriteString("Create an execution plan that uses available context efficiently.")

	return prompt.String()
}

// parsePlanFromJSON parses the execution plan from LLM response
func parsePlanFromJSON(text string) (*ExecutionPlan, error) {
	// Try direct JSON parse first
	var plan ExecutionPlan
	err := json.Unmarshal([]byte(text), &plan)
	if err == nil && len(plan.Steps) > 0 {
		return &plan, nil
	}

	// Try to extract JSON from markdown code blocks
	jsonPattern := "```json\n"
	startIdx := strings.Index(text, jsonPattern)
	if startIdx >= 0 {
		startIdx += len(jsonPattern)
		endIdx := strings.Index(text[startIdx:], "```")
		if endIdx >= 0 {
			jsonText := text[startIdx : startIdx+endIdx]
			err = json.Unmarshal([]byte(jsonText), &plan)
			if err == nil && len(plan.Steps) > 0 {
				return &plan, nil
			}
		}
	}

	// Try without "json" keyword
	jsonPattern = "```\n"
	startIdx = strings.Index(text, jsonPattern)
	if startIdx >= 0 {
		startIdx += len(jsonPattern)
		endIdx := strings.Index(text[startIdx:], "```")
		if endIdx >= 0 {
			jsonText := text[startIdx : startIdx+endIdx]
			err = json.Unmarshal([]byte(jsonText), &plan)
			if err == nil && len(plan.Steps) > 0 {
				return &plan, nil
			}
		}
	}

	// Fallback: return error
	return nil, fmt.Errorf("failed to parse plan from LLM response: %w", err)
}
