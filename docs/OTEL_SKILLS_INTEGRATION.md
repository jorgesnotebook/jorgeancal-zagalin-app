# OpenTelemetry + Skills Integration

## Overview

Zagalin's **skill system** is fully integrated with **OpenTelemetry (OTel) scope enforcement** to ensure that all LLM-generated queries respect multi-tenant boundaries and security policies.

When OTel enforcement is enabled, the LLM automatically includes `service.name` and `deployment.environment.name` attributes in all generated queries, preventing cross-service data leakage and ensuring proper scoping.

---

## How It Works

### 1. Configuration

**IMPORTANT**: All OTel integration features are **ONLY active when OTel enforcement is enabled**.

When OTel enforcement is **disabled**, the skill system works normally without any OTel parameters or label injection.

Enable OTel enforcement in the plugin settings:

```json
{
  "otelEnforcement": {
    "enabled": true,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "my-service",
    "defaultEnvironmentName": "production",
    "rejectIfNoScope": true
  }
}
```

**Settings Explained:**
- `enabled` - Master switch for OTel enforcement
- `requireServiceName` - Mandate `service.name` label on all queries
- `requireEnvironmentName` - Mandate `deployment.environment.name` label
- `defaultServiceName` - Fallback if LLM doesn't provide a service name
- `defaultEnvironmentName` - Fallback if LLM doesn't provide an environment
- `rejectIfNoScope` - Reject queries vs apply defaults when scope missing

### 2. LLM Tool Updates

**Dynamic Tool Definitions**: Tool parameters are built dynamically based on OTel settings.

**When OTel enforcement is ENABLED**, the following tools automatically include OTel parameters:

**When OTel enforcement is DISABLED**, these parameters are NOT present, and tools work as before.

**`create_promql_query` Tool:**
```json
{
  "metric": "http_requests_total",
  "filters": { "status": "200" },
  "aggregation": "rate",
  "timeRange": "5m",
  "serviceName": "api-gateway",          //  NEW
  "environmentName": "production"         //  NEW
}
```

**`create_logql_query` Tool:**
```json
{
  "logStream": "{job=\"app\"}",
  "filter": "error",
  "parser": "json",
  "serviceName": "payment-service",       //  NEW
  "environmentName": "staging"            //  NEW
}
```

### 3. Query Generation Pipeline

When the LLM calls a tool to generate a query, Zagalin:

1. **Extracts OTel parameters** from tool arguments (`serviceName`, `environmentName`)
2. **Injects OTel labels** into the query:
   - PromQL: `metric{service_name="api",deployment_environment_name="prod"}`
   - LogQL: `{job="app",service_name="api",deployment_environment_name="prod"}`
3. **Validates** the query (syntax, complexity, OTel scope)
4. **Returns** the properly scoped query to the user

**Example:**

**User:** "Show me error rate for the payment service"

**LLM Tool Call:**
```json
{
  "function": "create_promql_query",
  "arguments": {
    "metric": "http_requests_total",
    "filters": { "status": "500" },
    "aggregation": "rate",
    "timeRange": "5m",
    "serviceName": "payment-service",
    "environmentName": "production"
  }
}
```

**Generated Query:**
```promql
rate(http_requests_total{service_name="payment-service",deployment_environment_name="production",status="500"}[5m])
```

---

## LLM Behavior with OTel Enforcement

### System Prompt Enhancement

When OTel enforcement is enabled, the LLM receives these instructions:

```
## OpenTelemetry Scope Enforcement

**CRITICAL**: This Grafana instance has OpenTelemetry scope enforcement enabled.

**Required Actions for Query Generation**:
-  **MUST include `serviceName`** parameter when calling query generation tools
-  **MUST include `environmentName`** parameter when calling query generation tools
-  **Default service**: my-service (used if not specified)
-  **Default environment**: production (used if not specified)

**How to Extract OTel Values**:
1. Check dashboard title, panel names, and queries for service names
2. Look for `service_name`, `service.name`, or similar labels in existing queries
3. Check `deployment_environment_name` or `environment` labels
4. If user mentions a specific service/environment, use that
5. If unclear, ASK the user which service/environment to query

 **Strict Mode**: Queries without proper OTel scoping will be REJECTED.
```

