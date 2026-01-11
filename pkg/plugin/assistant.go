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

type AssistantRequest struct {
	Message  string             `json:"message"`
	History  []AssistantMessage `json:"history"`
	Context  AssistantContext   `json:"context"`
	SkillHint string            `json:"skillHint,omitempty"`
	Mode     string             `json:"mode,omitempty"` 
}

type AssistantResponse struct {
	Message   string `json:"message"`
	SkillUsed string `json:"skillUsed"`
}

func detectSignalType(message string, context AssistantContext) string {
	messageLower := strings.ToLower(message)

	if context.Dashboard != nil && len(context.Dashboard.Panels) > 0 {
		hasMetrics := false
		hasLogs := false
		hasTraces := false

		for _, panel := range context.Dashboard.Panels {
			for _, target := range panel.Targets {
				if target.Datasource != nil {
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

		if strings.Contains(messageLower, "dashboard") ||
			strings.Contains(messageLower, "panel") ||
			strings.Contains(messageLower, "what am i seeing") ||
			strings.Contains(messageLower, "what do i see") ||
			strings.Contains(messageLower, "what am i looking at") {
			return "dashboard"
		}

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

	logsKeywords := []string{
		"logql", "loki", "log", "error message", "stack trace",
		"log line", "log stream", "log entry",
	}
	for _, keyword := range logsKeywords {
		if strings.Contains(messageLower, keyword) {
			return "logs"
		}
	}

	tracesKeywords := []string{
		"traceql", "tempo", "trace", "span", "jaeger", "zipkin",
		"trace id", "traceid", "distributed tracing",
	}
	for _, keyword := range tracesKeywords {
		if strings.Contains(messageLower, keyword) {
			return "traces"
		}
	}

	investigationKeywords := []string{
		"why", "investigate", "troubleshoot", "debug", "root cause",
		"error", "spike", "increase", "decrease", "problem",
	}
	for _, keyword := range investigationKeywords {
		if strings.Contains(messageLower, keyword) {
			return "investigation"
		}
	}

	return "general"
}

type llmChatPipeline struct {
	user         *UserIdentity
	assistantReq AssistantRequest
	skill        string
	mode         string
	messages     []AssistantMessage
	llmReq       LLMStreamRequest
}

func (a *App) authenticateRequest(req *http.Request) (*UserIdentity, error) {
	return a.extractUserFromRequest(req)
}

func (a *App) checkRateLimit(user *UserIdentity) error {
	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("Rate limit exceeded", "user", user.UserLogin)
			return fmt.Errorf("too many requests")
		}
	}
	return nil
}

func (a *App) parseAssistantRequest(req *http.Request) (*AssistantRequest, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer req.Body.Close()

	var assistantReq AssistantRequest
	if err := json.Unmarshal(body, &assistantReq); err != nil {
		return nil, fmt.Errorf("invalid request format: %w", err)
	}

	if assistantReq.Message == "" && len(assistantReq.History) == 0 {
		return nil, fmt.Errorf("message or history required")
	}

	return &assistantReq, nil
}

func (a *App) determineSkillAndMode(assistantReq *AssistantRequest) (skill string, mode string) {
	skill = assistantReq.SkillHint
	if skill == "" {
		skill = DetectSkill(assistantReq.Message, assistantReq.Context)
	}

	mode = assistantReq.Mode
	if mode == "" {
		mode = "standard"
	}

	return skill, mode
}

func (a *App) buildMessages(skill string, assistantReq *AssistantRequest) []AssistantMessage {
	systemPrompt := BuildSystemPrompt(skill, assistantReq.Context, a.settings, a.contextManager, assistantReq.Mode)
	userPrompt := BuildUserPrompt(skill, assistantReq.Message, assistantReq.Context)

	messages := []AssistantMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	messages = append(messages, assistantReq.History...)

	if skill == "" || skill == "generate_query" || skill == "troubleshoot" {
		messages = append(messages, AssistantMessage{
			Role:    "user",
			Content: userPrompt,
		})
	} else {
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

	return messages
}

func (a *App) buildLLMRequest(messages []AssistantMessage, mode string) LLMStreamRequest {
	maxTokens := a.settings.StandardModeMaxTokens
	temperature := a.settings.StandardModeTemperature

	if mode == "design" {
		maxTokens = a.settings.DesignModeMaxTokens
		temperature = a.settings.DesignModeTemperature
	}

	model := ""
	if a.settings != nil && a.settings.LLMModel != "" {
		model = a.settings.LLMModel
		backend.Logger.Debug("Using model from plugin settings", "model", model, "mode", mode)
	} else {
		backend.Logger.Debug("Using default model from grafana-llm-app (no model configured in Zagalin settings)", "mode", mode)
	}

	return LLMStreamRequest{
		Model:               model,
		Messages:            messages,
		Temperature:         temperature,
		MaxTokens:           getMaxTokensForModel(model, maxTokens),
		MaxCompletionTokens: getMaxCompletionTokensForModel(model, maxTokens),
		Tools:               GetTools(true, a.settings),
		ToolChoice:          "auto",
		Stream:              true,
	}
}

func (a *App) createLLMClient() *LLMClient {
	serviceAccountToken := ""
	if a.settings != nil && a.settings.ServiceAccountToken != "" {
		serviceAccountToken = a.settings.ServiceAccountToken
		backend.Logger.Debug("Using service account token from plugin settings")
	} else {
		backend.Logger.Debug("No service account token in settings, will try plugin context fallback")
	}

	return NewLLMClient(
		getGrafanaURL(),
		serviceAccountToken,
		http.DefaultClient,
		backend.Logger,
	)
}

func (a *App) handleLLMChat(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(rw, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	if err := a.checkRateLimit(user); err != nil {
		sendErrorResponse(rw, "Rate limit exceeded", err, http.StatusTooManyRequests)
		return
	}

	assistantReq, err := a.parseAssistantRequest(req)
	if err != nil {
		sendErrorResponse(rw, "Invalid request", err, http.StatusBadRequest)
		return
	}

	skill, mode := a.determineSkillAndMode(assistantReq)
	messages := a.buildMessages(skill, assistantReq)
	llmReq := a.buildLLMRequest(messages, mode)
	llmClient := a.createLLMClient()

	// DEBUG: Log tools being sent
	backend.Logger.Info("LLM request prepared",
		"skill", skill,
		"mode", mode,
		"model", llmReq.Model,
		"toolCount", len(llmReq.Tools),
		"temperature", llmReq.Temperature,
	)
	if len(llmReq.Tools) > 0 {
		toolNames := make([]string, 0, len(llmReq.Tools))
		for _, tool := range llmReq.Tools {
			toolNames = append(toolNames, tool.Function.Name)
		}
		backend.Logger.Info("Tools included in LLM request", "tools", toolNames)
	}

	chunkChan, err := llmClient.StreamChat(ctx, llmReq, req)
	if err != nil {
		backend.Logger.Error("Failed to call grafana-llm-app", "error", err, "user", user.UserLogin)
		sendErrorResponse(rw, "Failed to call LLM", err, http.StatusInternalServerError)
		return
	}

	signalType := detectSignalType(assistantReq.Message, assistantReq.Context)

	backend.Logger.Info("LLM chat request",
		"user", user.UserLogin,
		"orgId", user.OrgID,
		"skill", skill,
		"signalType", signalType,
		"hasDashboardContext", assistantReq.Context.Dashboard != nil,
		"messageLength", len(assistantReq.Message),
		"historyLength", len(assistantReq.History),
	)

	a.streamSSEResponseWithValidation(ctx, rw, chunkChan, skill, user)
}

func streamSSEResponse(ctx context.Context, rw http.ResponseWriter, chunkChan <-chan LLMStreamChunk, skill string) {
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.WriteHeader(http.StatusOK)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		backend.Logger.Error("ResponseWriter does not support flushing")
		return
	}

	if skill != "" {
		metadataJSON, _ := json.Marshal(map[string]string{"skill": skill})
		fmt.Fprintf(rw, "data: %s\n\n", string(metadataJSON))
		flusher.Flush()
	}

	for {
		select {
		case <-ctx.Done():
			backend.Logger.Debug("Client disconnected")
			return

		case chunk, ok := <-chunkChan:
			if !ok {
				fmt.Fprintf(rw, "data: {\"done\":true}\n\n")
				flusher.Flush()
				return
			}

			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				backend.Logger.Error("Failed to marshal chunk", "error", err)
				continue
			}

			fmt.Fprintf(rw, "data: %s\n\n", string(chunkJSON))
			flusher.Flush()

			if chunk.Done || chunk.Error != "" {
				return
			}
		}
	}
}

func (a *App) streamSSEResponseWithValidation(ctx context.Context, rw http.ResponseWriter, chunkChan <-chan LLMStreamChunk, skill string, user *UserIdentity) {
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.WriteHeader(http.StatusOK)

	flusher, ok := rw.(http.Flusher)
	if !ok {
		backend.Logger.Error("ResponseWriter does not support flushing")
		return
	}

	if skill != "" {
		metadataJSON, _ := json.Marshal(map[string]string{"skill": skill})
		fmt.Fprintf(rw, "data: %s\n\n", string(metadataJSON))
		flusher.Flush()
	}

	toolCallAccumulator := make(map[string]*ToolCallChunk)

	backend.Logger.Debug("Starting SSE stream with validation")

	startTime := time.Now()
	chunkCount := 0

	for {
		select {
		case <-ctx.Done():
			// Client disconnected (user clicked stop or network issue)
			backend.Logger.Info("LLM stream cancelled - client disconnected",
				"user", user.UserLogin,
				"skill", skill,
				"duration", time.Since(startTime),
				"chunksDelivered", chunkCount,
				"reason", ctx.Err(),
			)
			return

		case chunk, ok := <-chunkChan:
			if !ok {
				backend.Logger.Debug("LLM channel closed, sending done")
				fmt.Fprintf(rw, "data: {\"done\":true}\n\n")
				flusher.Flush()
				return
			}

			chunkCount++

			backend.Logger.Debug("Received chunk from LLM", "hasToolCall", chunk.ToolCall != nil, "hasChunk", chunk.Chunk != "", "hasError", chunk.Error != "", "done", chunk.Done)

			if chunk.ToolCall != nil {
				if existing, exists := toolCallAccumulator[chunk.ToolCall.ID]; exists {
					existing.Function.Arguments += chunk.ToolCall.Function.Arguments
				} else {
					toolCallAccumulator[chunk.ToolCall.ID] = chunk.ToolCall
				}

				accumulated := toolCallAccumulator[chunk.ToolCall.ID]
				if isCompleteJSON(accumulated.Function.Arguments) {
					validatedChunk := a.validateToolCall(ctx, accumulated, user)

					chunkJSON, err := json.Marshal(validatedChunk)
					if err != nil {
						backend.Logger.Error("Failed to marshal validated chunk", "error", err)
						continue
					}

					fmt.Fprintf(rw, "data: %s\n\n", string(chunkJSON))
					flusher.Flush()

					delete(toolCallAccumulator, chunk.ToolCall.ID)
				}
				continue
			}

			chunkJSON, err := json.Marshal(chunk)
			if err != nil {
				backend.Logger.Error("Failed to marshal chunk", "error", err)
				continue
			}

			backend.Logger.Debug("Sending chunk to client", "chunkSize", len(chunkJSON))
			fmt.Fprintf(rw, "data: %s\n\n", string(chunkJSON))
			flusher.Flush()

			if chunk.Done || chunk.Error != "" {
				// Stream completed successfully or with error
				backend.Logger.Info("LLM stream completed",
					"user", user.UserLogin,
					"skill", skill,
					"duration", time.Since(startTime),
					"chunksDelivered", chunkCount,
					"completedSuccessfully", chunk.Done && chunk.Error == "",
					"error", chunk.Error,
				)
				return
			}
		}
	}
}

func isCompleteJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) == 0 || s[0] != '{' {
		return false
	}
	braceCount := 0
	for _, ch := range s {
		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			braceCount--
		}
	}
	return braceCount == 0
}

