# OpenTelemetry Label Discovery

## Problem

OpenTelemetry labels can use different naming conventions across datasources:

- **Prometheus/Loki**: Typically use underscore notation (`service_name`, `deployment_environment_name`)
- **Tempo/TraceQL**: Use dot notation (`span.service.name`, `resource.service.name`, `service.name`)
- **Custom setups**: May use completely different names (`app`, `service`, `env`, `environment`)

**Hardcoding label names doesn't work** - you need to discover what labels actually exist in each datasource.

---

## Solution: Automatic Label Discovery

Zagalin now **automatically discovers** which OTel label names are used by each datasource and adapts accordingly.

### How It Works

1. **First Query to Datasource**

   - When OTel enforcement is enabled and a datasource is queried for the first time
   - Zagalin checks the context manager for available labels
   - Tries to find labels that match OTel conventions

2. **Discovery Process**

   - **Prometheus/Loki**: Checks labels like `service_name`, `service.name`, `service`, `app`, `job`
   - **Tempo**: Checks `span.service.name`, `resource.service.name`, `service.name`, `service`
   - **Environment**: Checks `deployment_environment_name`, `deployment.environment.name`, `environment`, `env`, `namespace`, `cluster`

3. **Caching**

   - Discovered label format is cached per datasource
   - No need to re-discover on subsequent queries
   - Format stored in memory (OTelLabelRegistry)

4. **Fallback**
   - If discovery finds nothing → Use defaults
   - Prometheus/Loki default: `service_name`, `deployment_environment_name`
   - Tempo default: `span.service.name`, `deployment.environment.name`

---

## Supported Label Variations

### Service Name Labels (Tried in Order)

**Prometheus/Loki:**

1. `service_name` - OpenTelemetry standard with underscore
2. `service.name` - OpenTelemetry standard with dot
3. `service` - Short form
4. `app` - Common alternative
5. `job` - Prometheus convention

**Tempo/TraceQL:**

1. `span.service.name` - Most common in Tempo
2. `resource.service.name` - Alternative in Tempo
3. `service.name` - Generic form
4. `service` - Short form

### Environment Labels (Tried in Order)

**Prometheus/Loki:**

1. `deployment_environment_name` - OpenTelemetry standard with underscore
2. `deployment.environment.name` - OpenTelemetry standard with dot
3. `environment` - Short form
4. `env` - Common alternative
5. `namespace` - Kubernetes convention
6. `cluster` - Multi-cluster setups

**Tempo/TraceQL:**

1. `deployment.environment.name` - OpenTelemetry standard
2. `deployment_environment_name` - Underscore variant
3. `environment` - Short form
4. `env` - Common alternative

---

## Example Scenarios

### Scenario 1: Standard OTel Setup

**Prometheus has labels:**

- `service_name`, `deployment_environment_name`, `instance`, `job`

**Discovery Result:**

```
 Service label: service_name
 Environment label: deployment_environment_name
 Discovered: true
```

**Generated Query:**

```promql
rate(http_requests_total{service_name="api-gateway",deployment_environment_name="production"}[5m])
```

---

### Scenario 2: Tempo with Dot Notation

**Tempo has attributes:**

- `span.service.name`, `span.http.method`, `deployment.environment.name`

**Discovery Result:**

```
 Service label: span.service.name
 Environment label: deployment.environment.name
 Discovered: true
```

**Generated Query:**

```traceql
{span.service.name="payment-service" && deployment.environment.name="production"}
```

---

### Scenario 3: Custom Setup with Different Names

**Prometheus has labels:**

- `app`, `env`, `instance`, `pod`

**Discovery Result:**

```
 Service label: app
 Environment label: env
 Discovered: true
```

**Generated Query:**

```promql
rate(http_requests_total{app="api-gateway",env="production"}[5m])
```

---

### Scenario 4: Mixed Notation (Underscore + Dot)

**Loki has labels:**

- `service.name`, `deployment_environment_name`, `namespace`

**Discovery Result:**

```
 Service label: service.name
 Environment label: deployment_environment_name
 Discovered: true
```

**Generated Query:**

```logql
{service.name="payment-service",deployment_environment_name="production"} |= "error"
```

---

## Architecture

### Files

**`pkg/plugin/otel_label_discovery.go` (NEW)**

