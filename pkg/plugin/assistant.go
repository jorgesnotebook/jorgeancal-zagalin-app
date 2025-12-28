package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// AssistantRequest represents an incoming assistant chat request
type AssistantRequest struct {
	Message  string             `json:"message"`
	History  []AssistantMessage `json:"history"`
	Context  AssistantContext   `json:"context"`
	SkillHint string            `json:"skillHint,omitempty"`
}

// AssistantResponse represents the response metadata
type AssistantResponse struct {
	Message   string `json:"message"`
	SkillUsed string `json:"skillUsed"`
}

// detectSignalType determines what observability signal the user is asking about
// This is privacy-conscious - it only detects patterns, not the full query content
func detectSignalType(message string, context AssistantContext) string {
	messageLower := strings.ToLower(message)

	// Priority 1: Check dashboard context for datasource types
	if context.Dashboard != nil && len(context.Dashboard.Panels) > 0 {
		hasMetrics := false
		hasLogs := false
		hasTraces := false

		for _, panel := range context.Dashboard.Panels {
			for _, target := range panel.Targets {
				if target.Datasource != nil {
					// Datasource is map[string]interface{}, extract type field
					if dsTypeVal, ok := target.Datasource["type"]; ok {
						if dsTypeStr, ok := dsTypeVal.(string); ok {
							dsType := strings.ToLower(dsTypeStr)
							switch dsType {
							case "prometheus", "mimir", "cortex":
								hasMetrics = true
							case "loki":
								hasLogs = true
							case "tempo", "jaeger", "zipkin":
								hasTraces = true
							}
						}
					}
				}
			}
		}

		// If asking about dashboard itself, return "dashboard"
		if strings.Contains(messageLower, "dashboard") ||
			strings.Contains(messageLower, "panel") ||
			strings.Contains(messageLower, "what am i seeing") ||
			strings.Contains(messageLower, "what do i see") {
			return "dashboard"
		}

		// Return primary signal from dashboard
		if hasMetrics && hasLogs && hasTraces {
			return "metrics+logs+traces"
		} else if hasMetrics && hasLogs {
			return "metrics+logs"
		} else if hasMetrics && hasTraces {
			return "metrics+traces"
		} else if hasLogs && hasTraces {
			return "logs+traces"
		} else if hasMetrics {
			return "metrics"
		} else if hasLogs {
			return "logs"
		} else if hasTraces {
			return "traces"
		}
	}

	// Priority 2: Detect from keywords in message
	// Metrics keywords
	metricsKeywords := []string{
		"promql", "prometheus", "metric", "rate", "increase", "counter", "gauge",
		"histogram", "summary", "cpu", "memory", "latency", "throughput",
		"requests per second", "rps", "qps",
	}
	for _, keyword := range metricsKeywords {
		if strings.Contains(messageLower, keyword) {
			return "metrics"
		}
	}

	// Logs keywords
	logsKeywords := []string{
		"logql", "loki", "log", "error message", "stack trace",
		"log line", "log stream", "log entry",
	}
	for _, keyword := range logsKeywords {
		if strings.Contains(messageLower, keyword) {
			return "logs"
		}
	}

	// Traces keywords
	tracesKeywords := []string{
		"traceql", "tempo", "trace", "span", "jaeger", "zipkin",
		"trace id", "traceid", "distributed tracing",
	}
	for _, keyword := range tracesKeywords {
		if strings.Contains(messageLower, keyword) {
			return "traces"
		}
	}

	// Priority 3: Investigation/troubleshooting keywords (usually multi-signal)
	investigationKeywords := []string{
		"why", "investigate", "troubleshoot", "debug", "root cause",
		"error", "spike", "increase", "decrease", "problem",
	}
	for _, keyword := range investigationKeywords {
		if strings.Contains(messageLower, keyword) {
			return "investigation"
		}
	}

	// Default: general chat
	return "general"
}