func (a *App) validateToolCall(ctx context.Context, toolCall *ToolCallChunk, user *UserIdentity) LLMStreamChunk {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return LLMStreamChunk{
			Error: "Invalid tool arguments: " + err.Error(),
		}
	}

	if a.settings != nil && !a.settings.ToolCallValidation {
		return LLMStreamChunk{
			ToolCall: toolCall,
		}
	}

	if toolCall.Function.Name == "create_promql_query" {
		query := a.extractPromQLFromToolArgs(args)
		if query != "" {
			result := a.queryValidator.ValidateQuery(ctx, query, DatasourcePrometheus)

			if !result.Valid {
				args["_validation_error"] = result.Error.Error()
				args["_validation_type"] = result.ViolationType

				backend.Logger.Warn("Tool PromQL validation failed",
					"user", user.UserLogin,
					"violation", result.ViolationType,
					"queryHash", hashQuery(query),
					"queryLength", len(query))

				userIdentity := &UserIdentity{
					UserID:    0, 
					UserLogin: user.UserLogin,
					UserEmail: "", 
					OrgID:     user.OrgID,
					OrgName:   "", 
				}
				a.logQueryValidationFailure(userIdentity, "prometheus", result)

			} else if result.Sanitized {
				args["_sanitized_query"] = result.SanitizedQuery

				backend.Logger.Info("Tool PromQL sanitized",
					"user", user.UserLogin,
					"originalHash", hashQuery(query),
					"sanitizedHash", hashQuery(result.SanitizedQuery),
					"originalLength", len(query),
					"sanitizedLength", len(result.SanitizedQuery))
			}

			modifiedArgs, _ := json.Marshal(args)
			toolCall.Function.Arguments = string(modifiedArgs)
		}
	}

	if toolCall.Function.Name == "create_logql_query" {
		query := a.extractLogQLFromToolArgs(args)
		if query != "" {
			result := a.queryValidator.ValidateQuery(ctx, query, DatasourceLoki)

			if !result.Valid {
				args["_validation_error"] = result.Error.Error()
				args["_validation_type"] = result.ViolationType

				backend.Logger.Warn("Tool LogQL validation failed",
					"user", user.UserLogin,
					"violation", result.ViolationType,
					"queryHash", hashQuery(query),
					"queryLength", len(query))

			userIdentity := &UserIdentity{
				UserID:    0, 
				UserLogin: user.UserLogin,
				UserEmail: "", 
				OrgID:     user.OrgID,
				OrgName:   "", 
			}
			a.logQueryValidationFailure(userIdentity, "loki", result)

			} else if result.Sanitized {
				args["_sanitized_query"] = result.SanitizedQuery

				backend.Logger.Info("Tool LogQL sanitized",
					"user", user.UserLogin,
					"originalHash", hashQuery(query),
					"sanitizedHash", hashQuery(result.SanitizedQuery),
					"originalLength", len(query),
					"sanitizedLength", len(result.SanitizedQuery))
			}

			modifiedArgs, _ := json.Marshal(args)
			toolCall.Function.Arguments = string(modifiedArgs)
		}
	}

	return LLMStreamChunk{
		ToolCall: toolCall,
	}
}

