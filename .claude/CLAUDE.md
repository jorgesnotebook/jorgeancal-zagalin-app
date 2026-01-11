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
- **Security Layers**: Rate limiting, datasource governance, query validation, OTel enforcement

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
- Context manager for Prometheus/Loki/Tempo metadata extraction
- Rate limiting, query validation, and guardrails system
- File-based user storage for conversations

## Comprehensive Documentation

**IMPORTANT**: Before modifying features, read the complete documentation:

- **[Features Overview](../docs/FEATURES_OVERVIEW.md)** - Complete inventory of ALL features, their status, and purpose
- **[API Endpoints](../docs/api/ENDPOINTS.md)** - Complete API reference with request/response examples
- **[Architecture](../docs/development/architecture.md)** - System architecture and design decisions

**Quick Reference**:

- Feature count: 18 active features + 4 implemented but unused
- Code volume: 7,887 lines (backend) + 3,725 lines (frontend services)
- Configuration fields: 25+ settings (only 1 required for basic use)
- API endpoints: 20+ endpoints across LLM, Storage, Query, Health, and Runs

**Before implementing new features**:

1. Check if feature already exists (see FEATURES_OVERVIEW.md)
2. Review API documentation (see api/ENDPOINTS.md)
3. Understand security pipeline (Rate limiting → Allowlist → Validation → OTel → Execute)

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

# Backend tests with coverage
mage -v coverage
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

Uses **backend proxy pattern** for secure LLM integration:

- Frontend calls backend `/llm/chat` endpoint
- Backend constructs system prompts securely (Senior Staff SRE persona)
- Backend auto-detects skills and injects context
- Backend proxies to grafana-llm-app with user authentication
- Supports function calling for structured tool execution
- SSE streaming for real-time responses

**Files**:

- `pkg/plugin/assistant.go` - LLM HTTP handler and orchestration
- `pkg/plugin/assistant_prompts.go` - System prompts (secure, server-side)
- `pkg/plugin/assistant_tools.go` - Function calling tool definitions
- `pkg/plugin/llm_client.go` - grafana-llm-app API client with SSE
- `src/services/assistantService.ts` - Frontend API client
- `src/services/zagalinTools.ts` - Tool execution handlers (frontend)

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

## Project Structure

```
src/
 components/          # React components
    App/            # Main app component
    AppConfig/      # Configuration UI
    FloatingChat/   # Floating chat components
    AskPanel/       # Dashboard panel component
 pages/              # Page components
    ChatPage.tsx    # Full-screen chat
    ConfigPage/     # Configuration page
    AssistantChatPage.tsx
 services/           # Business logic
    conversationStorage.ts    # Conversation persistence
    contextService.ts         # Grafana context extraction
    assistantService.ts       # Backend LLM API client
    zagalinTools.ts           # Function calling tool handlers
 hooks/              # React hooks
 types/              # TypeScript types
 module.tsx          # Plugin entry point

pkg/
 main.go             # Plugin binary entry
 plugin/
     app.go          # Main plugin app
     resources.go    # HTTP route handlers
     storage.go      # Conversation storage
     guardrails.go   # Rate limiting
     assistant.go    # LLM chat endpoint handler (/llm/chat)
     assistant_prompts.go       # System prompts (SRE persona, skills)
     assistant_tools.go         # Function calling tool definitions
     llm_client.go              # grafana-llm-app API client with SSE
     query_proxy.go  # Query proxy with security pipeline
     query_validation.go        # Query injection prevention
     query_validation_test.go   # Validation tests
     datasource.go   # Datasource type detection
     otel_enforcement.go        # OTel scope enforcement
     context/        # Context extraction
         manager.go
         metrics.go
         logs.go
         traces.go
```

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

- **Don't:** Create abstractions for one-time use
- **Do:** Wait for 3+ similar uses before abstracting
- **Don't:** Add error handling for impossible scenarios
- **Do:** Validate at boundaries (user input, external APIs)
- **Don't:** Add feature flags or backward-compatibility shims unnecessarily
- **Do:** Just change the code when you can
- **Don't:** Add configurability for every possible option
- **Do:** Start with sensible defaults, add config only when needed

