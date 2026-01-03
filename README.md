# Zagalin
Zagalin - AI Assistant for Grafana

**Zagalin** is a context-aware AI assistant that brings the power of Large Language Models (LLMs) directly into your Grafana experience. Chat with your metrics, generate queries, and troubleshoot issues using natural language.

<img src="https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/logo.png" alt="Zagalin Logo" width="200"/>

## Features

### Context-Aware Chat
- **Dashboard Context**: Zagalin automatically understands which dashboard and panels you're viewing
- **Time Range Awareness**: Queries are aware of your current time range selection
- **Floating Chat**: Access Zagalin from any dashboard with the floating chat button
- **Full Chat Page**: Dedicated chat interface for longer conversations

### Query Generation
- **Natural Language to PromQL**: "Show me CPU usage over the last hour"
- **Natural Language to LogQL**: "Find errors in my application logs"
- **Query Explanation**: Get detailed explanations of complex queries
- **Best Practices**: Suggestions for query optimization and improvements

### Troubleshooting Assistant
- **Guided Troubleshooting**: Step-by-step guidance for common issues
- **Pattern Recognition**: Identifies common patterns in metrics and logs
- **Root Cause Analysis**: Helps identify the source of problems
- **Actionable Insights**: Provides specific recommendations and next steps

### Panel Analysis
- **Panel Explanation**: Understand what each panel shows and why it matters
- **Data Interpretation**: Get insights from your metrics
- **Trend Analysis**: Identify patterns and anomalies
- **Alert Suggestions**: Recommendations for setting up alerts

### Customization
- **Personality Presets**: Choose from helpful, technical, beginner-friendly, or concise modes
- **Custom Instructions**: Add your own instructions to tailor Zagalin's behavior
- **Temperature Control**: Adjust creativity vs. consistency
- **Token Limits**: Control response length

