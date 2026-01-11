# Clean Code & KISS Methodology

This document defines clean code principles and KISS (Keep It Simple, Stupid) methodology for this project.

## Core Philosophy

> "Simplicity is the ultimate sophistication." - Leonardo da Vinci

**Primary Goal**: Write code that is simple, readable, and maintainable. Complexity is the enemy of quality software.

## KISS Principles (Keep It Simple, Stupid)

### 1. Solve the Actual Problem

**DO**: Address the requirement directly without adding extra features.

```typescript
//  BAD - Over-engineered
class ConfigurationManager {
  private cache: Map<string, any>;
  private observers: Set<Function>;
  private validator: ConfigValidator;

  constructor() {
    this.cache = new Map();
    this.observers = new Set();
    this.validator = new ConfigValidator();
  }

  get(key: string): any {
    return this.cache.get(key);
  }

  set(key: string, value: any): void {
    this.validator.validate(key, value);
    this.cache.set(key, value);
    this.notifyObservers(key, value);
  }

  notifyObservers(key: string, value: any): void {
    this.observers.forEach((fn) => fn(key, value));
  }
}

//  GOOD - Simple and direct
const config = {
  apiUrl: 'https://api.example.com',
  timeout: 5000,
};
```

**Why**: The simple object solves the problem. Only add complexity when you need observers, validation, etc.

### 2. Prefer Simple Solutions

**DO**: Use the simplest approach that works. Don't invent complex solutions to simple problems.

```typescript
//  BAD - Unnecessarily complex
function getUserName(user: User): string {
  return user?.profile?.personalInfo?.displayName ??
    (user?.profile?.personalInfo?.firstName && user?.profile?.personalInfo?.lastName)
    ? `${user.profile.personalInfo.firstName} ${user.profile.personalInfo.lastName}`
    : user?.account?.username ?? 'Anonymous';
}

//  GOOD - Simple and clear
function getUserName(user: User): string {
  if (user.displayName) return user.displayName;
  if (user.firstName && user.lastName) return `${user.firstName} ${user.lastName}`;
  return user.username || 'Anonymous';
}
```

### 3. Avoid Over-Engineering

**DO**: Wait for 3+ similar uses before creating abstractions.

```go
//  BAD - Premature abstraction
type QueryBuilder interface {
    Build() string
    WithFilter(filter string) QueryBuilder
    WithLimit(limit int) QueryBuilder
    WithOffset(offset int) QueryBuilder
}

type PromQLBuilder struct {
    query string
    filters []string
    limit int
    offset int
}

// Used only once in the entire codebase

//  GOOD - Direct implementation
func buildPromQLQuery(metric string, labels map[string]string) string {
    labelPairs := []string{}
    for k, v := range labels {
        labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, k, v))
    }
    return fmt.Sprintf(`%s{%s}`, metric, strings.Join(labelPairs, ","))
}
```

### 4. Delete Unused Code

**DO**: Remove code that isn't being used. Don't keep it "just in case."

```typescript
//  BAD - Commented "just in case"
function fetchData(url: string): Promise<Response> {
  return fetch(url);

  // Old implementation using XMLHttpRequest
  // const xhr = new XMLHttpRequest();
  // xhr.open('GET', url);
  // xhr.send();
  // return new Promise((resolve, reject) => {
  //   xhr.onload = () => resolve(xhr.response);
  //   xhr.onerror = () => reject(xhr.statusText);
  // });
}

//  GOOD - Clean and focused
function fetchData(url: string): Promise<Response> {
  return fetch(url);
}
```

**Why**: Version control keeps history. Commented code is visual noise and becomes outdated.

### 5. Minimal Dependencies

**DO**: Every dependency is a liability. Only add dependencies when truly necessary.

```json
//  BAD - Unnecessary dependencies
{
  "dependencies": {
    "lodash": "^4.17.21",        // Used only for _.isEqual()
    "moment": "^2.29.4",         // Used only for date formatting
    "axios": "^1.6.0",           // Grafana provides fetch
    "uuid": "^9.0.0"             // Browser has crypto.randomUUID()
  }
}

//  GOOD - Use built-ins or existing dependencies
{
  "dependencies": {
    "@grafana/runtime": "^12.3.0"  // Already provides fetch
  }
}
```

**When to add a dependency**:

1. Solves a complex problem (e.g., markdown parsing)
2. Widely used and well-maintained
3. No reasonable built-in alternative
4. Worth the bundle size cost

## Clean Code Principles

### 1. Meaningful Names

**Use descriptive names that reveal intent.**

