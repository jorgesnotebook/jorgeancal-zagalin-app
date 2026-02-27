/**
 * React hook for conversation management
 * Provides easy access to conversation storage and state management
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  ConversationStorage,
  type Conversation,
  type ConversationMessage,
  type ConversationMetadata,
  type ConversationContext,
} from '../services/conversationStorage';
import { useConversationStorage } from './useConversationStorage';
import type { GrafanaContext } from '../services/contextTypes';

export interface UseConversationReturn {
  conversation: Conversation | null;
  messages: ConversationMessage[];

  conversations: ConversationMetadata[];

  createNew: (context?: GrafanaContext) => void;
  loadConversation: (id: string) => void;
  addMessage: (message: ConversationMessage, context?: GrafanaContext) => void;
  replaceMessages: (messages: ConversationMessage[]) => Promise<void>;
  addContext: (context: GrafanaContext) => Promise<void>;
  removeContext: (dashboardUid: string) => Promise<void>;
  deleteConversation: (id: string) => void;
  deleteAll: () => void;
  pruneByAge: (retentionDays: number) => Promise<number>;
  updateTitle: (id: string, title: string) => void;
  togglePin: (id: string) => void;
  clearCurrent: () => void;

  isLoading: boolean;
  currentId: string | null;
}

/**
 * Helper to convert GrafanaContext to ConversationContext
 */
function grafanaContextToConversationContext(grafanaContext?: GrafanaContext): ConversationContext | undefined {
  if (!grafanaContext?.dashboard?.uid) {
    return undefined;
  }

  return {
    dashboardUid: grafanaContext.dashboard.uid,
    dashboardTitle: grafanaContext.dashboard.title || 'Unknown Dashboard',
    panelId: grafanaContext.panel?.id,
    panelTitle: grafanaContext.panel?.title,
    timeFrom: grafanaContext.timeRange?.from,
    timeTo: grafanaContext.timeRange?.to,
    addedAt: new Date(),
  };
}

