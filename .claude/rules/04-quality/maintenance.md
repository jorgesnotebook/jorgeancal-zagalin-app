---
paths: '**/*.{ts,tsx,js,jsx,go,json}'
---

# Plugin Maintenance & Updates

This document covers keeping your Grafana plugin up-to-date, managing backwards compatibility, and implementing automated maintenance workflows.

## Automated Plugin Updates

### Overview

The `@grafana/create-plugin` tool provides an **automated update command** that maintains plugins by:

- Updating configuration files (webpack, eslint, prettier, tsconfig)
- Managing dependency upgrades for major versions
- Refactoring code to align with configuration changes

### Running Updates

**Command**:

```bash
npx @grafana/create-plugin@latest update
```

**Supported package managers**: npm, Yarn, pnpm

**Prerequisites**:

- Clean Git repository (no uncommitted changes)
- All changes committed or stashed
- Working directory is plugin root

**The update command will exit if**:

- Git repository has uncommitted changes
- Not running from plugin root directory

### Update Options

```bash
# Auto-commit after each migration
npx @grafana/create-plugin@latest update --commit

# Bypass safety checks (use cautiously)
npx @grafana/create-plugin@latest update --force
```

**Recommended**: Use `--commit` flag to track individual migrations via Git history.

### How Updates Work

The update process:

1. **Detects** current create-plugin version
2. **Identifies** necessary migrations
3. **Runs** migrations sequentially with progress output
4. **Updates** dependencies and lock files automatically
5. **Reports** affected files and changes

**Output example**:

```
Running migration: Update ESLint config
Description: Migrates from .eslintrc to eslint.config.mjs
Affected files:
  - .eslintrc (removed)
  - eslint.config.mjs (created)

Installing dependencies...
 Dependencies updated
```

## Continuous Automation Strategies

### GitHub Workflow (Recommended)

Enable automated updates via GitHub Actions:

**Workflow** (`.github/workflows/update-plugin.yml`):

```yaml
name: Update Plugin

on:
  schedule:
    - cron: '0 0 * * 1' # Weekly on Mondays
  workflow_dispatch: # Manual trigger

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '22'

      - name: Run plugin update
        run: npx @grafana/create-plugin@latest update --commit

      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          title: 'chore: update plugin configuration'
          body: 'Automated update from @grafana/create-plugin'
          branch: 'chore/update-plugin'
          commit-message: 'chore: update plugin configuration'
```

**Benefits**:

- Runs automatically on schedule
- Creates PRs for review
- Tracks changes via Git history
- Can be manually triggered

### Dependabot Integration

Use Dependabot for ongoing dependency updates:

**Configuration** (`.github/dependabot.yml`):

```yaml
version: 2
updates:
  # Frontend dependencies
  - package-ecosystem: 'npm'
    directory: '/'
    schedule:
      interval: 'weekly'
    open-pull-requests-limit: 10
    versioning-strategy: increase-if-necessary
    groups:
      grafana-packages:
        patterns:
          - '@grafana/*'
      dev-dependencies:
        dependency-type: 'development'

  # Go dependencies (backend)
  - package-ecosystem: 'gomod'
    directory: '/pkg'
    schedule:
      interval: 'weekly'
```

**Complementary approach**:

- `create-plugin update` → Configuration and tooling
- Dependabot → Dependency versions between migrations

### Renovate Bot (Alternative)

**Configuration** (`renovate.json`):

```json
{
  "extends": ["config:base"],
  "packageRules": [
    {
      "matchPackagePatterns": ["^@grafana/"],
      "groupName": "Grafana packages"
    }
  ],
  "schedule": ["before 3am on Monday"]
}
```

## Backwards Compatibility Management

### Why Runtime Checks Are Necessary

**Key constraint**: "NPM dependencies are shared between plugins and the Grafana application at runtime."

**Problem**: Plugins must support multiple Grafana versions, but APIs change over time.