### Provider-Agnostic LLM Support
Zagalin works with multiple LLM providers through the [Grafana LLM App](https://grafana.com/grafana/plugins/grafana-llm-app/):
- **OpenAI** (GPT-4, GPT-3.5)
- **Azure OpenAI**
- **Anthropic Claude**
- **Grafana's Managed LLM**

## Installation

### Prerequisites
1. Grafana 10.4.0 or later
2. [Grafana LLM App plugin](https://grafana.com/grafana/plugins/grafana-llm-app/) installed and configured

### Install Zagalin

#### Option 1: Grafana CLI
```bash
grafana-cli plugins install jorgeancal-zagalin-app
```

#### Option 2: Docker
```bash
docker run -d \
  -p 3000:3000 \
  -e "GF_INSTALL_PLUGINS=grafana-llm-app,jorgeancal-zagalin-app" \
  --name=grafana \
  grafana/grafana:latest
```

#### Option 3: Manual Installation
1. Download the latest release from [GitHub Releases](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases)
2. Extract to your Grafana plugins directory
3. Restart Grafana

## Getting Started

### 1. Configure LLM Provider
Before using Zagalin, you need to set up the Grafana LLM App:

1. Navigate to **Configuration → Plugins → LLM App**
2. Enable the plugin
3. Configure your preferred LLM provider:
   - Add your API key (OpenAI, Anthropic, etc.)
   - Or configure Azure OpenAI endpoint
   - Or use Grafana's managed LLM service

### 2. Enable Zagalin
1. Go to **Apps → Zagalin**
2. Click **Enable**
3. Configure your preferences in the Configuration tab

### 3. Start Chatting

#### Floating Chat Button
The floating chat button appears automatically when you're viewing a **dashboard** (not on the home page or other Grafana pages).

**When it appears:**
- When viewing any dashboard
- Automatically positioned in the bottom-right corner
- Shows context badge when dashboard context is available

**How to use it:**
1. Click the floating orange chat button
2. A chat panel slides up from the bottom
3. Type your question and press Cmd/Ctrl+Enter or click Send
4. Zagalin will respond with context about your current dashboard
5. Click the X button or press Escape to close

**Pro Tips:**
- The button turns green when dashboard context is active
- You can drag and reposition the chat panel
- Chat history persists during your session
- Use natural language - no special commands needed

#### Other Access Methods
- **Full Chat Page**: Navigate to **Apps → Zagalin → Chat** for a dedicated full-screen experience
- **Ask Panel**: Add the "Ask AI" panel to any dashboard for inline queries

## LLM Backend Modes

Zagalin offers three different ways to connect to LLM services, each optimized for different use cases. Choose the one that best fits your needs.

### Mode Comparison

| Feature | Official Grafana<br>(Default) | Zagalin Backend<br>(Production) | Direct LLM API<br>(Advanced) |
|---------|:---:|:---:|:---:|
| **Service Account Required** | No | Yes | No |
| **Setup Complexity** | Easy | Moderate | Advanced |
| **Rate Limiting** | Yes | Yes | Yes |
| **Query Validation** | Yes | Yes | Yes |
| **Datasource Governance** | Yes | Yes | Yes |
| **Audit Logging** | Yes | Yes | Yes |
| **Requires grafana-llm-app** | Yes | Yes | No |
| **Best For** | Getting started<br>Single user | Production<br>Multiple users | Custom setups<br>No grafana-llm-app |

### 1. Official Grafana (Default)

**Hybrid architecture** that combines the best of both worlds:
- Uses **@grafana/llm** package for LLM calls (no service account needed)
- Uses **Zagalin backend** for queries, security, and storage

**Key Features**:
- **No service account needed** - Works immediately after install
- **Session-based authentication** - Uses your Grafana login automatically
- **Backend security features** - Rate limiting, query validation, audit logging
- **Datasource governance** - Allowlist enforcement, OTel scope checking
- **Persistent storage** - Conversations saved via Grafana User Storage API

**Prerequisites**:
1. Install **grafana-llm-app** plugin from Grafana catalog
2. Configure it with your LLM provider (Administration → Plugins → LLM App)
3. Zagalin backend must be running

**Configuration**:
1. Go to **Apps → Zagalin → Configuration**
2. Select **"Official Grafana (Default)"** card
3. Click **Save**
4. Start chatting!

---

### 2. Zagalin Backend (Production)

**Full backend proxy** mode with complete security pipeline:
- All LLM requests go through **Zagalin backend → grafana-llm-app**
- Requires **service account token** for backend-to-backend authentication

**Key Features**:
- **Complete audit trail** - All LLM calls logged with user context
- **Service account authentication** - Backend-to-backend security
- **Full security pipeline** - Rate limiting, validation, governance
- **Production-ready** - Designed for multi-user environments
- **Centralized control** - Single point for LLM access management

**Prerequisites**:
1. Install **grafana-llm-app** plugin from Grafana catalog
2. Configure it with your LLM provider
3. **Create a Grafana service account** with Admin or Editor role
4. Generate a service account token
5. Provide the token in Zagalin configuration

**Configuration**:
1. Go to **Administration → Service Accounts**
2. Create new service account (e.g., "Zagalin Plugin")
3. Assign **Admin** or **Editor** role
4. Generate a token and copy it
5. Go to **Apps → Zagalin → Configuration**
6. Select **"Zagalin Backend (Production)"** card
7. Paste the service account token
8. Click **Save**

---

### 3. Direct LLM API (Advanced)

**Direct API integration** where Zagalin backend calls LLM providers directly:
- **No grafana-llm-app required** - Direct OpenAI/Anthropic/Azure integration
- Requires **your own API keys** from LLM providers
- Full security features still enabled (rate limiting, validation, governance)

**Key Features**:
- **No grafana-llm-app dependency** - One less plugin to manage
- **Direct API control** - Full control over API parameters
- **Multiple providers** - OpenAI, Anthropic, Azure, custom endpoints
- **Full security pipeline** - All backend features enabled
- **Custom configurations** - Advanced endpoint and model settings

**Prerequisites**:
1. Obtain API keys from your LLM provider:
   - **OpenAI**: https://platform.openai.com/api-keys (+ Organization ID if your key belongs to multiple orgs)
   - **Anthropic**: https://console.anthropic.com/settings/keys
   - **Azure**: Your Azure OpenAI deployment endpoint + key
2. Know your model name (e.g., `gpt-4o-mini`, `claude-3-5-sonnet-20241022`)

**Configuration**:
1. Go to **Apps → Zagalin → Configuration**
2. Select **"Direct LLM API"** card
3. Choose your **LLM Provider**:
   - OpenAI
   - Anthropic
   - Azure OpenAI
4. (OpenAI only) Enter your **Organization ID** if your API key belongs to multiple organizations
5. Enter your **API Key** (stored securely in Grafana's encrypted storage)
6. Configure **Model Name** (e.g., `gpt-4o-mini`)
7. (Optional) Set **Custom Endpoint** for Azure or custom deployments
8. Click **Save**

**Supported Providers**:

| Provider | Model Examples | API Key Format | Additional Fields |
|----------|---------------|----------------|-------------------|
| **OpenAI** | `gpt-4o`, `gpt-4o-mini`, `gpt-3.5-turbo` | `sk-...` | Organization ID (optional) |
| **Anthropic** | `claude-3-5-sonnet-20241022`, `claude-3-opus-20240229` | `sk-ant-...` | - |
| **Azure OpenAI** | Your deployment name | Azure API key + endpoint | Custom endpoint required |

---

### Switching Between Modes

You can change LLM backend modes at any time without losing data:

1. Go to **Apps → Zagalin → Configuration**
2. Select your desired mode card
3. Configure required settings (service account token or API key if needed)
4. Click **Save**
5. Existing conversations are preserved

**Note**: Conversations are stored per-user and persist across mode changes. Switching modes only affects how LLM calls are made, not your conversation history.

---

## Usage Examples

### Generate a PromQL Query
```
User: Show me memory usage above 80% in the last 24 hours
Zagalin: Here's the PromQL query:

  (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)
  / node_memory_MemTotal_bytes * 100 > 80
```

### Explain a Dashboard
```
User: What does this dashboard show?
Zagalin: This dashboard monitors Kubernetes cluster health.
The top panels show CPU and memory usage across nodes,
the middle section displays pod status and restarts,
and the bottom shows network I/O. The red spike in
the CPU panel indicates a resource contention issue
around 14:30.
```

### Troubleshoot an Issue
```
User: My application is throwing 500 errors
Zagalin: Let's investigate:
1. Check your error logs: {logql_query}
2. Look at response times: {promql_query}
3. Verify database connections
4. Check recent deployments

Based on the current metrics, I see increased latency
starting at 15:00. This correlates with...
```

## Configuration

### Personality & Behavior
- **Personality Preset**: Choose how Zagalin communicates
  - Helpful (Recommended): Balanced and practical
  - Technical: For experienced SREs
  - Beginner-Friendly: Patient and educational
  - Concise: Brief and efficient
  - Custom: Write your own instructions

### Skills & Features
Enable or disable specific capabilities:
- **Explain Panel**: Analyze and explain dashboard panels
- **Generate Queries**: Create PromQL/LogQL queries from natural language
- **Troubleshooting**: Structured troubleshooting guidance
- **Vector Search**: Semantic search (requires vector service)
- **Function Calling**: Structured tool execution

### LLM Parameters
- **Temperature**: 0.0 (factual) to 1.0 (creative)
- **Max Tokens**: Control response length (1000-4000 tokens)

### UI Preferences
- Show/hide context badge
- Display token count and cost estimates
- Auto-open chat on dashboard view

## Architecture

### Frontend
- Built with React and TypeScript
- Uses Grafana UI components
- Streaming responses with RxJS
- Context-aware message handling

### Backend
- Go plugin for Grafana
- Integrates with Grafana LLM App
- Secure API key management
- Context extraction from Grafana

### LLM Integration
- Provider-agnostic through Grafana LLM App
- Streaming completions for responsive UX
- Function calling for structured actions
- Vector search for semantic context (optional)

## Screenshots

### Floating Chat Interface
![Floating Chat](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/floating-chat.png)

### Dashboard Context Awareness
![Dashboard Context](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/dashboard-context.png)

### Query Generation
![Query Generation](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/query-generation.png)

### Configuration
![Configuration](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/configuration.png)

## Privacy & Security

Zagalin implements multiple layers of security to protect your data and infrastructure:

### Data Privacy
- **No Data Storage**: Zagalin doesn't store your queries or responses
- **API Keys**: Managed securely through Grafana's secrets management
- **Context Optimization**: Only sends relevant context to minimize data exposure
- **Provider Choice**: Use your own LLM provider for full control
- **User Isolation**: Conversations are stored per-user with access control

### Query Security
- **Query Validation**: Parser-based validation for PromQL, LogQL, and TraceQL queries
  - Prevents injection attacks using official parsers
  - Configurable complexity limits and function allowlists
  - Comprehensive audit logging of validation events
- **Datasource Governance**: Allowlist-based datasource access control
- **Rate Limiting**: Per-user rate limiting to prevent abuse (default: 60 req/min)
- **User Context**: All queries execute with the user's security context and permissions
- **OTel Enforcement**: Automatic scope injection for multi-tenant observability

### Defense in Depth
The plugin implements multiple security layers that work together:
1. User authentication and identity extraction
2. Per-user rate limiting
3. Datasource allowlist validation
4. Query injection prevention and validation
5. OpenTelemetry scope enforcement
6. Query execution with user permissions
7. Comprehensive audit logging

For detailed security documentation, see [`.claude/CLAUDE.md`](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/.claude/CLAUDE.md#security-first-development).

## Version History & Releases

### Current Version: 0.0.2 - "Security & Governance" (December 27, 2025)

This release transforms Zagalin into a production-ready observability assistant with enterprise-grade security controls.

**Highlights**:
- **Query Validation System** - Pattern-based validation for PromQL, LogQL, TraceQL
- **OpenTelemetry Scope Enforcement** - Automatic service/environment labeling
- **Datasource Governance** - Allowlist system for approved datasources
- **Conversation History** - Persistent storage with dual-tier architecture
- **AI Development Tools** - Configurations for Claude, ChatGPT, Copilot, Cursor
- **Privacy-Conscious Logging** - Usage analytics without exposing queries

**Security Pipeline**: Every query now flows through 6 security validation stages
**Configuration**: 25+ new settings (all opt-in, disabled by default)
**Breaking Changes**: None

**Documentation**:
- [Release Notes](docs/releases/v0.0.2.md)
- [Detailed Changelog](CHANGELOG.md#002---2025-12-27)
- [Upgrade Guide](CHANGELOG.md#upgrading-to-002-from-001)

### Previous Versions

#### Version 0.0.1 - "Foundation" (December 24, 2025)
Initial release with core AI assistant capabilities.

**Highlights**:
- Context-aware chat with dashboard, panel, and time range awareness
- Floating chat interface on every dashboard
- Query generation for PromQL, LogQL, TraceQL
- Skills system (explain_panel, generate_query, troubleshoot, analyze_dashboard)
- LLM integration via grafana-llm-app
- Function calling for structured tool execution

**Documentation**:
- [Changelog](CHANGELOG.md#001---2025-12-24)

### Coming Soon: v0.0.3 - "Intelligence & Orchestration" (Expected: January 2026)

Next release will focus on structured investigation workflows:
- Frontend orchestration with planning and step execution
- Artifact management with syntax highlighting
- Smart routing between dashboard questions and investigations
- Enhanced context extraction and summarization
- Conversation export to JSON/Markdown

### Full Documentation

- **Changelog**: [CHANGELOG.md](CHANGELOG.md) - Detailed changes following Keep a Changelog format
- **Version History**: [docs/VERSION_HISTORY.md](docs/VERSION_HISTORY.md) - Comprehensive version overview
- **Release Notes**: [docs/releases/](docs/releases/) - Detailed release narratives
- **GitHub Releases**: [Releases Page](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/releases)

### Version Comparison

| Version | Release Date | Theme | Major Features | Breaking Changes |
|---------|-------------|-------|----------------|------------------|
| **0.0.2** | Dec 27, 2025 | Security & Governance | Query validation, OTel enforcement, datasource governance, conversation history | None |
| 0.0.1 | Dec 24, 2025 | Foundation | Context-aware chat, floating UI, query generation, skills system | N/A |

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/LICENSE) file for details.

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/CONTRIBUTING.md) for details on how to submit pull requests, report issues, and contribute to the project.

## Support

- **Documentation**: [GitHub README](https://github.com/jorgesnotebook/jorgeancal-zagalin-app#readme)
- **Issues**: [GitHub Issues](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)

## Acknowledgments

- Built on [Grafana Plugin Tools](https://grafana.com/developers/plugin-tools/)
- Powered by [Grafana LLM App](https://grafana.com/grafana/plugins/grafana-llm-app/)
- Inspired by the Grafana community

---

Made by [Jorge Andreu Calatayud](https://github.com/jorgeancal)
