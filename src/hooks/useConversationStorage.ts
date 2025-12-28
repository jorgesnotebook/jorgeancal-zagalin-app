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

import { usePluginUserStorage } from '@grafana/runtime';
import { StorageBackend } from '../services/conversationStorage';

/**
 * Hook that provides a storage backend for conversation storage
 * Wraps Grafana's usePluginUserStorage() to provide automatic fallback
 *
 * Note: Grafana's PluginUserStorage doesn't have removeItem,
 * so we implement it by setting the value to empty string.
 */
export function useConversationStorage(): StorageBackend {
  const pluginStorage = usePluginUserStorage();

  return {
    getItem: (key: string) => {
      return pluginStorage.getItem(key);
    },
    setItem: (key: string, value: string) => {
      return pluginStorage.setItem(key, value);
    },
    removeItem: async (key: string) => {
      // PluginUserStorage doesn't have removeItem, so we clear by setting empty string
      await pluginStorage.setItem(key, '');
    },
  };
}
