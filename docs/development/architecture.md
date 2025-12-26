# Zagalin Plugin Architecture

This document describes the architecture and anatomy of the Zagalin plugin, including its components, data flow, and integration points.

## Overview

Zagalin is a **Grafana App Plugin** that provides AI-powered assistance through integration with the Grafana LLM App. It combines frontend UI components, backend services, and external LLM providers to deliver context-aware chat capabilities.

## Plugin Type: App Plugin

### Why App Plugin?

App plugins are Grafana's most comprehensive plugin type, offering:

- **Flexibility**: Can include multiple pages, routes, and components
- **Configuration UI**: Organization-level settings management
- **Backend Support**: Optional backend component for advanced features
- **Bundling**: Can nest other plugin types (panels, data sources)
- **UI Extensions**: Extend Grafana's native UI
- **Navigation**: Custom routes and menu entries

### App Plugin Structure

```
jorgeancal-zagalin-app/
├── src/                          # Frontend source code
│   ├── components/               # React components
│   │   ├── AppConfig/           # Configuration UI
│   │   ├── FloatingChat/        # Main chat interface
│   │   └── AskPanel/            # Panel plugin integration
│   ├── pages/                    # Route pages
│   │   ├── ConfigPage.tsx       # Settings page
│   │   ├── ChatPage.tsx         # Full-page chat
│   │   └── AssistantChatPage.tsx
│   ├── services/                 # Business logic
│   │   ├── contextService.ts    # Grafana context extraction
│   │   ├── conversationStorage.ts # Persistence
│   │   └── llmHealthService.ts  # LLM integration
│   ├── hooks/                    # Custom React hooks
│   ├── types/                    # TypeScript definitions
│   ├── theme/                    # Styling and branding
│   └── module.ts                 # Plugin entry point
├── pkg/                          # Backend (Go)
│   ├── main.go                  # Backend entry point
│   └── plugin/                  # Plugin logic
├── provisioning/                 # Docker configs
│   ├── dashboards/
│   ├── datasources/
│   └── plugins/
├── dist/                         # Build output
├── plugin.json                   # Plugin metadata
├── docker-compose.yml           # Development environment
└── tests/                        # E2E tests
```

## Architecture Layers

### 1. Presentation Layer (Frontend)

#### React Components

**FloatingChatButton** (`src/components/FloatingChat/FloatingChatButton.tsx`)
- Floating chat button on all dashboard pages
- Draggable and resizable panel
- Persists position/size to localStorage
- Portal rendering for global access

**ChatPanel** (`src/components/FloatingChat/ChatPanel.tsx`)
- Main chat interface
- Message display and streaming
- Context-aware badge system
- Input handling
- Integration with useConversation hook

**ConversationListSidebar** (`src/components/FloatingChat/ConversationListSidebar.tsx`)
- Conversation history management
- Search and filter
- Pin/rename/delete operations
- Split-view layout

**AppConfig** (`src/components/AppConfig/AppConfig.tsx`)
- Plugin configuration UI
- Personality presets
- Skill toggles
- LLM parameters

#### Global Mount

**globalChatMount.tsx** (`src/globalChatMount.tsx`)
- Mounts FloatingChatButton globally
- Excludes login/admin pages
- Uses React Portal for rendering
- Route-aware visibility

### 2. Business Logic Layer (Services)

#### Context Management

**ContextService** (`src/services/contextService.ts`)
```typescript
class ContextService {
  static async getContext(): Promise<GrafanaContext>
  static getDashboardContext(uid: string): Promise<DashboardContext>
  static formatContextPrompt(context: GrafanaContext): string
}
```

Extracts and formats:
- Current dashboard metadata
- Active panel information
- Time range selection
- Template variables
- User information

**contextOptimizer.ts** (`src/services/contextOptimizer.ts`)
- Reduces context size to fit token budgets
- Prioritizes essential information
- Drops optional metadata when needed

#### Conversation Persistence

