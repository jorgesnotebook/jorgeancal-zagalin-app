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

type LLMClient struct {
	httpClient          *http.Client
	grafanaURL          string
	serviceAccountToken string 
	logger              log.Logger
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
	}
}

type AssistantMessage struct {
	Role    string `json:"role"`    
	Content string `json:"content"`
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
}

type LLMStreamChunk struct {
	Chunk   string `json:"chunk,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Error   string `json:"error,omitempty"`
	ToolCall *ToolCallChunk `json:"tool_call,omitempty"`
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

func (c *LLMClient) StreamChat(ctx context.Context, req LLMStreamRequest, incomingReq *http.Request) (<-chan LLMStreamChunk, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	llmURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions", c.grafanaURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpReq.Header.Set("X-Zagalin-Service", "backend")

	pluginCtx := backend.PluginConfigFromContext(ctx)


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
		"hasServiceAccountToken", httpReq.Header.Get("Authorization") != "",
		"hasUserContext", httpReq.Header.Get("X-Grafana-User") != "",
		"hasUserEmail", httpReq.Header.Get("X-Grafana-User-Email") != "",
		"hasOrgId", httpReq.Header.Get("X-Grafana-Org-Id") != "",
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

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

		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("authentication failed (401): %s\n\nPossible causes:\n1. Zagalin plugin needs a service account token to communicate with grafana-llm-app\n2. Configure a service account token in: Administration → Plugins → Zagalin → Settings → Service Account Token\n3. Alternatively, ensure grafana-llm-app is configured with LLM provider credentials\n4. Check Administration → Plugins → LLM App → Configuration", errorMsg)
		}

		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	chunkChan := make(chan LLMStreamChunk, 10)

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		c.logger.Debug("Starting to read SSE stream from grafana-llm-app")

		reader := bufio.NewReader(resp.Body)
		lineCount := 0

		for {
			select {
			case <-ctx.Done():
				c.logger.Debug("Stream context cancelled")
				return
			default:
			}

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
