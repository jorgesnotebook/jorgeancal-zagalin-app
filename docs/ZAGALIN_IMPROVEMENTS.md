# Zagalin Plugin - Recommended Improvements

**Based on:** grafana-llm-app analysis  
**Date:** 2025-12-24

---

## Executive Summary

After analyzing both the official **grafana-llm-app** (Grafana's LLM infrastructure) and your **Zagalin** plugin, I've identified **8 key improvements** that will make Zagalin more robust, maintainable, and feature-rich.

**Priority Levels:**
- 🔴 **CRITICAL** - Must fix (breaking changes, deprecated APIs)
- 🟡 **HIGH** - Should add (major features, better UX)
- 🟢 **MEDIUM** - Nice to have (improvements, optimizations)

---

## Current State Analysis

### What Zagalin Does Well ✅

1. **Good UI/UX** - Floating chat panel, context awareness badges
2. **Dashboard Context Integration** - Captures dashboard/panel context
3. **Custom Configuration** - Personality presets, configurable behavior
4. **Action Extraction** - Detects and renders actionable items
5. **Grafana Context Service** - Well-structured context gathering

### Critical Issues Found 🔴

#### 1. Using Deprecated `@grafana/experimental`

**Current Code:**
```typescript
import { llms } from '@grafana/experimental';

llms.openai.enabled()
llms.openai.streamChatCompletions()
```

**Issues:**
- `@grafana/experimental` is deprecated
- Causes peer dependency conflicts (`--legacy-peer-deps` needed)
- Will break in future Grafana versions
- Limited TypeScript support

**Impact:** HIGH RISK - Plugin may break with Grafana updates

---

## Recommended Improvements

### 🔴 CRITICAL PRIORITY

#### 1. Migrate to `@grafana/llm` Package

**Why:** Current dependency is deprecated and causes conflicts

**Changes Required:**

**Step 1: Update package.json**
```json
{
  "dependencies": {
    "@grafana/llm": "^1.0.1",    // ADD THIS
    // REMOVE: "@grafana/experimental"
  }
}
```

**Step 2: Update all imports**

**Before:**
```typescript
import { llms } from '@grafana/experimental';
llms.openai.enabled()
llms.openai.streamChatCompletions()
llms.openai.accumulateContent()
```

**After:**
```typescript
import { llm } from '@grafana/llm';
llm.enabled()
llm.streamChatCompletions()
llm.accumulateContent()
```

**Files to Update:**
- `src/components/FloatingChat/ChatPanel.tsx`
- `src/components/AskPanel/AskPanel.tsx`
- `src/pages/AssistantChatPage.tsx`
- `src/globalChatMount.tsx`

**Step 3: Update Jest config**

Add to `.config/jest-setup.js`:
```javascript
import { TransformStream } from 'node:stream/web';
Object.assign(global, { TransformStream });
```

Update `.config/jest.config.js`:
```javascript
const { grafanaLLMESModules } = require('@grafana/llm');

module.exports = {
  // ... existing config
  transformIgnorePatterns: [
    ...grafanaLLMESModules,
  ],
};
```

**Benefits:**
- ✅ No more `--legacy-peer-deps`
- ✅ Official, maintained package
- ✅ Better TypeScript support
- ✅ Future-proof

**Estimated Effort:** 2-3 hours

---

#### 2. Provider-Agnostic Implementation

**Current:** Hardcoded to OpenAI

**Issue:** Won't work with Azure, Anthropic, or Grafana providers

**Solution:**

**Update health check in ChatPanel:**

**Before:**
```typescript
const checkLLMAvailability = async () => {
  try {
    const enabled = await llms.openai.enabled();
    if (!enabled) {
      setError('OpenAI is not configured...');
    }
  } catch (err) {
    setError('Failed to check OpenAI availability');
  }
};
```

**After:**
```typescript
import { llm } from '@grafana/llm';

const checkLLMAvailability = async () => {
  try {
    const enabled = await llm.enabled();
    const health = await llm.health();
    
    if (!enabled) {
      setError('LLM provider is not configured. Please configure grafana-llm-app plugin.');
      return;
    }
    
    // Optional: Log which provider is active
    console.log('LLM Provider:', health.details?.llmProvider);
  } catch (err) {
    setError('Failed to check LLM availability');
  }
};
```

**Update configuration UI:**

Currently shows "OpenAI" everywhere. Update to show actual provider:

```typescript
// src/components/AppConfig/AppConfig.tsx
import { llm } from '@grafana/llm';

export function AppConfig({ plugin }: PluginConfigPageProps) {
  const [providerInfo, setProviderInfo] = useState<string>('');
  
  useEffect(() => {
    llm.health().then(health => {
      const provider = health.details?.llmProvider?.provider || 'Unknown';
      setProviderInfo(`Connected to: ${provider}`);
    });
  }, []);
  
  return (
    <div>
      <Alert severity="info">
        {providerInfo || 'Configure grafana-llm-app plugin to enable Zagalin'}
      </Alert>
      {/* Rest of config */}
    </div>
  );
}
```

**Benefits:**
- ✅ Works with all LLM providers
- ✅ Better error messages
- ✅ User-friendly

**Estimated Effort:** 3-4 hours

---

### 🟡 HIGH PRIORITY

#### 3. Use Official React Hook (`useLLMStream`)

**Current:** Manual Observable management with lots of boilerplate

**Current Code (ChatPanel.tsx ~120 lines):**
```typescript
const [messages, setMessages] = useState<Message[]>([]);
const [isStreaming, setIsStreaming] = useState(false);
const [streamingContent, setStreamingContent] = useState('');
const [error, setError] = useState<string | null>(null);

// Manual subscription management
llms.openai.streamChatCompletions({...})
  .pipe(llms.openai.accumulateContent(), finalize(() => {
    // Manual cleanup
  }))
  .subscribe({
    next: (content) => setStreamingContent(content),
    complete: () => {
      // Manual state management
    },
    error: (err) => {
      // Manual error handling
    }
  });
```

**Recommended:** Use official hook

```typescript
import { llm } from '@grafana/llm';

export function ChatPanel() {
  const { messages, reply, status, error, send, reset } = llm.useLLMStream({
    timeout: 30000, // 30 second timeout
  });
  
  const [input, setInput] = useState('');
  const { context, hasContext } = useGrafanaContext();
  const { config } = useZagalinConfig();

  const handleSend = () => {
    if (!input.trim()) return;
    
    // Build system prompt with context
    const systemPrompt = buildSystemPrompt(config, context, hasContext);
    
    send({
      role: 'system',
      content: systemPrompt
    });
    
    send({
      role: 'user',
      content: input.trim()
    });
    
    setInput('');
  };

  return (
    <div>
      {messages.map((msg, i) => (
        <MessageBubble key={i} message={msg} />
      ))}
      
      {status === 'streaming' && reply && (
        <MessageBubble message={reply} streaming />
      )}
      
      {status === 'error' && (
        <Alert severity="error">{error?.message}</Alert>
      )}
      
      <TextArea
        value={input}
        onChange={(e) => setInput(e.currentTarget.value)}
        disabled={status === 'streaming'}
      />
      
      <Button onClick={handleSend} disabled={status === 'streaming'}>
        {status === 'streaming' ? <Spinner /> : 'Send'}
      </Button>
    </div>
  );
}
```

**Benefits:**
- ✅ 50% less code
- ✅ Built-in error handling
- ✅ Automatic cleanup
- ✅ Timeout management
- ✅ Better type safety

**Estimated Effort:** 4-5 hours (refactor existing components)

---

#### 4. Add Vector Search for Context Enhancement

**Missing:** Semantic search capabilities

**Use Cases:**
1. Search similar dashboards/queries
2. Find relevant documentation
3. Historical query lookup
4. Alert similarity search

**Implementation:**

**Step 1: Install vector support**

Requires `grafana-llm-app` with vector services enabled.

**Step 2: Create vector service**

```typescript
// src/services/vectorSearchService.ts
import { vector } from '@grafana/llm';

export interface DashboardPayload {
  uid: string;
  title: string;
  description: string;
  tags: string[];
}

export interface QueryPayload {
  query: string;
  datasource: string;
  dashboard: string;
  frequency: number;
}

export class VectorSearchService {
  async searchSimilarDashboards(query: string, limit = 5): Promise<DashboardPayload[]> {
    try {
      const enabled = await vector.enabled();
      if (!enabled) {
        console.log('Vector search not enabled');
        return [];
      }

      const results = await vector.search<DashboardPayload>({
        collection: 'dashboards',
        query,
        topK: limit
      });

      return results
        .filter(r => r.score > 0.7) // Only high relevance
        .map(r => r.payload);
    } catch (err) {
      console.error('Vector search failed:', err);
      return [];
    }
  }

  async searchSimilarQueries(query: string, limit = 3): Promise<QueryPayload[]> {
    try {
      const enabled = await vector.enabled();
      if (!enabled) return [];

      const results = await vector.search<QueryPayload>({
        collection: 'queries',
        query,
        topK: limit
      });

      return results
        .filter(r => r.score > 0.8)
        .map(r => r.payload);
    } catch (err) {
      console.error('Query search failed:', err);
      return [];
    }
  }
}
```

**Step 3: Integrate into ChatPanel**

```typescript
import { VectorSearchService } from '../../services/vectorSearchService';

export function ChatPanel() {
  const vectorService = new VectorSearchService();
  const { messages, send, status } = llm.useLLMStream();
  
  const handleSendWithContext = async () => {
    // Get user question
    const userQuestion = input.trim();
    
    // Search for similar content
    const [similarDashboards, similarQueries] = await Promise.all([
      vectorService.searchSimilarDashboards(userQuestion),
      vectorService.searchSimilarQueries(userQuestion)
    ]);
    
    // Build enhanced context
    let contextInfo = buildGrafanaContext(context); // Existing
    
    if (similarDashboards.length > 0) {
      contextInfo += `\n\nSimilar dashboards:\n${similarDashboards.map(d => 
        `- ${d.title}: ${d.description}`
      ).join('\n')}`;
    }
    
    if (similarQueries.length > 0) {
      contextInfo += `\n\nFrequent queries:\n${similarQueries.map(q => 
        `- ${q.query} (${q.datasource})`
      ).join('\n')}`;
    }
    
    // Send with enhanced context
    send({
      role: 'system',
      content: getFullSystemPrompt(config) + '\n\n' + contextInfo
    });
    
    send({
      role: 'user',
      content: userQuestion
    });
  };
  
  // ...
}
```

**Benefits:**
- ✅ Context-aware responses
- ✅ Discovers related content
- ✅ Suggests relevant queries
- ✅ Better troubleshooting

**Prerequisites:**
- Vector services must be enabled in `grafana-llm-app`
- Collections must be populated (dashboards, queries)

**Estimated Effort:** 6-8 hours (+ collection setup time)

---

#### 5. Implement Tool/Function Calling

**Current:** Actions extracted via text parsing (brittle)

**Example current code:**
```typescript
// Parses "Query: SELECT * FROM..." from text
const actions = extractActions(content);
```

**Recommended:** Use structured function calling

**Define Tools:**

```typescript
// src/services/zagalinTools.ts
import { Tool } from '@grafana/llm';

export const ZAGALIN_TOOLS: Tool[] = [
  {
    type: 'function',
    function: {
      name: 'navigate_to_dashboard',
      description: 'Navigate to a specific Grafana dashboard',
      parameters: {
        type: 'object',
        properties: {
          dashboardUid: {
            type: 'string',
            description: 'The UID of the dashboard to navigate to'
          },
          panelId: {
            type: 'number',
            description: 'Optional panel ID to focus on'
          }
        },
        required: ['dashboardUid']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'create_promql_query',
      description: 'Generate a PromQL query for Prometheus',
      parameters: {
        type: 'object',
        properties: {
          metric: {
            type: 'string',
            description: 'The metric to query'
          },
          filters: {
            type: 'object',
            description: 'Label filters to apply'
          },
          aggregation: {
            type: 'string',
            enum: ['sum', 'avg', 'min', 'max', 'count'],
            description: 'Aggregation function'
          }
        },
        required: ['metric']
      }
    }
  },
  {
    type: 'function',
    function: {
      name: 'get_panel_data',
      description: 'Retrieve data from a dashboard panel',
      parameters: {
        type: 'object',
        properties: {
          dashboardUid: { type: 'string' },
          panelId: { type: 'number' }
        },
        required: ['dashboardUid', 'panelId']
      }
    }
  }
];
```

**Use in ChatPanel:**

```typescript
import { ZAGALIN_TOOLS } from '../../services/zagalinTools';

export function ChatPanel() {
  const { messages, send, status } = llm.useLLMStream();
  
  const handleSend = () => {
    // Include tools in request
    send({
      role: 'user',
      content: input.trim()
    }, {
      tools: ZAGALIN_TOOLS,
      temperature: config.temperature,
      max_tokens: config.maxTokens
    });
  };
  
  // Handle tool calls in responses
  useEffect(() => {
    const lastMessage = messages[messages.length - 1];
    if (lastMessage?.tool_calls) {
      lastMessage.tool_calls.forEach(toolCall => {
        switch (toolCall.function.name) {
          case 'navigate_to_dashboard':
            const args = JSON.parse(toolCall.function.arguments);
            navigateToDashboard(args.dashboardUid, args.panelId);
            break;
          
          case 'create_promql_query':
            const queryArgs = JSON.parse(toolCall.function.arguments);
            const query = buildPromQLQuery(queryArgs);
            setGeneratedQuery(query);
            break;
          
          case 'get_panel_data':
            const dataArgs = JSON.parse(toolCall.function.arguments);
            fetchPanelData(dataArgs).then(data => {
              // Send data back to LLM
              send({
                role: 'tool',
                tool_call_id: toolCall.id,
                content: JSON.stringify(data)
              });
            });
            break;
        }
      });
    }
  }, [messages]);
  
  // ...
}
```

**Benefits:**
- ✅ Structured, reliable actions
- ✅ LLM chooses when to use tools
- ✅ Better error handling
- ✅ Type-safe parameters

**Estimated Effort:** 8-10 hours

---

### 🟢 MEDIUM PRIORITY

#### 6. Add Comprehensive Health Checks

**Current:** Limited error handling

**Recommended:**

```typescript
// src/services/llmHealthService.ts
import { llm, vector } from '@grafana/llm';

export interface HealthStatus {
  llm: {
    enabled: boolean;
    provider?: string;
    models?: string[];
    error?: string;
  };
  vector: {
    enabled: boolean;
    version?: string;
    error?: string;
  };
}

export async function checkZagalinHealth(): Promise<HealthStatus> {
  const status: HealthStatus = {
    llm: { enabled: false },
    vector: { enabled: false }
  };

  // Check LLM
  try {
    status.llm.enabled = await llm.enabled();
    if (status.llm.enabled) {
      const health = await llm.health();
      status.llm.provider = health.details?.llmProvider?.provider;
      status.llm.models = health.details?.llmProvider?.models;
    }
  } catch (err) {
    status.llm.error = err instanceof Error ? err.message : 'Unknown error';
  }

  // Check Vector
  try {
    status.vector.enabled = await vector.enabled();
    if (status.vector.enabled) {
      const health = await vector.health();
      status.vector.version = health.version;
    }
  } catch (err) {
    status.vector.error = err instanceof Error ? err.message : 'Unknown error';
  }

  return status;
}
```

**Use in Components:**

```typescript
export function ChatPanel() {
  const [healthStatus, setHealthStatus] = useState<HealthStatus | null>(null);
  
  useEffect(() => {
    checkZagalinHealth().then(setHealthStatus);
  }, []);
  
  if (!healthStatus?.llm.enabled) {
    return (
      <Alert severity="warning">
        <p>LLM provider not configured.</p>
        <p>Please install and configure the grafana-llm-app plugin.</p>
        <Button href="/plugins/grafana-llm-app">Configure Now</Button>
      </Alert>
    );
  }
  
  return (
    <div>
      {healthStatus.vector.enabled && (
        <Tooltip content="Vector search enabled - context-aware responses active">
          <Badge text="Enhanced" color="green" />
        </Tooltip>
      )}
      {/* Chat UI */}
    </div>
  );
}
```

**Benefits:**
- ✅ Better UX (clear error states)
- ✅ Guides users to fix issues
- ✅ Shows available features

**Estimated Effort:** 3-4 hours

---

#### 7. Optimize Context Building

**Current:** Sends full context every message

**Issue:** Token waste, slower responses

**Recommended:**

```typescript
// src/services/contextOptimizer.ts

export interface OptimizedContext {
  essential: string;     // Always included
  supplemental: string;  // Include if tokens allow
  metadata: Record<string, any>;
}

export function optimizeContext(
  fullContext: GrafanaContext,
  maxTokens: number = 1000
): OptimizedContext {
  const estimated = estimateTokens(fullContext);
  
  if (estimated <= maxTokens) {
    return {
      essential: formatFullContext(fullContext),
      supplemental: '',
      metadata: fullContext.metadata || {}
    };
  }
  
  // Prioritize
  const essential = {
    dashboard: fullContext.dashboard?.title,
    panel: fullContext.panel?.title,
    timeRange: fullContext.timeRange
  };
  
  const supplemental = {
    dashboardTags: fullContext.dashboard?.tags,
    panelDescription: fullContext.panel?.description,
    queries: fullContext.queries?.map(q => q.expr).slice(0, 3)
  };
  
  return {
    essential: JSON.stringify(essential, null, 2),
    supplemental: JSON.stringify(supplemental, null, 2),
    metadata: fullContext.metadata || {}
  };
}

function estimateTokens(context: any): number {
  const str = JSON.stringify(context);
  return Math.ceil(str.length / 4); // Rough estimate
}
```

**Use smartly:**

```typescript
const handleSend = () => {
  const optimized = optimizeContext(context, config.maxTokens * 0.3); // Use 30% of budget for context
  
  const systemPrompt = `${getFullSystemPrompt(config)}\n\nCurrent Context:\n${optimized.essential}`;
  
  send({
    role: 'system',
    content: systemPrompt
  });
  
  send({
    role: 'user',
    content: input.trim()
  });
};
```

**Benefits:**
- ✅ Lower costs
- ✅ Faster responses
- ✅ More room for conversation

**Estimated Effort:** 2-3 hours

---

#### 8. Add Model Context Protocol (MCP) Support

**What is MCP?**

Model Context Protocol enables:
- Stateful conversations
- Context persistence
- Multi-turn reasoning
- External tool integration

**Basic Implementation:**

```typescript
import { mcp } from '@grafana/llm';

// Preliminary - full API not documented
// Monitor grafana-llm-app releases for MCP documentation
```

**Benefits:**
- ✅ Advanced context handling
- ✅ Better multi-turn conversations
- ✅ State management

**Status:** Monitor for availability

**Estimated Effort:** TBD (awaiting full MCP API documentation)

---

## Implementation Roadmap

### Phase 1: Critical Fixes (1-2 days)
1. ✅ Migrate to `@grafana/llm`
2. ✅ Provider-agnostic implementation
3. ✅ Remove `--legacy-peer-deps` dependency

### Phase 2: Enhanced Features (3-5 days)
4. ✅ Implement `useLLMStream()` hook
5. ✅ Add comprehensive health checks
6. ✅ Optimize context building

### Phase 3: Advanced Features (1-2 weeks)
7. ✅ Vector search integration
8. ✅ Tool/function calling
9. ✅ MCP support (when available)

---

## Testing Checklist

After each phase:

- [ ] Unit tests pass
- [ ] E2E tests pass
- [ ] Works with OpenAI provider
- [ ] Works with Azure provider
- [ ] Works with Anthropic provider
- [ ] Works with Grafana provider
- [ ] Context awareness functional
- [ ] Streaming works correctly
- [ ] Error states handled gracefully
- [ ] No console errors

---

## Migration Guide (Phase 1)

### Step-by-Step

1. **Backup Current State**
   ```bash
   git checkout -b migration/grafana-llm
   ```

2. **Update Dependencies**
   ```bash
   npm uninstall @grafana/experimental
   npm install @grafana/llm@^1.0.1
   ```

3. **Update Imports**
   - Search: `import { llms } from '@grafana/experimental'`
   - Replace: `import { llm } from '@grafana/llm'`

4. **Update API Calls**
   - `llms.openai.enabled()` → `llm.enabled()`
   - `llms.openai.streamChatCompletions()` → `llm.streamChatCompletions()`
   - `llms.openai.accumulateContent()` → `llm.accumulateContent()`

5. **Update Tests**
   - Add TransformStream polyfill
   - Add grafanaLLMESModules to transformIgnorePatterns

6. **Test Thoroughly**
   ```bash
   npm run test:ci
   npm run build
   npm run e2e
   ```

7. **Update Documentation**
   - Update README with new dependency
   - Note provider compatibility

---

## Expected Outcomes

### After Phase 1 ✅
- No more peer dependency warnings
- Works with all LLM providers
- Future-proof codebase
- Cleaner dependency tree

### After Phase 2 ✅
- Simpler, more maintainable code
- Better error handling
- Improved UX
- Optimized token usage

### After Phase 3 ✅
- Context-aware AI (vector search)
- Reliable actions (function calling)
- Advanced conversations (MCP)
- Production-ready features

---

## Conclusion

These improvements will transform Zagalin from a prototype into a production-ready, feature-rich AI assistant for Grafana. The migration to `@grafana/llm` is **critical** and should be done **immediately** to avoid future breaking changes.

**Priority Order:**
1. 🔴 Migrate to `@grafana/llm` (MUST DO)
2. 🔴 Provider-agnostic implementation (MUST DO)
3. 🟡 Use `useLLMStream()` hook (SHOULD DO)
4. 🟡 Add vector search (SHOULD DO)
5. 🟡 Implement tool calling (SHOULD DO)
6. 🟢 Optimize context (NICE TO HAVE)
7. 🟢 Add health checks (NICE TO HAVE)
8. 🟢 MCP support (FUTURE)

**Total Estimated Effort:** 2-4 weeks

---

**Document END**
