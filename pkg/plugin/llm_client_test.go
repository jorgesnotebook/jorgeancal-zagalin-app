package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// mockRoundTripper implements http.RoundTripper for testing
type mockRoundTripper struct {
	response        *http.Response
	err             error
	capturedRequest *http.Request
	roundTripFunc   func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.capturedRequest = req

	if m.roundTripFunc != nil {
		return m.roundTripFunc(req)
	}

	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

// newMockHTTPClient creates an http.Client with a mock transport
func newMockHTTPClient(transport *mockRoundTripper) *http.Client {
	return &http.Client{Transport: transport}
}

// createMockSSEResponse creates a mock SSE response with given chunks
func createMockSSEResponse(chunks []string) *http.Response {
	var body bytes.Buffer
	for _, chunk := range chunks {
		body.WriteString("data: " + chunk + "\n\n")
	}
	body.WriteString("data: [DONE]\n\n")

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&body),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
}

// TestNewLLMClient tests client initialization
func TestNewLLMClient(t *testing.T) {
	tests := []struct {
		name                string
		grafanaURL          string
		serviceAccountToken string
		httpClient          *http.Client
		wantNil             bool
	}{
		{
			name:                "with all parameters",
			grafanaURL:          "http://localhost:3000",
			serviceAccountToken: "test-token",
			httpClient:          &http.Client{},
			wantNil:             false,
		},
		{
			name:                "with nil http client",
			grafanaURL:          "http://localhost:3000",
			serviceAccountToken: "test-token",
			httpClient:          nil,
			wantNil:             false,
		},
		{
			name:                "with empty token",
			grafanaURL:          "http://localhost:3000",
			serviceAccountToken: "",
			httpClient:          &http.Client{},
			wantNil:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewLLMClient(tt.grafanaURL, tt.serviceAccountToken, tt.httpClient, log.DefaultLogger)

			if (client == nil) != tt.wantNil {
				t.Errorf("NewLLMClient() nil = %v, want %v", client == nil, tt.wantNil)
			}

			if client != nil {
				if client.grafanaURL != tt.grafanaURL {
					t.Errorf("NewLLMClient() grafanaURL = %v, want %v", client.grafanaURL, tt.grafanaURL)
				}

				if client.serviceAccountToken != tt.serviceAccountToken {
					t.Errorf("NewLLMClient() token = %v, want %v", client.serviceAccountToken, tt.serviceAccountToken)
				}

				if client.httpClient == nil {
					t.Error("NewLLMClient() httpClient should not be nil")
				}
			}
		})
	}
}

