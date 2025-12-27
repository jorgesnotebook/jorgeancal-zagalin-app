# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Zagalin** is a context-aware AI assistant Grafana plugin that brings LLM capabilities directly into Grafana. It enables users to chat with metrics, generate PromQL/LogQL queries, and troubleshoot issues using natural language.

### Key Architecture

- **Hybrid Plugin**: Frontend (React/TypeScript) + Backend (Go)
- **LLM Integration**: Provider-agnostic through grafana-llm-app (supports OpenAI, Azure, Anthropic, etc.)
- **Storage**: Dual-tier conversation storage (backend Go storage + localStorage fallback)
- **Context System**: Backend Go service that extracts and caches Prometheus/Loki context
- **Floating UI**: Global chat button that appears on dashboards using portal mounting

### Technology Stack

**Frontend:**
- React 18 with TypeScript
- Grafana UI components (@grafana/ui)
- RxJS for streaming LLM responses
- Webpack 5 for bundling
- Jest for unit tests, Playwright for E2E

**Backend:**
- Go 1.21+ with grafana-plugin-sdk-go
- Mage build system
- Context manager for Prometheus/Loki metadata extraction
- Rate limiting and guardrails system
- File-based user storage for conversations

## Common Commands

### Development Workflow

```bash
# Install dependencies
npm ci

# Start development mode (frontend watch build)
npm run dev

# Run Grafana with plugin in Docker
npm run server

# Full build (frontend + backend)
npm run build && mage -v buildAll
```

### Testing

```bash
# Run unit tests (watch mode)
npm run test

# Run unit tests (CI mode)
npm run test:ci

# Run E2E tests (requires Grafana running)
npm run e2e

# Type checking
npm run typecheck
```

### Linting & Formatting

```bash
# Run linter
npm run lint

# Fix linting and formatting issues
npm run lint:fix
```

### Local CI Pipeline

```bash
# Run full CI pipeline locally (includes frontend + backend tests)
./ci-local.sh

# This runs: npm ci → typecheck → lint → test:ci → build → mage coverage → mage buildAll → plugin validation
```

### Backend Build (Go)

```bash
# Build all backend targets (Linux, Darwin, Windows)
mage -v buildAll

# Run backend tests with coverage
mage -v coverage

# Build specific target
mage -v build
```

### Plugin Signing

```bash
# Sign plugin (requires GRAFANA_ACCESS_POLICY_TOKEN)
npm run sign
```

## Critical Architecture Patterns

### 1. Dual Storage System

The plugin uses a **fallback storage pattern**:
- **Primary**: Backend Go storage (file-based in `GF_PLUGIN_APP_DATA_PATH`)
- **Fallback**: Browser localStorage
- **Migration**: Automatic migration from localStorage to backend on first backend availability

**Files**:
- `src/services/conversationStorage.ts` - Main storage interface
- `src/services/storageApiClient.ts` - Backend API client
- `pkg/plugin/storage.go` - Backend storage implementation

**Key behavior**: All storage operations check backend availability first, fall back to localStorage if unavailable.

### 2. Context Manager (Backend)

The Go backend maintains a **cached context** of Prometheus metrics, Loki streams, and Tempo traces:
- Runs background refresh every N minutes (configurable)
- Extracts metric names, labels, log streams from datasources
- Provides `/context/status` and `/context/refresh` endpoints

**Files**:
- `pkg/plugin/context/manager.go` - Main context manager
- `pkg/plugin/context/metrics.go` - Prometheus integration
- `pkg/plugin/context/logs.go` - Loki integration
- `pkg/plugin/context/traces.go` - Tempo integration

**Purpose**: Reduces LLM token usage by providing pre-extracted context instead of raw metrics.

### 3. Global Chat Mounting

The floating chat button uses a **portal mounting pattern**:
- `globalChatMount.tsx` creates a global div (`#zagalin-global-chat-root`)
- Mounts once when the module loads (not per page)
- Persists across Grafana navigation
- Only displays on dashboard pages

**Files**:
- `src/globalChatMount.tsx` - Global mount logic
- `src/components/FloatingChat/FloatingChatButton.tsx` - Floating button component

### 4. LLM Streaming

