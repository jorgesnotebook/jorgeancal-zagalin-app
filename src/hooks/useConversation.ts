/**
 * React hook for conversation management
 * Provides easy access to conversation storage and state management
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  ConversationStorage,
  type Conversation,
  type ConversationMessage,
  type ConversationMetadata
} from '../services/conversationStorage';
import { useConversationStorage } from './useConversationStorage';
import type { GrafanaContext } from '../services/contextTypes';

export interface UseConversationReturn {
  // Current conversation
  conversation: Conversation | null;
  messages: ConversationMessage[];

  // Conversation list
  conversations: ConversationMetadata[];

  // Actions
  createNew: (context?: GrafanaContext) => void;
  loadConversation: (id: string) => void;
  addMessage: (message: ConversationMessage, context?: GrafanaContext) => void;
  deleteConversation: (id: string) => void;
  deleteAll: () => void;
  updateTitle: (id: string, title: string) => void;
  togglePin: (id: string) => void;
  clearCurrent: () => void;

  // State
  isLoading: boolean;
  currentId: string | null;
}

export function useConversation(): UseConversationReturn {
  const storage = useConversationStorage();
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [conversations, setConversations] = useState<ConversationMetadata[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const conversationRef = useRef<Conversation | null>(null);

  // Keep ref in sync with state
  useEffect(() => {
    conversationRef.current = conversation;
  }, [conversation]);

  // Refresh the conversation list
  const refreshConversationList = useCallback(async () => {
    const list = await ConversationStorage.getConversationList(storage);
    setConversations(list);
  }, [storage]);

  // Load conversation list on mount and auto-load most recent conversation
  useEffect(() => {
    const initConversations = async () => {
      await refreshConversationList();

      // If no active conversation, load the most recent one
      if (!conversationRef.current) {
        const list = await ConversationStorage.getConversationList(storage);
        if (list.length > 0) {
          // Load the most recent conversation (first in the sorted list)
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
  const createNew = useCallback(async (grafanaContext?: GrafanaContext) => {
    const context = grafanaContext ? {
      dashboardUid: grafanaContext.dashboard?.uid,
      dashboardTitle: grafanaContext.dashboard?.title,
      panelId: grafanaContext.panel?.id,
      panelTitle: grafanaContext.panel?.title,
      timeRange: grafanaContext.timeRange
    } : undefined;

    const newConv = await ConversationStorage.createConversation(storage, context);
    setConversation(newConv);
    conversationRef.current = newConv; // Update ref immediately
    await ConversationStorage.saveConversation(storage, newConv);
    await refreshConversationList();
  }, [refreshConversationList, storage]);

  /**
   * Load an existing conversation
   */
  const loadConversation = useCallback(async (id: string) => {
    setIsLoading(true);
    try {
      const conv = await ConversationStorage.getConversation(storage, id);
      if (conv) {
        setConversation(conv);
        conversationRef.current = conv; // Update ref too
      } else {
        console.warn(`Conversation ${id} not found`);
      }
    } catch (error) {
      console.error('Failed to load conversation:', error);
    } finally {
      setIsLoading(false);
    }
  }, [storage]);

  /**
   * Add a message to the current conversation
   */
  const addMessage = useCallback(async (message: ConversationMessage, grafanaContext?: GrafanaContext) => {
    // Use ref to get the LATEST conversation state, not the stale closure value
    const currentConv = conversationRef.current;

    console.log('[useConversation] addMessage called. Current conversation:', currentConv?.id);
    console.log('[useConversation] Current message count:', currentConv?.messages?.length || 0);

    if (!currentConv) {
      console.log('[useConversation] No active conversation. Creating new one with context.');

      const context = grafanaContext ? {
        dashboardUid: grafanaContext.dashboard?.uid,
        dashboardTitle: grafanaContext.dashboard?.title,
        panelId: grafanaContext.panel?.id,
        panelTitle: grafanaContext.panel?.title,
        timeRange: grafanaContext.timeRange
      } : undefined;

      const newConv = await ConversationStorage.createConversation(storage, context);
      const updatedConv = {
        ...newConv,
        messages: [message]
      };
      console.log('[useConversation] Created new conversation:', updatedConv.id);
      setConversation(updatedConv);
      conversationRef.current = updatedConv; // Update ref immediately
      await ConversationStorage.saveConversation(storage, updatedConv);
      await refreshConversationList();
      console.log('[useConversation] New conversation saved');
      return;
    }

    console.log('[useConversation] Adding to existing conversation:', currentConv.id);

    const updated = {
      ...currentConv,
      messages: [...currentConv.messages, message],
      updatedAt: new Date()
    };

    if (currentConv.messages.length === 0 && message.role === 'user') {
      updated.title = message.content.slice(0, 50);
      if (message.content.length > 50) {
        updated.title += '...';
      }
    }

    setConversation(updated);
    conversationRef.current = updated; // Update ref immediately
    await ConversationStorage.saveConversation(storage, updated);
    await refreshConversationList();
  }, [refreshConversationList, storage]);

  /**
   * Delete a conversation
   */
  const deleteConversation = useCallback(async (id: string) => {
    console.log('[useConversation] Deleting conversation:', id);
    try {
      await ConversationStorage.deleteConversation(storage, id);

      if (conversation?.id === id) {
        setConversation(null);
        conversationRef.current = null; // Clear the ref too
      }

      await refreshConversationList();
      console.log('[useConversation] Conversation deleted successfully');
    } catch (error) {
      console.error('[useConversation] Failed to delete conversation:', error);
      throw error;
    }
  }, [conversation, refreshConversationList, storage]);

  /**
   * Delete all conversations
   */
  const deleteAll = useCallback(async () => {
    console.log('[useConversation] Deleting all conversations');
    try {
      await ConversationStorage.clearAll(storage);
      setConversation(null);
      conversationRef.current = null; // Clear the ref too
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
  const updateTitle = useCallback(async (id: string, title: string) => {
    // Sanitize and validate title
    const sanitizedTitle = title.trim().slice(0, 200);

    if (sanitizedTitle.length === 0) {
      console.warn('Title cannot be empty');
      return;
    }

    // Check for control characters (except newline, carriage return, tab)
    if (/[\x00-\x08\x0B\x0C\x0E-\x1F]/.test(sanitizedTitle)) {
      console.warn('Title contains invalid characters');
      return;
    }

    await ConversationStorage.updateTitle(storage, id, sanitizedTitle);

    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        title: sanitizedTitle
      });
    }

    await refreshConversationList();
  }, [conversation, refreshConversationList, storage]);

  /**
   * Toggle pin status
   */
  const togglePin = useCallback(async (id: string) => {
    await ConversationStorage.togglePin(storage, id);

    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        isPinned: !conversation.isPinned
      });
    }

    await refreshConversationList();
  }, [conversation, refreshConversationList, storage]);

  /**
   * Clear the current conversation (start fresh)
   */
  const clearCurrent = useCallback(() => {
    setConversation(null);
    conversationRef.current = null; // Also clear the ref
  }, []);

  return {
    conversation,
    messages: conversation?.messages || [],
    conversations,
    createNew,
    loadConversation,
    addMessage,
    deleteConversation,
    deleteAll,
    updateTitle,
    togglePin,
    clearCurrent,
    isLoading,
    currentId: conversation?.id || null
  };
}