// TestStreamChat tests SSE streaming
func TestStreamChat(t *testing.T) {
	tests := []struct {
		name       string
		response   *http.Response
		err        error
		wantErr    bool
		wantChunks int
	}{
		{
			name: "successful stream with text chunks",
			response: createMockSSEResponse([]string{
				`{"choices":[{"delta":{"content":"Hello"}}]}`,
				`{"choices":[{"delta":{"content":" world"}}]}`,
			}),
			wantErr:    false,
			wantChunks: 3, // 2 content chunks + 1 done
		},
		{
			name: "stream with tool call",
			response: createMockSSEResponse([]string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"query_prometheus","arguments":"{\"query\":\"up\"}"}}]}}]}`,
			}),
			wantErr:    false,
			wantChunks: 2, // 1 tool call + 1 done
		},
		{
			name: "stream with finish reason",
			response: createMockSSEResponse([]string{
				`{"choices":[{"delta":{"content":"Test"},"finish_reason":"stop"}]}`,
			}),
			wantErr:    false,
			wantChunks: 2, // 1 content + 1 done
		},
		{
			name: "empty stream",
			response: createMockSSEResponse([]string{}),
			wantErr:    false,
			wantChunks: 1, // just done
		},
		{
			name:     "HTTP error",
			response: nil, // overridden by roundTripFunc below
			wantErr:    true,
			wantChunks: 0,
		},
		{
			name:     "authentication error",
			response: &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
			},
			wantErr:    true,
			wantChunks: 0,
		},
		{
			name:       "network error",
			response:   nil,
			err:        fmt.Errorf("network error"),
			wantErr:    true,
			wantChunks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := &mockRoundTripper{
				response: tt.response,
				err:      tt.err,
			}

			// For retryable status codes (5xx), use roundTripFunc to return a fresh body per attempt.
			if tt.name == "HTTP error" {
				mockTransport.roundTripFunc = func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader(`{"error": "internal error"}`)),
					}, nil
				}
			}

			httpClient := &http.Client{
				Transport: mockTransport,
			}

			client := NewLLMClient("http://localhost:3000", "", httpClient, log.DefaultLogger)
			// Zero out retry delays so retryable-error tests don't slow down the suite.
			if tt.name == "HTTP error" || tt.name == "network error" {
				client.retryDelays = []time.Duration{0, 0}
			}

			ctx := context.Background()
			incomingReq := httptest.NewRequest("GET", "/", nil)

			chunks, err := client.StreamChat(ctx, LLMStreamRequest{
				Model:    "gpt-4",
				Messages: []AssistantMessage{{Role: "user", Content: "test"}},
			}, incomingReq)

			if (err != nil) != tt.wantErr {
				t.Errorf("StreamChat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				count := 0
				for chunk := range chunks {
					count++
					if chunk.Error != "" && !chunk.Done {
						t.Errorf("received chunk with error: %s", chunk.Error)
					}
				}

				if count != tt.wantChunks {
					t.Errorf("received %d chunks, want %d", count, tt.wantChunks)
				}
			}
		})
	}
}

// TestStreamChatAuthenticationHeaders tests auth header setup
func TestStreamChatAuthenticationHeaders(t *testing.T) {
	tests := []struct {
		name                string
		serviceAccountToken string
		pluginContext       *backend.PluginContext
		wantAuthHeader      bool
		wantUserHeaders     bool
	}{
		{
			name:                "with service account token",
			serviceAccountToken: "test-token",
			pluginContext: &backend.PluginContext{
				User:  &backend.User{Login: "testuser", Email: "test@example.com"},
				OrgID: 1,
			},
			wantAuthHeader:  true,
			wantUserHeaders: true,
		},
		{
			name:                "without service account token",
			serviceAccountToken: "",
			pluginContext: &backend.PluginContext{
				User:  &backend.User{Login: "testuser", Email: "test@example.com"},
				OrgID: 1,
			},
			wantAuthHeader:  false,
			wantUserHeaders: true,
		},
		{
			name:                "with empty user context",
			serviceAccountToken: "test-token",
			pluginContext: &backend.PluginContext{
				OrgID: 1,
			},
			wantAuthHeader:  true,
			wantUserHeaders: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTransport := &mockRoundTripper{
				response: createMockSSEResponse([]string{
					`{"choices":[{"delta":{"content":"test"}}]}`,
				}),
			}

			httpClient := &http.Client{
				Transport: mockTransport,
			}

			client := &LLMClient{
				httpClient:          httpClient,
				grafanaURL:          "http://localhost:3000",
				serviceAccountToken: tt.serviceAccountToken,
				logger:              log.DefaultLogger,
			}

			ctx := context.Background()
			if tt.pluginContext != nil {
				ctx = backend.WithPluginContext(ctx, *tt.pluginContext)
			}

			incomingReq := httptest.NewRequest("GET", "/", nil)

			_, err := client.StreamChat(ctx, LLMStreamRequest{
				Model:    "gpt-4",
				Messages: []AssistantMessage{{Role: "user", Content: "test"}},
			}, incomingReq)

			if err != nil {
				t.Fatalf("StreamChat() unexpected error: %v", err)
			}

			if mockTransport.capturedRequest != nil {
				hasAuthHeader := mockTransport.capturedRequest.Header.Get("Authorization") != ""
				if hasAuthHeader != tt.wantAuthHeader {
					t.Errorf("Authorization header present = %v, want %v", hasAuthHeader, tt.wantAuthHeader)
				}

				if tt.wantUserHeaders {
					if mockTransport.capturedRequest.Header.Get("X-Grafana-User") == "" {
						t.Error("Expected X-Grafana-User header to be set")
					}
					if mockTransport.capturedRequest.Header.Get("X-Grafana-Org-Id") == "" {
						t.Error("Expected X-Grafana-Org-Id header to be set")
					}
				}
			}
		})
	}
}

// TestStreamChatContextCancellationSlowStream verifies the reader goroutine
// exits promptly when the context is cancelled while no data is flowing
// (simulates a slow LLM that hasn't sent the first token yet).
func TestStreamChatContextCancellationSlowStream(t *testing.T) {
	// A pipe where the write end never writes — simulates a silent LLM.
	pr, pw := io.Pipe()
	defer pw.Close()

	mockHTTP := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       pr,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		},
	}

	client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)

	ctx, cancel := context.WithCancel(context.Background())
	incomingReq := httptest.NewRequest("GET", "/", nil)

	chunks, err := client.StreamChat(ctx, LLMStreamRequest{
		Model:    "gpt-4",
		Messages: []AssistantMessage{{Role: "user", Content: "test"}},
	}, incomingReq)
	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	// Cancel while no data is flowing; the goroutine must exit promptly.
	cancel()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-chunks:
			if !ok {
				return // channel closed — goroutine exited as expected
			}
		case <-deadline:
			t.Fatal("reader goroutine did not exit within 2s after context cancellation")
		}
	}
}

// TestStreamChatContextCancellation tests context cancellation
func TestStreamChatContextCancellation(t *testing.T) {
	// Create a response that would stream indefinitely
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for i := 0; i < 100; i++ {
			pw.Write([]byte(`data: {"choices":[{"delta":{"content":"test"}}]}` + "\n\n"))
		}
	}()

	mockHTTP := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       pr,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		},
	}

	client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)

	ctx, cancel := context.WithCancel(context.Background())
	incomingReq := httptest.NewRequest("GET", "/", nil)

	chunks, err := client.StreamChat(ctx, LLMStreamRequest{
		Model:    "gpt-4",
		Messages: []AssistantMessage{{Role: "user", Content: "test"}},
	}, incomingReq)

	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	// Read a few chunks
	count := 0
	for range chunks {
		count++
		if count == 3 {
			cancel() // Cancel after 3 chunks
		}
		if count > 10 {
			break // Safety limit
		}
	}

	if count < 3 {
		t.Errorf("Expected at least 3 chunks before cancellation, got %d", count)
	}
}

// TestStreamChatInvalidJSON tests handling of malformed JSON in SSE stream
func TestStreamChatInvalidJSON(t *testing.T) {
	var body bytes.Buffer
	body.WriteString("data: {invalid json}\n\n")
	body.WriteString("data: " + `{"choices":[{"delta":{"content":"valid"}}]}` + "\n\n")
	body.WriteString("data: [DONE]\n\n")

	mockHTTP := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&body),
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		},
	}

	client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)

	ctx := context.Background()
	incomingReq := httptest.NewRequest("GET", "/", nil)

	chunks, err := client.StreamChat(ctx, LLMStreamRequest{
		Model:    "gpt-4",
		Messages: []AssistantMessage{{Role: "user", Content: "test"}},
	}, incomingReq)

	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	count := 0
	for chunk := range chunks {
		count++
		if chunk.Chunk == "valid" {
			// Found the valid chunk, invalid JSON was skipped
			break
		}
	}

	if count == 0 {
		t.Error("Expected at least one valid chunk")
	}
}

// TestStreamChatEmptyChoices tests handling of chunks with no choices
func TestStreamChatEmptyChoices(t *testing.T) {
	var body bytes.Buffer
	body.WriteString("data: " + `{"choices":[]}` + "\n\n")
	body.WriteString("data: " + `{"choices":[{"delta":{"content":"test"}}]}` + "\n\n")
	body.WriteString("data: [DONE]\n\n")

	mockHTTP := &mockRoundTripper{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&body),
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		},
	}

	client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)

	ctx := context.Background()
	incomingReq := httptest.NewRequest("GET", "/", nil)

	chunks, err := client.StreamChat(ctx, LLMStreamRequest{
		Model:    "gpt-4",
		Messages: []AssistantMessage{{Role: "user", Content: "test"}},
	}, incomingReq)

	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	validChunks := 0
	for chunk := range chunks {
		if chunk.Chunk == "test" {
			validChunks++
		}
	}

	if validChunks != 1 {
		t.Errorf("Expected 1 valid chunk, got %d", validChunks)
	}
}

// TestStreamChatToolCalls tests handling of tool call chunks
func TestStreamChatToolCalls(t *testing.T) {
	tests := []struct {
		name         string
		chunks       []string
		wantToolCall bool
		wantFuncName string
	}{
		{
			name: "single tool call",
			chunks: []string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"query_prometheus","arguments":"{\"query\":\"up\"}"}}]}}]}`,
			},
			wantToolCall: true,
			wantFuncName: "query_prometheus",
		},
		{
			name: "multiple tool calls",
			chunks: []string{
				`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"query_prometheus","arguments":"{\"query\":\"up\"}"}}]}}]}`,
				`{"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_456","type":"function","function":{"name":"query_loki","arguments":"{\"query\":\"{job=\\\"api\\\"}\"}"}}]}}]}`,
			},
			wantToolCall: true,
			wantFuncName: "query_prometheus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := &mockRoundTripper{
				response: createMockSSEResponse(tt.chunks),
			}

			client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)

			ctx := context.Background()
			incomingReq := httptest.NewRequest("GET", "/", nil)

			chunks, err := client.StreamChat(ctx, LLMStreamRequest{
				Model:    "gpt-4",
				Messages: []AssistantMessage{{Role: "user", Content: "test"}},
			}, incomingReq)

			if err != nil {
				t.Fatalf("StreamChat() unexpected error: %v", err)
			}

			foundToolCall := false
			for chunk := range chunks {
				if chunk.ToolCall != nil {
					foundToolCall = true
					if chunk.ToolCall.Function.Name == tt.wantFuncName {
						break
					}
				}
			}

			if foundToolCall != tt.wantToolCall {
				t.Errorf("Found tool call = %v, want %v", foundToolCall, tt.wantToolCall)
			}
		})
	}
}

