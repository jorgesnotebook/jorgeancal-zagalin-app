# Contributing to Zagalin

Thank you for your interest in contributing to Zagalin! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Be respectful and constructive in all interactions. We're here to build something useful together.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Grafana version and plugin version
- Any relevant logs or screenshots

### Suggesting Features

Feature requests are welcome! Please:
- Check if the feature has already been requested
- Describe the use case and benefits
- Provide examples of how it would work

### Pull Requests

1. **Fork the repository** and create a new branch from `main`
2. **Make your changes** following our coding standards
3. **Test your changes** thoroughly
4. **Update documentation** if needed
5. **Submit a pull request** with a clear description

## Development Setup

### Prerequisites

- Node.js 22+ (required by package.json)
- Go 1.24.6+ (required by go.mod)
- Docker (for local Grafana testing)

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/jorgeancal-zagalin-app.git
cd jorgeancal-zagalin-app

# Install dependencies
npm install

# Build frontend
npm run build

# Build backend (if you have Go installed)
mage -v buildAll

# Run development server
npm run dev
```

### Running Tests

```bash
# Type checking
npm run typecheck

# Linting
npm run lint

# Unit tests
npm run test

# E2E tests
npm run e2e
```

### Local Testing with Grafana

```bash
# Start Grafana with the plugin
npm run server

# Access at http://localhost:3000
# Default credentials: admin/admin
```

## Coding Standards

### TypeScript/React

- Use TypeScript for all new code
- Follow existing code style (enforced by ESLint)
- Use functional components with hooks
- Keep components small and focused
- Add types for all props and functions

### Go

- Follow standard Go conventions
- Run `go fmt` before committing
- Add tests for new backend features
- Keep functions small and well-documented

### Commits

- Use clear, descriptive commit messages
- Follow conventional commits format:
  - `feat: add new feature`
  - `fix: resolve bug`
  - `docs: update documentation`
  - `test: add tests`
  - `refactor: code improvements`
  - `chore: maintenance tasks`

## Project Structure

```
jorgeancal-zagalin-app/
├── src/                    # Frontend source
│   ├── components/         # React components
│   ├── services/           # API services
│   ├── types/              # TypeScript types
│   └── utils/              # Utility functions
├── pkg/                    # Go backend
│   ├── plugin/             # Plugin implementation
│   └── resources/          # HTTP handlers
├── tests/                  # E2E tests
└── .config/                # Build configuration
```

## Testing Guidelines

- Write tests for new features
- Ensure existing tests pass
- Test with multiple Grafana versions if possible
- Test with the Grafana LLM App integration

## Documentation

- Update README.md for user-facing changes
- Add JSDoc comments for complex functions
- Update CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com/)

## Release Process

Releases are handled by maintainers:
1. Version bump in package.json and plugin.json
2. Update CHANGELOG.md
3. Create git tag (e.g., `v1.0.0`)
4. GitHub Actions builds and creates release

## Questions?

- Open an issue for questions
- Check existing issues and pull requests
- Review the [Grafana Plugin Development Guide](https://grafana.com/developers/plugin-tools/)

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
