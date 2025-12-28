/**
 * Frontend Orchestrator - Client-side orchestration for grafana-llm-app mode
 *
 * This provides structured planning/steps/artifacts workflow by making
 * multiple sequential calls to grafana-llm-app from the frontend.
 */

import { Observable } from 'rxjs';
import type { AssistantMessage, AssistantContext } from './assistantService';
import type { Artifact } from './runService';
import {
  type ExecutionPlan,
  PLANNING_SYSTEM_PROMPT,
  buildPlanningPrompt,
  buildStepPrompt,
  buildSynthesisPrompt,
  parsePlanFromResponse,
} from './frontendPrompts';

// Event types for orchestration
export type OrchestratorEventType =
  | 'run_started'
  | 'plan'
  | 'step_started'
  | 'progress'
  | 'artifact'
  | 'assistant_delta'
  | 'step_done'
  | 'assistant_message'
  | 'final'
  | 'error';

export interface OrchestratorEvent {
  type: OrchestratorEventType;
  timestamp: Date;
  data: any;
}

export interface PlannedStep {
  index: number;
  title: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
}

/**
 * Frontend Orchestrator for grafana-llm-app mode
 *
 * Performs client-side orchestration with planning, step execution, and artifact extraction.
 */
export class FrontendOrchestrator {
  private plan: ExecutionPlan | null = null;
  private steps: PlannedStep[] = [];
  private currentStepIndex = 0;
  private artifacts: Artifact[] = [];
  private stepResults: Array<{ title: string; content: string }> = [];
  private cancelled = false;

  /**
   * Start orchestration
   */
  start(
    message: string,
    history: AssistantMessage[],
    context: AssistantContext
  ): Observable<OrchestratorEvent> {
    return new Observable<OrchestratorEvent>((subscriber) => {
      this.cancelled = false;
      this.artifacts = [];
      this.stepResults = [];
      this.currentStepIndex = 0;

      // Emit run started
      subscriber.next({
        type: 'run_started',
        timestamp: new Date(),
        data: { message: 'Starting investigation' },
      });

      // Phase 1: Planning
      this.executePlanningPhase(message, history, context, subscriber)
        .then(() => {
          if (this.cancelled) {
            subscriber.complete();
            return;
          }

          // Phase 2: Execute steps
          return this.executeStepsPhase(message, history, context, subscriber);
        })
        .then(() => {
          if (this.cancelled) {
            subscriber.complete();
            return;
          }

          // Phase 3: Synthesize results
          return this.executeSynthesisPhase(message, context, subscriber);
        })
        .then(() => {
          // Emit final event
          subscriber.next({
            type: 'final',
            timestamp: new Date(),
            data: {
              status: 'completed',
              totalSteps: this.steps.length,
              totalArtifacts: this.artifacts.length,
            },
          });
          subscriber.complete();
        })
        .catch((error) => {
          console.error('[FrontendOrchestrator] Error:', error);
          subscriber.next({
            type: 'error',
            timestamp: new Date(),
            data: { message: error.message || 'Orchestration failed' },
          });
          subscriber.error(error);
        });
    });
  }

  /**
   * Cancel orchestration
   */
  cancel() {
    this.cancelled = true;
  }

  /**
   * Phase 1: Planning
   */
  private async executePlanningPhase(
    message: string,
    history: AssistantMessage[],
    context: AssistantContext,
    subscriber: any
  ): Promise<void> {
    console.log('[FrontendOrchestrator] Phase 1: Planning');

    subscriber.next({
      type: 'progress',
      timestamp: new Date(),
      data: { message: 'Creating execution plan...' },
    });

    // Build planning messages
    const planningMessages = [
      { role: 'system' as const, content: PLANNING_SYSTEM_PROMPT },
      { role: 'user' as const, content: buildPlanningPrompt(message, context) },
    ];

    // Call grafana-llm-app for planning
    const planResponse = await this.callGrafanaLLM(planningMessages);

    // Parse plan
    this.plan = parsePlanFromResponse(planResponse);

    if (!this.plan) {
      throw new Error('Failed to generate execution plan');
    }

    // Convert to PlannedSteps
    this.steps = this.plan.steps.map((step, idx) => ({
      index: idx,
      title: step.title,
      description: step.description,
      status: 'pending' as const,
    }));

    // Emit plan event
    subscriber.next({
      type: 'plan',
      timestamp: new Date(),
      data: {
        goal: this.plan.goal,
        steps: this.steps,
        estimatedDuration: this.plan.estimatedDuration || '2-3 minutes',
      },
    });

    console.log('[FrontendOrchestrator] Plan created:', this.plan.goal);
  }

