package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// RunStartRequest represents a request to start a new run
type RunStartRequest struct {
	ConversationID string             `json:"conversationId"`
	Message        string             `json:"message"`
	History        []AssistantMessage `json:"history"`
	Context        AssistantContext   `json:"context"`
	Attachments    []Attachment       `json:"attachments,omitempty"`
}

// Attachment represents attached Grafana context
type Attachment struct {
	Type         string                 `json:"type"` // "grafana_context"
	Source       string                 `json:"source"`
	DashboardUID string                 `json:"dashboardUid,omitempty"`
	PanelID      int                    `json:"panelId,omitempty"`
	TimeRange    *TimeRange             `json:"timeRange,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	Links        map[string]string      `json:"links,omitempty"`
}

// TimeRange is defined in query_proxy.go

// RunStartResponse represents the response from starting a run
type RunStartResponse struct {
	RunID string `json:"runId"`
}

// RunStatusResponse represents the current status of a run
type RunStatusResponse struct {
	RunID            string          `json:"runId"`
	ConversationID   string          `json:"conversationId"`
	Status           string          `json:"status"`
	Plan             *ExecutionPlan  `json:"plan,omitempty"`
	CurrentStepIndex int             `json:"currentStepIndex"`
	Artifacts        []Artifact      `json:"artifacts"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
}

// handleStartRun handles POST /runs/start
func (a *App) handleStartRun(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Rate limiting
	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("Rate limit exceeded for run start", "user", user.UserLogin)
			sendErrorResponse(rw, "Rate limit exceeded", fmt.Errorf("too many requests"), http.StatusTooManyRequests)
			return
		}
	}

	// Parse request body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		sendErrorResponse(rw, "Failed to read request body", err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var runReq RunStartRequest
	if err := json.Unmarshal(body, &runReq); err != nil {
		sendErrorResponse(rw, "Invalid request format", err, http.StatusBadRequest)
		return
	}

	// Check if run orchestration is supported for the current backend mode
	if a.settings != nil && a.settings.LLMBackend == "grafana-llm-app" {
		sendErrorResponse(rw, "Run orchestration with planning/steps is not supported in grafana-llm-app mode. Please switch to Direct API mode in plugin configuration to use this feature.", fmt.Errorf("grafana-llm-app mode does not support run orchestration"), http.StatusNotImplemented)
		return
	}

	// Validate request
	if runReq.ConversationID == "" {
		sendErrorResponse(rw, "conversationId is required", fmt.Errorf("empty conversationId"), http.StatusBadRequest)
		return
	}
	if runReq.Message == "" && len(runReq.History) == 0 {
		sendErrorResponse(rw, "message or history required", fmt.Errorf("empty request"), http.StatusBadRequest)
		return
	}

	// Check user run limit (max 3 active runs per user)
	if a.runManager.GetUserRunCount(user.UserLogin) >= 3 {
		backend.Logger.Warn("User run limit exceeded", "user", user.UserLogin)
		sendErrorResponse(rw, "Maximum concurrent runs exceeded", fmt.Errorf("too many active runs"), http.StatusTooManyRequests)
		return
	}

	// Create run
	// Pass req.Context() to preserve plugin metadata (PluginContext, GrafanaConfig)
	// CreateRun will create an independent cancelable context that preserves this metadata
	run, err := a.runManager.CreateRun(req.Context(), runReq.ConversationID, user.UserLogin)
	if err != nil {
		sendErrorResponse(rw, "Failed to create run", err, http.StatusInternalServerError)
		return
	}

	// Start orchestration in goroutine
	// IMPORTANT: Use run.CancelCtx which contains plugin metadata from req.Context()
	// but is independent of the HTTP request lifecycle.
	// The HTTP request context gets canceled when the request completes,
	// but run.CancelCtx continues and allows proper cancellation via CancelRun().
	go a.orchestrateRunFull(run.CancelCtx, run, AssistantRequest{
		Message: runReq.Message,
		History: runReq.History,
		Context: runReq.Context,
	}, req)

	// Return run ID immediately
	response := RunStartResponse{
		RunID: run.RunID,
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(response)

	backend.Logger.Info("Run started", "runId", run.RunID, "conversationId", runReq.ConversationID, "user", user.UserLogin)
}

// handleRunEvents handles GET /runs/{runId}/events (SSE)
func (a *App) handleRunEvents(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Get run
	run, err := a.runManager.GetRun(runID)
	if err != nil {
		sendErrorResponse(rw, "Run not found", err, http.StatusNotFound)
		return
	}

	// Verify user owns this run
	run.mu.RLock()
	runOwner := run.UserLogin
	run.mu.RUnlock()

	if runOwner != user.UserLogin {
		sendErrorResponse(rw, "Forbidden", fmt.Errorf("not your run"), http.StatusForbidden)
		return
	}

	// Set SSE headers
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	rw.WriteHeader(http.StatusOK)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		backend.Logger.Error("ResponseWriter does not support flushing")
		return
	}

	backend.Logger.Info("SSE stream started", "runId", runID, "user", user.UserLogin)

	// Stream events from the run's event channel
	for {
		select {
		case <-ctx.Done():
			backend.Logger.Debug("Client disconnected from SSE stream", "runId", runID)
			return

		case event, ok := <-run.EventChan:
			if !ok {
				// Channel closed, send final marker and exit
				fmt.Fprintf(rw, "data: [DONE]\n\n")
				flusher.Flush()
				backend.Logger.Info("SSE stream completed", "runId", runID)
				return
			}

			// Serialize event to JSON
			eventJSON, err := json.Marshal(event)
			if err != nil {
				backend.Logger.Error("Failed to marshal SSE event", "error", err, "runId", runID)
				continue
			}

			// Write SSE event
			fmt.Fprintf(rw, "data: %s\n\n", string(eventJSON))
			flusher.Flush()
		}
	}
}