// handleLLMChat handles POST /llm/chat requests
func (a *App) handleLLMChat(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// Extract user from request context (set by Grafana)
	user, err := a.extractUserFromRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Rate limiting check
	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("Rate limit exceeded", "user", user.UserLogin)
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

	var assistantReq AssistantRequest
	if err := json.Unmarshal(body, &assistantReq); err != nil {
		sendErrorResponse(rw, "Invalid request format", err, http.StatusBadRequest)
		return
	}

	// Validate request
	if assistantReq.Message == "" && len(assistantReq.History) == 0 {
		sendErrorResponse(rw, "Message or history required", fmt.Errorf("empty request"), http.StatusBadRequest)
		return
	}

	// Detect skill (use hint if provided, otherwise auto-detect)
	skill := assistantReq.SkillHint
	if skill == "" {
		skill = DetectSkill(assistantReq.Message, assistantReq.Context)
	}

	// Build prompts
	systemPrompt := BuildSystemPrompt(skill, assistantReq.Context)
	userPrompt := BuildUserPrompt(skill, assistantReq.Message, assistantReq.Context)

	// Construct full message history
	messages := []AssistantMessage{}

	// Add system prompt
	messages = append(messages, AssistantMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add conversation history
	messages = append(messages, assistantReq.History...)

	// Add current user message (if skill didn't override it)
	if skill == "" || skill == "generate_query" || skill == "troubleshoot" {
		// For these skills, use the built user prompt
		messages = append(messages, AssistantMessage{
			Role:    "user",
			Content: userPrompt,
		})
	} else {
		// For other skills, append user message then context
		if assistantReq.Message != "" {
			messages = append(messages, AssistantMessage{
				Role:    "user",
				Content: assistantReq.Message,
			})
		}
		if userPrompt != "" && userPrompt != assistantReq.Message {
			messages = append(messages, AssistantMessage{
				Role:    "user",
				Content: userPrompt,
			})
		}
	}

	// Prepare LLM request
	// LLM configuration is handled by grafana-llm-app plugin, so we use sensible defaults here
	llmReq := LLMStreamRequest{
		Model:       "gpt-4o-mini", // Default model
		Messages:    messages,
		Temperature: 0.7,    // Default temperature
		MaxTokens:   2000,   // Default max tokens
		Tools:       GetTools(true), // Function calling enabled by default
		ToolChoice:  "auto",
	}

	// Create LLM client based on settings
	var chunkChan <-chan LLMStreamChunk

	// Check if LLM features are disabled
	if a.settings.LLMBackend == "disabled" {
		backend.Logger.Info("LLM features disabled", "user", user.UserLogin)
		sendErrorResponse(rw, "LLM features are disabled", fmt.Errorf("LLM backend is set to disabled mode"), http.StatusServiceUnavailable)
		return
	}

	if a.settings.LLMBackend == "direct" {
		// Direct mode
		if a.settings.LLMAPIKey == "" {
			sendErrorResponse(rw, "Direct LLM mode requires an API key", fmt.Errorf("no API key configured"), http.StatusBadRequest)
			return
		}

		directClient := NewDirectLLMClient(
			a.settings.LLMProvider,
			a.settings.LLMModel,
			a.settings.LLMEndpoint,
			a.settings.LLMAPIKey,
			http.DefaultClient,
			backend.Logger,
		)

		chunkChan, err = directClient.StreamChat(ctx, llmReq)
		if err != nil {
			backend.Logger.Error("Failed to call direct LLM", "error", err, "user", user.UserLogin)
			sendErrorResponse(rw, "Failed to call LLM", err, http.StatusInternalServerError)
			return
		}
	} else {
		// grafana-llm-app mode (default) - frontend should call grafana-llm-app directly
		backend.Logger.Warn("grafana-llm-app mode called from backend. Frontend should call grafana-llm-app directly.", "user", user.UserLogin)
		sendErrorResponse(rw, "grafana-llm-app mode should be called from frontend", fmt.Errorf("use frontend to call grafana-llm-app directly"), http.StatusBadRequest)
		return
	}

	// Detect signal type for usage tracking (privacy-conscious - no message content logged)
	signalType := detectSignalType(assistantReq.Message, assistantReq.Context)

	// Audit log - tracks usage without exposing query content
	backend.Logger.Info("LLM chat request",
		"user", user.UserLogin,
		"orgId", user.OrgID,
		"skill", skill,
		"signalType", signalType, // metrics, logs, traces, dashboard, general
		"hasDashboardContext", assistantReq.Context.Dashboard != nil,
		"messageLength", len(assistantReq.Message),
		"historyLength", len(assistantReq.History),
	)

	// Stream response as SSE
	streamSSEResponse(ctx, rw, chunkChan, skill)
}

// streamSSEResponse streams LLM chunks to the client as Server-Sent Events
func streamSSEResponse(ctx context.Context, rw http.ResponseWriter, chunkChan <-chan LLMStreamChunk, skill string) {
	// Set SSE headers
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.WriteHeader(http.StatusOK)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		backend.Logger.Error("ResponseWriter does not support flushing")
		return
	}

	// Send initial metadata (optional)
	if skill != "" {
		metadataJSON, _ := json.Marshal(map[string]string{"skill": skill})
		fmt.Fprintf(rw, "data: %s\n\n", string(metadataJSON))
		flusher.Flush()
	}

	// Stream chunks
	for {
		select {
		case <-ctx.Done():
			backend.Logger.Debug("Client disconnected")
			return

		case chunk, ok := <-chunkChan:
			if !ok {
				// Channel closed - send done event
				fmt.Fprintf(rw, "data: {\"done\":true}\n\n")
				flusher.Flush()
				return
			}

			// Send chunk as SSE
			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				backend.Logger.Error("Failed to marshal chunk", "error", err)
				continue
			}

			fmt.Fprintf(rw, "data: %s\n\n", string(chunkJSON))
			flusher.Flush()

			// Check if done or error
			if chunk.Done || chunk.Error != "" {
				return
			}
		}
	}
}

