package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// JSON-RPC 2.0 envelope types.

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP result types.

type mcpInitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      mcpServerInfo          `json:"serverInfo"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type mcpCallToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// mcpToolHandlers maps MCP tool names to their App method implementations.
var mcpToolHandlers = map[string]func(*App, context.Context, map[string]interface{}, *UserIdentity) (string, error){
	"execute_promql":    (*App).executePromQL,
	"execute_logql":     (*App).executeLogQL,
	"execute_traceql":   (*App).executeTraceQL,
	"get_firing_alerts": (*App).getFiringAlerts,
	"search_dashboards": (*App).searchDashboards,
	"get_dashboard":     (*App).getDashboard,
	"get_annotations":   (*App).getAnnotations,
	"list_folders":      (*App).listFolders,
}

// handleMCP is the HTTP Streamable MCP endpoint (protocol version 2024-11-05).
// It accepts a single POST with a JSON-RPC 2.0 request and returns a JSON-RPC response.
func (a *App) handleMCP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := extractUserIdentity(req)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}

	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			backend.Logger.Warn("MCP: rate limit exceeded", "user", user.UserLogin)
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	var rpcReq mcpRequest
	if err := json.NewDecoder(req.Body).Decode(&rpcReq); err != nil {
		writeMCPError(w, nil, -32700, "parse error")
		return
	}

	// Notifications (id == null) — acknowledge without a body.
	if rpcReq.ID == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	result, rpcErr := a.dispatchMCP(req.Context(), rpcReq.Method, rpcReq.Params, user)
	if rpcErr != nil {
		writeMCPError(w, rpcReq.ID, rpcErr.Code, rpcErr.Message)
		return
	}

	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      rpcReq.ID,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// dispatchMCP routes a JSON-RPC method to its handler and returns the result or an error.
func (a *App) dispatchMCP(ctx context.Context, method string, rawParams json.RawMessage, user *UserIdentity) (interface{}, *mcpError) {
	switch method {
	case "initialize":
		return mcpInitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			ServerInfo: mcpServerInfo{
				Name:    "zagalin",
				Version: "1.0.0",
			},
		}, nil

	case "ping":
		return map[string]interface{}{}, nil

	case "tools/list":
		return a.mcpToolsList(), nil

	case "tools/call":
		return a.mcpCallTool(ctx, rawParams, user)

	default:
		return nil, &mcpError{Code: -32601, Message: fmt.Sprintf("method not found: %s", method)}
	}
}

// mcpToolsList converts GetTools output into the MCP tools/list response.
func (a *App) mcpToolsList() mcpToolsListResult {
	datasources := a.getCachedDatasources()
	tools := GetTools(true, a.settings, datasources)

	result := make([]mcpTool, 0, len(tools))
	for _, t := range tools {
		// Only expose tools that have a backend execution handler.
		if _, ok := mcpToolHandlers[t.Function.Name]; !ok {
			continue
		}
		result = append(result, mcpTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}
	return mcpToolsListResult{Tools: result}
}

// mcpCallTool executes a named tool with the provided arguments.
func (a *App) mcpCallTool(ctx context.Context, rawParams json.RawMessage, user *UserIdentity) (mcpCallToolResult, *mcpError) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return mcpCallToolResult{}, &mcpError{Code: -32602, Message: "invalid params"}
	}

	handler, ok := mcpToolHandlers[params.Name]
	if !ok {
		return mcpCallToolResult{}, &mcpError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", params.Name)}
	}

	if params.Arguments == nil {
		params.Arguments = map[string]interface{}{}
	}

	backend.Logger.Info("MCP tool call", "tool", params.Name, "user", user.UserLogin)

	result, err := handler(a, ctx, params.Arguments, user)
	if err != nil {
		backend.Logger.Error("MCP tool call failed", "tool", params.Name, "error", err, "user", user.UserLogin)
		return mcpCallToolResult{
			IsError: true,
			Content: []mcpContent{{Type: "text", Text: err.Error()}},
		}, nil
	}

	return mcpCallToolResult{
		Content: []mcpContent{{Type: "text", Text: result}},
	}, nil
}

// writeMCPError writes a JSON-RPC 2.0 error response.
func writeMCPError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &mcpError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
