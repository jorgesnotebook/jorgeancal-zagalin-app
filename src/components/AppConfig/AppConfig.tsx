import React, { useState, useEffect } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2, PluginConfigPageProps } from '@grafana/data';
import {
  Button,
  Field,
  TextArea,
  Switch,
  useStyles2,
  Alert,
  Slider,
  InlineField,
  InlineFieldRow,
  Combobox,
  Badge,
  Spinner,
} from '@grafana/ui';
import { getBackendSrv } from '@grafana/runtime';
import { ZagalinConfig, DEFAULT_CONFIG, PERSONALITY_PRESETS, BASE_SYSTEM_PROMPT } from '../../types/zagalinConfig';
import { checkZagalinHealth, type HealthStatus } from '../../services/llmHealthService';

export function AppConfig({ plugin }: PluginConfigPageProps<any>) {
  const s = useStyles2(getStyles);
  // Load config from plugin settings using lazy initialization
  const [config, setConfig] = useState<ZagalinConfig>(() => {
    if (plugin.meta.jsonData) {
      return { ...DEFAULT_CONFIG, ...plugin.meta.jsonData };
    }
    return DEFAULT_CONFIG;
  });
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isDirty, setIsDirty] = useState(false);
  const [healthStatus, setHealthStatus] = useState<HealthStatus | null>(null);
  const [checkingHealth, setCheckingHealth] = useState(true);

  // Check health on mount
  useEffect(() => {
    const loadHealth = async () => {
      setCheckingHealth(true);
      try {
        const status = await checkZagalinHealth();
        setHealthStatus(status);
      } catch (err) {
        console.error('Failed to check health:', err);
        // Set a default status if health check fails
        setHealthStatus({
          llm: { enabled: false, error: 'Health check failed' },
          vector: { enabled: false },
        });
      } finally {
        setCheckingHealth(false);
      }
    };
    loadHealth();
  }, []);

  const handleSave = async () => {
    try {
      setError(null);
      // Save to Grafana's plugin settings (stored in database)
      await getBackendSrv().post(`/api/plugins/${plugin.meta.id}/settings`, {
        enabled: plugin.meta.enabled,
        pinned: plugin.meta.pinned,
        jsonData: config,
      });

      setSaved(true);
      setIsDirty(false);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to save configuration');
      console.error('Failed to save config:', err);
    }
  };

  const handleReset = () => {
    setConfig(DEFAULT_CONFIG);
    setIsDirty(true);
  };

  const updateConfig = (updates: Partial<ZagalinConfig>) => {
    setConfig({ ...config, ...updates });
    setIsDirty(true);
  };

  const handlePersonalityChange = (personality: ZagalinConfig['personality']) => {
    updateConfig({
      personality,
      customInstructions: personality === 'custom' ? config.customInstructions : PERSONALITY_PRESETS[personality],
    });
  };

  const personalityOptions = [
    { label: 'Helpful (Recommended)', value: 'helpful', description: 'Balanced, clear, and practical' },
    { label: 'Technical', value: 'technical', description: 'For experienced SREs' },
    { label: 'Beginner-Friendly', value: 'beginner-friendly', description: 'Explains concepts in detail' },
    { label: 'Concise', value: 'concise', description: 'Brief and to the point' },
    { label: 'Custom', value: 'custom', description: 'Write your own prompt' },
  ];

  return (
    <div className={s.container}>
      <div className={s.header}>
        <div>
          <h2>Zagalin Configuration</h2>
          <p className={s.subtitle}>Customize Zagalin&apos;s behavior, personality, and features</p>
        </div>
        <div className={s.actions}>
          <Button variant="secondary" onClick={handleReset} disabled={!isDirty}>
            Reset to Defaults
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={!isDirty}>
            Save Configuration
          </Button>
        </div>
      </div>

      {saved && (
        <Alert title="Configuration saved successfully" severity="success">
          Your changes have been saved to the database and will apply to all users in this organization.
        </Alert>
      )}

      {error && (
        <Alert title="Failed to save configuration" severity="error" onRemove={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* System Status */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>System Status</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Current health status of Zagalin&apos;s backend services
          </p>

          {checkingHealth ? (
            <div className={s.statusRow}>
              <Spinner inline /> Checking service health...
            </div>
          ) : (
            <>
              <div className={s.statusRow}>
                <span className={s.statusLabel}>LLM Service:</span>
                {healthStatus?.llm.enabled ? (
                  <>
                    <Badge color="green" text="Ready" icon="check" />
                    {healthStatus.llm.provider && (
                      <span className={s.statusDetail}>Provider: {healthStatus.llm.provider}</span>
                    )}
                    {healthStatus.llm.models && healthStatus.llm.models.length > 0 && (
                      <span className={s.statusDetail}>Models: {healthStatus.llm.models.join(', ')}</span>
                    )}
                  </>
                ) : (
                  <>
                    <Badge color="red" text="Unavailable" icon="exclamation-triangle" />
                    {healthStatus?.llm.error && (
                      <span className={s.statusError}>{healthStatus.llm.error}</span>
                    )}
                  </>
                )}
              </div>

              <div className={s.statusRow}>
                <span className={s.statusLabel}>Vector Search:</span>
                {healthStatus?.vector.enabled ? (
                  <>
                    <Badge color="green" text="Ready" icon="check" />
                    {healthStatus.vector.version && (
                      <span className={s.statusDetail}>Version: {healthStatus.vector.version}</span>
                    )}
                  </>
                ) : (
                  <>
                    <Badge color="orange" text="Not Available" icon="info-circle" />
                    <span className={s.statusDetail}>Optional feature for semantic search</span>
                  </>
                )}
              </div>

              {!healthStatus?.llm.enabled && (
                <Alert title="LLM Service Not Available" severity="warning">
                  <p>Zagalin requires the Grafana LLM plugin to be installed and configured.</p>
                  <ol>
                    <li>Install the <code>grafana-llm-app</code> plugin</li>
                    <li>Configure it with your LLM provider (OpenAI, Azure, Anthropic, or Grafana)</li>
                    <li>Enable the plugin and refresh this page</li>
                  </ol>
                </Alert>
              )}
            </>
          )}
        </div>
      </div>

      {/* Personality & Behavior */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Personality & Behavior</h3>
        <div className={s.sectionContent}>
          <Field
            label="Personality Preset"
            description="Choose how Zagalin communicates with you"
          >
            <Combobox
              options={personalityOptions}
              value={config.personality}
              onChange={(option) => handlePersonalityChange(option.value as ZagalinConfig['personality'])}
              width={50}
            />
          </Field>

          <Field
            label="Base System Prompt (Read-Only)"
            description="Core identity and rules that cannot be changed - ensures consistent behavior"
          >
            <TextArea
              value={BASE_SYSTEM_PROMPT}
              rows={6}
              disabled={true}
              className={s.readOnlyPrompt}
            />
          </Field>

          <Field
            label="Custom Instructions"
            description="Additional instructions to customize Zagalin's behavior (appended to base prompt)"
          >
            <TextArea
              value={config.customInstructions}
              onChange={(e) => updateConfig({ customInstructions: e.currentTarget.value })}
              rows={10}
              placeholder="Enter custom instructions..."
              disabled={config.personality !== 'custom'}
              className={s.customInstructions}
            />
          </Field>

          <Field
            label="Temperature"
            description="Controls creativity vs. consistency (0.0 = factual, 1.0 = creative)"
          >
            <div>
              <div className={s.sliderContainer}>
                <Slider
                  inputId="temperature-slider"
                  min={0}
                  max={1}
                  step={0.1}
                  value={config.temperature}
                  onChange={(value) => updateConfig({ temperature: value })}
                />
                <span className={s.sliderValue}>{config.temperature.toFixed(1)}</span>
              </div>
              <div className={s.sliderLabels}>
                <span>Factual</span>
                <span>Balanced</span>
                <span>Creative</span>
              </div>
            </div>
          </Field>

          <Field
            label="Max Tokens per Response"
            description="Maximum length of responses (higher = longer, more expensive)"
          >
            <Combobox
              options={[
                { label: '1000 tokens (~750 words)', value: 1000 },
                { label: '2000 tokens (~1500 words) - Recommended', value: 2000 },
                { label: '3000 tokens (~2250 words)', value: 3000 },
                { label: '4000 tokens (~3000 words)', value: 4000 },
              ]}
              value={config.maxTokens}
              onChange={(option) => updateConfig({ maxTokens: option.value as number })}
              width={50}
            />
          </Field>
        </div>
      </div>

      {/* Skills & Features */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Skills & Features</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Enable or disable specific assistant capabilities
          </p>

          <InlineFieldRow>
            <InlineField label="Explain Panel" labelWidth={24}>
              <Switch
                value={config.enabledSkills.explainPanel}
                onChange={(e) =>
                  updateConfig({
                    enabledSkills: { ...config.enabledSkills, explainPanel: e.currentTarget.checked },
                  })
                }
              />
            </InlineField>
            <span className={s.skillDescription}>Analyzes and explains dashboard panels</span>
          </InlineFieldRow>

          <InlineFieldRow>
            <InlineField label="Generate Queries" labelWidth={24}>
              <Switch
                value={config.enabledSkills.generateQuery}
                onChange={(e) =>
                  updateConfig({
                    enabledSkills: { ...config.enabledSkills, generateQuery: e.currentTarget.checked },
                  })
                }
              />
            </InlineField>
            <span className={s.skillDescription}>Creates PromQL/LogQL queries from natural language</span>
          </InlineFieldRow>

          <InlineFieldRow>
            <InlineField label="Troubleshooting" labelWidth={24}>
              <Switch
                value={config.enabledSkills.troubleshooting}
                onChange={(e) =>
                  updateConfig({
                    enabledSkills: { ...config.enabledSkills, troubleshooting: e.currentTarget.checked },
                  })
                }
              />
            </InlineField>
            <span className={s.skillDescription}>Provides structured troubleshooting guidance</span>
          </InlineFieldRow>
        </div>
      </div>

      {/* UI Preferences */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>UI Preferences</h3>
        <div className={s.sectionContent}>
          <InlineFieldRow>
            <InlineField label="Show Context Badge" labelWidth={24}>
              <Switch
                value={config.showContextBadge}
                onChange={(e) => updateConfig({ showContextBadge: e.currentTarget.checked })}
              />
            </InlineField>
            <span className={s.skillDescription}>Display green badge when dashboard context is active</span>
          </InlineFieldRow>

          <InlineFieldRow>
            <InlineField label="Show Cost Information" labelWidth={24}>
              <Switch
                value={config.showCostInfo}
                onChange={(e) => updateConfig({ showCostInfo: e.currentTarget.checked })}
              />
            </InlineField>
            <span className={s.skillDescription}>Show token count and cost estimates in messages</span>
          </InlineFieldRow>

          <InlineFieldRow>
            <InlineField label="Auto-Open on Dashboard" labelWidth={24}>
              <Switch
                value={config.autoOpenOnDashboard}
                onChange={(e) => updateConfig({ autoOpenOnDashboard: e.currentTarget.checked })}
              />
            </InlineField>
            <span className={s.skillDescription}>Automatically open chat when viewing dashboards</span>
          </InlineFieldRow>
        </div>
      </div>

      <div className={s.footer}>
        <Button variant="primary" onClick={handleSave} disabled={!isDirty} size="lg">
          Save Configuration
        </Button>
      </div>
    </div>
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  container: css`
    padding: ${theme.spacing(3)};
    max-width: 1200px;
    overflow-y: visible;
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: ${theme.spacing(3)};
    padding-bottom: ${theme.spacing(2)};
    border-bottom: 1px solid ${theme.colors.border.weak};
  `,
  subtitle: css`
    color: ${theme.colors.text.secondary};
    margin-top: ${theme.spacing(0.5)};
  `,
  actions: css`
    display: flex;
    gap: ${theme.spacing(1)};
  `,
  section: css`
    margin-top: ${theme.spacing(3)};
    padding: ${theme.spacing(2)};
    border: 1px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.secondary};
  `,
  sectionTitle: css`
    margin: 0 0 ${theme.spacing(2)} 0;
    font-size: ${theme.typography.h5.fontSize};
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  sectionContent: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(2)};
  `,
  marginTop: css`
    margin-top: ${theme.spacing(3)};
  `,
  systemPrompt: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  readOnlyPrompt: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    font-size: ${theme.typography.bodySmall.fontSize};
    background: ${theme.colors.background.secondary};
    opacity: 0.7;
  `,
  customInstructions: css`
    font-family: ${theme.typography.fontFamilyMonospace};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  sliderContainer: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(1)};
  `,
  sliderValue: css`
    font-weight: ${theme.typography.fontWeightBold};
    min-width: 40px;
  `,
  sliderLabels: css`
    display: flex;
    justify-content: space-between;
    font-size: ${theme.typography.bodySmall.fontSize};
    color: ${theme.colors.text.secondary};
    margin-top: ${theme.spacing(0.5)};
  `,
  description: css`
    color: ${theme.colors.text.secondary};
    margin-bottom: ${theme.spacing(2)};
  `,
  skillDescription: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
    margin-left: ${theme.spacing(1)};
  `,
  footer: css`
    margin-top: ${theme.spacing(4)};
    padding-top: ${theme.spacing(3)};
    border-top: 1px solid ${theme.colors.border.weak};
    display: flex;
    justify-content: center;
  `,
  statusRow: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.primary};
    border-radius: ${theme.shape.radius.default};
  `,
  statusLabel: css`
    font-weight: ${theme.typography.fontWeightMedium};
    min-width: 120px;
  `,
  statusDetail: css`
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
  statusError: css`
    color: ${theme.colors.error.text};
    font-size: ${theme.typography.bodySmall.fontSize};
  `,
});

export default AppConfig;
