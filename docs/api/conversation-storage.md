# Conversation Storage API

This document describes the ConversationStorage service API for managing persistent chat conversations in Zagalin.

## Overview

The `ConversationStorage` service provides localStorage-based persistence for chat conversations, including automatic pruning, message trimming, and conversation management features.

**Location**: `src/services/conversationStorage.ts`

## Data Types

### StoredMessage

```typescript
interface StoredMessage {
  id: string;                    // Unique message identifier
  role: 'user' | 'assistant' | 'system';
  content: string;               // Message text
  timestamp: Date;               // When message was created
  tokens?: number;               // Token count (optional)
  cost?: number;                 // Estimated cost (optional)
}
```

### Conversation

```typescript
interface Conversation {
  id: string;                    // Unique conversation identifier
  title: string;                 // Conversation title
  messages: StoredMessage[];     // Array of messages
  createdAt: Date;              // Creation timestamp
  updatedAt: Date;              // Last modification timestamp
  isPinned: boolean;            // Pin status
  context?: {                   // Optional Grafana context
    dashboardUid?: string;
    dashboardTitle?: string;
    panelId?: number;
    panelTitle?: string;
  };
}
```

### ConversationMetadata

```typescript
interface ConversationMetadata {
  id: string;                    // Conversation identifier
  title: string;                 // Conversation title
  messageCount: number;          // Number of messages
  lastMessagePreview: string;    // Preview of last message (100 chars)
  updatedAt: Date;              // Last update time
  isPinned: boolean;            // Pin status
}
```

## Constants

```typescript
const STORAGE_KEY = 'zagalin-conversations';       // localStorage key
const MAX_CONVERSATIONS = 50;                      // Auto-prune threshold
const MAX_MESSAGES_PER_CONVERSATION = 100;        // Message limit
```

## Static Methods

### getConversationList()

Returns metadata for all conversations, sorted by pin status and update time.

```typescript
static getConversationList(): ConversationMetadata[]
```

**Returns**: Array of conversation metadata

**Sorting**: Pinned conversations first, then by updatedAt (newest first)

**Example**:
```typescript
const conversations = ConversationStorage.getConversationList();

conversations.forEach(conv => {
  console.log(`${conv.title} - ${conv.messageCount} messages`);
});
```

### getConversation(id)

Retrieves a full conversation by ID.

```typescript
static getConversation(id: string): Conversation | null
```

**Parameters**:
- `id`: Conversation identifier

**Returns**: Conversation object or `null` if not found

**Example**:
```typescript
const conversation = ConversationStorage.getConversation('abc123');

if (conversation) {
  console.log(`Loaded: ${conversation.title}`);
  console.log(`Messages: ${conversation.messages.length}`);
}
```

### createConversation(context?)

Creates a new conversation with optional Grafana context.

```typescript
static createConversation(context?: Conversation['context']): Conversation
```

**Parameters**:
- `context`: Optional Grafana context (dashboard, panel info)

**Returns**: Newly created conversation

**Side Effects**:
- Saves to localStorage
- Triggers auto-prune if > 50 conversations

**Example**:
```typescript
const conversation = ConversationStorage.createConversation({
  dashboardUid: 'd/abc123',
  dashboardTitle: 'My Dashboard',
  panelId: 5,
  panelTitle: 'CPU Usage',
});

console.log(`Created: ${conversation.id}`);
```

### saveConversation(conversation)

Saves or updates a conversation.

```typescript
static saveConversation(conversation: Conversation): void
```

**Parameters**:
- `conversation`: Conversation object to save

**Side Effects**:
- Trims messages if > 100
- Auto-generates title from first message if needed
- Updates `updatedAt` timestamp
- Saves to localStorage
- Triggers auto-prune if needed

**Example**:
```typescript
conversation.messages.push({
  id: generateId(),
  role: 'user',
  content: 'Hello',
  timestamp: new Date(),
});

ConversationStorage.saveConversation(conversation);
```

### deleteConversation(id)

Deletes a conversation permanently.

```typescript
static deleteConversation(id: string): void
```

