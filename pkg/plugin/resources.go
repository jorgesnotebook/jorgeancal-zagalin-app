package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func (a *App) handlePing(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"message": "ok"}`)); err != nil {
		sendErrorResponse(w, "Failed to write ping response", err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleEcho(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		sendErrorResponse(w, "Failed to decode echo request", err, http.StatusBadRequest)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		sendErrorResponse(w, "Failed to encode echo response", err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleGetSettings(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings := map[string]interface{}{
		"jsonData": map[string]interface{}{},
	}

	if a.settings != nil {
		settings["jsonData"] = map[string]interface{}{
			"llmBackend":             a.settings.LLMBackend,
			"llmProvider":            a.settings.LLMProvider,
			"llmModel":               a.settings.LLMModel,
			"maxRequestsPerMinute":   a.settings.MaxRequestsPerMinute,
			"monthlyBudgetUSD":       a.settings.MonthlyBudgetUSD,
			"contextRefreshMinutes":  a.settings.ContextRefreshMinutes,
		}
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		sendErrorResponse(w, "Failed to encode settings response", err, http.StatusInternalServerError)
		return
	}
}

func (a *App) handleSetupStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hasServiceAccountToken := false
	hasLLMConfig := false
	setupComplete := false

	if a.settings != nil {
		hasServiceAccountToken = a.settings.ServiceAccountToken != ""
		hasLLMConfig = a.settings.LLMBackend != "" || a.settings.LLMProvider != ""
	}

	setupComplete = hasServiceAccountToken || (a.settings != nil && a.settings.LLMBackend == "grafana-llm-app")

	status := map[string]interface{}{
		"setupComplete":           setupComplete,
		"hasServiceAccountToken":  hasServiceAccountToken,
		"hasLLMConfig":            hasLLMConfig,
		"llmBackend":              "",
		"recommendServiceAccount": !hasServiceAccountToken,
		"steps": []map[string]interface{}{
			{
				"id":          "service_account",
				"title":       "Create Service Account",
				"description": "Create a Grafana service account for backend authentication",
				"completed":   hasServiceAccountToken,
				"required":    true,
				"url":         "/admin/serviceaccounts",
			},
			{
				"id":          "configure_token",
				"title":       "Configure Token",
				"description": "Paste the service account token in plugin settings",
				"completed":   hasServiceAccountToken,
				"required":    true,
				"url":         "/plugins/jorgeancal-zagalin-app?page=configuration",
			},
			{
				"id":          "verify_llm",
				"title":       "Verify LLM Connection",
				"description": "Test the connection to grafana-llm-app",
				"completed":   setupComplete,
				"required":    true,
				"url":         "",
			},
		},
	}

	if a.settings != nil {
		status["llmBackend"] = a.settings.LLMBackend
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		sendErrorResponse(w, "Failed to encode setup status response", err, http.StatusInternalServerError)
		return
	}
}

// versionDetectionMiddleware extracts Grafana version from X-Grafana-Version header
// and caches it for use in health checks and logging
func (a *App) versionDetectionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Attempt to detect version from header (optional, non-blocking)
		version := a.versionDetector.DetectFromHeader(r)

		// Log warnings only on first successful detection
		if version.IsAvailable && !a.versionDetector.GetVersion().IsAvailable {
			a.versionDetector.LogVersionWarnings(r.Context())
		}

		// Continue to next handler regardless of version detection result
		next(w, r)
	}
}

func (a *App) registerRoutes(mux *http.ServeMux) {
	// Wrap handlers with version detection middleware
	mux.HandleFunc("/ping", a.versionDetectionMiddleware(a.handlePing))
	mux.HandleFunc("/echo", a.versionDetectionMiddleware(a.handleEcho))
	mux.HandleFunc("/settings", a.versionDetectionMiddleware(a.handleGetSettings))
	mux.HandleFunc("/health/setup", a.versionDetectionMiddleware(a.handleSetupStatus))
	mux.HandleFunc("/context/status", a.versionDetectionMiddleware(a.handleContextStatus))
	mux.HandleFunc("/context/refresh", a.versionDetectionMiddleware(a.handleContextRefresh))

	mux.HandleFunc("/query", a.versionDetectionMiddleware(a.handleQuery))
	mux.HandleFunc("/datasources", a.versionDetectionMiddleware(a.handleListDatasources))

	mux.HandleFunc("/llm/chat", a.versionDetectionMiddleware(a.handleLLMChat))

	mux.HandleFunc("/runs/start", a.versionDetectionMiddleware(a.handleStartRun))
	mux.HandleFunc("/runs/", a.versionDetectionMiddleware(a.handleRunRoutes))
}

func (a *App) handleContextStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := a.contextManager.GetContext()
	lastUpdated := a.contextManager.GetLastUpdated()

	status := map[string]interface{}{
		"lastUpdated": lastUpdated,
		"hasContext":  ctx != nil,
	}

	if ctx != nil {
		status["metrics"] = map[string]interface{}{
			"available":   ctx.Metrics != nil,
			"metricCount": 0,
			"labelCount":  0,
		}
		if ctx.Metrics != nil {
			status["metrics"] = map[string]interface{}{
				"available":   true,
				"metricCount": len(ctx.Metrics.MetricNames),
				"labelCount":  len(ctx.Metrics.Labels),
			}
		}

		status["logs"] = map[string]interface{}{
			"available":   ctx.Logs != nil,
			"streamCount": 0,
			"labelCount":  0,
		}
		if ctx.Logs != nil {
			status["logs"] = map[string]interface{}{
				"available":   true,
				"streamCount": len(ctx.Logs.Streams),
				"labelCount":  len(ctx.Logs.Labels),
			}
		}

		status["traces"] = map[string]interface{}{
			"available":      ctx.Traces != nil,
			"serviceCount":   0,
			"operationCount": 0,
		}
		if ctx.Traces != nil {
			status["traces"] = map[string]interface{}{
				"available":      true,
				"serviceCount":   len(ctx.Traces.Services),
				"operationCount": len(ctx.Traces.Operations),
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (a *App) handleContextRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(w, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("Rate limit exceeded for context refresh", "user", user.UserLogin)
			sendErrorResponse(w, "Rate limit exceeded", fmt.Errorf("too many requests"), http.StatusTooManyRequests)
			return
		}
	}

	backend.Logger.Info("Manual context refresh triggered",
		"user", user.UserLogin,
		"orgId", user.OrgID,
	)

	err = a.contextManager.Refresh(req.Context())
	if err != nil {
		backend.Logger.Error("Context refresh failed",
			"error", err,
			"user", user.UserLogin,
		)
		sendErrorResponse(w, "Context refresh failed", err, http.StatusInternalServerError)
		return
	}

	backend.Logger.Info("Context refresh completed successfully", "user", user.UserLogin)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Context refreshed successfully",
	})
}
