import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import {
  Button,
  IconButton,
  Input,
  useStyles2,
  Tooltip,
  Alert,
  Badge,
} from '@grafana/ui';
import type { ConversationMetadata } from '../../services/conversationStorage';
import { ZagalinColors } from '../../theme/colors';

interface ConversationListSidebarProps {
  conversations: ConversationMetadata[];
  currentId: string | null;
  onSelectConversation: (id: string) => void;
  onRenameConversation: (id: string, title: string) => void;
  onDeleteConversation: (id: string) => void;
  onTogglePin: (id: string) => void;
  onCreateNew: () => void;
}

interface ConversationItemProps {
  conversation: ConversationMetadata;
  isActive: boolean;
  onSelect: () => void;
  onRename: (newTitle: string) => void;
  onDelete: () => void;
  onTogglePin: () => void;
}

function ConversationItem({
  conversation,
  isActive,
  onSelect,
  onRename,
  onDelete,
  onTogglePin,
}: ConversationItemProps) {
  const s = useStyles2(getItemStyles);
  const [isEditing, setIsEditing] = useState(false);
  const [editTitle, setEditTitle] = useState(conversation.title);

  const handleStartEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    setEditTitle(conversation.title);
    setIsEditing(true);
  };

  const handleSaveEdit = () => {
    const trimmed = editTitle.trim();
    if (trimmed && trimmed !== conversation.title) {
      onRename(trimmed);
    }
    setIsEditing(false);
  };

  const handleCancelEdit = () => {
    setEditTitle(conversation.title);
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSaveEdit();
    } else if (e.key === 'Escape') {
      handleCancelEdit();
    }
  };

  const formatTime = (date: Date | string) => {
    const now = new Date();
    const dateObj = typeof date === 'string' ? new Date(date) : date;
    const diff = now.getTime() - dateObj.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) {return 'Just now';}
    if (minutes < 60) {return `${minutes}m ago`;}
    if (hours < 24) {return `${hours}h ago`;}
    if (days < 7) {return `${days}d ago`;}
    return dateObj.toLocaleDateString();
  };

  const className = `${s.item} ${isActive ? s.activeItem : ''} ${
    conversation.isPinned ? s.pinnedItem : ''
  }`;

  return (
    <div className={className}>
      {isEditing ? (
        <Input
          value={editTitle}
          onChange={(e) => setEditTitle(e.currentTarget.value)}
          onBlur={handleSaveEdit}
          onKeyDown={handleKeyDown}
          autoFocus
          className={s.editInput}
        />
      ) : (
        <>
          <Tooltip content={conversation.preview || 'No messages yet'} placement="right">
            <div className={s.itemContent} onClick={onSelect}>
              <div className={s.itemTitle}>{conversation.title}</div>
              <div className={s.itemMeta}>
                {formatTime(conversation.updatedAt)}
              </div>
            </div>
          </Tooltip>
          <div className={s.itemActions}>
            {conversation.messageCount > 0 && (
              <Badge text={conversation.messageCount.toString()} color="blue" />
            )}
            <IconButton
              name="star"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onTogglePin();
              }}
              tooltip={conversation.isPinned ? 'Unpin' : 'Pin'}
              className={conversation.isPinned ? s.pinnedStar : s.unpinnedStar}
            />
            <IconButton
              name="edit"
              size="sm"
              onClick={handleStartEdit}
              tooltip="Rename"
            />
            <IconButton
              name="trash-alt"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              tooltip="Delete"
              className={s.deleteButton}
            />
          </div>
        </>
      )}
    </div>
  );
}