**Solution**: Conditional logic based on feature/function availability.

**Failure to implement runtime checks**: Plugin crashes and poor user experience.

### Strategy 1: Function Availability Checks

**Pattern**: Test if functions exist before using them.

**Example** (createDataFrame introduced in Grafana 10.1.0):

```typescript
import { createDataFrame, MutableDataFrame } from '@grafana/data';

function createFrame(data: any[]) {
  // Use new API if available, fall back to legacy
  if (typeof createDataFrame === 'function') {
    return createDataFrame({ fields: data });
  } else {
    // Legacy fallback for Grafana < 10.1.0
    return new MutableDataFrame({ fields: data });
  }
}
```

**When to use**:

- New functions introduced in specific Grafana versions
- Optional features that may not exist
- Experimental APIs

### Strategy 2: React Hook Conditionals

**Pattern**: Dynamically select API version based on availability.

**Example** (usePluginLinks introduced in Grafana 11.1.0):

```typescript
import { getPluginLinkExtensions, usePluginLinks as usePluginLinksOriginal } from '@grafana/runtime';

// Select implementation based on availability
const usePluginLinks = usePluginLinksOriginal !== undefined ? usePluginLinksOriginal : useLegacyLinkExtensions;

function MyComponent() {
  const links = usePluginLinks();
  // Use links regardless of Grafana version
}

// Legacy implementation for older versions
function useLegacyLinkExtensions() {
  return getPluginLinkExtensions({
    /* ... */
  });
}
```

**When to use**:

- Hook APIs replaced by newer implementations
- Major API refactoring between versions
- Deprecated but still functional APIs

### Strategy 3: Component Rendering Guards

**Pattern**: Conditionally render components based on existence.

**Example** (UserIcon introduced in Grafana 10.1.0):

```typescript
import { UserIcon } from '@grafana/ui';

function UserProfile() {
  return (
    <div>
      <h1>User Profile</h1>
      {/* Only render if component exists */}
      {UserIcon && <UserIcon />}

      {/* Alternative with fallback */}
      {UserIcon ? (
        <UserIcon />
      ) : (
        <span></span> // Fallback for older versions
      )}
    </div>
  );
}
```

**When to use**:

- New UI components added to @grafana/ui
- Optional visual enhancements
- Non-critical UI features

### Strategy 4: Version Detection

**Pattern**: Check Grafana version explicitly.

**Example**:

```typescript
import { config } from '@grafana/runtime';

function getGrafanaVersion(): string {
  return config.buildInfo.version;
}

function isVersionAtLeast(minVersion: string): boolean {
  const current = getGrafanaVersion();
  // Use semver library for robust comparison
  return semver.gte(current, minVersion);
}

// Usage
if (isVersionAtLeast('10.1.0')) {
  // Use new API
} else {
  // Use legacy API
}
```

**When to use**:

- Multiple APIs changed in same version
- Version-specific workarounds
- Complex compatibility logic

### Strategy 5: E2E Test Coverage

**Pattern**: Test against multiple Grafana versions in CI.

**GitHub Actions Matrix**:

```yaml
jobs:
  e2e:
    strategy:
      matrix:
        grafana-version: ['10.0.0', '10.4.0', '11.0.0', '11.1.0']
    steps:
      - name: Start Grafana ${{ matrix.grafana-version }}
        run: |
          docker run -d -p 3000:3000 \
            grafana/grafana:${{ matrix.grafana-version }}

      - name: Run E2E tests
        run: npm run e2e
```

**Test implementation**:

```typescript
import semver from 'semver';
import { test, expect } from '@grafana/plugin-e2e';

test('feature works correctly', async ({ page, grafanaVersion }) => {
  if (semver.gte(grafanaVersion, '10.1.0')) {
    // Test new feature availability
    await expect(page.getByTestId('new-feature')).toBeVisible();
  } else {
    // Verify graceful degradation
    await expect(page.getByTestId('legacy-fallback')).toBeVisible();
  }
});
```

