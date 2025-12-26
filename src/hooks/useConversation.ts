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
  const refreshConversationList = useCallback(() => {
    const list = ConversationStorage.getConversationList();
    setConversations(list);
  }, []);

  // Load conversation list on mount
  useEffect(() => {
    refreshConversationList();
  }, [refreshConversationList]);

  /**
   * Create a new conversation
   */
  const createNew = useCallback((grafanaContext?: GrafanaContext) => {
    // Convert GrafanaContext to conversation context
    const context = grafanaContext ? {
      dashboardUid: grafanaContext.dashboard?.uid,
      dashboardTitle: grafanaContext.dashboard?.title,
      panelId: grafanaContext.panel?.id,
      panelTitle: grafanaContext.panel?.title,
      timeRange: grafanaContext.timeRange
    } : undefined;

    const newConv = ConversationStorage.createConversation(undefined, context);
    setConversation(newConv);
    // Save empty conversation
    ConversationStorage.saveConversation(newConv);
    refreshConversationList();
  }, [refreshConversationList]);

  /**
   * Load an existing conversation
   */
  const loadConversation = useCallback((id: string) => {
    setIsLoading(true);
    try {
      const conv = ConversationStorage.getConversation(id);
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
  const addMessage = useCallback((message: ConversationMessage) => {
    if (!conversation) {
      console.warn('No active conversation. Creating new one.');
      const newConv = ConversationStorage.createConversation(message);
      setConversation(newConv);
      ConversationStorage.saveConversation(newConv);
      refreshConversationList();
      return;
    }

    const updated = {
      ...conversation,
      messages: [...conversation.messages, message],
      updatedAt: new Date()
    };

    // Update title if this is the first user message
    if (conversation.messages.length === 0 && message.role === 'user') {
      updated.title = message.content.slice(0, 50);
      if (message.content.length > 50) {
        updated.title += '...';
      }
    }

    setConversation(updated);
    ConversationStorage.saveConversation(updated);
    refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Delete a conversation
   */
  const deleteConversation = useCallback((id: string) => {
    ConversationStorage.deleteConversation(id);

    // If we deleted the current conversation, clear it
    if (conversation?.id === id) {
      setConversation(null);
    }

    refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Update conversation title
   */
  const updateTitle = useCallback((id: string, title: string) => {
    ConversationStorage.updateTitle(id, title);

    // Update current conversation if it's the one being renamed
    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        title
      });
    }

    refreshConversationList();
  }, [conversation, refreshConversationList]);

  /**
   * Toggle pin status
   */
  const togglePin = useCallback((id: string) => {
    ConversationStorage.togglePin(id);

    // Update current conversation if it's the one being pinned/unpinned
    if (conversation?.id === id) {
      setConversation({
        ...conversation,
        isPinned: !conversation.isPinned
      });
    }

    refreshConversationList();
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
