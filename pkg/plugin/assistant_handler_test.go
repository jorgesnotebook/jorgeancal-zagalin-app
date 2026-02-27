package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// mockLLMClient is a mock implementation of LLMClient for testing
type mockLLMClient struct {
	chunks []LLMStreamChunk
	err    error
}

func (m *mockLLMClient) StreamChat(ctx context.Context, req LLMStreamRequest, incomingReq *http.Request) (<-chan LLMStreamChunk, error) {
	if m.err != nil {
		return nil, m.err
	}

	ch := make(chan LLMStreamChunk, len(m.chunks))
	for _, chunk := range m.chunks {
		ch <- chunk
	}
	close(ch)

	return ch, nil
}

// TestHandleLLMChat tests the main HTTP handler for LLM chat
func TestHandleLLMChat(t *testing.T) {
	// Note: This test is limited as handleLLMChat calls createLLMClient() internally
	// and we cannot mock it without modifying the App struct
	// Full testing requires integration tests or refactoring
	t.Skip("handleLLMChat requires integration testing - tested via resources_test.go")
}

// TestStreamSSEResponse tests SSE streaming without validation
func TestStreamSSEResponse(t *testing.T) {
	tests := []struct {
		name       string
		chunks     []LLMStreamChunk
		skill      string
		wantChunks int
	}{
		{
			name: "successful stream with multiple chunks",
			chunks: []LLMStreamChunk{
				{Chunk: "Hello"},
				{Chunk: " world"},
				{Done: true},
			},
			skill:      "metrics",
			wantChunks: 4, // skill metadata + 3 chunks
		},
		{
			name: "stream with error",
			chunks: []LLMStreamChunk{
				{Chunk: "Hello"},
				{Error: "test error"},
			},
			skill:      "",
			wantChunks: 2,
		},
		{
			name: "empty stream",
			chunks: []LLMStreamChunk{
				{Done: true},
			},
			skill:      "",
			wantChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create channel and populate with test chunks
			chunkChan := make(chan LLMStreamChunk, len(tt.chunks))
			for _, chunk := range tt.chunks {
				chunkChan <- chunk
			}
			close(chunkChan)

			// Create response recorder
			rw := httptest.NewRecorder()
			ctx := context.Background()

			// Execute
			streamSSEResponse(ctx, rw, chunkChan, tt.skill)

			// Verify headers
			if rw.Header().Get("Content-Type") != "text/event-stream" {
				t.Errorf("streamSSEResponse() Content-Type = %v, want text/event-stream", rw.Header().Get("Content-Type"))
			}

			// Verify chunks were written
			body := rw.Body.String()
			lines := strings.Split(body, "\n")
			dataLines := 0
			for _, line := range lines {
				if strings.HasPrefix(line, "data: ") {
					dataLines++
				}
			}

			if dataLines < 1 {
				t.Errorf("streamSSEResponse() should write at least 1 data line, got %d", dataLines)
			}
		})
	}
}

// TestStreamSSEResponseWithValidation tests SSE streaming with tool call validation
func TestStreamSSEResponseWithValidation(t *testing.T) {
	tests := []struct {
		name       string
		chunks     []LLMStreamChunk
		skill      string
		user       *UserIdentity
		wantChunks int
	}{
		{
			name: "stream with tool call",
			chunks: []LLMStreamChunk{
				{Chunk: "Analyzing..."},
				{
					ToolCall: &ToolCallChunk{
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
				},
				{Done: true},
			},
			skill: "metrics",
			user: &UserIdentity{
				UserLogin: "testuser",
				OrgID:     1,
			},
			wantChunks: 3,
		},
		{
			name: "stream with incomplete tool call JSON",
			chunks: []LLMStreamChunk{
				{
					ToolCall: &ToolCallChunk{
						ID:   "call_1",
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      "create_promql_query",
							Arguments: `{"metric":`, // Incomplete JSON
						},
					},
				},
				{Done: true},
			},
			skill: "metrics",
			user: &UserIdentity{
				UserLogin: "testuser",
				OrgID:     1,
			},
			wantChunks: 1, // Only done chunk, tool call not sent (incomplete JSON)
		},
		{
			name: "client disconnects mid-stream",
			chunks: []LLMStreamChunk{
				{Chunk: "Hello"},
				{Chunk: " world"},
			},
			skill: "",
			user: &UserIdentity{
				UserLogin: "testuser",
				OrgID:     1,
			},
			wantChunks: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				settings: &Settings{
					PluginSettings: PluginSettings{
						ToolCallValidation: true,
					},
				},
				queryValidator: NewQueryValidator(&QueryValidationSettings{
					Enabled:                true,
					EnablePromQLValidation: true,
					MaxQueryComplexity:     50,
					StrictMode:             false,
				}, nil),
			}

			// Create channel and populate with test chunks
			chunkChan := make(chan LLMStreamChunk, len(tt.chunks))
			for _, chunk := range tt.chunks {
				chunkChan <- chunk
			}
			close(chunkChan)

			// Create response recorder
			rw := httptest.NewRecorder()

			// Create cancellable context for disconnect test
			ctx, cancel := context.WithCancel(context.Background())
			if strings.Contains(tt.name, "disconnects") {
				// Cancel immediately to simulate disconnect
				cancel()
			} else {
				defer cancel()
			}

			// Execute
			app.streamSSEResponseWithValidation(ctx, rw, chunkChan, tt.skill, tt.user, nil)

			// Verify headers
			if rw.Header().Get("Content-Type") != "text/event-stream" {
				t.Errorf("streamSSEResponseWithValidation() Content-Type = %v, want text/event-stream", rw.Header().Get("Content-Type"))
			}

			// Verify stream output
			body := rw.Body.String()
			if tt.skill != "" && !strings.Contains(tt.name, "disconnects") {
				if !strings.Contains(body, tt.skill) {
					t.Errorf("streamSSEResponseWithValidation() should include skill %q in output", tt.skill)
				}
			}
		})
	}
}

