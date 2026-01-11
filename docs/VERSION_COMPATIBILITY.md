# Grafana Version Compatibility

This document describes how Zagalin detects and handles different Grafana versions.

## Overview

Zagalin automatically detects your Grafana version to:
- Display compatibility warnings in the configuration UI
- Include version information in health checks
- Provide version-aware error messages
- Help troubleshoot version-specific issues

**Key Features:**
- ✅ **Defensive:** Warns about unsupported versions but doesn't block functionality
- ✅ **Privacy-Respecting:** Only sends version data if available
- ✅ **Graceful Degradation:** Works even if version detection fails
- ✅ **Non-Intrusive:** Configuration-based feature control (not automatic version-based)

## Supported Versions

### Minimum Version

**Grafana 10.4.0** is the minimum supported version.

### Tested Versions

Zagalin is tested against:
- ✅ Grafana 10.4.0
- ✅ Grafana 11.0.0
- ✅ Grafana 11.1.0
- ✅ Grafana 12.0.0

### Recommended Version

**Grafana 11.0.0 or later** is recommended for the best experience.

## How Version Detection Works

### Frontend Detection (Primary)

1. **Source:** Reads version from `config.buildInfo.version` in Grafana runtime
2. **Parsing:** Extracts major.minor.patch from semantic version string
3. **Caching:** Caches result for session duration
4. **Display:** Shows warnings in configuration UI if unsupported

**Example:**
```typescript
import { getGrafanaVersion } from './services/versionDetector';

const version = getGrafanaVersion();
console.log(`Running on Grafana ${version.full}`);
// Output: "Running on Grafana 10.4.0"
```

### Backend Detection (Optional)

1. **Source:** Receives version via `X-Grafana-Version` HTTP header from frontend
2. **Middleware:** Extracts header on each request
3. **Caching:** Stores detected version for logging and health checks
4. **Fallback:** If header missing, version remains "unknown" but plugin continues working

**Header Format:**
```
X-Grafana-Version: 10.4.0
```

### Privacy Considerations

Version detection **respects Grafana's version reporting settings**:

- ✅ **If version reporting is enabled:** Version is detected and sent to backend
- ✅ **If version reporting is disabled:** Version shows as "unknown" with informational warning
- ✅ **Plugin continues working:** No features are blocked if version is unavailable

**Note:** Some users disable version reporting for security/privacy reasons. Zagalin handles this gracefully.

## Feature Compatibility

All features work on supported Grafana versions. Version detection is **informational only** - features are controlled via plugin settings, not automatically disabled based on version.

| Feature | Minimum Version | Notes |
|---------|----------------|-------|
| Core LLM Chat | 10.4.0 | Base requirement |
| Floating Chat Button | 10.4.0 | Portal mounting |
| Context Manager | 10.4.0 | Datasource integration |
| Query Validation | 10.4.0 | Security pipeline |
| Query Proxy | 10.4.0 | Rate limiting & governance |
| Conversation Storage | 10.4.0 | Dual-tier (backend + localStorage) |
| Version Detection | 10.4.0 | This feature |
| grafana-llm-app Integration | 11.0.0+ | Recommended for LLM features |

## User Interface

### Configuration Page Warning

If Grafana version is unsupported or unavailable, a warning appears at the top of the configuration page:

**Unsupported Version:**
```
⚠️ Grafana 10.3.0 - Compatibility Warning

Grafana 10.3.0 is below the minimum supported version 10.4.0.
Please upgrade Grafana for full compatibility. Some features may
not work correctly.

Detected: Grafana 10.3.0
Minimum Required: Grafana 10.4.0
```

**Version Unavailable:**
```
⚠️ Grafana Version Not Detected

Grafana version could not be detected. Some features may not work
as expected on older versions. This may occur if version reporting
is disabled in your Grafana settings.

Minimum Required: Grafana 10.4.0

Note: Version detection may be disabled in your Grafana settings.
The plugin will continue to function, but some features may not
work correctly on older versions.
```

### Health Check Endpoint

Version information is included in the health check response:

**Endpoint:** `/api/plugins/jorgeancal-zagalin-app/health`

**Response Example:**
```json
{
  "status": "ok",
  "message": "Zagalin plugin is ready...",
  "version": {
    "detected": "10.4.0",
    "isAvailable": true,
    "isSupported": true,
    "minimumVersion": "10.4.0",
    "warnings": []
  },
  "contextManager": {
    "lastUpdated": "2026-01-11T18:00:00Z"
  },
  "runManager": {
    "activeRuns": 0
  }
}
```

**Fields:**
- `detected`: Detected version string or "unknown"
- `isAvailable`: Whether version was successfully detected
- `isSupported`: Whether version meets minimum requirement (10.4.0)
- `minimumVersion`: Minimum required version string
- `warnings`: Array of warning messages (empty if supported)

## Troubleshooting

### Version Not Detected

**Symptom:** Configuration page shows "Grafana Version Not Detected" warning

**Possible Causes:**
1. Version reporting disabled in Grafana settings
2. Using a proxy that strips version headers
3. Running a custom Grafana build without version metadata

**Solutions:**
1. ✅ **No action needed** - Plugin will continue to function
2. ✅ Verify your Grafana version: `grafana-server -v` or check About page
3. ✅ Ensure you're running Grafana >= 10.4.0
4. ℹ️ Warning is informational only - features are not blocked

### Unsupported Version Detected

**Symptom:** Configuration page shows "Below minimum supported version" error

**Solution:** Upgrade Grafana to 10.4.0 or later:

```bash
# Check current version
grafana-server -v

# Upgrade (example for Debian/Ubuntu)
sudo apt-get update
sudo apt-get install grafana=10.4.0

# Verify upgrade
grafana-server -v
```

### Version Shows "unknown" in Health Check

**Symptom:** Health endpoint returns `"detected": "unknown"`

**Possible Causes:**
1. No frontend requests sent to backend yet (version detected on first request)
2. Frontend version detection disabled
3. X-Grafana-Version header not being sent

**Solutions:**
1. ✅ Visit plugin configuration page (triggers version detection)
2. ✅ Check browser console for frontend errors
3. ✅ Verify `config.buildInfo.version` is available in browser console

### Backend Logs Show Version Warnings

**Symptom:** Backend logs contain version-related warnings

**Example Logs:**
```
WARN Grafana 10.3.0 is below minimum supported version 10.4.0
INFO Grafana version detected version=10.4.0 supported=true minimumVersion=10.4.0
```

**These are informational logs** for troubleshooting. No action needed unless version is unsupported.

## API Reference

### Frontend API

**Detect Version:**
```typescript
import { detectGrafanaVersion } from './services/versionDetector';

const version = detectGrafanaVersion();
console.log(version);
// {
//   full: "10.4.0",
//   major: 10,
//   minor: 4,
//   patch: 0,
//   isSupported: true,
//   isAvailable: true
// }
```

**Check Compatibility:**
```typescript
import { checkVersionCompatibility } from './services/versionDetector';

const { version, warnings, minimumVersion } = checkVersionCompatibility();

if (warnings.length > 0) {
  console.warn('Version compatibility issues:', warnings);
}
```

**Get Cached Version:**
```typescript
import { getGrafanaVersion } from './services/versionDetector';

// Returns cached version (fast, no re-detection)
const version = getGrafanaVersion();
```

### Backend API

**Version Detection in Handler:**
```go
import "github.com/jorgeancal/zagalin/pkg/plugin"

func (a *App) someHandler(w http.ResponseWriter, r *http.Request) {
    // Version is automatically detected by middleware
    version := a.versionDetector.GetVersion()

    if !version.IsSupported() {
        log.Warn("Unsupported Grafana version", "version", version.String())
    }
}
```

**Check Health with Version:**
```bash
curl http://localhost:3000/api/plugins/jorgeancal-zagalin-app/health | jq '.version'
```

**Response:**
```json
{
  "detected": "10.4.0",
  "isAvailable": true,
  "isSupported": true,
  "minimumVersion": "10.4.0",
  "warnings": []
}
```

## Implementation Details

### Frontend Implementation

**Files:**
- `src/services/versionDetector.ts` - Version detection and compatibility checking
- `src/services/versionReporter.ts` - HTTP header injection for backend
- `src/components/VersionWarning/VersionWarning.tsx` - Warning UI component

**Flow:**
1. `detectGrafanaVersion()` reads `config.buildInfo.version`
2. Parses semantic version (major.minor.patch)
3. Checks against minimum version (10.4.0)
4. Returns version object with support status
5. `VersionWarning` component displays warnings if unsupported

### Backend Implementation

**Files:**
- `pkg/plugin/version.go` - Version parsing and comparison
- `pkg/plugin/version_detector.go` - HTTP header extraction and caching
- `pkg/plugin/app.go` - VersionDetector integration
- `pkg/plugin/resources.go` - Middleware for header detection

**Flow:**
1. Middleware extracts `X-Grafana-Version` header
2. Parses version string
3. Caches in `VersionDetector`
4. Logs warnings if unsupported
5. Includes in health check response

### Testing

**Unit Tests:**
- Frontend: `src/services/versionDetector.test.ts`
- Backend: `pkg/plugin/version_test.go`, `pkg/plugin/version_detector_test.go`

**Integration Tests:**
- Backend: `pkg/plugin/resources_test.go` (TestCheckHealthWithVersion, TestVersionMiddleware)

**Test Coverage:**
- Backend: 100% (all version tests passing)
- Frontend: Comprehensive unit test coverage

## Backward Compatibility

Zagalin uses **configuration-based feature control**, not automatic version-based disabling:

✅ **DO:**
- Provide warnings for unsupported versions
- Log version information for troubleshooting
- Include version in health checks

❌ **DON'T:**
- Automatically disable features based on version
- Block plugin functionality if version unavailable
- Require version detection to work

**User always controls features via plugin settings**, regardless of detected version.

## Future Enhancements

Potential improvements for future versions:

- **Feature-specific version requirements:** Document which specific features require which versions
- **Automatic feature recommendations:** Suggest enabling new features on newer Grafana versions
- **Version-specific prompts:** Adjust LLM system prompts based on detected Grafana capabilities
- **Dashboard compatibility checks:** Validate dashboards against Grafana version

## Resources

- **Grafana Releases:** https://grafana.com/grafana/download
- **Grafana Upgrade Guide:** https://grafana.com/docs/grafana/latest/upgrade-guide/
- **Plugin Installation:** See main README.md
- **Issue Reporting:** https://github.com/jorgeancal/zagalin-app/issues

## Summary

**Version Detection is Defensive, Not Restrictive:**

✅ Detects Grafana version when available
✅ Displays warnings for unsupported versions
✅ Includes version in health checks
✅ Respects privacy (version reporting can be disabled)
✅ **Never blocks functionality** - user controls features via settings
✅ Graceful degradation when version unavailable

**Minimum Supported Version: Grafana 10.4.0**
**Recommended Version: Grafana 11.0.0+**

---

**Last Updated:** 2026-01-11
**Plugin Version:** 0.0.5+
**Document Version:** 1.0