Uses **RxJS observables** for streaming LLM responses:
- Frontend calls grafana-llm-app's streaming API
- Backend proxies queries to add guardrails (rate limiting, validation)
- Supports function calling for structured tool execution

**Files**:
- `src/services/queryService.ts` - LLM query orchestration
- `src/services/assistantSkills.ts` - System prompts and skills
- `src/services/zagalinTools.ts` - Function calling tools
- `pkg/plugin/query_proxy.go` - Backend query proxy

### 5. Context Service (Frontend)

Extracts runtime context from Grafana:
- Dashboard UID, title, panels
- Time range from URL params
- Panel queries (PromQL/LogQL)
- Template variables

**Files**:
- `src/services/contextService.ts` - Context extraction
- `src/services/useGrafanaContext.ts` - React hook
- `src/services/contextOptimizer.ts` - Context size reduction

## Plugin Configuration

### Plugin Metadata
- `src/plugin.json` - Plugin metadata (ID, name, version, dependencies)
  - Version and date are templated (`%VERSION%`, `%TODAY%`)
  - Requires `grafana-llm-app` as dependency

### Build Configuration
- `.config/webpack/webpack.config.ts` - Webpack configuration
- `Magefile.go` - Go build configuration (delegates to grafana-plugin-sdk-go)
- `tsconfig.json` - TypeScript configuration
- `jest.config.js` - Test configuration
- `playwright.config.ts` - E2E test configuration

### Docker Development
- `docker-compose.yaml` - Development Grafana environment
- `.config/docker-compose-base.yaml` - Base Docker Compose config
- `.config/Dockerfile` - Custom Grafana image with plugin

## Project Structure

```
src/
├── components/          # React components
│   ├── App/            # Main app component
│   ├── AppConfig/      # Configuration UI
│   ├── FloatingChat/   # Floating chat components
│   └── AskPanel/       # Dashboard panel component
├── pages/              # Page components
│   ├── ChatPage.tsx    # Full-screen chat
│   ├── ConfigPage/     # Configuration page
│   └── AssistantChatPage.tsx
├── services/           # Business logic
│   ├── conversationStorage.ts    # Conversation persistence
│   ├── contextService.ts         # Grafana context extraction
│   ├── queryService.ts           # LLM query orchestration
│   ├── assistantSkills.ts        # System prompts
│   └── zagalinTools.ts           # Function calling tools
├── hooks/              # React hooks
├── types/              # TypeScript types
└── module.tsx          # Plugin entry point

pkg/
├── main.go             # Plugin binary entry
└── plugin/
    ├── app.go          # Main plugin app
    ├── resources.go    # HTTP route handlers
    ├── storage.go      # Conversation storage
    ├── guardrails.go   # Rate limiting
    ├── query_proxy.go  # LLM query proxy
    └── context/        # Context extraction
        ├── manager.go
        ├── metrics.go
        ├── logs.go
        └── traces.go
```

## Development Notes

### Backend Changes Require Full Rebuild

When modifying Go code:
1. Build backend: `mage -v buildAll`
2. Restart Docker: `npm run server` (or `docker compose restart`)
3. Backend binary is `dist/gpx_zagalin_*` (platform-specific)

### Frontend Hot Reload

- `npm run dev` watches frontend changes
- Grafana Dev mode must be enabled in Docker
- Refresh browser to see changes

### Testing Conversations Storage

- Backend storage: Check `$GF_PLUGIN_APP_DATA_PATH/users/{userId}/conversations/`
- Frontend storage: Check browser localStorage key `zagalin-conversations`
- Migration happens automatically when backend becomes available

### CI/CD Pipeline

**GitHub Actions**:
- `.github/workflows/ci.yml` - Main CI (build, test, E2E)
- `.github/workflows/release.yml` - Release on version tags
- `.github/workflows/is-compatible.yml` - Grafana compatibility check

**Local CI**:
- `ci-local.sh` - Runs full pipeline locally
- Use before pushing to catch issues early

### Plugin Security & Signing

- Signing requires `GRAFANA_ACCESS_POLICY_TOKEN` (Grafana Cloud)
- Unsigned plugins work in development mode
- See: https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin

### Grafana Version Compatibility

- Minimum Grafana: 10.4.0
- E2E tests run against multiple Grafana versions (matrix in CI)
- Check `src/plugin.json` dependencies

