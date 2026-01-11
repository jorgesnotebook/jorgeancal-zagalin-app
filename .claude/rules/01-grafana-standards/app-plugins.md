---
paths: '{src/**/*.{ts,tsx},pkg/**/*.go}'
---

# App Plugin Development Guide

This document provides comprehensive guidance for developing Grafana app plugins based on official Grafana documentation and best practices.

## What is an App Plugin?

**App plugins** enable comprehensive, integrated solutions with:

- Custom pages and navigation
- Multiple plugin types bundled together (panels, data sources)
- Preconfigured dashboards
- Backend components for API integration
- Role-based access control (RBAC)

**When to use**: You're creating a complete monitoring experience, need to bundle multiple components, or require custom pages beyond standard visualizations.

**Limitation**: One app instance per organization.

## App Plugin Architecture

### Frontend Components

- **Pages**: React components for custom views
- **Navigation**: Links in Grafana's main navigation
- **Configuration**: Settings UI for plugin setup
- **UI Extensions**: Extend existing Grafana UI

### Backend Components (Optional)

- **HTTP Resources**: Custom API endpoints
- **Data Queries**: Query proxying with security
- **Health Checks**: Plugin status monitoring
- **Metrics**: Prometheus-format metrics

**This plugin is a hybrid app plugin** with both frontend (React/TypeScript) and backend (Go).

## Core Capabilities

### 1. Custom Pages

Create custom React pages accessible via Grafana navigation:

```typescript
// src/pages/MyPage.tsx
import React from 'react';
import { PluginPage } from '@grafana/runtime';

export const MyPage = () => {
  return (
    <PluginPage>
      <h1>My Custom Page</h1>
      {/* Your content here */}
    </PluginPage>
  );
};
```

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

### 2. Backend Integration

Add backend components for secure API integration:

**Why add a backend?**

- Secure API calls (hide credentials from frontend)
- Query proxying with validation
- Custom HTTP resources
- Data caching and transformation
- Rate limiting and governance

**Backend Structure**:

```
pkg/
 main.go              # Plugin binary entry
 plugin/
     app.go           # Main app implementing backend.App
     resources.go     # HTTP resource handlers
     [feature].go     # Feature implementations
```

**Backend Capabilities**:

1. **Custom Resources**: HTTP endpoints for flexible integrations
2. **Query Handler**: Process data queries
3. **Health Check**: Report plugin status
4. **Metrics**: Expose Prometheus metrics
5. **Streaming**: Real-time data streaming

### 3. Authentication

**Security Model**: All backend requests include user context from Grafana.

**User Identity Extraction**:

```go
import "github.com/grafana/grafana-plugin-sdk-go/backend"

func extractUserIdentity(ctx context.Context) (*UserIdentity, error) {
    pluginCtx := backend.PluginConfigFromContext(ctx)
    return &UserIdentity{
        UserID:   pluginCtx.User.Login,
        OrgID:    pluginCtx.OrgID,
        Role:     pluginCtx.User.Role,
    }, nil
}
```

**Never**:

- Store user credentials in frontend
- Bypass Grafana's authentication
- Make authenticated API calls from frontend

**Always**:

- Proxy through backend with user context
- Use Grafana's secure storage for secrets
- Forward user identity to downstream services

**This plugin implements**: `pkg/plugin/query_proxy.go::extractUserIdentity()`

### 4. Resource Handlers

Implement custom HTTP endpoints for flexible integrations:

```go
func (a *App) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    switch req.Path {
    case "health":
        return a.handleHealth(ctx, req, sender)
    case "query":
        return a.handleQuery(ctx, req, sender)
    default:
        return sender.Send(&backend.CallResourceResponse{
            Status: http.StatusNotFound,
        })
    }
}
```

**Resource Handler Use Cases**:

- Custom authentication proxies
- Auto-complete data for query editors
- File uploads/downloads
- IoT device communication
- Streaming responses (SSE, chunked transfer)

**This plugin implements**: `pkg/plugin/resources.go` with 20+ resource endpoints

### 5. Service Accounts

Use service accounts for backend-to-Grafana API calls:

**Why use service accounts?**

- Backend operations independent of user sessions
- Background tasks (context refresh, caching)
- API calls without user context
- Token management and rotation

**Implementation**:

