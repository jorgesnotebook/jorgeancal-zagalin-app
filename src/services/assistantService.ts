import { Observable } from 'rxjs';
import { getPluginApiUrl } from './pluginUrl';

export interface AssistantMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
}

export interface AssistantContext {
  dashboard?: DashboardContext;
  panel?: PanelContext;
  timeRange?: TimeRange;
  templateVars?: TemplateVariable[];
  evidencePacks?: EvidencePack[];
}

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

export interface EvidencePack {
  type: string;
  datasource: string;
  timeRange: TimeRange;
  query: string;
  metrics?: MetricsEvidence;
  logs?: LogsEvidence;
  traces?: TracesEvidence;
  quality: string;
}

export interface MetricsEvidence {
  unit: string;
  seriesCount: number;
  current: number;
  min: number;
  max: number;
  avg: number;
  trend: string;
  slopePerHour: number;
  quality: string;
  topContributors?: LabelContributor[];
}

export interface LabelContributor {
  labels: Record<string, string>;
  value: number;
}

export interface LogsEvidence {
  totalCount: number;
  rate: number;
  maxRate: number;
  trend: string;
  topLabels: Record<string, string[]>;
  notableMessages: string[];
}

export interface TracesEvidence {
  traceID: string;
  rootService: string;
  rootOperation: string;
  totalDuration: number;
  spanCount: number;
  errorSpanCount: number;
  topSlowestSpans: SlowSpan[];
  criticalPath: string[];
  notableAttributes: Record<string, string>;
}

export interface SlowSpan {
  service: string;
  operation: string;
  duration: number;
}

export interface AssistantRequest {
  message: string;
  history: AssistantMessage[];
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

export interface StreamChunk {
  chunk?: string;
  done?: boolean;
  error?: string;
  tool_call?: ToolCallChunk;
}

export interface ToolCallChunk {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

export function streamAssistantChat(request: AssistantRequest): Observable<StreamChunk> {
  return new Observable<StreamChunk>((subscriber) => {
    const url = getPluginApiUrl('/llm/chat');

    const backendRequest = {
      message: request.message,
      history: request.history,
      context: request.context,
      skillHint: request.skillHint,
      enrichedMessage: request.enrichedMessage,
      mode: request.mode || 'standard',
    };

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
      },
      body: JSON.stringify(backendRequest),
      credentials: 'same-origin',
    })
      .then(async (response) => {
        if (!response.ok) {
          const error = await response.text();
          throw new Error(`Error: ${response.status} - ${error}`);
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
              subscriber.complete();
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
                subscriber.complete();
                return;
              }

              try {
                const chunk = JSON.parse(data) as StreamChunk;
                subscriber.next(chunk);

                if (chunk.done || chunk.error) {
                  subscriber.complete();
                  return;
                }
              } catch (parseError) {
                console.warn('Failed to parse SSE chunk:', data, parseError);
              }
            }
          }
        } catch (readError) {
          subscriber.error(readError);
        }
      })
      .catch((fetchError) => {
        subscriber.error(fetchError);
      });

    return () => {};
  });
}
