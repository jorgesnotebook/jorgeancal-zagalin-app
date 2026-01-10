/**
 * Grafana LLM Client - Uses @grafana/llm package
 *
 * This is the official way to call grafana-llm-app from the frontend.
 * Benefits:
 * - Official Grafana API (less likely to break with updates)
 * - Proper type safety and error handling
 * - Built-in health checking (llm.enabled(), llm.health())
 * - Support for MCP tools and vector search
 */

import { Observable } from 'rxjs';

export interface LLMMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface LLMStreamRequest {
  model: string;
  messages: LLMMessage[];
  temperature?: number;
  max_tokens?: number;
  stream?: boolean;
}

export interface LLMStreamChunk {
  chunk?: string;
  done?: boolean;
  error?: string;
}

/**
 * Stream chat using @grafana/llm package
 *
 * Uses the official @grafana/llm package to call grafana-llm-app.
 * Falls back to direct HTTP if package is unavailable.
 *
 * @param request The LLM request
 * @returns Observable stream of response chunks
 */
export function streamGrafanaLLM(request: LLMStreamRequest): Observable<LLMStreamChunk> {
  return new Observable<LLMStreamChunk>((subscriber) => {
    import('@grafana/llm')
      .then(({ llm }) => {
        llm.enabled()
          .then((enabled) => {
            if (!enabled) {
              subscriber.error(new Error('grafana-llm-app is not enabled or configured'));
              return;
            }

            const stream = llm.streamChatCompletions({
              messages: request.messages,
              temperature: request.temperature,
              max_tokens: request.max_tokens,
            });

            stream.subscribe({
              next: (chunk: any) => {
                if (chunk.choices && chunk.choices[0]?.delta?.content) {
                  subscriber.next({
                    chunk: chunk.choices[0].delta.content,
                    done: false,
                  });
                }
                if (chunk.choices && chunk.choices[0]?.finish_reason) {
                  subscriber.next({ done: true });
                  subscriber.complete();
                }
              },
              error: (err) => {
                console.error('[Zagalin] @grafana/llm error:', err);
                subscriber.error(err);
              },
              complete: () => {
                subscriber.next({ done: true });
                subscriber.complete();
              },
            });
          })
          .catch((err) => {
            console.error('[Zagalin] Failed to check LLM availability:', err);
            subscriber.error(err);
          });
      })
      .catch((importError) => {
        console.warn('[Zagalin] @grafana/llm not available, using fallback HTTP client');
        streamLLMFallback(request, subscriber);
      });
  });
}

/**
 * Fallback HTTP client (used if @grafana/llm package is unavailable)
 */
function streamLLMFallback(request: LLMStreamRequest, subscriber: any): void {
  const url = '/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions';

    const requestBody = {
      ...request,
      stream: true,
    };

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream',
      },
      body: JSON.stringify(requestBody),
      credentials: 'same-origin',
    })
      .then(async (response) => {
        if (!response.ok) {
          const error = await response.text();
          throw new Error(`LLM API error: ${response.status} - ${error}`);
        }

        if (!response.body) {
          throw new Error('No response body');
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();

        try {
          while (true) {
            const { done, value } = await reader.read();

            if (done) {
              subscriber.complete();
              break;
            }

            const text = decoder.decode(value, { stream: true });

            const lines = text.split('\n');
            for (const line of lines) {
              if (!line.trim() || !line.startsWith('data: ')) {
                continue;
              }

              const data = line.substring(6);

              if (data === '[DONE]') {
                subscriber.next({ done: true });
                subscriber.complete();
                return;
              }

              try {
                const openAIChunk = JSON.parse(data);

                if (openAIChunk.choices && openAIChunk.choices[0]?.delta?.content) {
                  subscriber.next({
                    chunk: openAIChunk.choices[0].delta.content,
                    done: false,
                  });
                }

                if (openAIChunk.choices && openAIChunk.choices[0]?.finish_reason) {
                  subscriber.next({ done: true });
                  subscriber.complete();
                  return;
                }
              } catch (parseError) {
                console.warn('[Zagalin] Failed to parse SSE chunk:', data, parseError);
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
}
