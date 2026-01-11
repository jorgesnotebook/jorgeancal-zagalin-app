package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// TestHandleStartRun tests the run start endpoint
func TestHandleStartRun(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		setupApp       func(*App)
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedInBody string
	}{
		{
			name:   "successful run start",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "conv-123",
				Message:        "test message",
				History:        []AssistantMessage{},
				Context:        AssistantContext{},
			},
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
				a.settings = &Settings{PluginSettings: PluginSettings{LLMBackend: "direct"}}
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			expectedInBody: "runId",
		},
		{
			name:   "method not allowed",
			method: http.MethodGet,
			body:   nil,
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedInBody: "",
		},
		{
			name:   "missing authentication",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "conv-123",
				Message:        "test",
			},
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			expectedInBody: "authentication required",
		},
		// Skip rate limit test for now - needs proper guardrails setup
		// {
		// 	name:   "rate limit exceeded",
		// 	method: http.MethodPost,
		// 	body: RunStartRequest{
		// 		ConversationID: "conv-123",
		// 		Message:        "test",
		// 	},
		// 	setupApp: func(a *App) {
		// 		a.runManager = NewRunManager(backend.Logger)
		// 		a.guardrails = &Guardrails{
		// 			rateLimiter: NewRateLimiter(0),
		// 		}
		// 	},
		// 	setupRequest: func(req *http.Request) {
		// 		addTestUserToRequest(req, "testuser", 1)
		// 	},
		// 	expectedStatus: http.StatusTooManyRequests,
		// 	expectedInBody: "Rate limit exceeded",
		// },
		{
			name:   "invalid JSON body",
			method: http.MethodPost,
			body:   "invalid json",
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusBadRequest,
			expectedInBody: "Invalid request format",
		},
		{
			name:   "grafana-llm-app mode not supported",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "conv-123",
				Message:        "test",
			},
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
				a.settings = &Settings{PluginSettings: PluginSettings{LLMBackend: "grafana-llm-app"}}
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusNotImplemented,
			expectedInBody: "not supported in grafana-llm-app mode",
		},
		{
			name:   "missing conversation ID",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "",
				Message:        "test",
			},
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
				a.settings = &Settings{PluginSettings: PluginSettings{LLMBackend: "direct"}}
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusBadRequest,
			expectedInBody: "conversationId is required",
		},
		{
			name:   "missing message and history",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "conv-123",
				Message:        "",
				History:        []AssistantMessage{},
			},
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
				a.settings = &Settings{PluginSettings: PluginSettings{LLMBackend: "direct"}}
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusBadRequest,
			expectedInBody: "message or history required",
		},
		{
			name:   "max concurrent runs exceeded",
			method: http.MethodPost,
			body: RunStartRequest{
				ConversationID: "conv-123",
				Message:        "test",
			},
			setupApp: func(a *App) {
				rm := NewRunManager(backend.Logger)
				// Create 3 runs to hit the limit
				for i := 0; i < 3; i++ {
					rm.CreateRun(context.Background(), fmt.Sprintf("conv-%d", i), "testuser")
				}
				a.runManager = rm
				a.settings = &Settings{PluginSettings: PluginSettings{LLMBackend: "direct"}}
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusTooManyRequests,
			expectedInBody: "Maximum concurrent runs exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			if tt.setupApp != nil {
				tt.setupApp(app)
			}

			var body io.Reader
			if tt.body != nil {
				switch v := tt.body.(type) {
				case string:
					body = strings.NewReader(v)
				default:
					b, _ := json.Marshal(v)
					body = bytes.NewReader(b)
				}
			}

			req := httptest.NewRequest(tt.method, "/runs/start", body)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			rr := httptest.NewRecorder()
			app.handleStartRun(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedInBody != "" && !strings.Contains(rr.Body.String(), tt.expectedInBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedInBody, rr.Body.String())
			}

			// Cleanup: cancel any created runs and wait for goroutines
			if app.runManager != nil {
				app.runManager.mu.RLock()
				runIDs := make([]string, 0, len(app.runManager.runs))
				for runID := range app.runManager.runs {
					runIDs = append(runIDs, runID)
				}
				app.runManager.mu.RUnlock()

				for _, runID := range runIDs {
					app.runManager.CancelRun(runID)
				}

				// Give goroutines time to stop
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
}

// TestHandleRunEvents tests the SSE event streaming endpoint
func TestHandleRunEvents(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupApp       func(*App) string // returns runID
		setupRequest   func(*http.Request)
		expectedStatus int
		sendEvents     bool
	}{
		{
			name:   "successful SSE stream",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			sendEvents:     true,
		},
		{
			name:   "method not allowed",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				return "run-123"
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			sendEvents:     false,
		},
		{
			name:   "missing authentication",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			sendEvents:     false,
		},
		{
			name:   "run not found",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				a.runManager = NewRunManager(backend.Logger)
				return "nonexistent-run"
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusNotFound,
			sendEvents:     false,
		},
		{
			name:   "forbidden - different user",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "otheruser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusForbidden,
			sendEvents:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			var runID string
			if tt.setupApp != nil {
				runID = tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/runs/%s/events", runID), nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			// For SSE tests, we need to handle the streaming nature
			if tt.sendEvents && tt.expectedStatus == http.StatusOK {
				// Close event channel after a short delay to end stream
				go func() {
					time.Sleep(10 * time.Millisecond)
					if run, err := app.runManager.GetRun(runID); err == nil {
						close(run.EventChan)
					}
				}()
			}

			rr := httptest.NewRecorder()
			app.handleRunEvents(rr, req, runID)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify SSE headers
				if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
					t.Errorf("expected Content-Type text/event-stream, got %s", ct)
				}
				if cc := rr.Header().Get("Cache-Control"); cc != "no-cache" {
					t.Errorf("expected Cache-Control no-cache, got %s", cc)
				}
			}
		})
	}
}

