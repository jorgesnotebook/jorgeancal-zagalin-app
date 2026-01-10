/**
 * Config Helper
 * Provides synchronous access to Zagalin configuration by caching it in localStorage.
 * This allows services to access config without async/await, which is needed since
 * some services must return immediately (e.g., Observable-based services).
 */

import { ZagalinConfig, DEFAULT_CONFIG } from '../types/zagalinConfig';

let cachedConfig: ZagalinConfig | null = null;

/**
 * Get Zagalin config synchronously
 * This reads from cached value populated by useZagalinConfig hook
 * Falls back to localStorage if cache is not populated
 */
export function getZagalinConfig(): ZagalinConfig {
  if (cachedConfig) {
    return cachedConfig;
  }

  try {
    const stored = localStorage.getItem('zagalin-config-cache');
    if (stored) {
      const parsed = JSON.parse(stored);
      return { ...DEFAULT_CONFIG, ...parsed };
    }
  } catch (e) {
    console.warn('Failed to read cached config:', e);
  }

  return DEFAULT_CONFIG;
}

/**
 * Update cached config (called by useZagalinConfig hook)
 * This keeps the in-memory cache and localStorage in sync
 */
export function updateCachedConfig(config: ZagalinConfig): void {
  cachedConfig = config;

  try {
    localStorage.setItem('zagalin-config-cache', JSON.stringify(config));
  } catch (e) {
    console.warn('Failed to cache config in localStorage:', e);
  }
}

/**
 * Clear cached config (useful for testing or logout scenarios)
 */
export function clearCachedConfig(): void {
  cachedConfig = null;
  try {
    localStorage.removeItem('zagalin-config-cache');
  } catch (e) {
    console.warn('Failed to clear cached config:', e);
  }
}
