---
name: deployment_correlation
description: Correlate incidents with recent deployments and changes
triggers:
  high_signal:
    - "after deploy"
    - "since deployment"
    - "after release"
    - "recent change"
    - "rollback"
  general:
    - "deploy"
    - "deployment"
    - "release"
    - "version"
    - "commit"
min_confidence: 40
max_steps: 6
score_weights:
  high_signal: 100
  general: 25
---

You are investigating whether a recent deployment caused an incident.

## CORRELATION METHODOLOGY

**Step 1: Establish Timeline**
- When did the symptoms start?
- When was the last deployment to affected services?
- Is there a clear temporal correlation?

**Step 2: Identify Changed Services**
- Which services were deployed recently?
- What changed in the deployment (code, config, dependencies)?

**Step 3: Compare Before/After**
- Query metrics for 1 hour before and after deployment
- Look for: error rate changes, latency changes, traffic patterns

**Step 4: Check Deployment-Specific Signals**
- Pod restart counts
- Memory/CPU changes post-deploy
- New error types in logs

**Step 5: Assess Rollback Risk**
- Is rollback safe?
- What would rollback break?
- Is there a partial rollback option?

## OUTPUT FORMAT

## Deployment Analysis
**Suspected Deployment**: [Service] @ [Time]
**Symptom Start**: [Time]
**Correlation Strength**: [Strong/Medium/Weak]

## Evidence
### Before Deployment (1h window)
- Error rate: [value]
- Latency p95: [value]
- Traffic: [value]

### After Deployment (1h window)
- Error rate: [value] ([change]%)
- Latency p95: [value] ([change]%)
- Traffic: [value] ([change]%)

## Changes Detected
- [Change 1]
- [Change 2]

## Recommendation
**Action**: [Rollback / Monitor / Investigate further]
**Reasoning**: [Why this action]
**Rollback Risk**: [Assessment if applicable]
