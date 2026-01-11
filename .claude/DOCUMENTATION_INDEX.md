# Zagalin Plugin - Complete Documentation Index

This document provides a comprehensive index of all documentation for the Zagalin Grafana plugin.

## Documentation Structure

```
.claude/                              # AI assistant configuration
 CLAUDE.md                        # Main project instructions (4,000+ lines)
 README.md                        # Claude Code configuration guide
 QUICK_REFERENCE.md              # Quick reference for common tasks
 DOCUMENTATION_INDEX.md          # This file - complete documentation index
 rules/                          # Modular, topic-specific standards
     grafana-plugin-standards.md       # Official Grafana standards (2,400+ lines)
     grafana-llm-integration.md        # LLM integration guide (1,200+ lines)
     app-plugin-development.md         # App plugin guide (1,800+ lines)
     clean-code-principles.md          # KISS methodology (1,400+ lines)
     code-quality-standards.md         # Quality standards (1,600+ lines)
     plugin-maintenance.md             # Updates & compatibility (1,500+ lines)
     e2e-testing.md                    # E2E testing guide (1,300+ lines)

docs/                               # Project documentation
 FEATURES_OVERVIEW.md           # Complete feature inventory
 api/ENDPOINTS.md               # API reference
 development/architecture.md    # System architecture
 CI_PIPELINE_SUMMARY.md        # CI/CD documentation
```

**Total Documentation**: 15,000+ lines covering all aspects of Grafana plugin development

## Quick Navigation

### For New Developers

**Start here**:

1. `.claude/QUICK_REFERENCE.md` - Quick start and common patterns
2. `.claude/CLAUDE.md` - Project overview and architecture
3. `docs/FEATURES_OVERVIEW.md` - What the plugin does
4. `.claude/rules/clean-code-principles.md` - Coding philosophy

**Then explore**:

- `.claude/rules/grafana-plugin-standards.md` - Grafana fundamentals
- `.claude/rules/app-plugin-development.md` - App plugin patterns
- `.claude/rules/code-quality-standards.md` - Quality requirements

### For Specific Tasks

**Building a new feature**:

1. Check `docs/FEATURES_OVERVIEW.md` - Does it already exist?
2. Read `.claude/rules/app-plugin-development.md` - Implementation patterns
3. Follow `.claude/rules/clean-code-principles.md` - KISS methodology
4. Test with `.claude/rules/e2e-testing.md` - Testing guide

**Integrating LLMs**:

1. `.claude/rules/grafana-llm-integration.md` - Complete LLM guide
2. `.claude/rules/app-plugin-development.md` - Section on LLM integration
3. `docs/api/ENDPOINTS.md` - Backend API reference

**Debugging issues**:

1. `.claude/QUICK_REFERENCE.md` - Troubleshooting section
2. `.claude/rules/e2e-testing.md` - E2E debugging
3. Check Grafana logs: `docker logs jorgeancal-zagalin-app`

**Maintaining/updating**:

1. `.claude/rules/plugin-maintenance.md` - Updates and compatibility
2. `.claude/rules/code-quality-standards.md` - CI/CD requirements
3. `docs/CI_PIPELINE_SUMMARY.md` - CI pipeline details

### For AI Assistants

**Claude Code automatically loads**:

- `.claude/CLAUDE.md` - Main instructions
- `.claude/rules/*.md` - All rule files (path-scoped)

**See**:

- `.claude/README.md` - How Claude Code memory works
- `.claude/rules/` - Topic-specific standards with path scoping

## Document Descriptions

### Main Configuration Files

#### `.claude/CLAUDE.md` (4,000+ lines)

**Purpose**: Comprehensive project instructions for Claude Code
**Contains**:

- Project overview and architecture
- Development workflow and commands
- Critical architecture patterns (dual storage, context manager, LLM streaming)
- Security-first development principles
- Query validation details
- Development notes and troubleshooting

**When to read**: Starting work on the project, understanding architecture

#### `.claude/README.md` (800+ lines)

**Purpose**: Overview of Claude Code configuration and rule system
**Contains**:

- Directory structure explanation
- Modular rules system documentation
- Path-scoped rule loading
- Memory hierarchy and priorities
- Development workflow integration

**When to read**: Setting up AI assistants, understanding documentation structure

#### `.claude/QUICK_REFERENCE.md` (600+ lines)

**Purpose**: Quick lookup for common tasks and patterns
**Contains**:

- Development commands cheatsheet
- Common patterns (8 patterns with code)
- Checklists (before commit, push, release)
- Debugging guides
- Security patterns
- Performance tips
- Troubleshooting common issues

