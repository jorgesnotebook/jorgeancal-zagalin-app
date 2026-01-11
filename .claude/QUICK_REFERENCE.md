# Grafana Plugin Development - Quick Reference

This is a quick reference guide for common Grafana plugin development tasks and patterns.

## Quick Start

### Development Commands

```bash
# Install dependencies
npm ci

# Start development (frontend watch)
npm run dev

# Start Grafana with plugin
npm run server

# Run tests
npm run test              # Unit tests (watch)
npm run test:ci           # Unit tests (CI)
npm run e2e              # E2E tests
npm run typecheck        # Type checking
npm run lint             # Linting

# Build
npm run build            # Frontend
mage -v buildAll         # Backend (all platforms)

# Full CI pipeline
./ci-local.sh
```

### Project Structure

```
src/                      # Frontend (React/TypeScript)
 components/          # React components
 pages/              # Page components
 services/           # Business logic
 hooks/              # Custom React hooks
 types/              # TypeScript types

pkg/                     # Backend (Go)
 main.go             # Binary entry
 plugin/             # Plugin implementation

tests/                   # E2E tests (Playwright)
docs/                    # Documentation
.claude/                 # AI assistant configuration
```

## Common Patterns

### 1. Create a New Page

**Register in plugin.json**:

```json
{
  "includes": [
    {
      "type": "page",
      "name": "My Page",
      "path": "/a/your-plugin-id/my-page",
      "role": "Viewer",
      "addToNav": true
    }
  ]
}
```

**Create page component** (`src/pages/MyPage.tsx`):

```typescript
import React from 'react';
import { PluginPage } from '@grafana/runtime';

export const MyPage = () => {
  return (
    <PluginPage>
      <h1>My Custom Page</h1>
    </PluginPage>
  );
};
```

### 2. Add Backend Resource Handler

**Define handler** (`pkg/plugin/resources.go`):

```go
func (a *App) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    switch req.Path {
    case "my-endpoint":
        return a.handleMyEndpoint(ctx, req, sender)
    default:
        return sender.Send(&backend.CallResourceResponse{
            Status: http.StatusNotFound,
        })
    }
}

func (a *App) handleMyEndpoint(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    data := map[string]string{"message": "Hello"}
    body, _ := json.Marshal(data)

    return sender.Send(&backend.CallResourceResponse{
        Status: http.StatusOK,
        Body:   body,
        Headers: map[string][]string{
            "Content-Type": {"application/json"},
        },
    })
}
```

**Call from frontend**:

```typescript
import { getBackendSrv } from '@grafana/runtime';

const response = await getBackendSrv().get('/api/plugins/your-plugin-id/resources/my-endpoint');
```

### 3. Store User Preferences

**Using Grafana user storage** (Grafana 11.5+):

```typescript
import { usePluginUserStorage } from '@grafana/runtime';

function MyComponent() {
  const storage = usePluginUserStorage();

  // Load
  const prefs = await storage.getItem('preferences');

  // Save
  await storage.setItem('preferences', { theme: 'dark' });
}
```

**Using custom backend storage**:

```typescript
// Save
await getBackendSrv().post('/api/plugins/your-plugin-id/resources/user-storage', { preferences: data });

// Load
const data = await getBackendSrv().get('/api/plugins/your-plugin-id/resources/user-storage');
```

### 4. LLM Integration

**Official pattern** (direct frontend):

```typescript
import { llm, isLLMPluginEnabled } from '@grafana/llm';

if (!isLLMPluginEnabled()) {
  return <div>LLM not available</div>;
}

const response = await llm.chatCompletions({
  model: llm.LLM_MODEL.BASE,
  messages: [
    { role: 'system', content: 'You are a helpful assistant.' },
    { role: 'user', content: 'Hello!' },
  ],
});
```

**Backend proxy pattern** (recommended for security):

```typescript
// Frontend calls backend
const response = await getBackendSrv().post('/api/plugins/your-plugin-id/resources/llm/chat', { message: 'Hello!' });

// Backend handles LLM call securely
// See pkg/plugin/assistant.go for implementation
```

### 5. Authentication & User Context

**Extract user from request** (backend):

