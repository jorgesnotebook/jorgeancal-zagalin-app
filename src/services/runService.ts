/**
 * Run Service - Client for run-based orchestration API
 */

import { Observable } from 'rxjs';
import { AssistantMessage, AssistantContext } from './assistantService';

export interface RunStartRequest {
  conversationId: string;
  message: string;
  history: AssistantMessage[];
  context: AssistantContext;
  attachments?: Attachment[];
}

export interface Attachment {
  type: string;
  source: string;
  dashboardUid?: string;
  panelId?: number;
  timeRange?: { from: string; to: string };
  variables?: Record<string, any>;
  links?: Record<string, string>;
}

export interface RunStartResponse {
  runId: string;
}

export interface RunState {
  runId: string;
  conversationId: string;
  status: 'pending' | 'planning' | 'executing' | 'paused' | 'completed' | 'cancelled' | 'failed';
  plan?: ExecutionPlan;
  currentStepIndex: number;
  artifacts: Artifact[];
  createdAt: string;
  updatedAt: string;
}

export interface ExecutionPlan {
  goal: string;
  steps: PlannedStep[];
  estimatedDuration: string;
}

export interface PlannedStep {
  index: number;
  title: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
}

export interface Artifact {
  id: string;
  type: 'query' | 'link' | 'trace_id' | 'dashboard_link' | 'tool_call';
  content: string;
  metadata: Record<string, any>;
  timestamp: string;
}

export interface RunEvent {
  type: string;
  runId: string;
  timestamp: string;
  data: any;
}

const BASE_URL = '/api/plugins/jorgeancal-zagalin-app/resources';

/**
 * Start a new run
 */
export async function startRun(request: RunStartRequest): Promise<RunStartResponse> {
  const response = await fetch(`${BASE_URL}/runs/start`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to start run: ${response.status} - ${error}`);
  }

  return response.json();
}

/**
 * Stream SSE events for a run
 */
export function streamRunEvents(runId: string): Observable<RunEvent> {
  return new Observable<RunEvent>((subscriber) => {
    const url = `${BASE_URL}/runs/${runId}/events`;

    fetch(url, {
      method: 'GET',
      headers: {
        Accept: 'text/event-stream',
      },
      credentials: 'same-origin',
    })
      .then(async (response) => {
        if (!response.ok) {
          const error = await response.text();
          throw new Error(`Failed to stream events: ${response.status} - ${error}`);
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
              return;
            }

            const text = decoder.decode(value, { stream: true });

            // Parse SSE lines
            const lines = text.split('\n');
            for (const line of lines) {
              if (!line.trim() || !line.startsWith('data: ')) {
                continue;
              }

              const data = line.substring(6); // Remove "data: " prefix

              // Check for done marker
              if (data === '[DONE]') {
                subscriber.complete();
                return;
              }

              try {
                const event = JSON.parse(data) as RunEvent;
                subscriber.next(event);
              } catch (parseError) {
                console.warn('Failed to parse SSE event:', data, parseError);
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
  });
}

/**
 * Pause a run
 */
export async function pauseRun(runId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/runs/${runId}/pause`, {
    method: 'POST',
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to pause run: ${response.status} - ${error}`);
  }
}

/**
 * Resume a run
 */
export async function resumeRun(runId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/runs/${runId}/resume`, {
    method: 'POST',
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to resume run: ${response.status} - ${error}`);
  }
}

/**
 * Cancel a run
 */
export async function cancelRun(runId: string): Promise<void> {
  const response = await fetch(`${BASE_URL}/runs/${runId}/cancel`, {
    method: 'POST',
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to cancel run: ${response.status} - ${error}`);
  }
}

/**
 * Get run status
 */
export async function getRunStatus(runId: string): Promise<RunState> {
  const response = await fetch(`${BASE_URL}/runs/${runId}/status`, {
    method: 'GET',
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to get run status: ${response.status} - ${error}`);
  }

  return response.json();
}