```typescript
//  BAD - Cryptic names
function calc(d: number): number {
  return d * 1000 * 60 * 60 * 24;
}

const t = calc(7);

//  GOOD - Clear names
function convertDaysToMilliseconds(days: number): number {
  const MILLISECONDS_PER_DAY = 1000 * 60 * 60 * 24;
  return days * MILLISECONDS_PER_DAY;
}

const weekInMilliseconds = convertDaysToMilliseconds(7);
```

### 2. Functions Should Do One Thing

**A function should have a single, clear purpose.**

```typescript
//  BAD - Function does too much
function saveUser(user: User): void {
  validateEmail(user.email);
  validatePassword(user.password);
  hashPassword(user.password);
  const userId = database.insert(user);
  sendWelcomeEmail(user.email);
  logAuditEvent('user_created', userId);
}

//  GOOD - Single responsibility
function saveUser(user: User): string {
  const validatedUser = validateUser(user);
  const hashedUser = hashUserPassword(validatedUser);
  return database.insert(hashedUser);
}
```

### 3. Small Functions

**Functions should be small. Really small.**

**Target**: 5-15 lines per function
**Maximum**: 30 lines (consider splitting beyond this)

```go
//  BAD - 50+ line function
func handleUserRequest(w http.ResponseWriter, r *http.Request) {
    // Extract user from context
    user, ok := r.Context().Value("user").(User)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse request body
    var req RequestBody
    decoder := json.NewDecoder(r.Body)
    if err := decoder.Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Validate input
    if req.Name == "" {
        http.Error(w, "Name required", http.StatusBadRequest)
        return
    }

    // ... 40 more lines of processing, database calls, etc.
}

//  GOOD - Small, focused functions
func handleUserRequest(w http.ResponseWriter, r *http.Request) {
    user, err := extractUser(r.Context())
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Unauthorized")
        return
    }

    req, err := parseRequestBody(r.Body)
    if err != nil {
        respondWithError(w, http.StatusBadRequest, "Invalid request")
        return
    }

    if err := validateRequest(req); err != nil {
        respondWithError(w, http.StatusBadRequest, err.Error())
        return
    }

    result := processRequest(user, req)
    respondWithJSON(w, http.StatusOK, result)
}
```

### 4. Comments: Quality Over Quantity

**Write comments that clarify WHY, not WHAT.**

```go
//  BAD - States the obvious
// Set user to admin
user.role = "admin"

// Increment counter by 1
counter++

//  BAD - Repeats what code already says
// Loop through all users
for _, user := range users {
    // Check if user is active
    if user.Active {
        // Add user to active list
        activeUsers = append(activeUsers, user)
    }
}

//  GOOD - Explains WHY, not WHAT
// Use cached value if less than 5 minutes old to reduce API calls
if time.Since(cache.lastRefresh) < 5*time.Minute {
    return cache.value
}

// Parser expects closing brace even inside strings, so we track string state
if !inString && ch == '}' {
    depth--
}

// Empty allowlist means all functions are allowed (backwards compatibility)
if len(allowedFunctions) == 0 {
    return nil
}
```

**When to comment**:

- Complex algorithms or non-obvious logic
- Important edge cases or gotchas
- Security-critical code sections
- Workarounds for external library bugs
- Performance optimizations with trade-offs

**When NOT to comment**:

- Function signatures (use clear names instead)
- Variable declarations (use descriptive names)
- Simple loops or conditionals
- Code that reads like plain English

### 5. Error Handling

**Handle errors explicitly at boundaries. Don't add error handling for impossible scenarios.**

```typescript
//  BAD - Error handling for internal functions
function add(a: number, b: number): number {
  if (typeof a !== 'number' || typeof b !== 'number') {
    throw new Error('Invalid arguments'); // TypeScript already ensures this
  }
  return a + b;
}

//  GOOD - Error handling at boundaries only
// External API - validate input
export function processUserInput(input: string): Result {
  if (!input || input.trim() === '') {
    return { error: 'Input cannot be empty' };
  }
  return processInternal(input);
}

// Internal function - trust caller
function processInternal(input: string): Result {
  return { value: input.toUpperCase() };
}
```

### 6. Avoid Feature Flags and Backward Compatibility Hacks

**When you can just change the code, change it.**

```typescript
//  BAD - Unnecessary feature flag
const USE_NEW_ALGORITHM = true;

function processData(data: Data): Result {
  if (USE_NEW_ALGORITHM) {
    return newAlgorithm(data);
  } else {
    return oldAlgorithm(data); // Dead code
  }
}

//  GOOD - Just use the new algorithm
function processData(data: Data): Result {
  return newAlgorithm(data);
}
```

**When backward compatibility IS needed**:

- Public APIs with external consumers
- Database schema migrations
- Plugin versions with different capabilities

### 7. Avoid Premature Configuration

**Start with sensible defaults. Add configuration only when needed.**

