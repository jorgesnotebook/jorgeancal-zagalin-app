/**
 * Hook to load and use Zagalin configuration
 * Loads org-wide configuration from Grafana's plugin settings (database-backed)
 */

import { useState, useEffect } from 'react';
import { getBackendSrv } from '@grafana/runtime';
import { ZagalinConfig, DEFAULT_CONFIG } from '../types/zagalinConfig';
import { updateCachedConfig } from '../services/configHelper';

const PLUGIN_ID = 'jorgeancal-zagalin-app';

export function useZagalinConfig() {
  const [config, setConfig] = useState<ZagalinConfig>(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadConfig = async () => {
      try {
        const pluginSettings = await getBackendSrv().get(`/api/plugins/${PLUGIN_ID}/settings`);

        if (pluginSettings?.jsonData) {
          const newConfig = { ...DEFAULT_CONFIG, ...pluginSettings.jsonData };
          setConfig(newConfig);
          updateCachedConfig(newConfig);
        }
      } catch (e: any) {
        console.error('Failed to load Zagalin config from plugin settings:', e);
      } finally {
        setLoading(false);
      }
    };

    loadConfig();

    const interval = setInterval(loadConfig, 30000);
    return () => clearInterval(interval);
  }, []);

  return { config, loading };
}
