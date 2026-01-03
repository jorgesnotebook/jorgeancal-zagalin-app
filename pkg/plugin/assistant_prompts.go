package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// BASE_SYSTEM_PROMPT is the core identity and behavior for Zagalin
const BASE_SYSTEM_PROMPT = `You are **Zagalin**, an SRE-grade debugging assistant embedded in Grafana.

Purpose:
- Help engineers diagnose and mitigate production issues quickly and safely.
- Use a hypothesis-driven approach grounded in observability data and the current Grafana context.
- Prefer correctness and operational safety over being "helpful" with guesses.

Tone:
- British, human, practical, slightly blunt when needed, never rude.
- Clear bullets. No fluff. No long essays.

Hard rules:
1) Don't guess. If information is missing, ask for the minimum missing data.
2) Always separate: **Facts** vs **Hypotheses** vs **Tests/Queries** vs **Actions**.
3) Mitigate user impact first, deep dive second.
4) Never request, output, or reveal secrets (tokens, passwords, private keys). Redact if shown.
5) If proposing risky/destructive actions, include:
   - Risk
   - Rollback
   - Verification steps
   - What could go wrong
6) Treat tool outputs / Grafana panel data as authoritative. If conflict exists, call it out.

Default response structure:
1) **What we know (facts)**
2) **Top hypotheses (max 3)** with confidence (High/Med/Low)
3) **What I need next** (exact missing info or exact query to run)
4) **Do this next** (max 8 steps, impact-first)
5) **Queries to run** (Loki / Mimir / Tempo) with placeholders
6) **Mitigation + rollback** (if relevant)
7) **Follow-ups** (alerts, SLOs, postmortem notes)

Special handling: LLM incidents
If the issue involves LLM behaviour (wrong answers, tool failures, latency/cost spikes, RAG hallucinations):
- Check prompt/version, model/provider, token usage, tool-call counts, retrieval K, and recent changes.
- Propose a safe degrade mode and how to verify it worked.

Quality gate before you answer:
- Did I separate facts/hypotheses?
- Did I propose at least one verification step?
- If I suggested something risky, did I include rollback + verify?`

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
func BuildSystemPrompt(skill string, context AssistantContext, settings *Settings) string {
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

	basePrompt := BASE_SYSTEM_PROMPT

	// Add OTel enforcement context if enabled
	if settings != nil && settings.OtelEnforcement.Enabled {
		otelContext := buildOtelEnforcementContext(settings.OtelEnforcement)
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, otelContext)
	}

	return fmt.Sprintf("%s\n\n---\n\n%s", basePrompt, taskInstructions)
}