func (a *App) extractOTelLabelsFromArgs(args map[string]interface{}, dsType DatasourceType) []string {
	if a.settings == nil || !a.settings.OtelEnforcement.Enabled {
		return []string{}
	}

	serviceName, _ := args["serviceName"].(string)
	environmentName, _ := args["environmentName"].(string)

	if serviceName == "" && environmentName == "" {
		return []string{}
	}

	serviceLabel, environmentLabel := getDefaultLabelNames(dsType)

	var labels []string
	if serviceName != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, serviceLabel, serviceName))
	}
	if environmentName != "" {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, environmentLabel, environmentName))
	}

	return labels
}

func injectLabelsIntoLogStream(logStream string, labels []string) string {
	if len(labels) == 0 {
		return logStream
	}

	logStream = strings.TrimSuffix(logStream, "}")

	if strings.Contains(logStream, "=") {
		return logStream + "," + strings.Join(labels, ",") + "}"
	}

	logStream = strings.TrimSuffix(logStream, "{")
	return "{" + strings.Join(labels, ",") + "}"
}

func (a *App) extractPromQLFromToolArgs(args map[string]interface{}) string {
	metric, _ := args["metric"].(string)
	if metric == "" {
		return ""
	}

	query := metric

	var filterParts []string

	filterParts = append(filterParts, a.extractOTelLabelsFromArgs(args, DatasourcePrometheus)...)

	if filters, ok := args["filters"].(map[string]interface{}); ok {
		for k, v := range filters {
			filterParts = append(filterParts, fmt.Sprintf("%s=\"%v\"", k, v))
		}
	}

	if len(filterParts) > 0 {
		query = fmt.Sprintf("%s{%s}", metric, strings.Join(filterParts, ","))
	}

	if agg, ok := args["aggregation"].(string); ok && agg != "" {
		if agg == "rate" {
			timeRange, _ := args["timeRange"].(string)
			if timeRange == "" {
				timeRange = "5m"
			}
			query = fmt.Sprintf("rate(%s[%s])", query, timeRange)
		} else {
			query = fmt.Sprintf("%s(%s)", agg, query)
		}
	}

	return query
}

