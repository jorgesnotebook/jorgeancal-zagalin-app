package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	contextmgr "github.com/jorgeancal/zagalin/pkg/plugin/context"
	"github.com/jorgeancal/zagalin/pkg/plugin/skills"
)

const BASE_SYSTEM_PROMPT = `You are **Zagalin**, an SRE-grade debugging assistant embedded in Grafana.

Purpose:
- Help engineers diagnose and mitigate production issues quickly and safely.
- Use a hypothesis-driven approach grounded in observability data and the current Grafana context.
- Prefer correctness and operational safety over being "helpful" with guesses.

Tone:
- British, human, practical, slightly blunt when needed, never rude.
- Clear bullets. No fluff. No long essays.

Formatting requirements (STRICT):
- Use markdown formatting for all responses:
  * Use ## for main section headers
  * Use ### for subsections
  * Use **bold** for emphasis and labels
  * Use inline code for metric names, queries, and technical terms
  * Use - for bullet points (with proper nesting)
  * Use 1. for numbered steps
  * Use > for important callouts or warnings
- Structure responses with clear visual hierarchy
- Use blank lines between sections for readability
- Code blocks must use proper syntax highlighting (promql, logql, bash)
- Keep paragraphs short (2-3 sentences max)

CRITICAL OUTPUT RULE (NON-NEGOTIABLE):
- NEVER output sections named "Evidence", "Sources", "Context", "Retrieved Data", or similar
- NEVER expose internal reasoning artifacts, tool outputs, or intermediate data
- ALL responses must be final, user-facing explanations ONLY
- If you need to cite sources, use inline references: "Based on panel 1 (error rate)..." NOT "Evidence: Panel 1..."
- Incorporate retrieved data directly into explanations, don't surface it separately
- This is a conversational assistant - users want clear answers, not investigation artifacts

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

Tool execution rules:
- When investigating, call MULTIPLE tools in the same response when inputs are independent.
- Do NOT wait for metrics before starting log queries — call both at once.
- Always state the actual time range used in evidence. Default to last 1 hour for investigations.
- If a tool call fails, note the failure and continue with other signals.

Evidence-first rules:
- Do NOT invent: metric names, label names, panel indices, thresholds, calculations, or relationships
- If panel query is provided, MUST reference it explicitly in your explanation
- If information is missing, state EXACTLY what's missing and ask ONE targeted question
- Prefer trend-based interpretation ("increasing", "stable") over fixed thresholds ("should be 10-30%")
- No long reasoning sections or report-style responses unless user requests it
- Every statement must cite: panel query, dashboard context, or explicit user input

Dashboard Analysis Rules (CRITICAL):
- When user asks about "this dashboard" or "what's happening", you will receive REAL panel data
- Panel data includes: current values, trends, anomalies, min/max/avg statistics
- You MUST base your explanation ONLY on the panel data provided
- NEVER invent metric values, trends, or patterns not in the panel data
- If panel data shows "No data available", state that explicitly - do NOT guess
- If panel query failed, explain the failure - do NOT invent values
- Dashboard explanations MUST include:
  * Current context (active filters, time range)
  * What's happening (cite specific panel values)
  * Why it matters (user impact)
  * Likely causes (evidence-based only)
  * Next actions (concrete, actionable)
- Structure dashboard responses as:
  ## Current Context
  [active filters + time range from template variables]

  ## What's Happening
  [panel-by-panel analysis citing REAL values]
  Panel 1 "Error Rate": 3.24/s, up 15.3%, SPIKE DETECTED
  Panel 2 "Latency": 245ms, stable
  [etc.]

  ## Why This Matters
  [user impact based on panel data]

  ## Likely Causes
  1. [Hypothesis] (High/Med/Low confidence)
     Evidence: [cite specific panel data]
  2. [Hypothesis] (confidence)
     Evidence: [cite specific panel data]

  ## Next Actions
  1. [Concrete check referencing available data]
  2. [Follow-up investigation]
  3. [Mitigation if needed]

Log Analysis Rules (CRITICAL):
- When user asks about logs they are viewing, IMMEDIATELY use get_logs tool - do NOT explain generically
- ONLY analyze log data after successfully fetching it via get_logs
- If log fetch succeeds, provide evidence-based analysis from the returned data
- If log fetch fails, explain the specific failure (not found / permission denied / datasource missing)
- NEVER invent log messages, error patterns, or log volumes
- Log analysis MUST include ONLY data from the tool response:
  * Total log count and time range
  * Error count and percentage
  * Warning count and percentage
  * Log level distribution
  * Trend (increasing/decreasing/stable) with change percentage
  * Top error messages (actual messages from logs)
  * Top labels (service, pod, namespace, level, etc.)
- If user asks "what's wrong with these logs" but does NOT provide query/datasource, use get_logs with panelId
- After fetching logs, structure response as:
  ## Current Context
  [time range, active label filters, log query]

  ## What the Logs Show
  [factual summary from tool response]
  - Total: X log lines
  - Errors: Y (Z%)
  - Trend: increasing/decreasing/stable (+/-N%)

  ## Notable Patterns
  [top error messages from actual logs]
  - "Error connecting to database"
  - "Timeout waiting for response"
  [etc.]

  ## Log Distribution
  [top labels: services, pods, namespaces]
  - Top service: api-gateway (45% of logs)
  - Top pod: api-gateway-7d8f9 (23% of logs)

  ## Assessment
  - Status: Normal/Abnormal (based on error rate and trends)
  - Severity: Low/Medium/High
  - Impact: [what this means for the system]

  ## Next Actions
  1. [Specific investigation step based on actual log data]
  2. [Follow-up query to drill down]
  3. [Mitigation if needed]

  ## Confidence: [High/Medium/Low]
  [Why this confidence level based on data quality]

Trace Analysis Rules (CRITICAL):
- When user provides a trace ID, IMMEDIATELY use get_trace_by_id tool - do NOT ask for queries or screenshots
- ONLY analyze trace data after successfully fetching it via get_trace_by_id
- If trace fetch succeeds, provide evidence-based analysis from the returned data
- If trace fetch fails, explain the specific failure (not found / permission denied / datasource missing)
- NEVER invent span names, services, durations, or error patterns
- Trace analysis MUST include ONLY data from the tool response:
  * Services involved (from traceStructure.services)
  * Root service and operation (from rootService/rootOperation)
  * Total span count and duration (from totalSpans/totalDuration)
  * Slowest spans (from slowestSpans array)
  * Errors (from errors array with specific status codes)
- If user asks "what's wrong with trace X" but does NOT provide datasource, ask ONLY for datasource name
- After fetching trace, structure response as:
  ## Trace Summary
  [factual summary from tool response]

  ## Service Flow
  [service → service chain from traceStructure]

  ## Performance Breakdown
  [slowest spans with actual durations]

  ## Issues Found
  [errors with service/operation/status from errors array]

  ## Next Actions
  [3-5 concrete investigative steps based on actual data]

Explicitly forbidden:
- "Typically X%..." or "Usually..." without evidence
- "Panel X should..." without query evidence
- Absolute thresholds (percentages, numbers) unless derived from query or user-provided
- Inventing metric names not present in context
- More than ONE clarifying question per response
- Asking for screenshots, files, exports, or "what the graph looks like"
- Continuing investigations without evidence
- Speculating about metrics you haven't seen
- Assuming what "normal" looks like without data
- Analyzing traces WITHOUT calling get_trace_by_id first
- Inventing trace spans, services, or error patterns
- Using "Goal / Plan / Evidence / Gaps" structure for trace analysis (use structured format above)

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
- Did I cite sources for all statements (query, context, user input)?
- Did I avoid inventing metrics, thresholds, or relationships?
- If I suggested something risky, did I include rollback + verify?
- Did I limit clarifying questions to ONE max?
- Did I include a confidence indicator at the end?

Response format requirements:
1) **Confidence Indicator** (MANDATORY at end of every response):
   - "Confidence: High" – Based directly on panel query + datasource + time range
   - "Confidence: Medium" – Based on partial context with stated assumptions
   - "Confidence: Low" – Conceptual explanation only (missing key data)

2) **Trend-Based Reasoning** (REQUIRED):
   - Prefer slopes (increasing/decreasing/flat), ratios (X ÷ Y), and correlations
   - Avoid absolute thresholds unless derived from query or user-provided
   - Example: "Memory usage increasing ~5% per hour" not "Should be under 70%"

3) **"What this is NOT" Guardrail** (when relevant):
   - If metric is commonly confused, explicitly state what it does NOT measure
   - Example: "This does NOT show JVM heap - it reflects OS-level process memory"

4) **Explicit Uncertainty** (when context incomplete):
   - State "I might be wrong because..." when missing critical data
   - Example: "I might be wrong because I can't see how total memory is calculated"

5) **One-Question Rule** (STRICT):
   - Ask at most ONE clarifying question, only if it blocks correctness
   - Example: "I can't explain this without the panel query. Can you paste it?"

6) **Evidence-Gated Investigations** (CRITICAL FOR TROUBLESHOOTING):
   - You may ONLY reason from: panel queries, metric results, log results, explicit absence statements
   - If you have not seen a metric, do NOT mention it
   - If you have not seen an error, do NOT assume failure
   - If you have not seen a trend, do NOT describe one
   - If required evidence is missing, STOP EARLY and ask for ONE concrete Grafana input
   - NEVER ask for screenshots or files - Grafana queries are the source of truth
   - An investigation is FORBIDDEN without at least one of:
     * Dashboard context with queries
     * Panel queries
     * Metric/log query results
     * Explicit statement of signal absence (e.g. "no errors observed")`

