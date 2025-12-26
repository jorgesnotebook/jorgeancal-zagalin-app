# AI Tools Setup Summary

This document explains the AI-assisted development configuration for this repository.

## File Structure

```
zagalin/
├── .claude/
│   ├── CLAUDE.md              # Claude Code instructions (comprehensive)
│   └── settings.local.json    # Claude Code permissions
├── .github/
│   └── copilot-instructions.md # GitHub Copilot instructions
├── .cursorrules               # Cursor AI instructions
└── docs/                      # Shared documentation (referenced by all AI tools)
    ├── development/
    ├── api/
    ├── testing/
    └── ...
```

## AI Tools Supported

### 1. Claude Code (claude.ai/code)
- **Config**: `.claude/CLAUDE.md` + `.claude/settings.local.json`
- **Features**: Comprehensive development guide with KISS mindset and security-first principles
- **Documentation reference**: `docs/`

### 2. GitHub Copilot
- **Config**: `.github/copilot-instructions.md`
- **Features**: Code generation guidance with security and simplicity focus
- **Documentation reference**: `docs/`

### 3. Cursor AI
- **Config**: `.cursorrules`
- **Features**: Concise rules for code completion and chat
- **Documentation reference**: `docs/`

## Key Principles (All Tools)

### KISS Mindset
- Prefer simple solutions over complex abstractions
- Wait for 3+ uses before abstracting
- Delete unused code immediately
- No premature optimization

### Security-First
- All queries through backend with user context
- Never bypass Grafana permissions
- Validate all input on backend
- Sanitize LLM output (XSS prevention)
- Rate limiting per user
- Audit logging with user identity

### Code Quality
- TypeScript: Explicit types for APIs, inference for locals
- React: Functional components with hooks
- Go: Return errors, structured logging, context usage
- Testing: Unit tests + E2E tests for all features

## Shared Documentation

All AI tools reference the same `docs/` folder to avoid duplication:
- **Architecture**: `docs/development/architecture.md`
- **API Reference**: `docs/api/`
- **Testing**: `docs/testing/overview.md`
- **CI/CD**: `docs/CI_PIPELINE_SUMMARY.md`

## Recent Implementations

- **Issue #16**: Backend Plugin & Identity Context
  - All queries now route through backend
  - User identity extracted and logged
  - Rate limiting per user
  - Security-first implementation

## Usage

Each AI tool automatically reads its configuration file:
- **Claude Code**: Reads `.claude/CLAUDE.md` on startup
- **GitHub Copilot**: Reads `.github/copilot-instructions.md`
- **Cursor AI**: Reads `.cursorrules`

All tools can reference `docs/` for detailed documentation.

## Maintenance

When updating documentation:
1. Update `docs/` folder (shared by all tools)
2. Update AI-specific instructions only if behavior needs to change
3. Keep security guidelines up-to-date in all AI configs
4. Document new patterns and architecture changes

## Benefits

✅ **No duplication**: Single source of truth in `docs/`
✅ **Consistent guidance**: All AI tools follow same principles
✅ **Security-first**: Built into every AI tool's instructions
✅ **KISS mindset**: Prevents over-engineering across all tools
✅ **Better code quality**: AI-generated code follows project standards