export function useConversation(): UseConversationReturn {
  const storage = useConversationStorage();
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [conversations, setConversations] = useState<ConversationMetadata[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const conversationRef = useRef<Conversation | null>(null);

  useEffect(() => {
    conversationRef.current = conversation;
  }, [conversation]);

  const refreshConversationList = useCallback(async () => {
    const list = await ConversationStorage.getConversationList(storage);
    setConversations(list);
  }, [storage]);

  useEffect(() => {
    const initConversations = async () => {
      await refreshConversationList();

      if (!conversationRef.current) {
        const list = await ConversationStorage.getConversationList(storage);
        if (list.length > 0) {
          const mostRecent = list[0];
          console.log('[useConversation] Auto-loading most recent conversation:', mostRecent.id);
          const conv = await ConversationStorage.getConversation(storage, mostRecent.id);
          if (conv) {
            setConversation(conv);
            conversationRef.current = conv;
          }
        }
      }
    };

    initConversations();
  }, [refreshConversationList, storage]);

  /**
   * Create a new conversation
   */
  const createNew = useCallback(
    async (grafanaContext?: GrafanaContext) => {
      const context = grafanaContextToConversationContext(grafanaContext);

      const newConv = await ConversationStorage.createConversation(storage, context);
      setConversation(newConv);
      conversationRef.current = newConv;
      await ConversationStorage.saveConversation(storage, newConv);
      await refreshConversationList();
    },
    [refreshConversationList, storage]
  );

  /**
   * Load an existing conversation
   */
  const loadConversation = useCallback(
    async (id: string) => {
      setIsLoading(true);
      try {
        const conv = await ConversationStorage.getConversation(storage, id);
        if (conv) {
          setConversation(conv);
          conversationRef.current = conv;
        } else {
          console.warn(`Conversation ${id} not found`);
        }
      } catch (error) {
        console.error('Failed to load conversation:', error);
      } finally {
        setIsLoading(false);
      }
    },
    [storage]
  );

  /**
   * Add a message to the current conversation
   */
  const addMessage = useCallback(
    async (message: ConversationMessage, grafanaContext?: GrafanaContext) => {
      const currentConv = conversationRef.current;

      console.log('[useConversation] addMessage called. Current conversation:', currentConv?.id);
      console.log('[useConversation] Current message count:', currentConv?.messages?.length || 0);

      if (!currentConv) {
        console.log('[useConversation] No active conversation. Creating new one with context.');

        const context = grafanaContextToConversationContext(grafanaContext);

        const newConv = await ConversationStorage.createConversation(storage, context);
        const updatedConv = {
          ...newConv,
          messages: [message],
        };
        console.log('[useConversation] Created new conversation:', updatedConv.id);
        setConversation(updatedConv);
        conversationRef.current = updatedConv;
        await ConversationStorage.saveConversation(storage, updatedConv);
        await refreshConversationList();
        console.log('[useConversation] New conversation saved');
        return;
      }

      console.log('[useConversation] Adding to existing conversation:', currentConv.id);

      const updated = {
        ...currentConv,
        messages: [...currentConv.messages, message],
        updatedAt: new Date(),
      };

      if (currentConv.messages.length === 0 && message.role === 'user') {
        updated.title = message.content.slice(0, 50);
        if (message.content.length > 50) {
          updated.title += '...';
        }
      }

      setConversation(updated);
      conversationRef.current = updated;
      await ConversationStorage.saveConversation(storage, updated);
      await refreshConversationList();
    },
    [refreshConversationList, storage]
  );

  /**
   * Replace all messages in the current conversation (used after context-window summarization).
   * Does not refresh the conversation list since no new conversation is created.
   */
  const replaceMessages = useCallback(
    async (newMessages: ConversationMessage[]) => {
      const currentConv = conversationRef.current;
      if (!currentConv) {
        console.warn('[useConversation] replaceMessages: no active conversation');
        return;
      }

      const updated = {
        ...currentConv,
        messages: newMessages,
        updatedAt: new Date(),
      };

      setConversation(updated);
      conversationRef.current = updated;
      await ConversationStorage.saveConversation(storage, updated);
    },
    [storage]
  );

  /**
   * Delete a conversation
   */
  const deleteConversation = useCallback(
    async (id: string) => {
      console.log('[useConversation] Deleting conversation:', id);
      try {
        await ConversationStorage.deleteConversation(storage, id);

        if (conversation?.id === id) {
          setConversation(null);
          conversationRef.current = null;
        }

        await refreshConversationList();
        console.log('[useConversation] Conversation deleted successfully');
      } catch (error) {
        console.error('[useConversation] Failed to delete conversation:', error);
        throw error;
      }
    },
    [conversation, refreshConversationList, storage]
  );

  /**
   * Delete all conversations
   */
  const deleteAll = useCallback(async () => {
    console.log('[useConversation] Deleting all conversations');
    try {
      await ConversationStorage.clearAll(storage);
      setConversation(null);
      conversationRef.current = null;
      await refreshConversationList();
      console.log('[useConversation] All conversations deleted successfully');
    } catch (error) {
      console.error('[useConversation] Failed to delete all conversations:', error);
      throw error;
    }
  }, [refreshConversationList, storage]);

  /**
   * Update conversation title
   */
  const updateTitle = useCallback(
    async (id: string, title: string) => {
      const sanitizedTitle = title.trim().slice(0, 200);

      if (sanitizedTitle.length === 0) {
        console.warn('Title cannot be empty');
        return;
      }

      if (/[\x00-\x08\x0B\x0C\x0E-\x1F]/.test(sanitizedTitle)) {
        console.warn('Title contains invalid characters');
        return;
      }

      await ConversationStorage.updateTitle(storage, id, sanitizedTitle);

      if (conversation?.id === id) {
        setConversation({
          ...conversation,
          title: sanitizedTitle,
        });
      }

      await refreshConversationList();
    },
    [conversation, refreshConversationList, storage]
  );

  /**
   * Toggle pin status
   */
  const togglePin = useCallback(
    async (id: string) => {
      await ConversationStorage.togglePin(storage, id);

      if (conversation?.id === id) {
        setConversation({
          ...conversation,
          isPinned: !conversation.isPinned,
        });
      }

      await refreshConversationList();
    },
    [conversation, refreshConversationList, storage]
  );

  /**
   * Delete conversations older than retentionDays, skipping pinned ones.
   */
  const pruneByAge = useCallback(
    async (retentionDays: number): Promise<number> => {
      const removed = await ConversationStorage.pruneByAge(storage, retentionDays);
      if (removed > 0) {
        await refreshConversationList();
      }
      return removed;
    },
    [refreshConversationList, storage]
  );

  /**
   * Clear the current conversation (start fresh)
   */
  const clearCurrent = useCallback(() => {
    setConversation(null);
    conversationRef.current = null;
  }, []);

  /**
   * Add a new context (dashboard) to the current conversation
   */
  const addContext = useCallback(
    async (grafanaContext: GrafanaContext) => {
      const currentConv = conversationRef.current;
      if (!currentConv) {
        console.warn('[useConversation] No active conversation to add context to');
        return;
      }

      const newContext = grafanaContextToConversationContext(grafanaContext);
      if (!newContext) {
        console.warn('[useConversation] Invalid context - missing dashboard UID');
        return;
      }

      const exists = currentConv.contexts.some(
        (ctx) => ctx.dashboardUid === newContext.dashboardUid && ctx.panelId === newContext.panelId
      );

      if (exists) {
        console.log('[useConversation] Context already attached:', newContext.dashboardUid);
        return;
      }

      console.log('[useConversation] Adding context to conversation:', newContext.dashboardUid);

      const updated = {
        ...currentConv,
        contexts: [...currentConv.contexts, newContext],
        updatedAt: new Date(),
      };

      setConversation(updated);
      conversationRef.current = updated;
      await ConversationStorage.saveConversation(storage, updated);
      console.log('[useConversation] Context added successfully');
    },
    [storage]
  );

  /**
   * Remove a context (dashboard) from the current conversation
   */
  const removeContext = useCallback(
    async (dashboardUid: string) => {
      const currentConv = conversationRef.current;
      if (!currentConv) {
        console.warn('[useConversation] No active conversation to remove context from');
        return;
      }

      console.log('[useConversation] Removing context from conversation:', dashboardUid);

      const updated = {
        ...currentConv,
        contexts: currentConv.contexts.filter((ctx) => ctx.dashboardUid !== dashboardUid),
        updatedAt: new Date(),
      };

      setConversation(updated);
      conversationRef.current = updated;
      await ConversationStorage.saveConversation(storage, updated);
      console.log('[useConversation] Context removed successfully');
    },
    [storage]
  );

  return {
    conversation,
    messages: conversation?.messages || [],
    conversations,
    createNew,
    loadConversation,
    addMessage,
    replaceMessages,
    addContext,
    removeContext,
    deleteConversation,
    deleteAll,
    pruneByAge,
    updateTitle,
    togglePin,
    clearCurrent,
    isLoading,
    currentId: conversation?.id || null,
  };
}