// extractUserFromRequest extracts user info from Grafana request context
func (a *App) extractUserFromRequest(req *http.Request) (*UserInfo, error) {
	// Get plugin context from request (Grafana SDK standard approach)
	pluginContext := backend.PluginConfigFromContext(req.Context())

	if pluginContext.User == nil {
		return nil, fmt.Errorf("user not found in context")
	}

	user := pluginContext.User

	if user.Login == "" {
		return nil, fmt.Errorf("user login is empty")
	}

	return &UserInfo{
		UserLogin: user.Login,
		OrgID:     pluginContext.OrgID,
	}, nil
}


// UserInfo represents authenticated user information
type UserInfo struct {
	UserLogin string
	OrgID     int64
}

// getGrafanaURL returns the Grafana URL for making internal API calls
func getGrafanaURL() string {
	// Get Grafana URL from environment or default to localhost
	url := "http://localhost:3000"
	// In production, this would be set by Grafana environment
	// For now, we assume local development
	return url
}

// orchestrateRunFull implements the complete run orchestration with planning and execution
func (a *App) orchestrateRunFull(ctx context.Context, run *RunState, req AssistantRequest, incomingReq *http.Request) {
	startTime := time.Now()

	defer func() {
		// Always close event channel when done
		a.runManager.CloseEventChannel(run.RunID)

		// Schedule cleanup after 1 hour
		go func() {
			time.Sleep(1 * time.Hour)
			a.runManager.CleanupRun(run.RunID)
		}()
	}()

	// Emit run_started
	EmitRunStarted(run.EventChan, run.RunID, run.ConversationID)

	// Skip health check - if LLM call fails, we'll get a clear error message
	// The health check was causing false positives and context cancellation issues
	backend.Logger.Debug("Skipping health check, proceeding directly to LLM call", "runId", run.RunID)

	// PHASE 1: Planning
	a.runManager.UpdateRunStatus(run.RunID, RunStatusPlanning)

	plan, err := a.generateExecutionPlan(ctx, req, incomingReq)
	if err != nil {
		backend.Logger.Error("Failed to generate execution plan", "error", err, "runId", run.RunID)

		// Provide more helpful error message
		errorMsg := fmt.Sprintf("Planning failed: %s\n\nIf this error persists, please check:\n1. grafana-llm-app plugin configuration\n2. LLM provider API credentials\n3. Network connectivity", err.Error())
		EmitError(run.EventChan, run.RunID, errorMsg, -1, false)

		a.runManager.UpdateRunStatus(run.RunID, RunStatusFailed)
		EmitFinal(run.EventChan, run.RunID, "failed", 0, 0, 0, 0, time.Since(startTime).String())
		return
	}

	// Store plan in run state
	a.runManager.SetPlan(run.RunID, plan)

	// Emit plan event
	EmitPlan(run.EventChan, run.RunID, plan)

	backend.Logger.Info("Execution plan generated", "runId", run.RunID, "steps", len(plan.Steps))

	// PHASE 2: Execution
	a.runManager.UpdateRunStatus(run.RunID, RunStatusExecuting)

	completedSteps := 0
	failedSteps := 0

	for i, step := range plan.Steps {
		// Check pause or cancel
		shouldContinue, wasPaused := a.runManager.CheckPauseOrCancel(run.RunID)
		if !shouldContinue {
			backend.Logger.Info("Run cancelled during execution", "runId", run.RunID, "stepIndex", i)
			EmitCancelled(run.EventChan, run.RunID, "Cancelled by user or timeout")
			EmitFinal(run.EventChan, run.RunID, "cancelled", len(plan.Steps), completedSteps, failedSteps, len(run.Artifacts), time.Since(startTime).String())
			return
		}
		if wasPaused {
			EmitResumed(run.EventChan, run.RunID)
		}

		// Update step status to in_progress
		a.runManager.UpdateStepStatus(run.RunID, i, "in_progress")

		// Emit step_started
		EmitStepStarted(run.EventChan, run.RunID, i, step.Title, step.Description)

		// Execute step
		stepResult, artifacts, stepErr := a.executeStep(ctx, run, req, step, i, incomingReq)

		// Add artifacts to run state and emit events
		for _, artifact := range artifacts {
			a.runManager.AddArtifact(run.RunID, artifact)
			EmitArtifact(run.EventChan, run.RunID, artifact, i)
		}

		// Determine step status
		stepStatus := "completed"
		if stepErr != nil {
			stepStatus = "failed"
			failedSteps++
			EmitError(run.EventChan, run.RunID, stepErr.Error(), i, false)
		} else {
			completedSteps++
		}

		// Update step status
		a.runManager.UpdateStepStatus(run.RunID, i, stepStatus)

		// Emit step_done
		errorMsg := ""
		if stepErr != nil {
			errorMsg = stepErr.Error()
		}
		EmitStepDone(run.EventChan, run.RunID, i, step.Title, stepStatus, stepResult, errorMsg)

		backend.Logger.Info("Step completed", "runId", run.RunID, "stepIndex", i, "status", stepStatus)
	}

	// PHASE 3: Finalization
	// Build final assistant message
	finalMessage := a.buildFinalMessage(plan, run.Artifacts)

	messageID := fmt.Sprintf("msg_%s", uuid.New().String())
	EmitAssistantMessage(run.EventChan, run.RunID, messageID, "assistant", finalMessage)

	// Determine final status
	finalStatus := "completed"
	if failedSteps == len(plan.Steps) {
		finalStatus = "failed"
	}

	a.runManager.UpdateRunStatus(run.RunID, RunStatus(finalStatus))

	// Emit final event
	EmitFinal(run.EventChan, run.RunID, finalStatus, len(plan.Steps), completedSteps, failedSteps, len(run.Artifacts), time.Since(startTime).String())

	backend.Logger.Info("Run completed", "runId", run.RunID, "status", finalStatus, "duration", time.Since(startTime))
}

