# Changelog

All notable changes to Zagalin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation system with feature inventory, API reference, and configuration guides
- Privacy-conscious usage logging with signal type detection (metrics/logs/traces/dashboard/investigation)
- Frontend orchestration system for structured investigation workflows with planning and step execution
- Artifact extraction and display for queries, links, and trace IDs

### Changed
- Disabled Direct API configuration option in UI (still available via backend for future use)
- Enhanced LLM integration with smart routing between dashboard questions and investigations

### Fixed
- TypeScript type consistency across ExecutionPlan interfaces
- Linting errors in frontend orchestration components
- React effect setState warnings with async wrappers

## [0.0.2] - 2025-12-27

### 🎯 Release Theme: "Security & Governance"

**This release transforms Zagalin from a proof-of-concept into a production-ready observability assistant with enterprise-grade security controls.**

Zagalin 0.0.2 introduces comprehensive security features that make it safe to deploy in production environments with multiple teams, strict compliance requirements, and sensitive observability data. Every query now flows through a six-step security pipeline, and administrators gain fine-grained control over datasource access, query complexity, and OpenTelemetry compliance.

### Added

#### Security & Governance

**Query Validation System** - Pattern-based validation for all query languages
- Manual pattern-based validation for PromQL, LogQL, and TraceQL (zero external dependencies)
- Syntax validation with balanced braces and operator detection
- Complexity scoring to prevent expensive queries (configurable max complexity)
- Function allowlists for PromQL (e.g., only allow safe aggregation functions)
- SQL injection pattern detection (UNION SELECT, DROP TABLE, etc.)
- Configurable per query language (enable/disable PromQL/LogQL/TraceQL independently)
- Two enforcement modes:
  - **Strict mode**: Reject invalid queries with detailed error messages
  - **Sanitization mode**: Attempt to fix queries (use with caution)
- Full audit logging with user context, violation types, and original queries
- Comprehensive test coverage (>90% code coverage)
- Files: `pkg/plugin/query_validation.go` (450 lines), `pkg/plugin/query_validation_test.go` (560 lines)

**OpenTelemetry Scope Enforcement** - Automatic service and environment labeling
- Enforces `service_name` and `deployment_environment_name` in all queries
- Configurable defaults and validation rules per organization
- Two enforcement modes:
  - **Full enforcement**: Inject labels, reject queries without scope
  - **Validation-only mode**: Warn but don't block (for gradual rollout)
- Scope extraction from existing queries (preserves user-specified values)
- Works seamlessly with PromQL, LogQL, and TraceQL
- Audit logging for all scope enforcement actions
- Example transformation:
  ```promql
  # Before
  rate(http_requests_total[5m])

  # After (with enforcement)
  rate(http_requests_total{service_name="my-service",deployment_environment_name="production"}[5m])
  ```
- Files: `pkg/plugin/otel_enforcement.go` (343 lines)

**Datasource Governance** - Allowlist system for approved datasources
- Restrict queries to approved Prometheus/Loki/Tempo instances
- Per-organization datasource control
- Default datasource fallback when no preference specified
- Automatic datasource type detection (Prometheus, Loki, Tempo, Mimir, Cortex, Jaeger)
- Audit logging for blocked datasource access attempts
- Prevents accidental queries to dev/staging from production dashboards
- Files: `pkg/plugin/datasource.go`, integrated into `pkg/plugin/query_proxy.go`

**Six-Step Security Pipeline** - Comprehensive query validation flow
1. **Authentication**: Extract user identity (user ID, org ID, email) from Grafana context
2. **Rate Limiting**: Token bucket algorithm (60 req/min default, configurable per user)
3. **Datasource Allowlist**: Verify datasource is approved for this organization
4. **Query Validation**: Pattern-based injection prevention and complexity limits
5. **OTel Scope Enforcement**: Inject service_name and deployment_environment_name
6. **Query Execution**: Forward to Grafana with full audit trail
- Every step logs actions with full user context for compliance
- Files: `pkg/plugin/query_proxy.go` (457 lines)

#### Storage & Persistence