**ConversationStorage** (`src/services/conversationStorage.ts`)
```typescript
class ConversationStorage {
  static getConversationList(): ConversationMetadata[]
  static getConversation(id: string): Conversation | null
  static saveConversation(conversation: Conversation): void
  static deleteConversation(id: string): void
  static updateTitle(id: string, title: string): void
  static togglePin(id: string): void
}
```

Features:
- localStorage-based persistence
- Auto-pruning (max 50 conversations)
- Message trimming (max 100 per conversation)
- Pin support
- Export/import functionality

#### LLM Integration

**LLM Service** (`src/services/llmHealthService.ts`)
- Health checks with caching (30s TTL)
- Integration with @grafana/llm
- Streaming support via RxJS
- Error handling

**Assistant Skills** (`src/services/assistantSkills.ts`)
- Skill detection from user queries
- Context-aware prompts
- Tool calling support

**Action Extractor** (`src/services/actionExtractor.ts`)
- Parses LLM responses for actions
- Creates Explore links
- Handles query generation

### 3. State Management Layer

#### React Hooks

**useConversation** (`src/hooks/useConversation.ts`)
```typescript
function useConversation(): {
  conversations: ConversationMetadata[];
  currentId: string | null;
  messages: StoredMessage[];
  createNew: (context?) => void;
  loadConversation: (id: string) => void;
  addMessage: (message) => void;
  deleteConversation: (id: string) => void;
  updateTitle: (id: string, title: string) => void;
  togglePin: (id: string) => void;
}
```

**useGrafanaContext** (`src/services/useGrafanaContext.ts`)
- Monitors Grafana context changes
- Listens to location changes
- Refreshes context on navigation

**useZagalinConfig** (`src/hooks/useZagalinConfig.ts`)
- Loads plugin configuration
- Polls for updates (30s interval)
- Merges with defaults

### 4. Backend Layer (Optional)

While Zagalin primarily uses frontend functionality, it's structured to support backend components:

#### Go Backend Structure

```go
// pkg/main.go
package main

import (
    "github.com/grafana/grafana-plugin-sdk-go/backend"
)

func main() {
    backend.Manage("jorgeancal-zagalin-app", newDatasourceInstance, backend.ServeOpts{})
}
```

#### Backend Use Cases
- Future: Secure API key storage
- Future: Server-side LLM calls
- Future: Advanced caching
- Future: Custom authentication

## Data Flow

### 1. User Sends Message

```
User Input (ChatPanel)
    ↓
handleSend()
    ↓
conversation.addMessage() → ConversationStorage → localStorage
    ↓
Context Detection (useGrafanaContext)
    ↓
Skill Detection (assistantSkills)
    ↓
LLM Streaming (@grafana/llm)
    ↓
Stream Observable (RxJS)
    ↓
setStreamingContent() → UI Update
    ↓
Complete → conversation.addMessage() → localStorage
```

### 2. Context Extraction

```
Dashboard Navigation
    ↓
locationService.getHistory().listen()
    ↓
ContextService.getContext()
    ↓
Extract Dashboard UID from URL
    ↓
Fetch Dashboard JSON (Grafana API)
    ↓
Parse Panels, Variables, Time Range
    ↓
Cache in useGrafanaContext
    ↓
Format for LLM (formatContextPrompt)
    ↓
Include in System Message
```

### 3. Conversation Management

```
User Action (Pin/Rename/Delete)
    ↓
ConversationListSidebar Event
    ↓
useConversation Hook Method
    ↓
ConversationStorage Static Method
    ↓
Update localStorage
    ↓
refreshList()
    ↓
Re-render UI with Updated List
```

## Integration Points

### Grafana LLM App

Zagalin depends on the Grafana LLM App plugin for LLM functionality:

```typescript
import { llm } from '@grafana/llm';

// Stream chat completions
const stream = llm.streamChatCompletions({
  model: 'gpt-4o-mini',
  messages: [...],
  temperature: 0.7,
  max_tokens: 2000,
  tools: ZAGALIN_TOOLS, // Optional
}).pipe(llm.accumulateContent());
```

