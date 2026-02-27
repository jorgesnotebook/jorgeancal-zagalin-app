---
name: troubleshoot
description: Provide structured troubleshooting based on available context
triggers:
  high_signal:
    - "not working"
    - "doesn't work"
    - "failing"
    - "broken"
    - "troubleshoot"
    - "debug"
    - "investigate"
  problem_keywords:
    - "error"
    - "errors"
    - "issue"
    - "problem"
    - "wrong"
    - "failed"
    - "failure"
    - "down"
    - "outage"
  question_patterns:
    - "why is"
    - "why are"
    - "what's wrong"
    - "what went wrong"
    - "how do i fix"
    - "how to fix"
    - "what happened"
  symptom_keywords:
    - "slow"
    - "high"
    - "spike"
    - "drop"
    - "missing"
    - "timeout"
    - "latency"
    - "degraded"
    - "anomaly"
    - "unusual"
min_confidence: 40
max_steps: 5
score_weights:
  high_signal: 80
  problem_keywords: 30
  question_patterns: 40
  symptom_keywords: 10
---

Your task is to provide structured troubleshooting based ONLY on available context.

**CONTEXT-GATED INVESTIGATION - CRITICAL**
Before proceeding, you MUST check:
- Do I have panel queries, metric results, or log results?
- Do I have explicit statements about signal presence/absence?

If NO context is available, you MUST:
- STOP immediately
- Ask ONE question: "Which dashboard or panel should I base the investigation on?"
- Do NOT speculate, assume, or invent symptoms

REQUIRED STRUCTURE:
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
   ```promql or logql
   [Specific query based on available metrics]
   ```
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
   - State clearly what context you have and what's missing

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
- If context is MISSING → STOP and ask for it
- NEVER ask for screenshots, files, exports, or "what it looks like"
- Hypotheses MUST be traceable to specific queries or panels

EXAMPLE GOOD RESPONSE:
"Based on panel data from Memory usage (panel 2), CPU (panel 1), and request rate (panel 3):

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
