---
name: analyze_dashboard
description: Describe and explain what is shown on a dashboard
triggers:
  high_signal:
    - "what do i see"
    - "what am i seeing"
    - "what am i looking at"
    - "describe this dashboard"
    - "what is this dashboard"
    - "what's on this dashboard"
  medium_signal:
    - "analyze this"
    - "overview"
    - "understand this"
    - "show me this"
    - "explain this dashboard"
  general:
    - "dashboard"
    - "this dashboard"
min_confidence: 40
max_steps: 4
score_weights:
  high_signal: 80
  medium_signal: 50
  general: 30
negative_signals:
  - "review"
  - "quality"
  - "issues"
  - "query"
  - "promql"
  - "logql"
  - "panel"
  - "graph"
  - "chart"
requires_dashboard_context: true
---

The user is asking about what they see on their screen.

Your task is to:
1. **Describe the overall purpose** - What story does this dashboard tell? What system/service is being monitored?
2. **Summarize key panels** - What are the most important visualizations? Group related panels together
3. **Identify patterns** - What should the user focus on? Are there any red flags or interesting trends?
4. **Provide context** - Why would someone use this dashboard? What decisions does it support?

Be conversational and insightful. Imagine you're sitting next to them explaining what you see.
