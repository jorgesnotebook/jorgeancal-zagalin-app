package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// LLMClient handles communication with grafana-llm-app
type LLMClient struct {
	httpClient          *http.Client
	grafanaURL          string
	serviceAccountToken string // Optional service account token for backend-to-backend auth
	logger              log.Logger
}

// NewLLMClient creates a new LLM client
func NewLLMClient(grafanaURL string, serviceAccountToken string, httpClient *http.Client, logger log.Logger) *LLMClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &LLMClient{
		httpClient:          httpClient,
		grafanaURL:          grafanaURL,
		serviceAccountToken: serviceAccountToken,
		logger:              logger,
	}
}

// AssistantMessage represents a chat message
type AssistantMessage struct {
	Role    string `json:"role"`    // "user" | "assistant" | "system"
	Content string `json:"content"`
}

// LLMStreamRequest represents a request to the LLM API
//
// Token Limit Compatibility:
// - MaxTokens: Used by older OpenAI models (gpt-4-turbo, gpt-3.5-turbo, gpt-4o-2024-08-06)
// - MaxCompletionTokens: Required by newer models (gpt-4o-2024-11-20+, o1-preview, o1-mini, o3)
//
// Both fields are sent with the same value for maximum compatibility.
// The receiving service (OpenAI API or grafana-llm-app) will use whichever field it supports.
type LLMStreamRequest struct {
	Model               string             `json:"model"`
	Messages            []AssistantMessage `json:"messages"`
	Temperature         float64            `json:"temperature"`
	MaxTokens           int                `json:"max_tokens,omitempty"`           // For older models
	MaxCompletionTokens int                `json:"max_completion_tokens,omitempty"` // For newer models (gpt-4o-2024-11-20+, o1, o3)
	Tools               []Tool             `json:"tools,omitempty"`
	ToolChoice          string             `json:"tool_choice,omitempty"`
	Stream              bool               `json:"stream"` // Enable SSE streaming
}

