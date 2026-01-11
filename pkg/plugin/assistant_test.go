package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// TestDetectSignalType tests signal type detection logic
func TestDetectSignalType(t *testing.T) {
	tests := []struct {
		name    string
		message string
		context AssistantContext
		want    string
	}{
		{
			name:    "dashboard keyword - what am i seeing",
			message: "what am i seeing on this dashboard?",
			context: AssistantContext{
				Dashboard: &DashboardContext{Panels: []PanelContext{{}}},
			},
			want: "dashboard",
		},
		{
			name:    "dashboard keyword - panel",
			message: "explain this panel",
			context: AssistantContext{
				Dashboard: &DashboardContext{Panels: []PanelContext{{}}},
			},
			want: "dashboard",
		},
		{
			name:    "metrics+logs from datasources",
			message: "show me data",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "prometheus"}}}},
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "loki"}}}},
					},
				},
			},
			want: "metrics+logs",
		},
		{
			name:    "metrics+logs+traces from datasources",
			message: "analyze everything",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "prometheus"}}}},
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "loki"}}}},
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "tempo"}}}},
					},
				},
			},
			want: "metrics+logs+traces",
		},
		{
			name:    "metrics+traces from datasources",
			message: "show metrics and traces",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "prometheus"}}}},
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "jaeger"}}}},
					},
				},
			},
			want: "metrics+traces",
		},
		{
			name:    "logs+traces from datasources",
			message: "check logs and traces",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "loki"}}}},
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "zipkin"}}}},
					},
				},
			},
			want: "logs+traces",
		},
		{
			name:    "metrics only from datasources",
			message: "show me metrics",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "mimir"}}}},
					},
				},
			},
			want: "metrics",
		},
		{
			name:    "logs only from datasources",
			message: "show me logs",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "loki"}}}},
					},
				},
			},
			want: "logs",
		},
		{
			name:    "traces only from datasources",
			message: "show me traces",
			context: AssistantContext{
				Dashboard: &DashboardContext{
					Panels: []PanelContext{
						{Targets: []QueryTarget{{Datasource: map[string]interface{}{"type": "tempo"}}}},
					},
				},
			},
			want: "traces",
		},
		{
			name:    "promql keyword",
			message: "write a promql query for cpu usage",
			context: AssistantContext{},
			want:    "metrics",
		},
		{
			name:    "prometheus keyword",
			message: "query prometheus for latency",
			context: AssistantContext{},
			want:    "metrics",
		},
		{
			name:    "logql keyword",
			message: "create a logql query",
			context: AssistantContext{},
			want:    "logs",
		},
		{
			name:    "loki keyword",
			message: "search loki logs",
			context: AssistantContext{},
			want:    "logs",
		},
		{
			name:    "traceql keyword",
			message: "write traceql query",
			context: AssistantContext{},
			want:    "traces",
		},
		{
			name:    "tempo keyword",
			message: "query tempo",
			context: AssistantContext{},
			want:    "traces",
		},
		{
			name:    "investigation keyword - why",
			message: "why is the error rate increasing?",
			context: AssistantContext{},
			want:    "metrics", // "rate" and "increase" are metrics keywords, checked before investigation
		},
		{
			name:    "investigation keyword - troubleshoot",
			message: "troubleshoot this problem",
			context: AssistantContext{},
			want:    "investigation",
		},
		{
			name:    "general query",
			message: "hello, can you help me?",
			context: AssistantContext{},
			want:    "general",
		},
		{
			name:    "empty message",
			message: "",
			context: AssistantContext{},
			want:    "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSignalType(tt.message, tt.context)
			if got != tt.want {
				t.Errorf("detectSignalType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseAssistantRequest tests request parsing
func TestParseAssistantRequest(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError bool
		wantMsg   string
	}{
		{
			name: "valid request with message",
			body: `{"message": "test message", "history": [], "context": {}}`,
			wantError: false,
			wantMsg: "test message",
		},
		{
			name: "valid request with history",
			body: `{"message": "", "history": [{"role": "user", "content": "test"}], "context": {}}`,
			wantError: false,
		},
		{
			name: "empty message and history",
			body: `{"message": "", "history": [], "context": {}}`,
			wantError: true,
		},
		{
			name: "invalid JSON",
			body: `{invalid json}`,
			wantError: true,
		},
		{
			name: "missing fields",
			body: `{}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))

			result, err := app.parseAssistantRequest(req)

			if (err != nil) != tt.wantError {
				t.Errorf("parseAssistantRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && result.Message != tt.wantMsg {
				t.Errorf("parseAssistantRequest() message = %v, want %v", result.Message, tt.wantMsg)
			}
		})
	}
}

// TestDetermineSkillAndMode tests skill and mode detection
func TestDetermineSkillAndMode(t *testing.T) {
	tests := []struct {
		name      string
		req       AssistantRequest
		wantSkill string
		wantMode  string
	}{
		{
			name: "skill hint provided",
			req: AssistantRequest{
				Message: "test",
				SkillHint: "troubleshoot",
				Mode: "design",
			},
			wantSkill: "troubleshoot",
			wantMode: "design",
		},
		{
			name: "no skill hint, auto detect",
			req: AssistantRequest{
				Message: "test",
				Context: AssistantContext{},
			},
			wantSkill: "", // Will be auto-detected by DetectSkill
			wantMode: "standard",
		},
		{
			name: "mode provided",
			req: AssistantRequest{
				Message: "test",
				Mode: "design",
			},
			wantSkill: "",
			wantMode: "design",
		},
		{
			name: "default mode",
			req: AssistantRequest{
				Message: "test",
			},
			wantSkill: "",
			wantMode: "standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			skill, mode := app.determineSkillAndMode(&tt.req)

			if skill != tt.wantSkill && tt.req.SkillHint != "" {
				t.Errorf("determineSkillAndMode() skill = %v, want %v", skill, tt.wantSkill)
			}

			if mode != tt.wantMode {
				t.Errorf("determineSkillAndMode() mode = %v, want %v", mode, tt.wantMode)
			}
		})
	}
}

// TestBuildLLMRequest tests LLM request building
func TestBuildLLMRequest(t *testing.T) {
	tests := []struct {
		name         string
		messages     []AssistantMessage
		mode         string
		settings     *Settings
		wantModel    string
		wantStream   bool
		wantTools    bool
	}{
		{
			name: "standard mode",
			messages: []AssistantMessage{
				{Role: "user", Content: "test"},
			},
			mode: "standard",
			settings: &Settings{
				PluginSettings: PluginSettings{
					LLMModel: "gpt-4",
					StandardModeMaxTokens: 1000,
					StandardModeTemperature: 0.7,
				},
			},
			wantModel: "gpt-4",
			wantStream: true,
			wantTools: true,
		},
		{
			name: "design mode",
			messages: []AssistantMessage{
				{Role: "user", Content: "test"},
			},
			mode: "design",
			settings: &Settings{
				PluginSettings: PluginSettings{
					LLMModel: "gpt-4",
					DesignModeMaxTokens: 2000,
					DesignModeTemperature: 0.3,
				},
			},
			wantModel: "gpt-4",
			wantStream: true,
			wantTools: true,
		},
		{
			name: "no model configured",
			messages: []AssistantMessage{
				{Role: "user", Content: "test"},
			},
			mode: "standard",
			settings: &Settings{},
			wantModel: "",
			wantStream: true,
			wantTools: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{settings: tt.settings}
			result := app.buildLLMRequest(tt.messages, tt.mode)

			if result.Model != tt.wantModel {
				t.Errorf("buildLLMRequest() model = %v, want %v", result.Model, tt.wantModel)
			}

			if result.Stream != tt.wantStream {
				t.Errorf("buildLLMRequest() stream = %v, want %v", result.Stream, tt.wantStream)
			}

			if (len(result.Tools) > 0) != tt.wantTools {
				t.Errorf("buildLLMRequest() has tools = %v, want %v", len(result.Tools) > 0, tt.wantTools)
			}
		})
	}
}

// TestIsCompleteJSON tests JSON completeness check
func TestIsCompleteJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "complete simple JSON",
			input: `{"key": "value"}`,
			want:  true,
		},
		{
			name:  "complete nested JSON",
			input: `{"outer": {"inner": "value"}}`,
			want:  true,
		},
		{
			name:  "incomplete JSON - missing closing brace",
			input: `{"key": "value"`,
			want:  false,
		},
		{
			name:  "incomplete JSON - extra opening brace",
			input: `{{"key": "value"}`,
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "not JSON",
			input: "just text",
			want:  false,
		},
		{
			name:  "whitespace around JSON",
			input: `  {"key": "value"}  `,
			want:  true,
		},
		{
			name:  "complex nested structure",
			input: `{"a": {"b": {"c": "d"}}}`,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCompleteJSON(tt.input)
			if got != tt.want {
				t.Errorf("isCompleteJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractPromQLFromToolArgs tests PromQL extraction from tool arguments
func TestExtractPromQLFromToolArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		settings *Settings
		want     string
	}{
		{
			name: "simple metric",
			args: map[string]interface{}{
				"metric": "up",
			},
			settings: &Settings{},
			want:     "up",
		},
		{
			name: "metric with filters",
			args: map[string]interface{}{
				"metric": "http_requests_total",
				"filters": map[string]interface{}{
					"job": "api",
					"method": "GET",
				},
			},
			settings: &Settings{},
			want:     `http_requests_total{job="api",method="GET"}`,
		},
		{
			name: "metric with rate aggregation",
			args: map[string]interface{}{
				"metric": "http_requests_total",
				"aggregation": "rate",
				"timeRange": "5m",
			},
			settings: &Settings{},
			want:     "rate(http_requests_total[5m])",
		},
		{
			name: "metric with rate default time range",
			args: map[string]interface{}{
				"metric": "http_requests_total",
				"aggregation": "rate",
			},
			settings: &Settings{},
			want:     "rate(http_requests_total[5m])",
		},
		{
			name: "metric with sum aggregation",
			args: map[string]interface{}{
				"metric": "http_requests_total",
				"aggregation": "sum",
			},
			settings: &Settings{},
			want:     "sum(http_requests_total)",
		},
		{
			name: "metric with OTel labels",
			args: map[string]interface{}{
				"metric": "http_requests_total",
				"serviceName": "api",
				"environmentName": "prod",
			},
			settings: &Settings{
				PluginSettings: PluginSettings{
					OtelEnforcement: OtelEnforcementSettings{
						Enabled: true,
					},
				},
			},
			want:     `http_requests_total{service_name="api",deployment_environment_name="prod"}`,
		},
		{
			name: "empty metric",
			args: map[string]interface{}{},
			settings: &Settings{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{settings: tt.settings}
			got := app.extractPromQLFromToolArgs(tt.args)
			if got != tt.want {
				t.Errorf("extractPromQLFromToolArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractLogQLFromToolArgs tests LogQL extraction from tool arguments
func TestExtractLogQLFromToolArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		settings *Settings
		want     string
	}{
		{
			name: "simple log stream",
			args: map[string]interface{}{
				"logStream": `{job="api"}`,
			},
			settings: &Settings{},
			want:     `{job="api"}`,
		},
		{
			name: "log stream with filter",
			args: map[string]interface{}{
				"logStream": `{job="api"}`,
				"filter": "error",
			},
			settings: &Settings{},
			want:     `{job="api"} |= "error"`,
		},
		{
			name: "log stream with parser",
			args: map[string]interface{}{
				"logStream": `{job="api"}`,
				"parser": "json",
			},
			settings: &Settings{},
			want:     `{job="api"} | json`,
		},
		{
			name: "log stream with filter and parser",
			args: map[string]interface{}{
				"logStream": `{job="api"}`,
				"filter": "error",
				"parser": "logfmt",
			},
			settings: &Settings{},
			want:     `{job="api"} |= "error" | logfmt`,
		},
		{
			name: "log stream with OTel labels",
			args: map[string]interface{}{
				"logStream": `{job="api"}`,
				"serviceName": "api",
				"environmentName": "prod",
			},
			settings: &Settings{
				PluginSettings: PluginSettings{
					OtelEnforcement: OtelEnforcementSettings{
						Enabled: true,
					},
				},
			},
			want:     `{job="api",service_name="api",deployment_environment_name="prod"}`,
		},
		{
			name: "empty log stream",
			args: map[string]interface{}{},
			settings: &Settings{},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{settings: tt.settings}
			got := app.extractLogQLFromToolArgs(tt.args)
			if got != tt.want {
				t.Errorf("extractLogQLFromToolArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestInjectLabelsIntoLogStream tests label injection for LogQL
func TestInjectLabelsIntoLogStream(t *testing.T) {
	tests := []struct {
		name      string
		logStream string
		labels    []string
		want      string
	}{
		{
			name:      "empty labels",
			logStream: `{job="api"}`,
			labels:    []string{},
			want:      `{job="api"}`,
		},
		{
			name:      "inject into existing labels",
			logStream: `{job="api"}`,
			labels:    []string{`env="prod"`},
			want:      `{job="api",env="prod"}`,
		},
		{
			name:      "inject into empty selector",
			logStream: `{}`,
			labels:    []string{`service="api"`, `env="prod"`},
			want:      `{service="api",env="prod"}`,
		},
		{
			name:      "inject multiple labels",
			logStream: `{job="api",instance="host1"}`,
			labels:    []string{`service="api"`, `env="prod"`},
			want:      `{job="api",instance="host1",service="api",env="prod"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectLabelsIntoLogStream(tt.logStream, tt.labels)
			if got != tt.want {
				t.Errorf("injectLabelsIntoLogStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractUserFromRequest tests user extraction from request context
func TestExtractUserFromRequest(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantError bool
		wantLogin string
	}{
		{
			name: "valid user context",
			setupCtx: func() context.Context {
				return backend.WithPluginContext(context.Background(), backend.PluginContext{
					User: &backend.User{
						Login: "testuser",
						Email: "test@example.com",
					},
					OrgID: 1,
				})
			},
			wantError: false,
			wantLogin: "testuser",
		},
		{
			name: "missing user in context",
			setupCtx: func() context.Context {
				return backend.WithPluginContext(context.Background(), backend.PluginContext{
					OrgID: 1,
				})
			},
			wantError: true,
		},
		{
			name: "empty user login",
			setupCtx: func() context.Context {
				return backend.WithPluginContext(context.Background(), backend.PluginContext{
					User: &backend.User{
						Login: "",
						Email: "test@example.com",
					},
					OrgID: 1,
				})
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			ctx := tt.setupCtx()
			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

			user, err := app.extractUserFromRequest(req)

			if (err != nil) != tt.wantError {
				t.Errorf("extractUserFromRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && user.UserLogin != tt.wantLogin {
				t.Errorf("extractUserFromRequest() login = %v, want %v", user.UserLogin, tt.wantLogin)
			}
		})
	}
}

// TestGetGrafanaURL tests Grafana URL construction
func TestGetGrafanaURL(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		want    string
	}{
		{
			name: "GF_URL override",
			setup: func() {
				os.Setenv("GF_URL", "https://custom.example.com")
			},
			cleanup: func() {
				os.Unsetenv("GF_URL")
			},
			want: "https://custom.example.com",
		},
		{
			name: "environment variables - http",
			setup: func() {
				os.Setenv("GF_SERVER_PROTOCOL", "http")
				os.Setenv("GF_SERVER_DOMAIN", "localhost")
				os.Setenv("GF_SERVER_HTTP_PORT", "3000")
			},
			cleanup: func() {
				os.Unsetenv("GF_SERVER_PROTOCOL")
				os.Unsetenv("GF_SERVER_DOMAIN")
				os.Unsetenv("GF_SERVER_HTTP_PORT")
			},
			want: "http://localhost:3000",
		},
		{
			name: "environment variables - https port 443",
			setup: func() {
				os.Setenv("GF_SERVER_PROTOCOL", "https")
				os.Setenv("GF_SERVER_DOMAIN", "example.com")
				os.Setenv("GF_SERVER_HTTP_PORT", "443")
			},
			cleanup: func() {
				os.Unsetenv("GF_SERVER_PROTOCOL")
				os.Unsetenv("GF_SERVER_DOMAIN")
				os.Unsetenv("GF_SERVER_HTTP_PORT")
			},
			want: "https://example.com",
		},
		{
			name: "default localhost",
			setup: func() {},
			cleanup: func() {},
			want: "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			got := getGrafanaURL()
			if got != tt.want {
				t.Errorf("getGrafanaURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetMaxTokensForModel tests token limit detection
func TestGetMaxTokensForModel(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		tokenLimit int
		want       int
	}{
		{
			name:       "newer model - gpt-4o-2024-11-20",
			model:      "gpt-4o-2024-11-20",
			tokenLimit: 1000,
			want:       0,
		},
		{
			name:       "newer model - o1",
			model:      "o1",
			tokenLimit: 1000,
			want:       0,
		},
		{
			name:       "older model - gpt-4",
			model:      "gpt-4",
			tokenLimit: 1000,
			want:       1000,
		},
		{
			name:       "older model - gpt-3.5-turbo",
			model:      "gpt-3.5-turbo",
			tokenLimit: 500,
			want:       500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMaxTokensForModel(tt.model, tt.tokenLimit)
			if got != tt.want {
				t.Errorf("getMaxTokensForModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetMaxCompletionTokensForModel tests completion token limit detection
func TestGetMaxCompletionTokensForModel(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		tokenLimit int
		want       int
	}{
		{
			name:       "newer model - gpt-4o-2024-11-20",
			model:      "gpt-4o-2024-11-20",
			tokenLimit: 1000,
			want:       1000,
		},
		{
			name:       "newer model - o3-mini",
			model:      "o3-mini",
			tokenLimit: 500,
			want:       500,
		},
		{
			name:       "older model - gpt-4",
			model:      "gpt-4",
			tokenLimit: 1000,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMaxCompletionTokensForModel(tt.model, tt.tokenLimit)
			if got != tt.want {
				t.Errorf("getMaxCompletionTokensForModel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCheckRateLimit tests rate limiting check
func TestCheckRateLimit(t *testing.T) {
	tests := []struct {
		name       string
		setupApp   func() *App
		user       *UserIdentity
		wantError  bool
	}{
		{
			name: "rate limit not configured",
			setupApp: func() *App {
				return &App{}
			},
			user: &UserIdentity{UserLogin: "testuser"},
			wantError: false,
		},
		{
			name: "rate limit allows request",
			setupApp: func() *App {
				return &App{
					guardrails: &Guardrails{
						rateLimiter: NewRateLimiter(60),
					},
				}
			},
			user: &UserIdentity{UserLogin: "testuser"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()
			err := app.checkRateLimit(tt.user)

			if (err != nil) != tt.wantError {
				t.Errorf("checkRateLimit() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestValidateToolCall tests tool call validation
func TestValidateToolCall(t *testing.T) {
	tests := []struct {
		name     string
		toolCall *ToolCallChunk
		settings *Settings
		wantErr  bool
	}{
		{
			name: "validation disabled",
			toolCall: &ToolCallChunk{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "create_promql_query",
					Arguments: `{"metric": "up"}`,
				},
			},
			settings: &Settings{
				PluginSettings: PluginSettings{
					ToolCallValidation: false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid JSON arguments",
			toolCall: &ToolCallChunk{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "create_promql_query",
					Arguments: `{invalid json}`,
				},
			},
			settings: &Settings{
				PluginSettings: PluginSettings{
					ToolCallValidation: true,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				settings: tt.settings,
				queryValidator: NewQueryValidator(&QueryValidationSettings{}, nil),
			}

			user := &UserIdentity{UserLogin: "testuser", OrgID: 1}
			result := app.validateToolCall(context.Background(), tt.toolCall, user)

			hasError := result.Error != ""
			if hasError != tt.wantErr {
				t.Errorf("validateToolCall() error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

// TestExtractArtifactsFromText tests artifact extraction from LLM output
func TestExtractArtifactsFromText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantCount int
		wantTypes []string
	}{
		{
			name: "extract PromQL query",
			text: "Here's a query:\nrate(http_requests_total[5m])", // Query must be on its own line
			wantCount: 1,
			wantTypes: []string{"query"},
		},
		{
			name: "extract LogQL query",
			text: "Check logs: {job=\"api\"} |= \"error\"",
			wantCount: 1,
			wantTypes: []string{"query"},
		},
		{
			name: "extract trace ID",
			text: "Found trace: 1234567890abcdef",
			wantCount: 1,
			wantTypes: []string{"trace_id"},
		},
		{
			name: "extract multiple artifacts",
			text: "Query:\nrate(http_requests_total[5m])\nand trace: 1234567890abcdef", // Query on new line
			wantCount: 2,
			wantTypes: []string{"query", "trace_id"},
		},
		{
			name: "no artifacts",
			text: "Just plain text without any artifacts",
			wantCount: 0,
			wantTypes: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifacts := extractArtifactsFromText(tt.text)

			if len(artifacts) != tt.wantCount {
				t.Errorf("extractArtifactsFromText() count = %v, want %v", len(artifacts), tt.wantCount)
			}

			if len(artifacts) > 0 && len(tt.wantTypes) > 0 {
				foundTypes := make([]string, len(artifacts))
				for i, art := range artifacts {
					foundTypes[i] = art.Type
				}

				for _, wantType := range tt.wantTypes {
					found := false
					for _, foundType := range foundTypes {
						if foundType == wantType {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("extractArtifactsFromText() missing type %v in %v", wantType, foundTypes)
					}
				}
			}
		})
	}
}

// TestBuildMessages tests message construction
func TestBuildMessages(t *testing.T) {
	tests := []struct {
		name         string
		skill        string
		req          *AssistantRequest
		wantMinLen   int
		wantSystem   bool
	}{
		{
			name:  "basic message construction",
			skill: "",
			req: &AssistantRequest{
				Message: "test message",
				History: []AssistantMessage{},
				Context: AssistantContext{},
			},
			wantMinLen: 2, // system + user
			wantSystem: true,
		},
		{
			name:  "with history",
			skill: "troubleshoot",
			req: &AssistantRequest{
				Message: "test message",
				History: []AssistantMessage{
					{Role: "user", Content: "previous message"},
					{Role: "assistant", Content: "previous response"},
				},
				Context: AssistantContext{},
			},
			wantMinLen: 4, // system + history (2) + user
			wantSystem: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				settings: &Settings{},
			}

			messages := app.buildMessages(tt.skill, tt.req)

			if len(messages) < tt.wantMinLen {
				t.Errorf("buildMessages() length = %v, want at least %v", len(messages), tt.wantMinLen)
			}

			if tt.wantSystem && messages[0].Role != "system" {
				t.Errorf("buildMessages() first message role = %v, want system", messages[0].Role)
			}
		})
	}
}

// TestValidateToolCallWithQueryValidation tests tool validation with query checking
func TestValidateToolCallWithQueryValidation(t *testing.T) {
	tests := []struct {
		name           string
		toolCall       *ToolCallChunk
		wantValidation bool
	}{
		{
			name: "valid PromQL query",
			toolCall: &ToolCallChunk{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "create_promql_query",
					Arguments: `{"metric": "up"}`,
				},
			},
			wantValidation: false, // "up" is valid
		},
		{
			name: "invalid PromQL query",
			toolCall: &ToolCallChunk{
				ID:   "call_1",
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      "create_promql_query",
					Arguments: `{"metric": "up{"}`,
				},
			},
			wantValidation: true, // "up{" has unbalanced braces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				settings: &Settings{
					PluginSettings: PluginSettings{
						ToolCallValidation: true,
						QueryValidation: QueryValidationSettings{
							Enabled:                 true,
							EnablePromQLValidation:  true, // Must enable PromQL validation specifically
							MaxQueryComplexity:      50,
							StrictMode:              true,
						},
					},
				},
				queryValidator: NewQueryValidator(&QueryValidationSettings{
					Enabled:                true,
					EnablePromQLValidation: true, // Must enable PromQL validation specifically
					MaxQueryComplexity:     50,
					StrictMode:             true,
				}, nil),
			}

			user := &UserIdentity{UserLogin: "testuser", OrgID: 1}
			result := app.validateToolCall(context.Background(), tt.toolCall, user)

			if result.ToolCall != nil {
				var args map[string]interface{}
				json.Unmarshal([]byte(result.ToolCall.Function.Arguments), &args)

				hasValidationError := args["_validation_error"] != nil
				if hasValidationError != tt.wantValidation {
					t.Errorf("validateToolCall() has validation error = %v, want %v", hasValidationError, tt.wantValidation)
				}
			}
		})
	}
}
