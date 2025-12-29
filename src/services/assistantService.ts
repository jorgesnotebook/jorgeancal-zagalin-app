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
  mode?: 'standard' | 'thinking'; // Chat mode: standard (fast) or thinking (extended reasoning)
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
 * Stream chat with the assistant
 *
 * All requests route through the backend for validation and security.
 *
 * @param request The assistant request
 * @returns Observable stream of response chunks
 */
export function streamAssistantChat(request: AssistantRequest): Observable<StreamChunk> {
  return new Observable<StreamChunk>((subscriber) => {
    // ALWAYS call backend (unified routing for validation)
    const url = '/api/plugins/jorgeancal-zagalin-app/resources/llm/chat';

    const backendRequest = {
      message: request.message,
      history: request.history,
      context: request.context,
      skillHint: request.skillHint,
      enrichedMessage: request.enrichedMessage,
      mode: request.mode || 'standard',
    };

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
                  // Parse backend format
                  const chunk = JSON.parse(data) as StreamChunk;

                  // Emit chunk
                  subscriber.next(chunk);

                  // Check if done
                  if (chunk.done || chunk.error) {
                    subscriber.complete();
                    return;
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

    // Cleanup function
    return () => {
      // No way to cancel fetch SSE, but we can at least signal we're done
    };
  });
}
