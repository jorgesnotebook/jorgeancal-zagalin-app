---
paths: '**/*'
---

# Architecture Tour - Zagalin Plugin

A visual walkthrough of how this plugin works, with ASCII diagrams and code pointers.

## High-Level Architecture

```mermaid
graph TB
    subgraph GRAFANA
        subgraph ZagalinPlugin["Zagalin Plugin"]
            Frontend["Frontend<br/>(React/TS)"]
            Backend["Backend<br/>(Go)"]
            Frontend ---|HTTP| Backend

            GrafanaUI["@grafana/ui"]
            PluginSDK["grafana-plugin-sdk-go"]

            Frontend -.uses.-> GrafanaUI
            Backend -.uses.-> PluginSDK
        end

        LLMApp["grafana-llm-app<br/>(LLM Integration)"]
        Backend --> LLMApp
    end

    Providers["LLM Providers<br/>(OpenAI, Anthropic)"]
    LLMApp --> Providers
```

**Key Points**:

- Frontend: React 18 + TypeScript
- Backend: Go 1.21+ (runs as subprocess)
- Communication: HTTP via Grafana Plugin SDK
- LLM: Proxied through grafana-llm-app

## Directory Structure

```
jorgeancal-zagalin-app/

 src/                          # Frontend (React/TypeScript)
    components/               # React components
       App/                  # Main app component
       AppConfig/            # Configuration UI
       FloatingChat/         # Global floating chat button
       AskPanel/             # Dashboard panel component

    pages/                    # Page components
       ChatPage.tsx          # Full-screen chat interface
       ConfigPage/           # Plugin configuration
       AssistantChatPage.tsx # AI assistant chat

    services/                 # Business logic
       conversationStorage.ts    # Dual-tier storage
       contextService.ts         # Extract Grafana context
       assistantService.ts       # LLM API client
       zagalinTools.ts           # Function calling handlers

    hooks/                    # Custom React hooks
    types/                    # TypeScript type definitions
    module.tsx                # Plugin entry point

 pkg/                          # Backend (Go)
    main.go                   # Binary entry point
    plugin/
        app.go                # Main plugin app
        resources.go          # HTTP route handlers
        storage.go            # Conversation storage
        guardrails.go         # Rate limiting
        assistant.go          # LLM chat endpoint
        assistant_prompts.go  # System prompts (secure)
        assistant_tools.go    # Function calling tools
        llm_client.go         # grafana-llm-app client
        query_proxy.go        # Query security pipeline
        query_validation.go   # Query injection prevention
        datasource.go         # Datasource type detection
        otel_enforcement.go   # OTel scope enforcement
        context/              # Context extraction
            manager.go        # Context manager
            metrics.go        # Prometheus integration
            logs.go           # Loki integration
            traces.go         # Tempo integration

 tests/                        # E2E tests (Playwright)
    appNavigation.spec.ts
    appConfig.spec.ts

 docs/                         # Documentation
    FEATURES_OVERVIEW.md
    api/ENDPOINTS.md
    development/architecture.md

 .claude/                      # AI assistant configuration
     CLAUDE.md
     QUICK_START.md
     DECISION_TREES.md
     rules/                    # Modular standards
```

**Code Volume**:

- Frontend: ~3,725 lines (services)
- Backend: ~7,887 lines
- Total: 11,600+ lines of code

## Request Flow Diagrams

### 1. User Sends Chat Message

```mermaid
sequenceDiagram
    actor User
    participant ChatPage as ChatPage.tsx<br/>(React)
    participant AssistantService as assistantService.ts<br/>(HTTP client)
    participant Resources as Backend (Go)<br/>resources.go
    participant Assistant as assistant.go<br/>(LLM orchestration)
    participant LLMClient as llm_client.go<br/>(grafana-llm-app client)
    participant LLMApp as grafana-llm-app<br/>(LLM proxy)
    participant LLM as LLM Provider<br/>(OpenAI, Anthropic)

    User->>ChatPage: types message
    ChatPage->>AssistantService: sendMessage()
    AssistantService->>Resources: POST /llm/chat
    Resources->>Assistant: handleLLMChat()
    Note over Assistant: Extract user<br/>Build prompt<br/>Inject context<br/>Auto-detect skills
    Assistant->>LLMClient: ChatStream()
    LLMClient->>LLMApp: HTTP POST
    LLMApp->>LLM: Request
    LLM-->>LLMApp: SSE Stream
    LLMApp-->>LLMClient: Stream chunks
    LLMClient-->>Assistant: Stream chunks
    Assistant-->>Resources: Stream chunks
    Resources-->>AssistantService: Stream chunks
    AssistantService-->>ChatPage: Stream chunks
    ChatPage-->>User: Display streaming response
```