const DESIGN_MODE_PROMPT = `## Design Mode

You are in Design mode for dashboard and observability design tasks.

**Key Rules**:
1. **State Assumptions** - Clearly label what you assume vs what you know
2. **Mark as Proposals** - Use "I propose..." or "Consider..." language
3. **Offer Validation** - Suggest how to validate each design choice
4. **Reference Best Practices** - Cite observability patterns (RED, USE, Golden Signals)
5. **Controlled Speculation** - Distinguish proposals from facts

**Design Principles**:
- Prefer actionable dashboards over vanity metrics
- Follow signal hierarchy: Errors → Latency → Traffic → Saturation
- Use consistent naming conventions and units
- Include panel descriptions for context
- Link related dashboards

**Response Structure for Design Tasks**:
1. **Current State** - What exists now (from context)
2. **Design Proposal** - What you suggest
3. **Rationale** - Why this design works
4. **Validation** - How to verify effectiveness
5. **Alternatives** - Other approaches considered

**When Proposing Panels**:
- Suggest specific metric names (if metadata available)
- Recommend visualization type with justification
- Propose thresholds based on SLO/SLI patterns
- Include time range recommendations`

// buildBasePrompt constructs the base system prompt with optional extensions
func buildBasePrompt(settings *Settings, contextManager *contextmgr.Manager, mode string) string {
	basePrompt := BASE_SYSTEM_PROMPT

	if settings != nil && settings.OtelEnforcement.Enabled {
		otelContext := buildOtelEnforcementContext(settings.OtelEnforcement)
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, otelContext)
	}

	if contextManager != nil {
		dsMetadata := buildDatasourceMetadataContext(contextManager)
		if dsMetadata != "" {
			basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, dsMetadata)
		}
	}

	if mode == "design" {
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, DESIGN_MODE_PROMPT)
	}

	return basePrompt
}

