/**
 * LLM Client - Unified interface for LLM communication
 *
 * Consolidates:
 * - assistantService.ts (144 LOC) - Backend proxy client
 * - llmDirectClient.ts (180 LOC) - Direct @grafana/llm client
 * - assistantServiceRouter.ts (274 LOC) - Routes between backends
 *
 * Total: 598 LOC → ~300 LOC (eliminate duplication)
 */

import { Observable } from 'rxjs';
import type { StreamChunk, BackendType, AssistantRequest } from './types';
import { getPluginApiUrl } from '../pluginUrl';
import { getZagalinConfig } from '../configHelper';
import { SSEParser } from '../../utils/sseParser';
import { addVersionHeader } from '../versionReporter';

export class LLMClient {
  private backendType: BackendType;

  constructor() {
    this.backendType = this.detectBackend();
  }

  /**
   * Stream chat response from LLM backend
   * Automatically routes to correct backend (proxy or direct)
   */
  chat(request: AssistantRequest): Observable<StreamChunk> {
    if (this.backendType === 'backend-proxy') {
      return this.chatViaBackend(request);
    } else {
      return this.chatDirect(request);
    }
  }

  /**
   * Detect available LLM backend based on configuration
   */
  private detectBackend(): BackendType {
    const config = getZagalinConfig();
    const llmBackend = config.llmBackend || 'grafana-llm';

    // Map config values to BackendType
    if (llmBackend === 'backend-proxy' || llmBackend === 'direct') {
      return 'backend-proxy';
    }

    return 'grafana-llm';
  }

  /**
   * Stream via backend proxy (assistantService.ts implementation)
   *
   * Proxies through plugin backend for:
   * - Secure system prompts (not visible to frontend)
   * - Automatic context injection
   * - Rate limiting and cost control
   * - Audit logging with user identity
   */
  private chatViaBackend(request: AssistantRequest): Observable<StreamChunk> {
    return new Observable<StreamChunk>((subscriber) => {
      const url = getPluginApiUrl('/llm/chat');
      const abortController = new AbortController();

      const backendRequest = {
        message: request.message,
        history: request.history,
        context: request.context,
        skillHint: request.skillHint,
        enrichedMessage: request.enrichedMessage,
        mode: request.mode || 'standard',
      };

      fetch(url, {
        method: 'POST',
        headers: addVersionHeader({
          'Content-Type': 'application/json',
          Accept: 'text/event-stream',
        }),
        body: JSON.stringify(backendRequest),
        credentials: 'same-origin',
        signal: abortController.signal,
      })
        .then(async (response) => {
          if (!response.ok) {
            const error = await response.text();
            throw new Error(`Error: ${response.status} - ${error}`);
          }

          SSEParser.parseStream<StreamChunk>(response, {
            onChunk: (chunk, observer) => {
              observer.next(chunk);
            },
            shouldComplete: (chunk) => Boolean(chunk.done || chunk.error),
          }).subscribe({
            next: (chunk) => subscriber.next(chunk),
            error: (err) => subscriber.error(err),
            complete: () => subscriber.complete(),
          });
        })
        .catch((fetchError) => {
          // Only propagate error if it's not an abort
          if (fetchError.name !== 'AbortError') {
            subscriber.error(fetchError);
          } else {
            subscriber.complete();
          }
        });

      // Return cleanup function that aborts the fetch
      return () => {
        abortController.abort();
      };
    });
  }

