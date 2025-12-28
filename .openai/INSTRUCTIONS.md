# ChatGPT Instructions for Zagalin Development

**For developers using ChatGPT**: Copy and paste this into your ChatGPT conversation when starting work on Zagalin.

---

# Project Context

I'm working on **Zagalin**, a Grafana plugin that's an AI assistant for observability. It's a hybrid React/TypeScript frontend + Go backend plugin.

## Tech Stack
- **Frontend**: React 18, TypeScript, Grafana UI components, RxJS for LLM streaming
- **Backend**: Go 1.21+, grafana-plugin-sdk-go, Mage build system
- **Testing**: Jest (unit), Playwright (E2E)
- **LLM**: Integrates with grafana-llm-app (provider-agnostic)

## Quick Commands
```bash
npm run dev              # Frontend watch mode
npm run server           # Start Grafana + plugin
npm run build            # Build frontend
mage -v buildAll         # Build backend (all platforms)
npm run test:ci          # Run tests
./ci-local.sh            # Full local CI pipeline
```

## CRITICAL: KISS Mindset

**Always keep code simple. Don't over-engineer.**

Rules:
- Wait for 3+ similar uses before creating abstractions
- Delete unused code immediately
- No commented code or "just in case" features
- Prefer 3 similar lines over 1 premature abstraction

```typescript
// ❌ BAD: Over-engineered
class QueryBuilderFactory {
  createBuilder(type: QueryType): AbstractQueryBuilder {
    return this.builderRegistry.get(type).instantiate();
  }
}

// ✅ GOOD: Simple and clear
function buildPromQLQuery(expr: string, range: TimeRange): Query {
  return { expr, from: range.from, to: range.to };
}
```

## CRITICAL: Security-First

**All code must be secure by default.**