func (a *App) extractLogQLFromToolArgs(args map[string]interface{}) string {
	logStream, _ := args["logStream"].(string)
	if logStream == "" {
		return ""
	}

	otelLabels := a.extractOTelLabelsFromArgs(args, DatasourceLoki)
	if len(otelLabels) > 0 {
		logStream = injectLabelsIntoLogStream(logStream, otelLabels)
	}

	query := logStream

	if filter, ok := args["filter"].(string); ok && filter != "" {
		query += fmt.Sprintf(" |= \"%s\"", filter)
	}

	if parser, ok := args["parser"].(string); ok && parser != "" {
		query += fmt.Sprintf(" | %s", parser)
	}

	return query
}

func (a *App) extractUserFromRequest(req *http.Request) (*UserIdentity, error) {
	pluginContext := backend.PluginConfigFromContext(req.Context())

	if pluginContext.User == nil {
		return nil, fmt.Errorf("user not found in context")
	}

	user := pluginContext.User

	if user.Login == "" {
		return nil, fmt.Errorf("user login is empty")
	}

	return &UserIdentity{
		UserID:    0, // backend.User doesn't expose ID
		UserLogin: user.Login,
		UserEmail: user.Email,
		OrgID:     pluginContext.OrgID,
		OrgName:   "", // backend.PluginContext doesn't expose OrgName
	}, nil
}

