import { StorageApiClient, migrateFromLocalStorage } from './storageApiClient';

const STORAGE_KEY = 'zagalin-conversations';
const MAX_CONVERSATIONS = 50;
const MAX_MESSAGES_PER_CONVERSATION = 100;

let backendAvailable: boolean | null = null;
let migrationAttempted = false;

export interface StoredMessage {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  tokens?: number;
  cost?: number;
  artifacts?: any[]; // Optional artifacts for messages
}

export type ConversationMessage = StoredMessage;

export interface Conversation {
  id: string;
  title: string;
  messages: StoredMessage[];
  createdAt: Date;
  updatedAt: Date;
  isPinned: boolean;
  context?: {
    dashboardUid?: string;
    dashboardTitle?: string;
    panelId?: number;
    panelTitle?: string;
    timeFrom?: string;
    timeTo?: string;
  };
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

async function ensureBackendReady(): Promise<boolean> {
  if (backendAvailable === null) {
    backendAvailable = await StorageApiClient.isAvailable();

    if (backendAvailable && !migrationAttempted) {
      migrationAttempted = true;
      const result = await migrateFromLocalStorage();
      if (result.success && result.migrated > 0) {
        console.log(`Successfully migrated ${result.migrated} conversations to backend storage`);
      }
    }
  }
  return backendAvailable;
}

async function loadAllConversations(): Promise<Conversation[]> {
  try {
    const useBackend = await ensureBackendReady();

    if (useBackend) {
      const metadata = await StorageApiClient.getConversations();

      const conversations = await Promise.all(
        metadata.map(async (meta) => {
          const conv = await StorageApiClient.getConversation(meta.id);
          return conv;
        })
      );

      return conversations.filter((c): c is Conversation => c !== null);
    }

    const data = localStorage.getItem(STORAGE_KEY);
    if (!data) {
      return [];
    }

    const parsed = JSON.parse(data);

    return parsed.map((conv: any) => ({
      ...conv,
      createdAt: new Date(conv.createdAt),
      updatedAt: new Date(conv.updatedAt),
      messages: conv.messages.map((msg: any) => ({
        ...msg,
        timestamp: new Date(msg.timestamp),
      })),
    }));
  } catch (error) {
    console.error('Failed to load conversations:', error);

    try {
      const data = localStorage.getItem(STORAGE_KEY);
      if (!data) {
        return [];
      }
      const parsed = JSON.parse(data);
      return parsed.map((conv: any) => ({
        ...conv,
        createdAt: new Date(conv.createdAt),
        updatedAt: new Date(conv.updatedAt),
        messages: conv.messages.map((msg: any) => ({
          ...msg,
          timestamp: new Date(msg.timestamp),
        })),
      }));
    } catch (localError) {
      console.error('localStorage fallback failed:', localError);
      return [];
    }
  }
}

async function saveConversation(conversation: Conversation): Promise<void> {
  try {
    const useBackend = await ensureBackendReady();

    if (useBackend) {
      await StorageApiClient.saveConversation(conversation);
    } else {
      const conversations = await loadAllConversations();
      const index = conversations.findIndex(c => c.id === conversation.id);

      if (index >= 0) {
        conversations[index] = conversation;
      } else {
        conversations.push(conversation);
      }

      const pruned = pruneOldConversations(conversations);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(pruned));
    }
  } catch (error) {
    console.error('Failed to save conversation:', error);

    try {
      const conversations = await loadAllConversations();
      const index = conversations.findIndex(c => c.id === conversation.id);

      if (index >= 0) {
        conversations[index] = conversation;
      } else {
        conversations.push(conversation);
      }

      const pruned = pruneOldConversations(conversations);
      localStorage.setItem(STORAGE_KEY, JSON.stringify(pruned));
    } catch (localError) {
      console.error('localStorage fallback failed:', localError);
      throw new Error('Failed to save conversation. Storage unavailable.');
    }
  }
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

export class ConversationStorage {
  static async getConversationList(): Promise<ConversationMetadata[]> {
    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        const metadata = await StorageApiClient.getConversations();
        return metadata;
      }