**Benefits**:

- Catches compatibility issues before release
- Validates both new features and fallbacks
- Ensures graceful degradation
- Documents supported Grafana versions

## Best Practices for Backwards Compatibility

### DO:

**Test against multiple versions** - Use CI matrix testing
**Set minimum Grafana version** - Document in plugin.json
**Graceful degradation** - Provide fallbacks for missing features
**Version checks at boundaries** - Check once, use everywhere
**Log compatibility warnings** - Help users understand limitations
**Document version requirements** - Clear in README and docs

### DON'T:

**Assume APIs exist** - Always check availability
**Skip version testing** - Compatibility bugs are common
**Ignore deprecated warnings** - Update code proactively
**Hard-code version checks** - Use semantic versioning library
**Fail silently** - Log when using fallback implementations
**Support ancient versions** - Set reasonable minimum (e.g., Grafana 10.0+)

## Maintenance Workflow

### Regular Maintenance Cadence

**Weekly**:

- Review Dependabot/Renovate PRs
- Merge minor dependency updates
- Monitor CI for compatibility issues

**Monthly**:

- Run `npx @grafana/create-plugin@latest update`
- Review and test configuration changes
- Update major dependencies if available

**Quarterly**:

- Review and update minimum Grafana version
- Remove compatibility code for EOL versions
- Audit and update documentation
- Review and optimize bundle size

**Yearly**:

- Major refactoring if needed
- Evaluate new Grafana APIs
- Update CI matrix for version testing
- Review security best practices

### Update Checklist

Before merging updates:

- [ ] Run `npm run typecheck` - No TypeScript errors
- [ ] Run `npm run lint` - No linting issues
- [ ] Run `npm run test:ci` - All tests pass
- [ ] Run `npm run e2e` - E2E tests pass
- [ ] Run `mage -v coverage` - Backend tests pass
- [ ] Test in development mode
- [ ] Review CHANGELOG for breaking changes
- [ ] Update documentation if needed
- [ ] Test with minimum Grafana version
- [ ] Verify no console warnings

### Breaking Changes Communication

**When updating causes breaking changes**:

1. **Bump major version** (semver: x.0.0)
2. **Update CHANGELOG.md** with clear migration guide
3. **Document in README** - Highlight version requirements
4. **Create migration guide** - Step-by-step instructions
5. **Announce in release notes** - Clear communication
6. **Consider deprecation period** - Give users time to migrate

**Example CHANGELOG entry**:

````markdown
## [2.0.0] - 2026-01-10

### Breaking Changes

- **Minimum Grafana version**: Now requires Grafana 11.0+
- **Removed legacy API**: `getPluginLinkExtensions` replaced by `usePluginLinks`

### Migration Guide

If you're using Grafana < 11.0:

1. Upgrade Grafana to 11.0+, or
2. Continue using plugin v1.x.x (legacy branch)

To migrate custom code:

```typescript
// Old (v1.x.x)
const links = getPluginLinkExtensions({ extensionPointId: 'foo' });

// New (v2.x.x)
const links = usePluginLinks({ extensionPointId: 'foo' });
```
````

````

## Dependency Management

### Frontend Dependencies

**Update strategy**:
- **Patch versions**: Auto-merge (1.0.x)
- **Minor versions**: Review and merge weekly (1.x.0)
- **Major versions**: Review carefully, test thoroughly (x.0.0)

**Critical dependencies** (review immediately):
- `@grafana/*` packages - Must stay compatible
- `react` / `react-dom` - Core framework
- TypeScript - May require code changes
- Webpack - May break build

