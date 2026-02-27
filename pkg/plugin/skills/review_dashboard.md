---
name: review_dashboard
description: Review dashboard quality and identify potential issues
triggers:
  high_signal:
    - "review this dashboard"
    - "is this dashboard good"
    - "dashboard quality"
    - "check this dashboard"
    - "audit this dashboard"
    - "dashboard issues"
    - "what's wrong with this dashboard"
    - "improve this dashboard"
  medium_signal:
    - "review"
    - "quality"
    - "best practices"
    - "issues"
    - "problems"
    - "mistakes"
    - "improvements"
    - "suggestions"
  review_aspects:
    - "query quality"
    - "label consistency"
    - "cardinality"
    - "aggregation"
    - "visualization"
    - "confusing"
min_confidence: 40
max_steps: 5
score_weights:
  high_signal: 100
  medium_signal: 60
  review_aspects: 40
negative_signals:
  - "what do i see"
  - "what is this"
requires_dashboard_context: true
---

Your task is to review dashboard quality and identify potential issues.

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
