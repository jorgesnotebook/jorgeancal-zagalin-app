/**
 * Service Provider - Singleton instances and dependency injection
 *
 * This is the main entry point for all services in the application.
 * Usage: import { ServiceProvider } from 'services';
 *        const client = ServiceProvider.llmClient;
 */

import { LLMClient } from './llm/LLMClient';
import { ResponseProcessor } from './response/ResponseProcessor';
import { ContextManager } from './context/ContextManager';
import { DataExtractor } from './data/DataExtractor';
import { ConversationManager } from './conversation/ConversationManager';
import { Orchestrator } from './orchestrator/Orchestrator';

/**
 * Service Provider - Centralized service access with singleton pattern
 */
export class ServiceProvider {
  private static _llmClient: LLMClient | null = null;
  private static _responseProcessor: ResponseProcessor | null = null;
  private static _contextManager: ContextManager | null = null;
  private static _dataExtractor: DataExtractor | null = null;
  private static _conversationManager: ConversationManager | null = null;
  private static _orchestrator: Orchestrator | null = null;

  /**
   * Get LLM Client singleton instance
   */
  static get llmClient(): LLMClient {
    if (!this._llmClient) {
      this._llmClient = new LLMClient();
    }
    return this._llmClient;
  }

  /**
   * Get Context Manager singleton instance
   */
  static get contextManager(): ContextManager {
    if (!this._contextManager) {
      this._contextManager = new ContextManager();
    }
    return this._contextManager;
  }

  /**
   * Get Response Processor singleton instance
   */
  static get responseProcessor(): ResponseProcessor {
    if (!this._responseProcessor) {
      this._responseProcessor = new ResponseProcessor();
    }
    return this._responseProcessor;
  }

  /**
   * Get Data Extractor singleton instance
   */
  static get dataExtractor(): DataExtractor {
    if (!this._dataExtractor) {
      this._dataExtractor = new DataExtractor();
    }
    return this._dataExtractor;
  }

  /**
   * Get Conversation Manager singleton instance
   */
  static get conversationManager(): ConversationManager {
    if (!this._conversationManager) {
      this._conversationManager = new ConversationManager();
    }
    return this._conversationManager;
  }

  /**
   * Get Orchestrator singleton instance
   * Orchestrator depends on LLMClient, ContextManager, and ResponseProcessor
   */
  static get orchestrator(): Orchestrator {
    if (!this._orchestrator) {
      this._orchestrator = new Orchestrator(
        this.llmClient,
        this.contextManager,
        this.responseProcessor
      );
    }
    return this._orchestrator;
  }

  /**
   * Reset all services (for testing)
   * This clears all singleton instances, forcing new instances on next access
   */
  static resetForTesting(): void {
    this._llmClient = null;
    this._responseProcessor = null;
    this._contextManager = null;
    this._dataExtractor = null;
    this._conversationManager = null;
    this._orchestrator = null;
  }

  /**
   * Inject mock services (for testing)
   * Allows replacing services with mocks for unit testing
   */
  static inject(
    services: Partial<{
      llmClient: LLMClient;
      responseProcessor: ResponseProcessor;
      contextManager: ContextManager;
      dataExtractor: DataExtractor;
      conversationManager: ConversationManager;
      orchestrator: Orchestrator;
    }>
  ): void {
    if (services.llmClient) {
      this._llmClient = services.llmClient;
    }
    if (services.responseProcessor) {
      this._responseProcessor = services.responseProcessor;
    }
    if (services.contextManager) {
      this._contextManager = services.contextManager;
    }
    if (services.dataExtractor) {
      this._dataExtractor = services.dataExtractor;
    }
    if (services.conversationManager) {
      this._conversationManager = services.conversationManager;
    }
    if (services.orchestrator) {
      this._orchestrator = services.orchestrator;
    }
  }
}

// Re-export services for direct import if needed
export { LLMClient } from './llm/LLMClient';
export { ResponseProcessor } from './response/ResponseProcessor';
export { ContextManager } from './context/ContextManager';
export { DataExtractor } from './data/DataExtractor';
export { ConversationManager } from './conversation/ConversationManager';
export { Orchestrator } from './orchestrator/Orchestrator';

// Re-export React hooks
export { useGrafanaContext } from './context/hooks';

// Re-export common types
export type { Message, ChatOptions, StreamChunk, BackendType } from './llm/types';
export type { ToolCall, ToolResult, ReasoningStep, Artifact, Action } from './response/types';
export type { GrafanaContext, DashboardContext, PanelContext, UserContext, TimeRange } from './context/types';
export type { PanelDataAnalysis, LogAnalysisResult, DatasourceInfo } from './data/types';
export type { Conversation, ExportData } from './conversation/types';
export type { OrchestrationOptions, MultiStepRequest, WorkflowUpdate, RunStatus } from './orchestrator/types';

// TODO: Re-export utility services when they're moved to utils/
// export { VectorSearchService } from './utils/vectorSearch';
// export { checkZagalinHealth } from './utils/health';
// export { getPluginUrl } from './utils/url';
