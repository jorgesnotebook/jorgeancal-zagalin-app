/**
 * Zagalin Configuration Types
 * These settings control Zagalin's behavior, personality, and features
 */

/**
 * Base system prompt - NOT editable by users
 * Defines core identity (name, purpose) that cannot be changed
 */
export const BASE_SYSTEM_PROMPT = `You are Zagalin, an AI assistant for Grafana.

Core Rules:
- Your name is ALWAYS "Zagalin" - never use any other name
- You are specifically designed to help with Grafana, observability, and monitoring
- You have full context about the dashboard and panels the user is viewing
- NEVER ask for screenshots or additional information - use the provided context directly

Keep responses concise and practical.`;

export interface ZagalinConfig {
  // Custom instructions (user-editable, appended to base prompt)
  customInstructions: string;

  // LLM parameters
  temperature: number; // 0.0 - 1.0
  maxTokens: number;

  // Personality preset
  personality: 'helpful' | 'technical' | 'beginner-friendly' | 'concise' | 'custom';

  // Feature toggles
  enabledSkills: {
    explainPanel: boolean;
    generateQuery: boolean;
    troubleshooting: boolean;
    searchContext: boolean; // Vector search for semantic context
    functionCalling: boolean; // Tool/function calling support
  };

  // UI preferences
  showContextBadge: boolean;
  showCostInfo: boolean;
  autoOpenOnDashboard: boolean; // Auto-open floating chat when viewing dashboards

  // LLM backend mode (defined in plugin settings)
  // This is read-only from frontend perspective, set by admin in plugin config
  llmBackend?: 'backend-proxy' | 'grafana-llm' | 'direct';
}

/**
 * Helper to get the full system prompt (base + custom)
 */
export function getFullSystemPrompt(config: ZagalinConfig): string {
  return `${BASE_SYSTEM_PROMPT}\n\n${config.customInstructions}`;
}

export const DEFAULT_CONFIG: ZagalinConfig = {
  customInstructions: `Balance clarity and detail. Explain dashboards, generate queries, troubleshoot issues. Use technical terms but explain them when needed. Context-aware and actionable.`,

  temperature: 0.7,
  maxTokens: 2000,
  personality: 'helpful',

  enabledSkills: {
    explainPanel: true,
    generateQuery: true,
    troubleshooting: true,
    searchContext: false, // Disabled by default, requires vector service
    functionCalling: true, // Enabled by default for rich interactions
  },

  showContextBadge: true,
  showCostInfo: true,
  autoOpenOnDashboard: false,

  llmBackend: 'grafana-llm', // Default to @grafana/llm (works immediately, no service account needed)
};

export const PERSONALITY_PRESETS: Record<string, string> = {
  helpful: `Balance clarity and detail. Explain dashboards, generate queries, troubleshoot issues. Use technical terms but explain them when needed. Context-aware and actionable.`,

  technical: `SRE-level communication. Assume expert knowledge of Prometheus, Loki, Tempo, PromQL, LogQL, TraceQL. Skip basics. Focus on optimization, advanced patterns, and edge cases.`,

  'beginner-friendly': `Educational and patient. Define technical terms, use analogies, explain the "why" behind suggestions. Break down complex concepts step by step. Make observability approachable.`,

  concise: `Minimal words, maximum value. Lead with the answer. Use bullet points and code blocks. Skip explanations unless asked.`,

  custom: DEFAULT_CONFIG.customInstructions,
};
