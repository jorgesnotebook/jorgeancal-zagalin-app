---
paths: '**/*.{ts,tsx,go}'
---

# Grafana Plugin Development Standards

This document defines the official Grafana plugin development standards based on Grafana Labs documentation and best practices.

## Plugin Architecture Requirements

### System Requirements

- **Node.js**: >=22 (LTS version)
- **Go**: >=1.21
- **Grafana Version**: Target v10.0+ (minimum: 10.4.0 for this plugin)
- **Build Tools**: Mage (Go), Webpack 5 (Frontend)
- **Operating Systems**: Linux, macOS, Windows 10+ with WSL

### Plugin Types

Grafana supports three primary plugin types:

1. **Panel plugins**: Custom data visualization methods
2. **Data source plugins**: Connections to databases/data sources
3. **App plugins**: Integrated experiences combining multiple components

**This plugin is an App plugin** with both frontend (React) and backend (Go) components.

## Project Structure Standards

### Frontend Structure

```
src/
 components/          # React components (organized by feature)
 pages/              # Page-level components
 services/           # Business logic layer
 hooks/              # Custom React hooks
 types/              # TypeScript type definitions
 module.tsx          # Plugin entry point
```

### Backend Structure

```
pkg/
 main.go             # Binary entry point
 plugin/
     app.go          # Main plugin app
     resources.go    # HTTP route handlers
     [feature].go    # Feature-specific implementations
```

### Configuration Files

```
.config/                # Build and tool configurations
 webpack/            # Webpack configuration
 .prettierrc.js      # Prettier formatting rules
 eslint.config.mjs   # ESLint rules
 tsconfig.json       # TypeScript configuration
```

## Build System Standards

### Frontend Build Commands

- **Development**: `npm run dev` (watch mode with hot reload)
- **Production**: `npm run build` (optimized bundle)
- **Type checking**: `npm run typecheck`
- **Linting**: `npm run lint`
- **Testing**: `npm run test` (watch) or `npm run test:ci` (single run)
- **E2E**: `npm run e2e` (Playwright tests)

### Backend Build Commands

- **Build all targets**: `mage -v buildAll` (Linux, Darwin, Windows)
- **Build single target**: `mage -v build`
- **Run tests**: `mage -v coverage`
- **List targets**: `mage -l`

### Development Workflow

1. Start frontend watcher: `npm run dev`
2. Start Grafana server: `npm run server` (Docker)
3. Make changes (frontend hot-reloads automatically)
4. For backend changes: rebuild with `mage -v buildAll` and restart container

## Backend Plugin Architecture

### Communication Pattern