### LLM Context Extraction

The LLM is trained to extract OTel values from:

**Dashboard Context:**
- Dashboard title: "Production API Gateway - Errors"
  - → Extracts: `serviceName="api-gateway"`, `environmentName="production"`

**Panel Queries:**
```promql
http_requests_total{service_name="payment-service",deployment_environment_name="staging"}
```
- → Extracts: `serviceName="payment-service"`, `environmentName="staging"`

**User Message:**
- "Show me logs for the payment service in production"
  - → Extracts: `serviceName="payment-service"`, `environmentName="production"`

**When Unclear:**
If the LLM cannot determine the service/environment from context, it will:
1. **Ask the user**: "Which service would you like to query? (e.g., api-gateway, payment-service)"
2. **Use defaults** if configured (fallback mode)
3. **Reject** if strict mode enabled and no defaults

---

## Skills Integration

### Affected Skills

OTel enforcement integrates with these skills:

#### 1. `generate_query` Skill

**Before OTel Integration:**
```
User: "Create a query for CPU usage"
LLM: rate(cpu_usage_seconds_total[5m])
```

**After OTel Integration:**
```
User: "Create a query for CPU usage"
LLM: [Extracts service from context or asks user]
Generated Query: rate(cpu_usage_seconds_total{service_name="api-gateway",deployment_environment_name="production"}[5m])
```

#### 2. `troubleshoot` Skill

When troubleshooting, the LLM generates diagnostic queries with proper OTel scoping:

```
User: "Why is my API slow?"
LLM: [Extracts service="api-gateway" from dashboard]
Generated Queries:
- rate(http_request_duration_seconds{service_name="api-gateway",deployment_environment_name="production"}[5m])
- {service_name="api-gateway",deployment_environment_name="production"} |= "error"
```

#### 3. `explain_panel` Skill

When explaining panels, the LLM recognizes and highlights OTel scope:

```
Panel Query: rate(http_requests_total{service_name="payment-service"}[5m])

LLM Explanation:
"This panel shows the HTTP request rate for the **payment-service** (scoped via service.name label)..."
```

#### 4. `analyze_dashboard` Skill

The LLM identifies which services/environments are covered by the dashboard:

```
LLM Analysis:
"This dashboard monitors 3 services: api-gateway, payment-service, and user-service,
all scoped to the **production** environment via OTel labels."
```

---

## Backend Implementation

### Dynamic Tool Building

**File:** `pkg/plugin/assistant_tools.go`

Tools are now built dynamically based on OTel enforcement settings:

```go
// GetTools builds tools conditionally (lines 198-219)
func GetTools(functionCallingEnabled bool, settings *Settings) []Tool {
    if !functionCallingEnabled {
        return nil
    }

    // Build tools dynamically based on settings
    tools := make([]Tool, 0, len(ZAGALIN_TOOLS))

    for _, tool := range ZAGALIN_TOOLS {
        // For query generation tools, conditionally add OTel parameters
        if tool.Function.Name == "create_promql_query" {
            tools = append(tools, buildPromQLTool(settings))
        } else if tool.Function.Name == "create_logql_query" {
            tools = append(tools, buildLogQLTool(settings))
        } else {
            // Other tools remain unchanged
            tools = append(tools, tool)
        }
    }

    return tools
}

// buildPromQLTool conditionally adds OTel parameters (lines 222-270)
func buildPromQLTool(settings *Settings) Tool {
    props := map[string]PropertyDefinition{
        "metric": {...},
        "filters": {...},
        "aggregation": {...},
        "timeRange": {...},
    }

    description := "Generate a PromQL query for Prometheus metrics"

    // Add OTel parameters ONLY if enforcement is enabled
    if settings != nil && settings.OtelEnforcement.Enabled {
        props["serviceName"] = PropertyDefinition{...}
        props["environmentName"] = PropertyDefinition{...}
        description += ". IMPORTANT: Include serviceName and environmentName for proper OTel scoping."
    }

    return Tool{...}
}
```