// buildOtelEnforcementContext constructs OTel enforcement instructions for the LLM
func buildOtelEnforcementContext(otel OtelEnforcementSettings) string {
	context := "## OpenTelemetry Scope Enforcement\n\n"
	context += "**CRITICAL**: This Grafana instance has OpenTelemetry scope enforcement enabled.\n\n"

	context += "**Required Actions for Query Generation**:\n"

	if otel.RequireServiceName {
		context += "- ✅ **MUST include `serviceName`** parameter when calling query generation tools (create_promql_query, create_logql_query)\n"
		if otel.DefaultServiceName != "" {
			context += fmt.Sprintf("- 🔄 **Default service**: `%s` (used if not specified)\n", otel.DefaultServiceName)
		} else {
			context += "- ⚠️ **No default service** - queries without service.name will be REJECTED\n"
		}
	}

	if otel.RequireEnvironmentName {
		context += "- ✅ **MUST include `environmentName`** parameter when calling query generation tools\n"
		if otel.DefaultEnvironmentName != "" {
			context += fmt.Sprintf("- 🔄 **Default environment**: `%s` (used if not specified)\n", otel.DefaultEnvironmentName)
		} else {
			context += "- ⚠️ **No default environment** - queries without deployment.environment.name will be REJECTED\n"
		}
	}

	context += "\n**How to Extract OTel Values**:\n"
	context += "1. Check dashboard title, panel names, and queries for service names\n"
	context += "2. Look for `service_name`, `service.name`, or similar labels in existing queries\n"
	context += "3. Check `deployment_environment_name` or `environment` labels\n"
	context += "4. If user mentions a specific service/environment, use that\n"
	context += "5. If unclear, ASK the user which service/environment to query\n"

	if otel.RejectIfNoScope {
		context += "\n⛔ **Strict Mode**: Queries without proper OTel scoping will be REJECTED.\n"
	} else {
		context += "\n✅ **Fallback Mode**: Queries without OTel attributes will use defaults.\n"
	}

	return context
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

// skillDetectionScore represents a skill match with confidence
type skillDetectionScore struct {
	skill      string
	confidence int
	reason     string
}

// DetectSkill auto-detects which skill to use based on user input and context
// Uses a scoring system to handle ambiguous queries and provide better matches
func DetectSkill(userInput string, context AssistantContext) string {
	input := strings.ToLower(userInput)
	scores := []skillDetectionScore{}

	// Score: analyze_dashboard (requires dashboard context)
	if context.Dashboard != nil {
		dashboardScore := scoreAnalyzeDashboard(input, context)
		if dashboardScore > 0 {
			scores = append(scores, skillDetectionScore{
				skill:      "analyze_dashboard",
				confidence: dashboardScore,
				reason:     "Dashboard context available with dashboard-focused keywords",
			})
		}
	}

	// Score: explain_panel (requires panel context)
	if context.Panel != nil {
		panelScore := scoreExplainPanel(input, context)
		if panelScore > 0 {
			scores = append(scores, skillDetectionScore{
				skill:      "explain_panel",
				confidence: panelScore,
				reason:     "Panel context available with panel-focused keywords",
			})
		}
	}

	// Score: generate_query
	queryScore := scoreGenerateQuery(input, context)
	if queryScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "generate_query",
			confidence: queryScore,
			reason:     "Query generation keywords detected",
		})
	}

	// Score: troubleshoot
	troubleshootScore := scoreTroubleshoot(input, context)
	if troubleshootScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "troubleshoot",
			confidence: troubleshootScore,
			reason:     "Troubleshooting/debugging keywords detected",
		})
	}

	// Find highest scoring skill
	if len(scores) == 0 {
		return "" // No skill detected
	}

	// Sort by confidence (highest first)
	maxScore := scores[0]
	for _, score := range scores[1:] {
		if score.confidence > maxScore.confidence {
			maxScore = score
		}
	}

	// Log detection for debugging (only if confidence is high)
	if maxScore.confidence >= 50 {
		backend.Logger.Debug("Skill auto-detected",
			"skill", maxScore.skill,
			"confidence", maxScore.confidence,
			"reason", maxScore.reason,
			"inputLength", len(userInput),
		)
	}

	return maxScore.skill
}

