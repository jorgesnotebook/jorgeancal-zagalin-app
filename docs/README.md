# Zagalin Documentation

Welcome to the Zagalin documentation. This comprehensive guide covers everything you need to know about developing, testing, and deploying the Zagalin AI Assistant for Grafana.

## Table of Contents

### AI-Assisted Development
- **[ChatGPT Instructions](../.openai/INSTRUCTIONS.md)** - Copy/paste this into ChatGPT for context
- See also: `.claude/CLAUDE.md` (Claude Code), `.github/copilot-instructions.md` (Copilot), `.cursorrules` (Cursor AI)

### Getting Started
- [Quick Start Guide](./getting-started/quick-start.md)
- [Development Environment Setup](./getting-started/development-setup.md)
- [System Requirements](./getting-started/requirements.md)

### Development Guide
- [Plugin Architecture](./development/architecture.md)
- [App Plugin Structure](./development/app-plugin-structure.md)
- [Backend Development](./development/backend.md)
- [Frontend Development](./development/frontend.md)
- [Working with LLM Integration](./development/llm-integration.md)
- [Conversation History System](./development/conversation-history.md)

### Testing
- [Testing Overview](./testing/overview.md)
- [Unit Testing](./testing/unit-tests.md)
- [End-to-End Testing](./testing/e2e-tests.md)
- [CI/CD Pipeline](./testing/ci-cd.md)

### API Reference
- [Configuration API](./api/configuration.md)
- [Context Service](./api/context-service.md)
- [Conversation Storage](./api/conversation-storage.md)
- [LLM Service](./api/llm-service.md)

### Publishing & Deployment
- [Build Process](./publishing/build.md)
- [Plugin Signing](./publishing/signing.md)
- [Publishing to Grafana Catalog](./publishing/catalog-submission.md)
- [Docker Deployment](./publishing/docker.md)

### Contributing
- [Contributing Guidelines](./contributing/guidelines.md)
- [Code Style](./contributing/code-style.md)
- [Pull Request Process](./contributing/pull-requests.md)

### User Guide
- [Installation](./user-guide/installation.md)
- [Configuration](./user-guide/configuration.md)
- [Using Zagalin](./user-guide/usage.md)
- [Troubleshooting](./user-guide/troubleshooting.md)

## About Zagalin

Zagalin is a context-aware AI assistant plugin for Grafana that brings the power of Large Language Models directly into your Grafana experience. It provides:

- 🤖 **Context-Aware Chat** - Understands your dashboards, panels, and time ranges
- 🔍 **Query Generation** - Natural language to PromQL/LogQL conversion
- 🛠️ **Troubleshooting** - Guided assistance for common issues
- 📊 **Panel Analysis** - Insights and explanations for your metrics
- 💬 **Conversation History** - Persistent chat sessions with management
- 🎨 **Customizable** - Personality presets and custom instructions

## Plugin Type

Zagalin is an **App Plugin** - Grafana's most flexible plugin type that allows bundling of:
- Custom pages and routes
- Configuration interfaces
- Backend services
- UI extensions
- Integration with the Grafana LLM App

## Technology Stack

- **Frontend**: React, TypeScript, Emotion CSS
- **Backend**: Go (Grafana plugin SDK)
- **Testing**: Jest, Playwright, @grafana/plugin-e2e
- **LLM Integration**: Grafana LLM App plugin
- **Build Tools**: Webpack, create-plugin CLI

## Quick Links

- [GitHub Repository](https://github.com/jorgesnotebook/jorgeancal-zagalin-app)
- [Grafana Plugin Portal](https://grafana.com/grafana/plugins/)
- [Grafana Community Forum](https://community.grafana.com/)
- [Report Issues](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)

## Support

For questions and support:
- Check the [User Guide](./user-guide/usage.md)
- Review [Troubleshooting](./user-guide/troubleshooting.md)
- Ask on [Grafana Community Forum](https://community.grafana.com/)
- [Open an issue](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](../LICENSE) file for details.
