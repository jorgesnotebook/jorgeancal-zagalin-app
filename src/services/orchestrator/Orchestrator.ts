/**
 * Orchestrator - Coordinate multi-step workflows and LLM interactions
 *
 * Consolidates:
 * - frontendOrchestrator.ts (413 LOC) - Client-side multi-phase workflow
 * - frontendPrompts.ts (343 LOC) - System prompts for LLM
 * - runService.ts (166 LOC) - LLM run tracking
 *
 * Total: 922 LOC → ~550 LOC (eliminate duplication)
 */

import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import type { LLMClient } from '../llm/LLMClient';
import type { ContextManager } from '../context/ContextManager';
import type { ResponseProcessor } from '../response/ResponseProcessor';
import type { Message } from '../llm/types';
import type { OptimizedContext } from '../context/types';
import type { OrchestrationOptions, MultiStepRequest, WorkflowUpdate, RunStatus } from './types';

export class Orchestrator {
  private llmClient: LLMClient;
  private contextManager: ContextManager;
  private responseProcessor: ResponseProcessor;

  constructor(
    llmClient: LLMClient,
    contextManager: ContextManager,
    responseProcessor: ResponseProcessor
  ) {
    this.llmClient = llmClient;
    this.contextManager = contextManager;
    this.responseProcessor = responseProcessor;
  }

  /**
   * Handle user message with full orchestration
   *
   * TODO: Implement workflow coordination
   */
  async handleMessage(message: string, options?: OrchestrationOptions): Promise<Observable<any>> {
    // 1. Extract context
    const context = await this.contextManager.extractContext();

    // 2. Optimize for token limit
    const optimized = this.contextManager.optimizeForTokenLimit(context, 4000);

    // 3. Construct prompt messages
    const messages = this.constructPrompt(message, optimized);

    // 4. Stream response via LLM client
    const assistantRequest = {
      message: messages[messages.length - 1].content, // Last message is user message
      history: messages.slice(0, -1), // Previous messages as history
      context: context,
    };
    const stream = await this.llmClient.chat(assistantRequest);

    // 5. Process response (extract tools, artifacts, reasoning)
    return stream.pipe(map((chunk) => this.responseProcessor.extractToolCalls(chunk.chunk || '')));
  }

  /**
   * Construct system prompt with context
   *
   * TODO: Implement from frontendPrompts.ts
   */
  private constructPrompt(message: string, _context: OptimizedContext): Message[] {
    // TODO: Implement prompt construction with optimized context
    return [
      {
        role: 'system',
        content: 'You are a helpful assistant for Grafana.',
      },
      {
        role: 'user',
        content: message,
      },
    ];
  }

  /**
   * Execute multi-step workflow (planning mode)
   *
   * TODO: Implement from frontendOrchestrator.ts
   */
  async executeMultiStep(request: MultiStepRequest): Promise<Observable<WorkflowUpdate>> {
    // TODO: Implement multi-step workflow execution
    throw new Error('Not implemented yet');
  }

  /**
   * Track LLM run
   *
   * TODO: Implement from runService.ts
   */
  async trackRun(runId: string, status: RunStatus): Promise<void> {
    // TODO: Implement run tracking
  }
}