```go
import "github.com/grafana/grafana-plugin-sdk-go/backend"

func extractUser(ctx context.Context) (*backend.User, error) {
    pluginCtx := backend.PluginConfigFromContext(ctx)
    return pluginCtx.User, nil
}

// Use in handler
user, err := extractUser(ctx)
if err != nil {
    return err
}
log.Printf("User %s (org %d) requested resource", user.Login, user.OrgID)
```

### 6. Rate Limiting

**Token bucket pattern** (`pkg/plugin/guardrails.go`):

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func (r *RateLimiter) Allow(userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    limiter, exists := r.limiters[userID]
    if !exists {
        limiter = rate.NewLimiter(rate.Limit(60), 60) // 60 req/min
        r.limiters[userID] = limiter
    }

    return limiter.Allow()
}
```

### 7. Query Validation

**Validate user input** (`pkg/plugin/query_validation.go`):

```go
func validateQuery(query string, queryType string) error {
    if len(query) > 10000 {
        return errors.New("query too long")
    }

    if queryType == "promql" {
        return validatePromQL(query)
    }

    return nil
}
```

### 8. E2E Testing

**Basic test**:

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('page loads', async ({ page, gotoAppPage }) => {
  await gotoAppPage('your-plugin-id');
  await expect(page.getByText('Welcome')).toBeVisible();
});
```

**With provisioned dashboard**:

```typescript
test('panel renders', async ({ readProvisionedDashboard, gotoDashboardPage }) => {
  const dashboard = await readProvisionedDashboard({ fileName: 'test.json' });
  const dashboardPage = await gotoDashboardPage(dashboard);

  const panel = dashboardPage.getPanelByTitle('My Panel');
  await expect(panel).toBeVisible();
});
```

## Checklists

### Before Committing

- [ ] `npm run typecheck` passes
- [ ] `npm run lint` passes
- [ ] `npm run test:ci` passes
- [ ] No console.log statements
- [ ] No commented code
- [ ] Updated tests for new code
- [ ] Followed KISS principles

### Before Pushing

- [ ] Run `./ci-local.sh` (full CI)
- [ ] All tests pass locally
- [ ] Backend tests pass: `mage -v coverage`
- [ ] E2E tests pass: `npm run e2e`
- [ ] Git commit message follows format
- [ ] No secrets in code

### Before Releasing

- [ ] Version bumped in package.json
- [ ] CHANGELOG.md updated
- [ ] Documentation updated
- [ ] E2E tests pass on all Grafana versions
- [ ] Plugin signed (for distribution)
- [ ] Release notes written
- [ ] Breaking changes documented

## Debugging

### Frontend Debugging

**Console logging**:

```typescript
console.log('Debug:', value);
console.warn('Warning:', message);
console.error('Error:', error);
```

**React DevTools**:

```bash
# Install React DevTools browser extension
# Then inspect components in browser
```

**Network debugging**:

```bash
# Open browser DevTools → Network tab
# Filter by "Fetch/XHR"
# Inspect API calls
```

### Backend Debugging

**Go logging**:

```go
import "github.com/grafana/grafana-plugin-sdk-go/backend/log"

log.DefaultLogger.Info("Debug message", "key", value)
log.DefaultLogger.Warn("Warning", "error", err)
log.DefaultLogger.Error("Error occurred", "error", err)
```

**Check logs**:

```bash
docker logs jorgeancal-zagalin-app
# Or check Grafana logs directory
```

### E2E Debugging

```bash
# Run with UI
npm run e2e -- --headed

# Debug mode (step through)
npm run e2e -- --debug

# View test report
npx playwright show-report

# View traces
npx playwright show-trace trace.zip
```

## Security Patterns

### Input Validation

```go
// Always validate on backend
func validateInput(input string) error {
    if len(input) > 10000 {
        return errors.New("input too large")
    }

    // Check for injection patterns
    if strings.Contains(input, "DROP TABLE") {
        return errors.New("invalid input")
    }

    return nil
}
```

### Output Sanitization

```typescript
import DOMPurify from 'dompurify';

// Sanitize HTML before rendering
const safeHTML = DOMPurify.sanitize(userInput);
```

### Secure Storage