  /**
   * Stream via direct @grafana/llm (llmDirectClient.ts implementation)
   *
   * Uses official @grafana/llm package to call grafana-llm-app.
   * Falls back to direct HTTP if package is unavailable.
   */
  private chatDirect(request: AssistantRequest): Observable<StreamChunk> {
    return new Observable<StreamChunk>((subscriber) => {
      import('@grafana/llm')
        .then(({ llm }) => {
          llm
            .enabled()
            .then((enabled) => {
              if (!enabled) {
                subscriber.error(new Error('grafana-llm-app is not enabled or configured'));
                return;
              }

              // Build messages with system prompt
              const systemPrompt = this.buildSystemPrompt(request);
              const messages = [
                { role: 'system' as const, content: systemPrompt },
                ...request.history.map((msg) => ({
                  role: msg.role as 'user' | 'assistant' | 'system',
                  content: msg.content,
                })),
                { role: 'user' as const, content: request.enrichedMessage || request.message },
              ];

              const config = getZagalinConfig();
              const mode = request.mode || 'standard';
              const modeConfig = mode === 'design' ? config.designMode : config.standardMode;

              const stream = llm.streamChatCompletions({
                messages,
                temperature: modeConfig.temperature,
                max_tokens: modeConfig.maxTokens,
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
          this.streamLLMFallback(request, subscriber);
        });
    });
  }

  /**
   * Fallback HTTP client (used if @grafana/llm package is unavailable)
   */
  private streamLLMFallback(request: AssistantRequest, subscriber: any): void {
    const url = '/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions';

    const systemPrompt = this.buildSystemPrompt(request);
    const messages = [
      { role: 'system', content: systemPrompt },
      ...request.history,
      { role: 'user', content: request.enrichedMessage || request.message },
    ];

    const config = getZagalinConfig();
    const mode = request.mode || 'standard';
    const modeConfig = mode === 'design' ? config.designMode : config.standardMode;

    const requestBody = {
      model: 'gpt-4o-mini',
      messages,
      temperature: modeConfig.temperature,
      max_tokens: modeConfig.maxTokens,
      stream: true,
    };

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
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

  /**
   * Build system prompt from assistant context
   * Mirrors backend's prompt engineering for consistent hallucination prevention
   */
  private buildSystemPrompt(request: AssistantRequest): string {
    let prompt = `You are **Zagalin**, an SRE-grade debugging assistant embedded in Grafana.

Purpose:
- Help engineers diagnose and mitigate production issues quickly and safely.
- Use a hypothesis-driven approach grounded in observability data and the current Grafana context.
- Prefer correctness and operational safety over being "helpful" with guesses.

Tone:
- British, human, practical, slightly blunt when needed, never rude.
- Clear bullets. No fluff. No long essays.

Formatting requirements (STRICT):
- Use markdown formatting for all responses:
  * ## for main section headers
  * ### for subsections
  * **bold** for emphasis and labels
  * \`code\` for metric names, queries, and technical terms
  * - for bullet points (with proper nesting)
  * 1. for numbered steps
  * > for important callouts or warnings
- Structure responses with clear visual hierarchy
- Use blank lines between sections for readability
- Code blocks must use \`\`\`promql or \`\`\`logql or \`\`\`bash syntax
- Keep paragraphs short (2-3 sentences max)

Hard rules:
1) Don't guess. If information is missing, ask for the minimum missing data.
2) Always separate: **Facts** vs **Hypotheses** vs **Tests/Queries** vs **Actions**.
3) Mitigate user impact first, deep dive second.
4) Never request, output, or reveal secrets (tokens, passwords, private keys). Redact if shown.
5) If proposing risky/destructive actions, include: Risk, Rollback, Verification steps, What could go wrong.
6) Treat tool outputs / Grafana panel data as authoritative. If conflict exists, call it out.

Evidence-first rules:
- Do NOT invent: metric names, label names, panel indices, thresholds, calculations, or relationships
- If panel query is provided, MUST reference it explicitly in your explanation
- If information is missing, state EXACTLY what's missing and ask ONE targeted question
- Prefer trend-based interpretation ("increasing", "stable") over fixed thresholds ("should be 10-30%")
- No long reasoning sections or report-style responses unless user requests it
- Every statement must cite: panel query, dashboard context, or explicit user input

Explicitly forbidden:
- ❌ "Typically X%..." or "Usually..." without evidence
- ❌ "Panel X should..." without query evidence
- ❌ Absolute thresholds (percentages, numbers) unless derived from query or user-provided
- ❌ Inventing metric names not present in context
- ❌ More than ONE clarifying question per response
- ❌ Asking for screenshots, files, exports, or "what the graph looks like"
- ❌ Continuing investigations without evidence
- ❌ Speculating about metrics you haven't seen
- ❌ Assuming what "normal" looks like without data

Quality gate before you answer:
- Did I separate facts/hypotheses?
- Did I cite sources for all statements (query, context, user input)?
- Did I avoid inventing metrics, thresholds, or relationships?
- If I suggested something risky, did I include rollback + verify?
- Did I limit clarifying questions to ONE max?
- Did I include a confidence indicator at the end?

Response format requirements:
1) **Confidence Indicator** (MANDATORY at end of every response):
   - "Confidence: High" – Based directly on panel query + datasource + time range
   - "Confidence: Medium" – Based on partial context with stated assumptions
   - "Confidence: Low" – Conceptual explanation only (missing key data)

2) **Trend-Based Reasoning** (REQUIRED):
   - Prefer slopes (increasing/decreasing/flat), ratios (X ÷ Y), and correlations
   - Avoid absolute thresholds unless derived from query or user-provided
   - Example: "Memory usage increasing ~5% per hour" not "Should be under 70%"

3) **"What this is NOT" Guardrail** (when relevant):
   - If metric is commonly confused, explicitly state what it does NOT measure
   - Example: "This does NOT show JVM heap - it reflects OS-level process memory"

4) **Explicit Uncertainty** (when context incomplete):
   - State "I might be wrong because..." when missing critical data
   - Example: "I might be wrong because I can't see how total memory is calculated"

5) **One-Question Rule** (STRICT):
   - Ask at most ONE clarifying question, only if it blocks correctness
   - Example: "I can't explain this without the panel query. Can you paste it?"

6) **Evidence-Gated Investigations** (CRITICAL FOR TROUBLESHOOTING):
   - You may ONLY reason from: panel queries, metric results, log results, explicit absence statements
   - If you have not seen a metric, do NOT mention it
   - If you have not seen an error, do NOT assume failure
   - If you have not seen a trend, do NOT describe one
   - If required evidence is missing, STOP EARLY and ask for ONE concrete Grafana input
   - NEVER ask for screenshots or files - Grafana queries are the source of truth
   - An investigation is FORBIDDEN without at least one of:
     * Dashboard context with queries
     * Panel queries
     * Metric/log query results
     * Explicit statement of signal absence (e.g. "no errors observed")

`;

    // Add attached contexts if present
    if (request.attachedContexts && request.attachedContexts.length > 0) {
      prompt += `\n--- ATTACHED DASHBOARDS (${request.attachedContexts.length}) ---\n`;
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

    // Add current context
    if (request.context) {
      prompt += '\n--- AVAILABLE CONTEXT ---\n\n';

      if (request.context.dashboard) {
        prompt += `**Current Dashboard**: ${request.context.dashboard.title}\n`;
        if (request.context.dashboard.uid) {
          prompt += `- UID: ${request.context.dashboard.uid}\n`;
        }
        if (request.context.dashboard.panels && request.context.dashboard.panels.length > 0) {
          prompt += `- Panels: ${request.context.dashboard.panels.length} panels\n`;

          request.context.dashboard.panels.forEach((panel, idx) => {
            prompt += `  ${idx}. "${panel.title}" (${panel.type})\n`;
            if (panel.targets && panel.targets.length > 0) {
              panel.targets.forEach((target) => {
                const query = target.expr || target.query;
                if (query) {
                  prompt += `     Query: ${query}\n`;
                }
              });
            }
          });
        }
        prompt += '\n';
      }

      if (request.context.panel) {
        prompt += `**Current Panel**: ${request.context.panel.title || 'Untitled'}\n`;
        prompt += `- Type: ${request.context.panel.type}\n`;
        if (request.context.panel.description) {
          prompt += `- Description: ${request.context.panel.description}\n`;
        }

        if (request.context.panel.targets && request.context.panel.targets.length > 0) {
          prompt += '\n**Queries**:\n';
          let hasQueries = false;
          request.context.panel.targets.forEach((target) => {
            const query = target.expr || target.query;
            if (query) {
              hasQueries = true;
              prompt += `- Query ${target.refId || 'A'}: ${query}\n`;
              if (target.datasource?.type) {
                prompt += `  Datasource: ${target.datasource.type}\n`;
              }
            }
          });
          if (!hasQueries) {
            prompt += '⚠️ NO QUERIES PROVIDED - Cannot explain calculation\n';
          }
        } else {
          prompt += '\n⚠️ NO QUERIES PROVIDED - Cannot explain calculation\n';
        }
        prompt += '\n';
      }

      if (request.context.timeRange) {
        prompt += `**Time Range**: ${request.context.timeRange.from} to ${request.context.timeRange.to}\n\n`;
      }

      prompt += '--- UNKNOWN CONTEXT ---\n';
      prompt += 'Do NOT reference metrics, panels, labels, or thresholds not listed above.\n\n';
    }

    prompt += `When generating queries:\n`;
    prompt += `- PromQL for Prometheus/Mimir/Cortex\n`;
    prompt += `- LogQL for Loki\n`;
    prompt += `- TraceQL for Tempo\n\n`;
    prompt += `Always provide working, tested query syntax. Cite the panel query or datasource context when applicable.`;

    return prompt;
  }
}