// SkillPrompt is kept for backward compatibility but skills now live in .md files
// This type can be removed once all consumers migrate to the registry
type SkillPrompt struct {
	Name             string
	TaskInstructions string
}

// SkillPrompts is deprecated - skills are now loaded from .md files via SkillRegistry
// This empty map is kept for backward compatibility during migration
var SkillPrompts = map[string]SkillPrompt{}


type AssistantContext struct {
	Dashboard    *DashboardContext  `json:"dashboard,omitempty"`
	Panel        *PanelContext      `json:"panel,omitempty"`
	TimeRange    *TimeRange         `json:"timeRange,omitempty"`
	TemplateVars []TemplateVariable `json:"templateVars,omitempty"`
	// Alert context - set by webhook handler for alert-triggered investigations
	AlertSource  string            `json:"alertSource,omitempty"`
	AlertLabels  map[string]string `json:"alertLabels,omitempty"`
}

type DashboardContext struct {
	UID    string         `json:"uid"`
	Title  string         `json:"title"`
	Tags   []string       `json:"tags,omitempty"`
	Panels []PanelContext `json:"panels,omitempty"`
}

type PanelContext struct {
	Title           string                 `json:"title"`
	Type            string                 `json:"type"`
	Description     string                 `json:"description,omitempty"`
	Targets         []QueryTarget          `json:"targets"`
	FieldConfig     map[string]interface{} `json:"fieldConfig,omitempty"`
	Transformations []interface{}          `json:"transformations,omitempty"`
}

type QueryTarget struct {
	RefID      string                 `json:"refId"`
	Expr       string                 `json:"expr,omitempty"`
	Query      string                 `json:"query,omitempty"`
	Datasource map[string]interface{} `json:"datasource,omitempty"`
}

type TemplateVariable struct {
	Name    string                 `json:"name"`
	Current map[string]interface{} `json:"current"`
}

func BuildSystemPrompt(skill string, context AssistantContext, settings *Settings, contextManager *contextmgr.Manager, mode string) string {
	return BuildSystemPromptWithRegistry(skill, context, settings, contextManager, mode, nil)
}

// BuildSystemPromptWithRegistry builds the system prompt using the skill registry
func BuildSystemPromptWithRegistry(skill string, context AssistantContext, settings *Settings, contextManager *contextmgr.Manager, mode string, registry *skills.SkillRegistry) string {
	var taskInstructions string

	// Get skill content from registry
	if registry != nil && skill != "" {
		taskInstructions = registry.GetContent(skill)
	}

	// If no skill content found, return base prompt
	if taskInstructions == "" {
		return buildBasePrompt(settings, contextManager, mode)
	}

	// Handle template placeholder for generate_query skill
	if skill == "generate_query" {
		queryLang := detectQueryLanguage(context)
		if registry != nil {
			if meta := registry.GetMetadata(skill); meta != nil && meta.SupportsTemplate {
				taskInstructions = fmt.Sprintf(taskInstructions, queryLang, queryLang, queryLang)
			}
		}
	}

	basePrompt := BASE_SYSTEM_PROMPT

	if settings != nil && settings.OtelEnforcement.Enabled {
		otelContext := buildOtelEnforcementContext(settings.OtelEnforcement)
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, otelContext)
	}

	if contextManager != nil {
		dsMetadata := buildDatasourceMetadataContext(contextManager)
		if dsMetadata != "" {
			basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, dsMetadata)
		}
	}

	if mode == "design" {
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, DESIGN_MODE_PROMPT)

		designSkills := map[string]bool{
			"design_dashboard": true,
			"review_dashboard": true,
		}

		if designSkills[skill] && contextManager != nil {
			referenceDashboards := contextManager.GetReferenceDashboards()
			if len(referenceDashboards) > 0 {
				dashboardContext := buildReferenceDashboardContext(referenceDashboards)
				basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, dashboardContext)
			}
		}
	}

	return fmt.Sprintf("%s\n\n---\n\n%s", basePrompt, taskInstructions)
}

