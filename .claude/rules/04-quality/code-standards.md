---
paths: '**/*.{ts,tsx,js,jsx,go}'
---

# Code Quality Standards

This document defines code quality standards including formatting, linting, testing, and type safety requirements.

## Code Formatting

### Frontend (TypeScript/JavaScript)

**Tool**: Prettier 2.8.7

**Configuration** (`.prettierrc.js`):

```javascript
module.exports = {
  ...require('./.config/.prettierrc.js'),
};
```

**Grafana's Prettier Config** (`.config/.prettierrc.js`):

- **2-space indentation**
- **Single quotes**
- **120-character line width**
- **Semicolons required**
- **Trailing commas** (ES5 compatible)

**Commands**:

```bash
npm run lint:fix           # Auto-format all files
prettier --write .         # Format specific files
prettier --check .         # Verify formatting (CI)
```

**Pre-commit**: Formatting is enforced via pre-commit hooks.

### Backend (Go)

**Tool**: `gofmt` (built-in)

**Commands**:

```bash
go fmt ./...                    # Format all Go files
gofmt -s -w pkg/                # Simplify and write
```

**Standards**:

- **Tabs for indentation** (Go convention)
- **Max line length**: 120 characters (soft limit)
- **Import grouping**: stdlib → external → internal
- **Error handling**: Always check errors explicitly

**Example**:

```go
package plugin

import (
    "context"
    "fmt"

    "github.com/grafana/grafana-plugin-sdk-go/backend"

    "github.com/yourorg/plugin/pkg/models"
)
```

## Linting

### Frontend Linting

**Tool**: ESLint 9.0 with TypeScript support

**Configuration** (`eslint.config.mjs`):

```javascript
import baseConfig from './.config/eslint.config.mjs';

export default defineConfig([{ ignores: ['dist/', 'node_modules/', 'coverage/'] }, ...baseConfig]);
```

**Key Rules**:

- `@typescript-eslint/no-unused-vars`: Error
- `@typescript-eslint/no-explicit-any`: Warn
- `react/prop-types`: Off (TypeScript provides types)
- `react-hooks/rules-of-hooks`: Error
- `react-hooks/exhaustive-deps`: Warn

**Commands**:

```bash
npm run lint           # Run linter
npm run lint:fix       # Auto-fix issues
```

**CI Enforcement**: Linting must pass before merge.

### Backend Linting

**Tool**: `golangci-lint` (recommended)

**Install**:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Configuration** (`.golangci.yml` - recommended):

```yaml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign

linters-settings:
  govet:
    check-shadowing: true
  errcheck:
    check-blank: true

run:
  timeout: 5m
  skip-dirs:
    - vendor
```

**Commands**:

```bash
golangci-lint run                  # Run all linters
golangci-lint run --fix            # Auto-fix issues
```

## Type Safety

### TypeScript Configuration

**File**: `tsconfig.json`

**Extends**: `.config/tsconfig.json` (Grafana's base config)

**Key Settings**:

```json
{
  "compilerOptions": {
    "strict": true, // Enable all strict checks
    "noImplicitAny": true, // Error on implicit any
    "strictNullChecks": true, // Error on potential null/undefined
    "noUnusedLocals": true, // Error on unused variables
    "noUnusedParameters": true, // Error on unused params
    "noImplicitReturns": true, // Error on missing return statements
    "esModuleInterop": true,
    "skipLibCheck": true,
    "jsx": "react",
    "target": "ES2015",
    "module": "ESNext"
  }
}
```

**Type Checking**:

```bash
npm run typecheck              # Run TypeScript compiler (no emit)
```

**CI Requirement**: Must pass type checking before merge.

### Go Type Safety

Go is statically typed by default. Follow these practices:

**DO**:

```go
//  Explicit types for clarity
func processUser(id string) (*User, error) {
    user := &User{
        ID:   id,
        Name: "John Doe",
    }
    return user, nil
}

//  Use custom types for domain concepts
type UserID string
type OrgID int64

func GetUser(userID UserID, orgID OrgID) (*User, error) {
    // ...
}
```

**DON'T**:

```go
//  Avoid interface{} unless absolutely necessary
func process(data interface{}) error {
    // Loses type safety
}

//  Use generics (Go 1.18+) or specific types
func process[T any](data T) error {
    // Type-safe
}
```

## Testing Standards

### Frontend Testing

**Framework**: Jest 29.5 with React Testing Library

**Test Structure**:

```typescript
// ComponentName.test.tsx
import { render, screen } from '@testing-library/react';
import { ComponentName } from './ComponentName';

describe('ComponentName', () => {
  it('should render with default props', () => {
    render(<ComponentName />);
    expect(screen.getByText('Expected text')).toBeInTheDocument();
  });

  it('should handle user interaction', async () => {
    const handleClick = jest.fn();
    render(<ComponentName onClick={handleClick} />);

    await userEvent.click(screen.getByRole('button'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });
});
```

**Coverage Requirements**:

- **Minimum**: 70% overall coverage
- **Target**: 80%+ for critical paths
- **Components**: Test all public components
- **Services**: Test all service functions
- **Hooks**: Test custom hooks

**Commands**:

```bash
npm run test              # Watch mode
npm run test:ci           # Single run with coverage
npm run test -- --coverage # Coverage report
```

**What to Test**:
Component rendering
User interactions
State changes
API integration (mocked)
Error handling
Edge cases

**What NOT to Test**:
Third-party libraries
Trivial getters/setters
Generated code
Constants

### Backend Testing

**Framework**: Go testing package

**Test Structure**:

```go
// feature_test.go
package plugin

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeature(t *testing.T) {
    t.Run("should handle valid input", func(t *testing.T) {
        result, err := ProcessInput("valid")
        require.NoError(t, err)
        assert.Equal(t, "expected", result)
    })

    t.Run("should return error for invalid input", func(t *testing.T) {
        _, err := ProcessInput("")
        require.Error(t, err)
        assert.Contains(t, err.Error(), "invalid input")
    })
}
```

**Coverage Requirements**:

- **Minimum**: 80% code coverage
- **Target**: 90%+ for critical paths
- **Security**: 100% coverage for validation, auth, rate limiting

**Commands**:

```bash
mage -v coverage                   # Run tests with coverage
go test ./... -v                   # Run all tests
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out   # View coverage report
```

**Table-Driven Tests** (recommended):

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    bool
        wantErr bool
    }{
        {name: "valid query", input: "up", want: true, wantErr: false},
        {name: "empty query", input: "", want: false, wantErr: true},
        {name: "malicious query", input: "'; DROP TABLE", want: false, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Validate(tt.input)
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.want, got)
            }
        })
    }
}
```

### E2E Testing

**Framework**: Playwright with @grafana/plugin-e2e

**Test Structure**:

```typescript
// e2e/plugin.spec.ts
import { test, expect } from '@grafana/plugin-e2e';