**Refactoring guide:**

- If you see duplicate code in 3+ places → refactor
- If you see complex logic → simplify first, optimize second
- If you see unused code → delete it immediately

### Code Comments - Quality Over Quantity

**Write comments that clarify complex logic, not obvious statements.**

```go
//  BAD - States the obvious
// Set the user to admin
user = "admin"

//  BAD - Repeats what the code already says
// Loop through all users
for _, user := range users {

//  BAD - Unnecessary function description
// GetUser returns a user by ID
func GetUser(id string) User {

//  GOOD - Explains WHY, not WHAT
// Use cached value if less than 5 minutes old to reduce API calls
if time.Since(cache.lastRefresh) < 5*time.Minute {

//  GOOD - Clarifies non-obvious behavior
// Parser expects closing brace even inside strings, so we track string state
if !inString && ch == '}' {

//  GOOD - Documents important edge cases
// Empty allowlist means all functions are allowed (backwards compatibility)
if len(allowedFunctions) == 0 {
    return nil
}
```

**When to comment:**

- Complex algorithms or non-obvious logic
- Important edge cases or gotchas
- Security-critical code sections
- Workarounds for external library bugs
- Performance optimizations with trade-offs

**When NOT to comment:**

- Function signatures (use clear names instead)
- Variable declarations (use descriptive names)
- Simple loops or conditionals
- Code that reads like plain English

### Decision-Making Guide for AI Assistants

**When to ask questions:**

- User request is ambiguous and has multiple valid interpretations
- Feature could be implemented in several architecturally different ways
- Unclear which existing pattern to follow
- Security or data privacy implications that user should approve
- Breaking changes that affect existing functionality

**When to proceed without asking:**

- Bug fix with clear root cause and solution
- Adding obvious missing functionality (e.g., missing test for existing feature)
- Following established patterns in the codebase
- Refactoring that doesn't change behavior
- Documentation updates

**When to use EnterPlanMode:**

- New feature implementation (unless trivial)
- Changes affecting multiple files or components
- Architectural decisions needed
- Multiple valid approaches exist
- User would benefit from seeing the plan before implementation

**When NOT to use EnterPlanMode:**

- Single-line or few-line fixes
- Adding obvious missing tests
- Documentation updates
- Simple bug fixes with clear solution
- User provided very specific detailed instructions

### Security-First Development

All code changes must consider security implications from the start.

**Security Principles:**

1. **Never trust user input** - Validate and sanitize everything
2. **Principle of least privilege** - Users only access what they're authorized for
3. **Defense in depth** - Multiple layers of security
4. **Secure by default** - Safe defaults, opt-in for risky features
5. **Fail securely** - Errors don't leak sensitive information

**Key Security Requirements:**

**Authentication & Authorization:**

- All backend queries use user's security context (see `pkg/plugin/query_proxy.go::extractUserIdentity`)
- No credential storage - use session-based auth
- User identity logged in audit trail
- Never bypass Grafana's permission system
- Never store API keys in localStorage or frontend code

**Query Execution Pipeline (pkg/plugin/query_proxy.go::handleQuery):**

1. Extract user identity from request context
2. Rate limiting per user (token bucket algorithm)
3. Datasource allowlist check (if configured)
4. **Query validation & injection prevention** (PromQL/LogQL/TraceQL)
5. OTel scope enforcement (if enabled)
6. Query execution with user's security context forwarded to Grafana
7. Audit logging with user identity

**Input Validation:**

- Validate all user input (query parameters, messages, config)
- Sanitize LLM output before rendering (use DOMPurify)
- Limit message size (max 50KB per message)
- Files: `pkg/plugin/query_validation.go`, frontend uses `dompurify`

**Rate Limiting & Abuse Prevention:**