func buildOtelEnforcementContext(otel OtelEnforcementSettings) string {
	context := "## OpenTelemetry Scope Enforcement\n\n"
	context += "**CRITICAL**: This Grafana instance has OpenTelemetry scope enforcement enabled.\n\n"

	context += "**Required Actions for Query Generation**:\n"

	if otel.RequireServiceName {
		context += "- **MUST include `serviceName`** parameter when calling query generation tools (create_promql_query, create_logql_query)\n"
		if otel.DefaultServiceName != "" {
			context += fmt.Sprintf("- **Default service**: `%s` (used if not specified)\n", otel.DefaultServiceName)
		} else {
			context += "- **No default service** - queries without service.name will be REJECTED\n"
		}
	}

	if otel.RequireEnvironmentName {
		context += "- **MUST include `environmentName`** parameter when calling query generation tools\n"
		if otel.DefaultEnvironmentName != "" {
			context += fmt.Sprintf("- **Default environment**: `%s` (used if not specified)\n", otel.DefaultEnvironmentName)
		} else {
			context += "- **No default environment** - queries without deployment.environment.name will be REJECTED\n"
		}
	}

	context += "\n**How to Extract OTel Values**:\n"
	context += "1. Check dashboard title, panel names, and queries for service names\n"
	context += "2. Look for `service_name`, `service.name`, or similar labels in existing queries\n"
	context += "3. Check `deployment_environment_name` or `environment` labels\n"
	context += "4. If user mentions a specific service/environment, use that\n"
	context += "5. If unclear, ASK the user which service/environment to query\n"

	if otel.RejectIfNoScope {
		context += "\n**Strict Mode**: Queries without proper OTel scoping will be REJECTED.\n"
	} else {
		context += "\n**Fallback Mode**: Queries without OTel attributes will use defaults.\n"
	}

	return context
}

func buildDatasourceMetadataContext(contextManager *contextmgr.Manager) string {
	if contextManager == nil {
		return ""
	}

	obsContext := contextManager.GetContext()
	if obsContext == nil {
		return ""
	}

	var metadata strings.Builder
	metadata.WriteString("## Datasource Metadata (GROUND TRUTH)\n\n")
	metadata.WriteString("**CRITICAL**: Only use metrics, labels, and services from this list. Do NOT invent names.\n\n")

	if obsContext.Metrics != nil && len(obsContext.Metrics.MetricNames) > 0 {
		metadata.WriteString("**Available Metrics** (from Prometheus/Mimir):\n")

		maxMetrics := 50
		for i, metric := range obsContext.Metrics.MetricNames {
			if i >= maxMetrics {
				metadata.WriteString(fmt.Sprintf("... and %d more metrics\n", len(obsContext.Metrics.MetricNames)-maxMetrics))
				break
			}
			metadata.WriteString(fmt.Sprintf("- %s\n", metric))
		}

		if len(obsContext.Metrics.Labels) > 0 {
			metadata.WriteString("\n**Available Labels**:\n")
			maxLabels := 20
			for i, label := range obsContext.Metrics.Labels {
				if i >= maxLabels {
					break
				}
				metadata.WriteString(fmt.Sprintf("- %s", label))

				if values, ok := obsContext.Metrics.LabelValues[label]; ok && len(values) > 0 {
					sampleCount := 3
					if len(values) < sampleCount {
						sampleCount = len(values)
					}
					metadata.WriteString(fmt.Sprintf(" (examples: %s)", strings.Join(values[:sampleCount], ", ")))
				}
				metadata.WriteString("\n")
			}
		}
	}

	if obsContext.Logs != nil && len(obsContext.Logs.Streams) > 0 {
		metadata.WriteString("\n**Available Log Streams** (from Loki):\n")
		maxStreams := 20
		for i, stream := range obsContext.Logs.Streams {
			if i >= maxStreams {
				metadata.WriteString(fmt.Sprintf("... and %d more streams\n", len(obsContext.Logs.Streams)-maxStreams))
				break
			}
			metadata.WriteString(fmt.Sprintf("- %s\n", stream))
		}
	}

	if obsContext.Traces != nil && len(obsContext.Traces.Services) > 0 {
		metadata.WriteString("\n**Available Services** (from Tempo/Jaeger):\n")
		for i, service := range obsContext.Traces.Services {
			if i >= 30 {
				break
			}
			metadata.WriteString(fmt.Sprintf("- %s\n", service))
		}
	}

	metadata.WriteString("\n**Usage Rule**: If a metric/label/service is NOT in this list, do NOT use it. Ask for clarification instead.\n")

	return metadata.String()
}

func buildReferenceDashboardContext(dashboards map[string]*contextmgr.DashboardResponse) string {
	if len(dashboards) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("## Reference Dashboards (Examples)\n\n")
	context.WriteString("**Use these as inspiration for design patterns**:\n\n")

	for uid, dashboard := range dashboards {
		context.WriteString(fmt.Sprintf("### %s (UID: %s)\n", dashboard.Dashboard.Title, uid))

		if len(dashboard.Dashboard.Tags) > 0 {
			context.WriteString(fmt.Sprintf("**Tags**: %s\n", strings.Join(dashboard.Dashboard.Tags, ", ")))
		}

		panelCount := len(dashboard.Dashboard.Panels)
		context.WriteString(fmt.Sprintf("**Panels**: %d total\n", panelCount))

		maxPanels := 5
		if panelCount < maxPanels {
			maxPanels = panelCount
		}

		for i := 0; i < maxPanels; i++ {
			panelJSON, _ := json.Marshal(dashboard.Dashboard.Panels[i])
			var panelMap map[string]interface{}
			json.Unmarshal(panelJSON, &panelMap)

			title, _ := panelMap["title"].(string)
			panelType, _ := panelMap["type"].(string)

			context.WriteString(fmt.Sprintf("  - %s (%s)\n", title, panelType))
		}

		if panelCount > maxPanels {
			context.WriteString(fmt.Sprintf("  ... and %d more panels\n", panelCount-maxPanels))
		}

		context.WriteString("\n")
	}

	context.WriteString("**Note**: Use these patterns as inspiration, not templates. Adapt to user needs.\n")

	return context.String()
}

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
	case "incident_investigate":
		return buildIncidentInvestigatePrompt(userMessage, context)
	default:
		return userMessage
	}
}