test('plugin should load successfully', async ({ page, createDataSourceConfigPage }) => {
  const configPage = await createDataSourceConfigPage({ type: 'your-plugin-id' });
  await expect(configPage.saveButton).toBeEnabled();
});

test('user can create a query', async ({ page, gotoPanel }) => {
  await gotoPanel('panel-id');
  await page.getByLabel('Query').fill('up');
  await expect(page.getByText('Success')).toBeVisible();
});
```

**Commands**:

```bash
npm run e2e                    # Run E2E tests
npm run e2e -- --headed        # Run with browser visible
npm run e2e -- --debug         # Debug mode
npx playwright show-report     # View test report
```

**E2E Coverage**:

- Plugin loads and renders
- Configuration saves correctly
- Queries execute successfully
- Error states display properly
- Critical user flows work end-to-end

## Code Review Standards

### Required Checks (CI Pipeline)

Before merging, all of the following must pass:

1.  **Type checking**: `npm run typecheck`
2.  **Linting**: `npm run lint`
3.  **Formatting**: `prettier --check .`
4.  **Unit tests**: `npm run test:ci` (frontend)
5.  **Backend tests**: `mage -v coverage` (Go)
6.  **E2E tests**: `npm run e2e`
7.  **Build**: `npm run build && mage -v buildAll`
8.  **Plugin validation**: Grafana plugin validator

### Local CI Pipeline

Run the full CI pipeline locally before pushing:

```bash
./ci-local.sh
```

This script runs:

1. `npm ci` - Clean install
2. `npm run typecheck` - Type checking
3. `npm run lint` - Linting
4. `npm run test:ci` - Frontend tests
5. `npm run build` - Frontend build
6. `mage -v coverage` - Backend tests
7. `mage -v buildAll` - Backend build
8. Plugin validation

### Code Review Checklist

**Functionality**:

- [ ] Code solves the stated problem
- [ ] No regressions in existing functionality
- [ ] Edge cases are handled
- [ ] Error handling is appropriate

**Code Quality**:

- [ ] Follows KISS principles (simple, not over-engineered)
- [ ] Functions are small (< 30 lines)
- [ ] Names are clear and descriptive
- [ ] No duplicate code (DRY, but wait for 3+ uses)
- [ ] Comments explain WHY, not WHAT

**Testing**:

- [ ] Unit tests cover new code
- [ ] Tests are meaningful (not just for coverage)
- [ ] E2E tests for critical paths
- [ ] Tests pass locally and in CI

**Security**:

- [ ] Input validation at boundaries
- [ ] No secrets in code
- [ ] User permissions respected
- [ ] XSS prevention (sanitize output)
- [ ] No SQL injection risks

**Performance**:

- [ ] No obvious performance issues
- [ ] Large data sets handled efficiently
- [ ] No memory leaks

**Documentation**:

- [ ] Complex logic is documented
- [ ] API changes documented
- [ ] README updated if needed

## Git Commit Standards

### Commit Message Format

```
<type>: <subject>

<body (optional)>

<footer (optional)>
```

**Types**:

- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring (no functional change)
- `test`: Adding or updating tests
- `docs`: Documentation changes
- `chore`: Build process, tooling, dependencies
- `perf`: Performance improvements
- `style`: Code formatting (not CSS)

**Examples**:

```
feat: add query validation for PromQL

