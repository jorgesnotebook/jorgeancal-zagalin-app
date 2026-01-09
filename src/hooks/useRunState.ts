/**
 * useRunState - React hook for managing run lifecycle
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { Subscription } from 'rxjs';
import {
  startRun,
  streamRunEvents,
  pauseRun,
  resumeRun,
  cancelRun,
  ExecutionPlan,
  Artifact,
  RunStartRequest,
} from '../services/runService';
import { AssistantMessage, AssistantContext } from '../services/assistantService';

export type RunStatus = 'pending' | 'planning' | 'executing' | 'paused' | 'completed' | 'cancelled' | 'failed';

export interface UseRunStateOptions {
  conversationId: string;
  onComplete?: (finalMessage: string, artifacts: Artifact[]) => void;
  onError?: (error: string) => void;
}

export interface UseRunStateReturn {
  runId: string | null;
  status: RunStatus;
  plan: ExecutionPlan | null;
  currentStepIndex: number;
  artifacts: Artifact[];
  streamingText: string;
  isRunning: boolean;
  isPaused: boolean;
  error: string | null;

  start: (message: string, history: AssistantMessage[], context: AssistantContext) => Promise<void>;
  pause: () => Promise<void>;
  resume: () => Promise<void>;
  cancel: () => Promise<void>;
}

export function useRunState({ conversationId, onComplete, onError }: UseRunStateOptions): UseRunStateReturn {
  const [runId, setRunId] = useState<string | null>(null);
  const [status, setStatus] = useState<RunStatus>('pending');
  const [plan, setPlan] = useState<ExecutionPlan | null>(null);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [artifacts, setArtifacts] = useState<Artifact[]>([]);
  const [streamingText, setStreamingText] = useState('');
  const [error, setError] = useState<string | null>(null);

  const subscriptionRef = useRef<Subscription | null>(null);
  const artifactsRef = useRef<Artifact[]>([]);

  useEffect(() => {
    artifactsRef.current = artifacts;
  }, [artifacts]);

  const start = useCallback(
    async (message: string, history: AssistantMessage[], context: AssistantContext) => {
      try {
        setError(null);
        setStreamingText('');
        setArtifacts([]);
        setPlan(null);
        setCurrentStepIndex(0);
        artifactsRef.current = [];

        const request: RunStartRequest = {
          conversationId,
          message,
          history,
          context,
        };

        const response = await startRun(request);
        setRunId(response.runId);
        setStatus('pending');

        subscriptionRef.current = streamRunEvents(response.runId).subscribe({
          next: (event) => {
            switch (event.type) {
              case 'run_started':
                setStatus('planning');
                break;

              case 'plan':
                setPlan(event.data as ExecutionPlan);
                setStatus('executing');
                break;

              case 'step_started':
                setCurrentStepIndex(event.data.stepIndex);
                setStreamingText('');
                break;

              case 'artifact':
                const artifact = event.data as Artifact;
                setArtifacts((prev) => [...prev, artifact]);
                artifactsRef.current = [...artifactsRef.current, artifact];
                break;

              case 'assistant_delta':
                setStreamingText((prev) => prev + event.data.delta);
                break;

              case 'step_done':
                break;

              case 'assistant_message':
                const finalMessage = event.data.text;
                if (onComplete) {
                  onComplete(finalMessage, artifactsRef.current);
                }
                break;

              case 'final':
                setStatus(event.data.status);
                setStreamingText('');
                break;

              case 'paused':
                setStatus('paused');
                break;

              case 'resumed':
                setStatus('executing');
                break;

              case 'cancelled':
                setStatus('cancelled');
                setStreamingText('');
                break;

              case 'error':
                setError(event.data.message);
                if (onError) {
                  onError(event.data.message);
                }
                break;
            }
          },
          error: (err) => {
            setError(err.message);
            setStatus('failed');
            if (onError) {
              onError(err.message);
            }
          },
          complete: () => {
          },
        });
      } catch (err: any) {
        setError(err.message);
        setStatus('failed');
        if (onError) {
          onError(err.message);
        }
      }
    },
    [conversationId, onComplete, onError]
  );

  const pause = useCallback(async () => {
    if (runId) {
      try {
        await pauseRun(runId);
      } catch (err: any) {
        setError(err.message);
      }
    }
  }, [runId]);

  const resume = useCallback(async () => {
    if (runId) {
      try {
        await resumeRun(runId);
      } catch (err: any) {
        setError(err.message);
      }
    }
  }, [runId]);

  const cancel = useCallback(async () => {
    if (runId) {
      try {
        await cancelRun(runId);
        subscriptionRef.current?.unsubscribe();
      } catch (err: any) {
        setError(err.message);
      }
    }
  }, [runId]);

  useEffect(() => {
    return () => {
      subscriptionRef.current?.unsubscribe();
    };
  }, []);

  return {
    runId,
    status,
    plan,
    currentStepIndex,
    artifacts,
    streamingText,
    isRunning: status === 'executing' || status === 'planning',
    isPaused: status === 'paused',
    error,
    start,
    pause,
    resume,
    cancel,
  };
}
