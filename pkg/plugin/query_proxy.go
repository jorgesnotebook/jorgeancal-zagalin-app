package plugin

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
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

func (a *App) handleQuery(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(w, "Failed to extract user identity", err, http.StatusUnauthorized)
		return
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

	response := QueryResponse{
		Results: make(map[string]QueryResult),
	}

	for _, query := range queryReq.Queries {
		result := QueryResult{
			RefID: query.RefID,
			Frames: []interface{}{},
		}
		response.Results[query.RefID] = result
	}

	auditLog := map[string]interface{}{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"user":           user.UserLogin,
		"userId":         user.UserID,
		"orgId":          user.OrgID,
		"datasource":     queryReq.Datasource,
		"queryCount":     len(queryReq.Queries),
		"executionTimeMs": time.Since(startTime).Milliseconds(),
		"success":        true,
	}

	backend.Logger.Info("Query audit log", "audit", auditLog)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		backend.Logger.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