```go
// Create service account token via Grafana API
// Store in plugin settings (secureJsonData)
// Use for authenticated API calls

req, _ := http.NewRequest("GET", grafanaURL+"/api/datasources", nil)
req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
```

**Security**:

- Store tokens in `secureJsonData` (encrypted)
- Rotate tokens regularly
- Use least-privilege permissions
- Never expose tokens to frontend

**This plugin uses service accounts**: See `pkg/plugin/context/manager.go` for datasource context extraction

### 6. Role-Based Access Control (RBAC)

Implement fine-grained permissions for plugin features:

**Grafana Roles**:

- **Viewer**: Read-only access
- **Editor**: Can create/edit
- **Admin**: Full access

**Page-Level RBAC** (plugin.json):

```json
{
  "includes": [
    {
      "type": "page",
      "name": "Admin Settings",
      "path": "/a/your-plugin-id/admin",
      "role": "Admin"
    }
  ]
}
```

**Runtime RBAC Check**:

```typescript
import { contextSrv } from '@grafana/runtime';

if (contextSrv.hasRole('Admin')) {
  // Show admin features
}
```

**Backend RBAC**:

```go
user := backend.PluginConfigFromContext(ctx).User

if user.Role != "Admin" {
    return errors.New("insufficient permissions")
}
```

### 7. Include Dashboards

Bundle preconfigured dashboards with your plugin:

**Dashboard Structure**:

```
src/dashboards/
 overview.json
 details.json
```

**Register in plugin.json**:

```json
{
  "includes": [
    {
      "type": "dashboard",
      "name": "Overview",
      "path": "dashboards/overview.json"
    }
  ]
}
```

**Best Practices**:

- Include default dashboards for quick start
- Use plugin's data sources in dashboards
- Provide both overview and detailed dashboards
- Document dashboard usage in README

### 8. Feature Toggles

Implement feature flags for conditional functionality:

**Configuration**:

```typescript
interface PluginSettings {
  enableBetaFeatures?: boolean;
  enableAdvancedMetrics?: boolean;
}
```

**Usage**:

```typescript
const settings = getPluginSettings();

if (settings.enableBetaFeatures) {
  // Show beta features
}
```

**Backend Feature Toggles**:

```go
type Settings struct {
    EnableQueryValidation bool `json:"enableQueryValidation"`
    EnableOTelEnforcement bool `json:"enableOtelEnforcement"`
}
```

**When to use**:

- Beta/experimental features
- Gradual rollouts
- A/B testing
- Per-organization configuration

**This plugin uses feature toggles**: Query validation, OTel enforcement, skill auto-detection

### 9. Navigation

Control how users navigate to your plugin:

**Default Root Page**:

```json
{
  "defaultNavUrl": "/a/your-plugin-id/home"
}
```

**Deep Linking**:

```typescript
import { locationService } from '@grafana/runtime';

// Navigate to specific page
locationService.push('/a/your-plugin-id/details?id=123');

// Go back
locationService.goBack();
```

**Breadcrumbs**:

```typescript
import { PluginPage } from '@grafana/runtime';

<PluginPage
  pageNav={{
    text: 'Details',
    parentItem: { text: 'Home', url: '/a/your-plugin-id/home' },
  }}
>
  {/* Content */}
</PluginPage>;
```

**This plugin implements**: Global floating chat button using portal mounting (persists across navigation)

### 10. Error Handling

Implement proper error handling patterns:

**Frontend Error Boundaries**:

```typescript
import { ErrorBoundaryAlert } from '@grafana/ui';

<ErrorBoundaryAlert>
  <YourComponent />
</ErrorBoundaryAlert>;
```

**Backend Error Responses**:

```go
func handleError(sender backend.CallResourceResponseSender, statusCode int, message string) error {
    return sender.Send(&backend.CallResourceResponse{
        Status: statusCode,
        Body:   []byte(fmt.Sprintf(`{"error": "%s"}`, message)),
        Headers: map[string][]string{
            "Content-Type": {"application/json"},
        },
    })
}
```

**Error Types**:

- **Validation errors**: 400 Bad Request
- **Authentication errors**: 401 Unauthorized
- **Permission errors**: 403 Forbidden
- **Not found**: 404 Not Found
- **Server errors**: 500 Internal Server Error

**Never**:

- Expose stack traces to users
- Log sensitive data in errors
- Return raw database errors

