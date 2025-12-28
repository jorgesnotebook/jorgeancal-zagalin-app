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
import { DatasourceService, type DatasourceInfo } from '../../services/datasourceService';

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
  const [datasources, setDatasources] = useState<DatasourceInfo[]>([]);
  const [loadingDatasources, setLoadingDatasources] = useState(true);
  const [allowedDatasources, setAllowedDatasources] = useState<string[]>(
    plugin.meta.jsonData?.allowedDatasources || []
  );
  const [defaultDatasource, setDefaultDatasource] = useState<string>(
    plugin.meta.jsonData?.defaultDatasource || ''
  );

  // OTel enforcement settings
  const [otelEnabled, setOtelEnabled] = useState<boolean>(
    plugin.meta.jsonData?.otelEnforcement?.enabled || false
  );
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

  // LLM Backend settings
  const [llmBackend, setLlmBackend] = useState<string>(
    plugin.meta.jsonData?.llmBackend || 'grafana-llm-app'
  );
  const [llmProvider, setLlmProvider] = useState<string>(
    plugin.meta.jsonData?.llmProvider || 'openai'
  );
  const [llmModel, setLlmModel] = useState<string>(
    plugin.meta.jsonData?.llmModel || 'gpt-4o-mini'
  );
  const [llmEndpoint, setLlmEndpoint] = useState<string>(
    plugin.meta.jsonData?.llmEndpoint || ''
  );
  const [llmApiKey, setLlmApiKey] = useState<string>(
    plugin.meta.secureJsonData?.llmApiKey || ''
  );
  const [serviceAccountToken, setServiceAccountToken] = useState<string>(
    plugin.meta.secureJsonData?.serviceAccountToken || ''
  );

  // Query Validation state
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

  // Check health on mount
  useEffect(() => {
    const loadHealth = async () => {
      setCheckingHealth(true);
      try {
        const status = await checkZagalinHealth();
        setHealthStatus(status);
      } catch (err) {
        console.error('Failed to check health:', err);
        // Set a default status if health check fails - never show internal errors
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

  // Load datasources on mount
  useEffect(() => {
    const loadDatasources = async () => {
      setLoadingDatasources(true);
      try {
        const response = await DatasourceService.listDatasources();
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
      // Save to Grafana's plugin settings (stored in database)
      const settings: any = {
        enabled: plugin.meta.enabled,
        pinned: plugin.meta.pinned,
        jsonData: {
          ...config,
          allowedDatasources,
          defaultDatasource,
          llmBackend,
          llmProvider,
          llmModel,
          llmEndpoint,
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
        },
      };

      // Add secure fields (API key, service account token) if provided
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
    setLlmApiKey('');
    setServiceAccountToken('');
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
    setIsDirty(true);
  };

  const handleDatasourceToggle = (uid: string) => {
    const newAllowed = allowedDatasources.includes(uid)
      ? allowedDatasources.filter(ds => ds !== uid)
      : [...allowedDatasources, uid];
    setAllowedDatasources(newAllowed);

    // If we removed the default datasource, clear it
    if (!newAllowed.includes(defaultDatasource)) {
      setDefaultDatasource('');
    }

    setIsDirty(true);
  };

  const handleDefaultDatasourceChange = (uid: string) => {
    setDefaultDatasource(uid);
    // Ensure default datasource is in allowed list
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

      {/* LLM Backend Configuration */}
      <div className={s.section}>
        <h3 className={s.sectionTitle}>LLM Backend Configuration</h3>
        <div className={s.sectionContent}>
          <p className={s.description}>
            Configure how Zagalin connects to LLM services. You can use grafana-llm-app for centralized configuration, bring your own API keys, or disable LLM features entirely.
          </p>

          {/* Backend Mode Selection with Visual Cards */}
          <div className={s.llmBackendCards}>
            {/* grafana-llm-app Card */}
            <div
              className={`${s.llmBackendCard} ${llmBackend === 'grafana-llm-app' ? s.llmBackendCardActive : ''}`}
              onClick={() => {
                setLlmBackend('grafana-llm-app');
                setIsDirty(true);
              }}
            >
              <div className={s.llmBackendCardHeader}>
                <input
                  type="radio"
                  checked={llmBackend === 'grafana-llm-app'}
                  onChange={() => {
                    setLlmBackend('grafana-llm-app');
                    setIsDirty(true);
                  }}
                  className={s.llmBackendCardRadio}
                />
                <Icon name="plug" size="xl" />
                <h4>grafana-llm-app</h4>
              </div>
              <p className={s.llmBackendCardDescription}>
                Use the grafana-llm-app plugin as a proxy. Centralize LLM configuration across all plugins. Supports OpenAI, Anthropic, Azure, and more.
              </p>
            </div>

            {/* Direct API Card - TEMPORARILY DISABLED */}
            {/* <div
              className={`${s.llmBackendCard} ${llmBackend === 'direct' ? s.llmBackendCardActive : ''}`}
              onClick={() => {
                setLlmBackend('direct');
                setIsDirty(true);
              }}
            >
              <div className={s.llmBackendCardHeader}>
                <input
                  type="radio"
                  checked={llmBackend === 'direct'}
                  onChange={() => {
                    setLlmBackend('direct');
                    setIsDirty(true);
                  }}
                  className={s.llmBackendCardRadio}
                />
                <Icon name="key-skeleton-alt" size="xl" />
                <h4>Direct API</h4>
              </div>
              <p className={s.llmBackendCardDescription}>
                Call LLM providers directly with your own API keys. Full control over provider, model, and endpoint configuration.
              </p>
            </div> */}

            {/* Disabled Card */}
            <div
              className={`${s.llmBackendCard} ${llmBackend === 'disabled' ? s.llmBackendCardActive : ''}`}
              onClick={() => {
                setLlmBackend('disabled');
                setIsDirty(true);
              }}
            >
              <div className={s.llmBackendCardHeader}>
                <input
                  type="radio"
                  checked={llmBackend === 'disabled'}
                  onChange={() => {
                    setLlmBackend('disabled');
                    setIsDirty(true);
                  }}
                  className={s.llmBackendCardRadio}
                />
                <Icon name="times" size="xl" />
                <h4>Disable LLM Features</h4>
              </div>
              <p className={s.llmBackendCardDescription}>
                Turn off all LLM-powered features. Zagalin will not make any LLM API calls.
              </p>
            </div>
          </div>

          {/* grafana-llm-app Mode Info */}
          {llmBackend === 'grafana-llm-app' && (
            <>
              <Alert title="Using grafana-llm-app Plugin" severity="info">
                <p>
                  Zagalin will use the grafana-llm-app plugin for LLM functionality. Make sure the plugin is installed and configured with your preferred provider.
                </p>
                <p>
                  <strong>Prerequisites:</strong>
                </p>
                <ol>
                  <li>Install grafana-llm-app plugin from Grafana catalog</li>
                  <li>Configure it with your LLM provider (Administration → Plugins → LLM App → Configuration)</li>
                  <li>(Optional) Provide a service account token below for backend-to-backend authentication</li>
                </ol>
              </Alert>

              <Field
                label="Service Account Token (Optional)"
                description="Grafana service account token for backend-to-backend authentication with grafana-llm-app. If not provided, Zagalin will try to use plugin context authentication. Stored securely in Grafana's encrypted storage."
              >
                <Input
                  type="password"
                  value={serviceAccountToken}
                  onChange={(e) => {
                    setServiceAccountToken(e.currentTarget.value);
                    setIsDirty(true);
                  }}
                  placeholder="glsa_..."
                  width={50}
                />
              </Field>

              {!serviceAccountToken && (
                <Alert title="Service Account Token" severity="info">
                  <p>
                    While optional, configuring a service account token is <strong>recommended</strong> for production use. It ensures reliable backend-to-backend authentication.
                  </p>
                  <p>
                    <strong>To create a service account token:</strong>
                  </p>
                  <ol>
                    <li>Go to Administration → Service Accounts</li>
                    <li>Create a new service account (e.g., &ldquo;Zagalin Plugin&rdquo;)</li>
                    <li>Assign the <code>Admin</code> or <code>Editor</code> role</li>
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
                      ? 'Your Azure OpenAI endpoint URL (e.g., https://YOUR-RESOURCE.openai.azure.com)'
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
                        ? 'https://YOUR-RESOURCE.openai.azure.com'
                        : 'https://api.example.com/v1/chat/completions'
                    }
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
                  <p>Get your API key from <a href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer">OpenAI Platform</a>.</p>
                  <p><strong>Recommended models:</strong> gpt-4o, gpt-4o-mini, gpt-4-turbo</p>
                </Alert>
              )}

              {llmProvider === 'anthropic' && (
                <Alert title="Anthropic Configuration" severity="info">
                  <p>Get your API key from <a href="https://console.anthropic.com/settings/keys" target="_blank" rel="noopener noreferrer">Anthropic Console</a>.</p>
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

          {/* Disabled Mode Info */}
          {llmBackend === 'disabled' && (
            <Alert title="LLM Features Disabled" severity="info">
              <p>
                All LLM-powered features are disabled. Zagalin will not make any API calls to LLM providers.
              </p>
              <p>
                You can re-enable LLM features at any time by selecting grafana-llm-app or Direct API mode above.
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
                      No datasources are configured in your Grafana instance. Add datasources to enable query governance.
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
                          {defaultDatasource === ds.uid && (
                            <Badge text="Default" color="green" icon="star" />
                          )}
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
                    Zagalin will only be able to query the {allowedDatasources.length} selected datasource(s).
                    Users will not be able to access data from other datasources through Zagalin.
                  </p>
                  {defaultDatasource && (
                    <p>
                      The default datasource will be used when no specific datasource is requested.
                    </p>
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
                  All queries will be validated and scoped with OpenTelemetry attributes. Queries without proper scoping will be rejected or have default values applied.
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
                  <p>
                    Queries without explicit scope will use default values. Fallback usage is logged for auditing.
                  </p>
                  {otelDefaultService && <p>Default service: <strong>{otelDefaultService}</strong></p>}
                  {otelDefaultEnvironment && <p>Default environment: <strong>{otelDefaultEnvironment}</strong></p>}
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
            Validate and sanitize PromQL, LogQL, and TraceQL queries using official parsers to prevent injection attacks.
            Hybrid validation combines parser-based syntax checking with optional LLM semantic analysis.
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
                  Master validation switch is ON. Enable specific query types below. Invalid queries will be rejected or sanitized based on strict mode.
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

              <Field
                label="Strict Mode"
                description="Reject invalid queries instead of attempting to sanitize them"
              >
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
                label="Enable LLM Semantic Validation"
                description="Use AI to check for expensive queries, best practices, and semantic issues"
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
                <Field
                  label="LLM Validation Mode"
                  description="Advisory: warnings only. Strict: can block problematic queries."
                >
                  <Combobox
                    options={[
                      { label: 'Advisory (warnings only)', value: 'advisory' },
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
              )}

              {!queryValidationStrictMode && (
                <Alert title="Sanitization Mode Active" severity="warning">
                  <p>
                    Invalid queries will be sanitized when possible. All sanitization attempts are logged for audit.
                  </p>
                  <p>
                    <strong>Warning:</strong> Sanitization may modify query semantics. Enable strict mode for production environments.
                  </p>
                </Alert>
              )}

              {queryValidationEnableLLM && queryValidationLLMMode === 'strict' && (
                <Alert title="LLM Strict Mode Active" severity="warning">
                  <p>
                    Queries deemed problematic by the LLM (e.g., too expensive, security concerns) will be blocked.
                  </p>
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
            label="Custom Instructions"
            description="Additional instructions to customize Zagalin's behavior"
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