      const conversations = await loadAllConversations();

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

  static async getConversation(id: string): Promise<Conversation | null> {
    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        return await StorageApiClient.getConversation(id);
      }

      const conversations = await loadAllConversations();
      return conversations.find(c => c.id === id) || null;
    } catch (error) {
      console.error('Failed to get conversation:', error);
      return null;
    }
  }

  static async createConversation(context?: Conversation['context']): Promise<Conversation> {
    const now = new Date();
    const conversation: Conversation = {
      id: generateId(),
      title: 'New Chat',
      messages: [],
      createdAt: now,
      updatedAt: now,
      isPinned: false,
      context,
    };

    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        await StorageApiClient.saveConversation(conversation);
      } else {
        const conversations = await loadAllConversations();
        conversations.push(conversation);

        const pruned = pruneOldConversations(conversations);
        localStorage.setItem(STORAGE_KEY, JSON.stringify(pruned));
      }
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }

    return conversation;
  }

  static async saveConversation(conversation: Conversation): Promise<void> {
    conversation.messages = trimMessages(conversation.messages);

    if (conversation.title === 'New Chat' && conversation.messages.length > 0) {
      conversation.title = generateTitle(conversation.messages);
    }

    conversation.updatedAt = new Date();

    await saveConversation(conversation);
  }

  static async deleteConversation(id: string): Promise<void> {
    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        await StorageApiClient.deleteConversation(id);
      } else {
        const conversations = await loadAllConversations();
        const filtered = conversations.filter(c => c.id !== id);
        localStorage.setItem(STORAGE_KEY, JSON.stringify(filtered));
      }
    } catch (error) {
      console.error('Failed to delete conversation:', error);
      throw error;
    }
  }

  static async updateTitle(id: string, title: string): Promise<void> {
    try {
      const sanitized = title.trim().slice(0, 50) || 'Untitled Chat';

      const useBackend = await ensureBackendReady();

      if (useBackend) {
        await StorageApiClient.updateTitle(id, sanitized);
      } else {
        const conversations = await loadAllConversations();
        const conversation = conversations.find(c => c.id === id);

        if (conversation) {
          conversation.title = sanitized;
          conversation.updatedAt = new Date();
          localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
        }
      }
    } catch (error) {
      console.error('Failed to update title:', error);
      throw error;
    }
  }

  static async togglePin(id: string): Promise<void> {
    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        await StorageApiClient.togglePin(id);
      } else {
        const conversations = await loadAllConversations();
        const conversation = conversations.find(c => c.id === id);

        if (conversation) {
          conversation.isPinned = !conversation.isPinned;
          conversation.updatedAt = new Date();
          localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
        }
      }
    } catch (error) {
      console.error('Failed to toggle pin:', error);
      throw error;
    }
  }

  static async addMessage(conversationId: string, message: Omit<StoredMessage, 'id'>): Promise<void> {
    const conversation = await this.getConversation(conversationId);

    if (conversation) {
      const storedMessage: StoredMessage = {
        ...message,
        id: generateId(),
      };

      conversation.messages.push(storedMessage);
      await this.saveConversation(conversation);
    }
  }

  static async clearAll(): Promise<void> {
    try {
      const useBackend = await ensureBackendReady();

      if (useBackend) {
        const conversations = await StorageApiClient.getConversations();
        await Promise.all(
          conversations.map(conv => StorageApiClient.deleteConversation(conv.id))
        );
      }

      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error('Failed to clear conversations:', error);
    }
  }

  static async exportConversations(): Promise<string> {
    const conversations = await loadAllConversations();
    return JSON.stringify(conversations, null, 2);
  }

  static async importConversations(jsonData: string): Promise<void> {
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

      const useBackend = await ensureBackendReady();

      if (useBackend) {
        await Promise.all(
          conversations.map(conv => StorageApiClient.saveConversation(conv))
        );
      } else {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(conversations));
      }
    } catch (error) {
      console.error('Failed to import conversations:', error);
      throw new Error('Failed to import conversations. Invalid format.');
    }
  }
}