**Conversation History Management** - Dual-tier storage with automatic migration
- **Backend Storage**: File-based storage with per-user isolation
  - User-specific storage directories (`$GF_PLUGIN_APP_DATA_PATH/user_<login>_<hash>/`)
  - Conversation CRUD operations (Create, Read, Update, Delete)
  - Pin favorite conversations to top of list
  - Title auto-generation from first user message
  - Manual title editing with inline editor
  - Context preservation (dashboard UID/title, panel ID/title, time range)
  - Metadata tracking (message count, last message preview, update timestamps)
  - Files: `pkg/plugin/storage.go` (600+ lines)
- **LocalStorage Fallback**: Browser-based storage when backend unavailable
  - Automatic fallback detection
  - Same API interface as backend storage
  - Files: `src/services/conversationStorage.ts`
- **Automatic Migration**: LocalStorage → Backend when backend becomes available
  - One-time migration on first backend availability
  - Preserves all conversation history, context, and metadata
  - Files: `src/services/storageApiClient.ts`
- **Bulk Operations**:
  - Delete individual conversations with confirmation modal
  - Delete all conversations (with double confirmation)
  - Export conversations (JSON/Markdown) - placeholder for future

#### Developer Experience

**AI Development Tools** - Standardized configurations for AI assistants
- **Claude Code** (`.claude/CLAUDE.md`, 500+ lines):
  - Complete architecture reference
  - Development philosophy (KISS principle, security-first)
  - Common commands and workflows
  - File structure and patterns
  - Security checklist for new features
  - Query validation and OTel enforcement documentation
- **ChatGPT Instructions** (`.openai/INSTRUCTIONS.md`):
  - Copy-paste instructions for ChatGPT conversations
  - Project overview and tech stack
  - Key patterns and architecture
- **GitHub Copilot** (`.github/copilot-instructions.md`):
  - Auto-loaded by Copilot in IDE
  - Code style and patterns
- **Cursor AI** (`.cursorrules`):
  - Cursor-specific configuration
- Makes AI assistants immediately productive with context

**Local CI Pipeline** - Run full CI checks locally before pushing
- `ci-local.sh` script:
  - Runs: npm ci → typecheck → lint → test:ci → build → mage coverage → mage buildAll → plugin validation
  - Catches issues before CI/CD runs
  - Saves time and CI minutes
- Pre-commit hooks (`.git-hooks/pre-commit`):
  - Type checking with `tsc --noEmit`
  - Linting with ESLint
  - Formatting check with Prettier
  - Prevents committing broken code
  - Skip with `git commit --no-verify` (not recommended)
- Files: `ci-local.sh`, `.git-hooks/pre-commit`

#### Backend Improvements

**Privacy-Conscious Usage Logging** - Track usage without exposing queries
- Signal type detection: metrics, logs, traces, dashboard, investigation, general
- Detects from dashboard context (datasource types) and message keywords
- Never logs actual query content or message text
- Only logs: user, org ID, signal type, message length, dashboard presence
- Example log output:
  ```json
  {
    "level": "info",
    "msg": "LLM chat request",
    "user": "alice@company.com",
    "orgId": 1,
    "skill": "troubleshoot",
    "signalType": "metrics",
    "hasDashboardContext": true,
    "messageLength": 42,
    "historyLength": 3
  }
  ```
- Enables usage analytics without privacy violations
- Files: `pkg/plugin/assistant.go` (`detectSignalType()` function, ~100 lines)

### Changed

- Enhanced security pipeline with comprehensive audit logging
- Improved plugin configuration UI with security controls (query validation, OTel enforcement, datasource governance)
- Updated API endpoints to include user context in all responses
- Conversation storage now includes context metadata (dashboard, time range, panel)
- Better error messages for validation failures (includes violation type and fix suggestions)

### Fixed

- E2E test compatibility across Grafana versions (10.4.0 - 12.x)
- Breadcrumb navigation checks in E2E tests (made more robust)
- Input validation edge cases (empty queries, very long queries)
- Conversation history persistence issues (duplicate IDs, race conditions)
- Modal UX improvements for deletion confirmations (clearer warnings)
- Build provenance attestation for plugin security compliance

