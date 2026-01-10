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
  RadioButtonGroup,
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
import { PlanVisualization } from './PlanVisualization';
import { ArtifactCard } from './ArtifactCard';
import { ContextBadges } from './ContextBadges';
import { ReasoningDisplay } from './ReasoningDisplay';
import { parseReasoningResponse } from '../../services/reasoningParser';
import type { Artifact } from '../../services/runService';
import { streamAssistantChatRouted as streamAssistantChat } from '../../services/assistantServiceRouter';
import { FrontendOrchestrator, type OrchestratorEvent } from '../../services/frontendOrchestrator';
import type { ExecutionPlan } from '../../services/frontendPrompts';
import { needsOrchestration } from '../../services/orchestrationDetector';
import { isDashboardQuestion, readDashboardPanels, buildDashboardSummaryPrompt } from '../../services/dashboardReader';

interface Message extends ConversationMessage {
  actions?: AssistantAction[];
  toolCalls?: ToolCall[];
  artifacts?: Artifact[];
}

function wrapEvidenceSections(content: string): string {
  const evidenceCheckPattern = /(Evidence Check:|0\.\s*\*\*Evidence Check\*\*[^\n]*\n)((?:[\s\S]*?)(?=\n\n(?:\d+\.|\*\*|#{1,3}\s|\z)|$))/gi;

  const availableContextPattern = /(---\s*AVAILABLE CONTEXT\s*---)((?:[\s\S]*?)(?=---\s*UNKNOWN CONTEXT|$))/gi;

  const unknownContextPattern = /(---\s*UNKNOWN CONTEXT\s*---)((?:[\s\S]*?)(?=\n\n(?:\d+\.|\*\*|#{1,3}\s|\z)|$))/gi;

  const investigationMemoryPattern = /(Investigation Memory:|2\.\s*\*\*Investigation Memory\*\*[^\n]*\n)((?:[\s\S]*?)(?=\n\n(?:\d+\.|\*\*|#{1,3}\s|\z)|$))/gi;

  const metadataPattern = /(\*\*Metadata\*\*:?\n)((?:[\s\S]*?)(?=\n\n(?:\d+\.|\*\*|#{1,3}\s|\z)|$))/gi;

  const technicalDetailsPattern = /(\*\*Technical Details\*\*:?\n|\*\*How the query works\*\*:?\n)((?:[\s\S]*?)(?=\n\n(?:\d+\.|\*\*|#{1,3}\s|\z)|$))/gi;

  let processed = content;

  processed = processed.replace(evidenceCheckPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">📋 ${header.trim()}</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  processed = processed.replace(availableContextPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">✅ Available Context</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  processed = processed.replace(unknownContextPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">⚠️ Unknown Context</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  processed = processed.replace(investigationMemoryPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">🧠 ${header.trim()}</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  processed = processed.replace(metadataPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">ℹ️ ${header.trim()}</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  processed = processed.replace(technicalDetailsPattern, (match, header, body) => {
    return `<details class="evidence-section">
<summary class="evidence-header">🔧 ${header.trim()}</summary>
<div class="evidence-body">

${body.trim()}

</div>
</details>

`;
  });

  return processed;
}

function sanitizeMarkdown(content: string): string {
  try {
    const contentWithCollapsible = wrapEvidenceSections(content);

    const rawHtml = marked.parse(contentWithCollapsible) as string;

    return DOMPurify.sanitize(rawHtml, {
      ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'code', 'pre', 'ul', 'ol', 'li', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'blockquote', 'details', 'summary', 'div'],
      ALLOWED_ATTR: ['href', 'title', 'target', 'rel', 'class'],
    });
  } catch (error) {
    console.error('Markdown sanitization error:', error);
    return DOMPurify.sanitize(content.replace(/</g, '&lt;').replace(/>/g, '&gt;'));
  }
}

export type ChatMode = 'standard' | 'design';

export function ChatPanel() {
  const s = useStyles2(getStyles);
  const [input, setInput] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [displayedContent, setDisplayedContent] = useState('');
  const [llmReady, setLlmReady] = useState<boolean | null>(null);
  const [showSidebar, setShowSidebar] = useState(true);
  const [optimisticMessages, setOptimisticMessages] = useState<Message[]>([]);
  const [mode, setMode] = useState<ChatMode>('standard');

  const [frontendPlan, setFrontendPlan] = useState<ExecutionPlan | null>(null);
  const [frontendCurrentStepIndex, setFrontendCurrentStepIndex] = useState(0);
  const [frontendArtifacts, setFrontendArtifacts] = useState<Artifact[]>([]);
  const [frontendStreamingText, setFrontendStreamingText] = useState('');
  const [isFrontendOrchestrating, setIsFrontendOrchestrating] = useState(false);

  const [isSimpleStreaming, setIsSimpleStreaming] = useState(false);
  const [simpleStreamingContent, setSimpleStreamingContent] = useState('');

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const vectorSearchRef = useRef(new VectorSearchService());
  const frontendOrchestratorRef = useRef<FrontendOrchestrator | null>(null);
  const simpleStreamingContentRef = useRef<string>('');
  const animationFrameRef = useRef<number | null>(null);
  const displayedLengthRef = useRef<number>(0);

  const { context, hasContext, loading: contextLoading } = useGrafanaContext();

  const { config: zagalinConfig } = useZagalinConfig();

  const {
    messages: conversationMessages,
    addMessage,
    addContext,
    removeContext,
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


  const handleNewChat = () => {
    clearCurrent();
    setOptimisticMessages([]);
    createNew(context);
  };

  const messages: Message[] = useMemo(() => [...conversationMessages, ...optimisticMessages], [conversationMessages, optimisticMessages]);

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

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, displayedContent]);

  useEffect(() => {
    let streamingContent = '';
    if (isFrontendOrchestrating) {
      streamingContent = frontendStreamingText;
    } else if (isSimpleStreaming) {
      streamingContent = simpleStreamingContent;
    }

    if (streamingContent.length === 0) {
      displayedLengthRef.current = 0;
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
      animationFrameRef.current = requestAnimationFrame(() => {
        setDisplayedContent('');
        animationFrameRef.current = null;
      });
      return;
    }

    let lastTime = performance.now();
    const charsPerSecond = 100;
    const msPerChar = 1000 / charsPerSecond;

    const animate = (currentTime: number) => {
      const elapsed = currentTime - lastTime;

      if (displayedLengthRef.current < streamingContent.length) {
        const charsToAdd = Math.max(1, Math.floor(elapsed / msPerChar));
        const newLength = Math.min(
          displayedLengthRef.current + charsToAdd,
          streamingContent.length
        );
        displayedLengthRef.current = newLength;
        setDisplayedContent(streamingContent.substring(0, newLength));
        lastTime = currentTime;
        animationFrameRef.current = requestAnimationFrame(animate);
      } else if (displayedLengthRef.current > streamingContent.length) {
        displayedLengthRef.current = streamingContent.length;
        setDisplayedContent(streamingContent);
      }
    };

    if (displayedLengthRef.current < streamingContent.length) {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
      animationFrameRef.current = requestAnimationFrame(animate);
    }

    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
    };
  }, [frontendStreamingText, simpleStreamingContent, isFrontendOrchestrating, isSimpleStreaming]);

  const handleSend = async () => {
    if (!input.trim() || isFrontendOrchestrating || isSimpleStreaming) {
      return;
    }

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

    setOptimisticMessages([userMessage]);
    setInput('');
    setError(null);
    setDisplayedContent('');

    try {
      await addMessage(userMessage, context);

      console.log('[ChatPanel] Message saved, current conversation:', conversation?.id);

      setOptimisticMessages([]);

      let enhancedQuery = userMessage.content;
      if (zagalinConfig.enabledSkills.searchContext) {
        const vectorContext = await vectorSearchRef.current.enhanceQueryWithContext(userMessage.content);
        if (vectorContext) {
          enhancedQuery = userMessage.content + vectorContext;
        }
      }

        if (isDashboardQuestion(userMessage.content, context)) {
          console.log('[ChatPanel] Dashboard question detected - using enriched context');

          const panels = await readDashboardPanels(context);
          console.log(`[ChatPanel] Extracted ${panels.length} panels from dashboard`);

          const enrichedPrompt = buildDashboardSummaryPrompt(userMessage.content, context, panels);

          console.log('[ChatPanel] Starting simple streaming with dashboard context');
          setIsSimpleStreaming(true);
          simpleStreamingContentRef.current = '';

          streamAssistantChat({
            message: userMessage.content,
            enrichedMessage: enrichedPrompt,
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
            attachedContexts: conversation?.contexts,
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

              const explainableResponse = parseReasoningResponse(finalContent);

              const assistantMessage: ConversationMessage = explainableResponse ? {
                id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
                role: 'assistant',
                content: explainableResponse.answer,
                timestamp: new Date(),
                reasoning: explainableResponse.reasoning,
                sources: explainableResponse.sources,
                confidence: explainableResponse.confidence,
                caveats: explainableResponse.caveats,
              } : {
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

          return;
        }

        const shouldOrchestrate = needsOrchestration(userMessage.content);

        if (shouldOrchestrate) {
          console.log('[ChatPanel] Using frontend orchestration mode (complex query)');

          setIsFrontendOrchestrating(true);
          setFrontendPlan(null);
          setFrontendCurrentStepIndex(0);
          setFrontendArtifacts([]);
          setFrontendStreamingText('');

          const orchestrator = new FrontendOrchestrator();
          frontendOrchestratorRef.current = orchestrator;

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
                  setFrontendStreamingText('');
                  break;

                case 'artifact':
                  setFrontendArtifacts(prev => [...prev, event.data]);
                  break;

                case 'assistant_delta':
                  setFrontendStreamingText(prev => prev + event.data.delta);
                  break;

                case 'assistant_message':
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
            attachedContexts: conversation?.contexts,
            mode,
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

              const explainableResponse = parseReasoningResponse(finalContent);

              const assistantMessage: ConversationMessage = explainableResponse ? {
                id: `${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
                role: 'assistant',
                content: explainableResponse.answer,
                timestamp: new Date(),
                reasoning: explainableResponse.reasoning,
                sources: explainableResponse.sources,
                confidence: explainableResponse.confidence,
                caveats: explainableResponse.caveats,
              } : {
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
      </div>

      {error && (
        <Alert title="Error" severity="error" onRemove={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Context Management Section */}
      {conversation && (
        <div className={s.contextSection}>
          {conversation.contexts && conversation.contexts.length > 0 && (
            <ContextBadges
              contexts={conversation.contexts}
              onRemove={removeContext}
            />
          )}
          {hasContext && (
            <div className={s.addContextButton}>
              <Button
                size="sm"
                variant="secondary"
                icon="plus"
                onClick={() => addContext(context)}
                tooltip="Attach current dashboard to this conversation"
              >
                Add Current Dashboard
              </Button>
            </div>
          )}
        </div>
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
              <ReasoningDisplay
                reasoning={message.reasoning}
                sources={message.sources}
                confidence={message.confidence}
                caveats={message.caveats}
                collapsed={true}
              />
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

        {/* Plan visualization */}
        {frontendPlan && isFrontendOrchestrating && (
          <PlanVisualization
            plan={frontendPlan}
            currentStepIndex={frontendCurrentStepIndex}
          />
        )}

        {/* Artifacts */}
        {frontendArtifacts.length > 0 && isFrontendOrchestrating && (
          <div className={s.artifactsSection}>
            <h4 className={s.artifactsHeading}>Evidence</h4>
            {frontendArtifacts.map(artifact => (
              <ArtifactCard key={artifact.id} artifact={artifact} />
            ))}
          </div>
        )}

        {/* Thinking indicator */}
        {(isFrontendOrchestrating || isSimpleStreaming) && !displayedContent && (
          <div className={`${s.message} ${s.assistantMessage}`}>
            <div className={s.thinkingIndicator}>
              <span className={s.thinkingDot}></span>
              <span className={s.thinkingDot}></span>
              <span className={s.thinkingDot}></span>
            </div>
          </div>
        )}

        {/* Streaming content display */}
        {displayedContent && (
          <div className={`${s.message} ${s.assistantMessage} ${s.streamingMessage}`}>
            <div className={s.messageContent}>
              <span
                dangerouslySetInnerHTML={{
                  __html: sanitizeMarkdown(displayedContent)
                }}
              />
              {(isSimpleStreaming || isFrontendOrchestrating) && (
                <span className={s.cursor}>▊</span>
              )}
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <div className={s.inputContainer}>
        <div className={s.modeSelector}>
          <RadioButtonGroup
            options={[
              { label: '⚡ Standard', value: 'standard', description: 'Fast responses' },
              { label: '🎨 Design', value: 'design', description: 'Dashboard design with examples and suggestions' },
            ]}
            value={mode}
            onChange={(value) => setMode(value as ChatMode)}
            size="sm"
          />
        </div>
        <div className={s.inputArea}>
          <TextArea
            value={input}
            onChange={e => setInput(e.currentTarget.value)}
            onKeyDown={handleKeyPress}
            placeholder={mode === 'design' ? 'Design a dashboard or suggest improvements...' : 'Ask anything...'}
            rows={2}
            className={s.input}
            disabled={isFrontendOrchestrating || isSimpleStreaming}
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
              disabled={!input.trim() || isFrontendOrchestrating || isSimpleStreaming}
              size="sm"
              className={s.sendButton}
            >
              {(isFrontendOrchestrating || isSimpleStreaming) ? 'Running...' : 'Send'}
            </Button>
          </div>
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
    padding: 0px 8px;
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
  contextSection: css`
    padding: ${theme.spacing(2)};
    padding-top: 0;
  `,
  addContextButton: css`
    display: flex;
    justify-content: flex-start;
    margin-top: ${theme.spacing(1)};
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
    max-width: 95%;
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

    /* Evidence section styles */
    details.evidence-section {
      margin: ${theme.spacing(1, 0)};
      border: 1px solid ${theme.colors.border.weak};
      border-radius: ${theme.shape.radius.default};
      background: ${theme.colors.background.secondary};
      overflow: hidden;
    }

    summary.evidence-header {
      padding: ${theme.spacing(1, 1.5)};
      cursor: pointer;
      user-select: none;
      font-weight: ${theme.typography.fontWeightMedium};
      color: ${theme.colors.text.secondary};
      background: ${theme.colors.background.secondary};
      transition: background 0.2s ease;
      display: flex;
      align-items: center;
      gap: ${theme.spacing(0.5)};

      &:hover {
        background: ${theme.colors.emphasize(theme.colors.background.secondary, 0.03)};
      }

      &::marker {
        content: '▶ ';
        font-size: 0.8em;
      }
    }

    details.evidence-section[open] summary.evidence-header::marker {
      content: '▼ ';
    }

    div.evidence-body {
      padding: ${theme.spacing(1.5)};
      border-top: 1px solid ${theme.colors.border.weak};
      background: ${theme.colors.background.primary};
      font-size: ${theme.typography.bodySmall.fontSize};
      color: ${theme.colors.text.secondary};
    }

    /* Headers */
    h1, h2, h3, h4, h5, h6 {
      margin: ${theme.spacing(2, 0, 1, 0)};
      line-height: 1.3;
      font-weight: ${theme.typography.fontWeightMedium};
      color: ${theme.colors.text.primary};
    }

    h1 {
      font-size: ${theme.typography.h3.fontSize};
      border-bottom: 2px solid ${theme.colors.border.medium};
      padding-bottom: ${theme.spacing(0.5)};
    }

    h2 {
      font-size: ${theme.typography.h4.fontSize};
      border-bottom: 1px solid ${theme.colors.border.weak};
      padding-bottom: ${theme.spacing(0.5)};
    }

    h3 {
      font-size: ${theme.typography.h5.fontSize};
      color: ${theme.colors.text.primary};
    }

    h4, h5, h6 {
      font-size: ${theme.typography.body.fontSize};
      font-weight: ${theme.typography.fontWeightMedium};
    }

    /* Paragraphs and spacing */
    p {
      margin: ${theme.spacing(1, 0)};
      line-height: 1.6;
    }

    /* Lists */
    ul, ol {
      margin: ${theme.spacing(1, 0)};
      padding-left: ${theme.spacing(3)};
      line-height: 1.6;
    }

    ul {
      list-style-type: disc;
    }

    ol {
      list-style-type: decimal;
    }

    li {
      margin: ${theme.spacing(0.5, 0)};
      padding-left: ${theme.spacing(0.5)};
    }

    li > ul, li > ol {
      margin-top: ${theme.spacing(0.5)};
    }

    /* Task lists */
    li input[type="checkbox"] {
      margin-right: ${theme.spacing(1)};
    }

    /* Code */
    code {
      background: ${theme.colors.background.primary};
      padding: ${theme.spacing(0.25, 0.5)};
      border-radius: 4px;
      font-family: ${theme.typography.fontFamilyMonospace};
      font-size: 0.9em;
      color: ${theme.colors.text.primary};
      border: 1px solid ${theme.colors.border.weak};
    }

    pre {
      background: ${theme.colors.background.primary};
      padding: ${theme.spacing(1.5)};
      border-radius: 8px;
      overflow-x: auto;
      margin: ${theme.spacing(1.5, 0)};
      border: 1px solid ${theme.colors.border.weak};

      code {
        background: transparent;
        padding: 0;
        border: none;
        font-size: 0.85em;
        line-height: 1.5;
      }
    }

    /* Blockquotes */
    blockquote {
      border-left: 3px solid ${ZagalinColors.orange};
      margin: ${theme.spacing(1.5, 0)};
      padding: ${theme.spacing(1, 0, 1, 2)};
      background: ${theme.colors.background.secondary};
      border-radius: 0 4px 4px 0;
      color: ${theme.colors.text.secondary};
      font-style: italic;

      p {
        margin: ${theme.spacing(0.5, 0)};
      }
    }

    /* Links */
    a {
      color: ${theme.colors.text.link};
      text-decoration: none;
      border-bottom: 1px solid transparent;
      transition: border-color 0.2s ease;

      &:hover {
        border-bottom-color: ${theme.colors.text.link};
      }
    }

    /* Horizontal rules */
    hr {
      border: none;
      border-top: 1px solid ${theme.colors.border.weak};
      margin: ${theme.spacing(2, 0)};
    }

    /* Tables (if ever used) */
    table {
      border-collapse: collapse;
      margin: ${theme.spacing(1.5, 0)};
      width: 100%;
    }

    th, td {
      border: 1px solid ${theme.colors.border.weak};
      padding: ${theme.spacing(1)};
      text-align: left;
    }

    th {
      background: ${theme.colors.background.secondary};
      font-weight: ${theme.typography.fontWeightMedium};
    }

    /* Strong and emphasis */
    strong {
      font-weight: ${theme.typography.fontWeightBold};
      color: ${theme.colors.text.primary};
    }

    em {
      font-style: italic;
      color: ${theme.colors.text.secondary};
    }

    /* Emojis and icons - ensure consistent size */
    img.emoji {
      display: inline-block;
      height: 1em;
      width: 1em;
      margin: 0 0.05em 0 0.1em;
      vertical-align: -0.1em;
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
  inputContainer: css`
    border-top: 1px solid ${theme.colors.border.weak};
  `,
  modeSelector: css`
    padding: ${theme.spacing(1)} ${theme.spacing(2)};
    background: ${theme.colors.background.secondary};
    border-bottom: 1px solid ${theme.colors.border.weak};
    display: flex;
    justify-content: center;
  `,
  inputArea: css`
    padding: ${theme.spacing(2)};
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