```json
// Store secrets in secureJsonData
{
  "jsonData": {
    "apiUrl": "https://api.example.com"
  },
  "secureJsonData": {
    "apiKey": "secret-key"
  }
}
```

```go
// Access in backend
apiKey := pluginCtx.DecryptedSecureJSONData["apiKey"]
```

## Performance Tips

### Frontend

- Use `React.memo()` for expensive components
- Implement code splitting with `React.lazy()`
- Debounce user input
- Use `useMemo()` and `useCallback()` hooks
- Minimize bundle size

### Backend

- Cache expensive operations
- Use connection pooling
- Implement proper timeouts
- Enable concurrent query execution
- Profile with `pprof`

## Resources

### Official Docs

- [Grafana Plugin Tools](https://grafana.com/developers/plugin-tools/)
- [grafana-llm-app](https://github.com/grafana/grafana-llm-app)
- [Plugin Examples](https://github.com/grafana/grafana-plugin-examples)

### Project Docs

- `docs/FEATURES_OVERVIEW.md` - Feature inventory
- `docs/api/ENDPOINTS.md` - API reference
- `docs/development/architecture.md` - Architecture
- `.claude/` - AI assistant configuration

### Community

- [Forum](https://community.grafana.com/)
- [Slack](https://grafana.slack.com/) - #plugin-development
- [GitHub Discussions](https://github.com/grafana/grafana/discussions)

## Troubleshooting

### Common Issues

**"Plugin not loading"**

- Check plugin is mounted in Docker
- Verify `plugin.json` syntax
- Check Grafana logs for errors
- Ensure backend binary has execute permissions

**"LLM queries failing"**

- Verify grafana-llm-app is installed
- Check LLM app configuration
- Test connection in LLM app settings
- Check backend logs for rate limiting

**"Build failing"**

- Run `npm ci` (clean install)
- Delete `node_modules` and reinstall
- Check Node.js version (>=22)
- Run `npm run typecheck` for TypeScript errors

**"Backend not starting"**

- Check binary exists: `ls dist/gpx_*`
- Verify execute permissions: `chmod +x dist/gpx_*`
- Check `GF_PLUGIN_APP_DATA_PATH` is writable
- View logs: `docker logs jorgeancal-zagalin-app`

**"Tests failing"**

- Ensure Grafana is running for E2E tests
- Check test environment setup
- Review test logs
- Run tests individually to isolate issues

## Tips & Tricks

### Faster Development

```bash
# Terminal 1: Frontend watch
npm run dev

# Terminal 2: Grafana server
npm run server

# Terminal 3: Tests
npm run test
```

### Quick Plugin Restart

```bash
# Rebuild backend only
mage -v build

# Restart Grafana container
docker restart jorgeancal-zagalin-app
```

### Version Control

```bash
# Useful Git aliases
git config alias.co checkout
git config alias.st status
git config alias.cm "commit -m"
git config alias.lg "log --oneline --graph --decorate"
```

### IDE Setup

**VS Code Extensions**:

- ESLint
- Prettier
- Go (for backend)
- Playwright Test Runner
- GitLens

**VS Code Settings** (`.vscode/settings.json`):

```json
{
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.fixAll.eslint": true
  },
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint"
}
```

## Learning Path

1. **Week 1**: Understand Grafana plugin architecture

   - Read `.claude/rules/grafana-plugin-standards.md`
   - Explore existing code
   - Run development environment

2. **Week 2**: Learn React patterns

   - Study `src/components/`
   - Read `.claude/rules/clean-code-principles.md`
   - Build a simple page

3. **Week 3**: Backend development

   - Study `pkg/plugin/`
   - Read `.claude/rules/app-plugin-development.md`
   - Add a resource handler

4. **Week 4**: Testing

   - Write unit tests
   - Create E2E tests
   - Read `.claude/rules/e2e-testing.md`

5. **Ongoing**: Stay updated
   - Run `npx @grafana/create-plugin@latest update`
   - Check Grafana release notes
   - Review plugin examples

---

**Last Updated**: 2026-01-10
**Grafana Version**: 10.4.0+ (tested up to 12.0.0)
**Plugin Version**: 0.0.5