func getGrafanaURL() string {
	if url := os.Getenv("GF_URL"); url != "" {
		backend.Logger.Debug("Using GF_URL override from environment", "url", url)
		return url
	}

	protocol := os.Getenv("GF_SERVER_PROTOCOL")
	domain := os.Getenv("GF_SERVER_DOMAIN")
	port := os.Getenv("GF_SERVER_HTTP_PORT")

	if protocol != "" || domain != "" || port != "" {
		if protocol == "" {
			protocol = "http"
		}
		if domain == "" {
			domain = "localhost"
		}
		if port == "" {
			port = "3000"
		}

		var url string
		if port == "80" || port == "443" {
			url = fmt.Sprintf("%s://%s", protocol, domain)
		} else {
			url = fmt.Sprintf("%s://%s:%s", protocol, domain, port)
		}

		backend.Logger.Debug("Using environment variable overrides",
			"url", url,
			"protocol", protocol,
			"domain", domain,
			"port", port,
		)
		return url
	}

	url := "http://localhost:3000"
	backend.Logger.Debug("Using default localhost URL (plugin runs inside Grafana)", "url", url)
	return url
}

func (a *App) orchestrateRunFull(ctx context.Context, run *RunState, req AssistantRequest, incomingReq *http.Request) {
	startTime := time.Now()

	defer func() {
		a.runManager.CloseEventChannel(run.RunID)
		a.runManager.ScheduleCleanup(run.RunID, 1*time.Hour)
	}()

	EmitRunStarted(run.EventChan, run.RunID, run.ConversationID)

	backend.Logger.Debug("Skipping health check, proceeding directly to LLM call", "runId", run.RunID)

	a.runManager.UpdateRunStatus(run.RunID, RunStatusPlanning)

	plan, err := a.generateExecutionPlan(ctx, req, incomingReq)
	if err != nil {
		backend.Logger.Error("Failed to generate execution plan", "error", err, "runId", run.RunID)

		errorMsg := fmt.Sprintf("Planning failed: %s\n\nIf this error persists, please check:\n1. grafana-llm-app plugin configuration\n2. LLM provider API credentials\n3. Network connectivity", err.Error())
		EmitError(run.EventChan, run.RunID, errorMsg, -1, false)

		a.runManager.UpdateRunStatus(run.RunID, RunStatusFailed)
		EmitFinal(run.EventChan, run.RunID, "failed", 0, 0, 0, 0, time.Since(startTime).String())
		return
	}

	a.runManager.SetPlan(run.RunID, plan)

	EmitPlan(run.EventChan, run.RunID, plan)

	backend.Logger.Info("Execution plan generated", "runId", run.RunID, "steps", len(plan.Steps))

	a.runManager.UpdateRunStatus(run.RunID, RunStatusExecuting)

	completedSteps := 0
	failedSteps := 0

	for i, step := range plan.Steps {
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

		a.runManager.UpdateStepStatus(run.RunID, i, "in_progress")

		EmitStepStarted(run.EventChan, run.RunID, i, step.Title, step.Description)

		stepResult, artifacts, stepErr := a.executeStep(ctx, run, req, step, i, incomingReq)

		for _, artifact := range artifacts {
			a.runManager.AddArtifact(run.RunID, artifact)
			EmitArtifact(run.EventChan, run.RunID, artifact, i)
		}

		stepStatus := "completed"
		if stepErr != nil {
			stepStatus = "failed"
			failedSteps++
			EmitError(run.EventChan, run.RunID, stepErr.Error(), i, false)
		} else {
			completedSteps++
		}

		a.runManager.UpdateStepStatus(run.RunID, i, stepStatus)

		errorMsg := ""
		if stepErr != nil {
			errorMsg = stepErr.Error()
		}
		EmitStepDone(run.EventChan, run.RunID, i, step.Title, stepStatus, stepResult, errorMsg)

		backend.Logger.Info("Step completed", "runId", run.RunID, "stepIndex", i, "status", stepStatus)
	}

	finalMessage := a.buildFinalMessage(plan, run.Artifacts)

	messageID := fmt.Sprintf("msg_%s", uuid.New().String())
	EmitAssistantMessage(run.EventChan, run.RunID, messageID, "assistant", finalMessage)

	finalStatus := "completed"
	if failedSteps == len(plan.Steps) {
		finalStatus = "failed"
	}

	a.runManager.UpdateRunStatus(run.RunID, RunStatus(finalStatus))

	EmitFinal(run.EventChan, run.RunID, finalStatus, len(plan.Steps), completedSteps, failedSteps, len(run.Artifacts), time.Since(startTime).String())

	backend.Logger.Info("Run completed", "runId", run.RunID, "status", finalStatus, "duration", time.Since(startTime))
}

