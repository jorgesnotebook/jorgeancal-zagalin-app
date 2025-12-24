/**
 * React hook for accessing Grafana context
 */

import { useState, useEffect } from 'react';
import { locationService } from '@grafana/runtime';
import { ContextService } from './contextService';
import type { GrafanaContext } from './contextTypes';

export function useGrafanaContext() {
  const [context, setContext] = useState<GrafanaContext>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const refreshContext = async () => {
    setLoading(true);
    setError(null);
    try {
      const newContext = await ContextService.getContext();
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

    // Listen for location changes to update context
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
