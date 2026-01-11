---
paths: 'tests/**/*.spec.ts'
---

# End-to-End Testing for Grafana Plugins

This document covers E2E testing for Grafana plugins using `@grafana/plugin-e2e` (Playwright-based).

## Overview

**@grafana/plugin-e2e** is Grafana's official E2E testing framework for plugins:

- Built on top of Playwright
- Provides Grafana-specific fixtures and assertions
- Supports testing across multiple Grafana versions
- Guaranteed to work with Grafana 8.5.0+

**Key advantages**:

- Cross-version compatibility (Grafana 8.5.0+)
- Pre-built fixtures for common scenarios
- Custom models (page objects) for Grafana UI
- Grafana-specific expect matchers
- Playwright's powerful automation

## Requirements

**Minimum versions**:

- `@playwright/test`: >=1.41.2
- `@grafana/plugin-e2e`: Latest version
- Node.js: >=22 (LTS)

**Installation**:

```bash
npm install --save-dev @playwright/test @grafana/plugin-e2e
```

## Configuration

**Playwright Config** (`playwright.config.ts`):

```typescript
import type { PluginOptions } from '@grafana/plugin-e2e';
import { defineConfig, devices } from '@playwright/test';
import { dirname } from 'node:path';

const pluginE2eAuth = `${dirname(require.resolve('@grafana/plugin-e2e'))}/auth`;

export default defineConfig<PluginOptions>({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: 'html',

  use: {
    baseURL: process.env.GRAFANA_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
  },

  projects: [
    // 1. Auth project - login and save cookie
    {
      name: 'auth',
      testDir: pluginE2eAuth,
      testMatch: [/.*\.js/],
    },
    // 2. Test project - authenticated as admin
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/admin.json',
      },
      dependencies: ['auth'],
    },
  ],
});
```

**Key configuration points**:

