package plugin

import (
	"encoding/json"
	"fmt"
)

type PluginSettings struct {
	LLMBackend string `json:"llmBackend"` 

	LLMProvider      string `json:"llmProvider"`      
	LLMModel         string `json:"llmModel"`         
	LLMEndpoint      string `json:"llmEndpoint"`      
	LLMOrganization  string `json:"llmOrganization"`  

	MaxRequestsPerMinute int     `json:"maxRequestsPerMinute"`
	MonthlyBudgetUSD     float64 `json:"monthlyBudgetUSD"`

	ContextRefreshMinutes int `json:"contextRefreshMinutes"`

	ReferenceDashboards []string `json:"referenceDashboards"`

	StandardModeMaxTokens   int     `json:"standardModeMaxTokens"`
	StandardModeTemperature float64 `json:"standardModeTemperature"`
	DesignModeMaxTokens     int     `json:"designModeMaxTokens"`
	DesignModeTemperature   float64 `json:"designModeTemperature"`

	AllowedDatasources []string `json:"allowedDatasources"`
	DefaultDatasource  string   `json:"defaultDatasource"`

	OtelEnforcement OtelEnforcementSettings `json:"otelEnforcement"`

	QueryValidation    QueryValidationSettings `json:"queryValidation"`
	ToolCallValidation bool                    `json:"toolCallValidation"`
}

type OtelEnforcementSettings struct {
	Enabled                   bool   `json:"enabled"`
	RequireServiceName        bool   `json:"requireServiceName"`
	RequireEnvironmentName    bool   `json:"requireEnvironmentName"`
	DefaultServiceName        string `json:"defaultServiceName"`
	DefaultEnvironmentName    string `json:"defaultEnvironmentName"`
	RejectIfNoScope           bool   `json:"rejectIfNoScope"`
}

type QueryValidationSettings struct {
	Enabled                 bool     `json:"enabled"`                 
	EnablePromQLValidation  bool     `json:"enablePromqlValidation"`  
	EnableLogQLValidation   bool     `json:"enableLogqlValidation"`   
	EnableTraceQLValidation bool     `json:"enableTraceqlValidation"` 
	StrictMode              bool     `json:"strictMode"`              
	MaxQueryComplexity      int      `json:"maxQueryComplexity"`      
	AllowedFunctions        []string `json:"allowedFunctions,omitempty"` 
	LogValidationAttempts   bool     `json:"logValidationAttempts"`

	EnableLLMValidation bool   `json:"enableLlmValidation"`
	LLMValidationMode   string `json:"llmValidationMode"` 
}

type Settings struct {
	PluginSettings
	LLMAPIKey              string
	ServiceAccountToken    string
	GrafanaURL             string
}

func LoadSettings(jsonData json.RawMessage, decryptedSecureJSONData map[string]string) (*Settings, error) {
	settings := &Settings{}

	if len(jsonData) > 0 {
		if err := json.Unmarshal(jsonData, &settings.PluginSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin settings: %w", err)
		}
	}

	if apiKey, ok := decryptedSecureJSONData["llmApiKey"]; ok {
		settings.LLMAPIKey = apiKey
	}
	if serviceAccountToken, ok := decryptedSecureJSONData["serviceAccountToken"]; ok {
		settings.ServiceAccountToken = serviceAccountToken
	}

	applyDefaults(&settings.PluginSettings)

	if err := settings.Validate(); err != nil {
		return nil, err
	}

	return settings, nil
}

func applyDefaults(s *PluginSettings) {
	if s.LLMBackend == "" {
		s.LLMBackend = "grafana-llm" 
	}
	if s.LLMProvider == "" {
		s.LLMProvider = "openai" 
	}
	if s.LLMModel == "" {
		s.LLMModel = "gpt-4o-mini" 
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

	if s.StandardModeMaxTokens == 0 {
		s.StandardModeMaxTokens = 2000
	}
	if s.StandardModeTemperature == 0 {
		s.StandardModeTemperature = 0.7
	}
	if s.DesignModeMaxTokens == 0 {
		s.DesignModeMaxTokens = 3000
	}
	if s.DesignModeTemperature == 0 {
		s.DesignModeTemperature = 0.8
	}

	if s.OtelEnforcement.Enabled {
		if !s.OtelEnforcement.RequireServiceName && !s.OtelEnforcement.RequireEnvironmentName {
			s.OtelEnforcement.RequireServiceName = true
			s.OtelEnforcement.RequireEnvironmentName = true
		}
	}

	if s.QueryValidation.Enabled {
		if s.QueryValidation.MaxQueryComplexity == 0 {
			s.QueryValidation.MaxQueryComplexity = 100
		}
		if !s.QueryValidation.LogValidationAttempts {
			s.QueryValidation.LogValidationAttempts = true
		}
		if s.QueryValidation.EnableLLMValidation && s.QueryValidation.LLMValidationMode == "" {
			s.QueryValidation.LLMValidationMode = "advisory"
		}
	}

	if s.QueryValidation.Enabled {
		s.ToolCallValidation = true
	}
}

func (s *Settings) Validate() error {
	if s.MaxRequestsPerMinute < 0 {
		return fmt.Errorf("max requests per minute must be >= 0")
	}
	if s.MonthlyBudgetUSD < 0 {
		return fmt.Errorf("monthly budget must be >= 0")
	}

	if s.DefaultDatasource != "" && len(s.AllowedDatasources) > 0 {
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