Implements manual pattern-based validation to prevent injection
attacks. Includes complexity scoring and function allowlist.

Closes #123
```

```
fix: prevent XSS in LLM output

Sanitize LLM responses with DOMPurify before rendering to prevent
cross-site scripting attacks.
```

```
chore: update Grafana SDK to 12.3.0

Update all @grafana/* packages to latest stable version.
```

### Commit Best Practices

**DO**:

- Write clear, descriptive commit messages
- Keep commits focused (one logical change per commit)
- Reference issue numbers (Closes #123)
- Use imperative mood ("add feature" not "added feature")

**DON'T**:

- Commit commented code
- Commit debug console.log statements
- Commit large, unfocused changes
- Use vague messages like "fix bug" or "update code"

## Pre-Commit Hooks

**Location**: `.git-hooks/pre-commit`

**Checks**:

1. Type checking (`npm run typecheck`)
2. Linting (`npm run lint`)
3. Formatting verification
4. No uncommitted changes to generated files

**Setup**:

```bash
git config core.hooksPath .git-hooks
```

**Skip** (not recommended):

```bash
git commit --no-verify
```

## Continuous Integration

### GitHub Actions Workflows

1. **`.github/workflows/ci.yml`** - Main CI pipeline

   - Runs on every push and PR
   - Matrix testing across Grafana versions
   - Uploads test reports and coverage

2. **`.github/workflows/release.yml`** - Release pipeline

   - Triggers on version tags (v\*)
   - Signs plugin
   - Creates GitHub release
   - Publishes to Grafana plugin catalog

3. **`.github/workflows/is-compatible.yml`** - Compatibility check
   - Validates plugin against Grafana versions
   - Runs on schedule and PRs

### CI Requirements for Merge

All checks must pass:

- Type checking
- Linting
- Unit tests (>70% coverage)
- E2E tests
- Build succeeds
- No security vulnerabilities

## Performance Standards

### Frontend Performance

**Targets**:

- Time to Interactive (TTI): < 3 seconds
- First Contentful Paint (FCP): < 1.5 seconds
- Bundle size: < 1 MB (gzipped)

**Optimization Techniques**:

- Code splitting (React.lazy)
- Tree shaking (remove unused code)
- Minimize dependencies
- Use production builds
- Lazy load non-critical components

**Bundle Analysis**:

```bash
npm run build -- --env analyze
```

### Backend Performance

**Targets**:

- API response time: < 200ms (p95)
- Query execution: < 5 seconds (p99)
- Memory usage: < 100 MB per instance
- CPU usage: < 50% average

**Profiling**:

```bash
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Security Standards

### Static Analysis

**Frontend**:

- ESLint security rules enabled
- No `eval()` or `Function()` constructor
- No `dangerouslySetInnerHTML` without sanitization
- Use DOMPurify for HTML sanitization

**Backend**:

- `gosec` for security scanning
- `golangci-lint` with security linters
- No SQL string concatenation
- Input validation at all boundaries

**Commands**:

```bash
npm audit                      # Check npm dependencies
gosec ./...                    # Go security scanner
```

### Dependency Management

**Audit Dependencies**:

```bash
npm audit                      # Check for vulnerabilities
npm audit fix                  # Auto-fix vulnerabilities
go list -m -u all              # Check for Go updates
```

**Update Policy**:

- Security updates: Apply immediately
- Minor/patch updates: Monthly
- Major updates: Quarterly (with testing)

## Documentation Standards

### Code Documentation

**TypeScript/JSDoc**:

```typescript
/**
 * Executes a PromQL query against Prometheus datasource.
 *
 * @param query - PromQL expression
 * @param datasource - Datasource UID
 * @param timeRange - Time range for query
 * @returns Query result with data frames
 * @throws {ValidationError} If query is invalid
 */
export async function executeQuery(query: string, datasource: string, timeRange: TimeRange): Promise<QueryResult> {
  // Implementation
}
```

**Go Documentation**:

```go
// ExecuteQuery executes a PromQL query against the specified datasource.
// It validates the query, applies rate limiting, and returns data frames.
//
// Parameters:
//   - query: PromQL expression to execute
//   - datasource: UID of the Prometheus datasource
//   - timeRange: Time range for the query
//
// Returns:
//   - QueryResult containing data frames
//   - error if validation fails or query execution fails
func ExecuteQuery(query string, datasource string, timeRange TimeRange) (*QueryResult, error) {
    // Implementation
}
```

### README Standards

Every module/feature should have:

- Purpose and overview
- Setup instructions
- Usage examples
- API documentation
- Contributing guidelines
- License information

## Summary

**Quality Gates** (must pass):

1.  Type checking
2.  Linting
3.  Formatting
4.  Unit tests (>70% coverage)
5.  E2E tests
6.  Build succeeds
7.  Security audit passes

**Before Every Commit**:

- Run `npm run typecheck`
- Run `npm run lint`
- Run `npm run test:ci`
- Review your changes (git diff)

**Before Every Push**:

- Run `./ci-local.sh` (full CI pipeline)
- Ensure all tests pass
- Verify no console warnings/errors

**Code Quality = Simplicity + Testing + Type Safety**
