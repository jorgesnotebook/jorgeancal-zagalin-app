import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import {
  Button,
  IconButton,
  Input,
  useStyles2,
  Tooltip,
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
  onDeleteAll: () => void;
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
          <Tooltip content={conversation.lastMessagePreview || 'No messages yet'} placement="right">
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
  onDeleteAll,
  onTogglePin,
  onCreateNew,
}: ConversationListSidebarProps) {
  const s = useStyles2(getSidebarStyles);
  const [searchTerm, setSearchTerm] = useState('');
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [showDeleteAll, setShowDeleteAll] = useState(false);

  // Filter conversations by search term
  const filteredConversations = conversations.filter((conv) =>
    conv.title.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleDelete = (id: string) => {
    setDeletingId(id);
  };

  const confirmDelete = () => {
    if (deletingId) {
      console.log('Deleting conversation:', deletingId);
      onDeleteConversation(deletingId);
      setDeletingId(null);
    }
  };

  const cancelDelete = () => {
    setDeletingId(null);
  };

  const handleDeleteAll = () => {
    setShowDeleteAll(true);
  };

  const confirmDeleteAll = () => {
    console.log('Deleting all conversations');
    onDeleteAll();
    setShowDeleteAll(false);
  };

  const cancelDeleteAll = () => {
    setShowDeleteAll(false);
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
        {conversations.length > 0 && (
          <Button
            icon="trash-alt"
            onClick={handleDeleteAll}
            size="sm"
            fullWidth
            variant="destructive"
            className={s.deleteAllButton}
          >
            Delete All
          </Button>
        )}
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
        <>
          <div className={s.deleteOverlay} onClick={cancelDelete} />
          <div className={s.deleteModal}>
            <div className={s.deleteModalContent}>
              <h4 className={s.deleteModalTitle}>Delete Conversation?</h4>
              <p className={s.deleteModalText}>
                Are you sure you want to delete &quot;{deletingConversation?.title}&quot;? This action cannot be undone.
              </p>
              <div className={s.deleteButtons}>
                <Button variant="destructive" size="md" onClick={confirmDelete} icon="trash-alt">
                  Delete
                </Button>
                <Button variant="secondary" size="md" onClick={cancelDelete}>
                  Cancel
                </Button>
              </div>
            </div>
          </div>
        </>
      )}

      {showDeleteAll && (
        <>
          <div className={s.deleteOverlay} onClick={cancelDeleteAll} />
          <div className={s.deleteModal}>
            <div className={s.deleteModalContent}>
              <h4 className={s.deleteModalTitle}>Delete All Conversations?</h4>
              <p className={s.deleteModalText}>
                Are you sure you want to delete ALL {conversations.length} conversation{conversations.length !== 1 ? 's' : ''}?
                This will permanently remove all your chat history. This action cannot be undone.
              </p>
              <div className={s.deleteButtons}>
                <Button variant="destructive" size="md" onClick={confirmDeleteAll} icon="trash-alt">
                  Delete All
                </Button>
                <Button variant="secondary" size="md" onClick={cancelDeleteAll}>
                  Cancel
                </Button>
              </div>
            </div>
          </div>
        </>
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
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
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
  deleteAllButton: css`
    margin-top: ${theme.spacing(0.5)};
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
  deleteOverlay: css`
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 2000;
    backdrop-filter: blur(2px);
  `,
  deleteModal: css`
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    z-index: 2001;
    min-width: 400px;
    max-width: 500px;
  `,
  deleteModalContent: css`
    background: ${theme.colors.background.primary};
    border: 2px solid ${theme.colors.error.border};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(3)};
    box-shadow: ${theme.shadows.z3};
  `,
  deleteModalTitle: css`
    margin: 0 0 ${theme.spacing(2)} 0;
    font-size: ${theme.typography.h4.fontSize};
    font-weight: ${theme.typography.fontWeightBold};
    color: ${theme.colors.error.text};
  `,
  deleteModalText: css`
    margin: 0 0 ${theme.spacing(3)} 0;
    color: ${theme.colors.text.primary};
    line-height: 1.5;
  `,
  deleteButtons: css`
    display: flex;
    gap: ${theme.spacing(2)};
    justify-content: flex-end;
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
    opacity: 0.7;
    transition: opacity 0.2s;

    &:hover {
      opacity: 1;
    }
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
