/**
 * Conversation storage using Grafana's official User Storage API
 *
 * Uses usePluginUserStorage() hook which:
 * - Stores in Grafana DB (11.5+ with userStorageAPI feature flag)
 * - Automatically falls back to localStorage if flag disabled
 * - Per-user storage with no backend code needed
 *
 * Note: This module provides the core storage logic. React components
 * should use the useConversationStorage hook which wraps this.
 */

const STORAGE_KEY = 'zagalin-conversations';
const MAX_CONVERSATIONS = 50;
const MAX_MESSAGES_PER_CONVERSATION = 100;

export interface StoredMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  tokens?: number;
  cost?: number;
  artifacts?: any[];
}

export type ConversationMessage = StoredMessage;

export interface ConversationContext {
  dashboardUid: string;
  dashboardTitle: string;
  panelId?: number;
  panelTitle?: string;
  timeFrom?: string;
  timeTo?: string;
  addedAt: Date; // Track when this context was added
}

export interface Conversation {
  id: string;
  title: string;
  messages: StoredMessage[];
  createdAt: Date;
  updatedAt: Date;
  isPinned: boolean;
  contexts: ConversationContext[]; // Changed from single context to array
}

export interface ConversationMetadata {
  id: string;
  title: string;
  messageCount: number;
  lastMessagePreview: string;
  updatedAt: Date;
  isPinned: boolean;
}

