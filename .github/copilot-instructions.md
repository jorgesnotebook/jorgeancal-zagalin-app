# GitHub Copilot Instructions

This file provides guidance to GitHub Copilot when generating code in this repository.

## Project Context

**Zagalin** is a context-aware AI assistant Grafana plugin. It's a hybrid frontend (React/TypeScript) + backend (Go) plugin that integrates with grafana-llm-app to provide LLM capabilities.

## Technology Stack

- **Frontend**: React 18, TypeScript, Grafana UI components, RxJS
- **Backend**: Go 1.21+, grafana-plugin-sdk-go, Mage build system
- **Testing**: Jest (unit), Playwright (E2E)
- **LLM Integration**: grafana-llm-app (provider-agnostic)

## Development Commands

```bash
# Development
npm run dev              # Frontend watch mode
npm run server           # Start Grafana with plugin
npm run build            # Build frontend
mage -v buildAll         # Build backend (all platforms)

# Testing
npm run test             # Unit tests (watch)
npm run test:ci          # Unit tests (CI)
npm run e2e              # E2E tests
npm run typecheck        # Type checking
npm run lint             # Linting

# Local CI
./ci-local.sh            # Full CI pipeline locally
```

## Code Style & Patterns

### KISS Mindset (Keep It Simple, Stupid)

**Always prefer simple solutions over complex abstractions:**

```typescript
// ❌ Avoid over-engineering
class QueryBuilderFactory {
  createBuilder(type: QueryType): AbstractQueryBuilder {
    return this.builderRegistry.get(type).instantiate();
  }
}

// ✅ Prefer simple functions
function buildPromQLQuery(expr: string, range: TimeRange): Query {
  return { expr, from: range.from, to: range.to };
}
```

**Core Principles:**

- Don't create abstractions until you have 3+ similar uses
- Delete unused code immediately
- No commented code or "just in case" features
- Use existing patterns before inventing new ones
- Three similar lines > one premature abstraction

### TypeScript Guidelines

```typescript
// ✅ Use explicit types for public APIs
export interface QueryRequest {
  datasource: string;
  queries: QueryPayload[];
  timeRange: TimeRange;
}

// ✅ Use type inference for locals
const response = await queryService.query(request);

// ❌ Don't use 'any'
function processData(data: any) {} // Bad

// ✅ Use proper types
function processData(data: QueryResponse) {} // Good
```

### React Patterns

```typescript
// ✅ Use functional components with hooks
export const ChatPanel: React.FC<Props> = ({ conversation }) => {
  const [messages, setMessages] = useState<Message[]>([]);

  useEffect(() => {
    loadMessages(conversation.id);
  }, [conversation.id]);

  return <div>{/* ... */}</div>;
};

// ❌ Don't use class components (except error boundaries)
```

### Go Patterns

```go
// ✅ Return errors, don't panic
func executeQuery(ctx context.Context, query Query) (*Result, error) {
    if err := validate(query); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// ✅ Use context for cancellation
func handleRequest(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()
    result, err := doWork(ctx)
    // ...
}

// ✅ Log structured data
backend.Logger.Info("Query executed",
    "user", user.Login,
    "datasource", dsUID,
    "duration", elapsed,
)
```

## Security-First Development

**All code must be secure by default. Consider security at every step.**

### Critical Security Rules

1. **Authentication & Authorization**

   - ✅ Always use user's security context for backend queries
   - ✅ Forward user auth cookies/headers to Grafana API
   - ❌ Never bypass Grafana's permission system
   - ❌ Never store API keys in frontend or localStorage

2. **Input Validation**

   - ✅ Validate all user input on the backend
   - ✅ Sanitize LLM output before rendering (use DOMPurify)
   - ✅ Limit message sizes (max 50KB)
   - ❌ Don't trust frontend validation alone

3. **Query Execution**

   - ✅ All datasource queries go through backend proxy
   - ✅ Backend forwards user identity to Grafana
   - ✅ Apply rate limiting per user (60 req/min default)
   - Files: `pkg/plugin/query_proxy.go`, `pkg/plugin/guardrails.go`

4. **Data Privacy**
   - ✅ Store conversations per-user with access control
   - ✅ Hash user IDs in logs (PII protection)
   - ❌ Never leak data between users
   - File: `pkg/plugin/storage.go`

### Security Checklist

When generating code that:

- **Accepts user input** → Add validation
- **Accesses data** → Check user permissions
- **Makes external calls** → Use proper auth
- **Stores data** → Implement access control
- **Renders content** → Sanitize for XSS
- **Has errors** → Don't leak sensitive info
- **Uses secrets** → Use Grafana secure storage
- **Logs data** → Don't log PII or credentials

### Security Anti-Patterns

