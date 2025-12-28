/**
 * React hook that provides conversation storage using Grafana's User Storage API
 *
 * Uses usePluginUserStorage() which:
 * - Stores in Grafana DB (11.5+ with userStorageAPI feature flag)
 * - Automatically falls back to localStorage if flag disabled
 * - Per-user storage
 *
 * Example usage:
 * ```tsx
 * const storage = useConversationStorage();
 * const conversations = await ConversationStorage.getConversationList(storage);
 * ```
 */

import { StorageBackend } from '../services/conversationStorage';
import { useMemo } from 'react';

/**
 * LocalStorage fallback for when plugin context is not available
 */
const localStorageBackend: StorageBackend = {
  getItem: (key: string) => {
    return localStorage.getItem(key);
  },
  setItem: (key: string, value: string) => {
    localStorage.setItem(key, value);
  },
  removeItem: (key: string) => {
    localStorage.removeItem(key);
  },
};

/**
 * Hook that provides a storage backend for conversation storage
 *
 * Currently uses localStorage for all scenarios to ensure:
 * - Consistent storage between main app and floating chat
 * - No split-brain issues (conversations visible in both places)
 * - Works everywhere (no plugin context required)
 * - Grafana validator compliant
 *
 * Future: Can be enhanced to use Grafana's User Storage API when available,
 * with proper migration between localStorage and backend storage.
 */
export function useConversationStorage(): StorageBackend {
  // Always use localStorage for consistency between main app and floating chat
  // The floating chat is mounted globally (no plugin context), so it must use localStorage
  // To avoid split-brain, we use localStorage everywhere
  return useMemo(() => localStorageBackend, []);
}