**Always**:

- Sanitize error messages
- Log errors with context
- Provide actionable error messages

## Performance Best Practices

### Code Splitting

Split large apps into smaller chunks:

```typescript
import { lazy, Suspense } from 'react';
import { LoadingPlaceholder } from '@grafana/ui';

const LazyPage = lazy(() => import('./pages/HeavyPage'));

<Suspense fallback={<LoadingPlaceholder text="Loading..." />}>
  <LazyPage />
</Suspense>;
```

### Caching Strategies

**Frontend Caching**:

```typescript
// Use React Query for data caching
import { useQuery } from '@tanstack/react-query';

const { data, isLoading } = useQuery({
  queryKey: ['metrics'],
  queryFn: fetchMetrics,
  staleTime: 5 * 60 * 1000, // 5 minutes
});
```

**Backend Caching**:

```go
// Cache expensive operations
type Cache struct {
    data         interface{}
    lastRefresh  time.Time
    mu           sync.RWMutex
}

func (c *Cache) Get() interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()

    if time.Since(c.lastRefresh) < 5*time.Minute {
        return c.data
    }
    return nil
}
```

**This plugin implements**: Context caching (`pkg/plugin/context/manager.go`) refreshes every N minutes

### Concurrent Query Execution

Enable parallel queries for better performance:

```json
{
  "backend": true,
  "executable": "gpx_your-plugin",
  "queryOptions": {
    "maxConcurrentQueries": 10
  }
}
```

## Security Best Practices

### 1. Input Validation

**Always validate user input on backend**:

```go
func validateInput(input string) error {
    if len(input) > 10000 {
        return errors.New("input too large")
    }

    if strings.Contains(input, "DROP TABLE") {
        return errors.New("invalid input")
    }

    return nil
}
```

**This plugin implements**: Query validation (`pkg/plugin/query_validation.go`)

### 2. Rate Limiting

**Implement per-user rate limiting**:

```go
type RateLimiter struct {
    limits map[string]*rate.Limiter
    mu     sync.RWMutex
}

func (r *RateLimiter) Allow(userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    limiter, exists := r.limits[userID]
    if !exists {
        limiter = rate.NewLimiter(rate.Limit(60), 60) // 60 req/min
        r.limits[userID] = limiter
    }

    return limiter.Allow()
}
```

**This plugin implements**: Token bucket rate limiting (`pkg/plugin/guardrails.go`)

### 3. Secure Storage

**Store secrets in secureJsonData**:

```json
{
  "jsonData": {
    "apiUrl": "https://api.example.com",
    "timeout": 30000
  },
  "secureJsonData": {
    "apiKey": "secret-key-here"
  }
}
```

**Access in backend**:

```go
apiKey := pluginCtx.DecryptedSecureJSONData["apiKey"]
```

### 4. Output Sanitization

**Sanitize HTML output**:

```typescript
import DOMPurify from 'dompurify';

const sanitizedHTML = DOMPurify.sanitize(llmResponse);
```

**This plugin uses**: DOMPurify for LLM output sanitization

## LLM Integration

For integrating Large Language Models into app plugins:

**Via grafana-llm-app**:

- Use backend proxy pattern (secure)
- Never call LLM APIs from frontend
- Construct system prompts on backend
- Implement function calling for tools
- Stream responses via SSE

**See**: `.claude/rules/grafana-llm-integration.md` for detailed LLM integration guide

**This plugin implements**: Complete LLM integration with function calling, streaming, and context injection

## Testing App Plugins

### Unit Tests (Frontend)

```typescript
import { render, screen } from '@testing-library/react';
import { MyPage } from './MyPage';

test('renders page title', () => {
  render(<MyPage />);
  expect(screen.getByText('My Custom Page')).toBeInTheDocument();
});
```

### Unit Tests (Backend)

```go
func TestResourceHandler(t *testing.T) {
    app := &App{}
    sender := &mockSender{}

    req := &backend.CallResourceRequest{
        Path: "health",
    }

    err := app.CallResource(context.Background(), req, sender)
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, sender.response.Status)
}
```

