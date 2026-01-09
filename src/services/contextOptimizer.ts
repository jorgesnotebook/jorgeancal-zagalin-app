import type { GrafanaContext } from './contextTypes';

export interface OptimizedContext {
  essential: string;
  supplemental: string;
  metadata: Record<string, any>;
}

/**
 * Estimate the number of tokens in a string
 * Rough estimation: 1 token ≈ 4 characters
 */
function estimateTokens(text: any): number {
  const str = typeof text === 'string' ? text : JSON.stringify(text);
  return Math.ceil(str.length / 4);
}

/**
 * Format full context as a string
 */
function formatFullContext(context: GrafanaContext): string {
  const parts: string[] = [];

  if (context.dashboard) {
    parts.push(`Dashboard: ${context.dashboard.title}`);
    if (context.dashboard.uid) {
      parts.push(`UID: ${context.dashboard.uid}`);
    }
    if (context.dashboard.tags && context.dashboard.tags.length > 0) {
      parts.push(`Tags: ${context.dashboard.tags.join(', ')}`);
    }
  }

  if (context.panel) {
    parts.push(`\nPanel: ${context.panel.title}`);
    if (context.panel.description) {
      parts.push(`Description: ${context.panel.description}`);
    }
  }

  if (context.timeRange) {
    parts.push(`\nTime Range: ${context.timeRange.from} to ${context.timeRange.to}`);
  }

  return parts.join('\n');
}

/**
 * Optimize context to fit within token budget
 * @param fullContext - The complete Grafana context
 * @param maxTokens - Maximum tokens to use for context (default: 1000)
 * @returns Optimized context with essential and supplemental parts
 */
export function optimizeContext(
  fullContext: GrafanaContext,
  maxTokens = 1000
): OptimizedContext {
  const estimated = estimateTokens(fullContext);

  if (estimated <= maxTokens) {
    return {
      essential: formatFullContext(fullContext),
      supplemental: '',
      metadata: {},
    };
  }

  const essential = {
    dashboard: fullContext.dashboard?.title,
    dashboardUid: fullContext.dashboard?.uid,
    panel: fullContext.panel?.title,
    timeRange: fullContext.timeRange
      ? `${fullContext.timeRange.from} to ${fullContext.timeRange.to}`
      : undefined,
  };

  const supplemental = {
    dashboardTags: fullContext.dashboard?.tags,
    panelDescription: fullContext.panel?.description,
  };

  return {
    essential: JSON.stringify(essential, null, 2),
    supplemental: JSON.stringify(supplemental, null, 2),
    metadata: {},
  };
}

/**
 * Calculate optimal context budget based on total max tokens
 * Generally use 20-30% of total budget for context
 */
export function calculateContextBudget(maxTokens: number): number {
  return Math.floor(maxTokens * 0.25);
}