// buildIncidentInvestigatePrompt formats alert context and user message for incident investigation
func buildIncidentInvestigatePrompt(userMessage string, context AssistantContext) string {
	var prompt strings.Builder

	prompt.WriteString("--- INCIDENT INVESTIGATION REQUEST ---\n\n")

	// If we have alert context (from webhook), format it prominently
	if context.AlertSource != "" {
		prompt.WriteString("**ALERT TRIGGERED INVESTIGATION**\n\n")
		prompt.WriteString(fmt.Sprintf("**Alert Source**: %s\n", context.AlertSource))

		if len(context.AlertLabels) > 0 {
			prompt.WriteString("**Alert Labels**:\n")
			// Extract key labels for prominent display
			if alertName, ok := context.AlertLabels["alertname"]; ok {
				prompt.WriteString(fmt.Sprintf("- **Alert Name**: %s\n", alertName))
			}
			if severity, ok := context.AlertLabels["severity"]; ok {
				prompt.WriteString(fmt.Sprintf("- **Severity**: %s\n", severity))
			}
			if service, ok := context.AlertLabels["service"]; ok {
				prompt.WriteString(fmt.Sprintf("- **Service**: %s\n", service))
			}
			// Other labels
			for k, v := range context.AlertLabels {
				if k != "alertname" && k != "severity" && k != "service" {
					prompt.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
				}
			}
		}
		prompt.WriteString("\n")
	}

	// User's question/message
	prompt.WriteString(fmt.Sprintf("**User Issue**: %s\n\n", userMessage))

	// Add dashboard context if available
	if context.Dashboard != nil {
		prompt.WriteString("**Dashboard Context Available**:\n")
		prompt.WriteString(fmt.Sprintf("- Dashboard: \"%s\"\n", context.Dashboard.Title))
		if len(context.Dashboard.Panels) > 0 {
			prompt.WriteString(fmt.Sprintf("- Panels: %d available\n", len(context.Dashboard.Panels)))
		}
		prompt.WriteString("\n")
	}

	// Time range if available
	if context.TimeRange != nil {
		prompt.WriteString(fmt.Sprintf("**Time Range**: %s to %s\n\n", context.TimeRange.From, context.TimeRange.To))
	} else {
		prompt.WriteString("**Time Range**: Not specified - default to last 1 hour\n\n")
	}

	prompt.WriteString("--- INVESTIGATION INSTRUCTIONS ---\n")
	prompt.WriteString("1. Start by calling get_firing_alerts to understand current alert state\n")
	prompt.WriteString("2. Call execute_promql AND execute_logql in PARALLEL for affected services\n")
	prompt.WriteString("3. Correlate findings and identify root cause\n")
	prompt.WriteString("4. Follow the MANDATORY OUTPUT FORMAT from your instructions\n")

	return prompt.String()
}

func buildExplainPanelPrompt(context AssistantContext) string {
	if context.Panel == nil {
		return "No panel context available."
	}

	panel := context.Panel

	prompt := "--- AVAILABLE CONTEXT ---\n\n"

	prompt += fmt.Sprintf("**Panel Title**: %s\n**Panel Type**: %s\n", panel.Title, panel.Type)

	if panel.Description != "" {
		prompt += fmt.Sprintf("**Description**: %s\n", panel.Description)
	}

	hasQueries := false
	prompt += "\n**Queries**:\n"
	for _, target := range panel.Targets {
		query := target.Expr
		if query == "" {
			query = target.Query
		}
		if query != "" {
			hasQueries = true
			prompt += fmt.Sprintf("- Query %s: %s\n", target.RefID, query)

			if dsType, ok := target.Datasource["type"].(string); ok {
				prompt += fmt.Sprintf("  Datasource: %s\n", dsType)
			}
		}
	}

	if !hasQueries {
		prompt += "NO QUERIES PROVIDED - Cannot explain calculation\n"
	}

	if fieldConfig, ok := panel.FieldConfig["defaults"].(map[string]interface{}); ok {
		if unit, ok := fieldConfig["unit"].(string); ok && unit != "" {
			prompt += fmt.Sprintf("\n**Unit**: %s\n", unit)
		}
		if thresholds, ok := fieldConfig["thresholds"].(map[string]interface{}); ok {
			prompt += "\n**Panel Thresholds** (configured by user):\n"
			if steps, ok := thresholds["steps"].([]interface{}); ok {
				for _, step := range steps {
					if stepMap, ok := step.(map[string]interface{}); ok {
						value, _ := stepMap["value"]
						color, _ := stepMap["color"]
						prompt += fmt.Sprintf("- Value: %v, Color: %v\n", value, color)
					}
				}
			}
		}
	}

	if len(panel.Transformations) > 0 {
		prompt += fmt.Sprintf("\n**Transformations**: %d applied\n", len(panel.Transformations))
	}

	prompt += "\n--- UNKNOWN CONTEXT ---\n"
	prompt += "Anything not listed above is UNKNOWN. Do not reference:\n"
	prompt += "- Other panels (unless their indices are provided)\n"
	prompt += "- Metrics not in the queries above\n"
	prompt += "- Thresholds not in panel configuration\n"
	prompt += "- Label values not in the queries\n"

	return prompt
}