- Uses **HashiCorp Go Plugin System over gRPC**
- Grafana launches plugins as isolated subprocesses
- Provides stability (crashes don't affect Grafana)
- Enforces security (sandboxed access)

### Core Backend Capabilities

1. **Query data**: Handle dashboard/alerting queries, return data frames
2. **Resources**: Custom HTTP endpoints for flexible integrations
3. **Health checks**: Report plugin status (auto-invoked on data source test)
4. **Metrics**: Prometheus-format metrics with built-in Go runtime metrics
5. **Streaming**: Real-time data source queries

### Resource Handler Patterns

Custom HTTP resources enable:

- Authentication proxies
- Auto-complete functionality
- IoT device communication
- Chunked transfer encoding for large datasets
- Server-Sent Events (SSE) for streaming

**This plugin uses resource handlers extensively** for:

- LLM chat endpoint (`/llm/chat`)
- Conversation storage (`/storage/*`)
- Query proxy with security pipeline (`/query`)
- Context management (`/context/*`)

## Data Frame Standards

Grafana's data frame is the **universal data structure** for all data:

- Columnar format (similar to Apache Arrow)
- Type-safe with schema
- Supports time series, tables, and logs
- Efficient serialization

**Always use data frames** when returning data from backend queries.

## Plugin Metadata (plugin.json)

### Required Fields

```json
{
  "id": "your-org-plugin-name",
  "type": "app",
  "name": "Plugin Display Name",
  "info": {
    "version": "x.y.z",
    "author": { "name": "Author Name" }
  },
  "dependencies": {
    "grafanaDependency": ">=10.0.0"
  }
}
```

### Best Practices

- Use semantic versioning (x.y.z)
- Set minimum Grafana version in dependencies
- Include clear description and screenshots
- Provide links to documentation and support

## Security Standards

### Plugin Signing

- **Development**: Unsigned plugins work in development mode
- **Production**: Plugins must be signed for distribution
- **Tool**: Use `@grafana/sign-plugin` CLI
- **Requirement**: `GRAFANA_ACCESS_POLICY_TOKEN` from Grafana Cloud

### Security Isolation

- Plugins run in isolated subprocesses
- Cannot access Grafana process memory
- Limited to provided interfaces only
- All configuration provided per request (stateless pattern)

### Backend Security Requirements

- All queries use user's security context
- Never store credentials in frontend
- Use Grafana's secure storage for secrets
- Validate all user input on backend
- Sanitize all output before rendering

## Testing Standards

### Frontend Testing

- **Unit tests**: Jest with React Testing Library
- **Component coverage**: Test all public components
- **E2E tests**: Playwright for critical user flows
- **Run before commit**: `npm run test:ci`

### Backend Testing

- **Unit tests**: Go testing package
- **Coverage target**: >80% code coverage
- **Integration tests**: Test HTTP handlers end-to-end
- **Run before commit**: `mage -v coverage`

### E2E Testing Requirements

- Test against multiple Grafana versions (CI matrix)
- Use `@grafana/plugin-e2e` for Grafana-specific helpers
- Mock external dependencies (LLM providers, data sources)
- Run in Docker containers for consistency

## Configuration Management

### Plugin Settings

- Store in `plugin.json` or backend settings API
- Use secure storage for API keys/tokens
- Provide sensible defaults
- Validate on backend (never trust frontend)

### User Preferences

- Store per-user data with access control
- Use backend storage for cross-device sync
- Fallback to localStorage if backend unavailable
- Migrate localStorage data to backend when available

**This plugin implements dual-tier storage** following these patterns.

## Grafana SDK Integration

### Required Frontend Dependencies

```json
{
  "@grafana/data": "^12.3.0",
  "@grafana/runtime": "^12.3.0",
  "@grafana/ui": "^12.3.0",
  "@grafana/schema": "^12.3.0"
}
```

### Required Backend Dependencies

```go
import (
    "github.com/grafana/grafana-plugin-sdk-go/backend"
)
```

### Version Compatibility

- Keep SDK versions aligned with target Grafana version
- Test against minimum supported Grafana version
- Use CI matrix to test multiple versions

## Development Tools

### Recommended Tools

- **@grafana/create-plugin**: Scaffold new plugins
- **@grafana/sign-plugin**: Sign plugins for distribution
- **grafana-toolkit**: Legacy tool (migrate to create-plugin)
- **Mage**: Go build automation
- **Docker**: Local development environment

### IDE Setup

- TypeScript language server
- ESLint integration
- Prettier integration
- Go language server (gopls)

## Distribution Standards

### Plugin Catalog Requirements

- Signed plugin binary
- Complete metadata in plugin.json
- Screenshots and documentation
- LICENSE file (Apache 2.0 recommended)
- README.md with setup instructions

### Versioning Strategy

- Semantic versioning (major.minor.patch)
- Breaking changes → major version bump
- New features → minor version bump
- Bug fixes → patch version bump

## Best Practices Summary

### DO:

Use official Grafana SDK packages
Follow Grafana's project structure
Test against multiple Grafana versions
Sign plugins before distribution
Use data frames for all data
Implement proper error handling
Write comprehensive tests
Use backend for sensitive operations
Follow security best practices
Keep dependencies up to date

### DON'T:

Store credentials in frontend code
Bypass Grafana's security context
Use deprecated @grafana/toolkit
Ship unsigned plugins to production
Skip backend validation
Trust client-side input
Hardcode Grafana versions
Ignore TypeScript errors
Skip E2E tests
Use non-standard project structure

## Resources

- **Official Documentation**: https://grafana.com/developers/plugin-tools/
- **Plugin Examples**: https://github.com/grafana/grafana-plugin-examples
- **SDK Reference**: https://grafana.com/developers/plugin-tools/reference/
- **Community Forum**: https://community.grafana.com/
- **Plugin Signing**: https://grafana.com/developers/plugin-tools/publish-a-plugin/sign-a-plugin