**Parameters**:
- `id`: Conversation identifier to delete

**Returns**: void

**Example**:
```typescript
ConversationStorage.deleteConversation('abc123');
console.log('Conversation deleted');
```

### updateTitle(id, title)

Updates conversation title.

```typescript
static updateTitle(id: string, title: string): void
```

**Parameters**:
- `id`: Conversation identifier
- `title`: New title (max 50 characters, trimmed)

**Validation**:
- Trims whitespace
- Truncates to 50 characters
- Falls back to "Untitled Chat" if empty

**Side Effects**:
- Updates `updatedAt` timestamp
- Saves to localStorage

**Example**:
```typescript
ConversationStorage.updateTitle('abc123', 'My Important Chat');
```

### togglePin(id)

Toggles conversation pin status.

```typescript
static togglePin(id: string): void
```

**Parameters**:
- `id`: Conversation identifier

**Side Effects**:
- Flips `isPinned` boolean
- Updates `updatedAt` timestamp
- Pinned conversations protected from auto-prune
- Changes sorting position in list

**Example**:
```typescript
ConversationStorage.togglePin('abc123');

const conversation = ConversationStorage.getConversation('abc123');
console.log(`Pinned: ${conversation?.isPinned}`);
```

### addMessage(conversationId, message)

Adds a message to a conversation.

```typescript
static addMessage(
  conversationId: string,
  message: Omit<StoredMessage, 'id'>
): void
```

**Parameters**:
- `conversationId`: Target conversation ID
- `message`: Message object (without `id` field)

**Side Effects**:
- Generates unique message ID
- Appends to conversation
- Calls `saveConversation` (triggers trimming, auto-prune)

**Example**:
```typescript
ConversationStorage.addMessage('abc123', {
  role: 'assistant',
  content: 'Hello! How can I help?',
  timestamp: new Date(),
  tokens: 15,
});
```

### clearAll()

Removes all conversations from storage.

```typescript
static clearAll(): void
```

**Warning**: This is destructive and cannot be undone!

**Example**:
```typescript
if (confirm('Delete all conversations?')) {
  ConversationStorage.clearAll();
}
```

### exportConversations()

Exports all conversations as JSON string.

```typescript
static exportConversations(): string
```

**Returns**: JSON string of all conversations

**Use Case**: Backup, data portability

**Example**:
```typescript
const backup = ConversationStorage.exportConversations();

// Download as file
const blob = new Blob([backup], { type: 'application/json' });
const url = URL.createObjectURL(blob);
const a = document.createElement('a');
a.href = url;
a.download = 'zagalin-backup.json';
a.click();
```

### importConversations(jsonData)

Imports conversations from JSON string.

```typescript
static importConversations(jsonData: string): void
```

**Parameters**:
- `jsonData`: JSON string from `exportConversations()`

**Validation**:
- Validates JSON format
- Converts date strings to Date objects
- Throws error if invalid format

**Side Effects**:
- Replaces all existing conversations
- Saves to localStorage

**Example**:
```typescript
try {
  const json = await file.text();
  ConversationStorage.importConversations(json);
  alert('Import successful!');
} catch (error) {
  alert('Import failed: ' + error.message);
}
```

## Internal Functions

### generateId()

Generates unique IDs for conversations and messages.

```typescript
function generateId(): string
```

**Returns**: Timestamp-based unique ID

**Format**: `{timestamp}-{random}`

**Example**: `"1640995200000-a3f7k2"`

### generateTitle(messages)

Auto-generates conversation title from first user message.

```typescript
function generateTitle(messages: StoredMessage[]): string
```

**Logic**:
1. Find first user message
2. Truncate to 50 characters
3. Add "..." if truncated
4. Fallback: "Chat from {timestamp}"

**Example**:
```typescript
// First message: "How do I create a dashboard with multiple panels?"
// Generated title: "How do I create a dashboard with multiple pan..."
```

### loadAllConversations()

Loads conversations from localStorage.

```typescript
function loadAllConversations(): Conversation[]
```