**Result:**
- OTel enabled → Tools have `serviceName` and `environmentName` parameters
- OTel disabled → Tools have NO OTel parameters (original behavior)

### Tool Definition (Original - No OTel)

When OTel enforcement is **disabled**, tools look like this:

```go
// create_promql_query tool definition (lines 53-82)
{
    Type: "function",
    Function: Function{
        Name: "create_promql_query",
        Description: "Generate a PromQL query for Prometheus metrics. If OTel enforcement is enabled, include serviceName and environmentName for proper scoping.",
        Parameters: ToolParameters{
            Type: "object",
            Properties: map[string]PropertyDefinition{
                "metric": {...},
                "filters": {...},
                "aggregation": {...},
                "timeRange": {...},
                "serviceName": {
                    Type: "string",
                    Description: "OpenTelemetry service.name for multi-tenant scoping",
                },
                "environmentName": {
                    Type: "string",
                    Description: "OpenTelemetry deployment.environment.name",
                },
            },
            Required: []string{"metric"},
        },
    },
}
```

### System Prompt Enhancement

**File:** `pkg/plugin/assistant_prompts.go`

```go
// BuildSystemPrompt adds OTel context when enforcement is enabled (lines 172-196)
func BuildSystemPrompt(skill string, context AssistantContext, settings *Settings) string {
    // ... base prompt construction ...

    // Add OTel enforcement context if enabled
    if settings != nil && settings.OtelEnforcement.Enabled {
        otelContext := buildOtelEnforcementContext(settings.OtelEnforcement)
        basePrompt = fmt.Sprintf("%s\n\n%s", basePrompt, otelContext)
    }

    return fmt.Sprintf("%s\n\n---\n\n%s", basePrompt, taskInstructions)
}

// buildOtelEnforcementContext constructs OTel instructions for the LLM (lines 198-237)
func buildOtelEnforcementContext(otel OtelEnforcementSettings) string {
    // Generates the system prompt section shown above
}
```

### Conditional Query Injection

**File:** `pkg/plugin/assistant.go`

OTel labels are **ONLY injected when enforcement is enabled**:

```go
// extractPromQLFromToolArgs conditionally injects OTel labels (lines 617-664)
func (a *App) extractPromQLFromToolArgs(args map[string]interface{}) string {
    // ... metric and filters extraction ...

    // Add OTel labels first (ONLY if OTel enforcement is enabled)
    if a.settings != nil && a.settings.OtelEnforcement.Enabled {
        if serviceName, ok := args["serviceName"].(string); ok && serviceName != "" {
            filterParts = append(filterParts, fmt.Sprintf("service_name=\"%s\"", serviceName))
        }
        if environmentName, ok := args["environmentName"].(string); ok && environmentName != "" {
            filterParts = append(filterParts, fmt.Sprintf("deployment_environment_name=\"%s\"", environmentName))
        }
    }

    // Build query with labels
    if len(filterParts) > 0 {
        query = fmt.Sprintf("%s{%s}", metric, strings.Join(filterParts, ","))
    }

    // ... aggregation logic ...
}

// extractLogQLFromToolArgs injects OTel labels (lines 663-705)
func extractLogQLFromToolArgs(args map[string]interface{}) string {
    // Similar injection logic for LogQL
}
```

---

## Security Pipeline

OTel enforcement integrates with Zagalin's security pipeline:

**Query Execution Flow:**

1. **User Request** → "Show me errors"
2. **Skill Detection** → `generate_query` skill activated
3. **LLM Tool Call** → `create_promql_query` with OTel parameters
4. **Query Generation** → OTel labels injected
5. **Query Validation** → Syntax, complexity, OTel scope checked
6. **OTel Scope Validation** → `validateOtelScope()` ensures required attributes present
7. **Query Execution** → Properly scoped query sent to datasource
8. **Audit Logging** → User, query hash, OTel scope logged

**Files:**
- `pkg/plugin/query_proxy.go` (lines 386-416) - OTel validation in query proxy
- `pkg/plugin/otel_enforcement.go` - Full OTel enforcement implementation

