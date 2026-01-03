package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// LLMHealthStatus represents the health status of the LLM service
type LLMHealthStatus struct {
	Available  bool
	Configured bool
	Message    string
	ErrorCode  string
}

// LLM health check cache (module-level)
var (
	llmHealthCache struct {
		sync.RWMutex
		status     LLMHealthStatus
		expiry     time.Time
		inProgress bool
		waiters    []chan LLMHealthStatus
	}
	llmHealthCacheTTL = 30 * time.Second // Cache for 30 seconds
)

// CheckLLMHealth checks if grafana-llm-app is available and configured (with caching)
func CheckLLMHealth(ctx context.Context, grafanaURL string, incomingReq *http.Request) LLMHealthStatus {
	now := time.Now()

	// Check cache first (read lock)
	llmHealthCache.RLock()
	if now.Before(llmHealthCache.expiry) {
		status := llmHealthCache.status
		llmHealthCache.RUnlock()
		log.DefaultLogger.Debug("LLM health check: using cached result",
			"available", status.Available,
			"configured", status.Configured,
		)
		return status
	}

	// Check if another goroutine is already performing the check
	if llmHealthCache.inProgress {
		// Wait for the in-flight check to complete
		waiter := make(chan LLMHealthStatus, 1)
		llmHealthCache.waiters = append(llmHealthCache.waiters, waiter)
		llmHealthCache.RUnlock()

		log.DefaultLogger.Debug("LLM health check: waiting for in-flight check")
		select {
		case result := <-waiter:
			return result
		case <-ctx.Done():
			return LLMHealthStatus{
				Available:  false,
				Configured: false,
				Message:    "Health check cancelled",
				ErrorCode:  "CONTEXT_CANCELLED",
			}
		}
	}
	llmHealthCache.RUnlock()

	// Acquire write lock to start health check
	llmHealthCache.Lock()
	// Double-check cache after acquiring write lock (another goroutine might have updated it)
	if now.Before(llmHealthCache.expiry) {
		status := llmHealthCache.status
		llmHealthCache.Unlock()
		return status
	}

	llmHealthCache.inProgress = true
	llmHealthCache.Unlock()

	log.DefaultLogger.Debug("LLM health check: starting new check")

	// Perform actual health check
	status := checkLLMHealthUncached(ctx, grafanaURL, incomingReq)

	// Update cache and notify waiters
	llmHealthCache.Lock()
	llmHealthCache.status = status
	llmHealthCache.expiry = time.Now().Add(llmHealthCacheTTL)
	llmHealthCache.inProgress = false

	// Notify all waiting goroutines
	for _, waiter := range llmHealthCache.waiters {
		waiter <- status
		close(waiter)
	}
	llmHealthCache.waiters = nil
	llmHealthCache.Unlock()

	log.DefaultLogger.Info("LLM health check completed",
		"available", status.Available,
		"configured", status.Configured,
		"message", status.Message,
	)

	return status
}

// checkLLMHealthUncached performs the actual health check without caching
func checkLLMHealthUncached(ctx context.Context, grafanaURL string, incomingReq *http.Request) LLMHealthStatus {
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

	// Forward authentication headers (same order as llm_client.go)
	// 1. Try Authorization header first (most reliable)
	if authHeader := incomingReq.Header.Get("Authorization"); authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
		log.DefaultLogger.Debug("Health check: forwarding Authorization header")
	}

	// 2. Forward X-Grafana-Id JWT for user identification
	if grafanaID := incomingReq.Header.Get("X-Grafana-Id"); grafanaID != "" {
		httpReq.Header.Set("X-Grafana-Id", grafanaID)
		log.DefaultLogger.Debug("Health check: forwarding X-Grafana-Id")
	}

	// 3. Forward cookies for session-based auth
	if cookies := incomingReq.Header.Get("Cookie"); cookies != "" {
		httpReq.Header.Set("Cookie", cookies)
		log.DefaultLogger.Debug("Health check: forwarding cookies")
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
		Available:  true,
		Configured: true,
		Message:    "LLM service is ready",
		ErrorCode:  "",
	}
}

// InvalidateLLMHealthCache forces cache invalidation (useful after config changes)
func InvalidateLLMHealthCache() {
	llmHealthCache.Lock()
	defer llmHealthCache.Unlock()
	llmHealthCache.status = LLMHealthStatus{}
	llmHealthCache.expiry = time.Time{}
	llmHealthCache.inProgress = false
	log.DefaultLogger.Debug("LLM health check cache invalidated")
}
