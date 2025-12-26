# Grafana LLM App - Comprehensive Analysis

**Repository:** https://github.com/grafana/grafana-llm-app  
**Purpose:** Official Grafana plugin for LLM integration  
**Architecture:** Monorepo with Frontend (TypeScript) + Backend (Go)

---

## Table of Contents

1. [Overview](#overview)
2. [Frontend Package (@grafana/llm)](#frontend-package-grafanallm)
3. [Backend Package (Go Plugin)](#backend-package-go-plugin)
4. [Architecture & Data Flow](#architecture--data-flow)
5. [Key Features](#key-features)
6. [Development Guide](#development-guide)
7. [Comparison with Zagalin](#comparison-with-zagalin)

---

## Overview

### What is grafana-llm-app?

The **grafana-llm-app** is Grafana's official plugin that provides:
- **Centralized LLM access** - Single authentication and configuration point
- **Multi-provider support** - OpenAI, Azure OpenAI, Anthropic, Grafana native
- **Streaming capabilities** - Real-time chat completions via RxJS Observables
- **Vector search** - Semantic search using embeddings and vector databases
- **Model Context Protocol (MCP)** - Advanced context handling

### Repository Structure

```
grafana-llm-app/
├── packages/
│   ├── grafana-llm-app/        # Backend Go plugin
│   │   └── pkg/
│   │       └── plugin/          # Main plugin code
│   │           ├── app.go
│   │           ├── provider.go
│   │           ├── llm_provider.go
│   │           ├── settings.go
│   │           ├── resources.go
│   │           ├── stream.go
│   │           ├── health.go
│   │           ├── openai_provider.go
│   │           ├── azure_provider.go
│   │           ├── anthropic_provider.go
│   │           ├── grafana_provider.go
│   │           ├── test_provider.go
│   │           └── vector/      # Vector search
│   │               └── service.go
│   └── grafana-llm-frontend/    # Frontend npm package
│       └── src/
│           ├── index.ts         # Main exports
│           ├── llm.ts           # LLM module
│           ├── openai.ts        # OpenAI compatibility
│           ├── vector.ts        # Vector search
│           └── mcp.ts           # Model Context Protocol
└── llmclient/                   # Go client library
```

---

## Frontend Package (@grafana/llm)

### Installation

```json
{
  "dependencies": {
    "@grafana/llm": "^0.22.0"
  }
}
```

### Core Modules

The package exports 4 namespaces:

#### 1. **llm** - Main LLM Module

```typescript
import { llm } from '@grafana/llm';
```

**Key Functions:**

##### Chat Completions (Non-Streaming)
```typescript
async chatCompletions(request: ChatCompletionsRequest): Promise<ChatCompletionsResponse>
```

##### Chat Completions (Streaming)
```typescript
streamChatCompletions(request: ChatCompletionsRequest): Observable<ChatCompletionsDelta>
```

**Types:**

```typescript
interface Message {
  role: 'system' | 'user' | 'assistant' | 'function' | 'tool';
  content: string;
  tool_calls?: ToolCall[];
  function_call?: FunctionCall;
}

interface ChatCompletionsRequest {
  model?: string;              // Model identifier
  messages: Message[];         // Conversation history
  temperature?: number;        // 0-2, creativity control
  max_tokens?: number;        // Response length limit
  tools?: Tool[];             // Available tools/functions
  stream?: boolean;           // Enable streaming
}

interface ChatCompletionsResponse {
  choices: Array<{
    message: Message;
    finish_reason: string;
  }>;
  usage: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
  model: string;
}

interface ChatCompletionsDelta {
  content?: string;          // Text chunk
  tool_calls?: ToolCall[];   // Tool invocation chunks
  function_call?: FunctionCall;
}
```

**RxJS Operators:**

```typescript
// Extract only text content
extractContent(): OperatorFunction<ChatCompletionsDelta, string>

// Accumulate chunks into complete message
accumulateContent(): OperatorFunction<ChatCompletionsDelta, string>

// Reconstruct tool calls from chunks
accumulateToolCalls(): OperatorFunction<ChatCompletionsDelta, Message>
```

##### Health Check
```typescript
async enabled(): Promise<boolean>
async health(): Promise<HealthResponse>
```

**React Hook:**

```typescript
function useLLMStream(options?: {
  timeout?: number;  // Default: 10000ms
}): {
  messages: Message[];
  reply: Message | null;
  status: 'idle' | 'streaming' | 'error' | 'success';
  error: Error | null;
  send: (message: Message) => void;
  reset: () => void;
}
```

#### 2. **openai** - Legacy OpenAI Module

```typescript
import { openai } from '@grafana/llm';
```

**Note:** Deprecated wrapper for backward compatibility. Delegates to `llm` module.

**Key Functions:**

```typescript
async enabled(): Promise<boolean>
```

Checks both legacy and new health response formats:
- Legacy: `details.openAI.configured && details.openAI.ok`
- New: `details.llmProvider.configured && details.llmProvider.ok`

**Migration:** Use `llm` module instead.

#### 3. **vector** - Vector Search Module

```typescript
import { vector } from '@grafana/llm';
```

**Functions:**

##### Search Vector Database
```typescript
async search<T = SearchResultPayload>(
  request: SearchRequest
): Promise<SearchResult<T>[]>

interface SearchRequest {
  collection: string;    // Collection name to search
  query: string;        // Search query text
  topK?: number;       // Max results (default varies)
  filter?: Record<string, any>;  // Metadata filters
}

interface SearchResult<T> {
  payload: T;          // Collection-specific data
  score: number;       // Similarity score (0-1)
}
```

##### Health Check
```typescript
async enabled(): Promise<boolean>
async health(): Promise<VectorHealthResponse>
```

**Use Cases:**
- Semantic search over documentation
- Similar issue/alert lookup
- PromQL query suggestion
- Context retrieval for RAG

#### 4. **mcp** - Model Context Protocol

```typescript
import { mcp } from '@grafana/llm';
```

**Purpose:** Advanced context management for LLM interactions

**Note:** Details not fully documented in fetched sources.

### Frontend Example Usage

```typescript
import { llm } from '@grafana/llm';
import { useState } from 'react';

function ChatComponent() {
  const [input, setInput] = useState('');
  const [response, setResponse] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    setLoading(true);
    
    // Streaming approach
    llm.streamChatCompletions({
      messages: [
        { role: 'system', content: 'You are a helpful assistant' },
        { role: 'user', content: input }
      ],
      model: 'gpt-4',
      temperature: 0.7
    })
    .pipe(
      llm.accumulateContent()
    )
    .subscribe({
      next: (content) => setResponse(content),
      complete: () => setLoading(false),
      error: (err) => {
        console.error(err);
        setLoading(false);
      }
    });
  };

  return (
    <div>
      <input value={input} onChange={(e) => setInput(e.target.value)} />
      <button onClick={handleSubmit} disabled={loading}>
        {loading ? 'Loading...' : 'Send'}
      </button>
      <div>{response}</div>
    </div>
  );
}
```

### Jest Configuration

Required for testing:

```javascript
// jest.config.js
const { grafanaLLMESModules } = require('@grafana/llm');

module.exports = {
  transformIgnorePatterns: [
    ...grafanaLLMESModules,
  ],
};
```

```javascript
// jest-setup.js
import { TransformStream } from 'node:stream/web';
Object.assign(global, { TransformStream });
```

---

## Backend Package (Go Plugin)

### Architecture

The Go plugin follows a modular provider-based architecture:

```
Backend Components:
├── Core Plugin (app.go)
├── Provider Interface (provider.go)
├── Provider Implementations
│   ├── OpenAI Provider
│   ├── Azure Provider
│   ├── Anthropic Provider
│   ├── Grafana Provider
│   └── Test Provider (for testing)
├── Settings Management (settings.go)
├── Resource Handlers (resources.go)
├── Streaming Handler (stream.go)
├── Health Checks (health.go)
└── Vector Service (vector/)
```

### Provider Interface

```go
type LLMProvider interface {
    // Get available models
    Models() []Model
    
    // Non-streaming chat completion
    ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error)
    
    // Streaming chat completion
    ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionChunk, error)
}
```

### Supported Providers

#### 1. **OpenAI Provider** (`openai_provider.go`)
- Standard OpenAI API
- Configurable API endpoint
- Supports all OpenAI models
- API key authentication

#### 2. **Azure OpenAI Provider** (`azure_provider.go`)
- Azure OpenAI Service integration
- Deployment mapping (abstract model → Azure deployment)
- Azure-specific authentication
- Regional endpoint support

#### 3. **Anthropic Provider** (`anthropic_provider.go`)
- Claude models (Claude 3 family)
- Anthropic API compatibility
- Streaming support

#### 4. **Grafana Provider** (`grafana_provider.go`)
- Grafana's native LLM service
- Integrated with Grafana Cloud
- No external API key needed

#### 5. **Test Provider** (`test_provider.go`)
- Mock provider for testing
- Configurable responses
- Development/CI usage

### Core Files

#### `settings.go` (9.5 KB)
- Provider configuration
- API key management
- Validation logic
- Provisioning support

#### `resources.go` (19.7 KB) - Largest file
- HTTP resource handlers
- Request routing
- Response formatting
- Error handling

#### `stream.go` (5.5 KB)
- Streaming implementation
- Grafana Live integration
- Chunk processing
- Real-time delivery

#### `health.go` (6.7 KB)
- Health check endpoints
- Provider status
- Connectivity tests
- Version reporting

### Vector Service (`vector/service.go`)

**Purpose:** Semantic search capabilities

**Features:**
- Embedding generation (OpenAI or Grafana VectorAPI)
- Vector storage (Qdrant or Grafana VectorAPI)
- Similarity search
- Collection management

**Subdirectories:**
- `embed/` - Embedding generation
- `store/` - Vector storage backends

### Adding New Providers

**Backend Steps:**

1. Create provider file: `myprovider_provider.go`
2. Implement `LLMProvider` interface
3. Add settings in `settings.go`
4. Register in `createProvider()` function
5. Add tests

**Example Structure:**

```go
type MyProviderConfig struct {
    APIKey   string
    Endpoint string
}

type MyProvider struct {
    config MyProviderConfig
    client *http.Client
}

func (p *MyProvider) Models() []Model {
    return []Model{
        {ID: "my-model-1", Name: "My Model 1"},
        {ID: "my-model-2", Name: "My Model 2"},
    }
}

func (p *MyProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
    // Implementation
}

func (p *MyProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionChunk, error) {
    // Implementation
}
```

**Frontend Steps:**

1. Add provider type to TypeScript enums
2. Create configuration component
3. Update `LLMConfig.tsx`
4. Add tests

**Reference:** PR #566 (Anthropic support) shows complete implementation

---

## Architecture & Data Flow

### Request Flow

```
Frontend Plugin
    ↓ (HTTP/WebSocket)
grafana-llm-app Plugin (Go)
    ↓ (Provider Selection)
LLM Provider Implementation
    ↓ (API Call)
External LLM Service
    ↓ (Response)
grafana-llm-app Plugin
    ↓ (Streaming via Grafana Live)
Frontend Plugin
```

### Streaming Architecture

1. **Frontend** initiates stream via `streamChatCompletions()`
2. **Backend** receives request at `/resources/chat/completions-stream`
3. **Backend** calls provider's `ChatCompletionStream()`
4. **Provider** streams chunks from external API
5. **Backend** forwards via Grafana Live channel
6. **Frontend** receives Observable stream
7. **Frontend** accumulates using RxJS operators

### Configuration Flow

```yaml
# Provisioning config
apps:
  - type: 'grafana-llm-app'
    jsonData:
      provider: openai
      openAI:
        url: https://api.openai.com/v1
    secureJsonData:
      openAI:
        apiKey: sk-...
```

Stored in Grafana DB → Loaded by plugin → Used for API calls

---

## Key Features

### 1. Multi-Provider Support

**Why it matters:** Single interface for multiple LLM backends

**Benefits:**
- Easy provider switching
- Vendor neutrality
- Centralized configuration

### 2. Streaming Support

**Implementation:** Server-Sent Events via Grafana Live

**Benefits:**
- Real-time responses
- Better UX (progressive display)
- Lower perceived latency

### 3. Type Safety

**TypeScript definitions** for all request/response types

**Benefits:**
- Compile-time validation
- IDE autocomplete
- Reduced runtime errors

### 4. RxJS Integration

**Observable-based streams** with operators

**Benefits:**
- Functional composition
- Easy cancellation
- Backpressure handling

### 5. Vector Search

**Semantic search capabilities** for RAG use cases

**Benefits:**
- Context-aware responses
- Documentation lookup
- Similar content discovery

### 6. Health Monitoring

**Built-in health checks** at multiple levels

**Checks:**
- Plugin enabled
- Provider configured
- API connectivity
- Vector service status

### 7. Secure Configuration

**API keys stored securely** in Grafana's secure JSON data

**Benefits:**
- No plaintext secrets
- Grafana's encryption
- Audit logging

---

## Development Guide

### Setup

```bash
# Requirements
# - Node.js ≥22
# - Go ≥1.21
# - Mage
# - Docker

# Install dependencies
npm install

# Start development
npm run dev          # Frontend watch mode
npm run server       # Start Grafana in Docker

# Access
# http://localhost:3000/plugins/grafana-llm-app
```

### Available Scripts

```bash
# Frontend
npm run build        # Production build
npm run test         # Jest unit tests
npm run lint         # ESLint
npm run format       # Prettier

# Backend
npm run backend:test # Go tests
npm run mage:build   # Build binaries

# E2E
npm run test:e2e     # Playwright tests
```

### Testing

**Unit Tests:**
```bash
npm run test
```

**E2E Tests:**
```bash
npm run test:e2e
# With vector services
COMPOSE_PROFILES=vector npm run server
npm run test:e2e
```

### Code Style

**Prettier:**
- 2 spaces indentation
- Single quotes
- 120 char line length

**ESLint:**
- Auto-fix on commit
- Plugin-specific rules

---

## Comparison with Zagalin

### Architecture Comparison

| Aspect | grafana-llm-app | Zagalin |
|--------|----------------|---------|
| **Purpose** | Generic LLM infrastructure | Context-aware AI assistant |
| **Scope** | Foundation/library | Complete application |
| **Backend** | Multi-provider proxy | Uses grafana-llm-app (should) |
| **Frontend** | Reusable npm package | Custom UI components |
| **Providers** | 4 built-in | Depends on llm-app |
| **Vector Search** | Built-in | Not implemented |
| **MCP** | Supported | Not used |

### What Zagalin Uses from grafana-llm-app

**Current Implementation:**

Zagalin imports from `@grafana/experimental` which likely wraps `@grafana/llm`:

```typescript
import { llms } from '@grafana/experimental';

// Used in ChatPanel.tsx, AskPanel.tsx, etc.
llms.openai.enabled()
llms.openai.streamChatCompletions()
llms.openai.accumulateContent()
```

**Issue:** `@grafana/experimental` is transitioning. Should use `@grafana/llm` directly.

### What Zagalin Could Improve

#### 1. **Direct @grafana/llm Usage**

**Current:**
```typescript
import { llms } from '@grafana/experimental';
```

**Should be:**
```typescript
import { llm } from '@grafana/llm';
```

**Benefits:**
- Official, maintained package
- Better TypeScript support
- Future-proof

#### 2. **Vector Search Integration**

**Missing:** Semantic search for context

**Could add:**
```typescript
import { vector } from '@grafana/llm';

// Search similar dashboards/panels
const results = await vector.search({
  collection: 'dashboards',
  query: userQuestion,
  topK: 5
});

// Use in LLM context
const context = results.map(r => r.payload).join('\n');
```

**Use cases:**
- Find similar issues
- Suggest relevant panels
- Query history lookup

#### 3. **Better Provider Configuration**

**Current:** Hardcoded to expect OpenAI

**Should:**
- Support all providers (OpenAI, Azure, Anthropic, Grafana)
- Check provider via health endpoint
- Adapt UI based on available provider

#### 4. **Tool/Function Calling**

**Missing:** Structured tool invocations

**Could add:**
```typescript
llm.streamChatCompletions({
  messages: [...],
  tools: [
    {
      type: 'function',
      function: {
        name: 'get_panel_data',
        description: 'Get data from a dashboard panel',
        parameters: {
          type: 'object',
          properties: {
            panelId: { type: 'number' },
            dashboardUid: { type: 'string' }
          }
        }
      }
    }
  ]
})
```

**Benefits:**
- Structured LLM-to-app communication
- More reliable actions
- Better error handling

#### 5. **MCP Integration**

**Missing:** Model Context Protocol

**Could enable:**
- Advanced context management
- Multi-turn conversations with state
- Complex reasoning chains

#### 6. **React Hooks**

**Current:** Manual Observable management

**Could use:**
```typescript
import { llm } from '@grafana/llm';

function ChatPanel() {
  const { messages, reply, status, send, reset } = llm.useLLMStream();
  
  // Much simpler!
}
```

#### 7. **Health Monitoring**

**Current:** Limited error handling

**Should add:**
```typescript
// Check before rendering UI
const isReady = await llm.enabled();
if (!isReady) {
  return <ConfigurationRequired />;
}

// Check vector support
const vectorReady = await vector.enabled();
```

#### 8. **Proper Dependency Management**

**Issue:** Using `--legacy-peer-deps` due to `@grafana/experimental`

**Solution:**
1. Migrate to `@grafana/llm` directly
2. Remove `@grafana/experimental` dependency
3. Cleaner dependency tree

---

## Summary

### grafana-llm-app Strengths

✅ **Official Grafana package**  
✅ **Multi-provider support**  
✅ **Production-ready streaming**  
✅ **Vector search built-in**  
✅ **Well-tested and maintained**  
✅ **Type-safe TypeScript API**  
✅ **RxJS-based reactive streams**  

### Zagalin Should Leverage

1. **Migrate from @grafana/experimental to @grafana/llm**
2. **Add vector search for context-aware responses**
3. **Support multiple LLM providers**
4. **Use official React hooks**
5. **Implement tool/function calling**
6. **Add comprehensive health checks**
7. **Consider MCP integration**

### Next Steps for Zagalin

1. Update dependencies to use `@grafana/llm`
2. Implement vector search for dashboard/panel context
3. Add provider-agnostic configuration
4. Refactor to use `useLLMStream()` hook
5. Add tool definitions for structured actions
6. Implement proper health checks and error states

---

**End of Analysis Document**
