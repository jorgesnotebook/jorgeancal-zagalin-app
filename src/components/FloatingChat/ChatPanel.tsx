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
import { llm } from '@grafana/llm';
import { finalize } from 'rxjs';
import { useGrafanaContext } from '../../services/useGrafanaContext';
import { detectSkill } from '../../services/assistantSkills';
import { createExploreLink } from '../../services/actionExtractor';
import { useZagalinConfig } from '../../hooks/useZagalinConfig';
import { getFullSystemPrompt } from '../../types/zagalinConfig';
import type { AssistantAction } from '../../services/contextTypes';
import { ZagalinColors } from '../../theme/colors';
import { isLLMReady } from '../../services/llmHealthService';
import { VectorSearchService } from '../../services/vectorSearchService';
import { ZAGALIN_TOOLS, type ToolCall } from '../../services/zagalinTools';
import { optimizeContext } from '../../services/contextOptimizer';
import { useConversation } from '../../hooks/useConversation';
import type { ConversationMessage } from '../../services/conversationStorage';

// Internal Message type for UI (extends ConversationMessage)
interface Message extends ConversationMessage {
  actions?: AssistantAction[];
  toolCalls?: ToolCall[];
}

export function ChatPanel() {
  const s = useStyles2(getStyles);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [streamingContent, setStreamingContent] = useState('');
  const [llmReady, setLlmReady] = useState<boolean | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const vectorSearchRef = useRef(new VectorSearchService());

  // Get Grafana context
  const { context, hasContext, loading: contextLoading } = useGrafanaContext();

  // Get Zagalin configuration
  const { config: zagalinConfig } = useZagalinConfig();

  // Get conversation management hook
  const { messages: conversationMessages, addMessage, conversation, createNew, clearCurrent } = useConversation();

  // Handle new chat
  const handleNewChat = () => {
    clearCurrent();
    createNew(context);
  };

  // Convert conversation messages to UI messages
  const messages: Message[] = conversationMessages;

  // Initialize conversation on mount if none exists
  useEffect(() => {
    if (!conversation) {
      createNew(context);
    }
  }, [conversation, createNew, context]);

  // Check LLM health on mount
  useEffect(() => {
    const checkHealth = async () => {
      const ready = await isLLMReady();
      setLlmReady(ready);
      if (!ready) {
        setError('LLM service is not available. Please ensure Grafana LLM plugin is installed and configured.');
      }
    };
    checkHealth();
  }, []);

  // Debug: Log context changes
  useEffect(() => {
    if (hasContext) {
      console.log('Zagalin: Context detected', {
        dashboard: context.dashboard?.title,
        dashboardUid: context.dashboard?.uid,
        panel: context.panel?.title,
        panelId: context.panel?.id,
        timeRange: context.timeRange,
      });
    } else {
      console.log('Zagalin: No dashboard context detected. Navigate to a dashboard to enable context-aware chat.');
    }
  }, [hasContext, context]);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingContent]);

  const handleSend = async () => {
    if (!input.trim() || isStreaming) {return;}

    // Health check before sending
    if (llmReady === false) {
      setError('LLM service is not available. Please check your configuration.');
      return;
    }

    const userMessage: ConversationMessage = {
      id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    // Save user message to conversation
    addMessage(userMessage);
    setInput('');
    setIsStreaming(true);
    setError(null);
    setStreamingContent('');

    try {
      // Enhance query with vector search if available
      let enhancedQuery = userMessage.content;
      if (zagalinConfig.enabledSkills.searchContext) {
        const vectorContext = await vectorSearchRef.current.enhanceQueryWithContext(userMessage.content);
        if (vectorContext) {
          enhancedQuery = userMessage.content + vectorContext;
        }
      }

      // Detect if a skill should be used (only if skill is enabled)
      const skill = detectSkill(userMessage.content, context);
      const shouldUseSkill = skill && (
        skill.name === 'analyze_dashboard' || // Always enabled - core functionality
        (skill.name === 'explain_panel' && zagalinConfig.enabledSkills.explainPanel) ||
        (skill.name === 'generate_query' && zagalinConfig.enabledSkills.generateQuery) ||
        (skill.name === 'guided_troubleshooting' && zagalinConfig.enabledSkills.troubleshooting)
      );

      // Build messages array
      let allMessages;
      if (shouldUseSkill && skill) {
        allMessages = [
          { role: 'system' as const, content: skill.systemPrompt },
          { role: 'user' as const, content: skill.userPrompt },
        ];
      } else {
        allMessages = [...messages, userMessage].map(m => ({
          role: m.role,
          content: m.content,
        }));

        // Add full system prompt (base + custom instructions)
        allMessages.unshift({
          role: 'system',
          content: getFullSystemPrompt(zagalinConfig),
        });

        if (hasContext) {
          // Optimize context to fit token budget
          const optimized = optimizeContext(context, 1000);
          const contextPrompt = `\n\n## Current Grafana Context\n${optimized.essential}`;

          // Add context after system prompt
          allMessages.splice(1, 0, {
            role: 'system',
            content: contextPrompt,
          });
        }

        // If vector search found relevant context, add it
        if (enhancedQuery !== userMessage.content) {
          allMessages[allMessages.length - 1] = {
            role: 'user',
            content: enhancedQuery,
          };
        }
      }

      // Use grafana-llm-app plugin for streaming with tool support
      let accumulatedContent = '';
      const streamOptions: any = {
        model: 'gpt-4o-mini',
        messages: allMessages,
        temperature: zagalinConfig.temperature,
        max_tokens: zagalinConfig.maxTokens,
      };

      // Add tools if function calling is enabled
      if (zagalinConfig.enabledSkills.functionCalling) {
        streamOptions.tools = ZAGALIN_TOOLS;
        streamOptions.tool_choice = 'auto';
      }

      const stream = llm.streamChatCompletions(streamOptions).pipe(
        llm.accumulateContent(),
        finalize(() => {
          setIsStreaming(false);
        })
      );

      // Subscribe to the stream
      stream.subscribe({
        next: (content: string) => {
          accumulatedContent = content;
          setStreamingContent(content);
        },
        complete: async () => {
          const assistantMessage: ConversationMessage = {
            id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
            role: 'assistant',
            content: accumulatedContent,
            timestamp: new Date(),
          };

          // Save assistant message to conversation
          addMessage(assistantMessage);
          setStreamingContent('');
        },
        error: (err: any) => {
          console.error('Zagalin: Stream error', err);
          setError(err.message || 'Failed to send message');
          setIsStreaming(false);
        }
      });
    } catch (err) {
      console.error('Zagalin: Send error', err);
      setError(err instanceof Error ? err.message : 'An unexpected error occurred');
      setIsStreaming(false);
    }
  };

  const handleKeyPress = (e: KeyboardEvent<HTMLTextAreaElement>) => {
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
          window.location.assign(`/d/${action.data.dashboardUid}`);
        }
        break;
      case 'panel':
        if (action.data.dashboardUid && action.data.panelId) {
          window.location.assign(`/d/${action.data.dashboardUid}?viewPanel=${action.data.panelId}`);
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
    <div className={s.container}>
      <div className={s.contextBar}>
        <div className={s.contextInfo}>
          {llmReady === null ? (
            <Badge color="blue" text="Checking LLM..." icon="sync" />
          ) : llmReady ? (
            <>
              {hasContext ? (
                <Tooltip content={getContextTooltip()}>
                  <Badge color="green" text={context.dashboard ? `📊 ${context.dashboard.title}` : "Context Active"} icon="info-circle" />
                </Tooltip>
              ) : (
                <Tooltip content="Navigate to a dashboard to enable context-aware assistance">
                  <Badge color="orange" text="No Dashboard Context" icon="info-circle" />
                </Tooltip>
              )}
              {contextLoading && <Spinner inline size={14} />}
            </>
          ) : (
            <Badge color="red" text="LLM Unavailable" icon="exclamation-triangle" />
          )}
        </div>
        <div className={s.contextActions}>
          <Tooltip content="Start a new conversation">
            <IconButton
              name="plus"
              size="sm"
              variant="secondary"
              onClick={handleNewChat}
              aria-label="New chat"
            />
          </Tooltip>
          {conversation && messages.length > 0 && (
            <Tooltip content={`${messages.length} messages`}>
              <Badge color="blue" text={`${messages.length}`} icon="comment-alt" />
            </Tooltip>
          )}
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
            <div
              className={s.messageContent}
              dangerouslySetInnerHTML={{
                __html: message.content
                  .replace(/\n/g, '<br />')
                  .replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
                  .replace(/`([^`]+)`/g, '<code>$1</code>')
                  .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
                  .replace(/\*(.*?)\*/g, '<em>$1</em>')
              }}
            />
            {message.actions && message.actions.length > 0 && renderActions(message.actions)}
            {message.role === 'assistant' && (
              <div className={s.messageActions}>
                <IconButton
                  name="copy"
                  size="sm"
                  tooltip="Copy to clipboard"
                  onClick={() => copyToClipboard(message.content)}
                />
              </div>
            )}
          </div>
        ))}

        {streamingContent && (
          <div className={`${s.message} ${s.assistantMessage} ${s.streamingMessage}`}>
            <div className={s.messageContent}>
              <span
                dangerouslySetInnerHTML={{
                  __html: streamingContent
                    .replace(/\n/g, '<br />')
                    .replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
                    .replace(/`([^`]+)`/g, '<code>$1</code>')
                    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
                    .replace(/\*(.*?)\*/g, '<em>$1</em>')
                }}
              />
              <Spinner inline className={s.streamingSpinner} />
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <div className={s.inputArea}>
        <TextArea
          value={input}
          onChange={e => setInput(e.currentTarget.value)}
          onKeyDown={handleKeyPress}
          placeholder="Ask anything..."
          rows={2}
          className={s.input}
          disabled={isStreaming}
          style={{
            height: 'auto',
            minHeight: '44px',
          }}
          onInput={(e) => {
            const target = e.target as HTMLTextAreaElement;
            target.style.height = 'auto';
            target.style.height = Math.min(target.scrollHeight, 200) + 'px';
          }}
        />
        <div className={s.inputActions}>
          <Button
            icon="comment-alt"
            onClick={handleSend}
            variant="primary"
            disabled={!input.trim() || isStreaming}
            size="sm"
            className={s.sendButton}
          >
            {isStreaming ? 'Sending...' : 'Send'}
          </Button>
        </div>
      </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    background: ${theme.colors.background.primary};
  `,
  contextBar: css`
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(1, 2)};
    border-bottom: 1px solid ${theme.colors.border.weak};
    min-height: 40px;
  `,
  contextInfo: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    flex: 1;
  `,
  contextActions: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
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
    padding: ${theme.spacing(1.5)};
    border-radius: ${theme.shape.radius.default};
    max-width: 85%;
  `,
  userMessage: css`
    align-self: flex-end;
    background: ${theme.colors.background.secondary};
    color: ${theme.colors.text.primary};
  `,
  assistantMessage: css`
    align-self: flex-start;
    background: transparent;
    color: ${theme.colors.text.primary};
  `,
  streamingSpinner: css`
    margin-left: ${theme.spacing(1)};
  `,
  streamingMessage: css`
    animation: pulse 1.5s ease-in-out infinite;
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.8; }
    }
  `,
  messageContent: css`
    word-break: break-word;
    line-height: 1.5;
    font-size: ${theme.typography.body.fontSize};

    code {
      background: ${theme.colors.background.primary};
      padding: ${theme.spacing(0.25, 0.5)};
      border-radius: 4px;
      font-family: monospace;
      font-size: 0.9em;
    }

    pre {
      background: ${theme.colors.background.primary};
      padding: ${theme.spacing(1.5)};
      border-radius: 8px;
      overflow-x: auto;
      margin: ${theme.spacing(1, 0)};

      code {
        background: transparent;
        padding: 0;
      }
    }

    strong {
      font-weight: ${theme.typography.fontWeightMedium};
    }

    em {
      font-style: italic;
    }
  `,
  messageActions: css`
    display: flex;
    gap: ${theme.spacing(0.5)};
    margin-top: ${theme.spacing(0.5)};
    opacity: 0.7;
  `,
  actionButtons: css`
    display: flex;
    gap: ${theme.spacing(1)};
    margin-top: ${theme.spacing(1)};
    flex-wrap: wrap;
  `,
  inputArea: css`
    padding: ${theme.spacing(2)};
    border-top: 1px solid ${theme.colors.border.weak};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  input: css`
    flex: 1;
    border: none !important;
    background: transparent !important;
    box-shadow: none !important;
    resize: none !important;
    min-height: 44px;
    max-height: 200px;
    overflow-y: auto;
    font-size: ${theme.typography.body.fontSize};
    line-height: 1.5;
    padding: ${theme.spacing(0.5, 0)} !important;

    &:focus {
      outline: none !important;
      box-shadow: none !important;
      border: none !important;
    }

    &::placeholder {
      color: ${theme.colors.text.secondary};
      opacity: 0.6;
    }

    /* Hide scrollbar for cleaner look */
    scrollbar-width: thin;
    &::-webkit-scrollbar {
      width: 4px;
    }
    &::-webkit-scrollbar-thumb {
      background: ${theme.colors.border.weak};
      border-radius: 2px;
    }
  `,
  inputActions: css`
    display: flex;
    justify-content: flex-end;
  `,
  sendButton: css`
    background: ${ZagalinColors.orangeGradient} !important;
    border: none !important;
    color: white !important;

    &:hover:not(:disabled) {
      background: ${ZagalinColors.orangeGradientHover} !important;
      box-shadow: 0 4px 8px rgba(255, 152, 48, 0.3);
    }

    &:disabled {
      opacity: 0.5;
    }
  `,
});