**Files**:

- Frontend: `src/pages/ChatPage.tsx:45-78`
- Service: `src/services/assistantService.ts:89-120`
- Backend: `pkg/plugin/assistant.go:123-234`
- LLM Client: `pkg/plugin/llm_client.go:67-145`

### 2. Query Execution with Security Pipeline

```mermaid
graph TD
    User[User executes query] --> Frontend[Frontend<br/>Sends query]
    Frontend -->|POST /query| Pipeline[query_proxy.go<br/>Security pipeline]

    Pipeline --> Step1[1. Extract User Identity<br/>extractUserIdentity]
    Step1 --> Step2[2. Rate Limiting<br/>guardrails.go<br/>60 req/min]
    Step2 --> Step3[3. Datasource Allowlist<br/>settings.go<br/>Check if allowed]
    Step3 --> Step4[4. Query Validation<br/>query_validation.go<br/>- PromQL/LogQL/TraceQL<br/>- Complexity check<br/>- Function allowlist]
    Step4 --> Step5[5. OTel Scope Enforcement<br/>otel_enforcement.go<br/>if enabled]
    Step5 --> Step6[6. Query Execution<br/>Forward to Grafana<br/>with user context]
    Step6 --> Step7[7. Audit Logging<br/>Log query + user]

    Step7 --> Datasource[Grafana Datasource<br/>Prometheus/Loki<br/>Execute with user permissions]
```

**Security Layers**:

1. **Rate Limiting**: 60 req/min per user (token bucket)
2. **Allowlist**: Only approved datasources
3. **Validation**: Injection prevention, complexity check
4. **OTel**: Scope enforcement (if enabled)
5. **Audit**: Full logging with user identity

**Files**:

- Pipeline: `pkg/plugin/query_proxy.go:156-289`
- Validation: `pkg/plugin/query_validation.go:45-450`
- Rate Limiting: `pkg/plugin/guardrails.go:78-134`

### 3. Context Extraction & Caching

```mermaid
graph TD
    Manager[Context Manager<br/>manager.go<br/>Background service]

    Manager --> Startup[Startup:<br/>- Initialize<br/>- Start refresh goroutine]
    Startup -->|Every N minutes<br/>configurable| Refresh[Refresh Context]

    Refresh --> Metrics[1. metrics.go<br/>Extract Prometheus<br/>- Metric names<br/>- Label values]
    Refresh --> Logs[2. logs.go<br/>Extract Loki<br/>- Stream labels]
    Refresh --> Traces[3. traces.go<br/>Extract Tempo<br/>- Trace metadata]

    Metrics --> Cache[Cache in Memory<br/>sync.RWMutex<br/>Thread-safe]
    Logs --> Cache
    Traces --> Cache

    Cache --> Status[Endpoint: /context/status<br/>Get context status]
    Cache --> ManualRefresh[Endpoint: /context/refresh<br/>Force refresh]
```

**Purpose**: Reduce LLM token usage by pre-extracting context

**Files**:

- Manager: `pkg/plugin/context/manager.go:34-189`
- Prometheus: `pkg/plugin/context/metrics.go:23-145`
- Loki: `pkg/plugin/context/logs.go:19-98`
- Tempo: `pkg/plugin/context/traces.go:21-87`

### 4. Dual-Tier Storage System

```mermaid
graph TD
    User[User Action<br/>Save/load conversation]
    User --> Storage[conversationStorage.ts<br/>Storage facade]
    Storage --> Check{Backend<br/>available?}

    Check -->|YES| ApiClient[storageApiClient.ts<br/>Backend API]
    ApiClient --> BackendAPI[POST /storage/conversations]
    BackendAPI --> BackendStorage[Backend storage.go<br/>File storage]
    BackendStorage --> FileSystem[$GF_PLUGIN_APP_DATA_PATH/<br/>users/userId/<br/>conversations/]

    Check -->|NO| LocalStorage[localStorage<br/>Browser storage<br/>Key: zagalin-conversations]
```

**Fallback Strategy**:

1. Try backend storage (preferred)
2. If unavailable, use localStorage
3. Auto-migrate localStorage → backend when available

**Files**:

- Facade: `src/services/conversationStorage.ts:23-167`
- API Client: `src/services/storageApiClient.ts:19-89`
- Backend: `pkg/plugin/storage.go:34-234`

## Frontend Architecture

### Component Hierarchy

