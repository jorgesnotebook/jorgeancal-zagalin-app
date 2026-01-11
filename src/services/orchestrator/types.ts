/**
 * Type definitions for Orchestrator module
 *
 * TODO: Import and consolidate types from:
 * - frontendOrchestrator.ts
 * - frontendPrompts.ts
 * - runService.ts
 */

import type { GrafanaContext } from '../context/types';

export interface OrchestrationOptions {
  temperature?: number;
  maxTokens?: number;
  includeContext?: boolean;
  mode?: 'simple' | 'multi-step';
}

export interface MultiStepRequest {
  userMessage: string;
  context: GrafanaContext;
  steps: Step[];
}

export interface Step {
  id: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  result?: any;
}

export interface WorkflowUpdate {
  stepId: string;
  status: string;
  message: string;
  progress?: number;
}

export interface RunStatus {
  runId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  startedAt?: number;
  completedAt?: number;
  error?: string;
}

// TODO: Add more types from existing services
