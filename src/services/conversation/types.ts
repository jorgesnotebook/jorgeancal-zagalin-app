/**
 * Type definitions for Conversation Management module
 *
 * TODO: Import and consolidate types from:
 * - conversationStorage.ts
 * - conversationExport.ts
 */

export interface Conversation {
  id: string;
  title: string;
  messages: Message[];
  createdAt: number;
  updatedAt: number;
  pinned?: boolean;
  context?: any;
}

export interface Message {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: number;
  metadata?: Record<string, any>;
}

export interface ExportData {
  version: string;
  conversation: Conversation;
  exportedAt: number;
}

export type StorageBackend = 'backend' | 'localStorage';

// TODO: Add more types from existing services
