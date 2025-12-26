# Testing Overview

This document provides a comprehensive overview of testing strategies, tools, and practices for the Zagalin plugin.

## Testing Philosophy

Zagalin follows a multi-layered testing approach to ensure:
- **Reliability**: Plugin works across different Grafana versions
- **Stability**: Changes don't break existing functionality
- **Performance**: Meets performance requirements
- **Security**: No vulnerabilities or data leaks
- **User Experience**: UI behaves as expected

## Testing Pyramid

```
         /\
        /E2E\        ← Few, slow, expensive
       /------\
      /Integration\ ← Some, medium speed
     /------------\
    /  Unit Tests  \ ← Many, fast, cheap
   /________________\
```

### Distribution
- **70%** Unit Tests - Fast feedback, isolated components
- **20%** Integration Tests - Component interactions
- **10%** E2E Tests - Critical user flows

## Testing Tools

### 1. Jest (Unit Testing)

**Purpose**: Test individual functions, components, and utilities

**Configuration**: `jest.config.js`
```javascript
module.exports = {
  preset: '@grafana/create-plugin',
  moduleNameMapper: {
    '\\.(css|scss)$': 'identity-obj-proxy',
  },
  setupFilesAfterEnv: ['<rootDir>/jest-setup.js'],
};
```

**Usage**:
```bash
# Run all tests
npm test

# Watch mode
npm test -- --watch

# Coverage report
npm test -- --coverage

# Specific test file
npm test ChatPanel.test.tsx
```

### 2. React Testing Library

**Purpose**: Test React components in a user-centric way

**Key Principles**:
- Test behavior, not implementation
- Query by accessibility attributes
- Simulate real user interactions

**Example**:
```typescript
import { render, screen, fireEvent } from '@testing-library/react';

test('sends message when button clicked', async () => {
  render(<ChatPanel />);

  const input = screen.getByPlaceholderText('Ask anything...');
  const button = screen.getByRole('button', { name: /send/i });

  fireEvent.change(input, { target: { value: 'Hello' } });
  fireEvent.click(button);

  expect(await screen.findByText('Hello')).toBeInTheDocument();
});
```

### 3. Playwright (@grafana/plugin-e2e)

**Purpose**: End-to-end testing across multiple Grafana versions

**Why @grafana/plugin-e2e?**
- Handles UI variations across Grafana versions
- Pre-built fixtures for plugin testing
- Custom assertions for Grafana elements
- Consistent selectors across versions

**Configuration**: `playwright.config.ts`
```typescript
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
```

**Usage**:
```bash
# Run E2E tests
npm run e2e

# Run in UI mode
npm run e2e:ui

# Run specific test
npm run e2e tests/chat.spec.ts

# Debug mode
npm run e2e -- --debug
```

### 4. TypeScript Compiler

**Purpose**: Type checking and validation

```bash
# Type check all files
npm run typecheck

# Watch mode
npm run typecheck -- --watch
```

### 5. ESLint

**Purpose**: Code quality and style enforcement

```bash
# Lint all files
npm run lint

# Fix auto-fixable issues
npm run lint:fix
```

## Test Types

### Unit Tests

**Location**: `src/**/*.test.tsx` or `src/**/*.test.ts`

**What to Test**:
- Pure functions and utilities
- React component rendering
- State management logic
- Service methods
- Data transformations

**Example: Testing ConversationStorage**

```typescript
// src/services/conversationStorage.test.ts
import { ConversationStorage } from './conversationStorage';

describe('ConversationStorage', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('creates a new conversation', () => {
    const conversation = ConversationStorage.createConversation();

    expect(conversation).toHaveProperty('id');
    expect(conversation).toHaveProperty('title');
    expect(conversation.messages).toEqual([]);
  });

  it('saves and retrieves conversation', () => {
    const conversation = ConversationStorage.createConversation();
    ConversationStorage.saveConversation(conversation);

    const retrieved = ConversationStorage.getConversation(conversation.id);
    expect(retrieved).toEqual(conversation);
  });

  it('auto-prunes old conversations', () => {
    // Create 51 conversations
    for (let i = 0; i < 51; i++) {
      const conv = ConversationStorage.createConversation();
      ConversationStorage.saveConversation(conv);
    }

    const list = ConversationStorage.getConversationList();
    expect(list).toHaveLength(50);
  });
});
```

