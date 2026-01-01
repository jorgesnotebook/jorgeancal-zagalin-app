/**
 * Assistant Service Router
 * Routes LLM requests based on llmBackend configuration:
 * - "backend-proxy" (default): Zagalin backend with full security
 * - "grafana-llm": Frontend calls via @grafana/llm package
 * - "direct": Backend calls LLM API directly (no grafana-llm-app)
 */

import { Observable } from 'rxjs';
import { streamAssistantChat as streamBackend, type AssistantRequest, type StreamChunk } from './assistantService';
import { streamGrafanaLLM } from './llmDirectClient';
import { getZagalinConfig } from './configHelper';
import { buildReasoningPrompt } from './reasoningParser';

/**
 * Stream assistant chat with automatic routing based on configuration
 *
 * @param request The assistant request
 * @returns Observable stream of response chunks
 */
export function streamAssistantChatRouted(request: AssistantRequest): Observable<StreamChunk> {
  const config = getZagalinConfig();
  const llmBackend = config.llmBackend || 'grafana-llm'; // Default to @grafana/llm (no service account needed)

  switch (llmBackend) {
    case 'grafana-llm':
      // Use @grafana/llm package (frontend direct)
      console.info('[Zagalin] Using @grafana/llm package');
      return streamGrafanaLLMMode(request);

    case 'direct':
      // Backend handles direct LLM calls (no grafana-llm-app)
      console.info('[Zagalin] Using direct LLM mode (backend)');
      return streamBackend(request);

    case 'backend-proxy':
    default:
      return streamBackend(request);
  }
}

/**
 * Use @grafana/llm package for frontend calls
 * Converts AssistantRequest to @grafana/llm format
 */
function streamGrafanaLLMMode(request: AssistantRequest): Observable<StreamChunk> {
  // Build system prompt from context
  const systemPrompt = buildSystemPrompt(request);

  // Build messages array
  const messages = [
    { role: 'system' as const, content: systemPrompt },
    ...request.history.map(msg => ({
      role: msg.role as 'user' | 'assistant' | 'system',
      content: msg.content,
    })),
    { role: 'user' as const, content: request.enrichedMessage || request.message },
  ];

  // Configure LLM request
  const llmRequest = {
    model: 'gpt-4o-mini', // grafana-llm-app will use configured model
    messages,
    temperature: 0.7,
    max_tokens: 2000,
    stream: true,
  };

  // Stream from @grafana/llm and convert response format
  return new Observable<StreamChunk>((subscriber) => {
    streamGrafanaLLM(llmRequest).subscribe({
      next: (llmChunk) => {
        // Convert LLMStreamChunk to StreamChunk format
        subscriber.next({
          chunk: llmChunk.chunk,
          done: llmChunk.done,
          error: llmChunk.error,
        });
      },
      error: (err) => {
        console.error('[Zagalin] Frontend-only LLM error:', err);
        subscriber.error(err);
      },
      complete: () => subscriber.complete(),
    });
  });
}

/**
 * Build system prompt from assistant context
 * This is a simplified version of the backend's prompt builder
 */
function buildSystemPrompt(request: AssistantRequest): string {
  let prompt = 'You are Zagalin, an AI assistant for Grafana that helps with observability, monitoring, and troubleshooting.\n\n';

  if (request.attachedContexts && request.attachedContexts.length > 0) {
    prompt += `ATTACHED DASHBOARDS (${request.attachedContexts.length}):\n`;
    request.attachedContexts.forEach((ctx, i) => {
      prompt += `\n[${i + 1}] Dashboard: ${ctx.dashboardTitle} (UID: ${ctx.dashboardUid})\n`;
      if (ctx.panelId) {
        prompt += `    Panel: ${ctx.panelTitle || `Panel ${ctx.panelId}`}\n`;
      }
      if (ctx.timeFrom && ctx.timeTo) {
        prompt += `    Time Range: ${ctx.timeFrom} to ${ctx.timeTo}\n`;
      }
      prompt += `    Added: ${ctx.addedAt.toLocaleString()}\n`;
    });
    prompt += '\n';
  }

  if (request.context) {
    if (request.context.dashboard) {
      prompt += `Current Dashboard Context:\n`;
      prompt += `- Dashboard: ${request.context.dashboard.title}\n`;
      if (request.context.dashboard.uid) {
        prompt += `- UID: ${request.context.dashboard.uid}\n`;
      }
      if (request.context.dashboard.panels && request.context.dashboard.panels.length > 0) {
        prompt += `- Panels: ${request.context.dashboard.panels.length} panels\n`;
      }
      prompt += '\n';
    }

    if (request.context.panel) {
      prompt += `Current Panel Context:\n`;
      prompt += `- Panel: ${request.context.panel.title || 'Untitled'}\n`;
      prompt += `- Type: ${request.context.panel.type}\n`;
      if (request.context.panel.description) {
        prompt += `- Description: ${request.context.panel.description}\n`;
      }
      prompt += '\n';
    }

    if (request.context.timeRange) {
      prompt += `Current Time Range: ${request.context.timeRange.from} to ${request.context.timeRange.to}\n\n`;
    }
  }

  prompt += `Your goal is to provide helpful, accurate, and concise answers. When generating queries:\n`;
  prompt += `- PromQL for Prometheus/Mimir/Cortex\n`;
  prompt += `- LogQL for Loki\n`;
  prompt += `- TraceQL for Tempo\n\n`;
  prompt += `Always provide working, tested query syntax. Explain your reasoning when appropriate.`;

  return buildReasoningPrompt(prompt, request.mode || 'standard');
}