```typescript
// ❌ Never hardcode credentials
const apiKey = 'sk-proj-...';

// ✅ Use secure storage
const apiKey = secureJsonData.apiKey;

// ❌ Don't bypass backend
datasource.query(query);

// ✅ Route through backend
backendProxy.query(userContext, query);
```

## Architecture Patterns

### Dual Storage System

Conversations use fallback pattern:

- **Primary**: Backend Go storage (file-based)
- **Fallback**: Browser localStorage
- **Migration**: Automatic on backend availability

Files: `src/services/conversationStorage.ts`, `pkg/plugin/storage.go`

### Context Manager

Backend caches Prometheus/Loki/Tempo metadata:

- Refreshes every N minutes (configurable)
- Provides `/context/status` and `/context/refresh` endpoints
- Reduces LLM token usage

Files: `pkg/plugin/context/*.go`

### Global Chat Mounting

Floating chat uses portal pattern:

- Mounts once in `globalChatMount.tsx`
- Persists across Grafana navigation
- Only displays on dashboard pages

Files: `src/globalChatMount.tsx`, `src/components/FloatingChat/`

### LLM Streaming

Uses RxJS for streaming responses:

- Frontend calls backend /llm/chat, backend calls grafana-llm-app streaming API
- Backend proxies with guardrails
- Supports function calling

Files: `src/services/queryService.ts`, `pkg/plugin/assistant_prompts.go (backend)`

## File Organization

```
src/
├── components/          # React components
│   ├── FloatingChat/   # Main chat UI
│   ├── AppConfig/      # Settings UI
│   └── AskPanel/       # Panel plugin
├── services/           # Business logic
│   ├── queryService.ts        # LLM queries
│   ├── contextService.ts      # Grafana context
│   └── conversationStorage.ts # Persistence
├── hooks/              # React hooks
├── pages/              # Route pages
└── module.tsx          # Plugin entry

pkg/
├── main.go             # Backend entry
└── plugin/
    ├── app.go          # Main app
    ├── query_proxy.go  # Query handler (Issue #16)
    ├── guardrails.go   # Rate limiting
    ├── storage.go      # User storage
    └── context/        # Context extraction
```

## Common Tasks

### Adding a New Service

```typescript
// src/services/myService.ts
export class MyService {
  static async doSomething(param: string): Promise<Result> {
    // Validate input
    if (!param || param.length > 1000) {
      throw new Error('Invalid parameter');
    }

    // Call backend
    const response = await getBackendSrv().post('/api/plugins/.../resources/my-endpoint', {
      data: param,
    });

    return response;
  }
}
```

### Adding a Backend Endpoint

```go
// pkg/plugin/resources.go
func (a *App) registerRoutes(mux *http.ServeMux) {
    // ... existing routes
    mux.HandleFunc("/my-endpoint", a.handleMyEndpoint)
}

func (a *App) handleMyEndpoint(w http.ResponseWriter, req *http.Request) {
    // Extract user identity
    user, err := extractUserIdentity(req)
    if err != nil {
        sendErrorResponse(w, "Unauthorized", err, http.StatusUnauthorized)
        return
    }

    // Process request with user context
    // ...
}
```

### Adding Tests

```typescript
// Frontend test
describe('MyService', () => {
  it('should validate input', async () => {
    await expect(MyService.doSomething('')).rejects.toThrow('Invalid parameter');
  });
});
```

```go
// Backend test
func TestMyEndpoint(t *testing.T) {
    app, _ := NewApp(context.Background(), backend.AppInstanceSettings{})

    ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
        User: &backend.User{Login: "testuser"},
        OrgID: 1,
    })

    req := httptest.NewRequest(http.MethodPost, "/my-endpoint", nil)
    req = req.WithContext(ctx)
    w := httptest.NewRecorder()

    app.(*App).handleMyEndpoint(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
}
```

## Documentation

All documentation is in `docs/`:

- Architecture: `docs/development/architecture.md`
- API Reference: `docs/api/`
- Testing: `docs/testing/overview.md`
- CI/CD: `docs/CI_PIPELINE_SUMMARY.md`

## Important Notes

1. **Backend changes require full rebuild**: `mage -v buildAll`
2. **Frontend has hot reload**: `npm run dev`
3. **Always validate on backend**: Don't trust frontend validation
4. **User identity must be preserved**: See Issue #16 implementation
5. **Rate limiting is per-user**: Token bucket algorithm
6. **All queries go through backend proxy**: No direct datasource access

## References

- Main documentation: `.claude/CLAUDE.md`
- Grafana Plugin SDK: https://grafana.com/docs/grafana/latest/developers/plugins/
- Security guidelines: OWASP Top 10
- Recent implementations: `.claude/docs/` folder