// handlePauseRun handles POST /runs/{runId}/pause
func (a *App) handlePauseRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Get run and verify ownership
	run, err := a.runManager.GetRun(runID)
	if err != nil {
		sendErrorResponse(rw, "Run not found", err, http.StatusNotFound)
		return
	}

	run.mu.RLock()
	runOwner := run.UserLogin
	run.mu.RUnlock()

	if runOwner != user.UserLogin {
		sendErrorResponse(rw, "Forbidden", fmt.Errorf("not your run"), http.StatusForbidden)
		return
	}

	// Pause run
	if err := a.runManager.PauseRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to pause run", err, http.StatusBadRequest)
		return
	}

	// Emit paused event
	EmitPaused(run.EventChan, runID)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run paused",
	})

	backend.Logger.Info("Run paused via API", "runId", runID, "user", user.UserLogin)
}

// handleResumeRun handles POST /runs/{runId}/resume
func (a *App) handleResumeRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Get run and verify ownership
	run, err := a.runManager.GetRun(runID)
	if err != nil {
		sendErrorResponse(rw, "Run not found", err, http.StatusNotFound)
		return
	}

	run.mu.RLock()
	runOwner := run.UserLogin
	run.mu.RUnlock()

	if runOwner != user.UserLogin {
		sendErrorResponse(rw, "Forbidden", fmt.Errorf("not your run"), http.StatusForbidden)
		return
	}

	// Resume run
	if err := a.runManager.ResumeRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to resume run", err, http.StatusBadRequest)
		return
	}

	// Emit resumed event
	EmitResumed(run.EventChan, runID)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run resumed",
	})

	backend.Logger.Info("Run resumed via API", "runId", runID, "user", user.UserLogin)
}

// handleCancelRun handles POST /runs/{runId}/cancel
func (a *App) handleCancelRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Get run and verify ownership
	run, err := a.runManager.GetRun(runID)
	if err != nil {
		sendErrorResponse(rw, "Run not found", err, http.StatusNotFound)
		return
	}

	run.mu.RLock()
	runOwner := run.UserLogin
	run.mu.RUnlock()

	if runOwner != user.UserLogin {
		sendErrorResponse(rw, "Forbidden", fmt.Errorf("not your run"), http.StatusForbidden)
		return
	}

	// Cancel run
	if err := a.runManager.CancelRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to cancel run", err, http.StatusBadRequest)
		return
	}

	// Emit cancelled event
	EmitCancelled(run.EventChan, runID, "User requested cancellation")

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run cancelled",
	})

	backend.Logger.Info("Run cancelled via API", "runId", runID, "user", user.UserLogin)
}

// handleRunStatus handles GET /runs/{runId}/status
func (a *App) handleRunStatus(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Get run and verify ownership
	run, err := a.runManager.GetRun(runID)
	if err != nil {
		sendErrorResponse(rw, "Run not found", err, http.StatusNotFound)
		return
	}

	run.mu.RLock()
	runOwner := run.UserLogin
	conversationID := run.ConversationID
	status := run.Status
	plan := run.Plan
	currentStepIndex := run.CurrentStepIndex
	artifacts := run.Artifacts
	createdAt := run.CreatedAt
	updatedAt := run.UpdatedAt
	run.mu.RUnlock()

	if runOwner != user.UserLogin {
		sendErrorResponse(rw, "Forbidden", fmt.Errorf("not your run"), http.StatusForbidden)
		return
	}

	// Build response
	response := RunStatusResponse{
		RunID:            runID,
		ConversationID:   conversationID,
		Status:           string(status),
		Plan:             plan,
		CurrentStepIndex: currentStepIndex,
		Artifacts:        artifacts,
		CreatedAt:        createdAt.Format(time.RFC3339),
		UpdatedAt:        updatedAt.Format(time.RFC3339),
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(response)
}

// handleRunRoutes routes run-specific endpoints based on path
func (a *App) handleRunRoutes(rw http.ResponseWriter, req *http.Request) {
	// Parse path: /runs/{runId}/{action}
	path := strings.TrimPrefix(req.URL.Path, "/runs/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		http.Error(rw, "invalid path", http.StatusBadRequest)
		return
	}

	runID := parts[0]
	if runID == "" {
		http.Error(rw, "runId required", http.StatusBadRequest)
		return
	}

	// No action means default (depends on method)
	if len(parts) == 1 {
		// No specific action, not supported for now
		http.Error(rw, "action required", http.StatusBadRequest)
		return
	}

	action := parts[1]

	switch action {
	case "events":
		a.handleRunEvents(rw, req, runID)
	case "pause":
		a.handlePauseRun(rw, req, runID)
	case "resume":
		a.handleResumeRun(rw, req, runID)
	case "cancel":
		a.handleCancelRun(rw, req, runID)
	case "status":
		a.handleRunStatus(rw, req, runID)
	default:
		http.Error(rw, "unknown action", http.StatusNotFound)
	}
}

// orchestrateRunFull is implemented in assistant.go