**Error Handling**:
- Returns empty array on error
- Logs errors to console
- Converts date strings to Date objects

### saveAllConversations(conversations)

Saves all conversations to localStorage.

```typescript
function saveAllConversations(conversations: Conversation[]): void
```

**Error Handling**:
- Throws error if storage quota exceeded
- Logs errors to console

### pruneOldConversations(conversations)

Auto-prunes conversations when limit exceeded.

```typescript
function pruneOldConversations(conversations: Conversation[]): Conversation[]
```

**Logic**:
1. If ≤ 50 conversations, return unchanged
2. Separate pinned and unpinned
3. Sort unpinned by updatedAt (oldest first)
4. Keep newest unpinned to fit limit
5. Return pinned + kept unpinned

**Example**:
```typescript
// 51 conversations: 5 pinned, 46 unpinned
// Result: 5 pinned + 45 newest unpinned = 50 total
// Oldest unpinned conversation is removed
```

### trimMessages(messages)

Trims messages when limit exceeded.

```typescript
function trimMessages(messages: StoredMessage[]): StoredMessage[]
```

**Logic**:
1. If ≤ 100 messages, return unchanged
2. Keep all system messages
3. Keep most recent non-system messages
4. Total stays under 100

**Example**:
```typescript
// 105 messages: 5 system, 100 user/assistant
// Result: 5 system + 95 newest user/assistant = 100 total
```

## Usage Patterns

### Creating and Managing Conversations

```typescript
// Create new conversation
const conv = ConversationStorage.createConversation({
  dashboardUid: currentDashboard.uid,
  dashboardTitle: currentDashboard.title,
});

// Add messages
ConversationStorage.addMessage(conv.id, {
  role: 'user',
  content: 'What is CPU usage?',
  timestamp: new Date(),
});

ConversationStorage.addMessage(conv.id, {
  role: 'assistant',
  content: 'CPU usage is...',
  timestamp: new Date(),
});

// Update title
ConversationStorage.updateTitle(conv.id, 'CPU Questions');

// Pin important conversation
ConversationStorage.togglePin(conv.id);
```

### Loading Conversations

```typescript
// Get all conversations
const list = ConversationStorage.getConversationList();

// Filter by search term
const filtered = list.filter(c =>
  c.title.toLowerCase().includes(searchTerm.toLowerCase())
);

// Load full conversation
const conversation = ConversationStorage.getConversation(selectedId);

if (conversation) {
  conversation.messages.forEach(msg => {
    console.log(`${msg.role}: ${msg.content}`);
  });
}
```

### Backup and Restore

```typescript
// Backup
const backup = ConversationStorage.exportConversations();
localStorage.setItem('zagalin-backup', backup);

// Restore
const backup = localStorage.getItem('zagalin-backup');
if (backup) {
  ConversationStorage.importConversations(backup);
}
```

## Error Handling

### localStorage Quota Exceeded

```typescript
try {
  ConversationStorage.saveConversation(conversation);
} catch (error) {
  if (error.message.includes('quota')) {
    alert('Storage full! Please delete old conversations.');
  }
}
```

### Invalid Conversation ID

```typescript
const conversation = ConversationStorage.getConversation(invalidId);

if (!conversation) {
  console.error('Conversation not found');
  // Handle gracefully
}
```

### Corrupted Data

```typescript
// loadAllConversations handles corrupted data gracefully
const conversations = ConversationStorage.getConversationList();
// Returns empty array if data is corrupted
```

## Performance Considerations

### localStorage Limits

- **Browser Limit**: ~5-10 MB
- **50 conversations × 100 messages**: ~2-3 MB
- **Well within limits**: Plenty of headroom

### Optimization Tips

1. **Auto-pruning**: Keeps storage size manageable
2. **Message trimming**: Prevents unlimited growth
3. **Lazy loading**: Load full conversation only when needed
4. **Metadata caching**: Use `getConversationList()` for lists

## See Also

- [useConversation Hook](./use-conversation-hook.md)
- [Conversation History Guide](../development/conversation-history.md)
- [Storage Best Practices](../development/storage-best-practices.md)
