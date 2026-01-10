import React, { useState } from 'react';
import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';
import {
  Button,
  Field,
  FieldSet,
  Combobox,
  TextArea,
  Switch,
  useStyles2,
  Alert,
  Slider,
  InlineField,
  InlineFieldRow,
} from '@grafana/ui';
import { ZagalinConfig, DEFAULT_CONFIG, PERSONALITY_PRESETS, BASE_SYSTEM_PROMPT } from '../../types/zagalinConfig';

const STORAGE_KEY = 'zagalin-config';

export function ConfigPage() {
  const s = useStyles2(getStyles);
  const [config, setConfig] = useState<ZagalinConfig>(() => {
    const savedConfig = localStorage.getItem(STORAGE_KEY);
    if (savedConfig) {
      try {
        return JSON.parse(savedConfig);
      } catch (e) {
        console.error('Failed to load config:', e);
      }
    }
    return DEFAULT_CONFIG;
  });
  const [saved, setSaved] = useState(false);
  const [isDirty, setIsDirty] = useState(false);

  const handleSave = () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
    setSaved(true);
    setIsDirty(false);
    setTimeout(() => setSaved(false), 3000);
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
          Your changes have been saved and will take effect immediately.
        </Alert>
      )}

      {/* Personality & Behavior */}
      <FieldSet label="Personality & Behavior">
        <div>
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
                  inputId="temperature-slider-config"
                  min={0}
                  max={1}
                  step={0.1}
                  value={config.standardMode.temperature}
                  onChange={(value) => updateConfig({
                    standardMode: { ...config.standardMode, temperature: value }
                  })}
                />
                <span className={s.sliderValue}>{config.standardMode.temperature.toFixed(1)}</span>
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
              value={config.standardMode.maxTokens}
              onChange={(option) => updateConfig({
                standardMode: { ...config.standardMode, maxTokens: option.value as number }
              })}
              width={50}
            />
          </Field>
        </div>
      </FieldSet>

      {/* Skills & Features */}
      <FieldSet label="Skills & Features" className={s.marginTop}>
        <div>
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
      </FieldSet>

      {/* UI Preferences */}
      <FieldSet label="UI Preferences" className={s.marginTop}>
        <div>
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
      </FieldSet>

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
    margin: 0 auto;
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
});

export default ConfigPage;