### Must Follow:
1. ✅ All datasource queries go through backend proxy (`pkg/plugin/query_proxy.go`)
2. ✅ Backend forwards user's auth cookies to Grafana (preserves permissions)
3. ✅ Rate limiting per user (60 req/min default, token bucket algorithm)
4. ✅ Validate ALL input on backend (don't trust frontend)
5. ✅ Sanitize LLM output with DOMPurify before rendering (XSS prevention)
6. ✅ Audit log all queries with user identity

### Never Do:
- ❌ Never bypass Grafana's permission system
- ❌ Never store API keys in frontend or localStorage
- ❌ Never trust frontend validation alone
- ❌ Never hardcode credentials
- ❌ Never leak sensitive info in error messages
- ❌ Never allow direct datasource access from frontend

### Security Pattern:
```typescript
// ❌ BAD: Direct datasource access
datasource.query(query);

// ✅ GOOD: Through backend with user context
backendProxy.query(userContext, query);
```

```go
// ✅ GOOD: Always extract user identity
func (a *App) handleQuery(w http.ResponseWriter, req *http.Request) {
    // Extract user identity from plugin context
    user, err := extractUserIdentity(req)
    if err != nil {
        sendErrorResponse(w, "Unauthorized", err, http.StatusUnauthorized)
        return
    }

    // Apply rate limiting per user
    if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
        sendErrorResponse(w, "Rate limit exceeded",
            fmt.Errorf("too many requests"), http.StatusTooManyRequests)
        return
    }

    // Execute with user's security context
    // ...
}
```

## Architecture Patterns

### 1. Dual Storage System
- **Primary**: Backend Go file storage (`pkg/plugin/storage.go`)
- **Fallback**: Browser localStorage
- **Migration**: Automatic when backend becomes available
- Frontend: `src/services/conversationStorage.ts`

### 2. Context Manager (Backend)
- Caches Prometheus/Loki/Tempo metadata
- Refreshes every N minutes (configurable)
- Reduces LLM token usage
- Files: `pkg/plugin/context/*.go`

### 3. Global Chat Mounting
- Portal pattern for floating chat button
- Mounts once, persists across Grafana navigation
- Only displays on dashboard pages
- Files: `src/globalChatMount.tsx`, `src/components/FloatingChat/`

### 4. LLM Streaming
- RxJS observables for streaming responses
- Backend proxy adds guardrails (rate limiting, validation)
- Supports function calling
- Files: `src/services/queryService.ts`, `pkg/plugin/assistant_prompts.go (backend)`

## Code Style

### TypeScript
```typescript
// ✅ Explicit types for public APIs
export interface QueryRequest {
  datasource: string;
  queries: QueryPayload[];
  timeRange: TimeRange;
}

// ✅ Type inference for locals
const response = await queryService.query(request);

// ❌ Never use 'any'
```

### React
```typescript
// ✅ Functional components with hooks
export const ChatPanel: React.FC<Props> = ({ conversation }) => {
  const [messages, setMessages] = useState<Message[]>([]);

  useEffect(() => {
    loadMessages(conversation.id);
  }, [conversation.id]);

  return <div>{/* ... */}</div>;
};
```

### Go
```go
// ✅ Return errors, don't panic
func executeQuery(ctx context.Context, query Query) (*Result, error) {
    if err := validate(query); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    return result, nil
}

// ✅ Structured logging
backend.Logger.Info("Query executed",
    "user", user.Login,
    "datasource", dsUID,
    "duration", elapsed,
)
```

## File Structure

```
src/
├── components/          # React components
│   ├── FloatingChat/   # Main chat UI
│   ├── AppConfig/      # Settings
│   └── AskPanel/       # Panel plugin
├── services/           # Business logic
│   ├── queryService.ts        # LLM queries
│   ├── contextService.ts      # Grafana context
│   └── conversationStorage.ts # Persistence
├── hooks/              # React hooks
├── pages/              # Routes
└── module.tsx          # Entry point

pkg/
├── main.go             # Backend entry
└── plugin/
    ├── app.go          # Main app
    ├── query_proxy.go  # Query handler (Issue #16 - USER IDENTITY!)
    ├── guardrails.go   # Rate limiting
    ├── storage.go      # User storage
    └── context/        # Context extraction
```

## Common Tasks

### Adding a Frontend Service
```typescript
// src/services/myService.ts
export class MyService {
  static async doSomething(param: string): Promise<Result> {
    // ALWAYS validate input
    if (!param || param.length > 1000) {
      throw new Error('Invalid parameter');
    }

    // Call backend (never call datasources directly)
    return await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/my-endpoint',
      { data: param }
    );
  }
}
```

### Adding a Backend Endpoint
```go
// pkg/plugin/resources.go
func (a *App) registerRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/my-endpoint", a.handleMyEndpoint)
}

func (a *App) handleMyEndpoint(w http.ResponseWriter, req *http.Request) {
    // ALWAYS extract user identity first
    user, err := extractUserIdentity(req)
    if err != nil {
        sendErrorResponse(w, "Unauthorized", err, http.StatusUnauthorized)
        return
    }

    // ALWAYS validate input
    // ALWAYS use user context
    // ...
}
```

### Writing Tests
```typescript
// Frontend test
describe('MyService', () => {
  it('validates input', async () => {
    await expect(MyService.doSomething('')).rejects.toThrow('Invalid parameter');
  });
});
```

```go
// Backend test
func TestMyEndpoint(t *testing.T) {
    ctx := backend.WithPluginContext(context.Background(),
        backend.PluginContext{
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

## Important Notes

1. **Backend changes require full rebuild**: `mage -v buildAll`
2. **All queries MUST go through backend proxy** (see Issue #16)
3. **Always preserve user identity** in backend handlers
4. **Rate limiting is per-user** (token bucket algorithm)
5. **Validate on backend, not frontend**
6. **Sanitize LLM output** before rendering (XSS)

## Documentation

Detailed docs in repo:
- Architecture: `docs/development/architecture.md`
- API Reference: `docs/api/`
- Testing: `docs/testing/overview.md`
- CI/CD: `docs/CI_PIPELINE_SUMMARY.md`

## Recent Major Changes

**Issue #16 (CRITICAL)**: Backend Plugin & Identity Context
- All queries now route through backend with user identity
- User auth forwarded to Grafana (preserves permissions)
- Rate limiting per user
- Audit logging with user context
- Files: `pkg/plugin/query_proxy.go`, `pkg/plugin/resources_test.go`

---

**When helping me code, please:**
1. Keep it simple (KISS)
2. Be security-first (never bypass auth)
3. Follow the patterns above
4. Don't over-engineer
5. Include tests
6. Ask if unsure about security implications