export function ConversationListSidebar({
  conversations,
  currentId,
  onSelectConversation,
  onRenameConversation,
  onDeleteConversation,
  onTogglePin,
  onCreateNew,
}: ConversationListSidebarProps) {
  const s = useStyles2(getSidebarStyles);
  const [searchTerm, setSearchTerm] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);

  // Filter conversations by search term
  const filteredConversations = conversations.filter((conv) =>
    conv.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleDelete = (id: string) => {
    setDeletingId(id);
  };

  const confirmDelete = () => {
    if (deletingId) {
      onDeleteConversation(deletingId);
      setDeletingId(null);
    }
  };

  const cancelDelete = () => {
    setDeletingId(null);
  };

  const deletingConversation = conversations.find((c) => c.id === deletingId);

  return (
    <div className={s.container}>
      <div className={s.header}>
        <Button
          icon="plus"
          onClick={onCreateNew}
          size="sm"
          fullWidth
          variant="secondary"
          className={s.newChatButton}
        >
          New Chat
        </Button>
      </div>

      {conversations.length > 5 && (
        <div className={s.searchContainer}>
          <Input
            placeholder="Search..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.currentTarget.value)}
            prefix={<span className={s.searchIcon}>🔍</span>}
            className={s.searchInput}
          />
        </div>
      )}

      <div className={s.listContainer}>
        {filteredConversations.length === 0 ? (
          <div className={s.emptyState}>
            {searchTerm ? 'No conversations found' : 'No conversations yet'}
          </div>
        ) : (
          filteredConversations.map((conversation) => (
            <ConversationItem
              key={conversation.id}
              conversation={conversation}
              isActive={conversation.id === currentId}
              onSelect={() => onSelectConversation(conversation.id)}
              onRename={(newTitle) => onRenameConversation(conversation.id, newTitle)}
              onDelete={() => handleDelete(conversation.id)}
              onTogglePin={() => onTogglePin(conversation.id)}
            />
          ))
        )}
      </div>

      {deletingId && (
        <div className={s.deleteConfirm}>
          <Alert
            title={`Delete "${deletingConversation?.title}"?`}
            severity="warning"
            onRemove={cancelDelete}
          >
            <div className={s.deleteButtons}>
              <Button variant="destructive" size="sm" onClick={confirmDelete}>
                Delete
              </Button>
              <Button variant="secondary" size="sm" onClick={cancelDelete}>
                Cancel
              </Button>
            </div>
          </Alert>
        </div>
      )}
    </div>
  );
}

const getSidebarStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 200px;
    border-right: 1px solid ${theme.colors.border.weak};
    background: ${theme.colors.background.secondary};
  `,
  header: css`
    padding: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  newChatButton: css`
    background: ${ZagalinColors.orangeGradient} !important;
    border: none !important;
    color: white !important;

    &:hover:not(:disabled) {
      background: ${ZagalinColors.orangeGradientHover} !important;
      box-shadow: 0 2px 4px rgba(255, 152, 48, 0.3);
    }
  `,
  searchContainer: css`
    padding: ${theme.spacing(1)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  searchInput: css`
    width: 100%;
  `,
  searchIcon: css`
    margin-right: ${theme.spacing(0.5)};
  `,
  listContainer: css`
    flex: 1;
    overflow-y: auto;
    padding: ${theme.spacing(0.5)};

    /* Scrollbar styling */
    scrollbar-width: thin;
    &::-webkit-scrollbar {
      width: 6px;
    }
    &::-webkit-scrollbar-thumb {
      background: ${theme.colors.border.medium};
      border-radius: 3px;
    }
    &::-webkit-scrollbar-track {
      background: transparent;
    }
  `,
  emptyState: css`
    padding: ${theme.spacing(3, 2)};
    text-align: center;
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  deleteConfirm: css`
    position: absolute;
    bottom: ${theme.spacing(2)};
    left: ${theme.spacing(1)};
    right: ${theme.spacing(1)};
    z-index: 1000;
  `,
  deleteButtons: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(1)};
  `,
});

const getItemStyles = (theme: GrafanaTheme2) => ({
  item: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(0.5)};
    border-radius: ${theme.shape.radius.default};
    cursor: pointer;
    transition: background-color 0.2s;
    position: relative;

    &:hover {
      background: ${theme.colors.background.primary};

      .item-actions {
        opacity: 1;
      }
    }
  `,
  activeItem: css`
    background: ${theme.colors.emphasize(theme.colors.background.primary, 0.15)} !important;
    border-left: 3px solid ${ZagalinColors.orange};
    padding-left: calc(${theme.spacing(1)} - 3px);
  `,
  pinnedItem: css`
    border-left: 2px solid ${ZagalinColors.orange};
    padding-left: calc(${theme.spacing(1)} - 2px);
  `,
  itemContent: css`
    flex: 1;
    min-width: 0;
    overflow: hidden;
  `,
  itemTitle: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: ${theme.colors.text.primary};
    margin-bottom: ${theme.spacing(0.25)};
  `,
  itemMeta: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  `,
  itemActions: css`
    display: flex;
    gap: ${theme.spacing(0.25)};
    align-items: center;
    opacity: 0;
    transition: opacity 0.2s;
  `,
  editInput: css`
    width: 100%;
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  pinnedStar: css`
    color: ${ZagalinColors.orange} !important;
    opacity: 1 !important;
  `,
  unpinnedStar: css`
    color: ${theme.colors.text.secondary};
  `,
  deleteButton: css`
    &:hover {
      color: ${theme.colors.error.text} !important;
    }
  `,
});