### Security

- ✅ All user queries now go through six-step security pipeline
- ✅ User identity tracked in audit logs (PII-safe with hashed user IDs)
- ✅ No credential storage in frontend (session-based auth only)
- ✅ XSS protection via DOMPurify for LLM output sanitization
- ✅ Query injection prevention (SQL, PromQL, LogQL, TraceQL)
- ✅ Rate limiting prevents abuse (60 req/min default, configurable)
- ✅ Datasource access control (allowlist enforcement)
- ✅ OpenTelemetry compliance (automatic scope labeling)
- ✅ Comprehensive audit trail (every query logged with user context)

### Performance

- Rate limiting: 60 requests/minute per user (configurable)
- Query complexity limits prevent expensive operations (default: 50)
- Efficient context caching reduces redundant API calls
- Minimal overhead from validation (pattern-matching, no AST parsing)

### Documentation

- Complete API documentation in `.claude/CLAUDE.md`
- Query validation guide with examples and best practices
- OTel enforcement configuration guide
- Datasource governance setup instructions
- Security pipeline architecture diagram
- Audit logging format reference

### Configuration

**New Settings** (all disabled by default for backward compatibility):

```json
{
  "queryValidation": {
    "enabled": false,                    // Master switch for all validation
    "enablePromqlValidation": false,     // Toggle PromQL validation
    "enableLogqlValidation": false,      // Toggle LogQL validation
    "enableTraceqlValidation": false,    // Toggle TraceQL validation
    "strictMode": false,                 // true = reject, false = sanitize
    "maxQueryComplexity": 50,            // Max complexity score
    "allowedFunctions": [],              // PromQL function allowlist (empty = all allowed)
    "logValidationAttempts": true        // Log all validation events
  },
  "otelEnforcement": {
    "enabled": false,                    // Master switch
    "requireServiceName": true,          // Enforce service_name label
    "requireEnvironmentName": true,      // Enforce deployment_environment_name label
    "defaultServiceName": "unknown",     // Default if not specified
    "defaultEnvironmentName": "unknown", // Default if not specified
    "rejectIfNoScope": false,            // true = reject, false = inject defaults
    "validationOnlyMode": false          // true = warn only, false = enforce
  },
  "allowedDatasources": [],              // Empty = all allowed, ["uid1", "uid2"] = allowlist
  "defaultDatasource": ""                // Default datasource UID if none specified
}
```

**Recommended Production Settings**:
```json
{
  "queryValidation": {
    "enabled": true,
    "enablePromqlValidation": true,
    "enableLogqlValidation": true,
    "enableTraceqlValidation": true,
    "strictMode": true,                  // Reject invalid queries
    "maxQueryComplexity": 50,
    "logValidationAttempts": true
  },
  "otelEnforcement": {
    "enabled": true,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "your-service-name",
    "defaultEnvironmentName": "production",
    "rejectIfNoScope": true              // Strict enforcement
  },
  "allowedDatasources": [
    "prometheus-prod-uid",
    "loki-prod-uid",
    "tempo-prod-uid"
  ]
}
```

## [0.0.1] - 2025-12-24

### 🎯 Release Theme: "Foundation"

**The first release of Zagalin - bringing AI assistance directly into Grafana's observability workflow.**

Zagalin 0.0.1 establishes the core foundation: a context-aware AI assistant that understands your dashboards, generates queries in natural language, and helps you troubleshoot issues using SRE best practices. The floating chat interface makes AI help available everywhere you work in Grafana.

### Added

#### Core Features

**Context-Aware Chat** - AI assistant understands your current Grafana context
- Dashboard awareness:
  - Dashboard UID, title, tags
  - Panel list with types and queries
  - Time range from URL parameters
  - Template variables and their values
- Panel context:
  - Panel title, type, datasource
  - PromQL/LogQL/TraceQL queries
  - Target configurations
