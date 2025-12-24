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

  try {
    // Dynamic import to avoid module loading errors
    const { llm, vector } = await import('@grafana/llm');

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
  } catch (importErr) {
    status.llm.error = importErr instanceof Error ? importErr.message : 'Failed to load @grafana/llm';
    status.vector.error = 'Module not available';
  }

  return status;
}

/**
 * Quick check if LLM is ready to use
 */
export async function isLLMReady(): Promise<boolean> {
  try {
    const { llm } = await import('@grafana/llm');
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
    const { vector } = await import('@grafana/llm');
    return await vector.enabled();
  } catch (err) {
    return false;
  }
}
