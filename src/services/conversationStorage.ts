export interface ConversationMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  tokens?: number;
  cost?: number;
}

export interface Conversation {
  id: string;
  title: string;
  messages: ConversationMessage[];
  createdAt: Date;
  updatedAt: Date;
  isPinned: boolean;
  context?: {
    dashboardUid?: string;
    dashboardTitle?: string;
    panelId?: number;
    panelTitle?: string;
    timeRange?: {
      from: string;
      to: string;
    };
  };
}

export interface ConversationMetadata {
  id: string;
  title: string;
  messageCount: number;
  createdAt: Date;
  updatedAt: Date;
  isPinned: boolean;
  preview: string; // Last user message
  context?: Conversation['context'];
}

const STORAGE_KEY = 'zagalin_conversations';
const MAX_CONVERSATIONS = 50;
const MAX_MESSAGES_PER_CONVERSATION = 100;

export class ConversationStorage {
  /**
   * Get all conversation metadata (without full message history)
   */
  static getConversationList(): ConversationMetadata[] {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) {
        return [];
      }

      const conversations: Conversation[] = JSON.parse(stored);

      // Convert to metadata and sort by update time (most recent first)
      return conversations
        .map(conv => this.toMetadata(conv))
        .sort((a, b) => {
          // Pinned conversations first
          if (a.isPinned && !b.isPinned) {
            return -1;
          }
          if (!a.isPinned && b.isPinned) {
            return 1;
          }
          // Then by update time
          return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
        });
    } catch (error) {
      console.error('Failed to load conversation list:', error);
      return [];
    }
  }

  /**
   * Get a specific conversation by ID
   */
  static getConversation(id: string): Conversation | null {
    try {
      const conversations = this.getAllConversations();
      const conversation = conversations.find(c => c.id === id);

      if (!conversation) {
        return null;
      }

      // Restore Date objects
      return {
        ...conversation,
        createdAt: new Date(conversation.createdAt),
        updatedAt: new Date(conversation.updatedAt),
        messages: conversation.messages.map(m => ({
          ...m,
          timestamp: new Date(m.timestamp)
        }))
      };
    } catch (error) {
      console.error('Failed to load conversation:', error);
      return null;
    }
  }

  /**
   * Save a new conversation or update existing one
   */
  static saveConversation(conversation: Conversation): void {
    try {
      let conversations = this.getAllConversations();

      // Update timestamp
      conversation.updatedAt = new Date();

      // Trim messages if too long
      if (conversation.messages.length > MAX_MESSAGES_PER_CONVERSATION) {
        // Keep system messages and recent messages
        const systemMessages = conversation.messages.filter(m => m.role === 'system');
        const recentMessages = conversation.messages
          .filter(m => m.role !== 'system')
          .slice(-MAX_MESSAGES_PER_CONVERSATION + systemMessages.length);

        conversation.messages = [...systemMessages, ...recentMessages];
      }

      // Find and update or append
      const existingIndex = conversations.findIndex(c => c.id === conversation.id);
      if (existingIndex >= 0) {
        conversations[existingIndex] = conversation;
      } else {
        conversations.push(conversation);
      }

      // Enforce max conversations limit
      if (conversations.length > MAX_CONVERSATIONS) {
        // Remove oldest non-pinned conversations
        conversations = this.pruneConversations(conversations);
      }

      localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
    } catch (error) {
      console.error('Failed to save conversation:', error);
      throw new Error('Failed to save conversation. Storage might be full.');
    }
  }

  /**
   * Create a new conversation
   */
  static createConversation(
    initialMessage?: ConversationMessage,
    context?: Conversation['context']
  ): Conversation {
    const now = new Date();
    const id = this.generateId();

    // Generate title from context or default
    let title = 'New Conversation';
    if (context?.dashboardTitle) {
      title = `Chat: ${context.dashboardTitle}`;
      if (context.panelTitle) {
        title += ` - ${context.panelTitle}`;
      }
    } else if (initialMessage) {
      // Use first 50 chars of initial message as title
      title = initialMessage.content.slice(0, 50);
      if (initialMessage.content.length > 50) {
        title += '...';
      }
    }

    const conversation: Conversation = {
      id,
      title,
      messages: initialMessage ? [initialMessage] : [],
      createdAt: now,
      updatedAt: now,
      isPinned: false,
      context
    };

    return conversation;
  }

  /**
   * Delete a conversation
   */
  static deleteConversation(id: string): void {
    try {
      const conversations = this.getAllConversations().filter(c => c.id !== id);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
    } catch (error) {
      console.error('Failed to delete conversation:', error);
    }
  }

  /**
   * Update conversation title
   */
  static updateTitle(id: string, title: string): void {
    try {
      const conversations = this.getAllConversations();
      const conversation = conversations.find(c => c.id === id);

      if (conversation) {
        conversation.title = title;
        conversation.updatedAt = new Date();
        localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
      }
    } catch (error) {
      console.error('Failed to update conversation title:', error);
    }
  }

  /**
   * Toggle pin status
   */
  static togglePin(id: string): void {
    try {
      const conversations = this.getAllConversations();
      const conversation = conversations.find(c => c.id === id);

      if (conversation) {
        conversation.isPinned = !conversation.isPinned;
        conversation.updatedAt = new Date();
        localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
      }
    } catch (error) {
      console.error('Failed to toggle pin:', error);
    }
  }

  /**
   * Clear all conversations (use with caution)
   */
  static clearAll(): void {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error('Failed to clear conversations:', error);
    }
  }

  /**
   * Get storage usage info
   */
  static getStorageInfo(): {
    conversationCount: number;
    totalMessages: number;
    storageSize: number;
  } {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      const conversations = stored ? JSON.parse(stored) : [];

      return {
        conversationCount: conversations.length,
        totalMessages: conversations.reduce((sum: number, c: Conversation) => sum + c.messages.length, 0),
        storageSize: stored ? new Blob([stored]).size : 0
      };
    } catch (error) {
      return {
        conversationCount: 0,
        totalMessages: 0,
        storageSize: 0
      };
    }
  }

  // ========== Private Helper Methods ==========

  private static getAllConversations(): Conversation[] {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return stored ? JSON.parse(stored) : [];
    } catch (error) {
      console.error('Failed to parse conversations:', error);
      return [];
    }
  }

  private static toMetadata(conversation: Conversation): ConversationMetadata {
    // Get last user message for preview
    const lastUserMessage = [...conversation.messages]
      .reverse()
      .find(m => m.role === 'user');

    const preview = lastUserMessage
      ? lastUserMessage.content.slice(0, 100)
      : 'No messages yet';

    return {
      id: conversation.id,
      title: conversation.title,
      messageCount: conversation.messages.length,
      createdAt: new Date(conversation.createdAt),
      updatedAt: new Date(conversation.updatedAt),
      isPinned: conversation.isPinned,
      preview,
      context: conversation.context
    };
  }

  private static pruneConversations(conversations: Conversation[]): Conversation[] {
    // Separate pinned and unpinned
    const pinned = conversations.filter(c => c.isPinned);
    const unpinned = conversations.filter(c => !c.isPinned);

    // Sort unpinned by update time (oldest first for removal)
    unpinned.sort((a, b) =>
      new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime()
    );

    // Calculate how many to keep
    const totalToKeep = MAX_CONVERSATIONS;
    const pinnedCount = pinned.length;
    const unpinnedToKeep = Math.max(0, totalToKeep - pinnedCount);

    // Keep most recent unpinned conversations
    const keptUnpinned = unpinned.slice(-unpinnedToKeep);

    return [...pinned, ...keptUnpinned];
  }

  private static generateId(): string {
    return `conv_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
  }
}
