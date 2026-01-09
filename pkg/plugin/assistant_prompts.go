package plugin

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	contextmgr "github.com/jorgeancal/zagalin/pkg/plugin/context"
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

Evidence-first rules:
- Do NOT invent: metric names, label names, panel indices, thresholds, calculations, or relationships
- If panel query is provided, MUST reference it explicitly in your explanation
- If information is missing, state EXACTLY what's missing and ask ONE targeted question
- Prefer trend-based interpretation ("increasing", "stable") over fixed thresholds ("should be 10-30%")
- No long reasoning sections or report-style responses unless user requests it
- Every statement must cite: panel query, dashboard context, or explicit user input

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

const EVIDENCE_BASED_PROMPT = `## Evidence-Based Explanations

You are provided with **Evidence Packs** — structured summaries of observability data from Grafana.

**Evidence Pack Structure:**

### Metrics Evidence:
- datasource, time range, query (verbatim)
- unit, series count
- current value, min, max, avg
- trend (increasing/decreasing/flat/spiky)
- slope per hour
- data quality (good/gaps/no_data)
- optional: top contributors (topk by label)

### Logs Evidence:
- total count, rate, max rate
- trend (increasing/decreasing/flat)
- top labels (level, service, pod, etc.)
- notable messages (capped at 10, prioritized by severity)

### Traces Evidence:
- traceID, root service + operation
- total duration, span count, error span count
- top slowest spans (service, operation, duration)
- critical path (longest sequential chain)
- notable attributes (errors, service versions)

---

**Your Responsibilities (MANDATORY):**

1. **Explain ONLY what is present in the evidence pack**
   - Do NOT invent metrics, spans, labels, or values
   - Do NOT assume data not shown in evidence
   - Do NOT reference "typical" or "normal" values without baseline evidence

2. **State what you are basing the explanation on**
   - "Based on the metrics evidence showing..."
   - "The logs evidence indicates..."
   - "According to the trace evidence..."

3. **Include a confidence level at the end (MANDATORY)**
   - **Confidence: High** — Evidence clearly shows the situation
   - **Confidence: Medium** — Evidence suggests but is incomplete
   - **Confidence: Low** — Evidence is insufficient, more data needed

4. **Never ask for screenshots, files, or raw data exports**
   - Evidence packs are the source of truth
   - Grafana queries provided the evidence
   - Request additional queries if needed, not screenshots

5. **If more data is required, request ONE specific derived field only**
   - Example: "To confirm, I need evidence for 'topk by pod' for error_count"
   - Do NOT request multiple queries at once
   - Do NOT request raw time-series data

---

**Response Structure (REQUIRED):**

### **What the Evidence Shows**
[Summarize key facts from evidence pack]

### **Interpretation**
[Explain what these facts mean]
- Use trend language from evidence
- Reference specific values from evidence
- Cite datasource and query

### **Assessment**
- **Normal/Abnormal**: [Based on trends, not invented thresholds]
- **Severity**: [Low/Medium/High based on error counts, slopes, patterns]
- **Impact**: [What this means for the system]

### **Next Steps** (if abnormal)
1. [Specific action based on evidence]
2. [What to check next - must reference available context]
3. [Mitigation if needed]

### **What's Missing** (if evidence insufficient)
[Exactly what additional evidence would help - ONE query maximum]

### **Confidence: [High/Medium/Low]**
[Why this confidence level]

---

**Forbidden Actions:**

- Inventing metric names not in evidence
- Assuming thresholds ("should be under 50%") without baseline
- Claiming to have executed queries (evidence was pre-computed)
- Asking for screenshots or file exports
- Requesting multiple queries (max ONE if needed)
- Continuing investigation without evidence
- Speculating about data not in evidence pack
- Using "typically" or "usually" without evidence

---

**Example Good Response:**

"Based on the metrics evidence from datasource 'prometheus-prod', query 'rate(http_errors_total[5m])', I can see:

**What the Evidence Shows:**
- Current error rate: 3.2 errors/sec
- Trend: increasing
- Slope: +2.7 errors/sec per hour
- Time range: last 1 hour
- Top contributor: service_name='api-gateway' (87% of errors)

**Interpretation:**
The error rate is climbing steadily at ~2.7 errors/sec per hour. The api-gateway service is responsible for most errors, suggesting the issue is localized rather than system-wide.

**Assessment:**
- **Status**: Abnormal (increasing trend)
- **Severity**: Medium (not yet critical but worsening)
- **Impact**: Degraded user experience for API consumers

**Next Steps:**
1. Check api-gateway logs for errors in the same time range
2. Verify if traffic to api-gateway increased (could be legitimate scale)
3. Check api-gateway resource utilization (CPU/memory)

**Confidence: High** — Clear evidence from metrics with trend data and top contributors"

---

**Example Bad Response:**

"The error rate of 3.2/sec is too high. It should typically be under 0.5/sec. This indicates a database connection issue. You should check the connection pool settings and restart the service."

**Why bad:**
- Invented threshold ("should be under 0.5/sec") without evidence
- Assumed root cause ("database connection") not in evidence
- Didn't cite the evidence pack
- No confidence indicator
- Proposed actions without evidence support`

