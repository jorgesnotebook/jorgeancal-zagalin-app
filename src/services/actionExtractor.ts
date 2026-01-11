/**
 * Action Extractor - Parses assistant responses for actionable items
 * Extracts queries, links, and other actionable content from LLM responses
 */

import type { AssistantAction, TimeRange } from './contextTypes';

export function extractQueries(content: string): AssistantAction[] {
  const actions: AssistantAction[] = [];

  const codeBlockRegex = /```(?:promql|logql|traceql|prometheus|loki|tempo)?\s*\n([\s\S]*?)\n```/gi;

  let match;
  let queryIndex = 1;

  while ((match = codeBlockRegex.exec(content)) !== null) {
    const query = match[1].trim();

    if (query) {
      actions.push({
        type: 'query',
        label: `Query ${queryIndex}`,
        data: {
          query,
        },
      });
      queryIndex++;
    }
  }

  return actions;
}

export function createExploreLink(query: string, datasourceUid?: string, timeRange?: TimeRange): string {
  const params = new URLSearchParams();

  const exploreQuery: any = {
    queries: [
      {
        refId: 'A',
        expr: query,
        datasource: datasourceUid ? { uid: datasourceUid } : undefined,
      },
    ],
    range: timeRange
      ? {
          from: timeRange.from,
          to: timeRange.to,
        }
      : undefined,
  };

  params.set('left', JSON.stringify(exploreQuery));

  return `/explore?${params.toString()}`;
}

export function extractActions(content: string): AssistantAction[] {
  const actions: AssistantAction[] = [];

  actions.push(...extractQueries(content));

  return actions;
}