// TestStreamChatRequestConstruction tests correct HTTP request building
func TestStreamChatRequestConstruction(t *testing.T) {
	var capturedRequest *http.Request

	mockHTTP := &mockRoundTripper{
		response: createMockSSEResponse([]string{
			`{"choices":[{"delta":{"content":"test"}}]}`,
		}),
	}

	// Override RoundTrip to capture request
	mockHTTP.roundTripFunc = func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		return mockHTTP.response, nil
	}

	client := NewLLMClient("http://localhost:3000", "test-token", newMockHTTPClient(mockHTTP), log.DefaultLogger)

	ctx := context.Background()
	incomingReq := httptest.NewRequest("GET", "/", nil)

	req := LLMStreamRequest{
		Model:       "gpt-4",
		Messages:    []AssistantMessage{{Role: "user", Content: "test"}},
		Temperature: 0.7,
		MaxTokens:   1000,
		Stream:      true,
	}

	_, err := client.StreamChat(ctx, req, incomingReq)

	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	if capturedRequest == nil {
		t.Fatal("Expected request to be captured")
	}

	// Verify URL
	expectedURL := "http://localhost:3000/api/plugins/grafana-llm-app/resources/openai/v1/chat/completions"
	if capturedRequest.URL.String() != expectedURL {
		t.Errorf("Request URL = %v, want %v", capturedRequest.URL.String(), expectedURL)
	}

	// Verify headers
	if capturedRequest.Header.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type: application/json")
	}

	if capturedRequest.Header.Get("Accept") != "text/event-stream" {
		t.Error("Expected Accept: text/event-stream")
	}

	if capturedRequest.Header.Get("X-Zagalin-Service") != "backend" {
		t.Error("Expected X-Zagalin-Service: backend")
	}

	// Verify Authorization header
	if capturedRequest.Header.Get("Authorization") != "Bearer test-token" {
		t.Errorf("Authorization header = %v, want Bearer test-token", capturedRequest.Header.Get("Authorization"))
	}
}

