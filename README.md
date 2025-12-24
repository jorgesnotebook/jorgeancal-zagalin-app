# Zagalin - AI Assistant for Grafana

**Zagalin** is a context-aware AI assistant that brings the power of Large Language Models (LLMs) directly into your Grafana experience. Chat with your metrics, generate queries, and troubleshoot issues using natural language.

![Zagalin Logo](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/logo.png)

## ✨ Features

### 🤖 Context-Aware Chat
- **Dashboard Context**: Zagalin automatically understands which dashboard and panels you're viewing
- **Time Range Awareness**: Queries are aware of your current time range selection
- **Floating Chat**: Access Zagalin from any dashboard with the floating chat button
- **Full Chat Page**: Dedicated chat interface for longer conversations

### 🔍 Query Generation
- **Natural Language to PromQL**: "Show me CPU usage over the last hour"
- **Natural Language to LogQL**: "Find errors in my application logs"
- **Query Explanation**: Get detailed explanations of complex queries
- **Best Practices**: Suggestions for query optimization and improvements

### 🛠️ Troubleshooting Assistant
- **Guided Troubleshooting**: Step-by-step guidance for common issues
- **Pattern Recognition**: Identifies common patterns in metrics and logs
- **Root Cause Analysis**: Helps identify the source of problems
- **Actionable Insights**: Provides specific recommendations and next steps

### 📊 Panel Analysis
- **Panel Explanation**: Understand what each panel shows and why it matters
- **Data Interpretation**: Get insights from your metrics
- **Trend Analysis**: Identify patterns and anomalies
- **Alert Suggestions**: Recommendations for setting up alerts

### 🎨 Customization
- **Personality Presets**: Choose from helpful, technical, beginner-friendly, or concise modes
- **Custom Instructions**: Add your own instructions to tailor Zagalin's behavior
- **Temperature Control**: Adjust creativity vs. consistency
- **Token Limits**: Control response length

### 🔌 Provider-Agnostic LLM Support
Zagalin works with multiple LLM providers through the [Grafana LLM App](https://grafana.com/grafana/plugins/grafana-llm-app/):
- **OpenAI** (GPT-4, GPT-3.5)
- **Azure OpenAI**
- **Anthropic Claude**
- **Grafana's Managed LLM**

## 📦 Installation

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

## 🚀 Getting Started

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

#### Floating Chat Button 🔵
The floating chat button appears automatically when you're viewing a **dashboard** (not on the home page or other Grafana pages).

**When it appears:**
- ✅ When viewing any dashboard
- ✅ Automatically positioned in the bottom-right corner
- ✅ Shows context badge when dashboard context is available

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

## 💬 Usage Examples

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

## 🔧 Configuration

### Personality & Behavior
- **Personality Preset**: Choose how Zagalin communicates
  - Helpful (Recommended): Balanced and practical
  - Technical: For experienced SREs
  - Beginner-Friendly: Patient and educational
  - Concise: Brief and efficient
  - Custom: Write your own instructions

### Skills & Features
Enable or disable specific capabilities:
- ✅ **Explain Panel**: Analyze and explain dashboard panels
- ✅ **Generate Queries**: Create PromQL/LogQL queries from natural language
- ✅ **Troubleshooting**: Structured troubleshooting guidance
- ⚠️ **Vector Search**: Semantic search (requires vector service)
- ✅ **Function Calling**: Structured tool execution

### LLM Parameters
- **Temperature**: 0.0 (factual) to 1.0 (creative)
- **Max Tokens**: Control response length (1000-4000 tokens)

### UI Preferences
- Show/hide context badge
- Display token count and cost estimates
- Auto-open chat on dashboard view

## 🏗️ Architecture

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

## 📸 Screenshots

### Floating Chat Interface
![Floating Chat](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/floating-chat.png)

### Dashboard Context Awareness
![Dashboard Context](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/dashboard-context.png)

### Query Generation
![Query Generation](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/query-generation.png)

### Configuration
![Configuration](https://raw.githubusercontent.com/jorgesnotebook/jorgeancal-zagalin-app/main/src/img/screenshots/configuration.png)

## 🔒 Privacy & Security

- **No Data Storage**: Zagalin doesn't store your queries or responses
- **API Keys**: Managed securely through Grafana's secrets management
- **Context Optimization**: Only sends relevant context to minimize data exposure
- **Provider Choice**: Use your own LLM provider for full control

## 🛠️ Development

### Prerequisites
- Node.js 20+
- Go 1.21+
- Docker (for local testing)

### Local Development
```bash
# Clone the repository
git clone https://github.com/jorgesnotebook/jorgeancal-zagalin-app.git
cd jorgeancal-zagalin-app

# Install dependencies
npm install

# Build frontend
npm run build

# Build backend
mage -v

# Run in development mode
npm run dev

# Start Grafana with plugin
npm run server
```

### Testing
```bash
# Run unit tests
npm run test

# Run E2E tests
npm run e2e

# Run linter
npm run lint
```

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guidelines](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/blob/main/CONTRIBUTING.md) for details on how to submit pull requests, report issues, and contribute to the project.

## 💬 Support

- **Documentation**: [GitHub README](https://github.com/jorgesnotebook/jorgeancal-zagalin-app#readme)
- **Issues**: [GitHub Issues](https://github.com/jorgesnotebook/jorgeancal-zagalin-app/issues)

## 🙏 Acknowledgments

- Built on [Grafana Plugin Tools](https://grafana.com/developers/plugin-tools/)
- Powered by [Grafana LLM App](https://grafana.com/grafana/plugins/grafana-llm-app/)
- Inspired by the Grafana community

---

Made with ❤️ by [Jorge Andreu Calatayud](https://github.com/jorgeancal)