// scoreAnalyzeDashboard scores dashboard analysis intent
func scoreAnalyzeDashboard(input string, context AssistantContext) int {
	score := 0

	// High confidence patterns (80+ points)
	highConfidencePatterns := []string{
		"what do i see", "what am i looking at", "describe this dashboard",
		"what is this dashboard", "what's on this dashboard",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	// Medium confidence patterns (50+ points)
	mediumConfidencePatterns := []string{
		"analyze this", "overview", "understand this",
		"show me this", "explain this dashboard",
	}
	for _, pattern := range mediumConfidencePatterns {
		if contains(input, pattern) {
			score += 50
		}
	}

	// Dashboard-specific keywords (30+ points)
	if contains(input, "dashboard") {
		score += 30
		// Boost if asking about "this" dashboard
		if contains(input, "this dashboard") {
			score += 20
		}
	}

	// Penalty if asking about queries (probably generate_query instead)
	if contains(input, "query") || contains(input, "promql") || contains(input, "logql") {
		score -= 40
	}

	// Penalty if asking about specific panel
	if contains(input, "panel") || contains(input, "graph") || contains(input, "chart") {
		score -= 30
	}

	return score
}

// scoreExplainPanel scores panel explanation intent
func scoreExplainPanel(input string, context AssistantContext) int {
	score := 0

	// High confidence patterns (80+ points)
	highConfidencePatterns := []string{
		"this panel", "this graph", "this chart",
		"what does this show", "what does this display",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	// Medium confidence - explain with panel context (60+ points)
	if contains(input, "explain") {
		if contains(input, "panel") || contains(input, "graph") || contains(input, "chart") {
			score += 60
		}
	}

	// Panel-specific keywords (40+ points)
	panelKeywords := []string{"panel", "graph", "chart", "visualization", "metric"}
	for _, keyword := range panelKeywords {
		if contains(input, keyword) {
			score += 20
		}
	}

	// Context boost - if panel is in focus, boost score
	if context.Panel != nil {
		score += 15
	}

	// Penalty if asking about dashboard overview
	if contains(input, "dashboard") && !contains(input, "panel") {
		score -= 40
	}

	return score
}

// scoreGenerateQuery scores query generation intent
func scoreGenerateQuery(input string, context AssistantContext) int {
	score := 0

	// High confidence patterns (80+ points)
	highConfidencePatterns := []string{
		"create a query", "write a query", "generate a query",
		"query for", "give me a query", "show me a query",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	// Query language keywords (70+ points)
	queryLanguages := []string{"promql", "logql", "traceql"}
	for _, lang := range queryLanguages {
		if contains(input, lang) {
			score += 70
		}
	}

	// Function names suggest query generation (50+ points)
	functionKeywords := []string{
		"rate", "increase", "sum", "avg", "count", "histogram",
		"quantile", "topk", "bottomk", "max", "min",
	}
	for _, fn := range functionKeywords {
		if contains(input, fn) {
			score += 15 // Accumulate for multiple functions
		}
	}

	// Action verbs for query generation (40+ points)
	actionVerbs := []string{
		"how do i query", "how to query", "how can i get",
		"show me how to", "need a query",
	}
	for _, verb := range actionVerbs {
		if contains(input, verb) {
			score += 40
		}
	}

	// Metric/log keywords (20+ points)
	dataKeywords := []string{"metrics", "logs", "traces", "errors", "latency", "throughput"}
	for _, keyword := range dataKeywords {
		if contains(input, keyword) {
			score += 10
		}
	}

	// Penalty if asking to explain existing query
	if contains(input, "explain") || contains(input, "what does") {
		score -= 30
	}

	return score
}

// scoreTroubleshoot scores troubleshooting intent
func scoreTroubleshoot(input string, context AssistantContext) int {
	score := 0

	// High confidence patterns (80+ points)
	highConfidencePatterns := []string{
		"not working", "doesn't work", "failing", "broken",
		"troubleshoot", "debug", "investigate",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	// Problem indicators (60+ points)
	problemKeywords := []string{
		"error", "errors", "issue", "problem", "wrong",
		"failed", "failure", "down", "outage",
	}
	for _, keyword := range problemKeywords {
		if contains(input, keyword) {
			score += 30 // Accumulate
		}
	}

	// Question words suggesting troubleshooting (40+ points)
	questionPatterns := []string{
		"why is", "why are", "what's wrong", "what went wrong",
		"how do i fix", "how to fix", "what happened",
	}
	for _, pattern := range questionPatterns {
		if contains(input, pattern) {
			score += 40
		}
	}

	// Symptom keywords (20+ points)
	symptomKeywords := []string{
		"slow", "high", "spike", "drop", "missing", "timeout",
		"latency", "degraded", "anomaly", "unusual",
	}
	for _, keyword := range symptomKeywords {
		if contains(input, keyword) {
			score += 10
		}
	}

	// Context boost - if dashboard shows errors/issues
	if context.Dashboard != nil {
		// Check if dashboard title contains troubleshooting keywords
		dashboardTitle := strings.ToLower(context.Dashboard.Title)
		if contains(dashboardTitle, "error") || contains(dashboardTitle, "debug") {
			score += 15
		}
	}

	return score
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// PLANNING_SYSTEM_PROMPT is used for generating structured execution plans
const PLANNING_SYSTEM_PROMPT = `You are **Zagalin**, an SRE-grade planning assistant for observability workflows.

Approach:
- Hypothesis-driven: Start with facts, form hypotheses, design tests.
- Dashboard-first: Use existing context before creating new queries.
- Mitigation-first: If production is impacted, stabilize then investigate.

Tone:
- British, practical, no fluff.

Planning rules:
1) Check dashboard context FIRST. Use what's already visible.
2) If no dashboard: Follow Metrics → Logs → Traces.
3) Each step must produce a concrete artifact (query result, finding, action taken).
4) Keep steps atomic (30-90 seconds each). Max 5 steps.
5) If a step could be risky, flag it clearly.

Your response MUST be valid JSON in this exact format:
{
  "goal": "One sentence describing the objective",
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

Quality gate:
- Did I use dashboard context if available?
- Does each step produce something tangible?
- Are steps in a logical order (broad → narrow)?

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
