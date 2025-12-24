import React, { useState, useEffect, KeyboardEvent } from 'react';
import { css } from '@emotion/css';
import { PanelProps, GrafanaTheme2 } from '@grafana/data';
import { Button, TextArea, Combobox, useStyles2, IconButton, Spinner, Alert, Badge } from '@grafana/ui';
import { llm } from '@grafana/llm';
import { finalize } from 'rxjs';
import { testIds } from '../testIds';
import { ZagalinColors } from '../../theme/colors';
import { isLLMReady } from '../../services/llmHealthService';

interface AskPanelOptions {
  includeTimeRange?: boolean;
  includeDashboardVariables?: boolean;
  promptTemplate?: string;
}

interface Props extends PanelProps<AskPanelOptions> {}

export const AskPanel: React.FC<Props> = ({ options, data, width, height, timeRange, replaceVariables }) => {
  const s = useStyles2(getStyles);
  const [input, setInput] = useState('');
  const [response, setResponse] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [llmReady, setLlmReady] = useState<boolean | null>(null);

  // Check LLM health on mount
  useEffect(() => {
    const checkHealth = async () => {
      const ready = await isLLMReady();
      setLlmReady(ready);
      if (!ready) {
        setError('LLM service is not available. Please ensure Grafana LLM plugin is configured.');
      }
    };
    checkHealth();
  }, []);

  const templateOptions = [
    { label: 'Custom', value: '' },
    { label: 'Explain this panel', value: 'Explain what this dashboard panel shows and what insights I can gain from it.' },
    { label: 'Create alert', value: 'Help me create an alert rule for this metric. Suggest appropriate thresholds and conditions.' },
    { label: 'Write PromQL', value: 'Help me write a PromQL query to monitor ' },
  ];

  const buildContextualPrompt = (userPrompt: string): string => {
    let contextualPrompt = userPrompt;

    // Add time range context if enabled
    if (options.includeTimeRange && timeRange) {
      const timeContext = `\n\nTime range: ${timeRange.from.format('YYYY-MM-DD HH:mm:ss')} to ${timeRange.to.format('YYYY-MM-DD HH:mm:ss')}`;
      contextualPrompt = userPrompt + timeContext;
    }

    // Add dashboard variables if enabled
    if (options.includeDashboardVariables && replaceVariables) {
      // Get all template variables
      const variables = replaceVariables('$__all_variables', {});
      if (variables && variables !== '$__all_variables') {
        contextualPrompt += `\n\nDashboard variables: ${variables}`;
      }
    }

    return contextualPrompt;
  };

  const handleAsk = async () => {
    if (!input.trim() || isLoading) {
      return;
    }

    // Health check before sending
    if (llmReady === false) {
      setError('LLM service is not available. Please check your configuration.');
      return;
    }

    setIsLoading(true);
    setError(null);
    setResponse('');

    try {
      const contextualPrompt = buildContextualPrompt(input);

      // Use Grafana LLM plugin (grafana-llm-app)
      const stream = llm
        .streamChatCompletions({
          model: 'gpt-4o-mini',
          messages: [
            {
              role: 'system',
              content: 'You are Zagalin, an AI assistant for Grafana. Help users understand their metrics, write queries, and troubleshoot issues. Keep responses concise and practical.',
            },
            {
              role: 'user',
              content: contextualPrompt,
            },
          ],
        })
        .pipe(
          llm.accumulateContent(),
          finalize(() => {
            setIsLoading(false);
          })
        );

      let fullResponse = '';
      stream.subscribe({
        next: (content: string) => {
          fullResponse = content;
          setResponse(fullResponse);
        },
        error: (err) => {
          console.error('Zagalin: Stream error', err);
          setError(err.message || 'Failed to get response from LLM');
          setIsLoading(false);
        },
      });
    } catch (err: any) {
      console.error('Zagalin: Ask error', err);
      setError(err.message || 'Failed to get response');
      setIsLoading(false);
    }
  };

  const handleKeyPress = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Cmd/Ctrl + Enter to send
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault();
      handleAsk();
    }
  };

  const copyToClipboard = () => {
    if (response) {
      navigator.clipboard.writeText(response);
    }
  };

  const applyTemplate = (template: string) => {
    if (template) {
      setInput(template);
    }
  };

  return (
    <div className={s.container} data-testid={testIds.askPanel.container} style={{ width, height }}>
      <div className={s.header}>
        <h5 className={s.title}>Ask AI</h5>
        {llmReady === null ? (
          <Badge color="blue" text="Checking..." icon="sync" />
        ) : llmReady ? (
          <Badge color="green" text="Ready" icon="check" />
        ) : (
          <Badge color="red" text="Unavailable" icon="exclamation-triangle" />
        )}
      </div>

      {error && (
        <Alert title="Error" severity="error" onRemove={() => setError(null)}>
          {error}
        </Alert>
      )}

      <div className={s.content}>
        <div className={s.inputSection}>
          <div className={s.templateRow}>
            <Combobox
              options={templateOptions}
              placeholder="Select a template"
              onChange={option => applyTemplate(option.value!)}
              width={30}
            />
          </div>

          <TextArea
            data-testid={testIds.askPanel.input}
            value={input}
            onChange={e => setInput(e.currentTarget.value)}
            onKeyDown={handleKeyPress}
            placeholder="Ask a question about your metrics, create queries, or get insights... (Cmd/Ctrl+Enter to send)"
            rows={3}
            className={s.input}
            disabled={isLoading}
          />

          <div className={s.inputActions}>
            <div className={s.keyboardHint}>
              <kbd>Cmd/Ctrl + Enter</kbd> to send
            </div>
            <Button
              data-testid={testIds.askPanel.submit}
              icon={isLoading ? undefined : 'comment-alt'}
              onClick={handleAsk}
              variant="primary"
              size="sm"
              disabled={!input.trim() || isLoading}
              className={s.askButton}
            >
              {isLoading ? <Spinner inline /> : 'Ask'}
            </Button>
          </div>
        </div>

        {response && (
          <div className={s.responseSection}>
            <div className={s.responseHeader}>
              <strong>Response</strong>
              <IconButton
                data-testid={testIds.askPanel.copy}
                name="copy"
                tooltip="Copy to clipboard"
                onClick={copyToClipboard}
              />
            </div>
            <div className={s.response}>{response}</div>
          </div>
        )}
      </div>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    display: flex;
    flex-direction: column;
    padding: ${theme.spacing(2)};
    background: ${theme.colors.background.primary};
    overflow: hidden;
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: ${theme.spacing(2)};
    padding-bottom: ${theme.spacing(1)};
    border-bottom: 2px solid;
    border-image: ${ZagalinColors.orangeGradient} 1;
  `,
  title: css`
    margin: 0;
    font-size: ${theme.typography.h5.fontSize};
    font-weight: ${theme.typography.h5.fontWeight};
    background: ${ZagalinColors.orangeGradient};
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  `,
  controls: css`
    display: flex;
    gap: ${theme.spacing(1)};
    align-items: center;
  `,
  content: css`
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
    overflow-y: auto;
  `,
  inputSection: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1)};
  `,
  templateRow: css`
    display: flex;
    gap: ${theme.spacing(1)};
  `,
  input: css`
    width: 100%;
  `,
  inputActions: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
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
      font-size: ${theme.typography.bodySmall.fontSize};
    }
  `,
  responseSection: css`
    flex: 1;
    display: flex;
    flex-direction: column;
    background: ${theme.colors.background.secondary};
    border-radius: ${theme.shape.radius.default};
    padding: ${theme.spacing(2)};
    overflow-y: auto;
  `,
  responseHeader: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: ${theme.spacing(1)};
  `,
  response: css`
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.6;
  `,
  askButton: css`
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
