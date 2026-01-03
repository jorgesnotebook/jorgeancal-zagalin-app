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

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// DirectLLMClient handles direct communication with LLM providers (OpenAI, Anthropic, etc.)
type DirectLLMClient struct {
	httpClient   *http.Client
	provider     string // "openai" | "anthropic" | "azure-openai"
	model        string
	endpoint     string
	apiKey       string
	organization string // OpenAI Organization ID (optional)
	logger       log.Logger
}

// NewDirectLLMClient creates a new direct LLM client
func NewDirectLLMClient(provider, model, endpoint, apiKey, organization string, httpClient *http.Client, logger log.Logger) *DirectLLMClient {
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	// Set default endpoints
	if endpoint == "" {
		switch provider {
		case "openai":
			endpoint = "https://api.openai.com/v1/chat/completions"
		case "anthropic":
			endpoint = "https://api.anthropic.com/v1/messages"
		}
	}

	return &DirectLLMClient{
		httpClient:   httpClient,
		provider:     provider,
		model:        model,
		endpoint:     endpoint,
		apiKey:       apiKey,
		organization: organization,
		logger:       logger,
	}
}

// StreamChat makes a streaming chat request directly to the LLM provider
func (c *DirectLLMClient) StreamChat(ctx context.Context, req LLMStreamRequest) (<-chan LLMStreamChunk, error) {
	switch c.provider {
	case "openai", "azure-openai":
		return c.streamOpenAI(ctx, req)
	case "anthropic":
		return c.streamAnthropic(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", c.provider)
	}
}

// streamOpenAI handles OpenAI API streaming
func (c *DirectLLMClient) streamOpenAI(ctx context.Context, req LLMStreamRequest) (<-chan LLMStreamChunk, error) {
	// Convert to OpenAI format
	openAIReq := map[string]interface{}{
		"model":       c.model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"stream":      true,
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		openAIReq["tools"] = req.Tools
		if req.ToolChoice != "" {
			openAIReq["tool_choice"] = req.ToolChoice
		}
	}

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for OpenAI
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Set OpenAI-Organization header if provided
	if c.organization != "" {
		httpReq.Header.Set("OpenAI-Organization", c.organization)
	}

	c.logger.Info("Making direct LLM request",
		"provider", c.provider,
		"model", c.model,
		"endpoint", c.endpoint,
		"hasOrganization", c.organization != "",
	)

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		c.logger.Error("LLM API request failed",
			"status", resp.StatusCode,
			"response", string(body),
		)
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Create channel for streaming chunks
	chunkChan := make(chan LLMStreamChunk, 10)

	// Start goroutine to read SSE stream
	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

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
					c.logger.Error("Error reading stream", "error", err)
					chunkChan <- LLMStreamChunk{
						Error: fmt.Sprintf("Stream error: %v", err),
						Done:  true,
					}
				}
				return
			}

			// Parse SSE line
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// OpenAI SSE format: "data: <json>"
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Check for "[DONE]" marker
			if data == "[DONE]" {
				chunkChan <- LLMStreamChunk{Done: true}
				return
			}

			// Parse JSON chunk
			var openAIChunk struct {
				Choices []struct {
					Delta struct {
						Content    string `json:"content"`
						ToolCalls  []struct {
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &openAIChunk); err != nil {
				c.logger.Warn("Failed to parse SSE chunk", "data", data, "error", err)
				continue
			}

			// Convert to our format
			if len(openAIChunk.Choices) > 0 {
				choice := openAIChunk.Choices[0]

				// Handle text content
				if choice.Delta.Content != "" {
					chunkChan <- LLMStreamChunk{
						Chunk: choice.Delta.Content,
					}
				}

				// Handle tool calls
				if len(choice.Delta.ToolCalls) > 0 {
					for _, tc := range choice.Delta.ToolCalls {
						chunkChan <- LLMStreamChunk{
							ToolCall: &ToolCallChunk{
								ID:   tc.ID,
								Type: tc.Type,
								Function: struct {
									Name      string `json:"name"`
									Arguments string `json:"arguments"`
								}{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							},
						}
					}
				}

				// Check if done
				if choice.FinishReason != "" {
					chunkChan <- LLMStreamChunk{Done: true}
					return
				}
			}
		}
	}()

	return chunkChan, nil
}

// streamAnthropic handles Anthropic API streaming
func (c *DirectLLMClient) streamAnthropic(ctx context.Context, req LLMStreamRequest) (<-chan LLMStreamChunk, error) {
	// TODO: Implement Anthropic streaming
	// For now, return error
	return nil, fmt.Errorf("anthropic streaming not yet implemented")
}
