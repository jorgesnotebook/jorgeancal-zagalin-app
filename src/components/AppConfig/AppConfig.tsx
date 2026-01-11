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
  Input,
  Icon,
} from '@grafana/ui';
import { getBackendSrv } from '@grafana/runtime';
import { ZagalinConfig, DEFAULT_CONFIG, PERSONALITY_PRESETS } from '../../types/zagalinConfig';
import { checkZagalinHealth, type HealthStatus } from '../../services/llmHealthService';
import { listDatasources, type DatasourceInfo } from '../../services/datasourceService';
import { VersionWarning } from '../VersionWarning/VersionWarning';

export function AppConfig({ plugin }: PluginConfigPageProps<any>) {
  const s = useStyles2(getStyles);
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
  const [datasources, setDatasources] = useState<DatasourceInfo[]>([]);
  const [loadingDatasources, setLoadingDatasources] = useState(true);
  const [allowedDatasources, setAllowedDatasources] = useState<string[]>(
    plugin.meta.jsonData?.allowedDatasources || []
  );
  const [defaultDatasource, setDefaultDatasource] = useState<string>(plugin.meta.jsonData?.defaultDatasource || '');

  const [otelEnabled, setOtelEnabled] = useState<boolean>(plugin.meta.jsonData?.otelEnforcement?.enabled || false);
  const [otelRequireService, setOtelRequireService] = useState<boolean>(
    plugin.meta.jsonData?.otelEnforcement?.requireServiceName !== false
  );
  const [otelRequireEnvironment, setOtelRequireEnvironment] = useState<boolean>(
    plugin.meta.jsonData?.otelEnforcement?.requireEnvironmentName !== false
  );
  const [otelDefaultService, setOtelDefaultService] = useState<string>(
    plugin.meta.jsonData?.otelEnforcement?.defaultServiceName || ''
  );
  const [otelDefaultEnvironment, setOtelDefaultEnvironment] = useState<string>(
    plugin.meta.jsonData?.otelEnforcement?.defaultEnvironmentName || ''
  );
  const [otelRejectIfNoScope, setOtelRejectIfNoScope] = useState<boolean>(
    plugin.meta.jsonData?.otelEnforcement?.rejectIfNoScope !== false
  );

  const [llmBackend, setLlmBackend] = useState<string>(plugin.meta.jsonData?.llmBackend || 'grafana-llm');
  const [llmProvider, setLlmProvider] = useState<string>(plugin.meta.jsonData?.llmProvider || 'openai');
  const [llmModel, setLlmModel] = useState<string>(plugin.meta.jsonData?.llmModel || 'gpt-4o-mini');
  const [llmEndpoint, setLlmEndpoint] = useState<string>(plugin.meta.jsonData?.llmEndpoint || '');
  const [llmOrganization, setLlmOrganization] = useState<string>(plugin.meta.jsonData?.llmOrganization || '');
  const [llmApiKey, setLlmApiKey] = useState<string>(plugin.meta.secureJsonData?.llmApiKey || '');
  const [serviceAccountToken, setServiceAccountToken] = useState<string>(
    plugin.meta.secureJsonData?.serviceAccountToken || ''
  );
  const [hasServiceAccountToken, setHasServiceAccountToken] = useState<boolean>(
    plugin.meta.secureJsonFields?.serviceAccountToken || false
  );

  const [queryValidationEnabled, setQueryValidationEnabled] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.enabled || false
  );
  const [queryValidationEnablePromQL, setQueryValidationEnablePromQL] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.enablePromqlValidation || false
  );
  const [queryValidationEnableLogQL, setQueryValidationEnableLogQL] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.enableLogqlValidation || false
  );
  const [queryValidationEnableTraceQL, setQueryValidationEnableTraceQL] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.enableTraceqlValidation || false
  );
  const [queryValidationStrictMode, setQueryValidationStrictMode] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.strictMode || false
  );
  const [queryValidationMaxComplexity, setQueryValidationMaxComplexity] = useState<number>(
    plugin.meta.jsonData?.queryValidation?.maxQueryComplexity || 100
  );
  const [queryValidationLogAttempts, setQueryValidationLogAttempts] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.logValidationAttempts !== false
  );
  const [queryValidationEnableLLM, setQueryValidationEnableLLM] = useState<boolean>(
    plugin.meta.jsonData?.queryValidation?.enableLlmValidation || false
  );
  const [queryValidationLLMMode, setQueryValidationLLMMode] = useState<string>(
    plugin.meta.jsonData?.queryValidation?.llmValidationMode || 'advisory'
  );

  const [maxQueryTimeRangeHours, setMaxQueryTimeRangeHours] = useState<number>(
    plugin.meta.jsonData?.maxQueryTimeRangeHours ?? 24
  );

  const [referenceDashboards, setReferenceDashboards] = useState<string[]>(
    plugin.meta.jsonData?.referenceDashboards || []
  );

  useEffect(() => {
    if (plugin.meta.jsonData) {
      let migratedConfig = { ...DEFAULT_CONFIG, ...plugin.meta.jsonData };

      const hasOldFormat =
        plugin.meta.jsonData.standardModeTemperature !== undefined || plugin.meta.jsonData.temperature !== undefined;
      const hasNewFormat = plugin.meta.jsonData.standardMode !== undefined;

      if (hasOldFormat && !hasNewFormat) {
        migratedConfig = {
          ...migratedConfig,
          standardMode: {
            temperature: plugin.meta.jsonData.standardModeTemperature ?? plugin.meta.jsonData.temperature ?? 0.5,
            maxTokens: plugin.meta.jsonData.standardModeMaxTokens ?? plugin.meta.jsonData.maxTokens ?? 2000,
          },
          designMode: {
            temperature: plugin.meta.jsonData.designModeTemperature ?? 0.8,
            maxTokens: plugin.meta.jsonData.designModeMaxTokens ?? 3000,
          },
        };
        console.log('Migrated old LLM config format to new nested structure');
      }

      setConfig(migratedConfig);
      setAllowedDatasources(plugin.meta.jsonData.allowedDatasources || []);
      setDefaultDatasource(plugin.meta.jsonData.defaultDatasource || '');
      setLlmBackend(plugin.meta.jsonData.llmBackend || 'grafana-llm');
      setLlmProvider(plugin.meta.jsonData.llmProvider || 'openai');
      setLlmModel(plugin.meta.jsonData.llmModel || 'gpt-4o-mini');
      setLlmEndpoint(plugin.meta.jsonData.llmEndpoint || '');
      setLlmOrganization(plugin.meta.jsonData.llmOrganization || '');

      setOtelEnabled(plugin.meta.jsonData.otelEnforcement?.enabled || false);
      setOtelRequireService(plugin.meta.jsonData.otelEnforcement?.requireServiceName !== false);
      setOtelRequireEnvironment(plugin.meta.jsonData.otelEnforcement?.requireEnvironmentName !== false);
      setOtelDefaultService(plugin.meta.jsonData.otelEnforcement?.defaultServiceName || '');
      setOtelDefaultEnvironment(plugin.meta.jsonData.otelEnforcement?.defaultEnvironmentName || '');
      setOtelRejectIfNoScope(plugin.meta.jsonData.otelEnforcement?.rejectIfNoScope !== false);

      setQueryValidationEnabled(plugin.meta.jsonData.queryValidation?.enabled || false);
      setQueryValidationEnablePromQL(plugin.meta.jsonData.queryValidation?.enablePromqlValidation || false);
      setQueryValidationEnableLogQL(plugin.meta.jsonData.queryValidation?.enableLogqlValidation || false);
      setQueryValidationEnableTraceQL(plugin.meta.jsonData.queryValidation?.enableTraceqlValidation || false);
      setQueryValidationStrictMode(plugin.meta.jsonData.queryValidation?.strictMode || false);
      setQueryValidationMaxComplexity(plugin.meta.jsonData.queryValidation?.maxQueryComplexity || 100);
      setQueryValidationLogAttempts(plugin.meta.jsonData.queryValidation?.logValidationAttempts !== false);
      setQueryValidationEnableLLM(plugin.meta.jsonData.queryValidation?.enableLlmValidation || false);
      setQueryValidationLLMMode(plugin.meta.jsonData.queryValidation?.llmValidationMode || 'advisory');

      setReferenceDashboards(plugin.meta.jsonData.referenceDashboards || []);
    }

    if (plugin.meta.secureJsonFields) {
      setHasServiceAccountToken(plugin.meta.secureJsonFields.serviceAccountToken || false);
    }
  }, [plugin.meta.jsonData, plugin.meta.secureJsonFields]);

  useEffect(() => {
    const loadHealth = async () => {
      setCheckingHealth(true);
      try {
        const status = await checkZagalinHealth();
        setHealthStatus(status);
      } catch (err) {
        console.error('Failed to check health:', err);
        setHealthStatus({
          llm: { enabled: false, error: 'Unable to check LLM status' },
          vector: { enabled: false, error: 'Unable to check vector status' },
        });
      } finally {
        setCheckingHealth(false);
      }
    };
    loadHealth();
  }, []);

  useEffect(() => {
    const loadDatasources = async () => {
      setLoadingDatasources(true);
      try {
        const response = await listDatasources();
        setDatasources(response.datasources);
      } catch (err) {
        console.error('Failed to load datasources:', err);
      } finally {
        setLoadingDatasources(false);
      }
    };
    loadDatasources();
  }, []);

  const handleSave = async () => {
    try {
      setError(null);

      if (llmBackend === 'backend-proxy' && !serviceAccountToken && !hasServiceAccountToken) {
        setError(
          'Service account token is required when using Backend Proxy mode. Please provide a token or switch to a different LLM backend.'
        );
        return;
      }

      const settings: any = {
        enabled: plugin.meta.enabled,
        pinned: plugin.meta.pinned,
        jsonData: {
          ...config,
          // Flatten mode settings for backend compatibility
          standardModeTemperature: config.standardMode.temperature,
          standardModeMaxTokens: config.standardMode.maxTokens,
          designModeTemperature: config.designMode.temperature,
          designModeMaxTokens: config.designMode.maxTokens,
          allowedDatasources,
          defaultDatasource,
          maxQueryTimeRangeHours,
          llmBackend: llmBackend === 'disabled' ? '' : llmBackend,
          llmProvider,
          llmModel,
          llmEndpoint,
          llmOrganization,
          otelEnforcement: {
            enabled: otelEnabled,
            requireServiceName: otelRequireService,
            requireEnvironmentName: otelRequireEnvironment,
            defaultServiceName: otelDefaultService,
            defaultEnvironmentName: otelDefaultEnvironment,
            rejectIfNoScope: otelRejectIfNoScope,
          },
          queryValidation: {
            enabled: queryValidationEnabled,
            enablePromqlValidation: queryValidationEnablePromQL,
            enableLogqlValidation: queryValidationEnableLogQL,
            enableTraceqlValidation: queryValidationEnableTraceQL,
            strictMode: queryValidationStrictMode,
            maxQueryComplexity: queryValidationMaxComplexity,
            logValidationAttempts: queryValidationLogAttempts,
            enableLlmValidation: queryValidationEnableLLM,
            llmValidationMode: queryValidationLLMMode,
          },
          referenceDashboards,
        },
      };

      settings.secureJsonData = {};
      if (llmApiKey) {
        settings.secureJsonData.llmApiKey = llmApiKey;
      }
      if (serviceAccountToken) {
        settings.secureJsonData.serviceAccountToken = serviceAccountToken;
      }

      await getBackendSrv().post(`/api/plugins/${plugin.meta.id}/settings`, settings);

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
    setAllowedDatasources([]);
    setDefaultDatasource('');
    setLlmBackend('grafana-llm-app');
    setLlmProvider('openai');
    setLlmModel('gpt-4o-mini');
    setLlmEndpoint('');
    setLlmOrganization('');
    setLlmApiKey('');
    setServiceAccountToken('');
    setHasServiceAccountToken(false);
    setOtelEnabled(false);
    setOtelRequireService(true);
    setOtelRequireEnvironment(true);
    setOtelDefaultService('');
    setOtelDefaultEnvironment('');
    setOtelRejectIfNoScope(true);
    setQueryValidationEnabled(false);
    setQueryValidationEnablePromQL(false);
    setQueryValidationEnableLogQL(false);
    setQueryValidationEnableTraceQL(false);
    setQueryValidationStrictMode(false);
    setQueryValidationMaxComplexity(100);
    setQueryValidationLogAttempts(true);
    setQueryValidationEnableLLM(false);
    setQueryValidationLLMMode('advisory');
    setMaxQueryTimeRangeHours(24);
    setIsDirty(true);
  };

  const handleDatasourceToggle = (uid: string) => {
    const newAllowed = allowedDatasources.includes(uid)
      ? allowedDatasources.filter((ds) => ds !== uid)
      : [...allowedDatasources, uid];
    setAllowedDatasources(newAllowed);

    if (!newAllowed.includes(defaultDatasource)) {
      setDefaultDatasource('');
    }

    setIsDirty(true);
  };

  const handleDefaultDatasourceChange = (uid: string) => {
    setDefaultDatasource(uid);
    if (!allowedDatasources.includes(uid)) {
      setAllowedDatasources([...allowedDatasources, uid]);
    }
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
      {/* Grafana Version Compatibility Warning */}
      <VersionWarning />

      {/* Admin-Only Banner */}
      <Alert title="Admin Configuration" severity="info">
        <div>
          <strong>⚠️ Administrator Access Required</strong>
          <p style={{ marginTop: '8px', marginBottom: '0' }}>
            This configuration page is only accessible to Organization Administrators. All changes made here affect all
            users in this Grafana organization.
          </p>
        </div>
      </Alert>

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
          <p className={s.description}>Current health status of Zagalin&apos;s backend services</p>

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
                    {healthStatus?.llm.error && <span className={s.statusError}>{healthStatus.llm.error}</span>}
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
                    <li>
                      Install the <code>grafana-llm-app</code> plugin
                    </li>
                    <li>Configure it with your LLM provider (OpenAI, Azure, Anthropic, or Grafana)</li>
                    <li>Enable the plugin and refresh this page</li>
                  </ol>
                </Alert>
              )}
            </>
          )}
        </div>
      </div>

      {/* LLM Backend Configuration */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>LLM Backend Configuration</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Configure how Zagalin connects to LLM services. You can use grafana-llm-app for centralized configuration,
            bring your own API keys, or disable LLM features entirely.
          </p>

          {/* LLM Backend Mode Selection - 3 Options */}
          <div className={s.llmBackendCards}>
            {/* Option 1: Official Grafana (@grafana/llm) - DEFAULT */}
            <div
              className={`${s.llmBackendCard} ${llmBackend === 'grafana-llm' ? s.llmBackendCardActive : ''}`}
              onClick={() => {
                setLlmBackend('grafana-llm');
                setIsDirty(true);
              }}
            >
              <div className={s.llmBackendCardHeader}>
                <input
                  type="radio"
                  checked={llmBackend === 'grafana-llm'}
                  onChange={() => {
                    setLlmBackend('grafana-llm');
                    setIsDirty(true);
                  }}
                  className={s.llmBackendCardRadio}
                />
                <Icon name="plug" size="xl" />
                <h4>Official Grafana (Default)</h4>
              </div>
              <p className={s.llmBackendCardDescription}>
                Hybrid mode: Uses @grafana/llm for LLM calls (no service account), backend for queries/security.
              </p>
              <ul style={{ marginTop: '8px', paddingLeft: '20px', fontSize: '12px' }}>
                <li>✅ No service account needed</li>
                <li>✅ Backend query validation</li>
                <li>✅ Rate limiting & security</li>
              </ul>
            </div>

            {/* Option 2: Backend Proxy (Production Recommended) */}
            <div
              className={`${s.llmBackendCard} ${llmBackend === 'backend-proxy' ? s.llmBackendCardActive : ''}`}
              onClick={() => {
                setLlmBackend('backend-proxy');
                setIsDirty(true);
              }}
            >
              <div className={s.llmBackendCardHeader}>
                <input
                  type="radio"
                  checked={llmBackend === 'backend-proxy'}
                  onChange={() => {
                    setLlmBackend('backend-proxy');
                    setIsDirty(true);
                  }}
                  className={s.llmBackendCardRadio}
                />
                <Icon name="shield" size="xl" />
                <h4>Zagalin Backend (Production)</h4>
              </div>
              <p className={s.llmBackendCardDescription}>
                Full security through Zagalin backend. Requires service account token.
              </p>
              <ul style={{ marginTop: '8px', paddingLeft: '20px', fontSize: '12px' }}>
                <li>✅ Rate limiting & validation</li>
                <li>✅ Full audit trail</li>
                <li>⚙️ Service account needed</li>
              </ul>
            </div>

            {/* Option 3: Direct LLM API - COMING SOON */}
            <div className={s.llmBackendCard} style={{ opacity: 0.6, cursor: 'not-allowed' }}>
              <div className={s.llmBackendCardHeader}>
                <input type="radio" checked={false} disabled={true} className={s.llmBackendCardRadio} />
                <Icon name="key-skeleton-alt" size="xl" />
                <h4>Direct LLM API</h4>
                <Badge color="orange" text="Coming Soon" style={{ marginLeft: '8px' }} />
              </div>
              <p className={s.llmBackendCardDescription}>
                Zagalin backend → OpenAI/Anthropic. Full security, no grafana-llm-app needed.
              </p>
              <ul style={{ marginTop: '8px', paddingLeft: '20px', fontSize: '12px' }}>
                <li>⏳ Under development</li>
                <li>🚧 Not yet tested</li>
              </ul>
            </div>
          </div>

          {/* Official Grafana Mode Info */}
          {llmBackend === 'grafana-llm' && (
            <Alert title="Official Grafana Mode (Default) - Hybrid Architecture" severity="success">
              <p>
                <strong>Best of both worlds!</strong> Uses @grafana/llm for LLM calls (no service account needed) while
                keeping Zagalin backend for security features.
              </p>
              <p style={{ marginTop: '8px' }}>
                <strong>What you get:</strong>
              </p>
              <ul>
                <li>
                  ✅ <strong>No service account needed</strong> - Uses session-based authentication for LLM calls
                </li>
                <li>
                  ✅ <strong>Backend security features</strong> - Rate limiting, query validation, audit logging
                </li>
                <li>
                  ✅ <strong>Datasource governance</strong> - Allowlist enforcement, OTel scope checking
                </li>
                <li>
                  ✅ <strong>Persistent storage</strong> - Conversations saved via Grafana User Storage API
                </li>
              </ul>
              <p style={{ marginTop: '8px' }}>
                <strong>Prerequisites:</strong>
              </p>
              <ol>
                <li>Install grafana-llm-app plugin from Grafana catalog</li>
                <li>Configure it with your LLM provider (Administration → Plugins → LLM App)</li>
                <li>Zagalin backend must be running (for queries and security features)</li>
              </ol>
            </Alert>
          )}

          {/* Backend Proxy Mode Info */}
          {llmBackend === 'backend-proxy' && (
            <>
              <Alert title="Backend Proxy Mode (Production Recommended)" severity="info">
                <p>
                  <strong>Full security pipeline.</strong> All requests go through Zagalin backend → grafana-llm-app
                  with rate limiting, validation, and audit logging.
                </p>
                <p style={{ marginTop: '8px' }}>
                  <strong>Prerequisites:</strong>
                </p>
                <ol>
                  <li>Install grafana-llm-app plugin from Grafana catalog</li>
                  <li>Configure it with your LLM provider (Administration → Plugins → LLM App)</li>
                  <li>
                    <strong>Required:</strong> Create a service account with <code>Editor</code> role and provide the
                    token below
                  </li>
                </ol>
              </Alert>

              <Field
                label="Service Account Token (Mandatory)"
                description={
                  hasServiceAccountToken && !serviceAccountToken
                    ? 'A service account token is currently configured (not shown for security). Leave empty to keep existing token, or enter a new token to replace it.'
                    : "Grafana service account token for backend-to-backend authentication with grafana-llm-app. Required for Backend Proxy mode. Stored securely in Grafana's encrypted storage."
                }
                invalid={!serviceAccountToken && !hasServiceAccountToken}
                error={
                  !serviceAccountToken && !hasServiceAccountToken
                    ? 'Service account token is required for Backend Proxy mode'
                    : undefined
                }
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <Input
                    type="password"
                    value={serviceAccountToken}
                    onChange={(e) => {
                      setServiceAccountToken(e.currentTarget.value);
                      setIsDirty(true);
                    }}
                    placeholder={
                      hasServiceAccountToken ? '●●●●●●●● Configured - enter new token to replace' : 'glsa_...'
                    }
                    width={50}
                    invalid={!serviceAccountToken && !hasServiceAccountToken}
                  />
                  {hasServiceAccountToken && !serviceAccountToken && (
                    <Badge color="green" text="Configured" icon="check" />
                  )}
                </div>
              </Field>

              {!serviceAccountToken && !hasServiceAccountToken && (
                <Alert title="Service Account Token Required" severity="error">
                  <p>
                    A service account token is <strong>mandatory</strong> for Backend Proxy mode. It ensures secure
                    backend-to-backend authentication with grafana-llm-app.
                  </p>
                  <p>
                    <strong>To create a service account token:</strong>
                  </p>
                  <ol>
                    <li>Go to Administration → Service Accounts</li>
                    <li>Create a new service account (e.g., &ldquo;Zagalin Plugin&rdquo;)</li>
                    <li>
                      Assign the <code>Editor</code> role
                    </li>
                    <li>Generate a token and paste it above</li>
                  </ol>
                </Alert>
              )}
            </>
          )}

          {/* Direct API Configuration - TEMPORARILY DISABLED */}
          {/* {llmBackend === 'direct' && (
            <>
              <Alert title="Direct API Mode" severity="warning">
                <p>
                  You are configuring Zagalin to call LLM providers directly. This requires your own API keys and may incur costs based on usage.
                </p>
              </Alert>

              <Field label="Provider" description="Select your LLM provider">
                <div className={s.providerCards}>
                  <div
                    className={`${s.providerCard} ${llmProvider === 'openai' ? s.providerCardActive : ''}`}
                    onClick={() => {
                      setLlmProvider('openai');
                      setLlmModel('gpt-4o-mini');
                      setIsDirty(true);
                    }}
                  >
                    <input
                      type="radio"
                      checked={llmProvider === 'openai'}
                      onChange={() => {
                        setLlmProvider('openai');
                        setLlmModel('gpt-4o-mini');
                        setIsDirty(true);
                      }}
                      className={s.providerCardRadio}
                    />
                    <Icon name="cloud" size="lg" />
                    <span className={s.providerCardLabel}>OpenAI</span>
                  </div>

                  <div
                    className={`${s.providerCard} ${llmProvider === 'anthropic' ? s.providerCardActive : ''}`}
                    onClick={() => {
                      setLlmProvider('anthropic');
                      setLlmModel('claude-3-5-sonnet-20241022');
                      setIsDirty(true);
                    }}
                  >
                    <input
                      type="radio"
                      checked={llmProvider === 'anthropic'}
                      onChange={() => {
                        setLlmProvider('anthropic');
                        setLlmModel('claude-3-5-sonnet-20241022');
                        setIsDirty(true);
                      }}
                      className={s.providerCardRadio}
                    />
                    <Icon name="cloud" size="lg" />
                    <span className={s.providerCardLabel}>Anthropic</span>
                  </div>

                  <div
                    className={`${s.providerCard} ${llmProvider === 'azure-openai' ? s.providerCardActive : ''}`}
                    onClick={() => {
                      setLlmProvider('azure-openai');
                      setLlmModel('gpt-4o-mini');
                      setIsDirty(true);
                    }}
                  >
                    <input
                      type="radio"
                      checked={llmProvider === 'azure-openai'}
                      onChange={() => {
                        setLlmProvider('azure-openai');
                        setLlmModel('gpt-4o-mini');
                        setIsDirty(true);
                      }}
                      className={s.providerCardRadio}
                    />
                    <Icon name="cloud" size="lg" />
                    <span className={s.providerCardLabel}>Azure OpenAI</span>
                  </div>

                  <div
                    className={`${s.providerCard} ${llmProvider === 'custom' ? s.providerCardActive : ''}`}
                    onClick={() => {
                      setLlmProvider('custom');
                      setIsDirty(true);
                    }}
                  >
                    <input
                      type="radio"
                      checked={llmProvider === 'custom'}
                      onChange={() => {
                        setLlmProvider('custom');
                        setIsDirty(true);
                      }}
                      className={s.providerCardRadio}
                    />
                    <Icon name="cog" size="lg" />
                    <span className={s.providerCardLabel}>Custom API</span>
                  </div>
                </div>
              </Field>

              <Field
                label="Model"
                description="Specify the model to use for completions"
              >
                <Input
                  value={llmModel}
                  onChange={(e) => {
                    setLlmModel(e.currentTarget.value);
                    setIsDirty(true);
                  }}
                  placeholder={
                    llmProvider === 'openai' || llmProvider === 'azure-openai'
                      ? 'gpt-4o-mini'
                      : llmProvider === 'anthropic'
                      ? 'claude-3-5-sonnet-20241022'
                      : 'model-name'
                  }
                  width={50}
                />
              </Field>

              {(llmProvider === 'custom' || llmProvider === 'azure-openai') && (
                <Field
                  label="API Endpoint"
                  description={
                    llmProvider === 'azure-openai'
                      ? 'Your Azure OpenAI endpoint URL (e.g., https:
                      : 'Custom API endpoint URL compatible with OpenAI API format'
                  }
                >
                  <Input
                    value={llmEndpoint}
                    onChange={(e) => {
                      setLlmEndpoint(e.currentTarget.value);
                      setIsDirty(true);
                    }}
                    placeholder={
                      llmProvider === 'azure-openai'
                        ? 'https:
                        : 'https:
                    }
                    width={50}
                  />
                </Field>
              )}

              {llmProvider === 'openai' && (
                <Field
                  label="Organization ID"
                  description="Your OpenAI Organization ID. Required if your API key belongs to multiple organizations (e.g., org-xxxxx)"
                >
                  <Input
                    value={llmOrganization}
                    onChange={(e) => {
                      setLlmOrganization(e.currentTarget.value);
                      setIsDirty(true);
                    }}
                    placeholder="org-xxxxx"
                    width={50}
                  />
                </Field>
              )}

              <Field
                label="API Key"
                description="Your LLM provider API key. This is stored securely in Grafana's encrypted storage."
              >
                <Input
                  type="password"
                  value={llmApiKey}
                  onChange={(e) => {
                    setLlmApiKey(e.currentTarget.value);
                    setIsDirty(true);
                  }}
                  placeholder={
                    llmProvider === 'openai' || llmProvider === 'azure-openai'
                      ? 'sk-...'
                      : llmProvider === 'anthropic'
                      ? 'sk-ant-...'
                      : 'your-api-key'
                  }
                  width={50}
                />
              </Field>

              {!llmApiKey && (
                <Alert title="API Key Required" severity="error">
                  Direct mode requires an API key to be configured. Enter your provider's API key above to enable LLM functionality.
                </Alert>
              )}

              {llmProvider === 'openai' && (
                <Alert title="OpenAI Configuration" severity="info">
                  <p>Get your API key from <a href="https:
                  <p><strong>Recommended models:</strong> gpt-4o, gpt-4o-mini, gpt-4-turbo</p>
                </Alert>
              )}

              {llmProvider === 'anthropic' && (
                <Alert title="Anthropic Configuration" severity="info">
                  <p>Get your API key from <a href="https:
                  <p><strong>Recommended models:</strong> claude-3-5-sonnet-20241022, claude-3-opus-20240229</p>
                </Alert>
              )}

              {llmProvider === 'azure-openai' && (
                <Alert title="Azure OpenAI Configuration" severity="info">
                  <p>You'll need:</p>
                  <ul>
                    <li>Your Azure OpenAI resource endpoint</li>
                    <li>API key from Azure Portal</li>
                    <li>Deployed model name</li>
                  </ul>
                </Alert>
              )}
            </>
          )} */}

          {/* Direct LLM Mode Info */}
          {llmBackend === 'direct' && (
            <Alert title="Direct LLM Mode" severity="info">
              <p>
                <strong>Backend calls LLM API directly.</strong> Full security features without grafana-llm-app
                dependency.
              </p>
              <p style={{ marginTop: '8px' }}>
                <strong>Requirements:</strong> Provide LLM provider API key below.
              </p>
            </Alert>
          )}
        </div>
      </div>

      {/* Datasource Governance */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Datasource Governance</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Control which datasources Zagalin can query. Leave empty to allow all datasources.
          </p>

          {loadingDatasources ? (
            <div className={s.statusRow}>
              <Spinner inline /> Loading datasources...
            </div>
          ) : (
            <>
              <Field
                label="Allowed Datasources"
                description="Select which datasources Zagalin can access. If none selected, all datasources are allowed."
              >
                <div className={s.datasourceList}>
                  {datasources.length === 0 ? (
                    <Alert title="No datasources found" severity="info">
                      No datasources are configured in your Grafana instance. Add datasources to enable query
                      governance.
                    </Alert>
                  ) : (
                    datasources.map((ds) => (
                      <div key={ds.uid} className={s.datasourceItem}>
                        <Switch
                          value={allowedDatasources.includes(ds.uid)}
                          onChange={() => handleDatasourceToggle(ds.uid)}
                        />
                        <div className={s.datasourceInfo}>
                          <span className={s.datasourceName}>{ds.name}</span>
                          <Badge text={ds.type} color="blue" />
                          {defaultDatasource === ds.uid && <Badge text="Default" color="green" icon="star" />}
                        </div>
                        {allowedDatasources.includes(ds.uid) && (
                          <Button
                            size="sm"
                            variant={defaultDatasource === ds.uid ? 'primary' : 'secondary'}
                            onClick={() => handleDefaultDatasourceChange(ds.uid)}
                            disabled={defaultDatasource === ds.uid}
                          >
                            {defaultDatasource === ds.uid ? 'Default' : 'Set as Default'}
                          </Button>
                        )}
                      </div>
                    ))
                  )}
                </div>
              </Field>

              {allowedDatasources.length > 0 && (
                <Alert title="Datasource Allowlist Active" severity="info">
                  <p>
                    Zagalin will only be able to query the {allowedDatasources.length} selected datasource(s). Users
                    will not be able to access data from other datasources through Zagalin.
                  </p>
                  {defaultDatasource && (
                    <p>The default datasource will be used when no specific datasource is requested.</p>
                  )}
                </Alert>
              )}
            </>
          )}
        </div>
      </div>

      {/* OTel Scope Governance */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>OpenTelemetry Scope Governance</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Enforce OpenTelemetry attributes on all queries for proper multi-tenant scoping and security.
          </p>

          <Field
            label="Enable OTel Scope Enforcement"
            description="Require service.name and deployment.environment.name labels on all queries"
          >
            <Switch
              value={otelEnabled}
              onChange={(e) => {
                setOtelEnabled(e.currentTarget.checked);
                setIsDirty(true);
              }}
            />
          </Field>

          {otelEnabled && (
            <>
              <Alert title="OTel Enforcement Active" severity="info">
                <p>
                  All queries will be validated and scoped with OpenTelemetry attributes. Queries without proper scoping
                  will be rejected or have default values applied.
                </p>
              </Alert>

              <InlineFieldRow>
                <InlineField label="Require service.name" labelWidth={30}>
                  <Switch
                    value={otelRequireService}
                    onChange={(e) => {
                      setOtelRequireService(e.currentTarget.checked);
                      setIsDirty(true);
                    }}
                  />
                </InlineField>
                <span className={s.skillDescription}>Mandate service.name label on all queries</span>
              </InlineFieldRow>

              <InlineFieldRow>
                <InlineField label="Require deployment.environment.name" labelWidth={30}>
                  <Switch
                    value={otelRequireEnvironment}
                    onChange={(e) => {
                      setOtelRequireEnvironment(e.currentTarget.checked);
                      setIsDirty(true);
                    }}
                  />
                </InlineField>
                <span className={s.skillDescription}>Mandate deployment.environment.name label on all queries</span>
              </InlineFieldRow>

              <Field
                label="Default Service Name"
                description="Fallback service name when not specified in query (leave empty to reject queries without service name)"
              >
                <Input
                  value={otelDefaultService}
                  onChange={(e) => {
                    setOtelDefaultService(e.currentTarget.value);
                    setIsDirty(true);
                  }}
                  placeholder="e.g., my-service"
                  width={50}
                />
              </Field>

              <Field
                label="Default Environment Name"
                description="Fallback environment when not specified in query (leave empty to reject queries without environment)"
              >
                <Input
                  value={otelDefaultEnvironment}
                  onChange={(e) => {
                    setOtelDefaultEnvironment(e.currentTarget.value);
                    setIsDirty(true);
                  }}
                  placeholder="e.g., production, staging, development"
                  width={50}
                />
              </Field>

              <Field
                label="Reject Queries Without Scope"
                description="Block queries that lack required attributes (even if defaults are configured). Recommended for strict governance."
              >
                <Switch
                  value={otelRejectIfNoScope}
                  onChange={(e) => {
                    setOtelRejectIfNoScope(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              {(otelDefaultService || otelDefaultEnvironment) && !otelRejectIfNoScope && (
                <Alert title="Fallback Mode Active" severity="warning">
                  <p>Queries without explicit scope will use default values. Fallback usage is logged for auditing.</p>
                  {otelDefaultService && (
                    <p>
                      Default service: <strong>{otelDefaultService}</strong>
                    </p>
                  )}
                  {otelDefaultEnvironment && (
                    <p>
                      Default environment: <strong>{otelDefaultEnvironment}</strong>
                    </p>
                  )}
                </Alert>
              )}
            </>
          )}
        </div>
      </div>

      {/* Query Injection Prevention */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Query Injection Prevention</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Validate and sanitize PromQL, LogQL, and TraceQL queries using official parsers to prevent injection
            attacks. Hybrid validation combines parser-based syntax checking with optional LLM semantic analysis.
          </p>

          <Field
            label="Enable Query Validation"
            description="Master switch for all query validation. Enable specific query types below."
          >
            <Switch
              value={queryValidationEnabled}
              onChange={(e) => {
                setQueryValidationEnabled(e.currentTarget.checked);
                setIsDirty(true);
              }}
            />
          </Field>

          {queryValidationEnabled && (
            <>
              <Alert title="Query Validation Active" severity="info">
                <p>
                  Master validation switch is ON. Enable specific query types below. Invalid queries will be rejected or
                  sanitized based on strict mode.
                </p>
              </Alert>

              <Field
                label="Enable PromQL Validation"
                description="Validate Prometheus queries (PromQL) for syntax errors, complexity, and injection attempts"
              >
                <Switch
                  value={queryValidationEnablePromQL}
                  onChange={(e) => {
                    setQueryValidationEnablePromQL(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              <Field
                label="Enable LogQL Validation"
                description="Validate Loki queries (LogQL) for syntax errors, complexity, and injection attempts"
              >
                <Switch
                  value={queryValidationEnableLogQL}
                  onChange={(e) => {
                    setQueryValidationEnableLogQL(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              <Field
                label="Enable TraceQL Validation"
                description="Validate Tempo queries (TraceQL) for syntax errors, complexity, and injection attempts"
              >
                <Switch
                  value={queryValidationEnableTraceQL}
                  onChange={(e) => {
                    setQueryValidationEnableTraceQL(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              <Field label="Strict Mode" description="Reject invalid queries instead of attempting to sanitize them">
                <Switch
                  value={queryValidationStrictMode}
                  onChange={(e) => {
                    setQueryValidationStrictMode(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              <Field
                label="Max Query Complexity"
                description="Maximum allowed complexity (AST node count). Prevents resource exhaustion attacks."
              >
                <Input
                  type="number"
                  value={queryValidationMaxComplexity}
                  onChange={(e) => {
                    setQueryValidationMaxComplexity(parseInt(e.currentTarget.value, 10) || 100);
                    setIsDirty(true);
                  }}
                  min={10}
                  max={1000}
                  width={20}
                />
              </Field>
            </>
          )}
        </div>
      </div>

      {/* Query Governance */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Query Governance</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Control query execution to prevent expensive operations and resource exhaustion. Set limits on query
            time ranges to balance data visibility with system performance.
          </p>

          <Field
            label="Max Query Time Range (hours)"
            description="Maximum time range allowed for queries. Set to 0 for unlimited. Prevents expensive long-range queries that could overwhelm your datasources."
          >
            <Input
              type="number"
              value={maxQueryTimeRangeHours}
              onChange={(e) => {
                setMaxQueryTimeRangeHours(parseInt(e.currentTarget.value, 10) || 24);
                setIsDirty(true);
              }}
              min={0}
              max={8760}
              width={20}
              placeholder="24"
            />
          </Field>

          {maxQueryTimeRangeHours === 0 && (
            <Alert title="Unlimited Time Range" severity="warning">
              <p>
                Time range clamping is disabled (0 = unlimited). Users can query any time range, which may cause
                performance issues with very large time windows.
              </p>
            </Alert>
          )}

          {maxQueryTimeRangeHours > 0 && maxQueryTimeRangeHours < 24 && (
            <Alert title="Short Time Range" severity="info">
              <p>
                Time range is limited to {maxQueryTimeRangeHours} hour{maxQueryTimeRangeHours !== 1 ? 's' : ''}. This provides
                strong protection but may limit some legitimate use cases (e.g., weekly reports).
              </p>
            </Alert>
          )}
        </div>
      </div>

      {/* Query Injection Prevention */}
      {queryValidationEnabled && (
        <div className={s.section}>
          <h3 className={s.sectionTitle}>Query Validation Options</h3>
          <div className={s.sectionContent}>
            <Field
              label="Log Validation Attempts"
                description="Audit log all validation failures and sanitizations for security monitoring"
              >
                <Switch
                  value={queryValidationLogAttempts}
                  onChange={(e) => {
                    setQueryValidationLogAttempts(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              <Field
                label={
                  <span>
                    Enable LLM Semantic Validation{' '}
                    <span style={{ color: '#ff9800', fontSize: '11px', fontWeight: 600, marginLeft: '8px' }}>
                      EXPERIMENTAL
                    </span>
                  </span>
                }
                description="Use AI to analyze queries for performance issues, best practices, and semantic concerns"
              >
                <Switch
                  value={queryValidationEnableLLM}
                  onChange={(e) => {
                    setQueryValidationEnableLLM(e.currentTarget.checked);
                    setIsDirty(true);
                  }}
                />
              </Field>

              {queryValidationEnableLLM && (
                <>
                  <Alert title="LLM Semantic Validation Enabled" severity="warning">
                    <p>
                      <strong>⚠️ Performance Impact:</strong> LLM validation adds 1-5 seconds of latency to every query execution.
                    </p>
                    <ul style={{ marginTop: '8px', marginBottom: '8px', paddingLeft: '20px' }}>
                      <li>
                        <strong>Timeout:</strong> Validation has a 5-second timeout; queries will proceed if timeout is exceeded
                      </li>
                      <li>
                        <strong>Cost:</strong> Each validation costs ~$0.00015 (using gpt-4o-mini)
                      </li>
                      <li>
                        <strong>Requirements:</strong> Service account token must be configured
                      </li>
                      <li>
                        <strong>Fail-open:</strong> On error, queries are allowed but warnings are logged
                      </li>
                    </ul>
                    <p style={{ marginTop: '12px' }}>
                      <strong>💡 Tip:</strong> Start with Advisory mode to understand the impact before enabling Strict mode.
                    </p>
                  </Alert>

                  {!hasServiceAccountToken && (
                    <Alert title="Service Account Token Required" severity="error">
                      <p>
                        LLM semantic validation requires a service account token. Configure one in the Authentication section
                        above.
                      </p>
                    </Alert>
                  )}

                  <Field
                    label="LLM Validation Mode"
                    description="Advisory: provides warnings but allows all queries. Strict: can block queries deemed unsafe."
                  >
                    <Combobox
                      options={[
                        { label: 'Advisory (warnings only, recommended)', value: 'advisory' },
                        { label: 'Strict (can block queries)', value: 'strict' },
                      ]}
                      value={queryValidationLLMMode}
                      onChange={(option) => {
                        setQueryValidationLLMMode(option.value as string);
                        setIsDirty(true);
                      }}
                      width={40}
                    />
                  </Field>

                  <Field
                    label="What LLM Validation Checks"
                    description="Semantic analysis goes beyond syntax to evaluate query quality"
                  >
                    <Alert title="Validation Scope" severity="info">
                      <ul style={{ marginBottom: 0, paddingLeft: '20px' }}>
                        <li>
                          <strong>Performance:</strong> Expensive operations, large time ranges, high cardinality
                        </li>
                        <li>
                          <strong>Best Practices:</strong> Suboptimal rate windows, inefficient aggregations
                        </li>
                        <li>
                          <strong>Security:</strong> Potential data exposure, overly broad selectors
                        </li>
                        <li>
                          <strong>Improvements:</strong> Suggestions for more efficient alternatives
                        </li>
                      </ul>
                    </Alert>
                  </Field>
                </>
              )}

              {!queryValidationStrictMode && (
                <Alert title="Sanitization Mode Active" severity="warning">
                  <p>
                    Invalid queries will be sanitized when possible. All sanitization attempts are logged for audit.
                  </p>
                  <p>
                    <strong>Warning:</strong> Sanitization may modify query semantics. Enable strict mode for production
                    environments.
                  </p>
                </Alert>
              )}

              {queryValidationEnableLLM && queryValidationLLMMode === 'strict' && (
                <Alert title="LLM Strict Mode Active" severity="warning">
                  <p>Queries deemed problematic by the LLM (e.g., too expensive, security concerns) will be blocked.</p>
                </Alert>
              )}
            </div>
          </div>
        )}

      <div className={s.section}>
        <h3 className={s.sectionTitle}>LLM Configuration</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Configure how Zagalin communicates and processes requests. Different modes use different settings for
            optimal performance.
          </p>

          <h4 style={{ marginTop: '24px', marginBottom: '16px', fontSize: '15px', fontWeight: 600 }}>
            Personality & Communication Style
          </h4>

          <Field label="Personality Preset" description="Choose how Zagalin communicates with you">
            <Combobox
              options={personalityOptions}
              value={config.personality}
              onChange={(option) => handlePersonalityChange(option.value as ZagalinConfig['personality'])}
              width={50}
            />
          </Field>

          <Field label="Custom Instructions" description="Additional instructions to customize Zagalin's behavior">
            <TextArea
              value={config.customInstructions}
              onChange={(e) => updateConfig({ customInstructions: e.currentTarget.value })}
              rows={10}
              placeholder="Enter custom instructions..."
              disabled={config.personality !== 'custom'}
              className={s.customInstructions}
            />
          </Field>

          <h4 style={{ marginTop: '32px', marginBottom: '16px', fontSize: '15px', fontWeight: 600 }}>
            Standard Mode Settings
          </h4>
          <p style={{ fontSize: '13px', color: '#999', marginBottom: '16px' }}>
            Fast responses for everyday questions and troubleshooting
          </p>

          <Field label="Temperature" description="Controls creativity vs. consistency (0.0 = factual, 1.0 = creative)">
            <div>
              <div className={s.sliderContainer}>
                <Slider
                  inputId="standard-temperature-slider"
                  min={0}
                  max={1}
                  step={0.1}
                  value={config.standardMode.temperature}
                  onChange={(value) =>
                    updateConfig({
                      standardMode: { ...config.standardMode, temperature: value },
                    })
                  }
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

          <Field label="Max Tokens" description="Maximum length of responses (higher = longer, more expensive)">
            <Combobox
              options={[
                { label: '1000 tokens (Short)', value: 1000 },
                { label: '2000 tokens (Medium)', value: 2000 },
                { label: '3000 tokens (Long)', value: 3000 },
                { label: '4000 tokens (Very Long)', value: 4000 },
              ]}
              value={config.standardMode.maxTokens}
              onChange={(option) =>
                updateConfig({
                  standardMode: { ...config.standardMode, maxTokens: option.value as number },
                })
              }
              width={50}
            />
          </Field>

          <h4 style={{ marginTop: '32px', marginBottom: '16px', fontSize: '15px', fontWeight: 600 }}>
            Design Mode Settings
          </h4>
          <p style={{ fontSize: '13px', color: '#999', marginBottom: '16px' }}>
            Dashboard design with examples, suggestions, and reference patterns
          </p>

          <Field label="Temperature" description="Higher temperature for creative design suggestions">
            <div>
              <div className={s.sliderContainer}>
                <Slider
                  inputId="design-temperature-slider"
                  min={0}
                  max={1}
                  step={0.1}
                  value={config.designMode.temperature}
                  onChange={(value) =>
                    updateConfig({
                      designMode: { ...config.designMode, temperature: value },
                    })
                  }
                />
                <span className={s.sliderValue}>{config.designMode.temperature.toFixed(1)}</span>
              </div>
              <div className={s.sliderLabels}>
                <span>Factual</span>
                <span>Balanced</span>
                <span>Creative</span>
              </div>
            </div>
          </Field>

          <Field label="Max Tokens" description="Longer responses for detailed design explanations">
            <Combobox
              options={[
                { label: '2000 tokens (Medium)', value: 2000 },
                { label: '3000 tokens (Long)', value: 3000 },
                { label: '4000 tokens (Very Long)', value: 4000 },
                { label: '5000 tokens (Maximum)', value: 5000 },
              ]}
              value={config.designMode.maxTokens}
              onChange={(option) =>
                updateConfig({
                  designMode: { ...config.designMode, maxTokens: option.value as number },
                })
              }
              width={50}
            />
          </Field>

          <Field label="Reference Dashboards" description="Dashboard UIDs to use as design examples (comma-separated)">
            <Input
              value={referenceDashboards.join(', ')}
              onChange={(e) => {
                const value = e.currentTarget.value;
                const uids = value
                  .split(',')
                  .map((uid) => uid.trim())
                  .filter((uid) => uid.length > 0);
                setReferenceDashboards(uids);
                setIsDirty(true);
              }}
              placeholder="dashboard-uid-1, dashboard-uid-2, dashboard-uid-3"
              width={60}
            />
          </Field>

          {referenceDashboards.length > 0 && (
            <Alert title={`${referenceDashboards.length} Reference Dashboard(s) Configured`} severity="info">
              <p>These dashboards will be fetched and cached on plugin startup to provide context for:</p>
              <ul style={{ marginTop: '8px', marginBottom: '8px', paddingLeft: '20px' }}>
                <li>Dashboard design assistance</li>
                <li>Panel layout suggestions</li>
                <li>Visualization best practices</li>
                <li>Naming conventions and patterns</li>
              </ul>
              <p style={{ marginTop: '8px' }}>
                <strong>Configured dashboards:</strong> {referenceDashboards.join(', ')}
              </p>
            </Alert>
          )}
        </div>
      </div>

      {/* Skills & Features */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>Skills & Features</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>Enable or disable specific assistant capabilities</p>

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
  datasourceList: css`
    display: flex;
    flex-direction: column;
    gap: ${theme.spacing(1.5)};
  `,
  datasourceItem: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(2)};
    padding: ${theme.spacing(1.5)};
    background: ${theme.colors.background.primary};
    border-radius: ${theme.shape.radius.default};
    border: 1px solid ${theme.colors.border.weak};
  `,
  datasourceInfo: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1)};
    flex: 1;
  `,
  datasourceName: css`
    font-weight: ${theme.typography.fontWeightMedium};
  `,
  llmBackendCards: css`
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: ${theme.spacing(2)};
    margin-bottom: ${theme.spacing(3)};
  `,
  llmBackendCard: css`
    padding: ${theme.spacing(2.5)};
    border: 2px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.primary};
    cursor: pointer;
    transition: all 0.2s ease-in-out;

    &:hover {
      border-color: ${theme.colors.primary.border};
      background: ${theme.colors.background.canvas};
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }
  `,
  llmBackendCardActive: css`
    border-color: ${theme.colors.primary.main};
    background: ${theme.colors.background.canvas};
    box-shadow: 0 0 0 2px ${theme.colors.primary.transparent};
  `,
  llmBackendCardHeader: css`
    display: flex;
    align-items: center;
    gap: ${theme.spacing(1.5)};
    margin-bottom: ${theme.spacing(1.5)};

    h4 {
      margin: 0;
      font-size: ${theme.typography.h5.fontSize};
      font-weight: ${theme.typography.fontWeightMedium};
      flex: 1;
    }
  `,
  llmBackendCardRadio: css`
    cursor: pointer;
    width: 18px;
    height: 18px;
    accent-color: ${theme.colors.primary.main};
  `,
  llmBackendCardDescription: css`
    margin: 0;
    color: ${theme.colors.text.secondary};
    font-size: ${theme.typography.bodySmall.fontSize};
    line-height: 1.5;
  `,
  providerCards: css`
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: ${theme.spacing(1.5)};
    margin-top: ${theme.spacing(1)};
  `,
  providerCard: css`
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: ${theme.spacing(1)};
    padding: ${theme.spacing(2)};
    border: 2px solid ${theme.colors.border.weak};
    border-radius: ${theme.shape.radius.default};
    background: ${theme.colors.background.primary};
    cursor: pointer;
    transition: all 0.2s ease-in-out;
    position: relative;

    &:hover {
      border-color: ${theme.colors.primary.border};
      background: ${theme.colors.background.canvas};
      transform: translateY(-2px);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    }
  `,
  providerCardActive: css`
    border-color: ${theme.colors.primary.main};
    background: ${theme.colors.background.canvas};
    box-shadow: 0 0 0 2px ${theme.colors.primary.transparent};
  `,
  providerCardRadio: css`
    position: absolute;
    top: ${theme.spacing(1)};
    right: ${theme.spacing(1)};
    cursor: pointer;
    width: 16px;
    height: 16px;
    accent-color: ${theme.colors.primary.main};
  `,
  providerCardLabel: css`
    font-weight: ${theme.typography.fontWeightMedium};
    font-size: ${theme.typography.body.fontSize};
    text-align: center;
  `,
});

export default AppConfig;
