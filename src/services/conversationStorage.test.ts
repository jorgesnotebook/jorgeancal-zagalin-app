import {
  ConversationStorage,
  Conversation,
  StoredMessage,
  ConversationContext,
  StorageBackend,
} from './conversationStorage';

describe('ConversationStorage', () => {
  let mockStorage: StorageBackend;
  let storedData: Map<string, string>;

  beforeEach(() => {
    storedData = new Map();

    mockStorage = {
      getItem: jest.fn((key: string) => {
        return Promise.resolve(storedData.get(key) || null);
      }),
      setItem: jest.fn((key: string, value: string) => {
        storedData.set(key, value);
        return Promise.resolve();
      }),
      removeItem: jest.fn((key: string) => {
        storedData.delete(key);
        return Promise.resolve();
      }),
    };
  });

  afterEach(() => {
    jest.clearAllMocks();
    storedData.clear();
  });

  describe('CRUD Operations', () => {
    describe('createConversation', () => {
      it('should create a conversation with default values', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        expect(conv.id).toBeDefined();
        expect(conv.title).toBe('New Chat');
        expect(conv.messages).toEqual([]);
        expect(conv.isPinned).toBe(false);
        expect(conv.createdAt).toBeInstanceOf(Date);
        expect(conv.updatedAt).toBeInstanceOf(Date);
        expect(conv.contexts).toEqual([]);
      });

      it('should create a conversation with context', async () => {
        const context: ConversationContext = {
          dashboardUid: 'dash-123',
          dashboardTitle: 'Test Dashboard',
          panelId: 1,
          panelTitle: 'Test Panel',
          timeFrom: 'now-1h',
          timeTo: 'now',
          addedAt: new Date(),
        };

        const conv = await ConversationStorage.createConversation(mockStorage, context);

        expect(conv.contexts).toHaveLength(1);
        expect(conv.contexts[0]).toEqual(context);
      });

      it('should save newly created conversation to storage', async () => {
        await ConversationStorage.createConversation(mockStorage);

        expect(mockStorage.setItem).toHaveBeenCalled();

        const saved = storedData.get('zagalin-conversations');
        expect(saved).toBeDefined();

        const parsed = JSON.parse(saved!);
        expect(parsed).toHaveLength(1);
      });

      it('should prune conversations when creating exceeds max', async () => {
        for (let i = 0; i < 51; i++) {
          await ConversationStorage.createConversation(mockStorage);
        }

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBeLessThanOrEqual(50);
      });
    });

    describe('getConversation', () => {
      it('should retrieve existing conversation by id', async () => {
        const created = await ConversationStorage.createConversation(mockStorage);
        const retrieved = await ConversationStorage.getConversation(mockStorage, created.id);

        expect(retrieved).not.toBeNull();
        expect(retrieved!.id).toBe(created.id);
      });

      it('should return null for non-existent conversation', async () => {
        const retrieved = await ConversationStorage.getConversation(mockStorage, 'non-existent-id');

        expect(retrieved).toBeNull();
      });

      it('should parse dates correctly when retrieving', async () => {
        const created = await ConversationStorage.createConversation(mockStorage);
        const retrieved = await ConversationStorage.getConversation(mockStorage, created.id);

        expect(retrieved!.createdAt).toBeInstanceOf(Date);
        expect(retrieved!.updatedAt).toBeInstanceOf(Date);
      });
    });

    describe('saveConversation', () => {
      it('should update existing conversation', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);
        conv.title = 'Updated Title';

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('Updated Title');
      });

      it('should update updatedAt timestamp', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);
        const originalUpdatedAt = conv.updatedAt;

        await new Promise((resolve) => setTimeout(resolve, 10));

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.updatedAt.getTime()).toBeGreaterThan(originalUpdatedAt.getTime());
      });

      it('should create new conversation if id does not exist', async () => {
        const conv: Conversation = {
          id: 'new-id',
          title: 'New Conv',
          messages: [],
          createdAt: new Date(),
          updatedAt: new Date(),
          isPinned: false,
          contexts: [],
        };

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved).not.toBeNull();
      });

      it('should auto-generate title from first user message', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        const message: StoredMessage = {
          id: '1',
          role: 'user',
          content: 'This is my first message',
          timestamp: new Date(),
        };

        conv.messages.push(message);
        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('This is my first message');
      });

      it('should truncate long title to 50 characters', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        const longMessage = 'a'.repeat(100);
        const message: StoredMessage = {
          id: '1',
          role: 'user',
          content: longMessage,
          timestamp: new Date(),
        };

        conv.messages.push(message);
        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toHaveLength(53); // 50 + '...'
        expect(retrieved!.title).toMatch(/\.\.\.$/);
      });
    });

    describe('deleteConversation', () => {
      it('should delete existing conversation', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.deleteConversation(mockStorage, conv.id);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved).toBeNull();
      });

      it('should not throw when deleting non-existent conversation', async () => {
        await expect(ConversationStorage.deleteConversation(mockStorage, 'non-existent')).resolves.not.toThrow();
      });
    });

    describe('updateTitle', () => {
      it('should update conversation title', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.updateTitle(mockStorage, conv.id, 'Custom Title');

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('Custom Title');
      });

      it('should trim whitespace from title', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.updateTitle(mockStorage, conv.id, '  Trimmed Title  ');

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('Trimmed Title');
      });

      it('should use default title for empty string', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.updateTitle(mockStorage, conv.id, '   ');

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('Untitled Chat');
      });

      it('should truncate title to 50 characters', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.updateTitle(mockStorage, conv.id, 'a'.repeat(100));

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toHaveLength(50);
      });
    });

    describe('addMessage', () => {
      it('should add message to existing conversation', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        const message: Omit<StoredMessage, 'id'> = {
          role: 'user',
          content: 'Test message',
          timestamp: new Date(),
        };

        await ConversationStorage.addMessage(mockStorage, conv.id, message);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.messages).toHaveLength(1);
        expect(retrieved!.messages[0].content).toBe('Test message');
        expect(retrieved!.messages[0].id).toBeDefined();
      });

      it('should not throw when adding to non-existent conversation', async () => {
        const message: Omit<StoredMessage, 'id'> = {
          role: 'user',
          content: 'Test message',
          timestamp: new Date(),
        };

        await expect(ConversationStorage.addMessage(mockStorage, 'non-existent', message)).resolves.not.toThrow();
      });
    });

    describe('getConversationList', () => {
      it('should return empty array when no conversations exist', async () => {
        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list).toEqual([]);
      });

      it('should return metadata for all conversations', async () => {
        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list).toHaveLength(2);
        expect(list[0]).toHaveProperty('id');
        expect(list[0]).toHaveProperty('title');
        expect(list[0]).toHaveProperty('messageCount');
        expect(list[0]).toHaveProperty('lastMessagePreview');
        expect(list[0]).toHaveProperty('updatedAt');
        expect(list[0]).toHaveProperty('isPinned');
      });

      it('should include message count', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        for (let i = 0; i < 5; i++) {
          await ConversationStorage.addMessage(mockStorage, conv.id, {
            role: 'user',
            content: `Message ${i}`,
            timestamp: new Date(),
          });
        }

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list[0].messageCount).toBe(5);
      });
    });

    describe('clearAll', () => {
      it('should remove all conversations', async () => {
        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.clearAll(mockStorage);

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list).toEqual([]);
      });
    });
  });

  describe('Pinning and Sorting', () => {
    describe('togglePin', () => {
      it('should pin unpinned conversation', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.togglePin(mockStorage, conv.id);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.isPinned).toBe(true);
      });

      it('should unpin pinned conversation', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.togglePin(mockStorage, conv.id);
        await ConversationStorage.togglePin(mockStorage, conv.id);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.isPinned).toBe(false);
      });

      it('should update updatedAt when toggling pin', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);
        const originalUpdatedAt = conv.updatedAt;

        await new Promise((resolve) => setTimeout(resolve, 10));

        await ConversationStorage.togglePin(mockStorage, conv.id);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.updatedAt.getTime()).toBeGreaterThan(originalUpdatedAt.getTime());
      });
    });

    describe('sorting', () => {
      it('should sort pinned conversations first', async () => {
        await ConversationStorage.createConversation(mockStorage);
        const conv2 = await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.togglePin(mockStorage, conv2.id);

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list[0].id).toBe(conv2.id);
        expect(list[0].isPinned).toBe(true);
      });

      it('should sort unpinned by updatedAt descending', async () => {
        const conv1 = await ConversationStorage.createConversation(mockStorage);
        await new Promise((resolve) => setTimeout(resolve, 10));

        const conv2 = await ConversationStorage.createConversation(mockStorage);
        await new Promise((resolve) => setTimeout(resolve, 10));

        const conv3 = await ConversationStorage.createConversation(mockStorage);

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list[0].id).toBe(conv3.id);
        expect(list[1].id).toBe(conv2.id);
        expect(list[2].id).toBe(conv1.id);
      });

      it('should sort pinned by updatedAt descending among themselves', async () => {
        const conv1 = await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.togglePin(mockStorage, conv1.id);
        await new Promise((resolve) => setTimeout(resolve, 10));

        const conv2 = await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.togglePin(mockStorage, conv2.id);
        await new Promise((resolve) => setTimeout(resolve, 10));

        await ConversationStorage.createConversation(mockStorage);

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list[0].isPinned).toBe(true);
        expect(list[1].isPinned).toBe(true);
        expect(list[2].isPinned).toBe(false);

        expect(list[0].id).toBe(conv2.id);
        expect(list[1].id).toBe(conv1.id);
      });

      it('should maintain sort order after updating conversation', async () => {
        const conv1 = await ConversationStorage.createConversation(mockStorage);
        const conv2 = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.togglePin(mockStorage, conv1.id);

        await ConversationStorage.saveConversation(mockStorage, conv2);

        const list = await ConversationStorage.getConversationList(mockStorage);

        expect(list[0].id).toBe(conv1.id);
      });
    });
  });

  describe('Pruning and Trimming', () => {
    describe('pruneOldConversations', () => {
      it('should keep conversations under max limit', async () => {
        for (let i = 0; i < 40; i++) {
          await ConversationStorage.createConversation(mockStorage);
        }

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBeLessThanOrEqual(50);
      });

      it('should prune oldest unpinned conversations when exceeding max', async () => {
        const conversations: Conversation[] = [];

        for (let i = 0; i < 55; i++) {
          await new Promise((resolve) => setTimeout(resolve, 5));
          const conv = await ConversationStorage.createConversation(mockStorage);
          conversations.push(conv);
        }

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBe(50);

        const retrievedFirst = await ConversationStorage.getConversation(mockStorage, conversations[0].id);
        expect(retrievedFirst).toBeNull();
      });

      it('should always keep pinned conversations when pruning', async () => {
        const pinnedConvs: Conversation[] = [];

        for (let i = 0; i < 10; i++) {
          const conv = await ConversationStorage.createConversation(mockStorage);
          await ConversationStorage.togglePin(mockStorage, conv.id);
          pinnedConvs.push(conv);
        }

        for (let i = 0; i < 50; i++) {
          await ConversationStorage.createConversation(mockStorage);
        }

        for (const conv of pinnedConvs) {
          const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
          expect(retrieved).not.toBeNull();
          expect(retrieved!.isPinned).toBe(true);
        }
      });

      it('should keep most recent unpinned when at capacity', async () => {
        const conversations: Conversation[] = [];

        for (let i = 0; i < 5; i++) {
          const conv = await ConversationStorage.createConversation(mockStorage);
          await ConversationStorage.togglePin(mockStorage, conv.id);
        }

        for (let i = 0; i < 50; i++) {
          await new Promise((resolve) => setTimeout(resolve, 5));
          const conv = await ConversationStorage.createConversation(mockStorage);
          conversations.push(conv);
        }

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBe(50);

        const latestUnpinned = conversations[conversations.length - 1];
        const retrieved = await ConversationStorage.getConversation(mockStorage, latestUnpinned.id);
        expect(retrieved).not.toBeNull();
      });
    });

    describe('trimMessages', () => {
      it('should keep all messages when under max', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        for (let i = 0; i < 50; i++) {
          await ConversationStorage.addMessage(mockStorage, conv.id, {
            role: 'user',
            content: `Message ${i}`,
            timestamp: new Date(),
          });
        }

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.messages.length).toBe(50);
      });

      it('should trim to max messages when exceeding limit', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        for (let i = 0; i < 150; i++) {
          conv.messages.push({
            id: `msg-${i}`,
            role: 'user',
            content: `Message ${i}`,
            timestamp: new Date(),
          });
        }

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.messages.length).toBeLessThanOrEqual(100);
      });

      it('should keep most recent messages when trimming', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        for (let i = 0; i < 150; i++) {
          conv.messages.push({
            id: `msg-${i}`,
            role: 'user',
            content: `Message ${i}`,
            timestamp: new Date(),
          });
        }

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        const lastMessage = retrieved!.messages[retrieved!.messages.length - 1];
        expect(lastMessage.content).toBe('Message 149');
      });

      it('should always preserve system messages', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        conv.messages.push({
          id: 'system-1',
          role: 'system',
          content: 'System message 1',
          timestamp: new Date(),
        });

        conv.messages.push({
          id: 'system-2',
          role: 'system',
          content: 'System message 2',
          timestamp: new Date(),
        });

        for (let i = 0; i < 120; i++) {
          conv.messages.push({
            id: `msg-${i}`,
            role: 'user',
            content: `Message ${i}`,
            timestamp: new Date(),
          });
        }

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);

        const systemMessages = retrieved!.messages.filter((m) => m.role === 'system');
        expect(systemMessages.length).toBe(2);
        expect(systemMessages[0].content).toBe('System message 1');
        expect(systemMessages[1].content).toBe('System message 2');
      });

      it('should handle empty messages array', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);
        conv.messages = [];

        await ConversationStorage.saveConversation(mockStorage, conv);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.messages).toEqual([]);
      });
    });
  });

  describe('Import and Export', () => {
    describe('exportConversations', () => {
      it('should export conversations as JSON string', async () => {
        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        const exported = await ConversationStorage.exportConversations(mockStorage);

        expect(typeof exported).toBe('string');

        const parsed = JSON.parse(exported);
        expect(Array.isArray(parsed)).toBe(true);
        expect(parsed.length).toBe(2);
      });

      it('should export empty array when no conversations exist', async () => {
        const exported = await ConversationStorage.exportConversations(mockStorage);

        const parsed = JSON.parse(exported);
        expect(parsed).toEqual([]);
      });

      it('should include all conversation data', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.addMessage(mockStorage, conv.id, {
          role: 'user',
          content: 'Test message',
          timestamp: new Date(),
        });

        await ConversationStorage.togglePin(mockStorage, conv.id);

        const exported = await ConversationStorage.exportConversations(mockStorage);
        const parsed = JSON.parse(exported);

        expect(parsed[0].id).toBe(conv.id);
        expect(parsed[0].isPinned).toBe(true);
        expect(parsed[0].messages.length).toBe(1);
      });

      it('should format JSON with indentation', async () => {
        await ConversationStorage.createConversation(mockStorage);

        const exported = await ConversationStorage.exportConversations(mockStorage);

        expect(exported).toContain('\n');
        expect(exported).toContain('  ');
      });
    });

    describe('importConversations', () => {
      it('should import valid JSON data', async () => {
        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        const exported = await ConversationStorage.exportConversations(mockStorage);

        await ConversationStorage.clearAll(mockStorage);

        await ConversationStorage.importConversations(mockStorage, exported);

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBe(2);
      });

      it('should parse dates correctly when importing', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.addMessage(mockStorage, conv.id, {
          role: 'user',
          content: 'Test',
          timestamp: new Date(),
        });

        const exported = await ConversationStorage.exportConversations(mockStorage);

        await ConversationStorage.clearAll(mockStorage);

        await ConversationStorage.importConversations(mockStorage, exported);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.createdAt).toBeInstanceOf(Date);
        expect(retrieved!.updatedAt).toBeInstanceOf(Date);
        expect(retrieved!.messages[0].timestamp).toBeInstanceOf(Date);
      });

      it('should throw error for invalid JSON', async () => {
        await expect(ConversationStorage.importConversations(mockStorage, 'invalid json')).rejects.toThrow();
      });

      it('should throw error for non-array JSON', async () => {
        await expect(ConversationStorage.importConversations(mockStorage, '{"invalid": "format"}')).rejects.toThrow(
          'Invalid format'
        );
      });

      it('should replace existing conversations when importing', async () => {
        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        const exported = await ConversationStorage.exportConversations(mockStorage);

        await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.createConversation(mockStorage);

        await ConversationStorage.importConversations(mockStorage, exported);

        const list = await ConversationStorage.getConversationList(mockStorage);
        expect(list.length).toBe(2);
      });

      it('should preserve all conversation properties', async () => {
        const conv = await ConversationStorage.createConversation(mockStorage);
        await ConversationStorage.updateTitle(mockStorage, conv.id, 'Custom Title');
        await ConversationStorage.togglePin(mockStorage, conv.id);

        await ConversationStorage.addMessage(mockStorage, conv.id, {
          role: 'user',
          content: 'Message 1',
          timestamp: new Date(),
          tokens: 10,
          cost: 0.001,
        });

        const exported = await ConversationStorage.exportConversations(mockStorage);

        await ConversationStorage.clearAll(mockStorage);

        await ConversationStorage.importConversations(mockStorage, exported);

        const retrieved = await ConversationStorage.getConversation(mockStorage, conv.id);
        expect(retrieved!.title).toBe('Custom Title');
        expect(retrieved!.isPinned).toBe(true);
        expect(retrieved!.messages[0].tokens).toBe(10);
        expect(retrieved!.messages[0].cost).toBe(0.001);
      });
    });
  });

  describe('Edge Cases and Error Handling', () => {
    it('should handle corrupted storage data gracefully', async () => {
      storedData.set('zagalin-conversations', 'corrupted data');

      const list = await ConversationStorage.getConversationList(mockStorage);
      expect(list).toEqual([]);
    });

    it('should handle malformed conversation objects', async () => {
      storedData.set('zagalin-conversations', JSON.stringify([{ malformed: 'object' }]));

      await expect(ConversationStorage.getConversationList(mockStorage)).resolves.not.toThrow();
    });

    it('should handle legacy context format', async () => {
      const legacyFormat = [
        {
          id: 'conv-1',
          title: 'Legacy Conv',
          messages: [],
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          isPinned: false,
          context: {
            dashboardUid: 'dash-123',
            dashboardTitle: 'Test Dashboard',
            panelId: 1,
          },
        },
      ];

      storedData.set('zagalin-conversations', JSON.stringify(legacyFormat));

      const retrieved = await ConversationStorage.getConversation(mockStorage, 'conv-1');
      expect(retrieved!.contexts).toHaveLength(1);
      expect(retrieved!.contexts[0].dashboardUid).toBe('dash-123');
    });

    it('should handle missing context gracefully', async () => {
      const noContext = [
        {
          id: 'conv-1',
          title: 'No Context',
          messages: [],
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
          isPinned: false,
        },
      ];

      storedData.set('zagalin-conversations', JSON.stringify(noContext));

      const retrieved = await ConversationStorage.getConversation(mockStorage, 'conv-1');
      expect(retrieved!.contexts).toEqual([]);
    });
  });
});
