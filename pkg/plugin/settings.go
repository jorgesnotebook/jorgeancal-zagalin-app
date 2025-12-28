package plugin

import (
	"encoding/json"
	"fmt"
)

// PluginSettings contains the plugin configuration
// This is stored in Grafana's jsonData field (non-secure)
type PluginSettings struct {
	// LLM Backend selection
	LLMBackend string `json:"llmBackend"` // "grafana-llm-app" (default) | "direct"

	// Direct LLM provider configuration (only used when llmBackend = "direct")
	LLMProvider string `json:"llmProvider"` // "openai" | "anthropic" | "azure-openai"
	LLMModel    string `json:"llmModel"`    // e.g. "gpt-4o-mini", "claude-3-5-sonnet-20241022"
	LLMEndpoint string `json:"llmEndpoint"` // Optional custom endpoint URL

	// Rate limits
	MaxRequestsPerMinute int     `json:"maxRequestsPerMinute"`
	MonthlyBudgetUSD     float64 `json:"monthlyBudgetUSD"`

	// Context settings
	ContextRefreshMinutes int `json:"contextRefreshMinutes"`

	// Datasource governance
	AllowedDatasources []string `json:"allowedDatasources"`
	DefaultDatasource  string   `json:"defaultDatasource"`

	// OTel scope enforcement
	OtelEnforcement OtelEnforcementSettings `json:"otelEnforcement"`

	// Query validation settings
	QueryValidation QueryValidationSettings `json:"queryValidation"`
}

// OtelEnforcementSettings configures OpenTelemetry scope enforcement
type OtelEnforcementSettings struct {
	Enabled                   bool   `json:"enabled"`
	RequireServiceName        bool   `json:"requireServiceName"`
	RequireEnvironmentName    bool   `json:"requireEnvironmentName"`
	DefaultServiceName        string `json:"defaultServiceName"`
	DefaultEnvironmentName    string `json:"defaultEnvironmentName"`
	RejectIfNoScope           bool   `json:"rejectIfNoScope"`
}

// QueryValidationSettings configures query injection prevention
type QueryValidationSettings struct {
	Enabled                 bool     `json:"enabled"`                 // Master switch for all validation
	EnablePromQLValidation  bool     `json:"enablePromqlValidation"`  // Enable PromQL validation
	EnableLogQLValidation   bool     `json:"enableLogqlValidation"`   // Enable LogQL validation
	EnableTraceQLValidation bool     `json:"enableTraceqlValidation"` // Enable TraceQL validation
	StrictMode              bool     `json:"strictMode"`              // true = reject, false = sanitize
	MaxQueryComplexity      int      `json:"maxQueryComplexity"`      // Max AST nodes
	AllowedFunctions        []string `json:"allowedFunctions,omitempty"` // Optional function allowlist
	LogValidationAttempts   bool     `json:"logValidationAttempts"`

	// LLM semantic validation settings
	EnableLLMValidation bool   `json:"enableLlmValidation"`
	LLMValidationMode   string `json:"llmValidationMode"` // "advisory" or "strict"
}

// Settings represents the plugin settings
type Settings struct {
	PluginSettings
	// Secure settings (from decryptedSecureJSONData)
	LLMAPIKey string // LLM provider API key (only used when llmBackend = "direct")
}

// LoadSettings loads and validates settings from Grafana backend settings
func LoadSettings(jsonData json.RawMessage, decryptedSecureJSONData map[string]string) (*Settings, error) {
	settings := &Settings{}

	// Parse non-secure settings
	if len(jsonData) > 0 {
		if err := json.Unmarshal(jsonData, &settings.PluginSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin settings: %w", err)
		}
	}

	// Extract secure settings (LLM API key)
	if apiKey, ok := decryptedSecureJSONData["llmApiKey"]; ok {
		settings.LLMAPIKey = apiKey
	}

	// Apply defaults
	applyDefaults(&settings.PluginSettings)

	// Validate settings
	if err := settings.Validate(); err != nil {
		return nil, err
	}

	return settings, nil
}

// applyDefaults sets default values for settings
func applyDefaults(s *PluginSettings) {
	// LLM backend defaults
	if s.LLMBackend == "" {
		s.LLMBackend = "grafana-llm-app" // Default to grafana-llm-app for backwards compatibility
	}
	if s.LLMProvider == "" {
		s.LLMProvider = "openai" // Default provider for direct mode
	}
	if s.LLMModel == "" {
		s.LLMModel = "gpt-4o-mini" // Default model
	}

	if s.MaxRequestsPerMinute == 0 {
		s.MaxRequestsPerMinute = 60
	}
	if s.MonthlyBudgetUSD == 0 {
		s.MonthlyBudgetUSD = 100.0
	}
	if s.ContextRefreshMinutes == 0 {
		s.ContextRefreshMinutes = 5
	}

	// OTel enforcement defaults - disabled by default for backwards compatibility
	// When enabled, both service.name and deployment.environment.name are required
	if s.OtelEnforcement.Enabled {
		if !s.OtelEnforcement.RequireServiceName && !s.OtelEnforcement.RequireEnvironmentName {
			// If enabled but no specific requirements, require both
			s.OtelEnforcement.RequireServiceName = true
			s.OtelEnforcement.RequireEnvironmentName = true
		}
	}

	// Query validation defaults - disabled by default for backwards compatibility
	if s.QueryValidation.Enabled {
		if s.QueryValidation.MaxQueryComplexity == 0 {
			s.QueryValidation.MaxQueryComplexity = 100
		}
		// Default to logging validation attempts
		if !s.QueryValidation.LogValidationAttempts {
			s.QueryValidation.LogValidationAttempts = true
		}
		// Default LLM validation mode to advisory
		if s.QueryValidation.EnableLLMValidation && s.QueryValidation.LLMValidationMode == "" {
			s.QueryValidation.LLMValidationMode = "advisory"
		}
	}
}

// Validate validates the settings
func (s *Settings) Validate() error {
	// Validate rate limits
	if s.MaxRequestsPerMinute < 0 {
		return fmt.Errorf("max requests per minute must be >= 0")
	}
	if s.MonthlyBudgetUSD < 0 {
		return fmt.Errorf("monthly budget must be >= 0")
	}

	// Validate datasource allowlist
	if s.DefaultDatasource != "" && len(s.AllowedDatasources) > 0 {
		// Ensure default datasource is in the allowed list
		found := false
		for _, ds := range s.AllowedDatasources {
			if ds == s.DefaultDatasource {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("default datasource '%s' must be in allowed datasources list", s.DefaultDatasource)
		}
	}

	return nil
}