## Common Issues & Solutions

### Backend not starting
- Check `GF_PLUGIN_APP_DATA_PATH` is writable
- Check backend binary has execute permissions: `chmod +x dist/gpx_*`
- Check Grafana logs: `docker logs jorgeancal-zagalin-app`

### LLM queries failing
- Verify grafana-llm-app is installed and configured
- Check Health API: `/api/plugins/jorgeancal-zagalin-app/health`
- Check backend logs for rate limiting or validation errors

### Conversation storage not persisting
- Check backend API: `/api/plugins/jorgeancal-zagalin-app/resources/storage/conversations`
- Verify `GF_PLUGIN_APP_DATA_PATH` exists and is writable
- Falls back to localStorage if backend unavailable

### Floating chat not appearing
- Only appears on dashboard pages (URL pattern: `/d/:uid/`)
- Check browser console for mount errors
- Check `globalChatMount.tsx` initialization

### E2E tests failing
- Ensure Grafana is running: `npm run server`
- Wait for Grafana to be ready: http://localhost:3000
- Check Playwright logs: `npx playwright show-report`
- E2E requires grafana-llm-app plugin installed

## Important Dependencies

- `@grafana/llm` - LLM integration library
- `@grafana/data`, `@grafana/ui`, `@grafana/runtime` - Grafana SDK
- `grafana-plugin-sdk-go` - Go plugin SDK
- `rxjs` - Reactive streaming for LLM responses
- `marked` - Markdown rendering
- `dompurify` - XSS protection for LLM output

## Documentation (`docs/` folder)

Comprehensive documentation is organized in the `docs/` directory:

```
docs/
├── README.md                      # Documentation index
├── getting-started/               # Setup and installation
│   └── development-setup.md
├── development/                   # Architecture and dev guides
│   └── architecture.md
├── api/                          # API reference docs
│   └── conversation-storage.md
├── testing/                      # Testing guides
│   └── overview.md
├── publishing/                   # Build and deployment
│   └── catalog-submission.md
├── user-guide/                   # End-user documentation
│   └── usage.md
├── CI_PIPELINE_SUMMARY.md        # CI/CD pipeline details
├── GRAFANA_LLM_APP_ANALYSIS.md   # LLM integration analysis
├── ZAGALIN_IMPROVEMENTS.md       # Improvement roadmap
└── TODO.md                       # Task tracking
```

**When to use docs:**
- **Architecture questions** → `docs/development/architecture.md`
- **API details** → `docs/api/` folder
- **Setup instructions** → `docs/getting-started/`
- **CI/CD info** → `docs/CI_PIPELINE_SUMMARY.md`
- **User-facing features** → `docs/user-guide/`

**Keep docs updated:** When implementing features, update relevant docs to reflect changes.

## Development Philosophy

### KISS Mindset (Keep It Simple, Stupid)

This project follows KISS principles to maintain code quality and reduce technical debt:

**Core Principles:**
1. **Solve the actual problem** - Don't add features that aren't needed
2. **Prefer simple solutions** - Use existing patterns before inventing new ones
3. **Avoid over-engineering** - Three similar lines are better than a premature abstraction
4. **Delete unused code** - No commented code, no "just in case" features
5. **Minimal dependencies** - Every dependency is a liability

**When coding:**
- ❌ **Don't:** Create abstractions for one-time use
- ✅ **Do:** Wait for 3+ similar uses before abstracting
- ❌ **Don't:** Add error handling for impossible scenarios
- ✅ **Do:** Validate at boundaries (user input, external APIs)
- ❌ **Don't:** Add feature flags or backward-compatibility shims unnecessarily
- ✅ **Do:** Just change the code when you can
- ❌ **Don't:** Add configurability for every possible option
- ✅ **Do:** Start with sensible defaults, add config only when needed

**Examples:**
```typescript
// ❌ Over-engineered
class QueryBuilderFactory {
  createBuilder(type: QueryType): AbstractQueryBuilder {
    return this.builderRegistry.get(type).instantiate();
  }
}

// ✅ Simple
function buildPromQLQuery(expr: string, range: TimeRange): Query {
  return { expr, from: range.from, to: range.to };
}
```

