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

type RunStartRequest struct {
	ConversationID string             `json:"conversationId"`
	Message        string             `json:"message"`
	History        []AssistantMessage `json:"history"`
	Context        AssistantContext   `json:"context"`
	Attachments    []Attachment       `json:"attachments,omitempty"`
}

type Attachment struct {
	Type         string                 `json:"type"` 
	Source       string                 `json:"source"`
	DashboardUID string                 `json:"dashboardUid,omitempty"`
	PanelID      int                    `json:"panelId,omitempty"`
	TimeRange    *TimeRange             `json:"timeRange,omitempty"`
	Variables    map[string]interface{} `json:"variables,omitempty"`
	Links        map[string]string      `json:"links,omitempty"`
}


type RunStartResponse struct {
	RunID string `json:"runId"`
}

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

func (a *App) handleStartRun(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("Rate limit exceeded for run start", "user", user.UserLogin)
			sendErrorResponse(rw, "Rate limit exceeded", fmt.Errorf("too many requests"), http.StatusTooManyRequests)
			return
		}
	}

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

	if a.settings != nil && a.settings.LLMBackend == "grafana-llm-app" {
		sendErrorResponse(rw, "Run orchestration with planning/steps is not supported in grafana-llm-app mode. Please switch to Direct API mode in plugin configuration to use this feature.", fmt.Errorf("grafana-llm-app mode does not support run orchestration"), http.StatusNotImplemented)
		return
	}

	if runReq.ConversationID == "" {
		sendErrorResponse(rw, "conversationId is required", fmt.Errorf("empty conversationId"), http.StatusBadRequest)
		return
	}
	if runReq.Message == "" && len(runReq.History) == 0 {
		sendErrorResponse(rw, "message or history required", fmt.Errorf("empty request"), http.StatusBadRequest)
		return
	}

	if a.runManager.GetUserRunCount(user.UserLogin) >= 3 {
		backend.Logger.Warn("User run limit exceeded", "user", user.UserLogin)
		sendErrorResponse(rw, "Maximum concurrent runs exceeded", fmt.Errorf("too many active runs"), http.StatusTooManyRequests)
		return
	}

	run, err := a.runManager.CreateRun(req.Context(), runReq.ConversationID, user.UserLogin)
	if err != nil {
		sendErrorResponse(rw, "Failed to create run", err, http.StatusInternalServerError)
		return
	}

	go a.orchestrateRunFull(run.CancelCtx, run, AssistantRequest{
		Message: runReq.Message,
		History: runReq.History,
		Context: runReq.Context,
	}, req)

	response := RunStartResponse{
		RunID: run.RunID,
	}

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(response)

	backend.Logger.Info("Run started", "runId", run.RunID, "conversationId", runReq.ConversationID, "user", user.UserLogin)
}

func (a *App) handleRunEvents(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := req.Context()

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

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

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("X-Accel-Buffering", "no") 
	rw.WriteHeader(http.StatusOK)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		backend.Logger.Error("ResponseWriter does not support flushing")
		return
	}

	backend.Logger.Info("SSE stream started", "runId", runID, "user", user.UserLogin)

	for {
		select {
		case <-ctx.Done():
			backend.Logger.Debug("Client disconnected from SSE stream", "runId", runID)
			return

		case event, ok := <-run.EventChan:
			if !ok {
				fmt.Fprintf(rw, "data: [DONE]\n\n")
				flusher.Flush()
				backend.Logger.Info("SSE stream completed", "runId", runID)
				return
			}

			eventJSON, err := json.Marshal(event)
			if err != nil {
				backend.Logger.Error("Failed to marshal SSE event", "error", err, "runId", runID)
				continue
			}

			fmt.Fprintf(rw, "data: %s\n\n", string(eventJSON))
			flusher.Flush()
		}
	}
}

func (a *App) handlePauseRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

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

	if err := a.runManager.PauseRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to pause run", err, http.StatusBadRequest)
		return
	}

	EmitPaused(run.EventChan, runID)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run paused",
	})

	backend.Logger.Info("Run paused via API", "runId", runID, "user", user.UserLogin)
}

func (a *App) handleResumeRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

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

	if err := a.runManager.ResumeRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to resume run", err, http.StatusBadRequest)
		return
	}

	EmitResumed(run.EventChan, runID)

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run resumed",
	})

	backend.Logger.Info("Run resumed via API", "runId", runID, "user", user.UserLogin)
}

func (a *App) handleCancelRun(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodPost {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

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

	if err := a.runManager.CancelRun(runID); err != nil {
		sendErrorResponse(rw, "Failed to cancel run", err, http.StatusBadRequest)
		return
	}

	EmitCancelled(run.EventChan, runID, "User requested cancellation")

	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success": true,
		"message": "Run cancelled",
	})

	backend.Logger.Info("Run cancelled via API", "runId", runID, "user", user.UserLogin)
}

func (a *App) handleRunStatus(rw http.ResponseWriter, req *http.Request, runID string) {
	if req.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

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

func (a *App) handleRunRoutes(rw http.ResponseWriter, req *http.Request) {
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

	if len(parts) == 1 {
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

