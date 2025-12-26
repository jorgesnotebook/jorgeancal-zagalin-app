# 🎯 Zagalin Roadmap

> **Governed AI Assistant for OpenTelemetry Investigations in Grafana**

This document outlines the development roadmap for Zagalin, organized into MVP phases with clear acceptance criteria and priorities.

---

## 📊 Roadmap Overview

| Phase | Status | Focus Area | Tickets |
|-------|--------|------------|---------|
| **MVP-0** | 🟢 In Progress | Foundational Delivery | 1-2 |
| **MVP-1** | 🟡 Planned | Governed Query Engine | 3-8 |
| **MVP-2** | 🔵 Future | Cost, Reasoning & Semantics | 9-12 |
| **MVP-3** | 🟣 Future | Sharing & Admin Governance | 13-17 |

**Legend:**
- 🟢 In Progress - Active development
- 🟡 Planned - Next in queue
- 🔵 Future - Medium-term goals
- 🟣 Future - Long-term vision

**Completed Features:**
- ✅ Global floating chat button (context-aware, LLM health check)
- ✅ Dashboard context extraction (panels, queries, time range, variables)

---

## 🟢 MVP-0: Foundational Delivery

**Goal:** Establish core chat functionality with conversation persistence.

### ✅ ~~Ticket 1: Conversation Model & User Storage~~ - **COMPLETED**

**Type:** Story
**Priority:** High
**Status:** ✅ Completed (2025-12-25)

**Description:**
Persist user conversations using localStorage (with path to migrate to Grafana User Storage API).

**Acceptance Criteria:**
- [x] Conversations persist per user (localStorage per browser)
- [x] Trimmed history (max 100 messages per conversation)
- [x] Limited number of conversations (max 50, auto-pruned)
- [x] User isolation (localStorage is browser-specific)

**Implementation:**
- ✅ `src/services/conversationStorage.ts` - Full CRUD operations
- ✅ `src/hooks/useConversation.ts` - React hook for state management
- ✅ Integrated into `ChatPanel.tsx` - Auto-save on every message
- ✅ New Chat button - Start fresh conversations
- ✅ Message counter badge - Shows conversation length
- ✅ Conversation metadata - Title, timestamps, pin status, context
- ✅ Auto-pruning - Keeps pinned conversations, removes oldest unpinned

---

### 🎟️ Ticket 2: Conversation History in Popup

**Type:** Story
**Priority:** Medium
**Status:** 📋 Planned

**Description:**
Expose stored conversations inside the popup UI with management capabilities.

**Acceptance Criteria:**
- [ ] List recent conversations with timestamps
- [ ] Resume a conversation (restore full context)
- [ ] Rename conversations
- [ ] Pin important conversations
- [ ] Delete conversations

**UI Mockup Considerations:**
- Sidebar with conversation list
- Search/filter conversations
- Preview last message
- Visual indicators for pinned items

**Dependencies:**
- Requires Ticket 1 (Conversation Model) to be completed first

---

## 🟡 MVP-1: Governed Query Engine

**Goal:** Implement query governance with OTel scope enforcement and datasource restrictions.

### 🎟️ Ticket 3: Backend Plugin & Identity Context

**Type:** Story
**Priority:** High
**Status:** 📋 Planned

**Description:**
Backend receives user identity and handles all queries with proper security context.

**Acceptance Criteria:**
- [ ] All queries routed via backend
- [ ] User identity available server-side
- [ ] Frontend has no direct datasource access

**Security Requirements:**
- User identity from Grafana context
- RBAC enforcement
- Audit logging for all queries

**Backend Changes:**
- New endpoint: `/api/plugins/jorgeancal-zagalin-app/resources/query`
- Identity middleware in `pkg/plugin/`
- Query proxy layer

---

### 🎟️ Ticket 4: Datasource Discovery & Allowlist

**Type:** Story
**Priority:** Medium
**Status:** 📋 Planned

**Description:**
Discover Grafana datasources and restrict usage via allowlist configuration.

**Acceptance Criteria:**
- [ ] Datasources listed via Grafana API
- [ ] Only allowlisted datasources usable
- [ ] Configurable via `jsonData` in plugin settings

**Configuration Schema:**
```json
{
  "allowedDatasources": [
    "prometheus-prod",
    "loki-prod",
    "tempo-prod"
  ],
  "defaultDatasource": "prometheus-prod"
}
```

---

### 🎟️ Ticket 5: Mandatory OTel Scope Enforcement

**Type:** Story
**Priority:** High
**Status:** 📋 Planned

**Description:**
Enforce `service.name` + `deployment.environment.name` on all queries for proper scoping.