// generateExecutionPlan calls LLM to generate a structured execution plan
func (a *App) generateExecutionPlan(ctx context.Context, req AssistantRequest, incomingReq *http.Request) (*ExecutionPlan, error) {
	// Check if we should use mock mode (for testing without LLM)
	// Set environment variable ZAGALIN_MOCK_LLM=true to enable
	if os.Getenv("ZAGALIN_MOCK_LLM") == "true" {
		backend.Logger.Info("Using mock LLM mode for planning")
		return &ExecutionPlan{
			Goal: fmt.Sprintf("Mock analysis of: %s", req.Message),
			Steps: []PlannedStep{
				{
					Index:       0,
					Title:       "Step 1: Analyze metrics",
					Description: "Check key performance indicators and identify anomalies",
					Status:      "pending",
				},
				{
					Index:       1,
					Title:       "Step 2: Review logs",
					Description: "Examine error logs for patterns and root causes",
					Status:      "pending",
				},
				{
					Index:       2,
					Title:       "Step 3: Trace requests",
					Description: "Follow distributed traces to identify bottlenecks",
					Status:      "pending",
				},
			},
			EstimatedDuration: "2-3 minutes",
		}, nil
	}

	// Build planning prompt
	planningPrompt := BuildPlanningPrompt(req.Message, req.Context)

	// Prepare LLM request for planning
	messages := []AssistantMessage{
		{
			Role:    "system",
			Content: PLANNING_SYSTEM_PROMPT,
		},
		{
			Role:    "user",
			Content: planningPrompt,
		},
	}

	llmReq := LLMStreamRequest{
		Model:       "gpt-4o-mini",
		Messages:    messages,
		Temperature: 0.3, // Lower temperature for structured output
		MaxTokens:   1000,
		Tools:       nil,       // No tools for planning
		ToolChoice:  "none",
	}

	// Create LLM client based on settings
	var chunkChan <-chan LLMStreamChunk
	var err error

	if a.settings.LLMBackend == "direct" {
		// Direct mode: call OpenAI/Anthropic directly
		if a.settings.LLMAPIKey == "" {
			return nil, fmt.Errorf("direct LLM mode requires an API key to be configured")
		}

		directClient := NewDirectLLMClient(
			a.settings.LLMProvider,
			a.settings.LLMModel,
			a.settings.LLMEndpoint,
			a.settings.LLMAPIKey,
			http.DefaultClient,
			backend.Logger,
		)

		chunkChan, err = directClient.StreamChat(ctx, llmReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call direct LLM for planning: %w", err)
		}
	} else {
		// grafana-llm-app mode should be called from frontend
		return nil, fmt.Errorf("grafana-llm-app mode must be called from frontend")
	}

	// Accumulate response
	var responseText strings.Builder
	for chunk := range chunkChan {
		if chunk.Error != "" {
			return nil, fmt.Errorf("LLM error during planning: %s", chunk.Error)
		}
		if chunk.Chunk != "" {
			responseText.WriteString(chunk.Chunk)
		}
	}

	planText := responseText.String()

	// Parse plan from JSON
	plan, err := parsePlanFromJSON(planText)
	if err != nil {
		backend.Logger.Warn("Failed to parse plan as JSON, using fallback", "error", err)
		// Fallback: create simple 1-step plan
		plan = &ExecutionPlan{
			Goal: "Analyze the request",
			Steps: []PlannedStep{
				{
					Index:       0,
					Title:       "Analyze and respond",
					Description: "Analyze the user's request and provide a comprehensive response",
					Status:      "pending",
				},
			},
			EstimatedDuration: "1-2 minutes",
		}
	}

	// Validate plan
	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}
	if len(plan.Steps) > 5 {
		plan.Steps = plan.Steps[:5] // Limit to 5 steps
	}

	// Set step indices and status
	for i := range plan.Steps {
		plan.Steps[i].Index = i
		plan.Steps[i].Status = "pending"
	}

	return plan, nil
}

