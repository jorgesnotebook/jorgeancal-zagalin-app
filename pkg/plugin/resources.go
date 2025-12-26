package plugin

import (
	"encoding/json"
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

func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ping", a.handlePing)
	mux.HandleFunc("/echo", a.handleEcho)
	mux.HandleFunc("/context/status", a.handleContextStatus)
	mux.HandleFunc("/context/refresh", a.handleContextRefresh)

	mux.HandleFunc("/query", a.handleQuery)

	mux.HandleFunc("/storage/conversations", a.handleGetConversations)
	mux.HandleFunc("/storage/conversation", a.handleGetConversation)
	mux.HandleFunc("/storage/conversation/save", a.handleSaveConversation)
	mux.HandleFunc("/storage/conversation/delete", a.handleDeleteConversation)
	mux.HandleFunc("/storage/conversation/title", a.handleUpdateConversationTitle)
	mux.HandleFunc("/storage/conversation/pin", a.handleTogglePin)
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

	backend.Logger.Info("Manual context refresh triggered")

	err := a.contextManager.Refresh(req.Context())
	if err != nil {
		sendErrorResponse(w, "Context refresh failed", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Context refreshed successfully",
	})
}