```mermaid
graph TD
    Module[module.tsx<br/>Plugin Entry]
    Module --> App[App.tsx<br/>Main App]
    Module --> Portal[globalChatMount.tsx<br/>Portal]

    App --> Config[ConfigPage/<br/>AppConfig.tsx]
    App --> Chat[ChatPage.tsx<br/>Full Chat]
    App --> Assistant[AssistantChatPage.tsx<br/>AI Assistant]

    Portal --> FloatingBtn[FloatingChatButton.tsx<br/>Global]
    FloatingBtn --> FloatingChat[FloatingChat.tsx<br/>Overlay]
```

**Key Pattern**: **Portal Mounting**

- `globalChatMount.tsx` creates global div
- Mounts once when module loads
- Persists across Grafana navigation
- Only shows on dashboard pages

**File**: `src/globalChatMount.tsx:12-45`

### State Management

```mermaid
graph TD
    Component[Component State<br/>useState]
    Component --> Services[Services<br/>Business Logic]
    Services --> API[API Calls<br/>Backend]
    API --> Handlers[Backend Handlers]
```

**No Redux/MobX** - Simple useState + services pattern

### Services Layer

```mermaid
graph LR
    Services[services/]
    Services --> Assistant[assistantService.ts<br/>LLM chat]
    Services --> Storage[conversationStorage.ts<br/>Storage facade]
    Services --> Context[contextService.ts<br/>Extract Grafana context]
    Services --> Tools[zagalinTools.ts<br/>Function calling]
```

**Purpose**: Separate business logic from UI

## Backend Architecture

### Main Components

```mermaid
graph TD
    App[app.go<br/>Main Plugin]
    App --> Resources[resources.go<br/>HTTP Router]
    App --> Guardrails[guardrails.go<br/>Rate Limiting]
    App --> Validation[query_validation.go<br/>Security]
    App --> ContextBg[context/<br/>Background Service]

    Resources --> LLM[/llm/chat → assistant.go]
    Resources --> Query[/query → query_proxy.go]
    Resources --> Storage[/storage/* → storage.go]
    Resources --> ContextAPI[/context/* → context/manager.go]
    Resources --> Health[/health → health check]

    ContextBg --> Manager[manager.go]
    ContextBg --> Metrics[metrics.go]
    ContextBg --> Logs[logs.go]
    ContextBg --> Traces[traces.go]
```

### HTTP Handler Pattern

```go
// resources.go routes to specific handlers
func (a *App) CallResource(ctx, req, sender) error {
    switch req.Path {
    case "llm/chat":
        return a.handleLLMChat(ctx, req, sender)
    case "query":
        return a.handleQuery(ctx, req, sender)
    // ...
    }
}

// Each handler:
// 1. Extract user identity
// 2. Validate input
// 3. Apply security checks
// 4. Execute operation
// 5. Return response
```

**Pattern**: Resource handler per feature

## Security Architecture

### Multi-Layer Defense

```mermaid
graph TD
    Request[Incoming Request]
    Request --> Layer1[1. Rate Limiting<br/>guardrails.go<br/>Token bucket: 60 req/min per user]
    Layer1 --> Layer2[2. Allowlist<br/>settings.go<br/>Only approved datasources]
    Layer2 --> Layer3[3. Validation<br/>query_validation.go<br/>PromQL/LogQL/TraceQL injection prevention]
    Layer3 --> Layer4[4. OTel Enforcement<br/>otel_enforcement.go<br/>Scope enforcement optional layer]
    Layer4 --> Layer5[5. Audit Logging<br/>All handlers<br/>Full audit trail with user tracking]
    Layer5 --> Execute[Execute Operation]
```

**Defense in Depth**: Multiple independent layers

## Testing Architecture

```mermaid
graph TD
    Testing[Testing Pyramid]

    Testing --> Frontend[Frontend Tests<br/>Jest]
    Testing --> Backend[Backend Tests<br/>Go]
    Testing --> E2E[E2E Tests<br/>Playwright]

    Frontend --> FUnit[Unit: components/**/*.test.tsx<br/>Coverage: >70%]
    Backend --> BUnit[Unit: pkg/**/*_test.go<br/>Coverage: >80%]
    E2E --> E2ESpecs[tests/**/*.spec.ts<br/>Cross-version: 10.4.0 - 12.0.0]
```

**Test Pyramid**:

- Many unit tests (fast, isolated)
- Some E2E tests (slow, integrated)
- Critical paths: 100% coverage

## Build Architecture

### Frontend Build (Webpack)

```mermaid
graph TD
    Source[src/<br/>TypeScript/React]
    Source --> Webpack[webpack<br/>via @grafana/create-plugin]
    Webpack --> TSCompile[TypeScript compilation]
    Webpack --> JSXTransform[React JSX transformation]
    Webpack --> Optimize[Bundle optimization]
    Webpack --> SourceMaps[Source maps]
    Webpack --> Dist[dist/<br/>JavaScript bundle]
    Dist --> Module[module.js<br/>plugin entry]
```