func (a *App) generateExecutionPlan(ctx context.Context, req AssistantRequest, incomingReq *http.Request) (*ExecutionPlan, error) {
	planningPrompt := BuildPlanningPrompt(req.Message, req.Context)

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
		Model:               "gpt-4o-mini",
		Messages:            messages,
		Temperature:         0.3,  
		MaxTokens:           getMaxTokensForModel("gpt-4o-mini", 1000),
		MaxCompletionTokens: getMaxCompletionTokensForModel("gpt-4o-mini", 1000),
		Tools:               nil, 
		ToolChoice:          "none",
	}

	var chunkChan <-chan LLMStreamChunk
	var err error

	if a.settings.LLMBackend == "direct" {
		backend.Logger.Warn("⚠️ Direct LLM mode is experimental and not fully tested. Use with caution.",
			"provider", a.settings.LLMProvider,
			"model", a.settings.LLMModel,
		)

		if a.settings.LLMAPIKey == "" {
			return nil, fmt.Errorf("direct LLM mode requires an API key to be configured")
		}

		directClient := NewDirectLLMClient(
			a.settings.LLMProvider,
			a.settings.LLMModel,
			a.settings.LLMEndpoint,
			a.settings.LLMAPIKey,
			a.settings.LLMOrganization,
			http.DefaultClient,
			backend.Logger,
		)

		chunkChan, err = directClient.StreamChat(ctx, llmReq)
		if err != nil {
			return nil, fmt.Errorf("failed to call direct LLM for planning: %w", err)
		}
	} else {
		return nil, fmt.Errorf("grafana-llm-app mode must be called from frontend")
	}

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

	plan, err := parsePlanFromJSON(planText)
	if err != nil {
		backend.Logger.Warn("Failed to parse plan as JSON, using fallback", "error", err)
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

	if len(plan.Steps) == 0 {
		return nil, fmt.Errorf("plan has no steps")
	}
	if len(plan.Steps) > 5 {
		plan.Steps = plan.Steps[:5] 
	}

	for i := range plan.Steps {
		plan.Steps[i].Index = i
		plan.Steps[i].Status = "pending"
	}

	return plan, nil
}