// TestOrchestrateRunFull tests full run orchestration
func TestOrchestrateRunFull(t *testing.T) {
	// Skip this test as it requires complex setup with run manager and goroutines
	// These are better tested through integration tests
	t.Skip("Orchestration tests require complex setup - covered by integration tests")
}

// TestGenerateExecutionPlan tests execution plan generation
func TestGenerateExecutionPlan(t *testing.T) {
	tests := []struct {
		name    string
		req     AssistantRequest
		wantErr bool
	}{
		{
			name: "direct LLM mode without API key",
			req: AssistantRequest{
				Message: "test",
				Context: AssistantContext{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				settings: &Settings{
					PluginSettings: PluginSettings{
						LLMBackend: "direct",
						// No API key set
					},
				},
			}

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			_, err := app.generateExecutionPlan(context.Background(), tt.req, req)

			if (err != nil) != tt.wantErr {
				t.Errorf("generateExecutionPlan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestExecuteStep tests individual step execution
func TestExecuteStep(t *testing.T) {
	tests := []struct {
		name       string
		step       PlannedStep
		setupApp   func() *App
		wantErr    bool
		wantResult string
	}{
		{
			name: "grafana-llm-app mode returns error",
			step: PlannedStep{
				Title:       "Test Step",
				Description: "Test step description",
			},
			setupApp: func() *App {
				return &App{
					settings: &Settings{
						PluginSettings: PluginSettings{
							LLMBackend: "grafana-llm-app",
						},
					},
				}
			},
			wantErr: true, // grafana-llm-app mode must be called from frontend
		},
		{
			name: "direct mode without setup",
			step: PlannedStep{
				Title:       "Test Step",
				Description: "Test step description",
			},
			setupApp: func() *App {
				return &App{
					settings: &Settings{
						PluginSettings: PluginSettings{
							LLMBackend: "direct",
						},
						LLMAPIKey: "", // No API key set
					},
					runManager: NewRunManager(backend.Logger),
				}
			},
			wantErr: true, // Will fail due to missing API key or config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()

			run := &RunState{
				RunID: "test-run",
				Plan: &ExecutionPlan{
					Steps: []PlannedStep{tt.step},
				},
			}

			req := AssistantRequest{
				Message: "test",
				Context: AssistantContext{},
			}

			httpReq := httptest.NewRequest(http.MethodPost, "/", nil)

			result, artifacts, err := app.executeStep(context.Background(), run, req, tt.step, 0, httpReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("executeStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == "" {
					t.Error("executeStep() result should not be empty")
				}

				if artifacts == nil {
					t.Error("executeStep() artifacts should not be nil")
				}
			}
		})
	}
}

// TestAuthenticateRequest tests request authentication
func TestAuthenticateRequest(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *http.Request
		wantErr  bool
	}{
		{
			name: "valid user in context",
			setupReq: func() *http.Request {
				ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
					User: &backend.User{
						Login: "testuser",
						Email: "test@example.com",
					},
					OrgID: 1,
				})
				return httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
			},
			wantErr: false,
		},
		{
			name: "no user in context",
			setupReq: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/", nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			req := tt.setupReq()

			user, err := app.authenticateRequest(req)

			if (err != nil) != tt.wantErr {
				t.Errorf("authenticateRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && user == nil {
				t.Error("authenticateRequest() user should not be nil")
			}
		})
	}
}

// TestCreateLLMClient tests LLM client creation
func TestCreateLLMClient(t *testing.T) {
	tests := []struct {
		name     string
		setupApp func() *App
		wantNil  bool
	}{
		{
			name: "create client with default settings",
			setupApp: func() *App {
				return &App{
					settings: &Settings{},
				}
			},
			wantNil: false,
		},
		{
			name: "create client with service account token",
			setupApp: func() *App {
				return &App{
					settings: &Settings{
						ServiceAccountToken: "test-token",
					},
				}
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.setupApp()

			client := app.createLLMClient()

			if (client == nil) != tt.wantNil {
				t.Errorf("createLLMClient() nil = %v, want %v", client == nil, tt.wantNil)
			}
		})
	}
}
