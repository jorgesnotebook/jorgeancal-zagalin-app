---
paths: "**/*.{ts,tsx,go}"
---

# Common Tasks - Step-by-Step Guides

Practical, copy-paste guides for common development tasks in this plugin.

##  Task Index

1. [Add a New Page](#1-add-a-new-page)
2. [Add a Backend Endpoint](#2-add-a-backend-endpoint)
3. [Add a Chat Tool/Function](#3-add-a-chat-toolfunction)
4. [Add Query Validation](#4-add-query-validation)
5. [Add a New Configuration Option](#5-add-a-new-configuration-option)
6. [Add User Storage for Preferences](#6-add-user-storage-for-preferences)
7. [Add Rate Limiting to an Endpoint](#7-add-rate-limiting-to-an-endpoint)
8. [Add a New Test](#8-add-a-new-test)

---

## 1. Add a New Page

**Goal**: Create a new page in the plugin (e.g., "Analytics Page")

### Step 1: Register page in plugin.json

**File**: `src/plugin.json`

```json
{
  "includes": [
    {
      "type": "page",
      "name": "Analytics",
      "path": "/a/jorgeancal-zagalin-app/analytics",
      "role": "Viewer",
      "addToNav": true,
      "defaultNav": false
    }
  ]
}
```

### Step 2: Create page component

**File**: `src/pages/AnalyticsPage.tsx`

```typescript
import React from 'react';
import { PluginPage } from '@grafana/runtime';
import { useStyles2, Stack } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { css } from '@emotion/css';

export const AnalyticsPage = () => {
  const styles = useStyles2(getStyles);

  return (
    <PluginPage>
      <Stack direction="column" gap={2}>
        <div className={styles.container}>
          <h1>Analytics</h1>
          <p>Your analytics content here</p>
        </div>
      </Stack>
    </PluginPage>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(2)};
  `,
});
```

### Step 3: Export from routes

**File**: `src/components/App/App.tsx`

```typescript
import { AnalyticsPage } from 'pages/AnalyticsPage';

// In Routes:
<Route path="/analytics" element={<AnalyticsPage />} />
```

### Step 4: Add to constants

**File**: `src/constants.ts`

```typescript
export const ROUTES = {
  // ... existing routes
  Analytics: 'analytics',
};
```

### Step 5: Test the page

```bash
npm run dev
# Visit: http://localhost:3000/a/jorgeancal-zagalin-app/analytics
```

 **Done!** Your page is now accessible.

---

## 2. Add a Backend Endpoint

**Goal**: Create `/api/plugins/jorgeancal-zagalin-app/resources/my-feature`

### Step 1: Add route handler

**File**: `pkg/plugin/resources.go`

```go
func (a *App) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    log.DefaultLogger.Info("CallResource", "path", req.Path)

    switch req.Path {
    // Add your new endpoint
    case "my-feature":
        return a.handleMyFeature(ctx, req, sender)

    // ... existing cases
    }
}
```

### Step 2: Implement handler

**File**: `pkg/plugin/my_feature.go` (new file)

```go
package plugin

import (
    "context"
    "encoding/json"
    "net/http"

    "github.com/grafana/grafana-plugin-sdk-go/backend"
    "github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type MyFeatureRequest struct {
    Input string `json:"input"`
}

type MyFeatureResponse struct {
    Result string `json:"result"`
}

func (a *App) handleMyFeature(
    ctx context.Context,
    req *backend.CallResourceRequest,
    sender backend.CallResourceResponseSender,
) error {
    // 1. Extract user identity
    user, err := extractUserIdentity(ctx)
    if err != nil {
        return sendError(sender, http.StatusUnauthorized, "unauthorized")
    }

    // 2. Parse request
    var featureReq MyFeatureRequest
    if err := json.Unmarshal(req.Body, &featureReq); err != nil {
        return sendError(sender, http.StatusBadRequest, "invalid request")
    }

    // 3. Validate input
    if featureReq.Input == "" {
        return sendError(sender, http.StatusBadRequest, "input required")
    }

    // 4. Process (your logic here)
    result := processFeature(featureReq.Input)

    // 5. Return response
    response := MyFeatureResponse{
        Result: result,
    }

    body, _ := json.Marshal(response)

    return sender.Send(&backend.CallResourceResponse{
        Status: http.StatusOK,
        Body:   body,
        Headers: map[string][]string{
            "Content-Type": {"application/json"},
        },
    })
}

func processFeature(input string) string {
    // Your logic here
    return "processed: " + input
}
```

### Step 3: Add frontend client

**File**: `src/services/myFeatureService.ts` (new file)

```typescript
import { getBackendSrv } from '@grafana/runtime';

interface MyFeatureRequest {
  input: string;
}

interface MyFeatureResponse {
  result: string;
}

export const myFeatureService = {
  async callFeature(input: string): Promise<string> {
    const response = await getBackendSrv().post<MyFeatureResponse>(
      '/api/plugins/jorgeancal-zagalin-app/resources/my-feature',
      { input }
    );
    return response.result;
  },
};
```

### Step 4: Use in component

```typescript
import { myFeatureService } from 'services/myFeatureService';

function MyComponent() {
  const [result, setResult] = useState('');

  const handleClick = async () => {
    const res = await myFeatureService.callFeature('test input');
    setResult(res);
  };

  return <Button onClick={handleClick}>Call Feature</Button>;
}
```

### Step 5: Test

```bash
# Rebuild backend
mage -v buildAll

# Restart Grafana
docker restart jorgeancal-zagalin-app

# Test from frontend
npm run dev
```

 **Done!** Your endpoint is working.

---

## 3. Add a Chat Tool/Function

**Goal**: Add a new function the LLM can call (e.g., "get_dashboard_list")

### Step 1: Define tool in backend

**File**: `pkg/plugin/assistant_tools.go`

```go
func getDashboardListTool() Tool {
    return Tool{
        Type: "function",
        Function: FunctionDefinition{
            Name:        "get_dashboard_list",
            Description: "Get a list of available dashboards in Grafana",
            Parameters: ParametersDefinition{
                Type: "object",
                Properties: map[string]PropertyDefinition{
                    "search": {
                        Type:        "string",
                        Description: "Optional search query to filter dashboards",
                    },
                },
                Required: []string{}, // No required params
            },
        },
    }
}

// Add to getAvailableTools()
func getAvailableTools(ctx context.Context, settings *Settings) []Tool {
    tools := []Tool{
        // ... existing tools
        getDashboardListTool(),
    }
    return tools
}
```

### Step 2: Implement tool execution (frontend)

**File**: `src/services/zagalinTools.ts`

```typescript
export const executeTool = async (
  toolName: string,
  args: any
): Promise<any> => {
  switch (toolName) {
    // ... existing tools

    case 'get_dashboard_list':
      return await getDashboardList(args.search);

    default:
      throw new Error(`Unknown tool: ${toolName}`);
  }
};

async function getDashboardList(search?: string): Promise<any> {
  const query = search ? `?query=${encodeURIComponent(search)}` : '';
  const response = await getBackendSrv().get(
    `/api/search${query}`
  );
  return {
    dashboards: response.map((d: any) => ({
      title: d.title,
      uid: d.uid,
      url: d.url,
    })),
  };
}
```

### Step 3: Test with LLM

```bash
# Rebuild and restart
mage -v buildAll
docker restart jorgeancal-zagalin-app
npm run dev

# In chat, ask:
# "List all dashboards with 'metrics' in the name"
```

 **Done!** LLM can now call your tool.

---

## 4. Add Query Validation

**Goal**: Add validation for a new query type

### Step 1: Add validation method

**File**: `pkg/plugin/query_validation.go`

```go
func (v *QueryValidator) validateMyQueryType(query string) *ValidationResult {
    result := &ValidationResult{
        Valid:      true,
        Query:      query,
        QueryType:  "myquerytype",
        Violations: []Violation{},
    }

    // Check length
    if len(query) > 10000 {
        result.Valid = false
        result.Violations = append(result.Violations, Violation{
            Type:    "length",
            Message: "Query exceeds maximum length",
        })
    }

    // Check for dangerous patterns
    dangerousPatterns := []string{"DROP", "DELETE", "TRUNCATE"}
    for _, pattern := range dangerousPatterns {
        if strings.Contains(strings.ToUpper(query), pattern) {
            result.Valid = false
            result.Violations = append(result.Violations, Violation{
                Type:    "injection",
                Message: fmt.Sprintf("Dangerous pattern detected: %s", pattern),
            })
        }
    }

    // Complexity check
    complexity := strings.Count(query, "AND") + strings.Count(query, "OR")
    if complexity > v.MaxQueryComplexity {
        result.Valid = false
        result.Violations = append(result.Violations, Violation{
            Type:    "complexity",
            Message: "Query complexity exceeds limit",
        })
    }

    return result
}
```

### Step 2: Add to ValidateQuery switch

```go
func (v *QueryValidator) ValidateQuery(query string, queryType string) *ValidationResult {
    switch queryType {
    // ... existing cases
    case "myquerytype":
        return v.validateMyQueryType(query)
    default:
        return v.validateGeneric(query)
    }
}
```

### Step 3: Add tests

**File**: `pkg/plugin/query_validation_test.go`

```go
func TestValidateMyQueryType(t *testing.T) {
    validator := NewQueryValidator(50, []string{}, true)

    tests := []struct {
        name      string
        query     string
        wantValid bool
    }{
        {
            name:      "valid query",
            query:     "SELECT * FROM table WHERE id = 1",
            wantValid: true,
        },
        {
            name:      "dangerous pattern",
            query:     "DROP TABLE users",
            wantValid: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := validator.ValidateQuery(tt.query, "myquerytype")
            if result.Valid != tt.wantValid {
                t.Errorf("got valid=%v, want %v", result.Valid, tt.wantValid)
            }
        })
    }
}
```

### Step 4: Enable in settings

**File**: `src/components/AppConfig/AppConfig.tsx`

```typescript
<Field label="Enable My Query Type Validation">
  <Switch
    value={jsonData.queryValidation?.enableMyQueryTypeValidation || false}
    onChange={(e) =>
      onOptionsChange({
        ...options,
        jsonData: {
          ...jsonData,
          queryValidation: {
            ...jsonData.queryValidation,
            enableMyQueryTypeValidation: e.currentTarget.checked,
          },
        },
      })
    }
  />
</Field>
```

 **Done!** Your validation is active.

---

## 5. Add a New Configuration Option

**Goal**: Add a new plugin setting (e.g., "max retries")

### Step 1: Add to settings struct

**File**: `pkg/plugin/settings.go`

```go
type Settings struct {
    // ... existing fields
    MaxRetries int `json:"maxRetries"`
}

func LoadSettings(config backend.AppInstanceSettings) (*Settings, error) {
    settings := &Settings{
        // ... existing defaults
        MaxRetries: 3, // Default value
    }

    if err := json.Unmarshal(config.JSONData, settings); err != nil {
        return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
    }

    return settings, nil
}
```

### Step 2: Add to TypeScript types

**File**: `src/types/types.ts`

```typescript
export interface AppJsonData {
  // ... existing fields
  maxRetries?: number;
}
```

### Step 3: Add UI in config

**File**: `src/components/AppConfig/AppConfig.tsx`

```typescript
<Field label="Max Retries" description="Maximum number of retry attempts">
  <Input
    type="number"
    value={jsonData.maxRetries || 3}
    onChange={(e) =>
      onOptionsChange({
        ...options,
        jsonData: {
          ...jsonData,
          maxRetries: parseInt(e.currentTarget.value, 10),
        },
      })
    }
  />
</Field>
```

### Step 4: Use the setting

**Backend**:
```go
func (a *App) someHandler(ctx context.Context) error {
    maxRetries := a.settings.MaxRetries
    for i := 0; i < maxRetries; i++ {
        // Retry logic
    }
}
```

**Frontend** (if needed):
```typescript
import { getAppEvents } from '@grafana/runtime';

// Get settings from context
const pluginMeta = getAppEvents().config.apps['jorgeancal-zagalin-app'];
const maxRetries = pluginMeta.jsonData.maxRetries || 3;
```

 **Done!** Setting is configurable.

---

## 6. Add User Storage for Preferences

**Goal**: Store user preferences (e.g., "preferred query type")

### Using Grafana 11.5+ usePluginUserStorage

```typescript
import { usePluginUserStorage } from '@grafana/runtime';
import { useState, useEffect } from 'react';

interface Preferences {
  preferredQueryType: 'promql' | 'logql';
  autoRefresh: boolean;
}

function MyComponent() {
  const storage = usePluginUserStorage<Preferences>();
  const [prefs, setPrefs] = useState<Preferences>({
    preferredQueryType: 'promql',
    autoRefresh: false,
  });

  // Load on mount
  useEffect(() => {
    async function load() {
      const saved = await storage.getItem('preferences');
      if (saved) {
        setPrefs(saved);
      }
    }
    load();
  }, [storage]);

  // Save when changed
  const updatePref = async (key: keyof Preferences, value: any) => {
    const updated = { ...prefs, [key]: value };
    setPrefs(updated);
    await storage.setItem('preferences', updated);
  };

  return (
    <Select
      value={prefs.preferredQueryType}
      options={[
        { label: 'PromQL', value: 'promql' },
        { label: 'LogQL', value: 'logql' },
      ]}
      onChange={(e) => updatePref('preferredQueryType', e.value)}
    />
  );
}
```

 **Done!** Preferences are stored per-user.

---

## 7. Add Rate Limiting to an Endpoint

**Goal**: Limit requests to 10 per minute for a specific endpoint

### Step 1: Create rate limiter

**File**: `pkg/plugin/my_feature_guardrails.go` (new file)

```go
package plugin

import (
    "sync"
    "golang.org/x/time/rate"
)

type MyFeatureRateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func NewMyFeatureRateLimiter() *MyFeatureRateLimiter {
    return &MyFeatureRateLimiter{
        limiters: make(map[string]*rate.Limiter),
    }
}

func (r *MyFeatureRateLimiter) Allow(userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    limiter, exists := r.limiters[userID]
    if !exists {
        // 10 requests per minute
        limiter = rate.NewLimiter(rate.Limit(10.0/60.0), 10)
        r.limiters[userID] = limiter
    }

    return limiter.Allow()
}
```

### Step 2: Add to app

**File**: `pkg/plugin/app.go`

```go
type App struct {
    // ... existing fields
    myFeatureRateLimiter *MyFeatureRateLimiter
}

func NewApp(ctx context.Context, settings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
    return &App{
        // ... existing fields
        myFeatureRateLimiter: NewMyFeatureRateLimiter(),
    }, nil
}
```

### Step 3: Check in handler

**File**: `pkg/plugin/my_feature.go`

```go
func (a *App) handleMyFeature(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
    user, err := extractUserIdentity(ctx)
    if err != nil {
        return sendError(sender, http.StatusUnauthorized, "unauthorized")
    }

    // Check rate limit
    if !a.myFeatureRateLimiter.Allow(user.UserID) {
        log.DefaultLogger.Warn("Rate limit exceeded", "user", user.UserID)
        return sendError(sender, http.StatusTooManyRequests, "rate limit exceeded")
    }

    // ... rest of handler
}
```

 **Done!** Endpoint is rate-limited.

---

## 8. Add a New Test

### Frontend Unit Test

**File**: `src/pages/AnalyticsPage.test.tsx` (new file)

```typescript
import { render, screen } from '@testing-library/react';
import { AnalyticsPage } from './AnalyticsPage';

describe('AnalyticsPage', () => {
  it('should render page title', () => {
    render(<AnalyticsPage />);
    expect(screen.getByText('Analytics')).toBeInTheDocument();
  });

  it('should display content', () => {
    render(<AnalyticsPage />);
    expect(screen.getByText(/analytics content/i)).toBeInTheDocument();
  });
});
```

### Backend Unit Test

**File**: `pkg/plugin/my_feature_test.go` (new file)

```go
package plugin

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestProcessFeature(t *testing.T) {
    result := processFeature("test")
    assert.Equal(t, "processed: test", result)
}

func TestHandleMyFeature(t *testing.T) {
    app := &App{
        myFeatureRateLimiter: NewMyFeatureRateLimiter(),
    }

    // Test with valid input
    // ... (see pkg/plugin/resources_test.go for examples)
}
```

### E2E Test

**File**: `tests/myFeature.spec.ts` (new file)

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('my feature works', async ({ page, gotoAppPage }) => {
  await gotoAppPage('jorgeancal-zagalin-app/analytics');

  await expect(page.getByText('Analytics')).toBeVisible();

  await page.getByRole('button', { name: 'Run Feature' }).click();

  await expect(page.getByText('Success')).toBeVisible();
});
```

### Run tests

```bash
# Frontend unit tests
npm run test

# Backend tests
mage -v coverage

# E2E tests
npm run e2e
```

 **Done!** Tests are passing.

---

##  Quick Reference

| Task | Files to Modify | Key Command |
|------|----------------|-------------|
| New page | `plugin.json`, `src/pages/`, `App.tsx` | `npm run dev` |
| New endpoint | `resources.go`, `pkg/plugin/my_feature.go` | `mage -v buildAll` |
| Chat tool | `assistant_tools.go`, `zagalinTools.ts` | Rebuild + restart |
| Validation | `query_validation.go` | Add + test |
| Config option | `settings.go`, `types.ts`, `AppConfig.tsx` | Save config |
| User storage | Component with `usePluginUserStorage` | Test in UI |
| Rate limiting | New limiter, add to handler | Rebuild backend |
| Tests | `*.test.tsx`, `*_test.go`, `*.spec.ts` | `npm test` |

---

##  Next Steps

**Learn more**:
- Architecture: `.claude/rules/00-getting-started/architecture-tour.md`
- Frontend patterns: `.claude/rules/00-getting-started/frontend-tour.md`
- Backend patterns: `.claude/rules/00-getting-started/backend-tour.md`
- Decision trees: `.claude/DECISION_TREES.md`

**Get help**:
- Check troubleshooting: `.claude/QUICK_START.md`
- Review code examples in existing files
- Ask your team!

---

**Last Updated**: 2026-01-10
**Plugin Version**: 0.0.5
