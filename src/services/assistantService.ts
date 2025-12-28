/**
 * Assistant Service - Client for backend LLM API
 * Replaces direct grafana-llm-app calls with backend proxy
 */

import { Observable } from 'rxjs';

export interface AssistantMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface AssistantContext {
  dashboard?: DashboardContext;
  panel?: PanelContext;
  timeRange?: TimeRange;
  templateVars?: TemplateVariable[];
}

export interface DashboardContext {
  uid: string;
  title: string;
  tags?: string[];
  panels?: PanelContext[];
}

export interface PanelContext {
  title: string;
  type: string;
  description?: string;
  targets: QueryTarget[];
  fieldConfig?: any;
  transformations?: any[];
}

export interface QueryTarget {
  refId: string;
  expr?: string;
  query?: string;
  datasource?: {
    type?: string;
    uid?: string;
  };
}

export interface TimeRange {
  from: string;
  to: string;
}

export interface TemplateVariable {
  name: string;
  current: {
    value: any;
  };
}

export interface AssistantRequest {
  message: string;
  history: AssistantMessage[];
  context: AssistantContext;
  skillHint?: string;
  enrichedMessage?: string; // Optional pre-enriched message with full context
}

export interface StreamChunk {
  chunk?: string;
  done?: boolean;
  error?: string;
  tool_call?: ToolCallChunk;
}

export interface ToolCallChunk {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

/**
 * Get the LLM backend mode from plugin configuration
 */
async function getLLMBackendMode(): Promise<string> {
  try {
    const response = await fetch('/api/plugins/jorgeancal-zagalin-app/resources/settings');
    if (!response.ok) {
      console.warn('Failed to fetch plugin settings, defaulting to grafana-llm-app mode');
      return 'grafana-llm-app';
    }
    const settings = await response.json();
    return settings?.jsonData?.llmBackend || 'grafana-llm-app';
  } catch (error) {
    console.warn('Error fetching plugin settings:', error);
    return 'grafana-llm-app'; // Safe fallback
  }
}

/**
 * Stream chat with the assistant
 *
 * Routing logic:
 * - grafana-llm-app mode: Call grafana-llm-app directly from frontend (as per their docs)
 * - direct mode: Call backend which then calls OpenAI/Anthropic/etc
 * - disabled mode: Return error
 *
 * @param request The assistant request
 * @returns Observable stream of response chunks
 */
export function streamAssistantChat(request: AssistantRequest): Observable<StreamChunk> {
  return new Observable<StreamChunk>((subscriber) => {
    // First, determine which backend mode we're using
    getLLMBackendMode().then((backendMode) => {
      if (backendMode === 'disabled') {
        subscriber.error(new Error('LLM features are disabled in plugin configuration'));
        return;
      }

      let url: string;
      let backendRequest: any;

      if (backendMode === 'grafana-llm-app') {
        // Call grafana-llm-app directly (as per grafana-llm-app documentation)
        url = '/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions';

        // Build OpenAI-compatible request
        const messages = [...request.history];
        if (request.enrichedMessage || request.message) {
          messages.push({ role: 'user', content: request.enrichedMessage || request.message });
        }

        backendRequest = {
          model: 'gpt-4o-mini', // Will use model configured in grafana-llm-app
          messages: messages,
          stream: true,
          temperature: 0.7,
          max_tokens: 2000,
        };
      } else {
        // Direct mode: Call backend which handles OpenAI/Anthropic/etc
        url = '/api/plugins/jorgeancal-zagalin-app/resources/llm/chat';

        backendRequest = {
          message: request.message,
          history: request.history,
          context: request.context,
          skillHint: request.skillHint,
        };
      }

      // Create fetch request for SSE
      fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'text/event-stream',
        },
        body: JSON.stringify(backendRequest),
        credentials: 'same-origin', // CRITICAL: Include cookies for authentication
      })
        .then(async (response) => {
          if (!response.ok) {
            const error = await response.text();
            throw new Error(`Error: ${response.status} - ${error}`);
          }

          if (!response.body) {
            throw new Error('No response body');
          }

          // Read SSE stream
          const reader = response.body.getReader();
          const decoder = new TextDecoder();

          try {
            while (true) {
              const { done, value } = await reader.read();

              if (done) {
                subscriber.complete();
                break;
              }

              // Decode chunk
              const text = decoder.decode(value, { stream: true });

              // Parse SSE lines
              const lines = text.split('\n');
              for (const line of lines) {
                if (!line.trim() || !line.startsWith('data: ')) {
                  continue;
                }

                const data = line.substring(6); // Remove "data: " prefix

                // Check for done marker
                if (data === '[DONE]') {
                  subscriber.complete();
                  return;
                }

                try {
                  if (backendMode === 'grafana-llm-app') {
                    // Parse OpenAI format from grafana-llm-app
                    const openaiChunk = JSON.parse(data);
                    const delta = openaiChunk.choices?.[0]?.delta?.content;

                    if (delta) {
                      subscriber.next({ chunk: delta });
                    }

                    // Check if done
                    if (openaiChunk.choices?.[0]?.finish_reason) {
                      subscriber.next({ done: true });
                      subscriber.complete();
                      return;
                    }
                  } else {
                    // Parse backend format (direct mode)
                    const chunk = JSON.parse(data) as StreamChunk;

                    // Emit chunk
                    subscriber.next(chunk);

                    // Check if done
                    if (chunk.done || chunk.error) {
                      subscriber.complete();
                      return;
                    }
                  }
                } catch (parseError) {
                  console.warn('Failed to parse SSE chunk:', data, parseError);
                }
              }
            }
          } catch (readError) {
            subscriber.error(readError);
          }
        })
        .catch((fetchError) => {
          subscriber.error(fetchError);
        });
    }).catch((error) => {
      subscriber.error(error);
    });

    // Cleanup function
    return () => {
      // No way to cancel fetch SSE, but we can at least signal we're done
    };
  });
}
