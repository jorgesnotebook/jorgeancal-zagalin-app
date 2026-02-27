package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type LLMClient struct {
	httpClient          *http.Client
	grafanaURL          string
	serviceAccountToken string
	logger              log.Logger
	retryDelays         []time.Duration // overridable in tests
}

func NewLLMClient(grafanaURL string, serviceAccountToken string, httpClient *http.Client, logger log.Logger) *LLMClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &LLMClient{
		httpClient:          httpClient,
		grafanaURL:          grafanaURL,
		serviceAccountToken: serviceAccountToken,
		logger:              logger,
		retryDelays:         llmRetryDelays,
	}
}

// CacheControl marks a content block as cacheable.
// Requires the underlying LLM provider (e.g. Anthropic Claude via grafana-llm-app)
// to support cache_control passthrough. Has no effect with OpenAI models.
// Dependency: grafana-llm-app must forward unknown message fields to the provider.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// ContentBlock is a structured content block used when cache_control is needed.
type ContentBlock struct {
	Type         string        `json:"type"`                    // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// AssistantMessage is a single chat message. Content holds the plain text for
// all internal operations. When ContentBlocks is non-empty it is marshaled as
// an array (required for Anthropic cache_control); otherwise Content is used.
//
// For the backend-only tool-calling loop (e.g. Slack bot):
//   - Set ToolCalls when role=="assistant" and the LLM wants to call tools.
//   - Set ToolCallID when role=="tool" to carry the tool result back.
type AssistantMessage struct {
	Role          string
	Content       string         // always the plain-text content
	ContentBlocks []ContentBlock // set only when cache_control is needed

	// Tool-calling loop (OpenAI format). Only used in backend-only agentic paths.
	ToolCalls  []openAIToolCall // non-nil → assistant wants to call tools
	ToolCallID string           // non-empty → this is a tool-result message
}

func (m AssistantMessage) MarshalJSON() ([]byte, error) {
	// Tool result: {"role":"tool","tool_call_id":"...","content":"..."}
	if m.ToolCallID != "" {
		return json.Marshal(struct {
			Role       string `json:"role"`
			ToolCallID string `json:"tool_call_id"`
			Content    string `json:"content"`
		}{Role: "tool", ToolCallID: m.ToolCallID, Content: m.Content})
	}

	// Assistant with tool_calls: {"role":"assistant","content":null,"tool_calls":[...]}
	if len(m.ToolCalls) > 0 {
		type assistantWithTools struct {
			Role      string          `json:"role"`
			Content   *string         `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		}
		return json.Marshal(assistantWithTools{Role: m.Role, Content: nil, ToolCalls: m.ToolCalls})
	}

	// ContentBlocks (Anthropic cache_control)
	if len(m.ContentBlocks) > 0 {
		return json.Marshal(struct {
			Role    string         `json:"role"`
			Content []ContentBlock `json:"content"`
		}{Role: m.Role, Content: m.ContentBlocks})
	}

	// Default: plain text
	return json.Marshal(struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: m.Role, Content: m.Content})
}

func (m *AssistantMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	// Try string content first (the common case).
	var s string
	if err := json.Unmarshal(raw.Content, &s); err == nil {
		m.Content = s
		return nil
	}
	// Fall back to content-block array (e.g. when cache_control is set).
	var blocks []ContentBlock
	if err := json.Unmarshal(raw.Content, &blocks); err != nil {
		return fmt.Errorf("unsupported message content format: %w", err)
	}
	m.ContentBlocks = blocks
	for _, b := range blocks {
		m.Content += b.Text
	}
	return nil
}

// isAnthropicModel returns true when the model name looks like an Anthropic
// Claude model. Used to decide whether to apply Anthropic-specific features
// such as prompt caching (cache_control). Has no effect on other providers.
func isAnthropicModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "claude")
}

type LLMStreamRequest struct {
	Model               string             `json:"model"`
	Messages            []AssistantMessage `json:"messages"`
	Temperature         float64            `json:"temperature"`
	MaxTokens           int                `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                `json:"max_completion_tokens,omitempty"`
	Tools               []Tool             `json:"tools,omitempty"`
	ToolChoice          string             `json:"tool_choice,omitempty"`
	Stream              bool               `json:"stream"`
	// FallbackModels is internal-only (not sent to the API). When the primary
	// Model returns a model-unavailable error, StreamChat retries with each
	// fallback in order and logs a warning when a fallback is used.
	FallbackModels []string `json:"-"`
}

type LLMStreamChunk struct {
	Chunk         string             `json:"chunk,omitempty"`
	Done          bool               `json:"done,omitempty"`
	Error         string             `json:"error,omitempty"`
	ErrorType     string             `json:"error_type,omitempty"`
	Retryable     bool               `json:"retryable,omitempty"`
	ToolCall      *ToolCallChunk     `json:"tool_call,omitempty"`
	HistoryUpdate []AssistantMessage `json:"history_update,omitempty"`
}

type ToolCallChunk struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIStreamChunk struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []openAIStreamChoice   `json:"choices"`
}

type openAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        openAIDelta        `json:"delta"`
	FinishReason *string            `json:"finish_reason"`
}

type openAIDelta struct {
	Content   string              `json:"content,omitempty"`
	ToolCalls []openAIToolCall    `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	Index    int                 `json:"index"`
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function openAIFunctionCall  `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// llmRetryDelays defines the wait before each successive attempt (len == maxAttempts-1).
var llmRetryDelays = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}

// isRetryableLLMStatus returns true for HTTP status codes that are safe to retry.
func isRetryableLLMStatus(status int) bool {
	return status == http.StatusTooManyRequests || (status >= 500 && status < 600)
}

// isRetryableLLMError returns true for network/timeout errors that are safe to retry.
// Context cancellation and deadline exceeded are NOT retried.
func isRetryableLLMError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// isModelUnavailableError returns true when the API response indicates the
// requested model does not exist or is unavailable, signalling that we should
// try the next model in the fallback chain rather than giving up.
func isModelUnavailableError(status int, body []byte) bool {
	if status != http.StatusBadRequest && status != http.StatusNotFound {
		return false
	}
	lower := strings.ToLower(string(body))
	return strings.Contains(lower, "model") &&
		(strings.Contains(lower, "not found") ||
			strings.Contains(lower, "does not exist") ||
			strings.Contains(lower, "unavailable") ||
			strings.Contains(lower, "not supported") ||
			strings.Contains(lower, "deprecated"))
}

func (c *LLMClient) StreamChat(ctx context.Context, req LLMStreamRequest, incomingReq *http.Request) (<-chan LLMStreamChunk, error) {
	llmURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions", c.grafanaURL)
	pluginCtx := backend.PluginConfigFromContext(ctx)
	corrID := incomingReq.Header.Get("X-Correlation-ID")
	maxAttempts := len(c.retryDelays) + 1

	// Build the ordered model chain: primary first, then fallbacks.
	models := append([]string{req.Model}, req.FallbackModels...)

	var resp *http.Response
	var lastErr error

	for modelIdx, modelName := range models {
		if modelIdx > 0 {
			c.logger.Warn("Primary model unavailable, trying fallback model",
				"failedModel", models[modelIdx-1],
				"fallbackModel", modelName,
				"correlationId", corrID,
			)
		}

		// Re-marshal the request with the current model name.
		modelReq := req
		modelReq.Model = modelName
		body, err := json.Marshal(modelReq)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		modelUnavailable := false

		for attempt := 0; attempt < maxAttempts; attempt++ {
			if attempt > 0 {
				delay := c.retryDelays[attempt-1]
				c.logger.Warn("Retrying LLM request after transient error",
					"attempt", attempt+1,
					"maxAttempts", maxAttempts,
					"delay", delay,
					"lastError", lastErr,
				)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
			}

			httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
			if err != nil {
				return nil, fmt.Errorf("failed to create request: %w", err)
			}

			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Accept", "text/event-stream")
			httpReq.Header.Set("X-Zagalin-Service", "backend")
			if corrID != "" {
				httpReq.Header.Set("X-Correlation-ID", corrID)
			}

			if c.serviceAccountToken != "" && c.serviceAccountToken != "existing-token-via-plugin-context" {
				httpReq.Header.Set("Authorization", "Bearer "+c.serviceAccountToken)
				c.logger.Debug("Using provisioned service account token for authentication")
			} else {
				grafanaConfig := backend.GrafanaConfigFromContext(ctx)
				if token, err := grafanaConfig.PluginAppClientSecret(); err == nil && token != "" {
					httpReq.Header.Set("Authorization", "Bearer "+token)
					c.logger.Debug("Using plugin context service account token for authentication")
				} else {
					c.logger.Warn("Service account token not available, trying fallback auth", "error", err)
				}
			}

			if pluginCtx.User != nil {
				if pluginCtx.User.Login != "" {
					httpReq.Header.Set("X-Grafana-User", pluginCtx.User.Login)
				}
				if pluginCtx.User.Email != "" {
					httpReq.Header.Set("X-Grafana-User-Email", pluginCtx.User.Email)
				}
				if pluginCtx.OrgID > 0 {
					httpReq.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", pluginCtx.OrgID))
				}
				c.logger.Debug("Forwarding user context",
					"user", pluginCtx.User.Login,
					"email", pluginCtx.User.Email,
					"orgId", pluginCtx.OrgID,
				)
			}

			if httpReq.Header.Get("Authorization") == "" {
				if authHeader := incomingReq.Header.Get("Authorization"); authHeader != "" {
					httpReq.Header.Set("Authorization", authHeader)
					c.logger.Debug("Fallback: forwarding Authorization header from incoming request")
				}
				if grafanaID := incomingReq.Header.Get("X-Grafana-Id"); grafanaID != "" {
					httpReq.Header.Set("X-Grafana-Id", grafanaID)
					c.logger.Debug("Fallback: forwarding X-Grafana-Id JWT")
				}
				if cookies := incomingReq.Header.Get("Cookie"); cookies != "" {
					httpReq.Header.Set("Cookie", cookies)
					c.logger.Debug("Fallback: forwarding cookies")
				}
			}

			c.logger.Info("Making LLM request",
				"url", llmURL,
				"model", modelName,
				"attempt", attempt+1,
				"hasServiceAccountToken", httpReq.Header.Get("Authorization") != "",
				"hasUserContext", httpReq.Header.Get("X-Grafana-User") != "",
				"correlationId", corrID,
			)

			resp, err = c.httpClient.Do(httpReq)
			if err != nil {
				if !isRetryableLLMError(err) {
					return nil, fmt.Errorf("failed to make request: %w", err)
				}
				lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				respBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var errorResponse map[string]interface{}
				errorMsg := string(respBody)
				if json.Unmarshal(respBody, &errorResponse) == nil {
					if msg, ok := errorResponse["message"].(string); ok {
						errorMsg = msg
					}
				}

				c.logger.Error("LLM API request failed",
					"status", resp.StatusCode,
					"response", string(respBody),
					"url", llmURL,
					"model", modelName,
					"attempt", attempt+1,
					"correlationId", corrID,
				)

				if resp.StatusCode == http.StatusUnauthorized {
					return nil, fmt.Errorf("authentication failed (401): %s\n\nPossible causes:\n1. Zagalin plugin needs a service account token to communicate with grafana-llm-app\n2. Configure a service account token in: Administration → Plugins → Zagalin → Settings → Service Account Token\n3. Alternatively, ensure grafana-llm-app is configured with LLM provider credentials\n4. Check Administration → Plugins → LLM App → Configuration", errorMsg)
				}

				// If this model is unavailable, advance to the next model in chain.
				if isModelUnavailableError(resp.StatusCode, respBody) {
					lastErr = fmt.Errorf("model %q unavailable: %s", modelName, errorMsg)
					modelUnavailable = true
					resp = nil
					break // break attempt loop; outer loop will try next model
				}

				if !isRetryableLLMStatus(resp.StatusCode) {
					return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))
				}

				lastErr = fmt.Errorf("attempt %d: LLM API returned status %d: %s", attempt+1, resp.StatusCode, errorMsg)
				resp = nil // mark this attempt as failed so the post-loop nil check works
				continue
			}

			// Success — break out of attempt loop
			break
		}

		if resp != nil {
			break // successful response — stop trying models
		}
		if !modelUnavailable {
			break // non-model failure — no point trying other models
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("LLM request failed after trying %d model(s): %w", len(models), lastErr)
	}

	chunkChan := make(chan LLMStreamChunk, 10)

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		c.logger.Debug("Starting to read SSE stream from grafana-llm-app")

		// readerDone signals the watchdog when this goroutine exits.
		readerDone := make(chan struct{})
		defer close(readerDone)

		// Watchdog: close resp.Body on context cancellation to unblock the
		// blocking reader.ReadString call below.
		go func() {
			select {
			case <-ctx.Done():
				c.logger.Info("LLM stream cancelled by client - closing body to unblock reader",
					"reason", ctx.Err(),
				)
				resp.Body.Close()
			case <-readerDone:
			}
		}()

		reader := bufio.NewReader(resp.Body)
		lineCount := 0

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if ctx.Err() != nil {
					// Context was cancelled; the watchdog closed the body.
					c.logger.Info("LLM stream cancelled by client",
						"linesRead", lineCount,
						"reason", ctx.Err(),
					)
					chunkChan <- LLMStreamChunk{Done: true}
				} else if err != io.EOF {
					c.logger.Error("Error reading stream", "error", err, "linesRead", lineCount)
					errType, retryable := ClassifyError(err)
					chunkChan <- LLMStreamChunk{
						Error:     fmt.Sprintf("Stream error: %v", err),
						ErrorType: string(errType),
						Retryable: retryable,
						Done:      true,
					}
				} else {
					c.logger.Debug("EOF reached", "linesRead", lineCount)
				}
				return
			}

			lineCount++
			c.logger.Debug("Read SSE line", "lineNumber", lineCount, "lineLength", len(line))

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				chunkChan <- LLMStreamChunk{Done: true}
				return
			}

			var openaiChunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
				c.logger.Warn("Failed to parse OpenAI SSE chunk", "dataLength", len(data), "error", err)
				continue
			}

			if len(openaiChunk.Choices) == 0 {
				continue
			}

			choice := openaiChunk.Choices[0]

			if choice.Delta.Content != "" {
				chunk := LLMStreamChunk{
					Chunk: choice.Delta.Content,
				}
				chunkChan <- chunk
				c.logger.Debug("Sent text chunk", "contentLength", len(choice.Delta.Content))
			}

			for _, toolCall := range choice.Delta.ToolCalls {
				chunk := LLMStreamChunk{
					ToolCall: &ToolCallChunk{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      toolCall.Function.Name,
							Arguments: toolCall.Function.Arguments,
						},
					},
				}
				chunkChan <- chunk
				c.logger.Debug("Sent tool call chunk", "id", toolCall.ID, "name", toolCall.Function.Name)
			}

			if choice.FinishReason != nil && *choice.FinishReason != "" {
				c.logger.Debug("Stream finished", "reason", *choice.FinishReason)
				chunkChan <- LLMStreamChunk{Done: true}
				return
			}
		}
	}()

	return chunkChan, nil
}