// executeStep executes a single step and returns the result and artifacts
func (a *App) executeStep(ctx context.Context, run *RunState, req AssistantRequest, step PlannedStep, stepIndex int, incomingReq *http.Request) (string, []Artifact, error) {
	// Check if we should use mock mode
	if os.Getenv("ZAGALIN_MOCK_LLM") == "true" {
		backend.Logger.Info("Using mock LLM mode for step execution", "step", stepIndex)

		// Simulate some processing time
		time.Sleep(1 * time.Second)

		// Generate mock response with artifacts
		mockResponse := fmt.Sprintf("Executing %s...\n\nBased on the analysis, here are the key findings:\n\n", step.Title)
		mockResponse += "1. System is operating within normal parameters\n"
		mockResponse += "2. No critical errors detected in the timeframe\n"
		mockResponse += "3. Query: rate(http_requests_total[5m])\n\n"
		mockResponse += "The metrics show stable performance with no anomalies."

		// Create mock artifacts
		artifacts := []Artifact{
			{
				ID:        fmt.Sprintf("artifact_%s", uuid.New().String()),
				Type:      "query",
				Content:   "rate(http_requests_total[5m])",
				Metadata: map[string]interface{}{
					"signal":       "metrics",
					"format":       "promql",
					"datasource":   "prometheus",
					"datasourceUid": "prom-1",
				},
				Timestamp: time.Now(),
			},
		}

		// Emit the response text gradually (simulate streaming)
		words := strings.Split(mockResponse, " ")
		for i, word := range words {
			if i > 0 {
				EmitAssistantDelta(run.EventChan, run.RunID, " ")
			}
			EmitAssistantDelta(run.EventChan, run.RunID, word)
			time.Sleep(50 * time.Millisecond)
		}

		return mockResponse, artifacts, nil
	}

	// Detect skill for this step
	skill := DetectSkill(step.Description, req.Context)

	// Build step execution prompt
	systemPrompt := BuildSystemPrompt(skill, req.Context)

	// Build context-aware step prompt
	var previousFindings strings.Builder
	if stepIndex > 0 {
		previousFindings.WriteString("Previous findings:\n")
		run.mu.RLock()
		for i := 0; i < stepIndex && i < len(run.Plan.Steps); i++ {
			previousFindings.WriteString(fmt.Sprintf("- Step %d: %s\n", i+1, run.Plan.Steps[i].Title))
		}
		run.mu.RUnlock()
	}

	stepPrompt := fmt.Sprintf(`You are executing step %d of %d in the plan.

Goal: %s

Current step:
- Title: %s
- Description: %s

%s

User's original message: %s

Please execute this step and provide concrete findings. Generate specific queries or provide analysis as needed.`,
		stepIndex+1,
		len(run.Plan.Steps),
		run.Plan.Goal,
		step.Title,
		step.Description,
		previousFindings.String(),
		req.Message,
	)

	// Prepare LLM request
	messages := []AssistantMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: stepPrompt,
		},
	}

	llmReq := LLMStreamRequest{
		Model:       "gpt-4o-mini",
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
		Tools:       GetTools(true), // Enable tools for execution
		ToolChoice:  "auto",
	}

	// Create LLM client based on settings
	var chunkChan <-chan LLMStreamChunk
	var err error

	if a.settings.LLMBackend == "direct" {
		// Direct mode
		directClient := NewDirectLLMClient(
			a.settings.LLMProvider,
			a.settings.LLMModel,
			a.settings.LLMEndpoint,
			a.settings.LLMAPIKey,
			http.DefaultClient,
			backend.Logger,
		)

		chunkChan, err = directClient.StreamChat(ctx, llmReq)
		if err != nil {
			return "", nil, fmt.Errorf("failed to call direct LLM for step execution: %w", err)
		}
	} else {
		// grafana-llm-app mode should be called from frontend
		return "", nil, fmt.Errorf("grafana-llm-app mode must be called from frontend")
	}

	// Stream response and collect artifacts
	var responseText strings.Builder
	var artifacts []Artifact
	var toolCalls []ToolCallChunk

	for chunk := range chunkChan {
		// Check pause/cancel
		shouldContinue, _ := a.runManager.CheckPauseOrCancel(run.RunID)
		if !shouldContinue {
			return responseText.String(), artifacts, fmt.Errorf("step cancelled")
		}

		if chunk.Error != "" {
			return responseText.String(), artifacts, fmt.Errorf("LLM error: %s", chunk.Error)
		}

		// Handle text chunks
		if chunk.Chunk != "" {
			responseText.WriteString(chunk.Chunk)
			// Emit streaming delta
			EmitAssistantDelta(run.EventChan, run.RunID, chunk.Chunk)
		}

		// Handle tool calls
		if chunk.ToolCall != nil {
			toolCalls = append(toolCalls, *chunk.ToolCall)
		}
	}

	finalText := responseText.String()

	// Extract artifacts from tool calls
	for _, toolCall := range toolCalls {
		artifact := Artifact{
			ID:        fmt.Sprintf("art_%s", uuid.New().String()),
			Type:      "tool_call",
			Content:   toolCall.Function.Name,
			Metadata: map[string]interface{}{
				"function":  toolCall.Function.Name,
				"arguments": toolCall.Function.Arguments,
			},
			Timestamp: time.Now().UTC(),
		}
		artifacts = append(artifacts, artifact)
	}

	// Extract artifacts from text (queries)
	textArtifacts := extractArtifactsFromText(finalText)
	artifacts = append(artifacts, textArtifacts...)

	return finalText, artifacts, nil
}