  /**
   * Phase 2: Execute steps
   */
  private async executeStepsPhase(
    message: string,
    history: AssistantMessage[],
    context: AssistantContext,
    subscriber: any
  ): Promise<void> {
    console.log('[FrontendOrchestrator] Phase 2: Executing steps');

    for (let i = 0; i < this.steps.length; i++) {
      if (this.cancelled) {
        break;
      }

      this.currentStepIndex = i;
      const step = this.steps[i];

      // Update step status
      step.status = 'in_progress';

      // Emit step started
      subscriber.next({
        type: 'step_started',
        timestamp: new Date(),
        data: {
          stepIndex: i,
          stepTitle: step.title,
          description: step.description,
        },
      });

      try {
        // Build step prompt with previous findings
        const stepPromptContent = buildStepPrompt(
          i,
          step.title,
          step.description,
          message,
          context,
          this.stepResults.map((r) => r.content)
        );

        const stepMessages = [{ role: 'user' as const, content: stepPromptContent }];

        // Execute step - stream response
        let stepContent = '';
        await this.streamGrafanaLLM(stepMessages, (chunk) => {
          if (chunk) {
            stepContent += chunk;
            subscriber.next({
              type: 'assistant_delta',
              timestamp: new Date(),
              data: { delta: chunk },
            });
          }
        });

        // Store step result
        this.stepResults.push({
          title: step.title,
          content: stepContent,
        });

        // Extract artifacts from step content
        const stepArtifacts = this.extractArtifacts(stepContent);
        for (const artifact of stepArtifacts) {
          this.artifacts.push(artifact);
          subscriber.next({
            type: 'artifact',
            timestamp: new Date(),
            data: artifact,
          });
        }

        // Update step status
        step.status = 'completed';

        // Emit step done
        subscriber.next({
          type: 'step_done',
          timestamp: new Date(),
          data: {
            stepIndex: i,
            status: 'completed',
          },
        });

        console.log(`[FrontendOrchestrator] Step ${i + 1} completed`);
      } catch (error) {
        console.error(`[FrontendOrchestrator] Step ${i + 1} failed:`, error);
        step.status = 'failed';
        subscriber.next({
          type: 'step_done',
          timestamp: new Date(),
          data: {
            stepIndex: i,
            status: 'failed',
            error: error instanceof Error ? error.message : 'Step failed',
          },
        });
        // Continue with next step
      }
    }
  }

  /**
   * Phase 3: Synthesize results
   */
  private async executeSynthesisPhase(
    message: string,
    context: AssistantContext,
    subscriber: any
  ): Promise<void> {
    console.log('[FrontendOrchestrator] Phase 3: Synthesis');

    subscriber.next({
      type: 'progress',
      timestamp: new Date(),
      data: { message: 'Synthesizing findings...' },
    });

    if (!this.plan) {
      throw new Error('No plan available for synthesis');
    }

    // Build synthesis prompt
    const synthesisPromptContent = buildSynthesisPrompt(
      message,
      this.plan.goal,
      this.stepResults,
      this.artifacts
    );

    const synthesisMessages = [{ role: 'user' as const, content: synthesisPromptContent }];

    // Call LLM for final synthesis
    const finalMessage = await this.callGrafanaLLM(synthesisMessages);

    // Emit assistant message
    subscriber.next({
      type: 'assistant_message',
      timestamp: new Date(),
      data: {
        text: finalMessage,
        artifacts: this.artifacts,
      },
    });

    console.log('[FrontendOrchestrator] Synthesis complete');
  }