- **baseURL**: Grafana instance URL (default: http://localhost:3000)
- **Auth project**: Logs in to Grafana and saves session cookie
- **Authenticated state**: Stored in `playwright/.auth/admin.json`
- **Dependencies**: Tests run after auth completes
- **Retries**: 2 retries on CI, 0 locally

## Test Structure

### Basic Test Pattern

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('plugin loads successfully', async ({ page }) => {
  await page.goto('/a/your-plugin-id');
  await expect(page.getByText('Welcome')).toBeVisible();
});
```

### App Plugin Testing

```typescript
test('app navigation works', async ({ page, gotoAppPage }) => {
  // Navigate to plugin page
  const appPage = await gotoAppPage('your-plugin-id');

  // Verify page loaded
  await expect(appPage.locator('h1')).toContainText('Your Plugin');

  // Test navigation
  await page.getByRole('link', { name: 'Settings' }).click();
  await expect(page).toHaveURL(/\/settings$/);
});
```

### Data Source Plugin Testing

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('data source configuration', async ({ page, createDataSourceConfigPage }) => {
  const configPage = await createDataSourceConfigPage({
    type: 'your-datasource-id',
  });

  // Fill configuration
  await configPage.getByLabel('API URL').fill('https://api.example.com');
  await configPage.getByLabel('API Key').fill('test-key');

  // Test connection
  await configPage.saveButton.click();
  await expect(page.getByText('Data source is working')).toBeVisible();
});
```

### Panel Plugin Testing

```typescript
test('panel renders data', async ({ page, gotoDashboardPage, readProvisionedDashboard }) => {
  const dashboard = await readProvisionedDashboard({ fileName: 'test-dashboard.json' });
  const dashboardPage = await gotoDashboardPage(dashboard);

  // Find panel
  const panel = dashboardPage.getPanelByTitle('Your Panel');

  // Verify rendering
  await expect(panel).toBeVisible();
  await expect(panel.getByText('Metric Value')).toBeVisible();
});
```

## Grafana-Specific Fixtures

### Navigation Fixtures

**gotoAppPage** - Navigate to app plugin page:

```typescript
const appPage = await gotoAppPage('your-plugin-id', { queryParams: { tab: 'settings' } });
```

**gotoDashboardPage** - Navigate to dashboard:

```typescript
const dashboardPage = await gotoDashboardPage({ uid: 'dashboard-uid' });
```

**gotoPanel** - Navigate to panel edit mode:

```typescript
await gotoPanel('panel-id');
```

### Configuration Fixtures

**createDataSourceConfigPage** - Create data source:

```typescript
const configPage = await createDataSourceConfigPage({
  type: 'prometheus',
  name: 'Test Prometheus',
});
```

**createDashboard** - Create dashboard:

```typescript
const dashboard = await createDashboard({
  title: 'Test Dashboard',
  uid: 'test-dash',
});
```

### Provisioning Fixtures

**readProvisionedDashboard** - Load provisioned dashboard:

```typescript
const dashboard = await readProvisionedDashboard({
  fileName: 'dashboard.json',
});
```

**readProvisionedDataSource** - Load provisioned data source:

```typescript
const datasource = await readProvisionedDataSource({
  fileName: 'datasource.yaml',
});
```

## Custom Models (Page Objects)

### Dashboard Models

```typescript
// Get dashboard
const dashboardPage = await gotoDashboardPage({ uid: 'test' });

// Get panel
const panel = dashboardPage.getPanelByTitle('CPU Usage');
await expect(panel).toBeVisible();

// Edit panel
await panel.click();
await page.getByRole('button', { name: 'Edit' }).click();

// Get time range
const timeRange = dashboardPage.getTimeRange();
await expect(timeRange).toContainText('Last 6 hours');
```

### Data Source Models

```typescript
const configPage = await createDataSourceConfigPage({ type: 'prometheus' });

// Access common elements
await configPage.getByLabel('URL').fill('http://prometheus:9090');
await configPage.saveButton.click();
await expect(configPage.alert.success()).toBeVisible();
```

## Custom Expect Matchers

**Grafana-specific assertions**:

```typescript
// Check if data source is working
await expect(dataSourceConfigPage).toHaveAlert('success');

// Check panel data
await expect(panel).toHaveData();

// Check loading state
await expect(panel).not.toBeLoading();

// Check query response
await expect(query).toHaveNoErrors();
```

## Testing Patterns

### Pattern 1: Test Isolation

**DO**: Each test should be independent and create its own resources.

```typescript
test('feature A works', async ({ page, createDashboard }) => {
  // Create fresh dashboard for this test
  const dashboard = await createDashboard({ title: 'Test A' });
  // ... test feature A
});

test('feature B works', async ({ page, createDashboard }) => {
  // Create separate dashboard for this test
  const dashboard = await createDashboard({ title: 'Test B' });
  // ... test feature B
});
```

**DON'T**: Share state between tests.

```typescript
//  BAD - Tests depend on shared state
let sharedDashboard;

test.beforeAll(async ({ createDashboard }) => {
  sharedDashboard = await createDashboard({ title: 'Shared' });
});

test('test 1', async () => {
  // Uses shared dashboard - breaks isolation
});
```

### Pattern 2: Wait for Elements

**DO**: Use Playwright's auto-waiting.

```typescript
//  Playwright waits automatically
await expect(page.getByText('Success')).toBeVisible();
await page.getByRole('button', { name: 'Submit' }).click();
```

**DON'T**: Use arbitrary timeouts.

```typescript
//  Flaky - timing dependent
await page.waitForTimeout(3000);
expect(page.getByText('Success')).toBeVisible();
```

### Pattern 3: Provisioning for Complex Setups

**DO**: Use provisioning for complex pre-configured resources.

**Provisioning file** (`provisioning/dashboards/test-dashboard.json`):

```json
{
  "dashboard": {
    "title": "Test Dashboard",
    "panels": [
      {
        "id": 1,
        "type": "your-panel-type",
        "title": "Test Panel"
      }
    ]
  },
  "overwrite": true
}
```

**Test using provisioned resource**:

```typescript
test('panel renders correctly', async ({ readProvisionedDashboard, gotoDashboardPage }) => {
  const dashboard = await readProvisionedDashboard({ fileName: 'test-dashboard.json' });
  const dashboardPage = await gotoDashboardPage(dashboard);

  const panel = dashboardPage.getPanelByTitle('Test Panel');
  await expect(panel).toBeVisible();
});
```

### Pattern 4: Testing with Feature Toggles

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('feature works when toggle enabled', async ({ page, gotoAppPage, featureToggles }) => {
  // Enable feature toggle
  await featureToggles.enable('myFeature');

  await gotoAppPage('your-plugin-id');
  await expect(page.getByText('New Feature')).toBeVisible();
});

test('fallback works when toggle disabled', async ({ page, gotoAppPage, featureToggles }) => {
  await featureToggles.disable('myFeature');

  await gotoAppPage('your-plugin-id');
  await expect(page.getByText('Legacy Feature')).toBeVisible();
});
```

### Pattern 5: Testing Authentication Flows

```typescript
test('user without permission sees error', async ({ page, gotoAppPage, createUser }) => {
  // Create viewer user
  const user = await createUser({ role: 'Viewer' });

  // Login as viewer
  await page.goto('/login');
  await page.getByLabel('Username').fill(user.login);
  await page.getByLabel('Password').fill(user.password);
  await page.getByRole('button', { name: 'Log in' }).click();

  // Access admin page
  await gotoAppPage('your-plugin-id/admin');

  // Verify permission error
  await expect(page.getByText('Insufficient permissions')).toBeVisible();
});
```

## Cross-Version Testing

### CI Matrix Configuration

**GitHub Actions** (`.github/workflows/e2e.yml`):

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    strategy:
      fail-fast: false
      matrix:
        grafana-version: ['10.0.0', '10.4.0', '11.0.0', '11.1.0', '12.0.0']

    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '22'

      - name: Install dependencies
        run: npm ci

      - name: Build plugin
        run: npm run build

      - name: Start Grafana ${{ matrix.grafana-version }}
        run: |
          docker run -d -p 3000:3000 \
            -e "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=your-plugin-id" \
            -e "GF_DEFAULT_APP_MODE=development" \
            -v ${{ github.workspace }}:/var/lib/grafana/plugins/your-plugin-id \
            grafana/grafana:${{ matrix.grafana-version }}

          # Wait for Grafana to start
          npx wait-on http://localhost:3000

      - name: Run E2E tests
        run: npm run e2e

      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: test-results-${{ matrix.grafana-version }}
          path: playwright-report/
```

### Version-Specific Tests

```typescript
import semver from 'semver';

test('feature availability', async ({ page, grafanaVersion, gotoAppPage }) => {
  await gotoAppPage('your-plugin-id');

  if (semver.gte(grafanaVersion, '11.0.0')) {
    // Test new feature (Grafana 11+)
    await expect(page.getByTestId('new-feature')).toBeVisible();
  } else {
    // Verify fallback (Grafana < 11)
    await expect(page.getByTestId('legacy-feature')).toBeVisible();
  }
});
```

## Running Tests

### Local Development

```bash
# Run all tests
npm run e2e

# Run with UI (headed mode)
npm run e2e -- --headed

# Run specific test file
npm run e2e tests/appNavigation.spec.ts

# Debug mode
npm run e2e -- --debug

# Run with specific Grafana version
GRAFANA_URL=http://localhost:3001 npm run e2e
```

### CI/CD

```bash
# CI mode (retries enabled, no UI)
npm run e2e -- --reporter=github

# Generate HTML report
npx playwright show-report
```

## Debugging E2E Tests

### Visual Debugging

**Playwright Inspector**:

```bash
npm run e2e -- --debug
```

**Step through test**:

- Pause execution
- Inspect elements
- View console logs
- Network requests

### Trace Viewer

**Enable traces**:

```typescript
// In playwright.config.ts
use: {
  trace: 'on-first-retry', // Or 'on', 'off', 'retain-on-failure'
}
```

**View traces**:

```bash
npx playwright show-trace trace.zip
```

### Screenshots on Failure

**Automatic screenshots**:

```typescript
test('feature works', async ({ page, screenshot }) => {
  await page.goto('/app');

  try {
    await expect(page.getByText('Success')).toBeVisible();
  } catch (error) {
    await screenshot('failure-state.png');
    throw error;
  }
});
```

### Console Logging

**Capture console messages**:

```typescript
test('logs are correct', async ({ page }) => {
  const logs: string[] = [];

  page.on('console', (msg) => logs.push(msg.text()));

  await page.goto('/app');

  // Verify no errors logged
  expect(logs.filter((log) => log.includes('Error'))).toHaveLength(0);
});
```

## Best Practices

### DO:

**Use test isolation** - Each test creates its own resources
**Use provisioning** - For complex pre-configured setups
**Test across versions** - CI matrix with multiple Grafana versions
**Use semantic locators** - `getByRole`, `getByLabel`, not `querySelector`
**Wait automatically** - Trust Playwright's auto-waiting
**Test critical paths** - Focus on user workflows
**Use page objects** - Grafana's built-in models
**Run on CI** - Catch issues before merge

### DON'T:

**Share state between tests** - Breaks test isolation
**Use arbitrary waits** - Flaky and slow
**Test implementation details** - Test user-visible behavior
**Skip version testing** - Compatibility bugs are common
**Hardcode selectors** - Use semantic locators
**Ignore flaky tests** - Fix or delete them
**Test third-party code** - Test your plugin only

## Migration from @grafana/e2e (Cypress)

### Key Differences

| Feature         | @grafana/e2e (Cypress) | @grafana/plugin-e2e (Playwright) |
| --------------- | ---------------------- | -------------------------------- |
| Framework       | Cypress                | Playwright                       |
| Browser         | Chrome only            | Chrome, Firefox, Safari          |
| Speed           | Slower                 | Faster                           |
| Parallelization | Limited                | Excellent                        |
| API             | Cypress commands       | Playwright API                   |
| Selectors       | `e2e-selectors`        | Semantic locators                |

### Migration Steps

1. **Install Playwright**:

```bash
npm install --save-dev @playwright/test @grafana/plugin-e2e
```

2. **Configure Playwright** (see Configuration section above)

3. **Rewrite tests** (no automatic conversion):

**Before** (Cypress):

```javascript
describe('Plugin', () => {
  beforeEach(() => {
    cy.visit('/a/your-plugin-id');
  });

  it('loads successfully', () => {
    cy.contains('Welcome').should('be.visible');
  });
});
```

**After** (Playwright):

```typescript
import { test, expect } from '@grafana/plugin-e2e';

test('plugin loads successfully', async ({ page, gotoAppPage }) => {
  await gotoAppPage('your-plugin-id');
  await expect(page.getByText('Welcome')).toBeVisible();
});
```

4. **Remove Cypress**:

```bash
rm -rf cypress cypress.config.ts
npm uninstall --save-dev @grafana/e2e @grafana/e2e-selectors cypress
```

5. **Update package.json**:

```json
{
  "scripts": {
    "e2e": "playwright test"
  }
}
```

## Troubleshooting

### Common Issues

**Issue**: Tests fail with "Grafana not ready"

```
Error: page.goto: net::ERR_CONNECTION_REFUSED
```

**Solution**: Ensure Grafana is running and accessible

```bash
# Check Grafana is running
curl http://localhost:3000/api/health

# Wait for Grafana to be ready
npx wait-on http://localhost:3000
```

**Issue**: Authentication fails

```
Error: storageState file not found
```

**Solution**: Ensure auth project runs first

```typescript
// In playwright.config.ts
projects: [
  { name: 'auth', ... },
  {
    name: 'chromium',
    dependencies: ['auth'], // Ensure auth runs first
  }
]
```

**Issue**: Tests are flaky

```
Error: Timeout waiting for element
```

**Solution**: Use proper waits and increase timeout if needed

```typescript
// Increase timeout for slow operations
await expect(page.getByText('Success')).toBeVisible({ timeout: 10000 });

// Or use Playwright's auto-waiting
await page.waitForLoadState('networkidle');
```

**Issue**: Plugin not loading

```
Error: Plugin not found
```

**Solution**: Check plugin is mounted in Docker

```bash
docker run -v $PWD:/var/lib/grafana/plugins/your-plugin-id ...
```

## Resources

### Official Documentation

- **@grafana/plugin-e2e**: https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/
- **Migration Guide**: https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/migrate-from-grafana-e2e
- **Playwright Docs**: https://playwright.dev/

### Examples

- **Plugin Examples**: https://github.com/grafana/grafana-plugin-examples
- **Data Source Example**: https://github.com/grafana/grafana-plugin-examples/tree/main/examples/datasource-http-backend
- **Panel Plugin Example**: https://github.com/grafana/grafana-plugin-examples/tree/main/examples/panel-basic

### Community

- **Forum**: https://community.grafana.com/c/plugin-development
- **GitHub Issues**: https://github.com/grafana/plugin-tools/issues
- **Slack**: #plugin-development channel

## This Plugin's E2E Tests

### Current Setup

- **Config**: `playwright.config.ts`
- **Tests**: `tests/*.spec.ts`
- **Coverage**: App navigation, configuration

### Running Tests

```bash
# Start Grafana with plugin
npm run server

# Run E2E tests
npm run e2e

# View test report
npx playwright show-report
```

### Test Files

- `tests/appNavigation.spec.ts` - App page navigation tests
- `tests/appConfig.spec.ts` - Plugin configuration tests
