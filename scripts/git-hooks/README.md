# Git Hooks

This directory contains git hooks that automatically run checks to catch issues before they reach CI.

## Installed Hooks

### pre-commit (Fast)

Runs on every commit:

- ✅ Type checking
- ✅ Linting

### pre-push (Full CI)

Runs before pushing:

- ✅ Type checking
- ✅ Linting
- ✅ Unit tests
- ✅ Build

## Installation

Hooks are automatically installed when you run `npm install`.

To manually install/reinstall:

```bash
npm run install-hooks
```

## Skipping Hooks

Sometimes you need to skip hooks (e.g., WIP commits):

```bash
# Skip pre-commit hook
git commit --no-verify -m "WIP: work in progress"

# Skip pre-push hook
git push --no-verify
```

**Note:** Use `--no-verify` sparingly. The hooks exist to catch issues early!

## Benefits

- 🚀 Catch issues before CI runs
- ⏱️ Save time - fix issues locally instead of waiting for CI
- 🛡️ No surprises - know your code will pass CI before pushing
- 👥 Team consistency - everyone runs the same checks
