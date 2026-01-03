import { PLUGIN_ID } from '../constants';

export function getPluginResourcePath(): string {
  return `/api/plugins/${PLUGIN_ID}/resources`;
}

export function getPluginApiUrl(endpoint: string): string {
  const base = getPluginResourcePath();
  const path = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  return `${base}${path}`;
}
