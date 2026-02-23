---
name: explain_panel
description: Explain what a panel shows using only the information provided
triggers:
  high_signal:
    - "this panel"
    - "this graph"
    - "this chart"
    - "what does this show"
    - "what does this display"
  medium_signal:
    - "explain panel"
    - "explain graph"
    - "explain chart"
  general:
    - "panel"
    - "graph"
    - "chart"
    - "visualization"
    - "metric"
min_confidence: 40
max_steps: 3
score_weights:
  high_signal: 80
  medium_signal: 60
  general: 20
requires_panel_context: true
---

Your task is to explain what a panel shows using ONLY the information provided.

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