---

## Configuration Examples

### Example 1: Strict Mode (Production)

**Use Case:** Multi-tenant SaaS, strict service isolation required

```json
{
  "otelEnforcement": {
    "enabled": true,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "",              // No defaults
    "defaultEnvironmentName": "",          // No defaults
    "rejectIfNoScope": true                // Reject queries without scope
  }
}
```

**Behavior:**
-  Queries without `serviceName` → **REJECTED**
-  Queries without `environmentName` → **REJECTED**
-  LLM **must** ask user for service/environment if unclear

### Example 2: Fallback Mode (Staging)

**Use Case:** Development/staging, convenience over strict enforcement

```json
{
  "otelEnforcement": {
    "enabled": true,
    "requireServiceName": true,
    "requireEnvironmentName": true,
    "defaultServiceName": "default-service",
    "defaultEnvironmentName": "staging",
    "rejectIfNoScope": false               // Apply defaults if missing
  }
}
```

**Behavior:**
-  Queries without `serviceName` → Use `"default-service"`
-  Queries without `environmentName` → Use `"staging"`
-  Less strict, but prevents accidental production queries

### Example 3: Disabled (Development)

**Use Case:** Local development, no OTel requirements

```json
{
  "otelEnforcement": {
    "enabled": false
  }
}
```

**Behavior:**
-  OTel parameters **NOT present** in tool definitions
-  LLM **does NOT see** OTel context in system prompt
-  Queries generated **without** OTel labels
-  Simplest setup - works exactly as before OTel integration
-  Zero OTel overhead

---

## Testing OTel + Skills Integration

### Test Scenarios

#### Test 1: Query Generation with OTel

**Setup:**
- Enable OTel enforcement (strict mode)
- Set `requireServiceName: true`, `requireEnvironmentName: true`

**Test:**
1. Open a dashboard
2. Ask: "Create a query for error rate"
3. **Expected:** LLM asks which service to query
4. Reply: "payment-service in production"
5. **Expected:** Query generated with proper OTel labels

**Verification:**
```promql
rate(http_requests_total{service_name="payment-service",deployment_environment_name="production",status="500"}[5m])
```

#### Test 2: Context Extraction

**Setup:**
- Dashboard with panel showing:
  ```promql
  http_requests_total{service_name="api-gateway",deployment_environment_name="production"}
  ```

**Test:**
1. Ask: "Show me CPU usage"
2. **Expected:** LLM extracts `service_name="api-gateway"` from panel context
3. **Expected:** Generated query includes same OTel scope

**Verification:**
```promql
rate(cpu_usage_seconds_total{service_name="api-gateway",deployment_environment_name="production"}[5m])
```

#### Test 3: Fallback to Defaults

**Setup:**
- Enable OTel enforcement (fallback mode)
- Set `defaultServiceName: "default-app"`
- Set `rejectIfNoScope: false`

**Test:**
1. Ask: "Create a query for memory usage"
2. **Expected:** Query generated with default service name

**Verification:**
```promql
memory_usage_bytes{service_name="default-app"}
```

#### Test 4: Strict Rejection

**Setup:**
- Enable OTel enforcement (strict mode)
- Set `rejectIfNoScope: true`
- No defaults configured

**Test:**
1. Ask: "Create a query for disk usage"
2. LLM doesn't provide `serviceName`
3. **Expected:** Query validation fails with error
4. **Expected:** User sees: "Query rejected: missing required OTel attributes: service.name"

---

## Best Practices

### For Administrators

1. **Start with Fallback Mode**
   - Enable OTel enforcement with defaults
   - Monitor which services are queried
   - Gradually tighten to strict mode

2. **Document Your Service Names**
   - Maintain a list of valid service names
   - Share with users so they know what to query
   - Consider adding to dashboard annotations

3. **Use Dashboard Templates**
   - Pre-scope dashboards to specific services
   - Users inherit OTel scope from dashboard context
   - Reduces need for manual service selection