func buildGenerateQueryPrompt(userMessage string, context AssistantContext) string {
	queryLang := detectQueryLanguage(context)
	prompt := fmt.Sprintf("Generate a %s query for: \"%s\"\n", queryLang, userMessage)

	if len(context.TemplateVars) > 0 {
		prompt += "\nAvailable template variables:\n"
		for _, v := range context.TemplateVars {
			if currentValue, ok := v.Current["value"]; ok {
				prompt += fmt.Sprintf("- $%s = %v\n", v.Name, currentValue)
			}
		}
	}

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
	prompt := "--- AVAILABLE CONTEXT ---\n\n"
	prompt += fmt.Sprintf("**User Issue**: \"%s\"\n\n", userMessage)

	if context.Dashboard != nil {
		prompt += fmt.Sprintf("**Current Dashboard**: \"%s\"\n", context.Dashboard.Title)

		if len(context.Dashboard.Panels) > 0 {
			prompt += "\n**Available Panels**:\n"
			for idx, panel := range context.Dashboard.Panels {
				prompt += fmt.Sprintf("%d. \"%s\" (%s)\n", idx, panel.Title, panel.Type)
				for _, target := range panel.Targets {
					query := target.Expr
					if query == "" {
						query = target.Query
					}
					if query != "" {
						prompt += fmt.Sprintf("   Query: %s\n", query)
					}
				}
			}
		}
	}

	if context.Panel != nil {
		prompt += fmt.Sprintf("\n**Focused Panel**: \"%s\" (%s)\n", context.Panel.Title, context.Panel.Type)
		for _, target := range context.Panel.Targets {
			query := target.Expr
			if query == "" {
				query = target.Query
			}
			if query != "" {
				prompt += fmt.Sprintf("   Query: %s\n", query)
			}
		}
	}

	if context.TimeRange != nil {
		prompt += fmt.Sprintf("\n**Time Range**: %s to %s\n", context.TimeRange.From, context.TimeRange.To)
	}

	if len(context.TemplateVars) > 0 {
		prompt += "\n**Template Variables**:\n"
		for _, v := range context.TemplateVars {
			if currentValue, ok := v.Current["value"]; ok {
				prompt += fmt.Sprintf("- $%s = %v\n", v.Name, currentValue)
			}
		}
	}

	prompt += "\n--- UNKNOWN CONTEXT ---\n"
	prompt += "Do NOT reference metrics, panels, or thresholds not listed above.\n"

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

	if context.TimeRange != nil {
		prompt += "\n## Time Range\n"
		prompt += fmt.Sprintf("Looking at data from %s to %s\n", context.TimeRange.From, context.TimeRange.To)
	}

	if len(context.TemplateVars) > 0 {
		prompt += "\n## Template Variables\n"
		for _, v := range context.TemplateVars {
			if currentValue, ok := v.Current["value"]; ok {
				prompt += fmt.Sprintf("- $%s = %v\n", v.Name, currentValue)
			}
		}
	}

	if context.Panel != nil {
		prompt += "\n## Currently Focused Panel\n"
		prompt += fmt.Sprintf("The user has \"%s\" (%s) in focus.\n", context.Panel.Title, context.Panel.Type)
	}

	return prompt
}

func detectQueryLanguage(context AssistantContext) string {
	if context.Panel == nil || len(context.Panel.Targets) == 0 {
		return "PromQL" 
	}

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

	return "PromQL" 
}

type skillDetectionScore struct {
	skill      string
	confidence int
	reason     string
}

func DetectSkill(userInput string, context AssistantContext) string {
	return DetectSkillWithRegistry(userInput, context, nil)
}

