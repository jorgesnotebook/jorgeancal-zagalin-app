/**
 * Type definitions for LLM module
 *
 * Consolidated from:
 * - assistantService.ts
 * - llmDirectClient.ts
 * - assistantServiceRouter.ts
 */

export type BackendType = 'backend-proxy' | 'grafana-llm';

// Message types
export interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

// Context types (from assistantService.ts)
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

export interface AssistantContext {
  dashboard?: DashboardContext;
  panel?: PanelContext;
  timeRange?: TimeRange;
  templateVars?: TemplateVariable[];
}

// Request types
export interface AssistantRequest {
  message: string;
  history: Message[];
  context: AssistantContext;
  skillHint?: string;
  enrichedMessage?: string;
  mode?: 'standard' | 'design';
  attachedContexts?: Array<{
    dashboardUid: string;
    dashboardTitle: string;
    panelId?: number;
    panelTitle?: string;
    timeFrom?: string;
    timeTo?: string;
    addedAt: Date;
  }>;
}

// Response types
export interface StreamChunk {
  chunk?: string;
  done?: boolean;
  error?: string;
  tool_call?: ToolCallChunk;
  history_update?: Message[];
}

export interface ToolCallChunk {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

// LLM client options
export interface ChatOptions {
  temperature?: number;
  maxTokens?: number;
  model?: string;
  stream?: boolean;
  tools?: any[];
  abortSignal?: AbortSignal;
}