### E2E Tests

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('app plugin loads successfully', async ({ page, gotoAppPage }) => {
  await gotoAppPage('your-plugin-id');
  await expect(page.getByText('Welcome')).toBeVisible();
});
```

## Plugin Composition

Work with nested plugins:

**Bundle other plugins**:

```json
{
  "dependencies": {
    "plugins": [
      {
        "id": "grafana-llm-app",
        "type": "app",
        "version": ">=1.0.0"
      }
    ]
  }
}
```

**Check if plugin is installed**:

```typescript
import { getBackendSrv } from '@grafana/runtime';

const isInstalled = await getBackendSrv().get('/api/plugins/grafana-llm-app');
```

**This plugin requires**: grafana-llm-app for LLM capabilities

## Provisioning

Enable GitOps workflows by supporting provisioning:

**Provisioning File** (`provisioning/plugins/your-plugin.yaml`):

```yaml
apiVersion: 1

apps:
  - type: your-plugin-id
    orgId: 1
    enabled: true
    jsonData:
      apiUrl: https://api.example.com
      timeout: 30000
    secureJsonData:
      apiKey: ${API_KEY}
```

**Test provisioning**:

```bash
# Place file in Grafana provisioning directory
cp provisioning/plugins/your-plugin.yaml /etc/grafana/provisioning/plugins/
grafana-server restart
```

## User Storage

Add per-user storage for preferences using Grafana's built-in user storage API.

### Requirements

**Grafana version**: 11.5+ (with `userStorageAPI` feature flag enabled)
**Fallback**: Automatically uses browser `localStorage` if unavailable
**Security**: Data is NOT encrypted - do not store sensitive information

### React Hook Pattern (Recommended)

**Using `usePluginUserStorage` hook**:

```typescript
import { usePluginUserStorage } from '@grafana/runtime';
import { useState, useEffect } from 'react';

interface UserPreferences {
  defaultQueryType?: 'timeseries' | 'table';
  theme?: 'light' | 'dark';
  autoRefresh?: boolean;
}

function MyComponent() {
  const storage = usePluginUserStorage<UserPreferences>();
  const [preferences, setPreferences] = useState<UserPreferences>({});
  const [loading, setLoading] = useState(true);

  // Load preferences on mount
  useEffect(() => {
    async function loadPreferences() {
      try {
        const saved = await storage.getItem('preferences');
        if (saved) {
          setPreferences(saved);
        }
      } catch (error) {
        console.error('Failed to load preferences:', error);
      } finally {
        setLoading(false);
      }
    }
    loadPreferences();
  }, [storage]);

  // Save preference
  const updatePreference = async (key: keyof UserPreferences, value: any) => {
    const updated = { ...preferences, [key]: value };
    setPreferences(updated);

    try {
      await storage.setItem('preferences', updated);
    } catch (error) {
      console.error('Failed to save preferences:', error);
    }
  };

  if (loading) {
    return <div>Loading preferences...</div>;
  }

  return (
    <div>
      <select
        value={preferences.defaultQueryType || 'timeseries'}
        onChange={(e) => updatePreference('defaultQueryType', e.target.value)}
      >
        <option value="timeseries">Timeseries</option>
        <option value="table">Table</option>
      </select>
    </div>
  );
}
```

### API Methods

**getItem(key)** - Retrieve stored value:

```typescript
const value = await storage.getItem<T>('myKey');
if (value) {
  // Use value
}
```

**setItem(key, value)** - Store value:

```typescript
await storage.setItem('myKey', { foo: 'bar' });
```

### Best Practices

**DO**:

- Use for user preferences (theme, default values, UI state)
- Provide defaults when no stored value exists
- Handle errors gracefully (storage may fail)
- Use TypeScript types for type safety
- Load preferences on component mount
- Save immediately when user changes preference

**DON'T**:

- Store sensitive data (passwords, API keys, tokens)
- Store large amounts of data (>1MB per user)
- Use for shared team data (user storage is per-user)
- Assume storage is always available (provide fallback)
- Store frequently changing data (use component state instead)

### Fallback Strategy

**Automatic fallback to localStorage**:

```typescript
// Grafana handles fallback automatically
// No additional code needed - same API works for both

// If userStorageAPI is unavailable, uses:
localStorage.setItem('grafana.plugin.your-plugin-id.preferences', JSON.stringify(data));
```

### Migration from localStorage

**If you previously used localStorage directly**:

```typescript
// Old approach (localStorage)
localStorage.setItem('myPlugin.preferences', JSON.stringify(prefs));

