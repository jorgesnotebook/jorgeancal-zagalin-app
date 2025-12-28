/**
 * Direct LLM Client - Calls grafana-llm-app directly from frontend
 *
 * This bypasses the backend proxy and calls grafana-llm-app directly,
 * which works because the frontend has access to session cookies.
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
 * Stream chat directly with grafana-llm-app
 *
 * This calls grafana-llm-app's /llm/chat endpoint directly from the frontend,
 * which automatically includes session cookies for authentication.
 *
 * @param request The LLM request
 * @returns Observable stream of response chunks
 */
export function streamLLMChat(request: LLMStreamRequest): Observable<LLMStreamChunk> {
  return new Observable<LLMStreamChunk>((subscriber) => {
    const url = '/api/plugins/grafana-llm-app/resources/llm/chat';

    // Add streaming flag
    const requestBody = {
      ...request,
      stream: true,
    };

    // Create fetch request for SSE
    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream',
      },
      body: JSON.stringify(requestBody),
      credentials: 'same-origin', // CRITICAL: Include session cookies
    })
      .then(async (response) => {
        if (!response.ok) {
          const error = await response.text();
          throw new Error(`LLM API error: ${response.status} - ${error}`);
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
                subscriber.next({ done: true });
                subscriber.complete();
                return;
              }

              try {
                const chunk: LLMStreamChunk = JSON.parse(data);

                // Emit chunk
                subscriber.next(chunk);

                // Check if done or error
                if (chunk.done || chunk.error) {
                  if (chunk.error) {
                    subscriber.error(new Error(chunk.error));
                  } else {
                    subscriber.complete();
                  }
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