func (a *App) executeStep(ctx context.Context, run *RunState, req AssistantRequest, step PlannedStep, stepIndex int, incomingReq *http.Request) (string, []Artifact, error) {
	skill := DetectSkill(step.Description, req.Context)

	systemPrompt := BuildSystemPrompt(skill, req.Context, a.settings, a.contextManager, req.Mode)

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
		Model:               "gpt-4o-mini",
		Messages:            messages,
		Temperature:         0.7,
		MaxTokens:           getMaxTokensForModel("gpt-4o-mini", 2000),
		MaxCompletionTokens: getMaxCompletionTokensForModel("gpt-4o-mini", 2000),
		Tools:               GetTools(true, a.settings), 
		ToolChoice:          "auto",
	}

	var chunkChan <-chan LLMStreamChunk
	var err error

	if a.settings.LLMBackend == "direct" {
		backend.Logger.Warn("⚠️ Direct LLM mode is experimental and not fully tested. Use with caution.",
			"provider", a.settings.LLMProvider,
			"model", a.settings.LLMModel,
		)

		directClient := NewDirectLLMClient(
			a.settings.LLMProvider,
			a.settings.LLMModel,
			a.settings.LLMEndpoint,
			a.settings.LLMAPIKey,
			a.settings.LLMOrganization,
			http.DefaultClient,
			backend.Logger,
		)

		chunkChan, err = directClient.StreamChat(ctx, llmReq)
		if err != nil {
			return "", nil, fmt.Errorf("failed to call direct LLM for step execution: %w", err)
		}
	} else {
		return "", nil, fmt.Errorf("grafana-llm-app mode must be called from frontend")
	}

	var responseText strings.Builder
	var artifacts []Artifact
	var toolCalls []ToolCallChunk

	for chunk := range chunkChan {
		shouldContinue, _ := a.runManager.CheckPauseOrCancel(run.RunID)
		if !shouldContinue {
			return responseText.String(), artifacts, fmt.Errorf("step cancelled")
		}

		if chunk.Error != "" {
			return responseText.String(), artifacts, fmt.Errorf("LLM error: %s", chunk.Error)
		}

		if chunk.Chunk != "" {
			responseText.WriteString(chunk.Chunk)
			EmitAssistantDelta(run.EventChan, run.RunID, chunk.Chunk)
		}

		if chunk.ToolCall != nil {
			toolCalls = append(toolCalls, *chunk.ToolCall)
		}
	}

	finalText := responseText.String()

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

	textArtifacts := extractArtifactsFromText(finalText)
	artifacts = append(artifacts, textArtifacts...)

	return finalText, artifacts, nil
}

