---
name: alert_triage
description: Prioritize and categorize firing alerts for efficient incident response
triggers:
  high_signal:
    - "triage alerts"
    - "prioritize alerts"
    - "which alert first"
    - "alert priority"
  general:
    - "too many alerts"
    - "alert storm"
    - "alert fatigue"
min_confidence: 40
max_steps: 5
score_weights:
  high_signal: 100
  general: 30
---

You are triaging firing alerts to help prioritize incident response.

## TRIAGE METHODOLOGY

**Step 1: Gather Alert State**
- Call get_firing_alerts to get current firing alerts
- Group alerts by service/namespace

**Step 2: Categorize by Impact**
- **Critical**: User-facing impact, revenue impact, data loss risk
- **High**: Service degradation, SLO breach imminent
- **Medium**: Performance degradation, non-critical service issues
- **Low**: Warnings, capacity planning alerts

**Step 3: Identify Relationships**
- Look for root cause vs symptom alerts
- Identify cascading failures
- Group related alerts

**Step 4: Recommend Priority Order**
- Address root causes before symptoms
- Stabilize critical path first
- Batch related issues

## OUTPUT FORMAT

## Alert Summary
**Total Firing**: [count]
**Critical**: [count] | **High**: [count] | **Medium**: [count] | **Low**: [count]

## Priority Queue
1. **[Alert Name]** - [Service] - [Severity]
   - Why first: [Reason]
   - Action: [What to do]

2. **[Alert Name]** - [Service] - [Severity]
   - Why: [Reason]
   - Action: [What to do]

## Patterns Detected
- [Pattern 1]: [Related alerts and likely root cause]
- [Pattern 2]: [Related alerts and likely root cause]

## Recommendation
[Which alerts to address first and why]
