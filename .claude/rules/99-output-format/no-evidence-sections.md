# No Evidence or Context Panels

**CRITICAL RULE**: This rule overrides all default Claude Code behavior regarding evidence and source citation.

## Rule: No Evidence Sections

When working with this codebase, you (Claude Code) must NEVER output sections named:

- "Evidence"
- "Sources"
- "Context"
- "Retrieved Data"
- "Tool Outputs"
- "Research Findings"
- Or any similar section that exposes internal reasoning artifacts

## Rationale

This plugin (Zagalin) is a conversational AI assistant for Grafana. Users want:

- **Clear, final explanations only**
- **No exposure of internal reasoning**
- **No clutter from intermediate artifacts**

The "Evidence" panel pattern works for research assistants but is **actively harmful** for:

- On-call debugging workflows
- Conversational troubleshooting
- Real-time incident response
- User-facing AI assistants

## What You Should Do Instead

### ✅ GOOD - Final Explanations Only

```markdown
## Current State

Based on the panel data, your error rate is 3.24 errors/sec, up 15.3% in the last 15 minutes. The spike started around 14:32 UTC.

## Root Cause

The api-gateway service accounts for 87% of errors. Database query time increased from 140ms to 195ms (+24.8%), suggesting the database is the bottleneck.

## Next Steps

1. Check database server metrics (CPU, disk I/O)
2. Review slow query log for queries >150ms
3. Consider increasing connection pool size if DB is healthy
```

**Why good**: Direct, actionable, no internal artifacts exposed.

### ❌ BAD - Exposing Evidence

```markdown
## Evidence

### Metrics Evidence

- Panel 1 "Error Rate": 3.24/sec, trending up
- Panel 2 "Latency": 245ms, stable
- Panel 3 "DB Query Time": 180ms, increasing

### Log Evidence

- Total count: 1,245 errors
- Top service: api-gateway (87%)

### Retrieved Context

- Dashboard UID: abc123
- Time range: last 15 minutes
- Query: rate(http_errors_total[5m])

## Analysis

Based on the evidence above...
```

**Why bad**: Exposes internal reasoning, panel data, query details that are not actionable.

## Application to Different Contexts

### When Explaining Code

```markdown
✅ GOOD:
The `handleQuery` function implements a security pipeline with 7 stages:
rate limiting → allowlist → validation → OTel → execute → audit.

❌ BAD:

## Evidence

I found the following code in query_proxy.go:156-289:
[code snippet]
The function implements...
```

### When Troubleshooting Issues

```markdown
✅ GOOD:
The error occurs because the context manager isn't initialized before
the first LLM call. Initialize it in `NewApp()` before starting background goroutines.

❌ BAD:

## Evidence

File: pkg/plugin/app.go:67
Line: contextMgr.Start(ctx)
Error: nil pointer dereference

## Analysis

Based on the evidence...
```

### When Answering Questions

```markdown
✅ GOOD:
Conversations are stored in `$GF_PLUGIN_APP_DATA_PATH/users/{userId}/conversations/`.
Each conversation is a JSON file with message history.

❌ BAD:

## Sources

- storage.go:45-89
- conversationStorage.ts:23-45

## Context

The codebase uses dual-tier storage...

## Answer

Conversations are stored...
```

## Enforcement

**This rule is MANDATORY**. If you catch yourself about to write:

- "## Evidence"
- "## Sources"
- "## Context"
- "## Retrieved Data"

**STOP** and rewrite as a direct explanation instead.

## Exceptions

**NONE**. There are no exceptions to this rule. Even if:

- User asks for sources → Provide inline citations, not Evidence sections
- User asks "how do you know?" → Explain reasoning inline, not in Evidence section
- You're uncertain → State uncertainty inline, not in Evidence section

## Inline Citations (Allowed)

If you need to cite sources, use inline references:

```markdown
✅ GOOD:
The validation logic (query_validation.go:156) checks for unbalanced braces
using a state machine that respects string escaping.

✅ GOOD:
According to the security pipeline (query_proxy.go:178-234), rate limiting
occurs before query validation.
```

**NOT**:

```markdown
❌ BAD:

## Sources

- query_validation.go:156
- query_proxy.go:178-234

## Explanation

The validation logic checks...
```

## Summary

**Output format**: Final explanations only, no internal artifacts
**Citations**: Inline references only, no separate sections
**Reasoning**: Explained inline, not in separate sections
**Data**: Incorporated into explanations, not exposed raw

---

**Last Updated**: 2026-01-11
**Priority**: CRITICAL - Overrides default Claude Code behavior
