/**
 * Assistant Service (MIGRATION WRAPPER)
 *
 * This file is kept as a thin wrapper during migration to the new LLMClient module.
 * Components using this will gradually migrate to use LLMClient directly.
 *
 * Original: Backend proxy client for LLM chat
 * New: Forwards to LLMClient which handles backend proxy internally
 */

import { Observable } from 'rxjs';
import { LLMClient } from './llm/LLMClient';

// Re-export types for backwards compatibility
export type {
  AssistantRequest,
  StreamChunk,
  ToolCallChunk,
  Message as AssistantMessage,
  AssistantContext,
  DashboardContext,
  PanelContext,
  QueryTarget,
  TimeRange,
  TemplateVariable,
} from './llm/types';

// Singleton instance for reuse
const llmClient = new LLMClient();

/**
 * Stream assistant chat via backend proxy
 *
 * @deprecated Use LLMClient directly: `new LLMClient().chat(request)`
 * @param request The assistant request
 * @returns Observable stream of response chunks
 */
export function streamAssistantChat(request: import('./llm/types').AssistantRequest): Observable<import('./llm/types').StreamChunk> {
  return llmClient.chat(request);
}
