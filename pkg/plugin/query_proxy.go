package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type QueryRequest struct {
	Datasource string          `json:"datasource"`
	Queries    []QueryPayload  `json:"queries"`
	TimeRange  TimeRange       `json:"timeRange"`
}

type QueryPayload struct {
	RefID         string                 `json:"refId"`
	DatasourceUID string                 `json:"datasourceUid,omitempty"`
	QueryType     string                 `json:"queryType,omitempty"`
	Expr          string                 `json:"expr,omitempty"`
	Query         string                 `json:"query,omitempty"`
	IntervalMs    int64                  `json:"intervalMs,omitempty"`
	MaxDataPoints int64                  `json:"maxDataPoints,omitempty"`
	Format        string                 `json:"format,omitempty"`
	Additional    map[string]interface{} `json:"-"`
}

type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type QueryResponse struct {
	Results map[string]QueryResult `json:"results"`
}

type QueryResult struct {
	RefID  string                 `json:"refId"`
	Frames []interface{}          `json:"frames"`
	Error  string                 `json:"error,omitempty"`
	Meta   map[string]interface{} `json:"meta,omitempty"`
}

type UserIdentity struct {
	UserID    int64  `json:"userId"`
	UserLogin string `json:"userLogin"`
	UserEmail string `json:"userEmail"`
	OrgID     int64  `json:"orgId"`
	OrgName   string `json:"orgName"`
}

func hashUserLogin(login string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(login))
	return h.Sum32()
}

func extractUserIdentity(req *http.Request) (*UserIdentity, error) {
	pluginCtx := backend.PluginConfigFromContext(req.Context())

	user := pluginCtx.User
	if user == nil {
		if pluginCtx.OrgID == 0 {
			return nil, fmt.Errorf("no user context available")
		}
		return &UserIdentity{
			OrgID:     pluginCtx.OrgID,
			UserID:    0,
			UserLogin: "anonymous",
		}, nil
	}

	userID := int64(0)
	if user.Login != "" {
		userID = int64(hashUserLogin(user.Login))
	}

	return &UserIdentity{
		UserID:    userID,
		UserLogin: user.Login,
		UserEmail: user.Email,
		OrgID:     pluginCtx.OrgID,
		OrgName:   user.Name,
	}, nil
}

// GrafanaQueryRequest represents the query request to Grafana's query API
type GrafanaQueryRequest struct {
	Queries []GrafanaQuery `json:"queries"`
	From    string         `json:"from,omitempty"`
	To      string         `json:"to,omitempty"`
}

// GrafanaQuery represents a single query in Grafana's format
type GrafanaQuery struct {
	RefID         string                 `json:"refId"`
	DatasourceUID string                 `json:"datasource,omitempty"`
	Expr          string                 `json:"expr,omitempty"`
	Query         string                 `json:"query,omitempty"`
	QueryType     string                 `json:"queryType,omitempty"`
	IntervalMs    int64                  `json:"intervalMs,omitempty"`
	MaxDataPoints int64                  `json:"maxDataPoints,omitempty"`
	Format        string                 `json:"format,omitempty"`
	Exemplar      bool                   `json:"exemplar,omitempty"`
	Additional    map[string]interface{} `json:"-"`
}

// executeQueries executes queries against Grafana datasources with user's security context
// The HTTP request context already contains the user's auth, which will be forwarded
func (a *App) executeQueries(ctx context.Context, incomingReq *http.Request, queryReq QueryRequest) (*QueryResponse, error) {
	// Build Grafana query request
	grafanaQueries := make([]GrafanaQuery, len(queryReq.Queries))
	for i, q := range queryReq.Queries {
		grafanaQueries[i] = GrafanaQuery{
			RefID:         q.RefID,
			DatasourceUID: queryReq.Datasource,
			Expr:          q.Expr,
			Query:         q.Query,
			QueryType:     q.QueryType,
			IntervalMs:    q.IntervalMs,
			MaxDataPoints: q.MaxDataPoints,
			Format:        q.Format,
		}
	}

	grafanaReq := GrafanaQueryRequest{
		Queries: grafanaQueries,
		From:    queryReq.TimeRange.From,
		To:      queryReq.TimeRange.To,
	}

	// Marshal request body
	reqBody, err := json.Marshal(grafanaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	// Get Grafana URL from environment or use localhost
	grafanaURL := os.Getenv("GF_URL")
	if grafanaURL == "" {
		grafanaURL = "http://localhost:3000"
	}

	queryURL := fmt.Sprintf("%s/api/ds/query", grafanaURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Forward authentication headers from the incoming request
	// This ensures the query executes with the user's permissions
	if authHeader := incomingReq.Header.Get("Authorization"); authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}
	if cookie := incomingReq.Header.Get("Cookie"); cookie != "" {
		httpReq.Header.Set("Cookie", cookie)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	backend.Logger.Debug("Executing query",
		"url", queryURL,
		"datasource", queryReq.Datasource,
		"queryCount", len(queryReq.Queries),
	)

	// Use default HTTP client for localhost calls
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Execute request with user's security context
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		backend.Logger.Error("Query failed",
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var queryResp QueryResponse
	if err := json.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	backend.Logger.Debug("Query executed successfully",
		"resultCount", len(queryResp.Results),
	)

	return &queryResp, nil
}

func (a *App) handleQuery(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user identity from plugin context
	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(w, "Failed to extract user identity", err, http.StatusUnauthorized)
		return
	}

	// Apply rate limiting per user
	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		if !a.guardrails.rateLimiter.Allow(user.UserLogin) {
			sendErrorResponse(w, "Rate limit exceeded",
				fmt.Errorf("too many requests from user %s", user.UserLogin),
				http.StatusTooManyRequests)
			return
		}
	}

	var queryReq QueryRequest
	if err := json.NewDecoder(req.Body).Decode(&queryReq); err != nil {
		sendErrorResponse(w, "Failed to decode query request", err, http.StatusBadRequest)
		return
	}

	startTime := time.Now()

	backend.Logger.Info("Query proxy request",
		"user", user.UserLogin,
		"userId", user.UserID,
		"orgId", user.OrgID,
		"datasource", queryReq.Datasource,
		"queryCount", len(queryReq.Queries),
	)

	// Execute queries against datasource with user's security context
	response, err := a.executeQueries(req.Context(), req, queryReq)
	if err != nil {
		backend.Logger.Error("Failed to execute queries",
			"error", err,
			"user", user.UserLogin,
			"datasource", queryReq.Datasource,
		)
		sendErrorResponse(w, "Failed to execute queries", err, http.StatusInternalServerError)
		return
	}

	// Audit log
	auditLog := map[string]interface{}{
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"user":            user.UserLogin,
		"userId":          user.UserID,
		"orgId":           user.OrgID,
		"datasource":      queryReq.Datasource,
		"queryCount":      len(queryReq.Queries),
		"executionTimeMs": time.Since(startTime).Milliseconds(),
		"success":         true,
	}

	backend.Logger.Info("Query audit log", "audit", auditLog)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		backend.Logger.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