- Time range awareness:
  - Parses `from` and `to` URL parameters
  - Relative ranges (e.g., "now-6h")
  - Absolute timestamps
- Files: `src/services/contextService.ts`, `src/services/useGrafanaContext.ts`

**Floating Chat Interface** - Global chat button on every dashboard
- Portal-based mounting (persists across Grafana navigation)
- Appears on all dashboard pages (`/d/:uid/`)
- Collapsible sidebar with conversation list
- Typewriter effect for streaming responses
- Markdown rendering with syntax highlighting
- Code block detection and formatting
- Responsive design (mobile-friendly)
- Files: `src/globalChatMount.tsx`, `src/components/FloatingChat/`

**Full Chat Page** - Dedicated page for extended conversations
- Available at Apps → Zagalin → Chat
- Full-screen chat experience
- Same context awareness as floating chat
- Conversation history preserved
- Files: `src/pages/AssistantChatPage.tsx`

#### LLM Integration

**Provider-Agnostic Support** - Works with any LLM through grafana-llm-app
- Supported providers:
  - OpenAI (GPT-4, GPT-3.5)
  - Azure OpenAI
  - Anthropic Claude (Claude 3.5 Sonnet, Claude 3 Opus)
  - Grafana Managed LLM
  - Custom OpenAI-compatible endpoints
- Streaming SSE responses for real-time interaction
- Files: `pkg/plugin/llm_client.go`, `src/services/assistantService.ts`

**System Prompt Engineering** - Senior Staff SRE persona
- Prompt defines Zagalin as experienced SRE with years of on-call experience
- Emphasizes:
  - Reliability > Performance > Features
  - Actionable insights over theory
  - Teaching ("explain the why")
  - Production risk awareness
- Observability workflow: Metrics → Logs → Traces (SRE pyramid)
- Files: `pkg/plugin/assistant_prompts.go`

**Skills System** - Auto-detected user intent
- **explain_panel**: Explains what a dashboard panel shows
  - Analyzes query, datasource, visualization type
  - Interprets current data and trends
- **generate_query**: Creates PromQL/LogQL/TraceQL queries
  - Natural language → query language
  - Suggests aggregations and filters
- **troubleshoot**: Investigates issues following SRE methodology
  - Metrics → Logs → Traces workflow
  - Root cause analysis
- **analyze_dashboard**: Provides dashboard-level insights
  - Overall health assessment
  - Suggestions for improvements
- Skill auto-detection from message content and context
- Files: `pkg/plugin/assistant_prompts.go` (skill detection logic)

**Function Calling** - Structured tool execution
- Available tools:
  - `navigate_to_dashboard`: Navigate to specific dashboard by UID
  - `create_promql_query`: Generate and validate PromQL queries
  - `create_logql_query`: Generate and validate LogQL queries
  - `open_explore`: Open Explore view with pre-filled query
- Tool definitions in backend, execution in frontend
- Files: `pkg/plugin/assistant_tools.go`, `src/services/zagalinTools.ts`

#### Query Generation

