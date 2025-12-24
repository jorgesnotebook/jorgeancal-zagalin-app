package plugin

import (
	"encoding/json"
	"fmt"
)

// PluginSettings contains the plugin configuration
// This is stored in Grafana's jsonData field (non-secure)
// All LLM provider configuration is handled by grafana-llm-app plugin
type PluginSettings struct {
	// Rate limits
	MaxRequestsPerMinute int     `json:"maxRequestsPerMinute"`
	MonthlyBudgetUSD     float64 `json:"monthlyBudgetUSD"`

	// Context settings
	ContextRefreshMinutes int `json:"contextRefreshMinutes"`
}

// Settings represents the plugin settings
type Settings struct {
	PluginSettings
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
	if s.MaxRequestsPerMinute == 0 {
		s.MaxRequestsPerMinute = 60
	}
	if s.MonthlyBudgetUSD == 0 {
		s.MonthlyBudgetUSD = 100.0
	}
	if s.ContextRefreshMinutes == 0 {
		s.ContextRefreshMinutes = 5
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

	return nil
}