- Token bucket algorithm per user (default: 60 req/min)
- Budget limits for LLM costs
- Query time range clamping (prevent expensive queries)
- File: `pkg/plugin/guardrails.go`

**Data Privacy:**

- Conversations stored per-user with access control
- No cross-user data leakage
- PII handling in logs (user IDs hashed)
- File: `pkg/plugin/storage.go`

**Security Checklist for New Features:**

- [ ] Does it accept user input? → Add validation
- [ ] Does it access data? → Check user permissions
- [ ] Does it make external calls? → Use proper auth
- [ ] Does it store data? → Implement access control
- [ ] Does it render user content? → Sanitize for XSS
- [ ] Does it have error messages? → Don't leak sensitive info
- [ ] Does it use secrets? → Use Grafana's secure storage
- [ ] Does it log? → Don't log PII or credentials

**Security Resources:**

- OWASP Top 10: https://owasp.org/www-project-top-ten/
- Grafana Security: https://grafana.com/docs/grafana/latest/setup-grafana/configure-security/

## Query Injection Prevention & Validation

The plugin implements **manual pattern-based query validation** to prevent injection attacks and enforce query governance for PromQL, LogQL, and TraceQL queries.

### Architecture

**No External Dependencies** - Uses manual pattern matching instead of external parsers to keep binary size small and avoid licensing issues.

**Per-Query-Type Controls** - Each query language can be enabled/disabled independently:

- `Enabled` - Master switch for all validation
- `EnablePromQLValidation` - Toggle PromQL validation
- `EnableLogQLValidation` - Toggle LogQL validation
- `EnableTraceQLValidation` - Toggle TraceQL validation

**Disabled by Default** - All validation types are off by default for backwards compatibility.

### Configuration

```go
type QueryValidationSettings struct {
    Enabled                 bool     `json:"enabled"`                 // Master switch
    EnablePromQLValidation  bool     `json:"enablePromqlValidation"`  // PromQL toggle
    EnableLogQLValidation   bool     `json:"enableLogqlValidation"`   // LogQL toggle
    EnableTraceQLValidation bool     `json:"enableTraceqlValidation"` // TraceQL toggle
    StrictMode              bool     `json:"strictMode"`              // reject vs sanitize
    MaxQueryComplexity      int      `json:"maxQueryComplexity"`      // Max complexity score
    AllowedFunctions        []string `json:"allowedFunctions,omitempty"` // PromQL function allowlist
    LogValidationAttempts   bool     `json:"logValidationAttempts"`
    EnableLLMValidation     bool     `json:"enableLlmValidation"`     // Future: LLM semantic validation
    LLMValidationMode       string   `json:"llmValidationMode"`       // "advisory" or "strict"
}
```

### Validation Methods

**PromQL Validation (`validatePromQL`):**

- Balanced braces, brackets, parentheses (respects string escaping)
- Invalid operator detection (`===`, `!==`, `<>`, `++`, `--`)
- Complexity estimation (count operators, functions, selectors)
- Function allowlist enforcement (pattern matching for `function(`)

**LogQL Validation (`validateLogQL`):**

- Log selector requirement (`{...}` must be present)
- Balanced braces (respects string escaping)
- Filter operator detection (`|=`, `!=`, `|~`, `!~`)
- Complexity estimation (count selectors, filters, operators)

**TraceQL Validation (`validateTraceQL`):**

- Balanced braces (respects string escaping)
- Valid attribute prefix detection (`span.`, `resource.`, intrinsic fields)
- Invalid operator detection (`===`, `!==`, `<>`)
- Complexity estimation (count selectors, operators)

**Generic Validation (`validateGeneric`):**

- Query length limits (max 10KB)
- SQL injection pattern detection (`DROP TABLE`, `UNION SELECT`, etc.)
- Dangerous command patterns

### Violation Types

- `syntax` - Invalid query syntax (unbalanced braces, invalid operators)
- `complexity` - Query exceeds complexity limit
- `function_blocked` - Used disallowed PromQL function
- `length` - Query exceeds size limit
- `injection` - Detected injection attempt pattern

