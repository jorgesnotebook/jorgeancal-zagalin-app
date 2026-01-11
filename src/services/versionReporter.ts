import { getBackendSrv } from '@grafana/runtime';
import { getGrafanaVersion } from './versionDetector';

/**
 * Adds X-Grafana-Version header to request headers
 *
 * The backend uses this header for optional version detection and logging.
 * Only includes the header if version is available (respects disabled version reporting).
 *
 * @param headers - Existing request headers
 * @returns Headers with X-Grafana-Version added (if version available)
 */
export function addVersionHeader(headers: Record<string, string> = {}): Record<string, string> {
  const version = getGrafanaVersion();

  // Only send if version is available (respects disabled reporting)
  if (version.isAvailable) {
    return {
      ...headers,
      'X-Grafana-Version': version.full,
    };
  }

  return headers;
}

/**
 * Enhanced getBackendSrv that automatically includes version header
 *
 * Use this instead of getBackendSrv() for plugin backend API calls
 * to enable version detection in the backend.
 *
 * @returns BackendSrv instance with version header injection
 */
export function getVersionAwareBackendSrv() {
  const backendSrv = getBackendSrv();

  // Wrap the request methods to inject version header
  return {
    ...backendSrv,

    get: async <T = any>(url: string, params?: any, options?: any): Promise<T> => {
      const headers = addVersionHeader(options?.headers);
      return backendSrv.get<T>(url, params, { ...options, headers });
    },

    post: async <T = any>(url: string, data?: any, options?: any): Promise<T> => {
      const headers = addVersionHeader(options?.headers);
      return backendSrv.post<T>(url, data, { ...options, headers });
    },

    patch: async <T = any>(url: string, data?: any, options?: any): Promise<T> => {
      const headers = addVersionHeader(options?.headers);
      return backendSrv.patch<T>(url, data, { ...options, headers });
    },

    put: async <T = any>(url: string, data?: any, options?: any): Promise<T> => {
      const headers = addVersionHeader(options?.headers);
      return backendSrv.put<T>(url, data, { ...options, headers });
    },

    delete: async <T = any>(url: string, options?: any): Promise<T> => {
      const headers = addVersionHeader(options?.headers);
      return backendSrv.delete<T>(url, { ...options, headers });
    },
  };
}