// New approach (user storage with automatic fallback)
const storage = usePluginUserStorage();
await storage.setItem('preferences', prefs);
```

**Migration helper**:

```typescript
function migrateFromLocalStorage() {
  const storage = usePluginUserStorage();

  useEffect(() => {
    async function migrate() {
      // Check if already migrated
      const hasNewData = await storage.getItem('preferences');
      if (hasNewData) return;

      // Migrate from old localStorage
      const oldData = localStorage.getItem('myPlugin.preferences');
      if (oldData) {
        try {
          const parsed = JSON.parse(oldData);
          await storage.setItem('preferences', parsed);
          localStorage.removeItem('myPlugin.preferences'); // Clean up
        } catch (error) {
          console.error('Migration failed:', error);
        }
      }
    }
    migrate();
  }, [storage]);
}
```

### Custom Backend Storage (Alternative)

**For complex storage needs or Grafana < 11.5**:

**Frontend**:

```typescript
import { getBackendSrv } from '@grafana/runtime';

// Save user preference
await getBackendSrv().post('/api/plugins/your-plugin-id/resources/user-storage', {
  preference: 'value',
});

// Load user preference
const data = await getBackendSrv().get('/api/plugins/your-plugin-id/resources/user-storage');
```

**Backend**:

```go
// Store per-user data in plugin data directory
userDir := filepath.Join(pluginDataPath, "users", userID)
os.MkdirAll(userDir, 0755)
ioutil.WriteFile(filepath.Join(userDir, "preferences.json"), data, 0644)
```

**This plugin implements**: Dual-tier storage (backend + localStorage fallback) for conversations

### Use Cases

**Good use cases**:

- Default query type selection
- UI theme preferences
- Preferred visualization settings
- Recent searches or filters
- Dashboard layout preferences
- Editor panel sizes

**Bad use cases**:

- API keys or tokens (use secure backend storage)
- Large datasets (use backend with caching)
- Shared team data (use backend or Grafana variables)
- Temporary state (use React state)
- Frequently updated data (causes excessive writes)

## Anonymous Usage Reporting

Track plugin usage without PII:

```typescript
import { reportInteraction } from '@grafana/runtime';

reportInteraction('plugin_page_viewed', {
  page: 'home',
  pluginVersion: '1.0.0',
});
```

**Best Practices**:

- Never send PII (emails, IPs, names)
- Use aggregated metrics only
- Make reporting opt-in or clearly documented
- Respect user privacy settings

## Migration and Compatibility

### Backward Compatibility

Use runtime checks for Grafana version compatibility:

```typescript
import { config } from '@grafana/runtime';

const grafanaVersion = config.buildInfo.version;

if (grafanaVersion >= '10.0.0') {
  // Use new API
} else {
  // Use legacy API
}
```

### Automated Updates

Use `create-plugin` update command:

```bash
npx @grafana/create-plugin@latest update
```

This updates:

- Configuration files (webpack, eslint, prettier)
- GitHub workflows
- Dependencies
- TypeScript configuration

## Publishing Checklist

Before publishing your app plugin:

- [ ] Plugin works with minimum Grafana version specified
- [ ] All tests pass (unit + E2E)
- [ ] No console errors or warnings
- [ ] Documentation is complete
- [ ] Screenshots included
- [ ] README has setup instructions
- [ ] Plugin is signed (for distribution)
- [ ] Provisioning tested (if supported)
- [ ] Default dashboards included (if applicable)
- [ ] Security review completed
- [ ] Performance tested with real data
- [ ] Beta testers validated plugin

## Resources

### Official Documentation

- **App Plugins**: https://grafana.com/developers/plugin-tools/how-to-guides/app-plugins/
- **Backend Plugins**: https://grafana.com/developers/plugin-tools/key-concepts/backend-plugins
- **Best Practices**: https://grafana.com/developers/plugin-tools/key-concepts/best-practices

### This Plugin's Implementation

- **Backend**: `pkg/plugin/` (app.go, resources.go, assistant.go, etc.)
- **Frontend**: `src/` (pages, components, services)
- **Configuration**: `src/plugin.json`
- **Documentation**: `docs/` folder

### Community Support

- **Grafana Community Forum**: https://community.grafana.com/
- **GitHub Discussions**: https://github.com/grafana/grafana/discussions
- **Slack**: #plugin-development channel