// DetectSkillWithRegistry detects the appropriate skill using the registry for scoring
func DetectSkillWithRegistry(userInput string, context AssistantContext, registry *skills.SkillRegistry) string {
	// Minimum confidence threshold - below this, fall through to base prompt
	const minConfidenceThreshold = 40

	hasAlertSource := context.AlertSource != ""
	hasDashboardContext := context.Dashboard != nil
	hasPanelContext := context.Panel != nil

	// Try registry-based scoring first
	if registry != nil && registry.IsLoaded() {
		bestSkill, bestScore := registry.ScoreInput(userInput, hasAlertSource, hasDashboardContext, hasPanelContext)

		if bestScore >= minConfidenceThreshold {
			backend.Logger.Debug("Skill auto-detected via registry",
				"skill", bestSkill,
				"confidence", bestScore,
				"inputLength", len(userInput),
			)
			return bestSkill
		}

		if bestScore > 0 {
			backend.Logger.Debug("Skill detection below threshold, using base prompt",
				"maxConfidence", bestScore,
				"threshold", minConfidenceThreshold,
				"inputLength", len(userInput),
			)
		}
		return "" // Fall through to base prompt
	}

	// Fallback to hardcoded scoring if registry not available
	input := strings.ToLower(userInput)
	scores := []skillDetectionScore{}

	// Incident investigation - check first as it can have highest priority (200 for alert-triggered)
	incidentScore := scoreIncidentInvestigation(input, context)
	if incidentScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "incident_investigate",
			confidence: incidentScore,
			reason:     "Incident investigation keywords or alert context detected",
		})
	}

	// Alert triage (stub for S3-3)
	alertTriageScore := scoreAlertTriage(input, context)
	if alertTriageScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "alert_triage",
			confidence: alertTriageScore,
			reason:     "Alert triage keywords detected",
		})
	}

	if context.Dashboard != nil {
		reviewScore := scoreReviewDashboard(input, context)
		if reviewScore > 0 {
			scores = append(scores, skillDetectionScore{
				skill:      "review_dashboard",
				confidence: reviewScore,
				reason:     "Dashboard review/quality assessment keywords detected",
			})
		}
	}

	designScore := scoreDesignDashboard(input, context)
	if designScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "design_dashboard",
			confidence: designScore,
			reason:     "Dashboard design keywords detected",
		})
	}

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

	queryScore := scoreGenerateQuery(input, context)
	if queryScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "generate_query",
			confidence: queryScore,
			reason:     "Query generation keywords detected",
		})
	}

	troubleshootScore := scoreTroubleshoot(input, context)
	if troubleshootScore > 0 {
		scores = append(scores, skillDetectionScore{
			skill:      "troubleshoot",
			confidence: troubleshootScore,
			reason:     "Troubleshooting/debugging keywords detected",
		})
	}

	if len(scores) == 0 {
		return "" // Fall through to base prompt / general chat
	}

	maxScore := scores[0]
	for _, score := range scores[1:] {
		if score.confidence > maxScore.confidence {
			maxScore = score
		}
	}

	// Apply minimum confidence threshold
	if maxScore.confidence < minConfidenceThreshold {
		backend.Logger.Debug("Skill detection below threshold, using base prompt",
			"maxConfidence", maxScore.confidence,
			"threshold", minConfidenceThreshold,
			"inputLength", len(userInput),
		)
		return "" // Fall through to base prompt / general chat
	}

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

func scoreReviewDashboard(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"review this dashboard", "is this dashboard good", "dashboard quality",
		"check this dashboard", "audit this dashboard", "dashboard issues",
		"what's wrong with this dashboard", "improve this dashboard",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 100
		}
	}

	mediumConfidencePatterns := []string{
		"review", "quality", "best practices", "issues",
		"problems", "mistakes", "improvements", "suggestions",
	}
	reviewKeywordCount := 0
	for _, pattern := range mediumConfidencePatterns {
		if contains(input, pattern) {
			reviewKeywordCount++
		}
	}
	if reviewKeywordCount >= 1 && contains(input, "dashboard") {
		score += 60
	}

	reviewAspects := []string{
		"query quality", "label consistency", "cardinality",
		"aggregation", "visualization", "confusing",
	}
	for _, aspect := range reviewAspects {
		if contains(input, aspect) {
			score += 40
		}
	}

	if contains(input, "what do i see") || contains(input, "what is this") {
		score -= 50
	}

	return score
}

func scoreDesignDashboard(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"create dashboard", "design dashboard", "build dashboard",
		"new dashboard", "dashboard for", "dashboard to monitor",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 100
		}
	}

	mediumConfidencePatterns := []string{
		"suggest panels", "what panels", "dashboard layout",
		"how to visualize", "best way to show",
	}
	for _, pattern := range mediumConfidencePatterns {
		if contains(input, pattern) {
			score += 70
		}
	}

	designKeywords := []string{"panels", "layout", "visualize", "charts", "graphs"}
	for _, keyword := range designKeywords {
		if contains(input, keyword) {
			score += 15
		}
	}

	return score
}

func scoreAnalyzeDashboard(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"what do i see", "what am i seeing", "what am i looking at", "describe this dashboard",
		"what is this dashboard", "what's on this dashboard",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	mediumConfidencePatterns := []string{
		"analyze this", "overview", "understand this",
		"show me this", "explain this dashboard",
	}
	for _, pattern := range mediumConfidencePatterns {
		if contains(input, pattern) {
			score += 50
		}
	}

	if contains(input, "dashboard") {
		score += 30
		if contains(input, "this dashboard") {
			score += 20
		}
	}

	if contains(input, "review") || contains(input, "quality") || contains(input, "issues") {
		score -= 50
	}

	if contains(input, "query") || contains(input, "promql") || contains(input, "logql") {
		score -= 40
	}

	if contains(input, "panel") || contains(input, "graph") || contains(input, "chart") {
		score -= 30
	}

	return score
}

func scoreExplainPanel(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"this panel", "this graph", "this chart",
		"what does this show", "what does this display",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	if contains(input, "explain") {
		if contains(input, "panel") || contains(input, "graph") || contains(input, "chart") {
			score += 60
		}
	}

	panelKeywords := []string{"panel", "graph", "chart", "visualization", "metric"}
	for _, keyword := range panelKeywords {
		if contains(input, keyword) {
			score += 20
		}
	}

	if context.Panel != nil {
		score += 15
	}

	if contains(input, "dashboard") && !contains(input, "panel") {
		score -= 40
	}

	return score
}