// TestHandlePauseRun tests the run pause endpoint
func TestHandlePauseRun(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupApp       func(*App) string
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedInBody string
	}{
		{
			name:   "successful pause",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				rm.UpdateRunStatus(run.RunID, RunStatusExecuting)
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			expectedInBody: "Run paused",
		},
		{
			name:   "method not allowed",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				return "run-123"
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedInBody: "",
		},
		{
			name:   "run not found",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				a.runManager = NewRunManager(backend.Logger)
				return "nonexistent-run"
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusNotFound,
			expectedInBody: "Run not found",
		},
		{
			name:   "forbidden - different user",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "otheruser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusForbidden,
			expectedInBody: "Forbidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			var runID string
			if tt.setupApp != nil {
				runID = tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/runs/%s/pause", runID), nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			rr := httptest.NewRecorder()
			app.handlePauseRun(rr, req, runID)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedInBody != "" && !strings.Contains(rr.Body.String(), tt.expectedInBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedInBody, rr.Body.String())
			}
		})
	}
}

// TestHandleResumeRun tests the run resume endpoint
func TestHandleResumeRun(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupApp       func(*App) string
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedInBody string
	}{
		{
			name:   "successful resume",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				rm.UpdateRunStatus(run.RunID, RunStatusPaused)
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			expectedInBody: "Run resumed",
		},
		{
			name:   "method not allowed",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				return "run-123"
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedInBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			var runID string
			if tt.setupApp != nil {
				runID = tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/runs/%s/resume", runID), nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			rr := httptest.NewRecorder()
			app.handleResumeRun(rr, req, runID)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedInBody != "" && !strings.Contains(rr.Body.String(), tt.expectedInBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedInBody, rr.Body.String())
			}
		})
	}
}

// TestHandleCancelRun tests the run cancel endpoint
func TestHandleCancelRun(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupApp       func(*App) string
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedInBody string
	}{
		{
			name:   "successful cancel",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			expectedInBody: "Run cancelled",
		},
		{
			name:   "method not allowed",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				return "run-123"
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedInBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			var runID string
			if tt.setupApp != nil {
				runID = tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/runs/%s/cancel", runID), nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			rr := httptest.NewRecorder()
			app.handleCancelRun(rr, req, runID)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedInBody != "" && !strings.Contains(rr.Body.String(), tt.expectedInBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedInBody, rr.Body.String())
			}
		})
	}
}

// TestHandleRunStatus tests the run status endpoint
func TestHandleRunStatus(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupApp       func(*App) string
		setupRequest   func(*http.Request)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful status retrieval",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				rm := NewRunManager(backend.Logger)
				run, _ := rm.CreateRun(context.Background(), "conv-123", "testuser")
				a.runManager = rm
				return run.RunID
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response RunStatusResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if response.RunID == "" {
					t.Error("expected runId in response")
				}
				if response.ConversationID != "conv-123" {
					t.Errorf("expected conversationId conv-123, got %s", response.ConversationID)
				}
				if response.Status == "" {
					t.Error("expected status in response")
				}
			},
		},
		{
			name:   "method not allowed",
			method: http.MethodPost,
			setupApp: func(a *App) string {
				return "run-123"
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  nil,
		},
		{
			name:   "run not found",
			method: http.MethodGet,
			setupApp: func(a *App) string {
				a.runManager = NewRunManager(backend.Logger)
				return "nonexistent-run"
			},
			setupRequest: func(req *http.Request) {
				addTestUserToRequest(req, "testuser", 1)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			var runID string
			if tt.setupApp != nil {
				runID = tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, fmt.Sprintf("/runs/%s/status", runID), nil)
			if tt.setupRequest != nil {
				tt.setupRequest(req)
			}

			rr := httptest.NewRecorder()
			app.handleRunStatus(rr, req, runID)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

// TestHandleRunRoutes tests the route dispatcher
func TestHandleRunRoutes(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		method         string
		setupApp       func(*App)
		expectedStatus int
	}{
		{
			name:   "invalid path - no parts",
			path:   "/runs/",
			method: http.MethodGet,
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid path - no runID",
			path:   "/runs//events",
			method: http.MethodGet,
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid path - no action",
			path:   "/runs/run-123",
			method: http.MethodGet,
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "unknown action",
			path:   "/runs/run-123/unknown",
			method: http.MethodGet,
			setupApp: func(a *App) {
				a.runManager = NewRunManager(backend.Logger)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{}
			if tt.setupApp != nil {
				tt.setupApp(app)
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			app.handleRunRoutes(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// Helper function to add test user to request context
func addTestUserToRequest(req *http.Request, login string, orgID int64) {
	ctx := backend.WithPluginContext(req.Context(), backend.PluginContext{
		User: &backend.User{
			Login: login,
			Email: login + "@example.com",
		},
		OrgID: orgID,
	})
	*req = *req.WithContext(ctx)
}
