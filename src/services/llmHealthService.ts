import { llm, vector } from '@grafana/llm';

export interface HealthStatus {
  llm: {
    enabled: boolean;
    provider?: string;
    models?: string[];
    error?: string;
  };
  vector: {
    enabled: boolean;
    version?: string;
    error?: string;
  };
}

/**
 * Check the health status of both LLM and Vector services
 */
export async function checkZagalinHealth(): Promise<HealthStatus> {
  const status: HealthStatus = {
    llm: { enabled: false },
    vector: { enabled: false },
  };

  // Check LLM
  try {
    status.llm.enabled = await llm.enabled();
    if (status.llm.enabled) {
      const health: any = await llm.health();
      status.llm.provider = health.details?.llmProvider?.provider || health.llmProvider?.provider;
      status.llm.models = health.details?.llmProvider?.models || health.llmProvider?.models;
    }
  } catch (err) {
    status.llm.error = err instanceof Error ? err.message : 'Unknown error';
  }

  // Check Vector
  try {
    status.vector.enabled = await vector.enabled();
    if (status.vector.enabled) {
      const health: any = await vector.health();
      status.vector.version = health.version || 'unknown';
    }
  } catch (err) {
    status.vector.error = err instanceof Error ? err.message : 'Unknown error';
  }

  return status;
}

/**
 * Quick check if LLM is ready to use
 */
export async function isLLMReady(): Promise<boolean> {
  try {
    return await llm.enabled();
  } catch (err) {
    console.error('LLM health check failed:', err);
    return false;
  }
}

/**
 * Quick check if Vector search is ready to use
 */
export async function isVectorReady(): Promise<boolean> {
  try {
    return await vector.enabled();
  } catch (err) {
    return false;
  }
}