// SimpleChat makes a synchronous non-streaming LLM request and returns the complete response.
// This is used for query validation where streaming is not needed.
func (c *LLMClient) SimpleChat(ctx context.Context, req LLMStreamRequest) (string, error) {
	// Force stream to false for synchronous response
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	llmURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions", c.grafanaURL)
	corrID := CorrelationIDFromContext(ctx)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Zagalin-Service", "backend-validation")
	if corrID != "" {
		httpReq.Header.Set("X-Correlation-ID", corrID)
	}

	// Use service account token for authentication
	if c.serviceAccountToken != "" && c.serviceAccountToken != "existing-token-via-plugin-context" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceAccountToken)
		c.logger.Debug("Using service account token for validation request")
	} else {
		grafanaConfig := backend.GrafanaConfigFromContext(ctx)
		if token, err := grafanaConfig.PluginAppClientSecret(); err == nil && token != "" {
			httpReq.Header.Set("Authorization", "Bearer "+token)
			c.logger.Debug("Using plugin context token for validation request")
		} else {
			c.logger.Warn("No service account token available for validation", "error", err)
			return "", fmt.Errorf("service account token required for LLM validation")
		}
	}

	// Add plugin context if available
	pluginCtx := backend.PluginConfigFromContext(ctx)
	if pluginCtx.OrgID > 0 {
		httpReq.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", pluginCtx.OrgID))
	}

	c.logger.Debug("Making synchronous LLM validation request", "url", llmURL, "correlationId", corrID)

	maxAttempts := 3
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelays[attempt-1]
			c.logger.Warn("Retrying synchronous LLM request",
				"attempt", attempt+1,
				"maxAttempts", maxAttempts,
				"delay", delay,
				"lastError", lastErr,
			)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		// Rebuild the request each attempt (body reader is consumed after each Do)
		attemptReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		// Copy headers from the prepared request
		for k, vs := range httpReq.Header {
			for _, v := range vs {
				attemptReq.Header.Add(k, v)
			}
		}

		resp, err = c.httpClient.Do(attemptReq)
		if err != nil {
			if !isRetryableLLMError(err) {
				return "", fmt.Errorf("failed to make request: %w", err)
			}
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			c.logger.Error("LLM validation request failed",
				"status", resp.StatusCode,
				"response", string(respBody),
				"attempt", attempt+1,
			)
			if !isRetryableLLMStatus(resp.StatusCode) {
				return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))
			}
			lastErr = fmt.Errorf("attempt %d: LLM API returned status %d: %s", attempt+1, resp.StatusCode, string(respBody))
			resp = nil // mark failed so post-loop nil check works
			continue
		}

		// Success
		break
	}

	if resp == nil {
		return "", fmt.Errorf("LLM validation request failed after %d attempts: %w", maxAttempts, lastErr)
	}
	defer resp.Body.Close()

	// Parse OpenAI-style response
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(responseBody, &openAIResp); err != nil {
		c.logger.Warn("Failed to parse LLM response", "error", err, "body", string(responseBody))
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("LLM API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	content := openAIResp.Choices[0].Message.Content
	c.logger.Debug("Received LLM validation response", "length", len(content))

	return content, nil
}

// SimpleChatResult is the result of a non-streaming LLM call that may include
// tool-use requests alongside (or instead of) a text response.
type SimpleChatResult struct {
	Content   string
	ToolCalls []openAIToolCall // non-empty when the model wants to call tools
}

// SimpleChatFull is like SimpleChat but also returns any tool calls the model
// requested. It is used by backend-only agentic loops (e.g. the Slack bot)
// that need to drive the full tool-calling cycle without SSE.
func (c *LLMClient) SimpleChatFull(ctx context.Context, req LLMStreamRequest) (*SimpleChatResult, error) {
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	llmURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions", c.grafanaURL)
	corrID := CorrelationIDFromContext(ctx)

	// Build a template request for header copying; the body is re-created per
	// retry attempt because the reader is consumed after the first Do().
	templateReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	templateReq.Header.Set("Content-Type", "application/json")
	templateReq.Header.Set("Accept", "application/json")
	templateReq.Header.Set("X-Zagalin-Service", "slack-bot")
	if corrID != "" {
		templateReq.Header.Set("X-Correlation-ID", corrID)
	}
	if c.serviceAccountToken != "" && c.serviceAccountToken != "existing-token-via-plugin-context" {
		templateReq.Header.Set("Authorization", "Bearer "+c.serviceAccountToken)
	} else {
		grafanaConfig := backend.GrafanaConfigFromContext(ctx)
		if token, err := grafanaConfig.PluginAppClientSecret(); err == nil && token != "" {
			templateReq.Header.Set("Authorization", "Bearer "+token)
		}
	}

	maxAttempts := 3
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelays[attempt-1]
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		attemptReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		for k, vs := range templateReq.Header {
			for _, v := range vs {
				attemptReq.Header.Add(k, v)
			}
		}

		resp, err = c.httpClient.Do(attemptReq)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if !isRetryableLLMStatus(resp.StatusCode) {
				return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))
			}
			lastErr = fmt.Errorf("attempt %d: status %d: %s", attempt+1, resp.StatusCode, string(respBody))
			resp = nil
			continue
		}
		break
	}

	if resp == nil {
		return nil, fmt.Errorf("LLM request failed after %d attempts: %w", maxAttempts, lastErr)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content   *string          `json:"content"`
				ToolCalls []openAIToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("LLM API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	result := &SimpleChatResult{
		ToolCalls: parsed.Choices[0].Message.ToolCalls,
	}
	if parsed.Choices[0].Message.Content != nil {
		result.Content = *parsed.Choices[0].Message.Content
	}
	return result, nil
}
