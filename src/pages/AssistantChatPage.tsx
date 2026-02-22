import React, { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import { Button, TextArea, useStyles2, IconButton, Spinner, Alert } from '@grafana/ui';
import { useAsync } from 'react-use';
import { finalize, Subscription } from 'rxjs';
import DOMPurify from 'dompurify';
import { marked } from 'marked';
import { ZagalinColors } from '../theme/colors';
import { isLLMReady } from '../services/llmHealthService';
import { VectorSearchService } from '../services/vectorSearchService';
import { type ToolCall } from '../services/zagalinTools';
import { useZagalinConfig } from '../hooks/useZagalinConfig';
import { LLMClient } from '../services/llm/LLMClient';
import type { AssistantRequest, StreamChunk } from '../services/llm/types';

// Singleton instance for reuse
const llmClient = new LLMClient();
import { ContextService } from '../services/contextService';

interface Message {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: Date;
  toolCalls?: ToolCall[];
}

function sanitizeMarkdown(content: string): string {
  try {
    const rawHtml = marked.parse(content) as string;
    return DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: [
        'p',
        'br',
        'strong',
        'em',
        'code',
        'pre',
        'ul',
        'ol',
        'li',
        'a',
        'h1',
        'h2',
        'h3',
        'h4',
        'h5',
        'h6',
        'blockquote',
      ],
      ALLOWED_ATTR: ['href', 'title', 'target', 'rel'],
    });
  } catch (error) {
    console.error('Markdown sanitization error:', error);
    return DOMPurify.sanitize(content.replace(/</g, '&lt;').replace(/>/g, '&gt;'));
  }
}

