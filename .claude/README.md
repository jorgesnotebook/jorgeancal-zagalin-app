# Claude Code Configuration

This directory contains configuration and guidance for Claude Code (AI assistant) when working with this Grafana plugin codebase.

## Directory Structure

```
.claude/
 CLAUDE.md                          # Main project instructions
 README.md                          # This file - overview of Claude configuration
 settings.local.json                # Permissions for tools (WebFetch, Bash)
 rules/                             # Modular, topic-specific rules
     grafana-plugin-standards.md    # Official Grafana plugin development standards
     grafana-llm-integration.md     # LLM integration patterns (grafana-llm-app)
     app-plugin-development.md      # App plugin development guide
     clean-code-principles.md       # KISS methodology and clean code practices
     code-quality-standards.md      # Formatting, linting, testing standards
```

## Purpose

This configuration enables Claude Code to:

- Understand the project's architecture and patterns
- Follow Grafana's official standards and best practices
- Apply KISS (Keep It Simple, Stupid) methodology
- Maintain consistent code quality and testing standards
- Integrate properly with grafana-llm-app
- Work effectively with this hybrid app plugin (React + Go)

## Modular Rules System

The `.claude/rules/` directory contains **focused, topic-specific documentation** that Claude Code automatically loads based on file paths (using frontmatter `paths` filters).

### Path-Specific Rules

Each rule file can specify which files it applies to using YAML frontmatter:

```markdown
---
paths: '**/*.{ts,tsx}'
---

# TypeScript-specific rules
```

### Rule Files Overview

#### 1. grafana-plugin-standards.md

**Scope**: All code files (`**/*.{ts,tsx,go}`)

**Content**:

- Official Grafana plugin development standards
- System requirements and project structure
- Build system and development workflow
- Backend plugin architecture (gRPC, subprocesses)
- Data frame standards
- Security and testing requirements
- Plugin signing and distribution

**When Claude uses this**: Working with any plugin code, build configuration, or architecture decisions.

#### 2. grafana-llm-integration.md

**Scope**: Frontend and backend files (`{src/**/*.{ts,tsx},pkg/**/*.go}`)

**Content**:

- Official Grafana LLM integration patterns
- `@grafana/llm` package usage
- Model Context Protocol (MCP) for agents
- Backend proxy pattern (this plugin's approach)
- Streaming responses with SSE
- Function calling / tool use
- Error handling and troubleshooting

**When Claude uses this**: Working with LLM features, chat interfaces, or assistant functionality.

#### 3. app-plugin-development.md

**Scope**: Frontend and backend files (`{src/**/*.{ts,tsx},pkg/**/*.go}`)

**Content**:

- App plugin architecture and capabilities
- Custom pages and navigation
- Backend integration patterns
- Authentication and user context
- Resource handlers (custom HTTP endpoints)
- Service accounts for background tasks
- Role-based access control (RBAC)
- Performance and security best practices

**When Claude uses this**: Implementing app plugin features, custom pages, or backend resources.

#### 4. clean-code-principles.md

**Scope**: All code files (no specific path filter)

**Content**:

- KISS (Keep It Simple, Stupid) principles
- Clean code practices (meaningful names, small functions)
- Comment guidelines (explain WHY, not WHAT)
- Refactoring guidelines
- Anti-patterns to avoid
- Code review checklist

**When Claude uses this**: Writing any new code or refactoring existing code.

#### 5. code-quality-standards.md

**Scope**: All code files (`**/*.{ts,tsx,js,jsx,go}`)

**Content**:

- Code formatting (Prettier, gofmt)
- Linting (ESLint, golangci-lint)
- Type safety (TypeScript, Go types)
- Testing standards (Jest, Go testing, Playwright E2E)
- CI/CD requirements
- Git commit standards
- Performance and security standards

**When Claude uses this**: Before committing code, reviewing PRs, or setting up tooling.

## Key Principles

### 1. KISS (Keep It Simple, Stupid)

**Core tenets**:

- Solve the actual problem without adding extra features
- Wait for 3+ similar uses before creating abstractions
- Delete unused code immediately (no "just in case")
- Every dependency is a liability - minimize them
- Simple is easy to understand, change, test, and debug

### 2. Grafana Standards

**Must follow**:

- Use `@grafana/ui` components (never custom UI)
- Use `@grafana/data` for data frames
- Backend as Go subprocess (gRPC isolation)
- Store secrets in `secureJsonData`
- Sign plugins before distribution
- Test against multiple Grafana versions

### 3. Security First

**Always**:

- Validate input on backend (never trust frontend)
- Sanitize output before rendering (XSS prevention)
- Use user's security context for all operations
- Rate limit per user (token bucket)
- Log with user identity for audit trail
- Never expose secrets or credentials

### 4. Testing Requirements

**Quality gates**:

- Type checking: `npm run typecheck`
- Linting: `npm run lint`
- Unit tests: >70% coverage
- E2E tests: Critical user flows
- Backend tests: >80% coverage
- Build succeeds: Frontend + backend

## Usage by Claude Code

### Automatic Loading

Claude Code automatically loads:

1. **CLAUDE.md** - Main project instructions
2. **rules/\*.md** - All markdown files in rules directory (recursively)

### Memory Hierarchy

**Priority order** (highest to lowest):

1. Enterprise policy (if configured)
2. Project memory (`.claude/CLAUDE.md`)
3. Project rules (`.claude/rules/*.md`)
4. User memory (`~/.claude/CLAUDE.md`)
5. Local project memory (`./CLAUDE.local.md` - gitignored)

### Path Filtering

Rules with `paths` frontmatter only apply when working with matching files:

```markdown
---
paths: 'src/**/*.{ts,tsx}'
---

# Rules here only apply to TypeScript files in src/
```

## Development Workflow Integration

### Before Starting Work

Claude reads:

- Project overview from CLAUDE.md
- Relevant rules based on files being modified
- Documentation from `docs/` folder

### During Development

Claude applies:

- KISS principles from clean-code-principles.md
- Grafana standards from grafana-plugin-standards.md
- Plugin-specific patterns from app-plugin-development.md
- Code quality rules from code-quality-standards.md

### Before Committing

Claude runs (via TodoWrite or explicit request):

```bash
npm run typecheck
npm run lint
npm run test:ci
```

And verifies:

- No TypeScript errors
- No linting issues
- Tests pass
- Code follows KISS principles

### Full CI Pipeline

```bash
./ci-local.sh
```

Runs complete pipeline:

1. Clean install (`npm ci`)
2. Type checking
3. Linting
4. Frontend tests
5. Frontend build
6. Backend tests with coverage
7. Backend build (all platforms)
8. Plugin validation

## Customization

### Adding New Rules

Create new `.md` files in `.claude/rules/`:

```markdown
---
paths: 'src/components/**/*.tsx'
---

# Component-Specific Rules

Your rules here...
```

### Organizing Rules

Use subdirectories for better organization:

```
.claude/rules/
 frontend/
    react-patterns.md
    ui-components.md
 backend/
    go-patterns.md
    api-design.md
 security/
     security-checklist.md
```

All `.md` files are discovered recursively.

### Local Overrides

Create `.claude/CLAUDE.local.md` for personal preferences (gitignored):

```markdown
# My Personal Preferences

- Use verbose logging during development
- Always run tests before commit
```

## Resources

### Official Documentation

- **Claude Code Memory**: https://code.claude.com/docs/en/memory
- **Grafana Plugin Tools**: https://grafana.com/developers/plugin-tools/
- **grafana-llm-app**: https://github.com/grafana/grafana-llm-app

### Project Documentation

- **Features Overview**: `../docs/FEATURES_OVERVIEW.md`
- **API Endpoints**: `../docs/api/ENDPOINTS.md`
- **Architecture**: `../docs/development/architecture.md`
- **CI Pipeline**: `../docs/CI_PIPELINE_SUMMARY.md`

### Community

- **Grafana Community Forum**: https://community.grafana.com/
- **Plugin Development Channel**: Slack #plugin-development
- **GitHub Discussions**: https://github.com/grafana/grafana/discussions

## Maintenance

### Updating Rules

When Grafana releases new versions or standards:

1. Update relevant rule files in `.claude/rules/`
2. Test with Claude Code to verify rules are clear
3. Commit changes to version control

### Rule Review

**Quarterly review checklist**:

- [ ] Are rules still relevant to current codebase?
- [ ] Do rules reflect latest Grafana standards?
- [ ] Are there new patterns that should be documented?
- [ ] Are there outdated practices that should be removed?
- [ ] Do rules align with team practices?

## Contributing

When adding new features or patterns:

1. Document the pattern in appropriate rule file
2. Include code examples
3. Reference official Grafana docs when applicable
4. Keep rules focused and actionable
5. Use YAML frontmatter to scope rules appropriately

## License

This configuration is part of the Zagalin Grafana plugin project.
License: Apache 2.0
