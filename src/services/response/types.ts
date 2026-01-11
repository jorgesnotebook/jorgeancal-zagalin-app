/**
 * Type definitions for Response Processing module
 *
 * Consolidated from:
 * - zagalinTools.ts
 * - artifactExtractor.ts (runService.ts Artifact)
 * - reasoningParser.ts (explainableAI.ts)
 * - actionExtractor.ts (contextTypes.ts AssistantAction)
 */

import type { TimeRange } from '../contextTypes';

/**
 * Tool call from LLM (OpenAI format)
 */
export interface ToolCall {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string; // JSON string
  };
}

/**
 * Result of tool execution
 */
export interface ToolResult {
  toolCallId: string;
  toolName: string;
  result: any;
  error?: string;
}

/**
 * Reasoning step from LLM response
 */
export interface ReasoningStep {
  id: string;
  type: 'observation' | 'hypothesis' | 'analysis' | 'conclusion' | 'verification';
  content: string;
  confidence: number;
  timestamp: Date;
  sources?: string[];
}

/**
 * Source reference for reasoning
 */
export interface SourceReference {
  type: 'dashboard' | 'panel' | 'metric' | 'log' | 'trace' | 'documentation';
  id: string;
  name: string;
  relevance: number;
}

/**
 * Artifact extracted from response (queries, trace IDs, etc.)
 */
export interface Artifact {
  id: string;
  type: 'query' | 'link' | 'trace_id' | 'dashboard_link' | 'tool_call';
  content: string;
  metadata: Record<string, any>;
  timestamp: string;
}

/**
 * Action suggestion from LLM response
 */
export interface Action {
  type: 'explore' | 'copy' | 'dashboard' | 'panel' | 'query';
  label: string;
  data: {
    query?: string;
    datasourceUid?: string;
    dashboardUid?: string;
    panelId?: number;
    timeRange?: TimeRange;
  };
}

/**
 * Complete parsed response with reasoning
 */
export interface ExplainableResponse {
  answer: string;
  reasoning: ReasoningStep[];
  sources: SourceReference[];
  confidence: number;
  alternativeApproaches?: string[];
  caveats?: string[];
}