**Acceptance Criteria:**
- [ ] Queries without scope rejected
- [ ] Fallback logic applied (configurable default)
- [ ] Fallback usage logged for auditing

**OTel Attributes Required:**
- `service.name` (mandatory)
- `deployment.environment.name` (mandatory)

**Fallback Strategy:**
1. Check user's current context for service/env
2. Use configured default if available
3. Reject query if no valid scope

---

### 🎟️ Ticket 6: PromQL Injection & Validation

**Type:** Story
**Priority:** High
**Status:** 📋 Planned

**Description:**
Inject and validate OTel filters in PromQL queries safely.

**Acceptance Criteria:**
- [ ] Safe query rewriting (no syntax errors)
- [ ] No duplicate series (deduplication logic)
- [ ] Errors mapped cleanly to user-friendly messages

**Example Transformation:**
```promql
# User query
rate(http_requests_total[5m])

# After injection
rate(http_requests_total{service_name="my-service",deployment_environment_name="production"}[5m])
```

**Validation Rules:**
- Parse query with PromQL parser
- Inject labels without breaking existing selectors
- Validate result query is syntactically correct

---

### 🎟️ Ticket 7: LogQL Injection & Validation

**Type:** Story
**Priority:** High
**Status:** 📋 Planned

**Description:**
Inject and validate OTel filters in LogQL queries with two-pass fallback.

**Acceptance Criteria:**
- [ ] Two-pass fallback (structured attributes → log line parsing)
- [ ] Deduplication of log lines
- [ ] Cardinality limits enforced (prevent expensive queries)

**Two-Pass Strategy:**
1. **Pass 1:** Try structured labels
   ```logql
   {service_name="my-service",deployment_environment_name="production"} |= "error"
   ```
2. **Pass 2:** Fallback to log line parsing if structured labels unavailable
   ```logql
   {job="logs"} | json | service_name="my-service" | deployment_environment_name="production" |= "error"
   ```

---

### 🎟️ Ticket 8: TraceQL Builder & Enforcement

**Type:** Story
**Priority:** Medium
**Status:** 📋 Planned

**Description:**
Build governed TraceQL queries with environment scope enforcement.

**Acceptance Criteria:**
- [ ] Env OR logic (support multi-environment queries)
- [ ] Trace hydration (fetch full trace details)
- [ ] Scoped search only (no cross-tenant data access)

**TraceQL Enforcement:**
```traceql
{ service.name="my-service" && (deployment.environment.name="staging" || deployment.environment.name="production") }
```

**Hydration:**
- Fetch trace by ID after search
- Include spans, metadata, and relationships
- Apply same governance rules

---

## 🔵 MVP-2: Cost, Reasoning & Semantics

**Goal:** Optimize query costs, implement explainable AI reasoning, and add OTel semantic awareness.

### 🎟️ Ticket 9: Time Slicing Engine

**Type:** Story
**Priority:** Medium
**Status:** 💡 Ideation

**Description:**
Split large time range queries into 6-hour chunks to reduce load and improve reliability.

**Acceptance Criteria:**
- [ ] Chunking enforced for queries > 6 hours
- [ ] Parallelism limited (max 4 concurrent chunks)
- [ ] Partial failures handled gracefully

**Time Slicing Logic:**
```
Query: 24h time range
Split into: 4 × 6h chunks
Execute: Max 4 parallel
Merge: Stitch results chronologically
```

**Benefits:**
- Lower memory usage per query
- Better error recovery
- Reduced backend load spikes

---

### 🎟️ Ticket 10: Result Merge Strategies

**Type:** Story
**Priority:** Medium
**Status:** 💡 Ideation

**Description:**
Merge chunked query results with signal-specific strategies.

**Acceptance Criteria:**
- [ ] Metrics merged by timestamp (sorted, aligned)
- [ ] Logs sorted & deduplicated
- [ ] Traces deduplicated by span ID

**Signal-Specific Merging:**

| Signal | Strategy | Deduplication |
|--------|----------|---------------|
| Metrics | Time-aligned merge | By timestamp + labels |
| Logs | Chronological sort | By log ID / hash |
| Traces | Span assembly | By trace ID + span ID |

---

### 🎟️ Ticket 11: Semantic Dictionary (OTel Awareness)

**Type:** Story
**Priority:** High
**Status:** 💡 Ideation

**Description:**
Implement OTel semantic conventions dictionary for better AI understanding.

**Acceptance Criteria:**
- [ ] Descriptions present for common OTel attributes
- [ ] Required flags enforced (mandatory attributes)
- [ ] Used by UI + AI for context