// LLMStreamChunk represents a chunk from the SSE stream
type LLMStreamChunk struct {
	Chunk   string `json:"chunk,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
	ToolCall *ToolCallChunk `json:"tool_call,omitempty"`
}

// ToolCallChunk represents a tool call in the stream
type ToolCallChunk struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// OpenAI streaming format structures
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

// StreamChat makes a streaming chat request to grafana-llm-app
// It uses service account authentication and forwards user context to ensure
// the LLM call executes with proper permissions
func (c *LLMClient) StreamChat(ctx context.Context, req LLMStreamRequest, incomingReq *http.Request) (<-chan LLMStreamChunk, error) {
	// Encode request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request (OpenAI-compatible endpoint)
	llmURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions", c.grafanaURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Add Zagalin service header for Traefik middleware authentication
	httpReq.Header.Set("X-Zagalin-Service", "backend")

	// Get plugin context for authentication
	pluginCtx := backend.PluginConfigFromContext(ctx)

	// Try multiple auth methods in order of preference:

	// 1. Use provisioned service account token (most reliable for backend-to-backend)
	if c.serviceAccountToken != "" && c.serviceAccountToken != "existing-token-via-plugin-context" {
		httpReq.Header.Set("Authorization", "Bearer "+c.serviceAccountToken)
		c.logger.Debug("Using provisioned service account token for authentication")
	} else {
		// 2. Try to get service account token from plugin context (if available)
		grafanaConfig := backend.GrafanaConfigFromContext(ctx)
		if token, err := grafanaConfig.PluginAppClientSecret(); err == nil && token != "" {
			httpReq.Header.Set("Authorization", "Bearer "+token)
			c.logger.Debug("Using plugin context service account token for authentication")
		} else {
			c.logger.Warn("Service account token not available, trying fallback auth", "error", err)
		}
	}

	// Forward user context from plugin context (not from HTTP headers)
	// This ensures the LLM call runs on behalf of the correct user
	if pluginCtx.User != nil {
		// Forward user login
		if pluginCtx.User.Login != "" {
			httpReq.Header.Set("X-Grafana-User", pluginCtx.User.Login)
		}

		// Forward user email
		if pluginCtx.User.Email != "" {
			httpReq.Header.Set("X-Grafana-User-Email", pluginCtx.User.Email)
		}

		// Forward org ID
		if pluginCtx.OrgID > 0 {
			httpReq.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", pluginCtx.OrgID))
		}

		c.logger.Debug("Forwarding user context",
			"user", pluginCtx.User.Login,
			"email", pluginCtx.User.Email,
			"orgId", pluginCtx.OrgID,
		)
	}

	// Fallback: try forwarding headers from incoming HTTP request if service account not available
	// This is less secure but may work in some Grafana configurations
	if httpReq.Header.Get("Authorization") == "" {
		// Try forwarding Authorization header from incoming request (if present)
		if authHeader := incomingReq.Header.Get("Authorization"); authHeader != "" {
			httpReq.Header.Set("Authorization", authHeader)
			c.logger.Debug("Fallback: forwarding Authorization header from incoming request")
		}

		// Forward X-Grafana-Id JWT for user identification
		if grafanaID := incomingReq.Header.Get("X-Grafana-Id"); grafanaID != "" {
			httpReq.Header.Set("X-Grafana-Id", grafanaID)
			c.logger.Debug("Fallback: forwarding X-Grafana-Id JWT")
		}

		// Forward cookies for session-based auth
		if cookies := incomingReq.Header.Get("Cookie"); cookies != "" {
			httpReq.Header.Set("Cookie", cookies)
			c.logger.Debug("Fallback: forwarding cookies")
		}
	}

	// Log authentication status for debugging
	c.logger.Info("Making LLM request",
		"url", llmURL,
		"hasServiceAccountToken", httpReq.Header.Get("Authorization") != "",
		"hasUserContext", httpReq.Header.Get("X-Grafana-User") != "",
		"hasUserEmail", httpReq.Header.Get("X-Grafana-User-Email") != "",
		"hasOrgId", httpReq.Header.Get("X-Grafana-Org-Id") != "",
	)

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Parse the error response to give better guidance
		var errorResponse map[string]interface{}
		errorMsg := string(body)
		if json.Unmarshal(body, &errorResponse) == nil {
			if msg, ok := errorResponse["message"].(string); ok {
				errorMsg = msg
			}
		}

		c.logger.Error("LLM API request failed",
			"status", resp.StatusCode,
			"response", string(body),
			"url", llmURL,
			"hadXGrafanaId", incomingReq.Header.Get("X-Grafana-Id") != "",
			"hadCookie", incomingReq.Header.Get("Cookie") != "",
			"hadAuthorization", incomingReq.Header.Get("Authorization") != "",
		)

		// Provide helpful error message based on status
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("authentication failed (401): %s\n\nPossible causes:\n1. Zagalin plugin needs a service account token to communicate with grafana-llm-app\n2. Configure a service account token in: Administration → Plugins → Zagalin → Settings → Service Account Token\n3. Alternatively, ensure grafana-llm-app is configured with LLM provider credentials\n4. Check Administration → Plugins → LLM App → Configuration", errorMsg)
		}

		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create channel for streaming chunks
	chunkChan := make(chan LLMStreamChunk, 10)

	// Start goroutine to read SSE stream
	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		c.logger.Debug("Starting to read SSE stream from grafana-llm-app")

		reader := bufio.NewReader(resp.Body)
		lineCount := 0

		for {
			// Check context cancellation
			select {
			case <-ctx.Done():
				c.logger.Debug("Stream context cancelled")
				return
			default:
			}

			// Read line
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					c.logger.Error("Error reading stream", "error", err, "linesRead", lineCount)
					chunkChan <- LLMStreamChunk{
						Error: fmt.Sprintf("Stream error: %v", err),
						Done:  true,
					}
				} else {
					c.logger.Debug("EOF reached", "linesRead", lineCount)
				}
				return
			}

			lineCount++
			c.logger.Debug("Read SSE line", "lineNumber", lineCount, "lineLength", len(line))

			// Parse SSE line
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// SSE format: "data: <json>"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Check for "[DONE]" marker
			if data == "[DONE]" {
				chunkChan <- LLMStreamChunk{Done: true}
				return
			}

			// Parse OpenAI format chunk
			var openaiChunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &openaiChunk); err != nil {
				c.logger.Warn("Failed to parse OpenAI SSE chunk", "dataLength", len(data), "error", err)
				continue
			}

			// Convert OpenAI format to our format
			if len(openaiChunk.Choices) == 0 {
				continue
			}

			choice := openaiChunk.Choices[0]

			// Handle text content
			if choice.Delta.Content != "" {
				chunk := LLMStreamChunk{
					Chunk: choice.Delta.Content,
				}
				chunkChan <- chunk
				c.logger.Debug("Sent text chunk", "contentLength", len(choice.Delta.Content))
			}

			// Handle tool calls
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

			// Check if stream is finished
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				c.logger.Debug("Stream finished", "reason", *choice.FinishReason)
				chunkChan <- LLMStreamChunk{Done: true}
				return
			}
		}
	}()

	return chunkChan, nil
}