function AssistantChatPage() {
  const s = useStyles2(getStyles);
  const { config: zagalinConfig } = useZagalinConfig();
  const vectorSearchRef = useRef(new VectorSearchService());

  const [messages, setMessages] = useState<Message[]>([
    {
      role: 'system',
      content:
        'You are Zagalin, a helpful assistant that helps engineers learn how to use Grafana. You explain dashboards, help write queries, and provide troubleshooting guidance. IMPORTANT: Never ask for screenshots or additional information - provide direct answers based on the context available.',
      timestamp: new Date(),
    },
  ]);
  const [input, setInput] = useState('');
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [streamingContent, setStreamingContent] = useState('');
  const [llmHealth, setLlmHealth] = useState<{ ok: boolean; error?: string; configured?: boolean } | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const streamSubscriptionRef = useRef<Subscription | null>(null);

  const { loading: checkingLlm } = useAsync(async () => {
    try {
      const ready = await isLLMReady();
      setLlmHealth({ ok: ready, error: ready ? undefined : 'LLM plugin not configured' });
      return { ok: ready };
    } catch (err: any) {
      setLlmHealth({ ok: false, error: 'Plugin not installed' });
      return { ok: false, error: err.message };
    }
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingContent]);

  // Cleanup subscription on unmount
  useEffect(() => {
    return () => {
      if (streamSubscriptionRef.current) {
        streamSubscriptionRef.current.unsubscribe();
      }
    };
  }, []);

  const handleSend = async () => {
    if (!input.trim() || isStreaming || !llmHealth?.ok) {
      return;
    }

    const userMessage: Message = {
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInput('');
    setIsStreaming(true);
    setError(null);
    setStreamingContent('');

    try {
      let enhancedQuery = userMessage.content;
      if (zagalinConfig.enabledSkills.searchContext) {
        const vectorContext = await vectorSearchRef.current.enhanceQueryWithContext(userMessage.content);
        if (vectorContext) {
          enhancedQuery = userMessage.content + vectorContext;
        }
      }

      // Extract fresh context at the moment of asking (includes current URL, template vars, time range)
      const currentContext = await ContextService.getContext();

      const assistantRequest: AssistantRequest = {
        message: enhancedQuery || userMessage.content,
        history: messages.map((m) => ({
          role: m.role,
          content: m.content,
        })),
        context: {
          dashboard: currentContext.dashboard,
          panel: currentContext.panel,
          timeRange: currentContext.timeRange,
          templateVars: currentContext.templateVariables,
        },
      };

      let accumulatedContent = '';
      const stream = llmClient.chat(assistantRequest).pipe(
        finalize(() => {
          setIsStreaming(false);
        })
      );

      const subscription = stream.subscribe({
        next: (chunk: StreamChunk) => {
          if (chunk.chunk) {
            accumulatedContent += chunk.chunk;
            setStreamingContent(accumulatedContent);
          }
          if (chunk.error) {
            console.error('Zagalin: Backend error', chunk.error);
            setError(chunk.error);
            setIsStreaming(false);
            setStreamingContent('');
          }
        },
        complete: () => {
          const assistantMessage: Message = {
            role: 'assistant',
            content: accumulatedContent,
            timestamp: new Date(),
          };
          setMessages((prev) => [...prev, assistantMessage]);
          setStreamingContent('');
          setIsStreaming(false);
          streamSubscriptionRef.current = null;
        },
        error: (err: Error) => {
          console.error('Zagalin: Stream error', err);
          setError(err.message || 'Failed to get response from LLM');
          setIsStreaming(false);
          setStreamingContent('');
          streamSubscriptionRef.current = null;
        },
      });

      // Store subscription for cancellation
      streamSubscriptionRef.current = subscription;
    } catch (err: any) {
      console.error('Zagalin: Send error', err);
      setError(err.message || 'An unexpected error occurred');
      setIsStreaming(false);
      setStreamingContent('');
    }
  };

  const handleStop = () => {
    // Unsubscribe from the stream (this will call the cleanup function in the Observable)
    if (streamSubscriptionRef.current) {
      streamSubscriptionRef.current.unsubscribe();
      streamSubscriptionRef.current = null;
    }

    // If there's partial streaming content, save it as a message
    if (streamingContent.trim()) {
      const assistantMessage: Message = {
        role: 'assistant',
        content: streamingContent + '\n\n*(Response stopped by user)*',
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, assistantMessage]);
    }

    // Reset streaming state
    setIsStreaming(false);
    setStreamingContent('');
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

  const clearChat = () => {
    setMessages([messages[0]]);
    setStreamingContent('');
    setError(null);
  };

  if (checkingLlm) {
    return (
      <div className={s.container}>
        <Spinner /> Checking LLM availability...
      </div>
    );
  }

  if (!llmHealth?.ok) {
    const isNotInstalled =
      llmHealth?.error?.includes('not installed') || llmHealth?.error?.includes('not found') || !llmHealth;

    return (
      <div className={s.container}>
        {isNotInstalled ? (
          <Alert title="LLM Plugin Not Installed" severity="error">
            <p>The Grafana LLM App plugin is not installed.</p>
            <p>To use Zagalin, you need to:</p>
            <ol>
              <li>
                Install the <code>grafana-llm-app</code> plugin
              </li>
              <li>Configure it with an OpenAI API key</li>
            </ol>
            <p>
              The plugin should auto-install when you restart the server with{' '}
              <code>GF_INSTALL_PLUGINS=grafana-llm-app</code>
            </p>
          </Alert>
        ) : (
          <Alert title="LLM Plugin Not Configured" severity="warning">
            <p>The Grafana LLM App plugin is installed but not configured.</p>
            <p>To use Zagalin:</p>
            <ol>
              <li>
                Go to{' '}
                <a href="/plugins/grafana-llm-app" target="_blank">
                  LLM App Settings
                </a>
              </li>
              <li>{"Enable the plugin if it's disabled"}</li>
              <li>Add your OpenAI API key in the configuration</li>
              <li>Save and refresh this page</li>
            </ol>
            {llmHealth?.error && (
              <p>
                <strong>Error details:</strong> {llmHealth.error}
              </p>
            )}
          </Alert>
        )}
      </div>
    );
  }

  return (
    <div className={s.container}>
      {/* Header - only show when conversation started */}
      {messages.length > 1 && (
        <div className={s.header}>
          <div className={s.headerContent}>
            <h2 className={s.title}>Zagalin</h2>
            <Button icon="trash-alt" variant="secondary" size="sm" onClick={clearChat}>
              Clear Chat
            </Button>
          </div>
        </div>
      )}

      {error && (
        <div className={s.errorContainer}>
          <Alert title="Error" severity="error" onRemove={() => setError(null)}>
            {error}
          </Alert>
        </div>
      )}

      {/* Messages area - scrollable middle section */}
      <div className={s.messagesWrapper}>
        {messages.length === 1 ? (
          <div className={s.welcomeScreen}>
            <img src="public/plugins/jorgeancal-zagalin-app/img/logo.png" alt="Zagalin" className={s.logo} />
          </div>
        ) : (
          <div className={s.messagesContainer}>
            {/* Messages when chat started */}
            {messages.slice(1).map((message, idx) => (
              <div
                key={idx}
                className={`${s.messageRow} ${message.role === 'user' ? s.userMessageRow : s.assistantMessageRow}`}
              >
                <div
                  className={`${s.message} ${message.role === 'user' ? s.userMessage : s.assistantMessage}`}
                  onMouseEnter={(e) => {
                    const actions = e.currentTarget.querySelector('[data-actions]');
                    if (actions) {
                      (actions as HTMLElement).style.opacity = '0.6';
                    }
                  }}
                  onMouseLeave={(e) => {
                    const actions = e.currentTarget.querySelector('[data-actions]');
                    if (actions) {
                      (actions as HTMLElement).style.opacity = '0';
                    }
                  }}
                >
                  <div
                    className={s.messageContent}
                    dangerouslySetInnerHTML={{
                      __html: sanitizeMarkdown(message.content),
                    }}
                  />
                  {message.role === 'assistant' && (
                    <div className={s.messageActions} data-actions>
                      <IconButton
                        name="copy"
                        size="sm"
                        tooltip="Copy to clipboard"
                        onClick={() => copyToClipboard(message.content)}
                      />
                    </div>
                  )}
                </div>
              </div>
            ))}

            {streamingContent && (
              <div className={`${s.messageRow} ${s.assistantMessageRow}`}>
                <div className={`${s.message} ${s.assistantMessage} ${s.streamingMessage}`}>
                  <div className={s.messageContent}>
                    <span
                      dangerouslySetInnerHTML={{
                        __html: sanitizeMarkdown(streamingContent),
                      }}
                    />
                    <Spinner inline className={s.streamingSpinner} />
                  </div>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Input area - fixed at bottom like ChatGPT */}
      <div className={s.inputArea}>
        <div className={s.inputContainer}>
          <div className={s.inputWrapper}>
            <TextArea
              value={input}
              onChange={(e) => setInput(e.currentTarget.value)}
              onKeyDown={handleKeyPress}
              placeholder="Ask anything..."
              rows={2}
              className={s.input}
              disabled={isStreaming}
              style={{
                height: 'auto',
                minHeight: '48px',
              }}
              onInput={(e) => {
                const target = e.target as HTMLTextAreaElement;
                target.style.height = 'auto';
                target.style.height = Math.min(target.scrollHeight, 400) + 'px';
              }}
            />
            <div className={s.sendButtonWrapper}>
              {isStreaming ? (
                <IconButton
                  name="times"
                  size="lg"
                  onClick={handleStop}
                  className={s.stopIconButton}
                  tooltip="Stop generation"
                  variant="destructive"
                />
              ) : (
                <IconButton
                  name="comment-alt"
                  size="lg"
                  onClick={handleSend}
                  disabled={!input.trim()}
                  className={s.sendIconButton}
                  tooltip="Send"
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default AssistantChatPage;

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    width: 100%;
    max-height: calc(100vh - 60px);
    overflow: hidden;
    background: ${theme.colors.background.primary};
  `,
  header: css`
    border-bottom: 2px solid;
    border-image: ${ZagalinColors.orangeGradient} 1;
    background: linear-gradient(135deg, rgba(242, 204, 12, 0.05) 0%, rgba(255, 152, 48, 0.05) 100%);
    flex-shrink: 0;
  `,
  headerContent: css`
    max-width: 900px;
    margin: 0 auto;
    padding: ${theme.spacing(2, 3)};
    display: flex;
    justify-content: space-between;
    align-items: center;
  `,
  title: css`
    margin: 0;
    font-size: ${theme.typography.h3.fontSize};
    font-weight: ${theme.typography.h3.fontWeight};
    background: ${ZagalinColors.orangeGradient};
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  `,
  errorContainer: css`
    max-width: 900px;
    margin: ${theme.spacing(2)} auto;
    padding: 0 ${theme.spacing(3)};
  `,
  messagesWrapper: css`
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  `,
  messagesContainer: css`
    width: 100%;
    max-width: 900px;
    margin: 0 auto;
    padding: ${theme.spacing(3)};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(3)};
  `,
  welcomeScreen: css`
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    padding: ${theme.spacing(4)};
    padding-bottom: ${theme.spacing(20)};
  `,
  logo: css`
    width: 220px;
    height: auto;
  `,
  messageRow: css`
    display: flex;
    width: 100%;
  `,
  userMessageRow: css`
    justify-content: flex-end;
  `,
  assistantMessageRow: css`
    justify-content: flex-start;
  `,
  message: css`
    padding: ${theme.spacing(2, 3)};
    border-radius: ${theme.shape.radius.default};
    max-width: 70%;
    position: relative;
  `,
  userMessage: css`
    background: ${theme.colors.background.secondary};
    color: ${theme.colors.text.primary};
  `,
  assistantMessage: css`
    background: transparent;
    color: ${theme.colors.text.primary};
  `,
  streamingMessage: css`
    animation: pulse 1.5s ease-in-out infinite;
    @keyframes pulse {
      0%,
      100% {
        opacity: 1;
      }
      50% {
        opacity: 0.8;
      }
    }
  `,
  streamingSpinner: css`
    margin-left: ${theme.spacing(1)};
  `,
  messageContent: css`
    word-break: break-word;
    line-height: 1.6;
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
    margin-top: ${theme.spacing(1)};
    opacity: 0;
    transition: opacity 0.2s;
  `,
  inputArea: css`
    background: ${theme.colors.background.primary};
    padding: ${theme.spacing(3)};
    flex-shrink: 0;
  `,
  inputContainer: css`
    max-width: 48rem;
    margin: 0 auto;
  `,
  inputWrapper: css`
    display: flex;
    align-items: flex-end;
    gap: ${theme.spacing(1.5)};
    padding: ${theme.spacing(1.5, 2)};
    background: ${theme.colors.background.secondary};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: 30px;
    transition: all 0.2s;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    min-height: 56px;

    &:focus-within {
      border-color: ${ZagalinColors.orange};
      box-shadow: 0 4px 16px rgba(255, 152, 48, 0.2);
    }
  `,
  input: css`
    flex: 1;
    border: none !important;
    background: transparent !important;
    box-shadow: none !important;
    resize: none !important;
    min-height: 48px;
    max-height: 400px;
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
      width: 6px;
    }
    &::-webkit-scrollbar-thumb {
      background: ${theme.colors.border.weak};
      border-radius: 3px;
    }
  `,
  sendButtonWrapper: css`
    display: flex;
    align-items: center;
  `,
  sendIconButton: css`
    background: ${ZagalinColors.orangeGradient} !important;
    border: none !important;
    color: white !important;
    border-radius: 50% !important;
    width: 40px !important;
    height: 40px !important;
    min-width: 40px !important;
    min-height: 40px !important;
    display: flex !important;
    align-items: center !important;
    justify-content: center !important;
    flex-shrink: 0 !important;
    transition: all 0.2s !important;
    margin-bottom: ${theme.spacing(0.5)};

    &:hover:not(:disabled) {
      background: ${ZagalinColors.orangeGradientHover} !important;
      transform: scale(1.08);
      box-shadow: 0 2px 8px rgba(255, 152, 48, 0.4);
    }

    &:disabled {
      opacity: 0.3;
      cursor: not-allowed;
    }

    svg {
      width: 20px !important;
      height: 20px !important;
    }
  `,
  stopIconButton: css`
    background: ${theme.colors.error.main} !important;
    border: none !important;
    color: white !important;
    border-radius: 50% !important;
    width: 40px !important;
    height: 40px !important;
    min-width: 40px !important;
    min-height: 40px !important;
    display: flex !important;
    align-items: center !important;
    justify-content: center !important;
    flex-shrink: 0 !important;
    transition: all 0.2s !important;
    margin-bottom: ${theme.spacing(0.5)};

    &:hover {
      background: ${theme.colors.error.shade} !important;
      transform: scale(1.08);
      box-shadow: 0 2px 8px rgba(204, 54, 54, 0.4);
    }

    svg {
      width: 16px !important;
      height: 16px !important;
    }
  `,
});