func scoreGenerateQuery(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"create a query", "write a query", "generate a query",
		"query for", "give me a query", "show me a query",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	queryLanguages := []string{"promql", "logql", "traceql"}
	for _, lang := range queryLanguages {
		if contains(input, lang) {
			score += 70
		}
	}

	functionKeywords := []string{
		"rate", "increase", "sum", "avg", "count", "histogram",
		"quantile", "topk", "bottomk", "max", "min",
	}
	for _, fn := range functionKeywords {
		if contains(input, fn) {
			score += 15 
		}
	}

	actionVerbs := []string{
		"how do i query", "how to query", "how can i get",
		"show me how to", "need a query",
	}
	for _, verb := range actionVerbs {
		if contains(input, verb) {
			score += 40
		}
	}

	dataKeywords := []string{"metrics", "logs", "traces", "errors", "latency", "throughput"}
	for _, keyword := range dataKeywords {
		if contains(input, keyword) {
			score += 10
		}
	}

	if contains(input, "explain") || contains(input, "what does") {
		score -= 30
	}

	return score
}

func scoreTroubleshoot(input string, context AssistantContext) int {
	score := 0

	highConfidencePatterns := []string{
		"not working", "doesn't work", "failing", "broken",
		"troubleshoot", "debug", "investigate",
	}
	for _, pattern := range highConfidencePatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	problemKeywords := []string{
		"error", "errors", "issue", "problem", "wrong",
		"failed", "failure", "down", "outage",
	}
	for _, keyword := range problemKeywords {
		if contains(input, keyword) {
			score += 30 
		}
	}

	questionPatterns := []string{
		"why is", "why are", "what's wrong", "what went wrong",
		"how do i fix", "how to fix", "what happened",
	}
	for _, pattern := range questionPatterns {
		if contains(input, pattern) {
			score += 40
		}
	}

	symptomKeywords := []string{
		"slow", "high", "spike", "drop", "missing", "timeout",
		"latency", "degraded", "anomaly", "unusual",
	}
	for _, keyword := range symptomKeywords {
		if contains(input, keyword) {
			score += 10
		}
	}

	if context.Dashboard != nil {
		dashboardTitle := strings.ToLower(context.Dashboard.Title)
		if contains(dashboardTitle, "error") || contains(dashboardTitle, "debug") {
			score += 15
		}
	}

	return score
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// scoreIncidentInvestigation scores input for incident investigation skill
func scoreIncidentInvestigation(input string, ctx AssistantContext) int {
	score := 0

	// AlertSource set = always wins (triggered by webhook)
	if ctx.AlertSource != "" {
		return 200
	}

	// High-signal phrases: +100 each
	highSignalPhrases := []string{
		"root cause", "why is", "down",
		"sla breach", "outage", "p0", "p1", "pager", "incident",
	}
	for _, phrase := range highSignalPhrases {
		if contains(input, phrase) {
			score += 100
		}
	}

	// Service + symptom patterns: +80 each
	serviceSymptomPatterns := []string{
		"returning 500", "returning 503", "returning 502",
		"error spike", "connection refused", "latency spike",
		"timeout", "failing requests", "high error rate",
		"service unavailable", "connection timeout",
	}
	for _, pattern := range serviceSymptomPatterns {
		if contains(input, pattern) {
			score += 80
		}
	}

	// General incident words: +20 each
	generalIncidentWords := []string{
		"broken", "failing", "spike", "crash", "crashed",
		"degraded", "degradation", "intermittent", "flapping",
		"alerting", "paging", "oncall",
	}
	for _, word := range generalIncidentWords {
		if contains(input, word) {
			score += 20
		}
	}

	return score
}

// scoreAlertTriage is a stub for future alert triage skill (S3-3)
func scoreAlertTriage(input string, ctx AssistantContext) int {
	// TODO: Implement in S3-3
	return 0
}

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
4) Keep steps atomic (30-90 seconds each). Max 5 steps (or 8 for incident investigations).
5) If a step could be risky, flag it clearly.

Signal Priority Order (for incident investigations):
1) **Alerts**: Start with get_firing_alerts to understand current state
2) **Errors**: Check error rates and status codes first (most direct impact)
3) **Latency**: Then check for latency degradation
4) **Logs**: Query logs for error details and stack traces
5) **Traffic**: Verify traffic patterns are normal
6) **Saturation**: Check resource limits (CPU, memory, connections)
7) **Traces**: If applicable, trace specific failing requests

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

func BuildPlanningPrompt(userMessage string, context AssistantContext) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("User request: %s\n\n", userMessage))

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

	if context.TimeRange != nil {
		prompt.WriteString(fmt.Sprintf("Time range: %s to %s\n\n", context.TimeRange.From, context.TimeRange.To))
	}

	prompt.WriteString("Create an execution plan that uses available context efficiently.")

	return prompt.String()
}

func parsePlanFromJSON(text string) (*ExecutionPlan, error) {
	var plan ExecutionPlan
	err := json.Unmarshal([]byte(text), &plan)
	if err == nil && len(plan.Steps) > 0 {
		return &plan, nil
	}

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

	return nil, fmt.Errorf("failed to parse plan from LLM response: %w", err)
}