type SkillPrompt struct {
	Name             string
	TaskInstructions string
}

var SkillPrompts = map[string]SkillPrompt{
	"explain_panel": {
		Name: "explain_panel",
		TaskInstructions: `Your task is to explain what a panel shows using ONLY the information provided.

REQUIRED STRUCTURE:
1. **What this panel measures** (1-2 sentences)
   - Based on: [datasource type, time range, query provided]

2. **How the query works** (plain English)
   - Break down the calculation step by step
   - Reference actual metric names and labels from the query

3. **How to interpret the pattern** (behavioral guidance)
   - What does "normal" look like for this query?
   - What patterns would be concerning?
   - Use trend language: "increasing", "stable", "dropping" (NOT fixed thresholds)

4. **What to check next** (only if pattern is abnormal)
   - Max 3 concrete checks
   - Must reference available context or related panels

5. **One thing to confirm** (optional, max ONE question)
   - Only ask if critical information is missing

6. **Confidence: [High/Medium/Low]** (MANDATORY)
   - High: Based directly on panel query + datasource + time range
   - Medium: Based on partial context with stated assumptions
   - Low: Conceptual explanation (missing key data)

CONTEXT STRUCTURE:
You will receive context in this format:

--- AVAILABLE CONTEXT ---
[Panel query, datasource, time range, dashboard panels]

--- UNKNOWN CONTEXT ---
[Anything not explicitly provided above]

ENFORCEMENT RULES:
- Do NOT reference metrics not in the query
- Do NOT provide thresholds unless derived from query
- Do NOT say "typically" or "usually" without evidence
- Do NOT reference panels by index unless provided
- If query is missing, state: "No query provided - cannot explain calculation"

EXAMPLES OF GOOD RESPONSES:
- "This panel measures HTTP request rate using rate(http_requests_total[5m]). The query calculates..."
- "An increasing trend could indicate traffic growth or a potential attack..."
- "To confirm: is this measuring heap memory or total process memory?"

EXAMPLES OF BAD RESPONSES:
- "This should typically be around 10-30%"
- "Panel 2 shows..."  (without panel index in context)
- "The cpu_usage metric indicates..." (if query uses different metric name)
`,
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
		TaskInstructions: `Your task is to provide structured troubleshooting based ONLY on available context.

**EVIDENCE-GATED INVESTIGATION - CRITICAL**
Before proceeding, you MUST check:
- Do I have panel queries, metric results, or log results?
- Do I have explicit statements about signal presence/absence?

If NO evidence is available, you MUST:
- STOP immediately
- Ask ONE question: "Which dashboard or panel should I base the investigation on?"
- Do NOT speculate, assume, or invent symptoms

REQUIRED STRUCTURE:
0. **Evidence Check** (MANDATORY FIRST STEP)
   - Available signals: [List what you can see from queries/panels]
   - Missing signals: [List what's not visible]
   - Cannot proceed without: [If blocking evidence missing, stop here]

1. **Most likely causes** (ranked by probability, with correlation analysis)
   - Cause 1: [Description] (Confidence: High/Med/Low)
     - Correlation: Does this correlate with traffic? Deploys? GC? CPU throttling?
   - Cause 2: [Description] (Confidence: High/Med/Low)
     - Correlation: [Multi-signal correlation if available]
   - Cause 3: [Description] (Confidence: High/Med/Low)

2. **Investigation Memory** (what we know so far)
   - Ruled out: [What's been checked and eliminated]
   - Still unknown: [Key missing data]
   - Leading hypothesis: [Current most likely cause]

3. **Immediate actions (next 5-10 minutes)**
   - [ ] Check X using [specific query or panel]
   - [ ] Verify Y in [specific location]
   - [ ] Confirm Z by [specific action]

   **Queries to run now:**
   ` + "```promql or logql" + `
   [Specific query based on available metrics]
   ` + "```" + `
   **Expected result**: [What normal vs abnormal looks like]

4. **Follow-up actions (post-incident / next sprint)**
   - Alerting improvements
   - Dashboard additions
   - Architecture changes
   - Monitoring gaps to fill

5. **Next decision point**
   - If [condition], then [action]
   - If [condition], then [action]

6. **Confidence: [High/Medium/Low]**
   - Explain what data this is based on

CONTEXT STRUCTURE:
You will receive:
--- AVAILABLE CONTEXT ---
[Dashboard panels, time range, current metrics/logs]

--- DATASOURCE METADATA --- (if available)
[Known metrics, labels, services]

ENFORCEMENT RULES (NON-NEGOTIABLE):
- Do NOT invent metric names - only use metrics from context or metadata
- Do NOT provide absolute thresholds without evidence
- Do NOT reference panels without their indices
- Rank causes by evidence strength (High = strong evidence, Low = speculation)
- Every query MUST use metrics that exist in datasource metadata
- PREFER multi-signal correlation over single-metric analysis
- SEPARATE immediate actions from long-term improvements

**HALLUCINATION PREVENTION (HARD CONSTRAINTS):**
- If you have not SEEN a metric → do NOT mention it
- If you have not SEEN an error → do NOT assume failure
- If you have not SEEN a trend → do NOT describe one
- If evidence is MISSING → STOP and ask for it
- NEVER ask for screenshots, files, exports, or "what it looks like"
- Hypotheses MUST be traceable to specific queries or panels

EXAMPLE GOOD RESPONSE (WITH EVIDENCE):
"Evidence Check:
   - Available signals: Memory usage (panel 2), CPU (panel 1), request rate (panel 3)
   - Missing signals: Error rate, GC metrics, database connections

   Most likely causes:
   1. Memory leak in application (High) - heap usage increasing linearly per panel 2
      - Correlation: No correlation with traffic (panel 3 stable), started after deploy 3h ago
   2. Database connection pool exhaustion (Low) - cannot confirm without connection metrics

   Investigation Memory:
   - Ruled out: CPU throttling (panel 1 shows CPU steady at 40%)
   - Still unknown: GC pressure, error rate
   - Leading hypothesis: Memory leak in recent deploy

   Immediate actions:
   - [ ] Check panel 2 query for heap vs non-heap breakdown
   - [ ] Correlate panel 2 spike time with deploy logs

   Follow-up actions:
   - Add GC pause time panel
   - Add error rate monitoring"

EXAMPLE BAD RESPONSE (FORBIDDEN):
- "Can you share a screenshot of the dashboard?"
- "It looks like a memory leak" (without evidence)
- "Usually this happens when GC pauses increase" (haven't seen GC data)
- "Check if cpu_usage is above 80%" (invented threshold)
- Proceeding with investigation when no queries/panels available
`,
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
	"review_dashboard": {
		Name: "review_dashboard",
		TaskInstructions: `Your task is to review dashboard quality and identify potential issues.

REVIEW AREAS:
1. **Query Quality Issues**
   - Detect inconsistent aggregations (mixing sum/avg across similar metrics)
   - Flag misuse of rate() on gauges or counters without rate()
   - Warn about unbounded cardinality (missing label filters)
   - Identify expensive queries (long time ranges, high cardinality)

2. **Label & Naming Consistency**
   - Highlight inconsistent labels across panels (e.g., service vs service_name)
   - Flag panels using different metric naming conventions
   - Identify panels that should be related but use different label selectors

3. **Visualization & UX Issues**
   - Misleading units (bytes shown as numbers, percentages as decimals)
   - Inconsistent time windows across related panels
   - Missing legends or unclear panel titles
   - Graphs that should be stacked but aren't (or vice versa)

4. **Observability Gaps**
   - Missing error rate panels when latency is shown
   - Missing memory metrics when CPU is tracked
   - Logs without corresponding metrics
   - No SLI/SLO indicators for user-facing services

5. **Actionability**
   - Panels that show symptoms but not causes
   - Missing annotations for deploys/incidents
   - No links to runbooks or related dashboards

RESPONSE STRUCTURE:
**Dashboard Purpose**: [Brief description]

**Strengths**:
- [What's done well]

**Issues Found**:
1. [Issue category]: [Specific problem]
   - Impact: [Why this matters]
   - Fix: [How to improve]

2. [Next issue]...

**Priority Improvements**:
1. [Most important fix]
2. [Second priority]
3. [Third priority]

**Confidence: [High/Medium/Low]**
- Based on: [What data was available for review]

ENFORCEMENT RULES:
- Only flag issues with specific panel references
- Provide actionable fixes, not just criticisms
- Prioritize user impact over aesthetic preferences
- If query is missing, state: "Can't review query quality without panel queries"

EXAMPLE GOOD REVIEW:
"Issues Found:
   1. Query Quality: Panel 2 'Request Rate' uses sum(rate(...)) without by() clause
      - Impact: Loses breakdown by service, makes debugging harder
      - Fix: Add 'by (service)' to preserve service-level detail

   2. Label Consistency: Panels 1-3 use 'service' label, Panel 4 uses 'service_name'
      - Impact: Can't correlate metrics across panels
      - Fix: Standardize on 'service' label across all queries

   Confidence: High - Based on 4 panel queries reviewed"

EXAMPLE BAD REVIEW:
- "This dashboard could be better"
- "Consider improving the layout"
- "Queries might be slow" (without specific evidence)
`,
	},

	"design_dashboard": {
		Name: "design_dashboard",
		TaskInstructions: `Your task is to design a new dashboard or propose improvements.

REQUIRED STRUCTURE:
1. **Purpose** - What story should this dashboard tell?
2. **Audience** - Who will use it? (SRE, developers, business)
3. **Key Metrics** - What are the critical signals?
4. **Panel Layout** - Suggested visualization structure
5. **Design Rationale** - Why these choices?

DESIGN PRINCIPLES:
- Start with user goals: "What decisions does this enable?"
- Follow signal hierarchy: Errors → Latency → Traffic → Saturation
- Use consistent naming and units
- Group related panels
- Include descriptions for context

PANEL DESIGN:
For each panel, suggest:
- Title and description
- Metric or log query
- Visualization type (graph, stat, table, etc.)
- Thresholds (if applicable)
- Time range

VALIDATION:
- How will we know if this dashboard is useful?
- What questions should it answer?
- What actions should it enable?

**Confidence: [High/Medium/Low]**
- Based on: Available metrics, reference dashboards, best practices`,
	},
}

type AssistantContext struct {
	Dashboard     *DashboardContext  `json:"dashboard,omitempty"`
	Panel         *PanelContext      `json:"panel,omitempty"`
	TimeRange     *TimeRange         `json:"timeRange,omitempty"`
	TemplateVars  []TemplateVariable `json:"templateVars,omitempty"`
	EvidencePacks []*EvidencePack    `json:"evidencePacks,omitempty"`
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
	skillPrompt, exists := SkillPrompts[skill]
	if !exists {
		return BASE_SYSTEM_PROMPT
	}

	taskInstructions := skillPrompt.TaskInstructions

	if skill == "generate_query" {
		queryLang := detectQueryLanguage(context)
		taskInstructions = fmt.Sprintf(taskInstructions, queryLang, queryLang, queryLang)
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

	if len(context.EvidencePacks) > 0 {
		basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, EVIDENCE_BASED_PROMPT)
		evidenceContext := buildEvidencePackContext(context.EvidencePacks)
		if evidenceContext != "" {
			basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, evidenceContext)
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

func buildEvidencePackContext(evidencePacks []*EvidencePack) string {
	if len(evidencePacks) == 0 {
		return ""
	}

	var context strings.Builder
	context.WriteString("## Evidence Packs (Query Results)\n\n")
	context.WriteString("**You MUST base your explanation on this evidence ONLY**:\n\n")

	for i, pack := range evidencePacks {
		context.WriteString(fmt.Sprintf("### Evidence Pack %d: %s\n", i+1, pack.Type))
		context.WriteString(fmt.Sprintf("**Datasource**: %s\n", pack.Datasource))
		context.WriteString(fmt.Sprintf("**Query**: `%s`\n", pack.Query))
		context.WriteString(fmt.Sprintf("**Time Range**: %s to %s\n", pack.TimeRange.From, pack.TimeRange.To))
		context.WriteString(fmt.Sprintf("**Quality**: %s\n\n", pack.Quality))

		if pack.Metrics != nil {
			m := pack.Metrics
			context.WriteString("**Metrics Evidence**:\n")
			context.WriteString(fmt.Sprintf("- Series Count: %d\n", m.SeriesCount))
			context.WriteString(fmt.Sprintf("- Current: %.2f %s\n", m.Current, m.Unit))
			context.WriteString(fmt.Sprintf("- Min: %.2f, Max: %.2f, Avg: %.2f\n", m.Min, m.Max, m.Avg))
			context.WriteString(fmt.Sprintf("- Trend: %s (slope: %.2f/hour)\n", m.Trend, m.SlopePerHour))

			if len(m.TopContributors) > 0 {
				context.WriteString("- Top Contributors:\n")
				for _, contrib := range m.TopContributors {
					var labelPairs []string
					for k, v := range contrib.Labels {
						labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
					}
					context.WriteString(fmt.Sprintf("  - {%s} (%.2f)\n", strings.Join(labelPairs, ", "), contrib.Value))
				}
			}
		}

		if pack.Logs != nil {
			l := pack.Logs
			context.WriteString("**Logs Evidence**:\n")
			context.WriteString(fmt.Sprintf("- Total Count: %d\n", l.TotalCount))
			context.WriteString(fmt.Sprintf("- Rate: %.2f logs/sec (max: %.2f/sec)\n", l.Rate, l.MaxRate))
			context.WriteString(fmt.Sprintf("- Trend: %s\n", l.Trend))

			if len(l.TopLabels) > 0 {
				context.WriteString("- Top Labels:\n")
				for labelKey, values := range l.TopLabels {
					context.WriteString(fmt.Sprintf("  - %s: %s\n", labelKey, strings.Join(values, ", ")))
				}
			}

			if len(l.NotableMessages) > 0 {
				context.WriteString("- Notable Messages (sampled):\n")
				for _, msg := range l.NotableMessages {
					context.WriteString(fmt.Sprintf("  - %s\n", msg))
				}
			}
		}

		if pack.Traces != nil {
			t := pack.Traces
			context.WriteString("**Traces Evidence**:\n")
			context.WriteString(fmt.Sprintf("- Trace ID: %s\n", t.TraceID))
			context.WriteString(fmt.Sprintf("- Root Service: %s (%s)\n", t.RootService, t.RootOperation))
			context.WriteString(fmt.Sprintf("- Total Duration: %d ms\n", t.TotalDuration))
			context.WriteString(fmt.Sprintf("- Span Count: %d (errors: %d)\n", t.SpanCount, t.ErrorSpanCount))

			if len(t.TopSlowestSpans) > 0 {
				context.WriteString("- Slowest Spans:\n")
				for _, span := range t.TopSlowestSpans {
					context.WriteString(fmt.Sprintf("  - %s:%s (%d ms)\n", span.Service, span.Operation, span.Duration))
				}
			}

			if len(t.CriticalPath) > 0 {
				context.WriteString("- Critical Path:\n")
				for _, step := range t.CriticalPath {
					context.WriteString(fmt.Sprintf("  - %s\n", step))
				}
			}

			if len(t.NotableAttributes) > 0 {
				context.WriteString("- Notable Attributes:\n")
				for key, value := range t.NotableAttributes {
					context.WriteString(fmt.Sprintf("  - %s: %s\n", key, value))
				}
			}
		}

		context.WriteString("\n")
	}

	context.WriteString("**CRITICAL**: You MUST explain this evidence. Do NOT invent data. Include confidence level.\n")

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
	default:
		return userMessage
	}
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
	input := strings.ToLower(userInput)
	scores := []skillDetectionScore{}

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
		return "" 
	}

	maxScore := scores[0]
	for _, score := range scores[1:] {
		if score.confidence > maxScore.confidence {
			maxScore = score
		}
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
		"what do i see", "what am i looking at", "describe this dashboard",
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
