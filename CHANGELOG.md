# Changelog

All notable changes to the Zagalin - AI Assistant for Grafana plugin will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.2] - Unreleased

### Fixed
- **Build Provenance Attestation**: Added required GitHub Actions permissions (`id-token: write`, `attestations: write`) and enabled `attestation: true` in the release workflow to comply with Grafana's plugin security requirements

## [0.0.1] - Released

### Added

#### Core Features
- **Context-Aware Chat System**: AI assistant that understands your current dashboard, panels, and time range
- **Floating Chat Button**: Accessible from any dashboard with automatic context detection
- **Full Chat Page**: Dedicated page for extended conversations at Apps → Zagalin → Chat
- **Ask Panel**: Custom Grafana panel type for inline AI queries on dashboards

#### Query Generation
- **PromQL Generation**: Natural language to Prometheus query conversion
- **LogQL Generation**: Natural language to Loki query conversion
- **Query Explanation**: Detailed explanations of complex queries
- **Best Practices Suggestions**: Query optimization recommendations

#### Troubleshooting & Analysis
- **Guided Troubleshooting**: Step-by-step issue investigation
- **Pattern Recognition**: Identifies common patterns in metrics and logs
- **Root Cause Analysis**: Helps identify problem sources
- **Panel Explanation**: Analyzes and explains dashboard panels
- **Data Interpretation**: Provides insights from metrics
- **Trend Analysis**: Identifies patterns and anomalies
- **Alert Suggestions**: Recommendations for alert configuration

#### Customization & Configuration
- **Personality Presets**: Multiple communication styles (Helpful, Technical, Beginner-Friendly, Concise, Custom)
- **Custom Instructions**: User-defined behavior customization
- **Temperature Control**: Adjustable creativity vs. consistency (0.0-1.0)
- **Token Limits**: Configurable response length (1000-4000 tokens)
- **UI Preferences**: Context badge visibility, token count display, auto-open settings

#### LLM Integration
- **Provider-Agnostic Support**: Works with OpenAI, Azure OpenAI, Anthropic Claude, and Grafana's Managed LLM
- **Streaming Responses**: Real-time response streaming for better UX
- **Function Calling**: Structured tool execution for complex tasks
- **Vector Search**: Semantic search capabilities (optional)
- **LLM Health Monitoring**: Service health checks and status monitoring

#### Backend Services
- **Context Service**: Extracts dashboard, panel, and time range context
- **Context Optimizer**: Minimizes data exposure by sending only relevant context
- **Grafana Client Integration**: Seamless integration with Grafana APIs for metrics, logs, and traces
- **Action Extractor**: Processes and extracts actionable items from conversations
- **Assistant Skills System**: Modular skill architecture for extensibility
- **Guardrails**: Safety and validation mechanisms

#### Developer Experience
- **TypeScript Support**: Full TypeScript implementation with strict typing
- **React Components**: Modern React 18 with hooks
- **Testing Suite**: Jest unit tests and Playwright E2E tests
- **Webpack Build System**: Optimized production builds with source maps
- **Docker Support**: Docker Compose setup for local development
- **CI/CD Pipeline**: GitHub Actions for testing, linting, and releases
- **ESLint & Prettier**: Code quality and formatting tools

#### Documentation
- **Comprehensive README**: Detailed usage instructions and examples
- **Screenshots**: Visual guides for floating chat, dashboard context, query generation, and configuration
- **Architecture Documentation**: Frontend and backend architecture overview
- **Development Guide**: Setup instructions for contributors

### Security
- **No Data Storage**: Queries and responses are not persisted
- **Secure API Key Management**: Leverages Grafana's secrets management
- **Context Optimization**: Minimizes data sent to LLM providers

### Dependencies
- Grafana 10.4.0+ compatibility
- Requires Grafana LLM App plugin
- Node.js 22+
- Go 1.21+