**Example: Testing React Component**

```typescript
// src/components/FloatingChat/ChatPanel.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ChatPanel } from './ChatPanel';

jest.mock('@grafana/llm', () => ({
  llm: {
    streamChatCompletions: jest.fn(),
    accumulateContent: jest.fn(),
  },
}));

describe('ChatPanel', () => {
  it('renders chat interface', () => {
    render(<ChatPanel />);

    expect(screen.getByPlaceholderText('Ask anything...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /send/i })).toBeInTheDocument();
  });

  it('sends message on button click', async () => {
    const user = userEvent.setup();
    render(<ChatPanel />);

    const input = screen.getByPlaceholderText('Ask anything...');
    const button = screen.getByRole('button', { name: /send/i });

    await user.type(input, 'Test message');
    await user.click(button);

    await waitFor(() => {
      expect(screen.getByText('Test message')).toBeInTheDocument();
    });
  });
});
```

### Integration Tests

**Purpose**: Test component interactions and data flow

**Example: Context Service Integration**

```typescript
// src/services/contextService.integration.test.ts
import { ContextService } from './contextService';
import { getBackendSrv } from '@grafana/runtime';

jest.mock('@grafana/runtime');

describe('ContextService Integration', () => {
  it('extracts dashboard context from Grafana API', async () => {
    const mockDashboard = {
      dashboard: {
        uid: 'test-uid',
        title: 'Test Dashboard',
        panels: [/* ... */],
      },
    };

    (getBackendSrv as jest.Mock).mockReturnValue({
      get: jest.fn().resolvedValue(mockDashboard),
    });

    const context = await ContextService.getContext();

    expect(context.dashboard).toBeDefined();
    expect(context.dashboard?.uid).toBe('test-uid');
  });
});
```

### End-to-End Tests

**Location**: `tests/e2e/*.spec.ts`

**What to Test**:
- Critical user workflows
- Plugin installation and configuration
- Cross-version compatibility
- Integration with Grafana features

**Example: Chat Flow**

```typescript
// tests/e2e/chat.spec.ts
import { test, expect } from '@grafana/plugin-e2e';

test('should send and receive chat messages', async ({ page }) => {
  // Navigate to dashboard
  await page.goto('/d/test-dashboard');

  // Open floating chat
  await page.click('[data-testid="floating-chat-button"]');

  // Type and send message
  const input = page.locator('textarea[placeholder="Ask anything..."]');
  await input.fill('Show me CPU usage');
  await page.click('button:has-text("Send")');

  // Verify message appears
  await expect(page.locator('text=Show me CPU usage')).toBeVisible();

  // Wait for LLM response
  await expect(page.locator('[data-testid="assistant-message"]'))
    .toBeVisible({ timeout: 10000 });
});

test('should persist conversation across page reload', async ({ page }) => {
  await page.goto('/d/test-dashboard');

  // Send message
  await page.click('[data-testid="floating-chat-button"]');
  await page.fill('textarea', 'Test persistence');
  await page.click('button:has-text("Send")');

  // Reload page
  await page.reload();

  // Open chat and verify message persists
  await page.click('[data-testid="floating-chat-button"]');
  await expect(page.locator('text=Test persistence')).toBeVisible();
});
```

## Test Coverage Goals

### Overall Coverage
- **Target**: 80% code coverage
- **Critical paths**: 100% coverage
- **UI components**: 70% coverage
- **Services**: 90% coverage
- **Utilities**: 100% coverage

### Coverage Reports

```bash
# Generate coverage report
npm test -- --coverage

# View HTML report
open coverage/lcov-report/index.html
```

### Coverage Thresholds

`jest.config.js`:
```javascript
module.exports = {
  coverageThresholds: {
    global: {
      statements: 80,
      branches: 75,
      functions: 80,
      lines: 80,
    },
  },
};
```

## Continuous Integration

### GitHub Actions Workflow

`.github/workflows/test.yml`:
```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Install dependencies
        run: npm ci

      - name: Run linter
        run: npm run lint

      - name: Type check
        run: npm run typecheck

      - name: Run unit tests
        run: npm test -- --coverage

      - name: Upload coverage
        uses: codecov/codecov-action@v3

      - name: Build plugin
        run: npm run build

      - name: Run E2E tests
        run: npm run e2e
```

## Testing Best Practices

### 1. Write Testable Code

