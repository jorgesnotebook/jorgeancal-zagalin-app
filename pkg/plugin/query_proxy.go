package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
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


func (a *App) executeQueries(ctx context.Context, incomingReq *http.Request, queryReq QueryRequest) (*QueryResponse, error) {
	grafanaQueries := make([]GrafanaDSQuery, len(queryReq.Queries))
	for i, q := range queryReq.Queries {
		grafanaQueries[i] = GrafanaDSQuery{
			RefID: q.RefID,
			Datasource: DatasourceRef{
				UID: queryReq.Datasource,
			},
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

	reqBody, err := json.Marshal(grafanaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}

	cfg := backend.GrafanaConfigFromContext(ctx)
	grafanaURL, err := cfg.AppURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get Grafana URL: %w", err)
	}

	queryURL := fmt.Sprintf("%s/api/ds/query", grafanaURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, queryURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

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

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		backend.Logger.Error("Query failed",
			"status", resp.StatusCode,
			"body", string(respBody),
		)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(respBody))
	}

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

	user, err := extractUserIdentity(req)
	if err != nil {
		sendErrorResponse(w, "Failed to extract user identity", err, http.StatusUnauthorized)
		return
	}

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

	if !a.isDatasourceAllowed(queryReq.Datasource) {
		backend.Logger.Warn("Query blocked: datasource not in allowlist",
			"user", user.UserLogin,
			"datasource", queryReq.Datasource,
			"allowedDatasources", a.settings.AllowedDatasources,
		)
		sendErrorResponse(w, "Datasource not allowed",
			fmt.Errorf("datasource '%s' is not in the allowed list", queryReq.Datasource),
			http.StatusForbidden)
		return
	}

	if a.queryValidator != nil && a.settings != nil && a.settings.QueryValidation.Enabled {
		dsTypeStr, err := a.getDatasourceType(req.Context(), req, queryReq.Datasource)
		if err != nil {
			backend.Logger.Warn("Failed to detect datasource type for validation",
				"error", err,
				"datasource", queryReq.Datasource,
			)
			dsTypeStr = "other"
		}

		var dsType DatasourceType
		switch dsTypeStr {
		case "prometheus":
			dsType = DatasourcePrometheus
		case "loki":
			dsType = DatasourceLoki
		case "tempo":
			dsType = DatasourceTempo
		default:
			dsType = DatasourceOther
		}

		for i := range queryReq.Queries {
			queryStr := queryReq.Queries[i].Expr
			if queryStr == "" {
				queryStr = queryReq.Queries[i].Query
			}
			if queryStr == "" {
				continue 
			}

			result := a.queryValidator.ValidateQuery(req.Context(), queryStr, dsType)

			if !result.Valid {
				backend.Logger.Warn("Query validation failed",
					"user", user.UserLogin,
					"userId", user.UserID,
					"orgId", user.OrgID,
					"datasource", queryReq.Datasource,
					"datasourceType", dsType,
					"violationType", result.ViolationType,
					"queryHash", hashQuery(result.OriginalQuery),
					"queryLength", len(result.OriginalQuery),
					"error", result.Error,
				)

				a.logQueryValidationFailure(user, queryReq.Datasource, result)

				sendErrorResponse(w, "Query validation failed", result.Error, http.StatusBadRequest)
				return
			}

			if result.Sanitized {
				backend.Logger.Warn("Query sanitized",
					"user", user.UserLogin,
					"userId", user.UserID,
					"orgId", user.OrgID,
					"datasource", queryReq.Datasource,
					"datasourceType", dsType,
					"originalHash", hashQuery(result.OriginalQuery),
					"sanitizedHash", hashQuery(result.SanitizedQuery),
					"originalLength", len(result.OriginalQuery),
					"sanitizedLength", len(result.SanitizedQuery),
				)

				a.logQuerySanitization(user, queryReq.Datasource, result)

				if queryReq.Queries[i].Expr != "" {
					queryReq.Queries[i].Expr = result.SanitizedQuery
				} else {
					queryReq.Queries[i].Query = result.SanitizedQuery
				}
			}

			if len(result.LLMWarnings) > 0 {
				backend.Logger.Info("LLM validation warnings",
					"user", user.UserLogin,
					"datasource", queryReq.Datasource,
					"warnings", result.LLMWarnings,
					"suggestions", result.LLMSuggestions,
				)
			}
		}

		backend.Logger.Debug("Query validation passed",
			"datasource", queryReq.Datasource,
			"datasourceType", dsType,
			"queryCount", len(queryReq.Queries),
		)
	}

	if a.settings != nil && a.settings.OtelEnforcement.Enabled {
		dsTypeStr, err := a.getDatasourceType(req.Context(), req, queryReq.Datasource)
		if err != nil {
			backend.Logger.Warn("Failed to detect datasource type, using validation-only mode",
				"error", err,
				"datasource", queryReq.Datasource,
			)
			dsTypeStr = "other"
		}

		var dsType DatasourceType
		switch dsTypeStr {
		case "prometheus":
			dsType = DatasourcePrometheus
		case "loki":
			dsType = DatasourceLoki
		case "tempo":
			dsType = DatasourceTempo
		default:
			dsType = DatasourceOther
		}

		backend.Logger.Debug("Datasource type detected",
			"datasource", queryReq.Datasource,
			"type", dsType,
		)

		scope := a.extractOtelScopeFromQuery(queryReq, dsType)

		a.applyOtelScopeDefaults(scope)

		if err := a.validateOtelScope(scope); err != nil {
			backend.Logger.Warn("Query blocked: OTel scope validation failed",
				"user", user.UserLogin,
				"error", err,
				"scope", scope,
				"datasourceType", dsType,
			)
			sendErrorResponse(w, "Query scope validation failed", err, http.StatusBadRequest)
			return
		}

		if err := a.injectOtelScope(&queryReq, scope, dsType); err != nil {
			backend.Logger.Error("Failed to inject OTel scope",
				"error", err,
				"user", user.UserLogin,
				"datasourceType", dsType,
			)
			sendErrorResponse(w, "Failed to apply query scope", err, http.StatusInternalServerError)
			return
		}

		a.logOtelScopeUsage(user, scope, queryReq.Datasource)
	}

	startTime := time.Now()

	backend.Logger.Info("Query proxy request",
		"user", user.UserLogin,
		"userId", user.UserID,
		"orgId", user.OrgID,
		"datasource", queryReq.Datasource,
		"queryCount", len(queryReq.Queries),
	)

	if a.grafanaQueryClient == nil {
		backend.Logger.Error("Grafana query client not initialized")
		sendErrorResponse(w, "Grafana query client not configured",
			fmt.Errorf("service account token required"), http.StatusInternalServerError)
		return
	}

	dsTypeStr, err := a.getDatasourceType(req.Context(), req, queryReq.Datasource)
	if err != nil {
		backend.Logger.Warn("Failed to detect datasource type", "error", err)
		dsTypeStr = "other"
	}

	evidencePacks := make([]*EvidencePack, 0, len(queryReq.Queries))

	for _, query := range queryReq.Queries {
		grafanaResp, err := a.grafanaQueryClient.ExecuteQuery(
			req.Context(),
			user,
			queryReq.Datasource,
			dsTypeStr,
			query,
			queryReq.TimeRange,
		)

		if err != nil {
			backend.Logger.Error("Failed to execute query via Grafana",
				"error", err,
				"user", user.UserLogin,
				"datasource", queryReq.Datasource,
			)
			sendErrorResponse(w, "Failed to execute query", err, http.StatusInternalServerError)
			return
		}

		evidencePack, err := a.buildEvidencePack(grafanaResp, query, queryReq.Datasource, dsTypeStr, queryReq.TimeRange)
		if err != nil {
			backend.Logger.Error("Failed to build evidence pack",
				"error", err,
				"user", user.UserLogin,
				"datasource", queryReq.Datasource,
			)
			sendErrorResponse(w, "Failed to build evidence pack", err, http.StatusInternalServerError)
			return
		}

		evidencePacks = append(evidencePacks, evidencePack)
	}

	auditLog := map[string]interface{}{
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"user":            user.UserLogin,
		"userId":          user.UserID,
		"orgId":           user.OrgID,
		"datasource":      queryReq.Datasource,
		"queryCount":      len(queryReq.Queries),
		"evidenceCount":   len(evidencePacks),
		"executionTimeMs": time.Since(startTime).Milliseconds(),
		"success":         true,
	}

	backend.Logger.Info("Query audit log with evidence", "audit", auditLog)

	responseData := map[string]interface{}{
		"evidencePacks": evidencePacks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		backend.Logger.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *App) logQueryValidationFailure(user *UserIdentity, datasource string, result *QueryValidationResult) {
	auditLog := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"event":         "query_validation_failed",
		"user":          user.UserLogin,
		"userId":        user.UserID,
		"orgId":         user.OrgID,
		"datasource":    datasource,
		"violationType": result.ViolationType,
		"queryHash":     hashQuery(result.OriginalQuery),
		"queryLength":   len(result.OriginalQuery),
		"error":         result.Error.Error(),
	}
	backend.Logger.Info("Query validation failure audit", "audit", auditLog)
}