```go
//  BAD - Configuration for everything
type Config struct {
    MaxRetries              int
    RetryDelay              time.Duration
    ExponentialBackoff      bool
    MaxBackoffDelay         time.Duration
    TimeoutSeconds          int
    EnableMetrics           bool
    MetricsPrefix           string
    EnableDebugLogging      bool
    LogFormat               string
    // ... 20 more options
}

//  GOOD - Sensible defaults, configure only what matters
type Config struct {
    Timeout time.Duration  // Default: 30s
    APIKey  string         // Required
}
```

## Refactoring Guidelines

### When to Refactor

**DO refactor when**:

- You see duplicate code in 3+ places
- A function exceeds 30 lines
- You need to add comments to explain complex logic
- You see deeply nested conditionals (3+ levels)
- Tests are difficult to write

**DON'T refactor when**:

- Code works and is clear enough
- You're changing functionality at the same time (refactor separately)
- The abstraction would be used only once or twice
- It would make the code harder to understand

### Refactoring Process

1. **Ensure tests exist** - Write tests before refactoring
2. **Make it work** - Get the feature working first
3. **Make it right** - Refactor for clarity and simplicity
4. **Make it fast** - Optimize only if necessary

### Example: Refactoring Complex Logic

```typescript
// BEFORE - Complex and hard to understand
function processRequest(req: Request): Response {
  if (
    (req.type === 'A' && req.status === 'active') ||
    (req.type === 'B' && req.priority > 5) ||
    (req.type === 'C' && req.user.role === 'admin')
  ) {
    if (req.data && req.data.length > 0) {
      const results = [];
      for (let i = 0; i < req.data.length; i++) {
        if (req.data[i].valid) {
          results.push(transform(req.data[i]));
        }
      }
      return { success: true, data: results };
    }
  }
  return { success: false, error: 'Invalid request' };
}

// AFTER - Clear and simple
function processRequest(req: Request): Response {
  if (!isEligibleRequest(req)) {
    return { success: false, error: 'Invalid request' };
  }

  if (!hasValidData(req)) {
    return { success: false, error: 'Invalid request' };
  }

  const results = req.data.filter((item) => item.valid).map((item) => transform(item));

  return { success: true, data: results };
}

function isEligibleRequest(req: Request): boolean {
  return (
    (req.type === 'A' && req.status === 'active') ||
    (req.type === 'B' && req.priority > 5) ||
    (req.type === 'C' && req.user.role === 'admin')
  );
}

function hasValidData(req: Request): boolean {
  return req.data && req.data.length > 0;
}
```

## Code Review Checklist

When reviewing code (or your own code), ask:

### Simplicity

- [ ] Is this the simplest solution that works?
- [ ] Can I remove any code without losing functionality?
- [ ] Are there any premature abstractions?

### Readability

- [ ] Can someone unfamiliar with this code understand it quickly?
- [ ] Are variable and function names descriptive?
- [ ] Is the function small enough (< 30 lines)?

### Maintainability

- [ ] Will this be easy to change later?
- [ ] Are edge cases documented?
- [ ] Are there tests for this code?

### Necessity

- [ ] Is this feature actually needed?
- [ ] Can we use an existing solution instead?
- [ ] Are all dependencies necessary?

## Anti-Patterns to Avoid

### 1. God Objects

Classes/modules that do everything.

**Solution**: Split into focused, single-responsibility modules.

### 2. Premature Optimization

Optimizing before knowing there's a performance problem.

**Solution**: Make it work, then profile, then optimize.

### 3. Magic Numbers

Unexplained numeric constants in code.

```typescript
//  BAD
if (user.age > 18 && user.status === 2) { ... }

//  GOOD
const MINIMUM_AGE = 18;
const STATUS_ACTIVE = 2;
if (user.age > MINIMUM_AGE && user.status === STATUS_ACTIVE) { ... }
```

### 4. Long Parameter Lists

Functions with 5+ parameters.

**Solution**: Group related parameters into objects.

### 5. Duplicate Code

Copy-pasted code blocks.

**Solution**: Extract common logic into shared functions (when used 3+ times).

## Summary: KISS in Practice

**Simple is not**:

- Fewer features → It's the right features, clearly implemented
- Less code → It's clear code that solves the problem
- No abstractions → It's abstractions when they reduce complexity

**Simple is**:

- Easy to understand
- Easy to change
- Easy to test
- Easy to debug

**Golden Rule**: When in doubt, choose the simpler option. You can always add complexity later if needed, but removing complexity is much harder.

## Resources

- **Clean Code** by Robert C. Martin
- **The Pragmatic Programmer** by Hunt & Thomas
- **YAGNI Principle**: You Aren't Gonna Need It
- **DRY Principle**: Don't Repeat Yourself (but wait for 3+ uses)
- **SOLID Principles**: For object-oriented design