**Good**: Pure functions, dependency injection
```typescript
// ✅ Testable
export function formatTime(date: Date): string {
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  // ...
}

// ✅ Testable with DI
export function sendMessage(
  message: string,
  llmService: LLMService
): Promise<Response> {
  return llmService.send(message);
}
```

**Bad**: Tight coupling, hidden dependencies
```typescript
// ❌ Hard to test
export function formatTime(): string {
  const date = new Date(); // Hidden dependency
  // ...
}

// ❌ Hard to test
export function sendMessage(message: string): Promise<Response> {
  return llm.send(message); // Global dependency
}
```

### 2. Use Test Doubles

**Mocks**: Replace implementations
```typescript
jest.mock('@grafana/llm');
```

**Stubs**: Provide fixed responses
```typescript
const mockLLM = {
  streamChatCompletions: jest.fn().mockReturnValue(of('response')),
};
```

**Spies**: Monitor function calls
```typescript
const spy = jest.spyOn(ConversationStorage, 'saveConversation');
expect(spy).toHaveBeenCalledWith(expect.objectContaining({ id: '123' }));
```

### 3. Follow AAA Pattern

```typescript
test('adds message to conversation', () => {
  // Arrange
  const conversation = ConversationStorage.createConversation();
  const message = { role: 'user', content: 'Hello', timestamp: new Date() };

  // Act
  ConversationStorage.addMessage(conversation.id, message);

  // Assert
  const updated = ConversationStorage.getConversation(conversation.id);
  expect(updated?.messages).toHaveLength(1);
});
```

### 4. Test Edge Cases

```typescript
describe('ConversationStorage edge cases', () => {
  it('handles invalid conversation ID', () => {
    const result = ConversationStorage.getConversation('invalid');
    expect(result).toBeNull();
  });

  it('handles localStorage quota exceeded', () => {
    jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('QuotaExceededError');
    });

    expect(() => {
      ConversationStorage.saveConversation(largeConversation);
    }).toThrow();
  });
});
```

### 5. Keep Tests Fast

```typescript
// ✅ Fast
expect(calculateSum(2, 2)).toBe(4);

// ❌ Slow
await new Promise(resolve => setTimeout(resolve, 1000));

// ✅ Mock timers instead
jest.useFakeTimers();
jest.advanceTimersByTime(1000);
```

## Debugging Tests

### Jest Debug Mode

```bash
# Run in debug mode
node --inspect-brk node_modules/.bin/jest --runInBand

# Then open chrome://inspect in Chrome
```

### VS Code Debug Configuration

`.vscode/launch.json`:
```json
{
  "type": "node",
  "request": "launch",
  "name": "Jest Debug",
  "program": "${workspaceFolder}/node_modules/.bin/jest",
  "args": ["--runInBand", "--no-cache"],
  "console": "integratedTerminal",
  "internalConsoleOptions": "neverOpen"
}
```

### Playwright Debug Mode

```bash
# Run with UI
npm run e2e:ui

# Debug specific test
PWDEBUG=1 npm run e2e tests/chat.spec.ts
```

## Common Testing Pitfalls

### 1. Testing Implementation Details

```typescript
// ❌ Don't test implementation
expect(component.state.isOpen).toBe(true);

// ✅ Test behavior
expect(screen.getByRole('dialog')).toBeVisible();
```

### 2. Overmocking

```typescript
// ❌ Mocking too much
jest.mock('./utils');
jest.mock('./service');
jest.mock('./helper');

// ✅ Mock external dependencies only
jest.mock('@grafana/llm');
```

### 3. Flaky Tests

```typescript
// ❌ Race condition
test('loads data', () => {
  loadData();
  expect(getData()).toBeDefined(); // May fail
});

// ✅ Wait for async operations
test('loads data', async () => {
  await loadData();
  expect(getData()).toBeDefined();
});
```

## Next Steps

- [Unit Testing Guide](./unit-tests.md)
- [E2E Testing Guide](./e2e-tests.md)
- [CI/CD Pipeline](./ci-cd.md)
- [Contributing Guidelines](../contributing/guidelines.md)

## Resources

- [Jest Documentation](https://jestjs.io/)
- [React Testing Library](https://testing-library.com/react)
- [Playwright Documentation](https://playwright.dev/)
- [@grafana/plugin-e2e](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin)
- [Testing Best Practices](https://testingjavascript.com/)