**Dictionary Structure:**
```json
{
  "service.name": {
    "type": "string",
    "required": true,
    "description": "Logical name of the service",
    "examples": ["checkout-service", "payment-api"]
  },
  "deployment.environment.name": {
    "type": "string",
    "required": true,
    "description": "Deployment environment",
    "enum": ["production", "staging", "development"]
  }
}
```

**Use Cases:**
- AI prompt context enrichment
- Query validation
- Auto-complete suggestions
- Documentation generation

---

### 🎟️ Ticket 12: AI Step-by-Step Explanation Engine

**Type:** Story
**Priority:** High
**Status:** 💡 Ideation

**Description:**
Produce explainable, structured AI responses following a clear reasoning pattern.

**Acceptance Criteria:**
- [ ] Goal / Plan / Steps / Evidence / Conclusion format
- [ ] No raw chain-of-thought exposed to user
- [ ] Reusable artifacts produced (queries, dashboards)

**Response Structure:**
```markdown
## Goal
[What we're trying to achieve]

## Plan
1. Check metric X
2. Analyze logs Y
3. Trace flows Z

## Steps Taken
- ✅ Queried http_requests_total: Found 500 errors spike at 14:30
- ✅ Analyzed error logs: Database connection timeout
- ✅ Traced sample request: 30s DB query

## Evidence
[Charts, queries, links]

## Conclusion
Database query performance issue detected at 14:30. Likely cause: missing index on orders table.

## Recommended Actions
- Add index to orders.created_at
- Review slow query logs
- Set up alert for DB query time > 5s
```

---

## 🟣 MVP-3: Sharing & Admin Governance

**Goal:** Enable investigation sharing, export to dashboards, and admin-controlled governance.

### 🎟️ Ticket 13: Export Investigation to Dashboard

**Type:** Story
**Priority:** Medium
**Status:** 💡 Ideation

**Description:**
Export AI-driven investigations to reusable Grafana dashboards.

**Acceptance Criteria:**
- [ ] New dashboard created programmatically
- [ ] Correct folder & tags applied
- [ ] Full context preserved (queries, time ranges, annotations)

**Export Features:**
- One-click "Export to Dashboard" button
- Choose folder and name
- Include all queries from investigation
- Add markdown panels with AI explanations
- Preserve time range and variables

---

### 🎟️ Ticket 14: Dashboard Notes & Evidence Panels

**Type:** Story
**Priority:** Medium
**Status:** 💡 Ideation

**Description:**
Generate markdown notes and optional evidence panels in exported dashboards.

**Acceptance Criteria:**
- [ ] Notes readable standalone (no external context needed)
- [ ] Evidence linked correctly (panels reference specific queries)
- [ ] Dashboard reproducible (can be re-run with updated time range)

**Panel Types:**
- Text panel: AI reasoning and findings
- Graph panel: Time series evidence
- Table panel: Log samples or trace data
- Stat panel: Key metrics summary

---

### 🎟️ Ticket 15: Admin-only Configuration UI

**Type:** Story
**Priority:** High
**Status:** 💡 Ideation

**Description:**
Restrict configuration editing to admin users only.

**Acceptance Criteria:**
- [ ] Admin-only edit access (checked via Grafana RBAC)
- [ ] Read-only view for non-admin users
- [ ] Effective config visible to all users

**RBAC Implementation:**
```typescript
// Check user role
if (!user.isGrafanaAdmin) {
  return <ConfigReadOnlyView config={effectiveConfig} />;
}

return <ConfigEditor config={config} onSave={handleSave} />;
```

---

### 🎟️ Ticket 16: Declarative Configuration

**Type:** Story
**Priority:** Medium
**Status:** 💡 Ideation

**Description:**
All configuration importable/exportable via `jsonData` and `secureJsonData`.

**Acceptance Criteria:**
- [ ] No hardcoded values in code
- [ ] Secrets only in `secureJsonData`
- [ ] Config reproducible across environments

**Configuration Schema:**
```json
{
  "jsonData": {
    "allowedDatasources": ["prometheus-prod", "loki-prod"],
    "otelEnforcement": {
      "required": true,
      "attributes": ["service.name", "deployment.environment.name"]
    },
    "timeSlicing": {
      "enabled": true,
      "chunkSizeHours": 6,
      "maxParallel": 4
    },
    "governance": {
      "maxQueryTimeRange": "24h",
      "rateLimit": 60
    }
  },
  "secureJsonData": {
    "llmApiKey": "sk-...",
    "vectorSearchApiKey": "..."
  }
}
```

**Provisioning:**
Support Grafana provisioning for automated deployment:
```yaml
# provisioning/plugins/zagalin.yaml
apiVersion: 1
apps:
  - type: jorgeancal-zagalin-app
    jsonData:
      allowedDatasources: [prometheus-prod]
    secureJsonData:
      llmApiKey: ${LLM_API_KEY}
```

