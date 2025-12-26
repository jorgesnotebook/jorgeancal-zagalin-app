/**
 * React hook for conversation management
 * Provides easy access to conversation storage and state management
 */

import { useState, useEffect, useCallback } from 'react';
import {
  ConversationStorage,
  type Conversation,
  type ConversationMessage,
  type ConversationMetadata
} from '../services/conversationStorage';
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
  addMessage: (message: ConversationMessage) => void;
  deleteConversation: (id: string) => void;
  updateTitle: (id: string, title: string) => void;
  togglePin: (id: string) => void;
  clearCurrent: () => void;

  // State
  isLoading: boolean;
  currentId: string | null;
}

export function useConversation(): UseConversationReturn {
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [conversations, setConversations] = useState<ConversationMetadata[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // Refresh the conversation list
  const refreshConversationList = useCallback(async () => {
    const list = await ConversationStorage.getConversationList();
    setConversations(list);
  }, []);

  // Load conversation list on mount
  useEffect(() => {
    refreshConversationList();
  }, [refreshConversationList]);

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

    const newConv = await ConversationStorage.createConversation(context);
    setConversation(newConv);
    await ConversationStorage.saveConversation(newConv);
    await refreshConversationList();
  }, [refreshConversationList]);

  /**
   * Load an existing conversation
   */
  const loadConversation = useCallback(async (id: string) => {
    setIsLoading(true);
    try {
      const conv = await ConversationStorage.getConversation(id);
      if (conv) {
        setConversation(conv);
      } else {
        console.warn(`Conversation ${id} not found`);
      }
    } catch (error) {
      console.error('Failed to load conversation:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  /**
   * Add a message to the current conversation
   */
  const addMessage = useCallback(async (message: ConversationMessage) => {
    if (!conversation) {
      console.warn('No active conversation. Creating new one.');
      const newConv = await ConversationStorage.createConversation();
      const updatedConv = {
        ...newConv,
        messages: [message]
      };
      setConversation(updatedConv);
      await ConversationStorage.saveConversation(updatedConv);
      await refreshConversationList();
      return;
    }

    const updated = {
      ...conversation,
      messages: [...conversation.messages, message],
      updatedAt: new Date()
    };

    if (conversation.messages.length === 0 && message.role === 'user') {
      updated.title = message.content.slice(0, 50);
      if (message.content.length > 50) {
        updated.title += '...';
      }
    }

    setConversation(updated);
    await ConversationStorage.saveConversation(updated);
    await refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Delete a conversation
   */
  const deleteConversation = useCallback(async (id: string) => {
    await ConversationStorage.deleteConversation(id);

    if (conversation?.id === id) {
      setConversation(null);
    }

    await refreshConversationList();
  }, [conversation, refreshConversationList]);

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

    await ConversationStorage.updateTitle(id, sanitizedTitle);

    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        title: sanitizedTitle
      });
    }

    await refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Toggle pin status
   */
  const togglePin = useCallback(async (id: string) => {
    await ConversationStorage.togglePin(id);

    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        isPinned: !conversation.isPinned
      });
    }

    await refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Clear the current conversation (start fresh)
   */
  const clearCurrent = useCallback(() => {
    setConversation(null);
  }, []);

  return {
    conversation,
    messages: conversation?.messages || [],
    conversations,
    createNew,
    loadConversation,
    addMessage,
    deleteConversation,
    updateTitle,
    togglePin,
    clearCurrent,
    isLoading,
    currentId: conversation?.id || null
  };
}
