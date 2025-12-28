import React, { useState, useRef, useEffect, useMemo, KeyboardEvent } from 'react';
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
import DOMPurify from 'dompurify';
import { marked } from 'marked';
import { useGrafanaContext } from '../../services/useGrafanaContext';
import { createExploreLink } from '../../services/actionExtractor';
import { useZagalinConfig } from '../../hooks/useZagalinConfig';
import type { AssistantAction } from '../../services/contextTypes';
import { ZagalinColors } from '../../theme/colors';
import { isLLMReady } from '../../services/llmHealthService';
import { VectorSearchService } from '../../services/vectorSearchService';
import { type ToolCall } from '../../services/zagalinTools';
import { useConversation } from '../../hooks/useConversation';
import type { ConversationMessage } from '../../services/conversationStorage';
import { ConversationListSidebar } from './ConversationListSidebar';
import { useRunState } from '../../hooks/useRunState';
import { PlanVisualization } from './PlanVisualization';
import { ArtifactCard } from './ArtifactCard';
import { RunControls } from './RunControls';
import type { Artifact } from '../../services/runService';
import { streamAssistantChat } from '../../services/assistantService';
import { FrontendOrchestrator, type OrchestratorEvent } from '../../services/frontendOrchestrator';
import type { ExecutionPlan } from '../../services/frontendPrompts';
import { needsOrchestration } from '../../services/orchestrationDetector';
import { isDashboardQuestion, readDashboardPanels, buildDashboardSummaryPrompt } from '../../services/dashboardReader';

// Internal Message type for UI (extends ConversationMessage)
interface Message extends ConversationMessage {
  actions?: AssistantAction[];
  toolCalls?: ToolCall[];
  artifacts?: Artifact[];
}

// Helper function to safely render markdown content
function sanitizeMarkdown(content: string): string {
  try {
    const rawHtml = marked.parse(content) as string;
    return DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'code', 'pre', 'ul', 'ol', 'li', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'blockquote'],
      ALLOWED_ATTR: ['href', 'title', 'target', 'rel'],
    });
  } catch (error) {
    console.error('Markdown sanitization error:', error);
    return DOMPurify.sanitize(content.replace(/</g, '&lt;').replace(/>/g, '&gt;'));
  }
}

