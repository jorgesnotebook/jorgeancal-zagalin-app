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
import { getZagalinConfig } from './configHelper';
import { LLM_CONFIG } from '../utils/constants';
import { extractArtifacts } from './artifactExtractor';

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

  start(message: string, history: AssistantMessage[], context: AssistantContext): Observable<OrchestratorEvent> {
    return new Observable<OrchestratorEvent>((subscriber) => {
      this.cancelled = false;
      this.artifacts = [];
      this.stepResults = [];
      this.currentStepIndex = 0;

      subscriber.next({
        type: 'run_started',
        timestamp: new Date(),
        data: { message: 'Starting investigation' },
      });

      this.executePlanningPhase(message, history, context, subscriber)
        .then(() => {
          if (this.cancelled) {
            subscriber.complete();
            return;
          }

          return this.executeStepsPhase(message, history, context, subscriber);
        })
        .then(() => {
          if (this.cancelled) {
            subscriber.complete();
            return;
          }

          return this.executeSynthesisPhase(message, context, subscriber);
        })
        .then(() => {
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

    const planningMessages = [
      { role: 'system' as const, content: PLANNING_SYSTEM_PROMPT },
      { role: 'user' as const, content: buildPlanningPrompt(message, context) },
    ];

    const planResponse = await this.callGrafanaLLM(planningMessages);

    this.plan = parsePlanFromResponse(planResponse);

    if (!this.plan) {
      throw new Error('Failed to generate execution plan');
    }

    this.steps = this.plan.steps.map((step, idx) => ({
      index: idx,
      title: step.title,
      description: step.description,
      status: 'pending' as const,
    }));

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

      step.status = 'in_progress';

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
        const stepPromptContent = buildStepPrompt(
          i,
          step.title,
          step.description,
          message,
          context,
          this.stepResults.map((r) => r.content)
        );

        const stepMessages = [{ role: 'user' as const, content: stepPromptContent }];

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

        this.stepResults.push({
          title: step.title,
          content: stepContent,
        });

        const stepArtifacts = extractArtifacts(stepContent);
        for (const artifact of stepArtifacts) {
          this.artifacts.push(artifact);
          subscriber.next({
            type: 'artifact',
            timestamp: new Date(),
            data: artifact,
          });
        }

        step.status = 'completed';

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
      }
    }
  }

  /**
   * Phase 3: Synthesize results
   */
  private async executeSynthesisPhase(message: string, context: AssistantContext, subscriber: any): Promise<void> {
    console.log('[FrontendOrchestrator] Phase 3: Synthesis');

    subscriber.next({
      type: 'progress',
      timestamp: new Date(),
      data: { message: 'Synthesizing findings...' },
    });

    if (!this.plan) {
      throw new Error('No plan available for synthesis');
    }

    const synthesisPromptContent = buildSynthesisPrompt(message, this.plan.goal, this.stepResults, this.artifacts);

    const synthesisMessages = [{ role: 'user' as const, content: synthesisPromptContent }];

    const finalMessage = await this.callGrafanaLLM(synthesisMessages);

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

  private async callGrafanaLLM(messages: AssistantMessage[]): Promise<string> {
    const config = getZagalinConfig();
    const standardConfig = config.standardMode;

    const response = await fetch('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'same-origin',
      body: JSON.stringify({
        model: LLM_CONFIG.DEFAULT_MODEL,
        messages: messages,
        stream: false,
        temperature: Math.max(0.3, standardConfig.temperature - LLM_CONFIG.PLANNING_TEMPERATURE_OFFSET),
        max_tokens: standardConfig.maxTokens,
      }),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`grafana-llm-app error: ${response.status} - ${errorText}`);
    }

    const data = await response.json();
    return data.choices?.[0]?.message?.content || '';
  }

  private async streamGrafanaLLM(messages: AssistantMessage[], onChunk: (chunk: string) => void): Promise<void> {
    const config = getZagalinConfig();
    const standardConfig = config.standardMode;

    const response = await fetch('/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      credentials: 'same-origin',
      body: JSON.stringify({
        model: LLM_CONFIG.DEFAULT_MODEL,
        messages: messages,
        stream: true,
        temperature: standardConfig.temperature,
        max_tokens: standardConfig.maxTokens,
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