**When to read**: Daily development, quick lookups, debugging

### Topic-Specific Standards (`.claude/rules/`)

#### `grafana-plugin-standards.md` (2,400+ lines)

**Path scope**: `**/*.{ts,tsx,go}`
**Purpose**: Official Grafana plugin development standards
**Contains**:

- System requirements and project structure
- Build system standards (npm, Mage)
- Backend plugin architecture (gRPC, subprocesses)
- Data frame standards
- Plugin metadata (plugin.json)
- Security standards (signing, sandboxing)
- Testing requirements
- Distribution standards

**Topics covered**:

- Plugin types (panel, datasource, app)
- Backend communication (HashiCorp Go Plugin System)
- Resource handlers (custom HTTP endpoints)
- Health checks and metrics
- Plugin signing and security

**When to read**: Understanding Grafana fundamentals, architecture decisions

#### `grafana-llm-integration.md` (1,200+ lines)

**Path scope**: `{src/**/*.{ts,tsx},pkg/**/*.go}`
**Purpose**: Complete guide to LLM integration
**Contains**:

- **Official Grafana LLM patterns** (new content)
  - `@grafana/llm` package usage
  - Model Context Protocol (MCP) for agents
  - Streaming with accumulateContent()
  - Error handling and availability checks
- Backend proxy pattern (this plugin's approach)
- Streaming with SSE
- Function calling / tool use
- Authentication and security
- Performance optimization

**Key sections**:

- Official frontend integration (llm.chatCompletions, streaming)
- MCP agent pattern (tools, loops, execution)
- Backend proxy for security
- Provider configuration (OpenAI, Anthropic, Azure)
- Troubleshooting and testing

**When to read**: Implementing LLM features, chat interfaces, agents

#### `app-plugin-development.md` (1,800+ lines)

**Path scope**: `{src/**/*.{ts,tsx},pkg/**/*.go}`
**Purpose**: Comprehensive app plugin development guide
**Contains**:

- What app plugins are and when to use them
- Core capabilities (10 major features)
- Custom pages and navigation
- Backend integration patterns
- Authentication and user context
- Resource handlers (custom HTTP)
- Service accounts for background tasks
- Role-based access control (RBAC)
- Feature toggles
- Error handling patterns
- **User storage** (complete guide with new usePluginUserStorage hook)
- Performance best practices
- Security patterns
- Testing strategies

**Key sections**:

- User storage with `usePluginUserStorage()` hook
- Migration from localStorage
- Custom backend storage alternative
- LLM integration overview
- Provisioning and automation

**When to read**: Building app plugin features, backend resources, RBAC

#### `clean-code-principles.md` (1,400+ lines)

**Path scope**: All files (no specific filter)
**Purpose**: KISS methodology and clean code practices
**Contains**:

- **KISS Principles** (5 core principles)
  - Solve the actual problem
  - Prefer simple solutions
  - Avoid over-engineering
  - Delete unused code
  - Minimal dependencies
- **Clean Code Principles**
  - Meaningful names
  - Functions do one thing
  - Small functions (5-15 lines)
  - Comments explain WHY
  - Error handling at boundaries
- Refactoring guidelines
- Anti-patterns to avoid
- Code review checklist

**Examples**: 20+ before/after code examples

**When to read**: Writing any code, code reviews, refactoring

#### `code-quality-standards.md` (1,600+ lines)

**Path scope**: `**/*.{ts,tsx,js,jsx,go}`
**Purpose**: Code formatting, linting, testing, and CI standards
**Contains**:

- Code formatting (Prettier, gofmt)
- Linting standards (ESLint, golangci-lint)
- Type safety (TypeScript strict mode, Go types)
- Testing standards
  - Frontend (Jest, >70% coverage)
  - Backend (Go testing, >80% coverage)
  - E2E (Playwright)
- Code review standards
- Git commit conventions
- Pre-commit hooks
- CI/CD requirements
- Performance standards
- Security standards (static analysis)

**When to read**: Before committing, setting up tooling, CI/CD

#### `plugin-maintenance.md` (1,500+ lines)

**Path scope**: `**/*.{ts,tsx,js,jsx,go,json}`
**Purpose**: Plugin updates and backwards compatibility
**Contains**:

- Automated updates with `@grafana/create-plugin`
- Continuous automation (GitHub workflows, Dependabot)
- **Backwards compatibility management** (5 strategies)
  - Function availability checks
  - React hook conditionals
  - Component rendering guards
  - Version detection
  - E2E test coverage
- Best practices for compatibility
- Maintenance workflow (weekly, monthly, quarterly)
- Dependency management (frontend & backend)
- Version support policy
- Security updates
- Troubleshooting updates

**When to read**: Updating dependencies, supporting multiple Grafana versions

#### `e2e-testing.md` (1,300+ lines)

**Path scope**: `tests/**/*.spec.ts`
**Purpose**: End-to-end testing with Playwright
**Contains**:

- `@grafana/plugin-e2e` framework overview
- Configuration (playwright.config.ts)
- Test structure and patterns
- Grafana-specific fixtures
- Custom models (page objects)
- Custom expect matchers
- Testing patterns (5 patterns)
- Cross-version testing
- Debugging techniques
- Migration from Cypress
- Best practices and troubleshooting

**When to read**: Writing E2E tests, debugging test failures

## Finding Information

### By Topic

**Architecture & Design**:

- `.claude/CLAUDE.md` → Critical Architecture Patterns
- `docs/development/architecture.md` → System architecture
- `.claude/rules/grafana-plugin-standards.md` → Grafana architecture

**Security**:

- `.claude/CLAUDE.md` → Security-First Development
- `.claude/rules/app-plugin-development.md` → Security Best Practices
- `.claude/rules/code-quality-standards.md` → Security Standards
- `.claude/QUICK_REFERENCE.md` → Security Patterns

**Testing**:

- `.claude/rules/code-quality-standards.md` → Testing standards
- `.claude/rules/e2e-testing.md` → E2E testing complete guide
- `.claude/QUICK_REFERENCE.md` → Testing checklists

**LLM/AI Integration**:

- `.claude/rules/grafana-llm-integration.md` → Complete LLM guide
- `.claude/rules/app-plugin-development.md` → LLM integration section
- `docs/api/ENDPOINTS.md` → `/llm/chat` endpoint

**Performance**:

- `.claude/rules/app-plugin-development.md` → Performance section
- `.claude/rules/code-quality-standards.md` → Performance standards
- `.claude/QUICK_REFERENCE.md` → Performance tips

**Maintenance**:

- `.claude/rules/plugin-maintenance.md` → Complete maintenance guide
- `.claude/rules/code-quality-standards.md` → CI/CD
- `docs/CI_PIPELINE_SUMMARY.md` → CI details

### By File Type

**TypeScript/React files**:

- `.claude/rules/grafana-plugin-standards.md`
- `.claude/rules/app-plugin-development.md`
- `.claude/rules/grafana-llm-integration.md`
- `.claude/rules/code-quality-standards.md`

**Go files**:

- `.claude/rules/grafana-plugin-standards.md`
- `.claude/rules/app-plugin-development.md`
- `.claude/rules/code-quality-standards.md`

**Test files**:

- `.claude/rules/e2e-testing.md` (E2E)
- `.claude/rules/code-quality-standards.md` (unit tests)

**All files**:

- `.claude/rules/clean-code-principles.md`
- `.claude/rules/plugin-maintenance.md`

## Documentation Statistics

### Total Coverage

- **Lines of documentation**: 15,000+
- **Code examples**: 200+
- **Topics covered**: 100+
- **Rule files**: 7
- **Checklists**: 10+

### By Category

- **Grafana standards**: 4,000+ lines
- **LLM integration**: 1,200+ lines
- **App development**: 1,800+ lines
- **Code quality**: 4,400+ lines (clean code + quality standards)
- **Testing**: 1,300+ lines
- **Maintenance**: 1,500+ lines
- **Quick reference**: 600+ lines

## Learning Paths

### Path 1: New Team Member

1. `.claude/QUICK_REFERENCE.md` - Get started quickly
2. `.claude/CLAUDE.md` - Understand the project
3. `docs/FEATURES_OVERVIEW.md` - Know what exists
4. `.claude/rules/clean-code-principles.md` - Learn coding philosophy
5. `.claude/rules/grafana-plugin-standards.md` - Grafana fundamentals
6. Build a simple feature following guides

### Path 2: Frontend Developer

1. `.claude/QUICK_REFERENCE.md` - Common patterns
2. `.claude/rules/grafana-plugin-standards.md` - Frontend section
3. `.claude/rules/app-plugin-development.md` - Pages, components
4. `.claude/rules/grafana-llm-integration.md` - If working on LLM features
5. `.claude/rules/e2e-testing.md` - Testing

### Path 3: Backend Developer

1. `.claude/QUICK_REFERENCE.md` - Backend patterns
2. `.claude/rules/grafana-plugin-standards.md` - Backend architecture
3. `.claude/rules/app-plugin-development.md` - Resource handlers, auth
4. `.claude/CLAUDE.md` - Security pipeline
5. `.claude/rules/code-quality-standards.md` - Go standards

### Path 4: AI/LLM Developer

1. `.claude/rules/grafana-llm-integration.md` - Complete LLM guide
2. `.claude/rules/app-plugin-development.md` - LLM integration section
3. `.claude/CLAUDE.md` - LLM streaming architecture
4. `docs/api/ENDPOINTS.md` - API reference
5. Implement features following patterns

### Path 5: DevOps/SRE

1. `.claude/rules/code-quality-standards.md` - CI/CD
2. `.claude/rules/plugin-maintenance.md` - Updates
3. `docs/CI_PIPELINE_SUMMARY.md` - Pipeline details
4. `.claude/rules/e2e-testing.md` - Cross-version testing
5. Setup automation workflows

## External Resources

### Official Grafana Documentation

- [Plugin Tools](https://grafana.com/developers/plugin-tools/) - Complete guide
- [Key Concepts](https://grafana.com/developers/plugin-tools/key-concepts/) - Fundamentals
- [Tutorials](https://grafana.com/developers/plugin-tools/tutorials/) - Step-by-step guides
- [How-to Guides](https://grafana.com/developers/plugin-tools/how-to-guides/) - Specific tasks
- [E2E Testing](https://grafana.com/developers/plugin-tools/e2e-test-a-plugin/) - Testing guide

### GitHub Repositories

- [grafana-llm-app](https://github.com/grafana/grafana-llm-app) - LLM plugin
- [grafana-plugin-examples](https://github.com/grafana/grafana-plugin-examples) - Examples
- [grafana-plugin-sdk-go](https://github.com/grafana/grafana-plugin-sdk-go) - Go SDK

### Community

- [Grafana Forum](https://community.grafana.com/) - General discussions
- [Plugin Development Forum](https://community.grafana.com/c/plugin-development) - Plugin-specific
- [Slack](https://grafana.slack.com/) - #plugin-development channel
- [GitHub Discussions](https://github.com/grafana/grafana/discussions) - Q&A

## Keeping Documentation Updated

### When to Update

**Quarterly** (every 3 months):

- Review all rule files for accuracy
- Update Grafana version references
- Add new patterns discovered
- Remove deprecated practices

**When Grafana releases**:

- Update `.claude/rules/grafana-plugin-standards.md`
- Update `.claude/rules/plugin-maintenance.md`
- Test and document new features
- Update compatibility matrices

**When adding features**:

- Update `docs/FEATURES_OVERVIEW.md`
- Update `docs/api/ENDPOINTS.md` if API changes
- Add patterns to relevant rule files
- Update `.claude/QUICK_REFERENCE.md` if common

**When fixing bugs**:

- Add troubleshooting entries
- Document workarounds if needed
- Update testing guides if relevant

### Documentation Checklist

When updating documentation:

- [ ] Keep examples current with latest Grafana
- [ ] Verify code examples compile/run
- [ ] Update version numbers
- [ ] Check links are not broken
- [ ] Maintain consistent formatting
- [ ] Update "Last Updated" dates
- [ ] Review against latest Grafana docs

## Contributing to Documentation

### Style Guidelines

**Tone**:

- Clear and concise
- Professional but friendly
- Focus on practical examples
- Explain WHY, not just WHAT

**Structure**:

- Use consistent heading hierarchy
- Include code examples
- Add "DO" and "DON'T" sections
- Provide troubleshooting tips

**Code Examples**:

- Complete, runnable examples
- Include imports and context
- Follow KISS principles
- Add comments for complex logic

### Template for New Rule Files

```markdown
---
paths: '**/*.{ts,tsx}' # Optional: scope to specific files
---

# Title

Brief description of what this document covers.

## Overview

Context and when to use this guide.

## Requirements

Prerequisites and versions.

## Patterns

### Pattern Name

**When to use**: Description

**Example**:
\`\`\`typescript
// Code example
\`\`\`

## Best Practices

**DO**:

- Item

**DON'T**:

- Item

## Resources

Links to official docs and examples.
```

## Summary

This documentation provides **comprehensive coverage** of:

- Grafana plugin development standards
- LLM integration patterns (official + custom)
- App plugin development
- Clean code and KISS principles
- Code quality and testing
- Maintenance and updates
- E2E testing with Playwright

**Total**: 15,000+ lines of documentation with 200+ code examples covering every aspect of Grafana plugin development.

---

**Last Updated**: 2026-01-10
**Documentation Version**: 1.0
**Plugin Version**: 0.0.5
**Grafana Compatibility**: 10.4.0 - 12.0.0+
