# OpenAI Tools Configuration

This folder contains instructions for OpenAI-based AI tools (ChatGPT, Codex).

## Files

- **`INSTRUCTIONS.md`** - Complete project context for ChatGPT/Codex

## Usage

### ChatGPT (Web or API)

1. Open `INSTRUCTIONS.md`
2. Copy the entire contents
3. Paste into your ChatGPT conversation at the start
4. ChatGPT will now understand the project context, architecture, and development guidelines

### OpenAI Codex (API)

If using Codex API directly, include `INSTRUCTIONS.md` as system context in your API calls.

## Why This Folder?

Unlike other AI tools (Claude Code, Copilot, Cursor) that automatically load their configuration files, ChatGPT and Codex require manual context loading. This folder provides that context in an easily accessible format.

## Consistency with Other AI Tools

All AI tools in this project follow the same principles:

- **KISS Mindset** - Keep it simple
- **Security-First** - All code secure by default
- **Shared Documentation** - All tools reference `docs/` folder

See `.claude/AI-TOOLS-SETUP.md` for the complete AI tools overview.
