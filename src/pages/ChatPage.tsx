import React, { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import {
  Button,
  TextArea,
  useStyles2,
  IconButton,
  Spinner,
  Alert,
  Badge,
  Tooltip,
} from '@grafana/ui';
import { testIds } from '../components/testIds';
import { useGrafanaContext } from '../services/useGrafanaContext';
import { ContextService } from '../services/contextService';
import { detectSkill } from '../services/assistantSkills';
import { extractActions, createExploreLink } from '../services/actionExtractor';
import type { AssistantAction } from '../services/contextTypes';

interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  tokens?: number;
  cost?: number;
  actions?: AssistantAction[];
}

function ChatPage() {
  const s = useStyles2(getStyles);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [streamingContent, setStreamingContent] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Get Grafana context
  const { context, hasContext, loading: contextLoading } = useGrafanaContext();

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingContent]);

  const handleSend = async () => {
    if (!input.trim() || isStreaming) {return;}

    const userMessage: Message = {
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    // Add user message
    setMessages(prev => [...prev, userMessage]);

    setInput('');
    setIsStreaming(true);
    setError(null);
    setStreamingContent('');

    try {
      const controller = new AbortController();
      abortControllerRef.current = controller;

      // Detect if a skill should be used
      const skill = detectSkill(userMessage.content, context);

      // Build messages array
      let allMessages;
      if (skill) {
        // Use skill-specific prompts
        allMessages = [
          { role: 'system' as const, content: skill.systemPrompt },
          { role: 'user' as const, content: skill.userPrompt },
        ];
      } else {
        // Normal conversation with context injection
        allMessages = [...messages, userMessage].map(m => ({
          role: m.role,
          content: m.content,
        }));

        // Inject Grafana context as system message if available
        if (hasContext) {
          const contextPrompt = ContextService.formatContextPrompt(context);
          if (contextPrompt) {
            allMessages.unshift({
              role: 'system',
              content: contextPrompt,
            });
          }
        }
      }

      // Use streaming endpoint (provider will be determined by backend default config)
      const response = await fetch(
        `/api/plugins/jorgeancal-zagalin-app/resources/chat/stream`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            messages: allMessages,
          }),
          signal: controller.signal,
        }
      );

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();
      let accumulatedContent = '';
      let lastUsage: any = null;

      if (reader) {
        while (true) {
          const { done, value } = await reader.read();
          if (done) {break;}

          const chunk = decoder.decode(value);
          const lines = chunk.split('\n');

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6);
              try {
                const parsed = JSON.parse(data);
                if (parsed.error) {
                  throw new Error(parsed.error);
                }
                if (parsed.delta) {
                  accumulatedContent += parsed.delta;
                  setStreamingContent(accumulatedContent);
                }
                if (parsed.usage) {
                  lastUsage = parsed.usage;
                }
              } catch (e) {
                // Ignore parse errors for incomplete chunks
              }
            }
          }
        }
      }

      // Extract actions from the response
      const actions = extractActions(accumulatedContent);

      // Add assistant message
      const assistantMessage: Message = {
        role: 'assistant',
        content: accumulatedContent,
        timestamp: new Date(),
        tokens: lastUsage?.total_tokens,
        cost: lastUsage?.estimated_cost_usd,
        actions: actions.length > 0 ? actions : undefined,
      };

      setMessages(prev => [...prev, assistantMessage]);
      setStreamingContent('');
    } catch (err: any) {
      if (err.name !== 'AbortError') {
        setError(err.message || 'Failed to send message');
      }
    } finally {
      setIsStreaming(false);
      abortControllerRef.current = null;
    }
  };

  const handleStop = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      setIsStreaming(false);
      setStreamingContent('');
    }
  };

  const handleRedo = () => {
    if (messages.length < 2) {return;}

    // Remove last assistant message and resend last user message
    const lastUserMessage = [...messages]
      .reverse()
      .find(m => m.role === 'user');

    if (lastUserMessage) {
      // Remove the last assistant message
      setMessages(prev => prev.slice(0, -1));

      // Set input and trigger send
      setInput(lastUserMessage.content);
      setTimeout(() => handleSend(), 100);
    }
  };

  const handleKeyPress = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Cmd/Ctrl + Enter to send
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      handleSend();
    }
  };

  const copyToClipboard = (content: string) => {
    navigator.clipboard.writeText(content);
  };

  const getContextTooltip = () => {
    const parts: string[] = [];
    if (context.dashboard) {
      parts.push(`Dashboard: ${context.dashboard.title}`);
    }
    if (context.panel) {
      parts.push(`Panel: ${context.panel.title}`);
    }
    if (context.timeRange) {
      parts.push(`Time: ${context.timeRange.from} to ${context.timeRange.to}`);
    }
    return parts.length > 0 ? parts.join('\n') : 'Context available';
  };

  const handleAction = (action: AssistantAction) => {
    switch (action.type) {
      case 'explore':
      case 'query':
        if (action.data.query) {
          const link = createExploreLink(action.data.query, action.data.datasourceUid, context.timeRange);
          window.open(link, '_blank');
        }
        break;
      case 'copy':
        if (action.data.query) {
          copyToClipboard(action.data.query);
        }
        break;
      case 'dashboard':
        if (action.data.dashboardUid) {
          window.location.href = `/d/${action.data.dashboardUid}`;
        }
        break;
      case 'panel':
        if (action.data.dashboardUid && action.data.panelId) {
          window.location.href = `/d/${action.data.dashboardUid}?viewPanel=${action.data.panelId}`;
        }
        break;
    }
  };

  const renderActions = (actions: AssistantAction[]) => {
    return (
      <div className={s.actionButtons}>
        {actions.map((action, idx) => (
          <Button
            key={idx}
            size="sm"
            variant="secondary"
            icon={action.type === 'explore' || action.type === 'query' ? 'compass' : action.type === 'copy' ? 'copy' : 'link'}
            onClick={() => handleAction(action)}
          >
            {action.label}
          </Button>
        ))}
      </div>
    );
  };

  return (
      <div className={s.container} data-testid={testIds.chatPage?.container}>
        <div className={s.chatArea}>
          <div className={s.header}>
            <h3>Chat with AI</h3>
            <div className={s.headerActions}>
              {hasContext && (
                <Tooltip content={getContextTooltip()}>
                  <Badge color="green" text="Context Active" icon="info-circle" />
                </Tooltip>
              )}
              {contextLoading && <Spinner inline size={16} />}
            </div>
          </div>

          {error && (
            <Alert title="Error" severity="error" onRemove={() => setError(null)}>
              {error}
            </Alert>
          )}

          <div className={s.messagesContainer}>
            {messages.map((message, idx) => (
              <div
                key={idx}
                className={`${s.message} ${
                  message.role === 'user' ? s.userMessage : s.assistantMessage
                }`}
              >
                <div className={s.messageHeader}>
                  <strong>{message.role === 'user' ? 'You' : 'Zagalin'}</strong>
                  {message.cost !== undefined && (
                    <span className={s.messageCost}>${message.cost.toFixed(4)}</span>
                  )}
                </div>
                <div className={s.messageContent}>{message.content}</div>
                {message.actions && message.actions.length > 0 && renderActions(message.actions)}
                <div className={s.messageActions}>
                  <IconButton
                    name="copy"
                    tooltip="Copy to clipboard"
                    onClick={() => copyToClipboard(message.content)}
                  />
                </div>
              </div>
            ))}

            {streamingContent && (
              <div className={`${s.message} ${s.assistantMessage} ${s.streamingMessage}`}>
                <div className={s.messageHeader}>
                  <strong>Zagalin</strong>
                  <Spinner inline />
                </div>
                <div className={s.messageContent}>{streamingContent}</div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          <div className={s.inputArea}>
            <TextArea
              value={input}
              onChange={e => setInput(e.currentTarget.value)}
              onKeyDown={handleKeyPress}
              placeholder="Type your message... (Cmd/Ctrl+Enter to send)"
              rows={4}
              className={s.input}
              disabled={isStreaming}
            />

            <div className={s.inputActions}>
              <div className={s.keyboardHint}>
                Press <kbd>Cmd/Ctrl + Enter</kbd> to send
              </div>
              <div className={s.actionButtons}>
                {isStreaming ? (
                  <Button
                    icon="times"
                    onClick={handleStop}
                    variant="destructive"
                  >
                    Stop
                  </Button>
                ) : (
                  <>
                    <Button
                      icon="history"
                      onClick={handleRedo}
                      variant="secondary"
                      disabled={messages.length < 2}
                      tooltip="Redo last message"
                    />
                    <Button
                      icon="comment-alt"
                      onClick={handleSend}
                      variant="primary"
                      disabled={!input.trim()}
                    >
                      Send
                    </Button>
                  </>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
  );
}

export default ChatPage;

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    height: 100vh;
    width: 100%;
  `,
  chatArea: css`
    flex: 1;
    display: flex;
    flex-direction: column;
    background: ${theme.colors.background.primary};
    border-radius: ${theme.shape.radius.default};
    overflow: hidden;
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: ${theme.spacing(2)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  headerActions: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
  `,
  controls: css`
    display: flex;
    gap: ${theme.spacing(1)};
    align-items: center;
  `,
  messagesContainer: css`
    flex: 1;
    overflow-y: auto;
    padding: ${theme.spacing(2)};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
  `,
  message: css`
    padding: ${theme.spacing(2)};
    border-radius: ${theme.shape.radius.default};
    max-width: 80%;
  `,
  userMessage: css`
    align-self: flex-end;
    background: ${theme.colors.primary.main};
    color: ${theme.colors.primary.contrastText};
  `,
  assistantMessage: css`
    align-self: flex-start;
    background: ${theme.colors.background.secondary};
  `,
  streamingMessage: css`
    animation: pulse 1.5s ease-in-out infinite;
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.8; }
    }
  `,
  messageHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    margin-bottom: ${theme.spacing(1)};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  messageModel: css`
    background: ${theme.colors.background.canvas};
    padding: ${theme.spacing(0.5, 1)};
    border-radius: ${theme.shape.radius.default};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  messageCost: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  messageContent: css`
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.6;
  `,
  messageActions: css`
    display: flex;
    gap: ${theme.spacing(0.5)};
    margin-top: ${theme.spacing(1)};
    opacity: 0.7;
  `,
  inputArea: css`
    padding: ${theme.spacing(2)};
    border-top: 1px solid ${theme.colors.border.weak};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  input: css`
    width: 100%;
  `,
  inputActions: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: ${theme.spacing(1)};
  `,
  keyboardHint: css`
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    kbd {
      background: ${theme.colors.background.secondary};
      padding: ${theme.spacing(0.25, 0.75)};
      border-radius: ${theme.shape.radius.default};
      font-family: monospace;
      border: 1px solid ${theme.colors.border.weak};
    }
  `,
  actionButtons: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(1)};
    flex-wrap: wrap;
  `,
});
