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

	// Return plugin settings (jsonData only, no secure fields)
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

func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ping", a.handlePing)
	mux.HandleFunc("/echo", a.handleEcho)
	mux.HandleFunc("/settings", a.handleGetSettings)
	mux.HandleFunc("/context/status", a.handleContextStatus)
	mux.HandleFunc("/context/refresh", a.handleContextRefresh)

	mux.HandleFunc("/query", a.handleQuery)
	mux.HandleFunc("/datasources", a.handleListDatasources)

	mux.HandleFunc("/llm/chat", a.handleLLMChat)

	// Run-based orchestration endpoints
	mux.HandleFunc("/runs/start", a.handleStartRun)
	mux.HandleFunc("/runs/", a.handleRunRoutes)
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

	// Extract user identity for authentication and rate limiting
	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(w, "Authentication required", err, http.StatusUnauthorized)
		return
	}

	// Apply rate limiting per user (reuse existing guardrails)
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