### Files

**Core Implementation:**

- `pkg/plugin/query_validation.go` - Validation engine (~450 lines)
  - `NewQueryValidator()` - Constructor with defaults
  - `ValidateQuery()` - Main entry point, routes to specific validators
  - `validatePromQL()`, `validateLogQL()`, `validateTraceQL()` - Language-specific validation
  - `hasBalancedBraces()` - Helper for balanced braces with string escaping support
  - Complexity counters and sanitization helpers

**Testing:**

- `pkg/plugin/query_validation_test.go` - Comprehensive test suite (~560 lines)
  - Valid query tests
  - Invalid syntax tests
  - Complexity limit tests
  - Function allowlist tests
  - Violation type tests
  - Integration tests

**Integration:**

- `pkg/plugin/query_proxy.go::handleQuery` - Validation integrated into security pipeline
- `src/components/AppConfig/AppConfig.tsx` - UI configuration with per-type toggles

**Supporting:**

- `pkg/plugin/settings.go` - Settings structure and defaults
- `pkg/plugin/datasource.go` - Datasource type detection for routing queries to correct validator
- `pkg/plugin/resources_test.go` - Integration tests

### Audit Logging

All validation events are logged with full user context:

```go
func (a *App) logQueryValidationFailure(user, datasource, result)
func (a *App) logQuerySanitization(user, datasource, result)
```

Logs include:

- User ID, org ID, datasource UID
- Violation type
- Original query (for audit)
- Sanitized query (if applicable)
- Timestamp in UTC

### Recommended Settings

**Development:**

```json
{
  "enabled": true,
  "enablePromqlValidation": true,
  "enableLogqlValidation": true,
  "enableTraceqlValidation": true,
  "strictMode": false,
  "maxQueryComplexity": 100,
  "logValidationAttempts": true
}
```

**Production:**

```json
{
  "enabled": true,
  "enablePromqlValidation": true,
  "enableLogqlValidation": true,
  "enableTraceqlValidation": true,
  "strictMode": true,
  "maxQueryComplexity": 50,
  "logValidationAttempts": true
}
```

### Security Notes

- Manual pattern-based validation (no external parser dependencies)
- Defense in depth - complements rate limiting, datasource governance, OTel enforcement
- Comprehensive audit logging with user context
- Zero external dependencies (no AGPL licensing concerns)
- Small binary size impact (~0 bytes, no new dependencies)
- Sanitization mode is risky - default to strict mode in production
- Pattern matching is less precise than AST parsing but sufficient for security
- LLM validation is placeholder for future implementation

### Testing Coverage

- Unit tests: >90% code coverage
- Valid queries: Simple, complex, nested, aggregations
- Invalid queries: Syntax errors, injection attempts, complexity violations
- Edge cases: Empty queries, very long queries, escaped strings
- Integration: Full request pipeline with validation enabled

## Grafana Version Compatibility

The plugin implements **defensive version detection** to warn users about unsupported Grafana versions without blocking functionality.

### Version Detection Strategy

**Frontend-First Approach:**
- Primary detection from `config.buildInfo.version` (Grafana runtime)
- Optional backend detection via `X-Grafana-Version` HTTP header
- Graceful fallback to "unknown" if version unavailable (respects disabled reporting)

**Minimum Supported Version**: Grafana 10.4.0

### Implementation

**Frontend:**
- `src/services/versionDetector.ts` - Version detection and compatibility checking
- `src/services/versionReporter.ts` - HTTP header injection for backend
- `src/components/VersionWarning/VersionWarning.tsx` - Warning UI component

**Backend:**
- `pkg/plugin/version.go` - Version parsing and comparison logic
- `pkg/plugin/version_detector.go` - HTTP header extraction and caching
- `pkg/plugin/app.go` - VersionDetector integration
- `pkg/plugin/resources.go` - Middleware wraps all handlers

### Key Features

