# Direct LLM API Mode - Status

##  Coming Soon - Experimental Feature

The **Direct LLM API mode** is currently under development and **not fully tested**. This feature allows Zagalin backend to call OpenAI/Anthropic APIs directly without requiring grafana-llm-app.

---

## Current Status

 **Status**: Experimental / Not Production-Ready
 **Last Updated**: 2026-01-03
 **Testing**: Incomplete

---

## What is Direct LLM Mode?

Direct LLM mode allows Zagalin to bypass grafana-llm-app and communicate directly with LLM providers:

```

   Frontend  

       
       

   Zagalin   
   Backend   

       
       

  OpenAI or  
  Anthropic  
  Direct API 

```

### Benefits (When Complete)
-  No grafana-llm-app dependency
-  Full security features (rate limiting, validation, audit)
-  Direct control over API calls
-  Bring your own API keys

### Implementation Status
-  Backend client code exists (`pkg/plugin/llm_direct.go`)
-  Settings infrastructure in place
-  OpenAI and Anthropic API support
-  **NOT** fully tested
-  **NOT** battle-tested in production
-  Function calling may have issues
-  Error handling incomplete

---

## Why Not Use It Yet?

1. **Insufficient Testing**
   - Only basic manual testing performed
   - No automated test coverage for direct mode
   - Edge cases not validated

2. **Production Risk**
   - May fail unexpectedly in real-world use
   - Error handling not robust
   - Recovery mechanisms incomplete

3. **Better Alternatives Available**
   - **Recommended**: Use `grafana-llm` mode (default) - hybrid approach, no service account needed
   - **Production**: Use `backend-proxy` mode - full security, requires service account
   - Both modes are **tested and stable**

---

## Current Workaround

### UI Changes (v0.0.4+)

The Direct LLM API option in the configuration UI is now **disabled** with a "Coming Soon" badge:

```tsx
<Badge color="orange" text="Coming Soon" style={{ marginLeft: '8px' }} />
```

The card is grayed out and cannot be selected.

### Backend Warnings

If you manually configure `llmBackend: "direct"` in settings, the backend will log a warning on every request:

```
 Direct LLM mode is experimental and not fully tested. Use with caution.
```

---

## Recommended Modes (Stable & Tested)

### 1. Official Grafana Mode (Default) 

**Configuration:**
```json
{
  "llmBackend": "grafana-llm"
}
```

**Architecture:**
- Frontend → `@grafana/llm` (for LLM calls)
- Backend → Query validation, rate limiting, storage

**Pros:**
-  No service account needed
-  Session-based auth
-  Backend security features work
-  **Fully tested and stable**

**Use When:**
- You want the easiest setup
- You have grafana-llm-app installed
- You don't need full backend control over LLM calls

---

### 2. Backend Proxy Mode (Production) 

**Configuration:**
```json
{
  "llmBackend": "backend-proxy",
  "serviceAccountToken": "glsa_..."
}
```

**Architecture:**
- Frontend → Zagalin Backend → grafana-llm-app → LLM Provider

**Pros:**
-  Full security pipeline
-  Complete audit trail
-  Rate limiting enforced
-  **Fully tested and production-ready**

**Use When:**
- Production environment
- Need full audit trail
- Want centralized LLM configuration

---

## When Will Direct Mode Be Ready?

**Estimated Timeline**: TBD (To Be Determined)

**Required Work:**
1. Comprehensive testing with all providers (OpenAI, Anthropic, Azure OpenAI)
2. Function calling validation
3. Error handling improvements
4. Automated test coverage
5. Production validation
6. Documentation completion

**Progress Tracking:**
- Issue: (TBD - will be created when work begins)
- Milestone: (TBD)

---

## For Developers: Testing Direct Mode

If you want to help test direct mode:

### 1. Enable in Settings (Manual)

Edit plugin settings JSON directly:

```json
{
  "llmBackend": "direct",
  "llmProvider": "openai",
  "llmModel": "gpt-4o-mini",
  "llmEndpoint": "", // Optional custom endpoint
  "llmOrganization": "" // Optional for OpenAI
}
```

Add API key to secure settings:

```json
{
  "llmApiKey": "sk-..." // OpenAI or Anthropic API key
}
```

### 2. Monitor Backend Logs

You'll see warnings on every request:

```
[WARN]  Direct LLM mode is experimental and not fully tested. Use with caution.
```

### 3. Test Scenarios

Test these scenarios and report issues:

- [ ] Simple chat queries
- [ ] Function calling (query generation, panel explanation)
- [ ] Error handling (invalid API key, rate limits, network errors)
- [ ] Streaming responses
- [ ] Long conversations
- [ ] Multiple concurrent requests

### 4. Report Issues

If you find bugs, please report them with:
- Provider (OpenAI/Anthropic)
- Model used
- Error messages
- Backend logs
- Steps to reproduce

---

## Summary

**Don't use Direct LLM mode in production yet.** It's experimental and incomplete.

**Use these instead:**
-  `grafana-llm` (default) - Easy, stable, tested
-  `backend-proxy` - Production-ready, full security

**When it's ready**, we'll:
- Enable the UI option
- Remove experimental warnings
- Update documentation
- Announce in release notes

---

## Questions?

If you have questions about Direct LLM mode or need help choosing the right backend mode, please:

1. Check the [CLAUDE.md](../.claude/CLAUDE.md) documentation
2. Review [API documentation](./api/ENDPOINTS.md)
3. Open a GitHub issue

---

**Last Updated**: 2026-01-03
**Status**:  Experimental - Do Not Use in Production
