---
name: design_dashboard
description: Design a new dashboard or propose improvements
triggers:
  high_signal:
    - "create dashboard"
    - "design dashboard"
    - "build dashboard"
    - "new dashboard"
    - "dashboard for"
    - "dashboard to monitor"
  medium_signal:
    - "suggest panels"
    - "what panels"
    - "dashboard layout"
    - "how to visualize"
    - "best way to show"
  general:
    - "panels"
    - "layout"
    - "visualize"
    - "charts"
    - "graphs"
min_confidence: 40
max_steps: 5
score_weights:
  high_signal: 100
  medium_signal: 70
  general: 15
---

Your task is to design a new dashboard or propose improvements.

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
- Based on: Available metrics, reference dashboards, best practices