- `OTelLabelFormat` - Stores discovered label names per datasource
- `OTelLabelRegistry` - Caches discovered formats
- `DiscoverOTelLabels()` - Main discovery function
- `discoverPromQLLabels()` - Prometheus/Loki discovery
- `discoverLogQLLabels()` - Loki discovery
- `discoverTraceQLLabels()` - Tempo/TraceQL discovery
- `BuildOTelLabels()` - Creates label strings based on discovered format

**`pkg/plugin/app.go` (MODIFIED)**

- Added `otelRegistry *OTelLabelRegistry` field
- Initialized in `NewApp()`

**`pkg/plugin/otel_enforcement.go` (MODIFIED)**

- `injectOtelScope()` now discovers label format before injection
- Uses `BuildOtelLabels()` instead of hardcoded names

**`pkg/plugin/assistant.go` (MODIFIED)**

- `extractPromQLFromToolArgs()` uses discovered labels
- `extractLogQLFromToolArgs()` uses discovered labels

---

## Logging

Discovery events are logged for debugging:

```
[INFO] Discovered OTel label format
  datasource: prometheus-uid
  type: prometheus
  serviceLabel: service_name
  environmentLabel: deployment_environment_name
  discovered: true
```

```
[DEBUG] OTel label discovery result
  datasource: tempo-uid
  type: tempo
  serviceLabel: span.service.name
  environmentLabel: deployment.environment.name
  discovered: true
```

---

## Benefits

**Works with any label naming convention** - No hardcoded assumptions

**Automatic adaptation** - Discovers what's actually in your datasources

**Performance** - Discovery happens once per datasource, then cached

**Fallback safe** - Uses sensible defaults if discovery fails

**Multi-datasource support** - Each datasource can use different conventions

**Zero configuration** - Works out of the box

---

## Future Enhancements

### Configuration Override (Planned)

Allow admins to manually specify label names in plugin settings:

```json
{
  "otelEnforcement": {
    "enabled": true,
    "labelOverrides": {
      "prometheus-uid": {
        "serviceNameLabel": "custom_service",
        "environmentNameLabel": "custom_env"
      },
      "tempo-uid": {
        "serviceNameLabel": "resource.service.name",
        "environmentNameLabel": "deployment.environment.name"
      }
    }
  }
}
```

### Active Discovery via Queries (Planned)

Instead of relying on cached context, actively query datasources:

- **Prometheus**: `label_names()` API to get all available labels
- **Loki**: `/loki/api/v1/labels` API
- **Tempo**: `/api/search/tag/.service.name/values` API

### Smart Discovery (Planned)

Use sample queries to detect which labels actually exist:

```promql
# Try: service_name="test"
# If fails, try: service.name="test"
# If fails, try: service="test"
# Cache what works
```

---

## Troubleshooting

### Issue: Wrong Label Names Used

**Symptom:** Queries fail with "unknown label" errors

**Cause:** Discovery found the wrong labels or used defaults

**Fix:**

1. Check what labels actually exist in your datasource
2. Verify context manager is running: `/api/plugins/jorgeancal-zagalin-app/resources/context/status`
3. Check logs for discovery results
4. Restart plugin to clear cache and re-discover

### Issue: Discovery Not Running

**Symptom:** Always uses default label names (service_name, deployment_environment_name)

**Cause:** Context manager not populated with labels

**Fix:**

1. Ensure context manager has refreshed: `/api/plugins/jorgeancal-zagalin-app/resources/context/refresh`
2. Check context contains labels: `/api/plugins/jorgeancal-zagalin-app/resources/context/status`
3. Wait for automatic context refresh (every N minutes)

### Issue: Mixed Results Across Datasources

**Symptom:** Prometheus works but Tempo fails (or vice versa)

**Cause:** Different datasources use different conventions

**Expected Behavior:** This is normal! Discovery adapts per-datasource

**Verify:**

```
# Check logs for each datasource discovery:
[INFO] Discovered OTel label format datasource=prometheus-uid serviceLabel=service_name
[INFO] Discovered OTel label format datasource=tempo-uid serviceLabel=span.service.name
```

---

## Summary

**Automatic discovery** - No hardcoded label names

**Supports both underscore and dot notation** - Prometheus, Loki, Tempo all work

**Tries multiple variations** - Finds what actually exists in your datasources

**Caches per datasource** - Efficient, no re-discovery overhead

**Graceful fallback** - Uses sensible defaults if discovery fails

The discovery system makes Zagalin **work with any OTel setup** without manual configuration!

---

**Last Updated:** 2026-01-03
**Status:** Implemented and Tested