// TestStreamChatFallbackAuthentication tests fallback auth header forwarding
func TestStreamChatFallbackAuthentication(t *testing.T) {
	var capturedRequest *http.Request

	mockHTTP := &mockRoundTripper{
		response: createMockSSEResponse([]string{
			`{"choices":[{"delta":{"content":"test"}}]}`,
		}),
	}

	mockHTTP.roundTripFunc = func(req *http.Request) (*http.Response, error) {
		capturedRequest = req
		return mockHTTP.response, nil
	}

	client := &LLMClient{
		httpClient:          newMockHTTPClient(mockHTTP),
		grafanaURL:          "http://localhost:3000",
		serviceAccountToken: "", // No service account token
		logger:              log.DefaultLogger,
	}

	ctx := context.Background()
	incomingReq := httptest.NewRequest("GET", "/", nil)
	incomingReq.Header.Set("Authorization", "Bearer incoming-token")
	incomingReq.Header.Set("X-Grafana-Id", "jwt-token")
	incomingReq.Header.Set("Cookie", "grafana_session=abc123")

	_, err := client.StreamChat(ctx, LLMStreamRequest{
		Model:    "gpt-4",
		Messages: []AssistantMessage{{Role: "user", Content: "test"}},
	}, incomingReq)

	if err != nil {
		t.Fatalf("StreamChat() unexpected error: %v", err)
	}

	if capturedRequest == nil {
		t.Fatal("Expected request to be captured")
	}

	// Verify fallback headers are forwarded
	if capturedRequest.Header.Get("Authorization") != "Bearer incoming-token" {
		t.Error("Expected fallback Authorization header to be forwarded")
	}

	if capturedRequest.Header.Get("X-Grafana-Id") != "jwt-token" {
		t.Error("Expected X-Grafana-Id to be forwarded")
	}

	if capturedRequest.Header.Get("Cookie") != "grafana_session=abc123" {
		t.Error("Expected Cookie to be forwarded")
	}
}

