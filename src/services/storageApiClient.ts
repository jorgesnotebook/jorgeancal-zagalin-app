import { getBackendSrv } from '@grafana/runtime';
import type { Conversation, ConversationMetadata } from './conversationStorage';

const PLUGIN_ID = 'jorgeancal-zagalin-app';

function getBasePath(): string {
  return `/api/plugins/${PLUGIN_ID}/resources/storage`;
}

export class StorageApiClient {
  static async getConversations(): Promise<ConversationMetadata[]> {
    try {
      const response = await getBackendSrv().get(`${getBasePath()}/conversations`);
      return response;
    } catch (error) {
      console.error('Failed to fetch conversations:', error);
      throw new Error('Failed to load conversations from server');
    }
  }

  static async getConversation(id: string): Promise<Conversation | null> {
    try {
      const response = await getBackendSrv().get(`${getBasePath()}/conversation`, { id });
      return response;
    } catch (error) {
      if ((error as any)?.status === 404) {
        return null;
      }
      console.error('Failed to fetch conversation:', error);
      throw new Error('Failed to load conversation from server');
    }
  }

  static async saveConversation(conversation: Conversation): Promise<void> {
    try {
      await getBackendSrv().post(`${getBasePath()}/conversation/save`, conversation);
    } catch (error) {
      console.error('Failed to save conversation:', error);
      throw new Error('Failed to save conversation to server');
    }
  }

  static async deleteConversation(id: string): Promise<void> {
    try {
      await getBackendSrv().delete(`${getBasePath()}/conversation/delete?id=${encodeURIComponent(id)}`);
    } catch (error) {
      console.error('Failed to delete conversation:', error);
      throw new Error('Failed to delete conversation from server');
    }
  }

  static async updateTitle(id: string, title: string): Promise<void> {
    try {
      await getBackendSrv().post(`${getBasePath()}/conversation/title`, { id, title });
    } catch (error) {
      console.error('Failed to update title:', error);
      throw new Error('Failed to update conversation title');
    }
  }

  static async togglePin(id: string): Promise<void> {
    try {
      await getBackendSrv().post(`${getBasePath()}/conversation/pin`, null, { params: { id } });
    } catch (error) {
      console.error('Failed to toggle pin:', error);
      throw new Error('Failed to toggle conversation pin');
    }
  }

  static async isAvailable(): Promise<boolean> {
    try {
      await getBackendSrv().get(`/api/plugins/${PLUGIN_ID}/health`);
      return true;
    } catch (error) {
      console.warn('Backend storage not available:', error);
      return false;
    }
  }
}

export async function migrateFromLocalStorage(): Promise<{
  success: boolean;
  migrated: number;
  errors: number;
}> {
  const STORAGE_KEY = 'zagalin-conversations';

  try {
    const available = await StorageApiClient.isAvailable();
    if (!available) {
      console.warn('Backend not available, skipping migration');
      return { success: false, migrated: 0, errors: 0 };
    }

    const localData = localStorage.getItem(STORAGE_KEY);
    if (!localData) {
      console.log('No localStorage data to migrate');
      return { success: true, migrated: 0, errors: 0 };
    }

    const conversations = JSON.parse(localData);
    if (!Array.isArray(conversations)) {
      console.error('Invalid localStorage data format');
      return { success: false, migrated: 0, errors: 1 };
    }

    let migrated = 0;
    let errors = 0;

    for (const conv of conversations) {
      try {
        const conversation: Conversation = {
          ...conv,
          createdAt: new Date(conv.createdAt),
          updatedAt: new Date(conv.updatedAt),
          messages: conv.messages.map((msg: any) => ({
            ...msg,
            timestamp: new Date(msg.timestamp),
          })),
        };

        await StorageApiClient.saveConversation(conversation);
        migrated++;
      } catch (error) {
        console.error('Failed to migrate conversation:', conv.id, error);
        errors++;
      }
    }

    if (migrated > 0 && errors === 0) {
      localStorage.setItem(`${STORAGE_KEY}-backup`, localData);
      localStorage.removeItem(STORAGE_KEY);
      console.log(`Migrated ${migrated} conversations successfully`);
    }

    return { success: errors === 0, migrated, errors };
  } catch (error) {
    console.error('Migration failed:', error);
    return { success: false, migrated: 0, errors: 1 };
  }
}
