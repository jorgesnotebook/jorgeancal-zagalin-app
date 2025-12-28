package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// LLMHealthStatus represents the health status of the LLM service
type LLMHealthStatus struct {
	Available bool
	Configured bool
	Message string
	ErrorCode string
}

// CheckLLMHealth checks if grafana-llm-app is available and configured
func CheckLLMHealth(ctx context.Context, grafanaURL string, incomingReq *http.Request) LLMHealthStatus {
	// Create a test request to grafana-llm-app
	testURL := fmt.Sprintf("%s/api/plugins/grafana-llm-app/health", grafanaURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return LLMHealthStatus{
			Available: false,
			Configured: false,
			Message: "Failed to create health check request",
			ErrorCode: "HEALTH_CHECK_ERROR",
		}
	}

	// Forward authentication headers
	if grafanaID := incomingReq.Header.Get("X-Grafana-Id"); grafanaID != "" {
		httpReq.Header.Set("X-Grafana-Id", grafanaID)
	}
	if cookies := incomingReq.Header.Get("Cookie"); cookies != "" {
		httpReq.Header.Set("Cookie", cookies)
	}

	// Make request with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.DefaultLogger.Warn("LLM health check failed", "error", err, "url", testURL)
		return LLMHealthStatus{
			Available: false,
			Configured: false,
			Message: "grafana-llm-app plugin is not available. Please install it from the Grafana plugin catalog.",
			ErrorCode: "LLM_PLUGIN_NOT_INSTALLED",
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode == 404 {
		return LLMHealthStatus{
			Available: false,
			Configured: false,
			Message: "grafana-llm-app plugin is not installed. Please install it from Administration → Plugins → search for 'LLM App'",
			ErrorCode: "LLM_PLUGIN_NOT_INSTALLED",
		}
	}

	if resp.StatusCode == 401 {
		// 401 could mean either:
		// 1. Authentication issue (our headers aren't being forwarded correctly)
		// 2. LLM plugin is not configured
		// Let's be optimistic and assume it's configured but there's an auth issue
		log.DefaultLogger.Warn("LLM health check returned 401 - assuming auth forwarding issue, will proceed with LLM call",
			"response", string(body),
			"hasXGrafanaId", incomingReq.Header.Get("X-Grafana-Id") != "",
			"hasCookie", incomingReq.Header.Get("Cookie") != "",
		)
		// Return as available and configured, let the actual LLM call handle auth
		return LLMHealthStatus{
			Available: true,
			Configured: true, // Assume configured, auth might be the issue
			Message: "LLM service detected (auth verification skipped)",
			ErrorCode: "",
		}
	}

	if resp.StatusCode != 200 {
		log.DefaultLogger.Warn("LLM health check returned non-200 status", "status", resp.StatusCode, "body", string(body))
		return LLMHealthStatus{
			Available: true,
			Configured: false,
			Message: fmt.Sprintf("grafana-llm-app returned status %d. Please check the plugin configuration.", resp.StatusCode),
			ErrorCode: "LLM_PLUGIN_ERROR",
		}
	}

	// Try to parse the health response
	var healthResp map[string]interface{}
	if err := json.Unmarshal(body, &healthResp); err == nil {
		// Check if it indicates configuration issues
		if msg, ok := healthResp["message"].(string); ok {
			if msg == "OK" || msg == "ok" {
				return LLMHealthStatus{
					Available: true,
					Configured: true,
					Message: "LLM service is ready",
					ErrorCode: "",
				}
			}
		}
	}

	// Default to configured if health check passed
	return LLMHealthStatus{
		Available: true,
		Configured: true,
		Message: "LLM service is ready",
		ErrorCode: "",
	}
}