// extractArtifactsFromText extracts queries and links from text using regex
func extractArtifactsFromText(text string) []Artifact {
	var artifacts []Artifact

	// Extract PromQL queries (rate(...), sum(...), etc.)
	promqlPattern := regexp.MustCompile(`(?m)^[a-z_]+\([^)]+\)(?:\{[^}]+\})?(?:\[[^\]]+\])?`)
	promqlMatches := promqlPattern.FindAllString(text, -1)
	for _, match := range promqlMatches {
		if len(match) > 10 { // Filter out very short matches
			artifacts = append(artifacts, Artifact{
				ID:      fmt.Sprintf("art_%s", uuid.New().String()),
				Type:    "query",
				Content: strings.TrimSpace(match),
				Metadata: map[string]interface{}{
					"signal": "metrics",
					"format": "promql",
				},
				Timestamp: time.Now().UTC(),
			})
		}
	}

	// Extract LogQL queries ({...})
	logqlPattern := regexp.MustCompile(`\{[^}]+\}\s*(?:\|[^|]+)*`)
	logqlMatches := logqlPattern.FindAllString(text, -1)
	for _, match := range logqlMatches {
		if strings.Contains(match, "=") { // Must have label selector
			artifacts = append(artifacts, Artifact{
				ID:      fmt.Sprintf("art_%s", uuid.New().String()),
				Type:    "query",
				Content: strings.TrimSpace(match),
				Metadata: map[string]interface{}{
					"signal": "logs",
					"format": "logql",
				},
				Timestamp: time.Now().UTC(),
			})
		}
	}

	// Extract trace IDs (hex strings 16-32 chars)
	traceIDPattern := regexp.MustCompile(`\b[0-9a-f]{16,32}\b`)
	traceIDMatches := traceIDPattern.FindAllString(strings.ToLower(text), -1)
	seen := make(map[string]bool)
	for _, match := range traceIDMatches {
		if !seen[match] && len(match) >= 16 {
			seen[match] = true
			artifacts = append(artifacts, Artifact{
				ID:      fmt.Sprintf("art_%s", uuid.New().String()),
				Type:    "trace_id",
				Content: match,
				Metadata: map[string]interface{}{
					"signal": "traces",
				},
				Timestamp: time.Now().UTC(),
			})
		}
	}

	return artifacts
}