// TestStreamChatErrorResponse tests different HTTP error status codes
func TestStreamChatErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantErrContain string
	}{
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "internal server error"}`,
			wantErrContain: "500",
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "unauthorized"}`,
			wantErrContain: "authentication failed",
		},
		{
			name:           "429 Too Many Requests",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"error": "rate limit exceeded"}`,
			wantErrContain: "429",
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"message": "invalid request format"}`,
			wantErrContain: "400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusCode := tt.statusCode
			responseBody := tt.responseBody

			// For retryable statuses (5xx, 429) use roundTripFunc so each attempt
			// gets a fresh response body, and zero out delays to keep the suite fast.
			mockHTTP := &mockRoundTripper{
				roundTripFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: statusCode,
						Body:       io.NopCloser(strings.NewReader(responseBody)),
					}, nil
				},
			}

			client := NewLLMClient("http://localhost:3000", "", newMockHTTPClient(mockHTTP), log.DefaultLogger)
			if isRetryableLLMStatus(tt.statusCode) {
				client.retryDelays = []time.Duration{0, 0}
			}

			ctx := context.Background()
			incomingReq := httptest.NewRequest("GET", "/", nil)

			_, err := client.StreamChat(ctx, LLMStreamRequest{
				Model:    "gpt-4",
				Messages: []AssistantMessage{{Role: "user", Content: "test"}},
			}, incomingReq)

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErrContain) {
				t.Errorf("Error message %v should contain %v", err.Error(), tt.wantErrContain)
			}
		})
	}
}