### Grafana APIs

**LocationService** - Route monitoring
```typescript
import { locationService } from '@grafana/runtime';

locationService.getHistory().listen((location) => {
  // Detect navigation changes
});
```

**BackendSrv** - API requests
```typescript
import { getBackendSrv } from '@grafana/runtime';

const dashboard = await getBackendSrv().get(`/api/dashboards/uid/${uid}`);
```

**TimeRangeService** - Time range utilities
```typescript
import { getTemplateSrv } from '@grafana/runtime';

const from = getTemplateSrv().replace('$__from');
```

### localStorage

Used for client-side persistence:
- **zagalin-conversations** - Conversation history
- **zagalin-chat-panel-position** - Floating panel position
- **zagalin-chat-panel-size** - Panel dimensions
- **zagalin-config** - Plugin settings

## Performance Considerations

### Optimization Strategies

1. **Lazy Loading**
   - Code splitting for large components
   - Dynamic imports for heavy libraries

2. **Caching**
   - LLM health check cache (30s TTL)
   - Context caching in hooks
   - Memoization of expensive computations

3. **Streaming**
   - Incremental message rendering
   - RxJS backpressure handling
   - Debounced UI updates

4. **Storage Limits**
   - Auto-pruning old conversations
   - Message count limits
   - Efficient JSON serialization

### Bundle Size

Target sizes:
- **module.js**: ~95 KB (main bundle)
- **Vendor chunks**: ~340 KB (React, Grafana UI)
- **Total**: ~460 KB (acceptable for app plugin)

## Security Considerations

### Data Privacy

1. **Local Storage Only** - No cloud sync by default
2. **User Isolation** - Browser-level separation
3. **No Credential Storage** - Uses Grafana LLM App for API keys
4. **Context Optimization** - Minimizes data sent to LLM

### Input Validation

1. **Sanitize User Input** - Prevent injection attacks
2. **Validate Configuration** - Type checking
3. **Rate Limiting** - Prevent abuse (future)

### Content Security

1. **Markdown Sanitization** - HTML content filtering
2. **XSS Prevention** - React's built-in protection
3. **CORS Handling** - Grafana proxy for external requests

## Extensibility

### Adding New Features

1. **New Skills** - Extend `assistantSkills.ts`
2. **New Actions** - Add to `actionExtractor.ts`
3. **New Pages** - Add routes in `module.ts`
4. **New UI Extensions** - Use Grafana's extensionPoint

### Plugin Hooks

Zagalin supports future extensibility through:
- Custom skill plugins
- Action handlers
- UI customization
- Theme overrides

## Testing Architecture

### Unit Tests (Jest)

```
tests/
└── components/
    └── *.test.tsx
```

### E2E Tests (Playwright)

```
tests/
└── e2e/
    └── *.spec.ts
```

### Integration Tests

- LLM integration mocking
- Context extraction validation
- Storage persistence tests

## Deployment Architecture

### Production Build

```
npm run build
    ↓
Webpack Bundle
    ↓
dist/
├── module.js (app logic)
├── module.js.map (sourcemap)
├── plugin.json (metadata)
├── img/ (assets)
└── gpx_zagalin_* (backend binaries)
```

### Docker Deployment

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./dist:/var/lib/grafana/plugins/jorgeancal-zagalin-app
    environment:
      - GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=jorgeancal-zagalin-app
```

## Future Architecture Enhancements

1. **Backend Integration**
   - Secure credential storage
   - Server-side LLM calls
   - Advanced caching strategies

2. **Cloud Sync**
   - Cross-device conversation sync
   - Team collaboration features
   - Backup and restore

3. **Advanced Analytics**
   - Usage metrics
   - Performance monitoring
   - Error tracking

4. **Multi-LLM Support**
   - Provider switching
   - Model selection
   - Cost optimization

## Related Documentation

- [Frontend Development Guide](./frontend.md)
- [Backend Development Guide](./backend.md)
- [Testing Documentation](../testing/overview.md)
- [API Reference](../api/README.md)