**Build**: `npm run build`
**Dev**: `npm run dev` (watch mode)

### Backend Build (Mage)

```mermaid
graph TD
    Source[pkg/<br/>Go source]
    Source --> Mage[Mage build system]
    Mage --> Linux[Linux build]
    Mage --> Darwin[Darwin build]
    Mage --> Windows[Windows build]
    Linux --> LinuxBin[dist/gpx_zagalin_linux_amd64]
    Darwin --> DarwinBin[dist/gpx_zagalin_darwin_amd64]
    Windows --> WindowsBin[dist/gpx_zagalin_windows_amd64.exe]
```

**Build**: `mage -v buildAll`
**Test**: `mage -v coverage`

## Integration Points

### With Grafana

```mermaid
graph LR
    Plugin[Plugin]
    SDK[Grafana SDK]
    Plugin <--> SDK
    SDK --> Auth[User authentication]
    SDK --> DS[Datasource access]
    SDK --> Nav[Navigation]
    SDK --> UI[UI components]
```

### With grafana-llm-app

```mermaid
graph LR
    Plugin[Plugin]
    LLMApp[grafana-llm-app]
    Providers[LLM Providers<br/>OpenAI, Anthropic, etc.]
    Plugin <--> LLMApp
    LLMApp <--> Providers
    LLMApp --> API[HTTP API]
    LLMApp --> SSE[SSE Streaming]
    LLMApp --> Forward[User auth forwarding]
    LLMApp --> Abstract[Provider abstraction]
```

### With Datasources

```mermaid
graph LR
    Plugin[Plugin]
    Grafana[Grafana]
    DS[Datasources]
    Plugin <--> Grafana
    Grafana <--> DS
    DS --> Prom[Prometheus]
    DS --> Loki[Loki]
    DS --> Tempo[Tempo]
    Grafana --> Perms[User permissions<br/>enforced]
```

## Critical Files by Feature

### LLM Chat

- Frontend: `src/pages/ChatPage.tsx`
- Service: `src/services/assistantService.ts`
- Backend: `pkg/plugin/assistant.go`
- Client: `pkg/plugin/llm_client.go`
- Tools: `pkg/plugin/assistant_tools.go`

### Query Security

- Handler: `pkg/plugin/query_proxy.go`
- Validation: `pkg/plugin/query_validation.go`
- Rate Limit: `pkg/plugin/guardrails.go`
- OTel: `pkg/plugin/otel_enforcement.go`

### Storage

- Frontend: `src/services/conversationStorage.ts`
- Backend: `pkg/plugin/storage.go`

### Context

- Manager: `pkg/plugin/context/manager.go`
- Prometheus: `pkg/plugin/context/metrics.go`
- Loki: `pkg/plugin/context/logs.go`

## Performance Characteristics

**Frontend**:

- Bundle size: <1 MB (gzipped)
- Initial load: <3 seconds
- React hot reload: <1 second

**Backend**:

- Memory: <100 MB per instance
- CPU: <50% average
- Query latency: <200ms (p95)

**LLM**:

- Streaming: Real-time SSE
- Context injection: <100ms overhead
- Token optimization: Context pre-extraction

## Data Flow Summary

```mermaid
graph TD
    User[User]
    User --> Frontend[Frontend]
    Frontend --> Backend[Backend]
    Frontend --> LocalStorage[localStorage<br/>fallback]

    Backend --> LLMApp[grafana-llm-app]
    LLMApp --> LLM[LLM Provider]

    Backend --> Datasources[Datasources<br/>Prometheus, Loki, Tempo]

    Backend --> Storage[Storage<br/>Backend file storage]
```

**Key Patterns**:

- Backend proxy for security
- Dual-tier storage for reliability
- Context caching for performance
- Streaming for UX

---

## Next Steps

**Understand specific areas**:

- Frontend tour: `.claude/rules/00-getting-started/frontend-tour.md`
- Backend tour: `.claude/rules/00-getting-started/backend-tour.md`
- Common tasks: `.claude/rules/00-getting-started/common-tasks.md`

**Deep dive**:

- LLM integration: `.claude/rules/03-integrations/llm-official.md`
- Security pipeline: `.claude/rules/01-grafana-standards/security.md`
- Testing: `.claude/rules/02-development/testing.md`

---

**Last Updated**: 2026-01-10
**Plugin Version**: 0.0.5
**Architecture Version**: 2.0 (hybrid app plugin)