**Refactoring guide:**
- If you see duplicate code in 3+ places → refactor
- If you see complex logic → simplify first, optimize second
- If you see unused code → delete it immediately

### Security-First Development

All code changes must consider security implications from the start.

**Security Principles:**
1. **Never trust user input** - Validate and sanitize everything
2. **Principle of least privilege** - Users only access what they're authorized for
3. **Defense in depth** - Multiple layers of security
4. **Secure by default** - Safe defaults, opt-in for risky features
5. **Fail securely** - Errors don't leak sensitive information

**Key Security Requirements:**

**1. Authentication & Authorization:**
- ✅ **All backend queries use user's security context** (Issue #16)
- ✅ **No credential storage** - Use session-based auth
- ✅ **User identity logged in audit trail**
- ❌ Never bypass Grafana's permission system
- ❌ Never store API keys in localStorage or frontend code

**2. Query Execution:**
- ✅ **Backend proxy for all datasource queries**
- ✅ **Forward user's auth cookies to Grafana**
- ✅ **Grafana enforces datasource permissions**
- ✅ **Rate limiting per user** (prevent DoS)
- Files: `pkg/plugin/query_proxy.go`, `pkg/plugin/guardrails.go`

**3. Input Validation:**
- Validate all user input (query parameters, messages, config)
- Sanitize LLM output before rendering (XSS prevention)
- Use DOMPurify for markdown rendering
- Limit message size (max 50KB per message)
- Files: `pkg/plugin/validation.go`, frontend uses `dompurify`

**4. Rate Limiting & Abuse Prevention:**
- Token bucket algorithm per user (default: 60 req/min)
- Budget limits for LLM costs
- Query time range clamping (prevent expensive queries)
- File: `pkg/plugin/guardrails.go`

**5. Data Privacy:**
- Conversations stored per-user with access control
- No cross-user data leakage
- PII handling in logs (user IDs hashed)
- File: `pkg/plugin/storage.go`

**6. Audit Logging:**
- Log all queries with user identity
- Log LLM requests with tokens/cost
- Log permission failures
- Timestamps in UTC for audit trail

**Security Checklist for New Features:**
- [ ] Does it accept user input? → Add validation
- [ ] Does it access data? → Check user permissions
- [ ] Does it make external calls? → Use proper auth
- [ ] Does it store data? → Implement access control
- [ ] Does it render user content? → Sanitize for XSS
- [ ] Does it have error messages? → Don't leak sensitive info
- [ ] Does it use secrets? → Use Grafana's secure storage
- [ ] Does it log? → Don't log PII or credentials

**Common Security Anti-Patterns to Avoid:**
```typescript
// ❌ Don't expose credentials
const apiKey = "sk-proj-..."; // Never hardcode keys

// ✅ Use secure storage
const apiKey = secureJsonData.apiKey;

// ❌ Don't trust frontend validation
if (userInput.length < 100) { /* frontend only */ }

// ✅ Validate on backend
func validateInput(input string) error {
  if len(input) > maxSize { return errors.New("too large") }
}

// ❌ Don't bypass permissions
datasource.query(query) // Direct access

// ✅ Use user's context
backendProxy.query(userContext, query) // Enforces permissions
```

**Security Resources:**
- OWASP Top 10: https://owasp.org/www-project-top-ten/
- Grafana Security: https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/
- Plugin Security: https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin

**Recent Security Implementations:**
- Issue #16: Backend query proxy with user identity (see `.claude/ISSUE-16-IMPLEMENTATION.md`)

### Query Injection Prevention & Validation

The plugin implements **hybrid query validation** to prevent injection attacks and enforce query governance for PromQL, LogQL, and TraceQL queries.

**Architecture:**
- **Phase 1**: Parser-based validation (security-critical, always runs)
  - Uses official parsers: `prometheus/prometheus`, `grafana/loki`
  - Validates syntax, checks complexity limits, enforces function allowlists
  - Can operate in strict mode (reject) or sanitization mode (fix and allow)
- **Phase 2**: LLM semantic validation (optional, advisory or strict)
  - Analyzes query performance concerns and best practices
  - Provides improvement suggestions
  - Can run in advisory mode (warnings only) or strict mode (can block)

