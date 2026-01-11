/**
 * Assistant Service Router (MIGRATION WRAPPER)
 *
 * This file is kept as a thin wrapper during migration to the new LLMClient module.
 * Components using this will gradually migrate to use LLMClient directly.
 *
 * Original: Routes LLM requests based on llmBackend configuration
 * New: Forwards to LLMClient which handles routing internally
 */

import { Observable } from 'rxjs';
import { LLMClient } from './llm/LLMClient';
import type { AssistantRequest, StreamChunk } from './llm/types';

// Singleton instance for reuse
const llmClient = new LLMClient();

/**
 * Stream assistant chat with automatic routing based on configuration
 *
 * @deprecated Use LLMClient directly: `new LLMClient().chat(request)`
 * @param request The assistant request
 * @returns Observable stream of response chunks
 */
export function streamAssistantChatRouted(request: AssistantRequest): Observable<StreamChunk> {
  return llmClient.chat(request);
}

// Re-export types for backwards compatibility
export type { AssistantRequest, StreamChunk } from './llm/types';
