/**
 * Hook to load and use Zagalin configuration
 * Loads org-wide configuration from Grafana's plugin settings (database-backed)
 */

import { useState, useEffect } from 'react';
import { getBackendSrv } from '@grafana/runtime';
import { ZagalinConfig, DEFAULT_CONFIG } from '../types/zagalinConfig';

const PLUGIN_ID = 'jorgeancal-zagalin-app';

export function useZagalinConfig() {
  const [config, setConfig] = useState<ZagalinConfig>(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadConfig = async () => {
      try {
        // Fetch plugin settings from Grafana API (stored in database)
        const pluginSettings = await getBackendSrv().get(`/api/plugins/${PLUGIN_ID}/settings`);

        if (pluginSettings?.jsonData) {
          setConfig({ ...DEFAULT_CONFIG, ...pluginSettings.jsonData });
        }
      } catch (e: any) {
        console.error('Failed to load Zagalin config from plugin settings:', e);
        // Fall back to defaults if plugin settings not available
      } finally {
        setLoading(false);
      }
    };

    loadConfig();

    // Poll for config changes every 30 seconds
    // This ensures users see updates when admins change settings
    const interval = setInterval(loadConfig, 30000);
    return () => clearInterval(interval);
  }, []);

  return { config, loading };
}