  /**
   * Call grafana-llm-app (non-streaming)
   */
  private async callGrafanaLLM(messages: AssistantMessage[]): Promise<string> {
    const response = await fetch('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'same-origin',
      body: JSON.stringify({
        model: 'gpt-4o-mini', // grafana-llm-app will use configured model
        messages: messages,
        stream: false,
        temperature: 0.3, // Lower for structured output
        max_tokens: 2000,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`grafana-llm-app error: ${response.status} - ${errorText}`);
    }

    const data = await response.json();
    return data.choices?.[0]?.message?.content || '';
  }

  /**
   * Stream from grafana-llm-app
   */
  private async streamGrafanaLLM(
    messages: AssistantMessage[],
    onChunk: (chunk: string) => void
  ): Promise<void> {
    const response = await fetch('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      credentials: 'same-origin',
      body: JSON.stringify({
        model: 'gpt-4o-mini',
        messages: messages,
        stream: true,
        temperature: 0.7,
        max_tokens: 2000,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`grafana-llm-app error: ${response.status} - ${errorText}`);
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
            return;
          }

          try {
            const chunk = JSON.parse(data);
            const delta = chunk.choices?.[0]?.delta?.content;

            if (delta) {
              onChunk(delta);
            }

            if (chunk.choices?.[0]?.finish_reason) {
              return;
            }
          } catch (e) {
            console.warn('[FrontendOrchestrator] Failed to parse SSE chunk:', data);
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  /**
   * Extract artifacts from text (queries, trace IDs, links)
   */
  private extractArtifacts(text: string): Artifact[] {
    const artifacts: Artifact[] = [];

    // Extract PromQL queries from markdown code blocks
    const promqlPattern = /```(?:promql|prometheus)\s*\n([\s\S]+?)\n```/gi;
    let match;

    while ((match = promqlPattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.length > 10) {
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'query',
          content: query,
          metadata: {
            signal: 'metrics',
            format: 'promql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    // Extract LogQL queries from markdown code blocks
    const logqlPattern = /```(?:logql|loki)\s*\n([\s\S]+?)\n```/gi;

    while ((match = logqlPattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.includes('=') && query.length > 10) {
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'query',
          content: query,
          metadata: {
            signal: 'logs',
            format: 'logql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    // Extract TraceQL queries from markdown code blocks
    const traceqlPattern = /```(?:traceql|tempo)\s*\n([\s\S]+?)\n```/gi;

    while ((match = traceqlPattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.length > 5) {
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'query',
          content: query,
          metadata: {
            signal: 'traces',
            format: 'traceql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    // Extract inline PromQL queries (rate(...), sum(...), etc.)
    const inlinePromQLPattern = /\b(rate|sum|avg|count|histogram_quantile|increase)\([^)]+\)(?:\{[^}]+\})?(?:\[[^\]]+\])?/g;

    while ((match = inlinePromQLPattern.exec(text)) !== null) {
      const query = match[0].trim();
      if (query.length > 10 && !artifacts.some((a) => a.content === query)) {
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'query',
          content: query,
          metadata: {
            signal: 'metrics',
            format: 'promql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    // Extract inline LogQL queries ({...})
    const inlineLogQLPattern = /\{[^}]+\}\s*(?:\|[^|\n]+)*/g;

    while ((match = inlineLogQLPattern.exec(text)) !== null) {
      const query = match[0].trim();
      if (query.includes('=') && query.length > 10 && !artifacts.some((a) => a.content === query)) {
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'query',
          content: query,
          metadata: {
            signal: 'logs',
            format: 'logql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    // Extract trace IDs (hex strings 16-32 chars)
    const traceIDPattern = /\b[0-9a-f]{16,32}\b/gi;
    const seenTraceIDs = new Set<string>();

    while ((match = traceIDPattern.exec(text)) !== null) {
      const traceID = match[0].toLowerCase();
      if (traceID.length >= 16 && !seenTraceIDs.has(traceID)) {
        seenTraceIDs.add(traceID);
        artifacts.push({
          id: `art_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`,
          type: 'trace_id',
          content: traceID,
          metadata: {
            signal: 'traces',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  /**
   * Get current state (for UI)
   */
  getState() {
    return {
      plan: this.plan,
      steps: this.steps,
      currentStepIndex: this.currentStepIndex,
      artifacts: this.artifacts,
      stepResults: this.stepResults,
    };
  }
}