4. **Monitor OTel Scope Usage**
   - Check backend logs for scope fallbacks
   - Identify queries using defaults vs explicit scope
   - Audit cross-service queries

### For Users

1. **Include Service in Questions**
   -  "Show me errors"
   -  "Show me errors for payment-service"

2. **Work from Dashboards**
   - Navigate to service-specific dashboard first
   - Zagalin extracts OTel scope from panels automatically
   - Less manual specification needed

3. **Check Existing Queries**
   - Look at panel queries to understand OTel labels used
   - Use same service/environment names
   - Maintain consistency across dashboard

4. **Ask When Unsure**
   - If unsure which service, ask Zagalin: "What services are on this dashboard?"
   - Zagalin will analyze and list available services

---

## Troubleshooting

### Issue 1: Queries Always Rejected

**Symptom:** All generated queries fail with "missing required OTel attributes"

**Cause:** Strict mode enabled, no defaults configured, LLM not providing OTel parameters

**Fix:**
1. Check LLM logs to see if `serviceName`/`environmentName` are in tool calls
2. Add defaults: `defaultServiceName: "default-app"`
3. Or disable strict mode: `rejectIfNoScope: false`

### Issue 2: Wrong Service Queried

**Symptom:** LLM generates queries for wrong service

**Cause:** Incorrect context extraction or ambiguous dashboard

**Fix:**
1. Be explicit in question: "Show me errors **for api-gateway**"
2. Check dashboard has clear service labels in panels
3. Review backend logs to see what OTel scope was extracted

### Issue 3: OTel Labels Not Injected

**Symptom:** Generated queries don't have `service_name` labels

**Cause:** OTel enforcement disabled or tool validation disabled

**Fix:**
1. Enable OTel enforcement: `otelEnforcement.enabled: true`
2. Enable tool validation: `toolCallValidation: true`
3. Check LLM is calling tools correctly (not generating raw queries)

### Issue 4: LLM Keeps Asking for Service

**Symptom:** Even on service-specific dashboards, LLM asks which service

**Cause:** Dashboard panels don't have OTel labels, context extraction fails

**Fix:**
1. Add OTel labels to dashboard panel queries
2. Or configure defaults so LLM can fall back
3. Or mention service in dashboard title: "Payment Service - Production"

---

## Label Discovery

**Important:** OpenTelemetry labels can use different naming conventions:
- **Prometheus/Loki**: `service_name` (underscore) or `service.name` (dot)
- **Tempo**: `span.service.name`, `resource.service.name`, or `service.name`
- **Custom setups**: `app`, `service`, `env`, `environment`, etc.

Zagalin **automatically discovers** which label names are actually used in your datasources and adapts accordingly.

**See:** [OpenTelemetry Label Discovery Guide](./OTEL_LABEL_DISCOVERY.md) for detailed information on:
- How discovery works
- Supported label variations (underscore vs dot notation)
- Fallback behavior
- Troubleshooting

**Key Features:**
-  Discovers labels from context manager (Prometheus, Loki, Tempo)
-  Tries multiple variations (`service_name`, `service.name`, `service`, `app`, `job`)
-  Caches per datasource for performance
-  Graceful fallback to sensible defaults

---

## Summary

 **Conditional Integration** - OTel features **ONLY active when enforcement is enabled**

 **Zero Overhead When Disabled** - No OTel parameters, prompts, or injection when disabled

 **Automatic Label Discovery** - Works with both underscore and dot notation automatically

 **Dynamic Tool Building** - Tools adapt based on OTel settings

 **Automatic Scope Injection** - LLM extracts service/environment from context or asks user (when enabled)

 **Security by Default** - Strict mode prevents cross-service data leakage (when enabled)

 **Flexible Fallbacks** - Defaults enable smooth UX while maintaining security (when enabled)

 **Full Audit Trail** - All OTel scope decisions logged for compliance (when enabled)

The integration makes Zagalin **production-ready for multi-tenant environments** without sacrificing ease of use or affecting non-OTel deployments.

---

**Last Updated:** 2026-01-03
**Status:**  Implemented, Tested, Conditional on OTel Flag, with Automatic Label Discovery