export function ChatPanel() {
  const s = useStyles2(getStyles);
  const [input, setInput] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [displayedContent, setDisplayedContent] = useState('');
  const [llmReady, setLlmReady] = useState<boolean | null>(null);
  const [showSidebar, setShowSidebar] = useState(true);
  const [optimisticMessages, setOptimisticMessages] = useState<Message[]>([]);
  const [llmBackendMode, setLlmBackendMode] = useState<string>('grafana-llm-app'); // 'grafana-llm-app' | 'direct' | 'disabled'

  // Frontend orchestrator state (for grafana-llm-app mode with orchestration)
  const [frontendPlan, setFrontendPlan] = useState<ExecutionPlan | null>(null);
  const [frontendCurrentStepIndex, setFrontendCurrentStepIndex] = useState(0);
  const [frontendArtifacts, setFrontendArtifacts] = useState<Artifact[]>([]);
  const [frontendStreamingText, setFrontendStreamingText] = useState('');
  const [isFrontendOrchestrating, setIsFrontendOrchestrating] = useState(false);

  // Simple streaming state (for grafana-llm-app mode without orchestration)
  const [isSimpleStreaming, setIsSimpleStreaming] = useState(false);
  const [simpleStreamingContent, setSimpleStreamingContent] = useState('');

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const vectorSearchRef = useRef(new VectorSearchService());
  const typewriterTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const frontendOrchestratorRef = useRef<FrontendOrchestrator | null>(null);
  const simpleStreamingContentRef = useRef<string>(''); // Accumulates simple streaming content

  // Get Grafana context
  const { context, hasContext, loading: contextLoading } = useGrafanaContext();

  // Get Zagalin configuration
  const { config: zagalinConfig } = useZagalinConfig();

  // Get conversation management hook
  const {
    messages: conversationMessages,
    addMessage,
    conversation,
    conversations,
    createNew,
    loadConversation,
    deleteConversation,
    deleteAll,
    updateTitle,
    togglePin,
    clearCurrent,
    currentId,
  } = useConversation();

  // Run state management with callbacks
  const handleRunComplete = (finalMessage: string, artifacts: Artifact[]) => {
    const assistantMessage: ConversationMessage = {
      id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
      role: 'assistant',
      content: finalMessage,
      timestamp: new Date(),
      artifacts,
    };
    addMessage(assistantMessage);
    setOptimisticMessages([]);
  };

  const handleRunError = (errorMessage: string) => {
    setError(errorMessage);
    setOptimisticMessages([]);
  };

  const runState = useRunState({
    conversationId: conversation?.id || 'temp',
    onComplete: handleRunComplete,
    onError: handleRunError,
  });

  // Handle new chat
  const handleNewChat = () => {
    clearCurrent();
    setOptimisticMessages([]);
    createNew(context);
  };

  // Convert conversation messages to UI messages, including optimistic messages
  const messages: Message[] = useMemo(() => [...conversationMessages, ...optimisticMessages], [conversationMessages, optimisticMessages]);

  // Fetch LLM backend mode on mount
  useEffect(() => {
    const fetchBackendMode = async () => {
      try {
        const response = await fetch('/api/plugins/jorgeancal-zagalin-app/resources/settings');
        if (response.ok) {
          const settings = await response.json();
          const mode = settings?.jsonData?.llmBackend || 'grafana-llm-app';
          setLlmBackendMode(mode);
          console.log('[ChatPanel] LLM backend mode:', mode);
        }
      } catch (error) {
        console.warn('[ChatPanel] Failed to fetch backend mode, defaulting to grafana-llm-app');
      }
    };
    fetchBackendMode();
  }, []);

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
  }, [messages, displayedContent]);

  // Typewriter effect: gradually display streaming content (from various sources)
  useEffect(() => {
    // Determine streaming content source based on mode and state
    let streamingContent = '';
    if (llmBackendMode === 'direct') {
      streamingContent = runState.streamingText;
    } else if (isFrontendOrchestrating) {
      streamingContent = frontendStreamingText;
    } else if (isSimpleStreaming) {
      streamingContent = simpleStreamingContent;
    }

    if (streamingContent.length === 0 && displayedContent.length > 0) {
      // Content was cleared, reset display (async to avoid setState-in-effect warning)
      setTimeout(() => setDisplayedContent(''), 0);
    } else if (displayedContent.length < streamingContent.length) {
      // Clear any existing timeout
      if (typewriterTimeoutRef.current) {
        clearTimeout(typewriterTimeoutRef.current);
      }

      // Add next character(s) with a small delay
      typewriterTimeoutRef.current = setTimeout(() => {
        // Display 2-4 characters at a time for a natural feel
        const charsToAdd = Math.min(3, streamingContent.length - displayedContent.length);
        setDisplayedContent(streamingContent.substring(0, displayedContent.length + charsToAdd));
      }, 20); // 20ms delay for smooth typing effect
    } else if (displayedContent.length > streamingContent.length) {
      // Handle case where streaming content was reset (async to avoid setState-in-effect warning)
      setTimeout(() => setDisplayedContent(streamingContent), 0);
    }

    // Cleanup on unmount
    return () => {
      if (typewriterTimeoutRef.current) {
        clearTimeout(typewriterTimeoutRef.current);
      }
    };
  }, [runState.streamingText, frontendStreamingText, simpleStreamingContent, displayedContent, llmBackendMode, isFrontendOrchestrating, isSimpleStreaming]);

  const handleSend = async () => {
    if (!input.trim() || runState.isRunning || isFrontendOrchestrating || isSimpleStreaming) {
      return;
    }

    // Health check before sending
    if (llmReady === false) {
      setError('LLM service is not available. Please check your configuration.');
      return;
    }

    console.log('[ChatPanel] Current conversation before send:', conversation?.id);
    console.log('[ChatPanel] Conversation message count:', conversationMessages.length);

    const userMessage: ConversationMessage = {
      id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    // Show message immediately (optimistic update)
    setOptimisticMessages([userMessage]);
    setInput('');
    setError(null);
    setDisplayedContent('');

    try {
      // Save user message to conversation (with context if this is the first message)
      // MUST await this to ensure conversation is created before assistant responds
      await addMessage(userMessage, context);

      console.log('[ChatPanel] Message saved, current conversation:', conversation?.id);

      // Clear optimistic messages once saved
      setOptimisticMessages([]);

      // Enhance query with vector search if available
      let enhancedQuery = userMessage.content;
      if (zagalinConfig.enabledSkills.searchContext) {
        const vectorContext = await vectorSearchRef.current.enhanceQueryWithContext(userMessage.content);
        if (vectorContext) {
          enhancedQuery = userMessage.content + vectorContext;
        }
      }

      // Route based on backend mode
      if (llmBackendMode === 'grafana-llm-app') {
        // Check if this is a dashboard question that needs enriched context
        if (isDashboardQuestion(userMessage.content, context)) {
          console.log('[ChatPanel] Dashboard question detected - using enriched context');

          // Extract dashboard panel data
          const panels = await readDashboardPanels(context);
          console.log(`[ChatPanel] Extracted ${panels.length} panels from dashboard`);

          // Build enriched prompt with actual panel queries
          const enrichedPrompt = buildDashboardSummaryPrompt(userMessage.content, context, panels);

          // Use simple streaming with enriched message
          console.log('[ChatPanel] Starting simple streaming with dashboard context');
          setIsSimpleStreaming(true);
          simpleStreamingContentRef.current = '';

          streamAssistantChat({
            message: userMessage.content,
            enrichedMessage: enrichedPrompt, // Pass enriched version with panel details
            history: messages.map(m => ({
              role: m.role,
              content: m.content,
            })),
            context: {
              dashboard: context.dashboard,
              panel: context.panel,
              timeRange: context.timeRange,
              templateVars: context.templateVariables,
            },
          }).subscribe({
            next: (chunk) => {
              if (chunk.chunk) {
                simpleStreamingContentRef.current += chunk.chunk;
                setSimpleStreamingContent(simpleStreamingContentRef.current);
              }
            },
            complete: async () => {
              console.log('[ChatPanel] Dashboard question streaming complete');
              const finalContent = simpleStreamingContentRef.current;

              // Save assistant message
              const assistantMessage: ConversationMessage = {
                id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
                role: 'assistant',
                content: finalContent,
                timestamp: new Date(),
              };

              await addMessage(assistantMessage);
              setIsSimpleStreaming(false);
              setSimpleStreamingContent('');
              simpleStreamingContentRef.current = '';
            },
            error: (err) => {
              console.error('[ChatPanel] Dashboard question streaming error:', err);
              setError(err instanceof Error ? err.message : 'Streaming failed');
              setIsSimpleStreaming(false);
              setSimpleStreamingContent('');
              simpleStreamingContentRef.current = '';
            },
          });

          return; // Don't continue to orchestration or other streaming
        }

        // Detect if this message needs orchestration
        const shouldOrchestrate = needsOrchestration(userMessage.content);

        if (shouldOrchestrate) {
          // Frontend orchestration mode - structured planning with grafana-llm-app
          console.log('[ChatPanel] Using frontend orchestration mode (complex query)');

          // Reset state
          setIsFrontendOrchestrating(true);
          setFrontendPlan(null);
          setFrontendCurrentStepIndex(0);
          setFrontendArtifacts([]);
          setFrontendStreamingText('');

          // Create orchestrator
          const orchestrator = new FrontendOrchestrator();
          frontendOrchestratorRef.current = orchestrator;

          // Start orchestration
          orchestrator.start(
            enhancedQuery || userMessage.content,
            messages.map(m => ({
              role: m.role,
              content: m.content,
            })),
            {
              dashboard: context.dashboard,
              panel: context.panel,
              timeRange: context.timeRange,
              templateVars: context.templateVariables,
            }
          ).subscribe({
            next: (event: OrchestratorEvent) => {
              console.log('[ChatPanel] Frontend orchestrator event:', event.type);

              switch (event.type) {
                case 'plan':
                  setFrontendPlan({
                    goal: event.data.goal,
                    steps: event.data.steps,
                    estimatedDuration: event.data.estimatedDuration || 'Estimating...',
                  });
                  break;

                case 'step_started':
                  setFrontendCurrentStepIndex(event.data.stepIndex);
                  setFrontendStreamingText(''); // Reset for new step
                  break;

                case 'artifact':
                  setFrontendArtifacts(prev => [...prev, event.data]);
                  break;

                case 'assistant_delta':
                  setFrontendStreamingText(prev => prev + event.data.delta);
                  break;

                case 'assistant_message':
                  // Save final message
                  const assistantMessage: ConversationMessage = {
                    id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
                    role: 'assistant',
                    content: event.data.text,
                    timestamp: new Date(),
                    artifacts: event.data.artifacts,
                  };
                  addMessage(assistantMessage);
                  break;

                case 'error':
                  setError(event.data.message);
                  break;
              }
            },
            complete: () => {
              console.log('[ChatPanel] Frontend orchestration complete');
              setIsFrontendOrchestrating(false);
              setOptimisticMessages([]);
            },
            error: (err) => {
              console.error('[ChatPanel] Frontend orchestration error:', err);
              setError(err instanceof Error ? err.message : 'Orchestration failed');
              setIsFrontendOrchestrating(false);
              setOptimisticMessages([]);
            },
          });
        } else {
          // Simple streaming mode - for greetings and simple questions
          console.log('[ChatPanel] Using simple streaming mode (greeting/simple query)');
          setIsSimpleStreaming(true);
          setSimpleStreamingContent('');
          simpleStreamingContentRef.current = '';

          streamAssistantChat({
            message: enhancedQuery || userMessage.content,
            history: messages.map(m => ({
              role: m.role,
              content: m.content,
            })),
            context: {
              dashboard: context.dashboard,
              panel: context.panel,
              timeRange: context.timeRange,
              templateVars: context.templateVariables,
            },
          }).subscribe({
            next: (chunk) => {
              if (chunk.chunk) {
                simpleStreamingContentRef.current += chunk.chunk;
                setSimpleStreamingContent(simpleStreamingContentRef.current);
              }
            },
            complete: async () => {
              console.log('[ChatPanel] Simple streaming complete');
              const finalContent = simpleStreamingContentRef.current;

              // Save assistant message
              const assistantMessage: ConversationMessage = {
                id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
                role: 'assistant',
                content: finalContent,
                timestamp: new Date(),
              };

              await addMessage(assistantMessage);
              setIsSimpleStreaming(false);
              setSimpleStreamingContent('');
              simpleStreamingContentRef.current = '';
            },
            error: (err) => {
              console.error('[ChatPanel] Simple streaming error:', err);
              setError(err instanceof Error ? err.message : 'Streaming failed');
              setIsSimpleStreaming(false);
              setSimpleStreamingContent('');
              simpleStreamingContentRef.current = '';
            },
          });
        }
      } else if (llmBackendMode === 'direct') {
        // Run orchestration mode - use backend with planning/steps/artifacts
        console.log('[ChatPanel] Using run orchestration mode (direct)');
        await runState.start(
          enhancedQuery || userMessage.content,
          messages.map(m => ({
            role: m.role,
            content: m.content,
          })),
          {
            dashboard: context.dashboard,
            panel: context.panel,
            timeRange: context.timeRange,
            templateVars: context.templateVariables,
          }
        );
      } else {
        // Disabled mode
        setError('LLM features are disabled in plugin configuration');
      }
    } catch (err) {
      console.error('Zagalin: Send error', err);
      setError(err instanceof Error ? err.message : 'An unexpected error occurred');
      setOptimisticMessages([]);
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
    <div className={s.outerContainer}>
      {showSidebar && (
        <ConversationListSidebar
          conversations={conversations}
          currentId={currentId}
          onSelectConversation={loadConversation}
          onRenameConversation={updateTitle}
          onDeleteConversation={deleteConversation}
          onDeleteAll={deleteAll}
          onTogglePin={togglePin}
          onCreateNew={handleNewChat}
        />
      )}
      <div className={s.container}>
        <div className={s.contextBar}>
        <div className={s.contextInfo}>
          <Tooltip content={showSidebar ? 'Hide conversation history' : 'Show conversation history'}>
            <IconButton
              name={showSidebar ? 'arrow-left' : 'bars'}
              size="sm"
              variant="secondary"
              onClick={() => setShowSidebar(!showSidebar)}
              aria-label="Toggle history"
            />
          </Tooltip>
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
                __html: sanitizeMarkdown(message.content)
              }}
            />
            {message.actions && message.actions.length > 0 && renderActions(message.actions)}
            {message.artifacts && message.artifacts.length > 0 && (
              <div className={s.artifactsSection}>
                {message.artifacts.map(artifact => (
                  <ArtifactCard key={artifact.id} artifact={artifact} />
                ))}
              </div>
            )}
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

        {/* Plan visualization - for both modes */}
        {llmBackendMode === 'direct' && runState.plan && runState.isRunning && (
          <PlanVisualization
            plan={runState.plan}
            currentStepIndex={runState.currentStepIndex}
          />
        )}

        {llmBackendMode === 'grafana-llm-app' && frontendPlan && isFrontendOrchestrating && (
          <PlanVisualization
            plan={frontendPlan}
            currentStepIndex={frontendCurrentStepIndex}
          />
        )}

        {/* Artifacts - for both modes */}
        {llmBackendMode === 'direct' && runState.artifacts.length > 0 && runState.isRunning && (
          <div className={s.artifactsSection}>
            <h4 className={s.artifactsHeading}>Evidence</h4>
            {runState.artifacts.map(artifact => (
              <ArtifactCard key={artifact.id} artifact={artifact} />
            ))}
          </div>
        )}

        {llmBackendMode === 'grafana-llm-app' && frontendArtifacts.length > 0 && isFrontendOrchestrating && (
          <div className={s.artifactsSection}>
            <h4 className={s.artifactsHeading}>Evidence</h4>
            {frontendArtifacts.map(artifact => (
              <ArtifactCard key={artifact.id} artifact={artifact} />
            ))}
          </div>
        )}

        {/* Thinking indicator for all modes */}
        {((llmBackendMode === 'direct' && runState.isRunning) || (llmBackendMode === 'grafana-llm-app' && (isFrontendOrchestrating || isSimpleStreaming))) && !displayedContent && (
          <div className={`${s.message} ${s.assistantMessage}`}>
            <div className={s.thinkingIndicator}>
              <span className={s.thinkingDot}></span>
              <span className={s.thinkingDot}></span>
              <span className={s.thinkingDot}></span>
            </div>
          </div>
        )}

        {/* Streaming content display for both modes */}
        {displayedContent && (
          <div className={`${s.message} ${s.assistantMessage} ${s.streamingMessage}`}>
            <div className={s.messageContent}>
              <span
                dangerouslySetInnerHTML={{
                  __html: sanitizeMarkdown(displayedContent)
                }}
              />
              {displayedContent.length < (llmBackendMode === 'direct' ? runState.streamingText : frontendStreamingText).length && (
                <span className={s.cursor}>▊</span>
              )}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      {/* Run controls - only for direct mode */}
      {llmBackendMode === 'direct' && runState.isRunning && (
        <RunControls
          status={runState.status}
          onPause={runState.pause}
          onResume={runState.resume}
          onCancel={runState.cancel}
        />
      )}

      <div className={s.inputArea}>
        <TextArea
          value={input}
          onChange={e => setInput(e.currentTarget.value)}
          onKeyDown={handleKeyPress}
          placeholder="Ask anything..."
          rows={2}
          className={s.input}
          disabled={runState.isRunning || isFrontendOrchestrating || isSimpleStreaming}
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
            disabled={!input.trim() || runState.isRunning || isFrontendOrchestrating || isSimpleStreaming}
            size="sm"
            className={s.sendButton}
          >
            {(runState.isRunning || isFrontendOrchestrating || isSimpleStreaming) ? 'Running...' : 'Send'}
          </Button>
        </div>
      </div>
    </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  outerContainer: css`
    display: flex;
    flex-direction: row;
    height: 100%;
    width: 100%;
  `,
  container: css`
    display: flex;
    flex-direction: column;
    flex: 1;
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
  thinkingIndicator: css`
    display: flex;
    gap: ${theme.spacing(0.75)};
    align-items: center;
    padding: ${theme.spacing(1.5, 2)};
  `,
  thinkingDot: css`
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: ${ZagalinColors.orange};
    animation: thinking 1.4s ease-in-out infinite;

    &:nth-child(1) {
      animation-delay: 0s;
    }
    &:nth-child(2) {
      animation-delay: 0.2s;
    }
    &:nth-child(3) {
      animation-delay: 0.4s;
    }

    @keyframes thinking {
      0%, 60%, 100% {
        opacity: 0.3;
        transform: scale(1) translateY(0);
      }
      30% {
        opacity: 1;
        transform: scale(1.3) translateY(-4px);
      }
    }
  `,
  cursor: css`
    display: inline-block;
    width: 2px;
    height: 1em;
    margin-left: 4px;
    background: ${ZagalinColors.orange};
    vertical-align: middle;
    animation: blink 0.9s ease-in-out infinite;

    @keyframes blink {
      0%, 49% { opacity: 1; }
      50%, 100% { opacity: 0; }
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
  artifactsSection: css`
    margin-top: ${theme.spacing(1.5)};
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  artifactsHeading: css`
    font-size: ${theme.typography.body.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
    color: ${theme.colors.text.primary};
    margin-bottom: ${theme.spacing(0.5)};
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
