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
 * LLM Status Cache (module-level)
 * Simple but bomb-proof: handles race conditions, caches results, fails gracefully
 */
let llmStatus: boolean | null = null;
let llmStatusExpiry = 0;
let pendingCheck: Promise<boolean> | null = null;
const LLM_CACHE_TTL = 30000; // 30 seconds

/**
 * Quick check if LLM is ready to use (with caching and deduplication)
 */
export async function isLLMReady(): Promise<boolean> {
  const now = Date.now();

  // Return cached status if still valid
  if (llmStatus !== null && now < llmStatusExpiry) {
    return llmStatus;
  }

  // If a check is already in progress, wait for it (prevents race conditions)
  if (pendingCheck) {
    return pendingCheck;
  }

  // Start new check
  pendingCheck = (async () => {
    try {
      const { llm } = await import('@grafana/llm');
      const result = await llm.enabled();

      // Cache the result
      llmStatus = result;
      llmStatusExpiry = now + LLM_CACHE_TTL;

      return result;
    } catch (err) {
      console.error('LLM health check failed:', err);
      return false;
    } finally {
      // Clear pending check
      pendingCheck = null;
    }
  })();

  return pendingCheck;
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