---

### 🎟️ Ticket 17: Security, Auditing & Guardrails

**Type:** Story
**Priority:** High
**Status:** 💡 Ideation

**Description:**
Add comprehensive safety measures, auditing, and operational limits.

**Acceptance Criteria:**
- [ ] Query audit logs (who, what, when, result)
- [ ] Rate limiting (per user, per org)
- [ ] RBAC respected (Grafana permissions enforced)

**Security Features:**

| Feature | Implementation | Purpose |
|---------|---------------|---------|
| **Audit Logging** | All queries logged to backend | Compliance, debugging |
| **Rate Limiting** | 60 queries/min per user | Prevent abuse |
| **Cost Limits** | Monthly token budget | Budget control |
| **Query Timeout** | 30s max per query | Resource protection |
| **Result Size Limit** | Max 10MB response | Memory protection |
| **RBAC Integration** | Check datasource permissions | Security |

**Audit Log Format:**
```json
{
  "timestamp": "2025-12-25T10:30:00Z",
  "user": "user@example.com",
  "orgId": 1,
  "action": "query_execute",
  "datasource": "prometheus-prod",
  "query": "rate(http_requests_total[5m])",
  "timeRange": "now-1h",
  "success": true,
  "resultCount": 42,
  "executionTimeMs": 234
}
```

---

## 📈 Progress Tracking

### ✅ Completed (Moved to Production)
- [x] **Global floating chat button** - Context-aware, LLM health checks, smart visibility
- [x] **Dashboard context extraction** - Panels, queries, time range, template variables
- [x] **Context optimizer** - Token-aware compression and prioritization
- [x] **Panel analysis skills** - Explain panel, generate query, troubleshooting, dashboard analysis
- [x] **PromQL/LogQL query generation** - Natural language to query conversion
- [x] **Streaming LLM responses** - RxJS-based streaming with @grafana/llm
- [x] **Configuration UI** - Personality presets, skill toggles, LLM parameters
- [x] **Vector search service** - Semantic search integration (optional)
- [x] **Action extraction** - Parse queries and links from LLM responses
- [x] **Rate limiting & guardrails** - Backend cost controls
- [x] **Conversation persistence** - LocalStorage-based, auto-save, pruning (Ticket 1) ⭐ NEW

### 🚧 In Progress (MVP-0)
- [ ] Conversation history UI (Ticket 2)

### 📋 Planned Next (MVP-1)
- [ ] Backend query proxy with identity (Ticket 3)
- [ ] Datasource allowlist (Ticket 4)
- [ ] OTel scope enforcement (Ticket 5)
- [ ] Query injection & validation (Tickets 6-8)

---

## 🔗 Related Documents

- **Architecture**: `../CLAUDE.md` - Comprehensive development guide
- **CI/CD**: `CI_PIPELINE_SUMMARY.md` - Build and deployment process
- **LLM Integration**: `GRAFANA_LLM_APP_ANALYSIS.md` - LLM infrastructure details
- **Improvements**: `ZAGALIN_IMPROVEMENTS.md` - Technical debt and enhancements

---

## 📝 Notes for Contributors

### Adding New Tickets
1. Choose appropriate MVP phase
2. Add ticket number (sequential within phase)
3. Include acceptance criteria
4. Add technical considerations if needed
5. Link to related tickets or docs

### Updating Status
Use these status indicators:
- 💡 Ideation - Still being designed
- 📋 Planned - Ready to start
- 🚧 In Progress - Active development
- ✅ Completed - Done and merged

### Priority Levels
- **High** - Critical for MVP delivery
- **Medium** - Important but not blocking
- **Low** - Nice to have

---

---

## 📝 Changelog

**2025-12-25 (Evening):**
- ✅ **COMPLETED Ticket 1: Conversation Model & User Storage**
  - Created `conversationStorage.ts` with full CRUD operations
  - Created `useConversation.ts` React hook
  - Integrated into ChatPanel with auto-save
  - Added New Chat button and message counter
  - Max 50 conversations, max 100 messages per conversation
  - Auto-pruning with pinned conversation support
- 📦 Build successful - No errors, ready for testing

**2025-12-25 (Morning):**
- ✅ Removed original completed Tickets 1 & 4 (Global Popup, Context Capture)
- 🔢 Renumbered all remaining tickets (2→1, 3→2, 5→3, etc.)
- 📊 Updated roadmap overview to reflect tickets
- ✨ Added comprehensive "Completed Features" section

**Total Remaining:** 16 tickets across 4 MVP phases (1 completed today!)

---

**Last Updated:** 2025-12-25
**Maintained By:** Zagalin Team