**Lock Grafana SDK versions**:
```json
{
  "dependencies": {
    "@grafana/data": "~12.3.0",
    "@grafana/ui": "~12.3.0",
    "@grafana/runtime": "~12.3.0"
  }
}
````

Use `~` (tilde) for patch updates only, or `^` (caret) for minor updates.

### Backend Dependencies (Go)

**Update strategy**:

```bash
# Check for updates
go list -m -u all

# Update specific package
go get -u github.com/grafana/grafana-plugin-sdk-go

# Update all (cautiously)
go get -u ./...

# Tidy dependencies
go mod tidy
```

**Critical Go dependencies**:

- `grafana-plugin-sdk-go` - Must stay compatible
- Security updates - Apply immediately

## Version Support Policy

### Recommended Policy

**Support matrix**:

- **Current version**: Full support
- **Previous minor**: Bug fixes only
- **2+ versions old**: No support

**Example**:

- Plugin 2.1.x: Full support
- Plugin 2.0.x: Critical bug fixes
- Plugin 1.x.x: End of life

**Grafana version support**:

- Minimum: Grafana 10.4.0 (this plugin)
- Tested: 10.4.0, 11.0.0, 11.1.0, 12.0.0
- Recommended: Latest stable

### End of Life Process

When dropping support for old Grafana versions:

1. **Announce deprecation** - 3 months notice
2. **Update documentation** - Clear version requirements
3. **Remove compatibility code** - Clean up codebase
4. **Bump major version** - Indicate breaking change
5. **Maintain legacy branch** - For critical fixes

## Security Updates

### Priority Response

Security updates require immediate action:

1. **Assess impact** - CVE severity and affected versions
2. **Update dependency** - Apply security patch
3. **Test thoroughly** - Ensure no regressions
4. **Release immediately** - Don't wait for scheduled release
5. **Notify users** - Security advisory + changelog

**Automated scanning**:

- GitHub Dependabot - Security alerts
- `npm audit` - Scan for vulnerabilities
- Snyk / other tools - Additional scanning

**Commands**:

```bash
# Check for vulnerabilities
npm audit

# Fix automatically (patches/minor updates)
npm audit fix

# Fix with breaking changes (review carefully)
npm audit fix --force
```

## Troubleshooting Updates

### Common Issues

**Issue**: Update command fails with uncommitted changes

```bash
Error: Git working directory is not clean
```

**Solution**: Commit or stash changes first

```bash
git add .
git commit -m "chore: prepare for update"
npx @grafana/create-plugin@latest update
```

**Issue**: Dependency conflicts after update

```bash
Error: ERESOLVE unable to resolve dependency tree
```

**Solution**: Use legacy peer deps or update conflicting packages

```bash
npm install --legacy-peer-deps
# Or update conflicting package
npm install @grafana/ui@latest
```

**Issue**: Build fails after configuration update

```bash
Error: Cannot find module 'webpack'
```

**Solution**: Reinstall dependencies

```bash
rm -rf node_modules package-lock.json
npm install
npm run build
```

**Issue**: Tests fail after API changes

```bash
TypeError: createDataFrame is not a function
```

**Solution**: Implement runtime checks

```typescript
// Add fallback for older versions
const frame = typeof createDataFrame === 'function' ? createDataFrame(data) : new MutableDataFrame(data);
```

## Resources

### Official Documentation

- **Update Guide**: https://grafana.com/developers/plugin-tools/how-to-guides/updating-a-plugin
- **Runtime Checks**: https://grafana.com/developers/plugin-tools/how-to-guides/runtime-checks
- **Migration Guides**: https://grafana.com/developers/plugin-tools/migration-guides/

### Tools

- **create-plugin**: https://www.npmjs.com/package/@grafana/create-plugin
- **Dependabot**: https://docs.github.com/en/code-security/dependabot
- **Renovate**: https://docs.renovatebot.com/

### Community

- **Forum**: https://community.grafana.com/c/plugin-development
- **GitHub**: https://github.com/grafana/grafana-plugin-tools
- **Slack**: #plugin-development channel