- **Defensive, Not Restrictive**: Warnings displayed but features not auto-disabled
- **Privacy-Respecting**: Only sends version if available (respects disabled reporting)
- **Configuration-Based Control**: User controls features via settings (not automatic)
- **Health Check Integration**: Version info included in `/health` endpoint

### User Experience

**Configuration Page**: Warning alert shown at top if version unsupported or unavailable
**Health Endpoint**: Returns version object with `detected`, `isAvailable`, `isSupported`, `warnings`
**Backend Logs**: Version detected and logged on startup with minimum version info

### Testing

- Frontend: `src/services/versionDetector.test.ts` (comprehensive unit tests)
- Backend: `pkg/plugin/version_test.go`, `pkg/plugin/version_detector_test.go`
- Integration: `pkg/plugin/resources_test.go` (health endpoint + middleware tests)

### Documentation

See **[docs/VERSION_COMPATIBILITY.md](../docs/VERSION_COMPATIBILITY.md)** for complete guide including:
- Supported versions and compatibility matrix
- Troubleshooting version detection issues
- API reference and examples
- Privacy considerations

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

**Pre-commit Hooks**:

- `.git-hooks/pre-commit` - Runs typecheck, lint, format checks
- Prevents committing broken code
- Skip with `git commit --no-verify` (not recommended)

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

### Query validation blocking valid queries

- Check which validation types are enabled in plugin settings
- Review backend logs for validation failure details
- Consider adjusting `maxQueryComplexity` if queries are legitimately complex
- Use sanitization mode in development, strict mode in production

## Important Dependencies

- `@grafana/llm` - LLM integration library
- `@grafana/data`, `@grafana/ui`, `@grafana/runtime` - Grafana SDK
- `grafana-plugin-sdk-go` - Go plugin SDK
- `rxjs` - Reactive streaming for LLM responses
- `marked` - Markdown rendering
- `dompurify` - XSS protection for LLM output

## Documentation (`docs/` folder)

Comprehensive documentation is organized in the `docs/` directory:

**When to use docs:**

- **Architecture questions** → `docs/development/architecture.md`
- **API details** → `docs/api/` folder
- **Setup instructions** → `docs/getting-started/`
- **CI/CD info** → `docs/CI_PIPELINE_SUMMARY.md`
- **User-facing features** → `docs/user-guide/`

**Keep docs updated:** When implementing features, update relevant docs to reflect changes.

## AI Tools Configuration

The repository has AI-specific configuration for multiple tools:

- **`.claude/`** - Claude Code configuration
  - `CLAUDE.md` - This file - Main project instructions
  - `README.md` - Overview of Claude configuration and rule system
  - `settings.local.json` - Permissions for WebFetch and Bash commands
  - `rules/` - Modular, topic-specific development standards
    - `grafana-plugin-standards.md` - Official Grafana development standards
    - `grafana-llm-integration.md` - LLM integration with grafana-llm-app
    - `app-plugin-development.md` - App plugin development guide
    - `clean-code-principles.md` - KISS methodology and clean code
    - `code-quality-standards.md` - Formatting, linting, testing
    - `plugin-maintenance.md` - Updates and backwards compatibility
    - `e2e-testing.md` - End-to-end testing with Playwright
- **`.openai/`** - ChatGPT/Codex instructions (copy/paste)
  - `INSTRUCTIONS.md` - Copy this into ChatGPT conversations
- **`.github/copilot-instructions.md`** - GitHub Copilot configuration
- **`.cursorrules`** - Cursor AI configuration

All AI tools reference `docs/` for detailed documentation.

### Modular Rules System

The `.claude/rules/` directory contains focused, path-scoped documentation:

- Rules automatically load based on file paths (using YAML frontmatter)
- Each rule file covers a specific topic (Grafana standards, LLM integration, testing, etc.)
- Rules can be organized in subdirectories for better structure
- See `.claude/README.md` for complete documentation of the rule system

Keep these in version control to help all developers and AI tools work effectively with this codebase.