function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`;
}

function generateTitle(messages: StoredMessage[]): string {
  const firstUserMessage = messages.find(m => m.role === 'user');
  if (firstUserMessage) {
    const truncated = firstUserMessage.content.slice(0, 50);
    return truncated.length < firstUserMessage.content.length
      ? `${truncated}...`
      : truncated;
  }

  const timestamp = new Date().toLocaleString();
  return `Chat from ${timestamp}`;
}

/**
 * Storage interface that works with any storage backend
 * (Grafana User Storage API or localStorage fallback)
 */
export interface StorageBackend {
  getItem(key: string): Promise<string | null> | string | null;
  setItem(key: string, value: string): Promise<void> | void;
  removeItem(key: string): Promise<void> | void;
}

/**
 * Load conversations from storage backend
 */
async function loadAllConversations(storage: StorageBackend): Promise<Conversation[]> {
  try {
    const data = await storage.getItem(STORAGE_KEY);
    if (!data) {
      return [];
    }

    const parsed = JSON.parse(data);

    return parsed.map((conv: any) => {
      // Migrate old single context to new contexts array
      let contexts: ConversationContext[] = [];
      if (conv.context && conv.context.dashboardUid) {
        contexts = [{
          dashboardUid: conv.context.dashboardUid,
          dashboardTitle: conv.context.dashboardTitle || 'Unknown Dashboard',
          panelId: conv.context.panelId,
          panelTitle: conv.context.panelTitle,
          timeFrom: conv.context.timeFrom,
          timeTo: conv.context.timeTo,
          addedAt: new Date(conv.createdAt), // Use conversation creation date
        }];
      } else if (conv.contexts) {
        contexts = conv.contexts.map((ctx: any) => ({
          ...ctx,
          addedAt: new Date(ctx.addedAt),
        }));
      }

      return {
        ...conv,
        contexts, // Always use array
        createdAt: new Date(conv.createdAt),
        updatedAt: new Date(conv.updatedAt),
        messages: conv.messages.map((msg: any) => ({
          ...msg,
          timestamp: new Date(msg.timestamp),
        })),
      };
    });
  } catch (error) {
    console.error('Failed to load conversations:', error);
    return [];
  }
}

/**
 * Save conversations to storage backend
 */
async function saveAllConversations(storage: StorageBackend, conversations: Conversation[]): Promise<void> {
  await storage.setItem(STORAGE_KEY, JSON.stringify(conversations));
}

function pruneOldConversations(conversations: Conversation[]): Conversation[] {
  if (conversations.length <= MAX_CONVERSATIONS) {
    return conversations;
  }

  const pinned = conversations.filter(c => c.isPinned);
  const unpinned = conversations.filter(c => !c.isPinned);

  unpinned.sort((a, b) => a.updatedAt.getTime() - b.updatedAt.getTime());

  const toKeep = MAX_CONVERSATIONS - pinned.length;
  const kept = unpinned.slice(-toKeep);

  return [...pinned, ...kept];
}

function trimMessages(messages: StoredMessage[]): StoredMessage[] {
  if (!messages || messages.length <= MAX_MESSAGES_PER_CONVERSATION) {
    return messages || [];
  }

  const systemMessages = messages.filter(m => m.role === 'system');
  const otherMessages = messages.filter(m => m.role !== 'system');

  const recentMessages = otherMessages.slice(-MAX_MESSAGES_PER_CONVERSATION + systemMessages.length);

  return [...systemMessages, ...recentMessages];
}

/**
 * ConversationStorage class that works with any storage backend
 * Pass in storage from usePluginUserStorage() hook in React components
 */
export class ConversationStorage {
  static async getConversationList(storage: StorageBackend): Promise<ConversationMetadata[]> {
    try {
      const conversations = await loadAllConversations(storage);

      conversations.sort((a, b) => {
        if (a.isPinned && !b.isPinned) {
          return -1;
        }
        if (!a.isPinned && b.isPinned) {
          return 1;
        }
        return b.updatedAt.getTime() - a.updatedAt.getTime();
      });

      return conversations.map(conv => ({
        id: conv.id,
        title: conv.title,
        messageCount: conv.messages.length,
        lastMessagePreview: conv.messages[conv.messages.length - 1]?.content.slice(0, 100) || '',
        updatedAt: conv.updatedAt,
        isPinned: conv.isPinned,
      }));
    } catch (error) {
      console.error('Failed to get conversation list:', error);
      return [];
    }
  }

  static async getConversation(storage: StorageBackend, id: string): Promise<Conversation | null> {
    try {
      const conversations = await loadAllConversations(storage);
      return conversations.find(c => c.id === id) || null;
    } catch (error) {
      console.error('Failed to get conversation:', error);
      return null;
    }
  }

  static async createConversation(storage: StorageBackend, context?: ConversationContext): Promise<Conversation> {
    const now = new Date();
    const conversation: Conversation = {
      id: generateId(),
      title: 'New Chat',
      messages: [],
      createdAt: now,
      updatedAt: now,
      isPinned: false,
      contexts: context ? [context] : [], // Initialize with optional context
    };

    try {
      const conversations = await loadAllConversations(storage);
      conversations.push(conversation);

      const pruned = pruneOldConversations(conversations);
      await saveAllConversations(storage, pruned);
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }

    return conversation;
  }

  static async saveConversation(storage: StorageBackend, conversation: Conversation): Promise<void> {
    conversation.messages = trimMessages(conversation.messages);

    if (conversation.title === 'New Chat' && conversation.messages.length > 0) {
      conversation.title = generateTitle(conversation.messages);
    }

    conversation.updatedAt = new Date();

    const conversations = await loadAllConversations(storage);
    const index = conversations.findIndex(c => c.id === conversation.id);

    if (index >= 0) {
      conversations[index] = conversation;
    } else {
      conversations.push(conversation);
    }

    const pruned = pruneOldConversations(conversations);
    await saveAllConversations(storage, pruned);
  }

  static async deleteConversation(storage: StorageBackend, id: string): Promise<void> {
    try {
      const conversations = await loadAllConversations(storage);
      const filtered = conversations.filter(c => c.id !== id);
      await saveAllConversations(storage, filtered);
    } catch (error) {
      console.error('Failed to delete conversation:', error);
      throw error;
    }
  }

  static async updateTitle(storage: StorageBackend, id: string, title: string): Promise<void> {
    try {
      const sanitized = title.trim().slice(0, 50) || 'Untitled Chat';
      const conversations = await loadAllConversations(storage);
      const conversation = conversations.find(c => c.id === id);

      if (conversation) {
        conversation.title = sanitized;
        conversation.updatedAt = new Date();
        await saveAllConversations(storage, conversations);
      }
    } catch (error) {
      console.error('Failed to update title:', error);
      throw error;
    }
  }

  static async togglePin(storage: StorageBackend, id: string): Promise<void> {
    try {
      const conversations = await loadAllConversations(storage);
      const conversation = conversations.find(c => c.id === id);

      if (conversation) {
        conversation.isPinned = !conversation.isPinned;
        conversation.updatedAt = new Date();
        await saveAllConversations(storage, conversations);
      }
    } catch (error) {
      console.error('Failed to toggle pin:', error);
      throw error;
    }
  }

  static async addMessage(storage: StorageBackend, conversationId: string, message: Omit<StoredMessage, 'id'>): Promise<void> {
    const conversation = await this.getConversation(storage, conversationId);

    if (conversation) {
      const storedMessage: StoredMessage = {
        ...message,
        id: generateId(),
      };

      conversation.messages.push(storedMessage);
      await this.saveConversation(storage, conversation);
    }
  }

  static async clearAll(storage: StorageBackend): Promise<void> {
    try {
      await storage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error('Failed to clear conversations:', error);
    }
  }

  static async exportConversations(storage: StorageBackend): Promise<string> {
    const conversations = await loadAllConversations(storage);
    return JSON.stringify(conversations, null, 2);
  }

  static async importConversations(storage: StorageBackend, jsonData: string): Promise<void> {
    try {
      const parsed = JSON.parse(jsonData);
      if (!Array.isArray(parsed)) {
        throw new Error('Invalid format: expected array of conversations');
      }

      const conversations: Conversation[] = parsed.map((conv: any) => ({
        ...conv,
        createdAt: new Date(conv.createdAt),
        updatedAt: new Date(conv.updatedAt),
        messages: conv.messages.map((msg: any) => ({
          ...msg,
          timestamp: new Date(msg.timestamp),
        })),
      }));

      await saveAllConversations(storage, conversations);
    } catch (error) {
      console.error('Failed to import conversations:', error);
      throw new Error('Failed to import conversations. Invalid format.');
    }
  }
}