// buildFinalMessage constructs the final assistant message with Goal/Plan/Evidence/Conclusion format
func (a *App) buildFinalMessage(plan *ExecutionPlan, artifacts []Artifact) string {
	var message strings.Builder

	message.WriteString(fmt.Sprintf("**Goal:** %s\n\n", plan.Goal))

	message.WriteString("**Plan:**\n")
	for i, step := range plan.Steps {
		message.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Title))
	}
	message.WriteString("\n")

	if len(artifacts) > 0 {
		message.WriteString("**Evidence:**\n")
		queryCount := 0
		linkCount := 0
		traceCount := 0

		for _, artifact := range artifacts {
			switch artifact.Type {
			case "query":
				queryCount++
				signal, _ := artifact.Metadata["signal"].(string)
				message.WriteString(fmt.Sprintf("- Query (%s): `%s`\n", signal, artifact.Content))
			case "link":
				linkCount++
				message.WriteString(fmt.Sprintf("- Link: %s\n", artifact.Content))
			case "trace_id":
				traceCount++
				message.WriteString(fmt.Sprintf("- Trace ID: `%s`\n", artifact.Content))
			}
		}

		if queryCount+linkCount+traceCount == 0 {
			message.WriteString("- Analysis completed without generating specific artifacts\n")
		}
		message.WriteString("\n")
	}

	message.WriteString("**Conclusion:**\n")
	message.WriteString("The investigation has been completed according to the plan. Review the evidence above and the detailed findings in each step.\n\n")

	message.WriteString("**Next Steps:**\n")
	message.WriteString("- Review the generated queries and artifacts\n")
	message.WriteString("- Drill down into specific findings\n")
	message.WriteString("- Ask follow-up questions if needed\n")

	return message.String()
}