**PromQL Generation** - Natural language to Prometheus queries
- Understands metric names and label selectors
- Suggests aggregations (sum, rate, avg, max, min)
- Function recommendations (rate, increase, histogram_quantile)
- Time range integration
- Example:
  - Input: "Show me the 95th percentile response time for the API service"
  - Output: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service="api"}[5m]))`

**LogQL Generation** - Natural language to Loki log queries
- Log stream selector generation
- Filter operators (|=, !=, |~, !~)
- Parser suggestions (json, logfmt, pattern)
- Line format and label extraction
- Example:
  - Input: "Show me error logs from the payment service"
  - Output: `{service="payment"} |= "error" | json`

**TraceQL Generation** - Natural language to Tempo trace queries
- Span attribute filters
- Resource selectors
- Duration and status filters
- Intrinsic field queries
- Example:
  - Input: "Find slow traces for the checkout endpoint"
  - Output: `{span.http.target="/checkout" && duration > 1s}`

#### Backend Services

**Context Manager** - Extracts and caches observability context
- Prometheus metric discovery:
  - Extracts metric names from Prometheus API
  - Label enumeration for common metrics
  - Caches results with configurable TTL
- Loki log stream enumeration:
  - Discovers log streams and labels
  - Caches for performance
- Tempo trace datasource integration:
  - Detects Tempo datasources
  - Provides trace query context
- Background refresh (configurable interval, default: 15 minutes)
- Endpoints:
  - `GET /context/status` - View current context cache
  - `POST /context/refresh` - Force refresh
- Files: `pkg/plugin/context/manager.go`, `pkg/plugin/context/metrics.go`, `pkg/plugin/context/logs.go`, `pkg/plugin/context/traces.go`

**Rate Limiting** - Token bucket algorithm per user
- Default: 60 requests/minute per user
- Configurable per organization
- Budget tracking for LLM costs
- Prevents abuse and cost overruns
- Files: `pkg/plugin/guardrails.go`

**Health Monitoring** - Status endpoints for observability
- `GET /health` - Plugin health check (uptime, version, status)
- `GET /settings` - Current plugin configuration
- grafana-llm-app availability detection
- Files: `pkg/plugin/resources.go`

#### Configuration

**Plugin Settings** - Flexible configuration system
- **LLM Backend Mode**:
  - `grafana-llm-app`: Use Grafana's LLM App plugin (recommended)
  - `direct`: Call LLM providers directly with API keys (future)
  - `disabled`: Disable LLM functionality
- **Rate Limiting**:
  - Requests per minute (default: 60)
  - Budget limits (cost tracking)
- **Context Manager**:
  - Refresh interval (default: 15 minutes)
  - Enable/disable metric/log/trace discovery
- **Model Selection**:
  - Choose from available models in grafana-llm-app
- **Advanced**:
  - Temperature (0.0-1.0, default: 0.7)
  - Max tokens (1000-4000, default: 2000)
- Files: `pkg/plugin/settings.go`, `src/components/AppConfig/AppConfig.tsx`

### Architecture

**Hybrid Plugin** - React frontend + Go backend
- **Frontend**:
  - React 18 with TypeScript
  - Grafana UI components (@grafana/ui)
  - RxJS for streaming LLM responses
  - Webpack 5 bundling
  - Jest for unit tests
- **Backend**:
  - Go 1.21+
  - grafana-plugin-sdk-go
  - Mage build system
  - Go test framework with coverage
- Communication via HTTP REST API
- Files:
  - Frontend: `src/`
  - Backend: `pkg/`

**Security-First Design**
- No credential storage (uses Grafana's secure storage)
- Session-based authentication (forwarded from Grafana)
- User context in all datasource queries (respects Grafana permissions)
- No conversation persistence (privacy by default)
- XSS protection via DOMPurify
- Files: `pkg/plugin/query_proxy.go` (security pipeline)

**Testing Infrastructure**
- **Frontend Unit Tests**: Jest with React Testing Library
  - Component tests
  - Service tests
  - Hook tests
  - Coverage reports
- **Backend Unit Tests**: Go test framework
  - Handler tests
  - Service tests
  - Integration tests
  - Coverage reports with `mage coverage`
- **E2E Tests**: Playwright
  - Floating chat functionality
  - Context awareness
  - Query generation
  - Navigation flows
  - Cross-browser testing
- **CI/CD**: GitHub Actions
  - Runs on every PR and push
  - Linting, type checking, tests, builds
  - Plugin validation
  - Automatic releases on version tags
- Files: `src/**/*.test.ts`, `src/**/*.test.tsx`, `pkg/**/*_test.go`, `tests/`

### Documentation

- **README.md**: Comprehensive setup and usage guide
- **Architecture docs**: Frontend and backend architecture overview
- **Development guide**: Local setup, testing, contributing
- **API documentation**: Backend endpoints and responses
- **Screenshots**: Visual guides for all major features

### Dependencies

- **Grafana**: 10.4.0+ compatibility
- **grafana-llm-app**: Required for LLM integration
- **Node.js**: 22+
- **Go**: 1.21+

---

## Version History Summary

| Version | Release Date | Theme | Major Features | Code Changes |
|---------|-------------|-------|----------------|--------------|
| 0.0.2 | 2025-12-27 | Security & Governance | Query validation, OTel enforcement, datasource governance, conversation history, AI dev tools, usage logging | +3,000 lines |
| 0.0.1 | 2025-12-24 | Foundation | Context-aware chat, floating UI, query generation, skills system, LLM integration, testing infrastructure | Initial release (~8,000 lines) |

---

## Upgrade Guides

### Upgrading to 0.0.2 from 0.0.1

**Breaking Changes**: None. All new features are disabled by default.

**New Configuration Options**:
All security features are opt-in. Add these to your plugin configuration:

```json
{
  "queryValidation": {
    "enabled": false,
    "enablePromqlValidation": false,
    "enableLogqlValidation": false,
    "enableTraceqlValidation": false,
    "strictMode": false,
    "maxQueryComplexity": 50,
    "allowedFunctions": [],
    "logValidationAttempts": true
  },
  "otelEnforcement": {
    "enabled": false,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "unknown",
    "defaultEnvironmentName": "unknown",
    "rejectIfNoScope": false,
    "validationOnlyMode": false
  },
  "allowedDatasources": [],
  "defaultDatasource": ""
}
```

**Migration Steps**:

1. **Update Plugin**:
   ```bash
   grafana-cli plugins upgrade jorgeancal-zagalin-app
   # or restart Grafana if using Docker/K8s
   ```

2. **Review Security Features**:
   - Read query validation documentation in `.claude/CLAUDE.md`
   - Understand OTel enforcement impact on existing queries
   - Identify approved datasources for allowlist

3. **Gradual Rollout** (recommended):
   - **Week 1**: Enable validation in validation-only mode
     ```json
     {
       "queryValidation": {"enabled": true, "strictMode": false, "logValidationAttempts": true},
       "otelEnforcement": {"enabled": true, "validationOnlyMode": true}
     }
     ```
   - **Week 2**: Review logs, adjust allowlists and defaults
   - **Week 3**: Enable strict mode
     ```json
     {
       "queryValidation": {"strictMode": true},
       "otelEnforcement": {"rejectIfNoScope": true, "validationOnlyMode": false}
     }
     ```

4. **Configure Datasource Governance** (optional but recommended):
   ```json
   {
     "allowedDatasources": [
       "prometheus-prod-uid",
       "loki-prod-uid",
       "tempo-prod-uid"
     ],
     "defaultDatasource": "prometheus-prod-uid"
   }
   ```

5. **Verify**:
   - Test query generation with validation enabled
   - Check audit logs: `grep "LLM chat request" /var/log/grafana/`
   - Verify OTel labels in generated queries

**Recommended Production Settings**:

For production environments with compliance requirements:

```json
{
  "llmBackend": "grafana-llm-app",
  "maxRequestsPerMinute": 60,
  "queryValidation": {
    "enabled": true,
    "enablePromqlValidation": true,
    "enableLogqlValidation": true,
    "enableTraceqlValidation": true,
    "strictMode": true,
    "maxQueryComplexity": 50,
    "allowedFunctions": ["rate", "sum", "avg", "max", "min", "count", "histogram_quantile"],
    "logValidationAttempts": true
  },
  "otelEnforcement": {
    "enabled": true,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "your-service-name",
    "defaultEnvironmentName": "production",
    "rejectIfNoScope": true,
    "validationOnlyMode": false
  },
  "allowedDatasources": ["<approved-datasource-uids>"],
  "defaultDatasource": "<default-datasource-uid>"
}
```

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on contributing to Zagalin.

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.

---

## Links

- **GitHub**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app
- **Issues**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues
- **Documentation**: https://github.com/jorgesnotebook/jorgeancal-zagalin-app#readme
- **Grafana Plugin Catalog**: https://grafana.com/grafana/plugins/jorgeancal-zagalin-app

[Unreleased]: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/compare/v0.0.2...HEAD
[0.0.2]: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases/tag/v0.0.1
