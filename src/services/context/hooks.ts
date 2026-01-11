/**
 * React hooks for Context Management
 * Consolidated from useGrafanaContext.ts
 */

import { useState, useEffect } from 'react';
import { locationService } from '@grafana/runtime';
import { ContextManager } from './ContextManager';
import type { GrafanaContext } from './types';

/**
 * Hook to access Grafana context in React components
 * Automatically refreshes when location changes
 */
export function useGrafanaContext() {
  const [context, setContext] = useState<GrafanaContext>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const refreshContext = async () => {
    setLoading(true);
    setError(null);
    try {
      const manager = new ContextManager();
      const newContext = await manager.extractContext();
      setContext(newContext);
    } catch (err) {
      setError(err as Error);
      console.error('Failed to load Grafana context:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshContext();

    const unsubscribe = locationService.getHistory().listen(() => {
      refreshContext();
    });

    return () => {
      unsubscribe();
    };
  }, []);

  return {
    context,
    loading,
    error,
    refresh: refreshContext,
    hasContext: Boolean(context.dashboard || context.panel),
  };
}