func (a *App) logQuerySanitization(user *UserIdentity, datasource string, result *QueryValidationResult) {
	auditLog := map[string]interface{}{
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"event":           "query_sanitized",
		"user":            user.UserLogin,
		"userId":          user.UserID,
		"orgId":           user.OrgID,
		"datasource":      datasource,
		"originalHash":    hashQuery(result.OriginalQuery),
		"sanitizedHash":   hashQuery(result.SanitizedQuery),
		"originalLength":  len(result.OriginalQuery),
		"sanitizedLength": len(result.SanitizedQuery),
	}
	backend.Logger.Info("Query sanitization audit", "audit", auditLog)
}

func (a *App) buildEvidencePack(
	grafanaResp *GrafanaQueryResponse,
	query QueryPayload,
	datasource string,
	datasourceType string,
	timeRange TimeRange,
) (*EvidencePack, error) {
	queryStr := query.Expr
	if queryStr == "" {
		queryStr = query.Query
	}

	switch datasourceType {
	case "prometheus", "mimir":
		return BuildMetricsEvidencePack(grafanaResp, queryStr, datasource, timeRange)

	case "loki":
		return BuildLogsEvidencePack(grafanaResp, queryStr, datasource, timeRange)

	case "tempo":
		return BuildTracesEvidencePack(grafanaResp, queryStr, datasource)

	default:
		return nil, fmt.Errorf("unsupported datasource type for evidence: %s", datasourceType)
	}
}