**Files:**
- `pkg/plugin/query_validation.go` - Core validation engine (~450 lines)
- `pkg/plugin/query_validation_test.go` - Comprehensive test suite (~530 lines)
- `pkg/plugin/query_proxy.go:255-350` - Integration point in request pipeline
- `src/components/AppConfig/AppConfig.tsx:471-598` - UI configuration

**Configuration Options:**
```go
type QueryValidationSettings struct {
    Enabled               bool     // Enable/disable validation
    StrictMode            bool     // true=reject invalid, false=attempt sanitization
    MaxQueryComplexity    int      // Max AST nodes (default: 100)
    AllowedFunctions      []string // PromQL function allowlist (empty=all allowed)
    LogValidationAttempts bool     // Audit log all validation events
    EnableLLMValidation   bool     // Enable semantic validation via LLM
    LLMValidationMode     string   // "advisory" or "strict"
}
```

**Security Pipeline Order:**
1. Extract user identity (`query_proxy.go:220`)
2. Rate limiting per user (`query_proxy.go:227-234`)
3. Datasource allowlist check (`query_proxy.go:242-253`)
4. **Query validation & injection prevention** (`query_proxy.go:255-350`) ← NEW
5. OTel scope enforcement (`query_proxy.go:352-413`)
6. Query execution with user context (`query_proxy.go:426`)
7. Audit logging (`query_proxy.go:438-456`)

**Validation Checks:**
- **PromQL**: Parse with `prometheus/prometheus`, count AST nodes, check function allowlist
- **LogQL**: Parse with `grafana/loki/v3`, count complexity
- **TraceQL**: Falls back to generic validation (Tempo parser disabled due to dependency conflicts)
- **Generic**: Length limits, dangerous pattern detection (SQL injection-like)

**Violation Types:**
- `syntax` - Invalid query syntax
- `injection` - Detected injection attempt
- `complexity` - Query exceeds complexity limit
- `function_blocked` - Used disallowed PromQL function
- `semantic` - LLM blocked query (only in LLM strict mode)
- `length` - Query exceeds size limit

**Audit Logging:**
All validation events are logged with full user context:
- Validation failures: `logQueryValidationFailure()`
- Query sanitizations: `logQuerySanitization()`
- Includes: user ID, org ID, datasource, violation type, original query

**Recommended Settings:**

*Development:*
```json
{
  "enabled": true,
  "strictMode": false,
  "maxQueryComplexity": 100,
  "logValidationAttempts": true,
  "enableLlmValidation": false,
  "llmValidationMode": "advisory"
}
```

*Production:*
```json
{
  "enabled": true,
  "strictMode": true,
  "maxQueryComplexity": 50,
  "logValidationAttempts": true,
  "enableLlmValidation": false,
  "llmValidationMode": "advisory"
}
```

**Security Notes:**
- ✅ Uses official parsers, not regex-based validation
- ✅ Defense in depth - complements rate limiting, datasource governance, OTel enforcement
- ✅ Comprehensive audit logging with user context
- ⚠️ Sanitization mode is risky - default to strict mode in production
- ⚠️ LLM validation is a placeholder for future implementation (requires API budget)
- ⚠️ AGPL-3.0 licensing (Loki, Tempo) - acceptable in Grafana ecosystem

**Testing:**
- Unit tests: >90% coverage across all validation scenarios
- Integration tests: Full request pipeline validation
- Test files: `pkg/plugin/query_validation_test.go`, `pkg/plugin/resources_test.go:334-429`

## AI Tools Configuration

The repository has AI-specific configuration for multiple tools:
- **`.claude/`** - Claude Code configuration
  - `CLAUDE.md` - This file - Comprehensive development guidance
  - `AI-TOOLS-SETUP.md` - Summary of all AI tools setup
  - `settings.local.json` - Permissions for WebFetch and Bash commands
- **`.openai/`** - ChatGPT/Codex instructions (copy/paste)
  - `INSTRUCTIONS.md` - Copy this into ChatGPT conversations
- **`.github/copilot-instructions.md`** - GitHub Copilot configuration
- **`.cursorrules`** - Cursor AI configuration

All AI tools reference `docs/` for detailed documentation.

Keep these in version control to help all developers and AI tools work effectively with this codebase.
