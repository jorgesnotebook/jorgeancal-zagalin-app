#!/bin/bash
# Install git hooks from scripts/git-hooks to .git/hooks

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_DIR="$PROJECT_ROOT/scripts/git-hooks"
GIT_HOOKS_DIR="$PROJECT_ROOT/.git/hooks"

echo "Installing git hooks..."
echo ""

# Check if we're in a git repository
if [ ! -d "$GIT_HOOKS_DIR" ]; then
  echo "❌ Error: Not in a git repository"
  exit 1
fi

# Install each hook
for hook in "$HOOKS_DIR"/*; do
  hook_name=$(basename "$hook")

  # Skip non-hook files (README, etc.)
  if [[ "$hook_name" == "README.md" ]]; then
    continue
  fi

  echo "  Installing $hook_name..."
  cp "$hook" "$GIT_HOOKS_DIR/$hook_name"
  chmod +x "$GIT_HOOKS_DIR/$hook_name"
done

echo ""
echo "✅ Git hooks installed successfully!"
echo ""
echo "Installed hooks:"
echo "  • pre-commit: Type checking + Linting (runs on every commit)"
echo "  • pre-push: Full CI checks (runs before push)"
echo ""
echo "To skip hooks temporarily, use:"
echo "  git commit --no-verify  # Skip pre-commit"
echo "  git push --no-verify    # Skip pre-push"
echo ""