func extractArtifactsFromText(text string) []Artifact {
	var artifacts []Artifact

	promqlPattern := regexp.MustCompile(`(?m)^[a-z_]+\([^)]+\)(?:\{[^}]+\})?(?:\[[^\]]+\])?`)
	promqlMatches := promqlPattern.FindAllString(text, -1)
	for _, match := range promqlMatches {
		if len(match) > 10 { 
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

	logqlPattern := regexp.MustCompile(`\{[^}]+\}\s*(?:\|[^|]+)*`)
	logqlMatches := logqlPattern.FindAllString(text, -1)
	for _, match := range logqlMatches {
		if strings.Contains(match, "=") { 
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

func (a *App) buildFinalMessage(plan *ExecutionPlan, artifacts []Artifact) string {
	var message strings.Builder

	message.WriteString(fmt.Sprintf("**Goal:** %s\n\n", plan.Goal))

	message.WriteString("**Plan:**\n")
	for i, step := range plan.Steps {
		message.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Title))
	}
	message.WriteString("\n")

	if len(artifacts) > 0 {
		message.WriteString("**Artifacts:**\n")
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
	message.WriteString("The investigation has been completed according to the plan. Review the artifacts above and the detailed findings in each step.\n\n")

	message.WriteString("**Next Steps:**\n")
	message.WriteString("- Review the generated queries and artifacts\n")
	message.WriteString("- Drill down into specific findings\n")
	message.WriteString("- Ask follow-up questions if needed\n")

	return message.String()
}

func getMaxTokensForModel(model string, tokenLimit int) int {
	newerModels := []string{
		"gpt-4o-2024-11-20",
		"gpt-4o-2024-12-17",
		"chatgpt-4o-latest",
		"o1-preview",
		"o1-mini",
		"o1",
		"o3-mini",
		"o3",
	}

	modelLower := strings.ToLower(model)
	for _, newerModel := range newerModels {
		if strings.Contains(modelLower, newerModel) {
			return 0 
		}
	}

	return tokenLimit
}

func getMaxCompletionTokensForModel(model string, tokenLimit int) int {
	newerModels := []string{
		"gpt-4o-2024-11-20",
		"gpt-4o-2024-12-17",
		"chatgpt-4o-latest",
		"o1-preview",
		"o1-mini",
		"o1",
		"o3-mini",
		"o3",
	}

	modelLower := strings.ToLower(model)
	for _, newerModel := range newerModels {
		if strings.Contains(modelLower, newerModel) {
			return tokenLimit 
		}
	}

	return 0
}
