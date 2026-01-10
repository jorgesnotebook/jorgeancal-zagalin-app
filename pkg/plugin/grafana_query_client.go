package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GrafanaQueryClient struct {
	baseURL             string
	serviceAccountToken string
	httpClient          *http.Client
}

func NewGrafanaQueryClient(baseURL, serviceAccountToken string) *GrafanaQueryClient {
	return &GrafanaQueryClient{
		baseURL:             baseURL,
		serviceAccountToken: serviceAccountToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type GrafanaQueryRequest struct {
	Queries []GrafanaDSQuery `json:"queries"`
	From    string           `json:"from,omitempty"`
	To      string           `json:"to,omitempty"`
}

type GrafanaDSQuery struct {
	RefID         string                 `json:"refId"`
	Datasource    DatasourceRef          `json:"datasource"`
	Expr          string                 `json:"expr,omitempty"`
	Query         string                 `json:"query,omitempty"`
	QueryType     string                 `json:"queryType,omitempty"`
	Format        string                 `json:"format,omitempty"`
	IntervalMs    int64                  `json:"intervalMs,omitempty"`
	MaxDataPoints int64                  `json:"maxDataPoints,omitempty"`
	Additional    map[string]interface{} `json:"-"`
}

type DatasourceRef struct {
	Type string `json:"type,omitempty"`
	UID  string `json:"uid"`
}

type GrafanaQueryResponse struct {
	Results map[string]QueryResultData `json:"results"`
}

type QueryResultData struct {
	RefID  string        `json:"refId"`
	Frames []interface{} `json:"frames"`
	Error  string        `json:"error,omitempty"`
	Meta   interface{}   `json:"meta,omitempty"`
}

func (c *GrafanaQueryClient) ExecuteQuery(
	ctx context.Context,
	user *UserIdentity,
	datasourceUID string,
	datasourceType string,
	query QueryPayload,
	timeRange TimeRange,
) (*GrafanaQueryResponse, error) {
	grafanaQuery := GrafanaDSQuery{
		RefID: "A",
		Datasource: DatasourceRef{
			Type: datasourceType,
			UID:  datasourceUID,
		},
		IntervalMs:    query.IntervalMs,
		MaxDataPoints: query.MaxDataPoints,
		Format:        query.Format,
		QueryType:     query.QueryType,
	}

	if query.Expr != "" {
		grafanaQuery.Expr = query.Expr
	}
	if query.Query != "" {
		grafanaQuery.Query = query.Query
	}

	reqBody := GrafanaQueryRequest{
		Queries: []GrafanaDSQuery{grafanaQuery},
		From:    timeRange.From,
		To:      timeRange.To,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/ds/query", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceAccountToken))
	httpReq.Header.Set("X-Grafana-User", user.UserLogin)
	httpReq.Header.Set("X-Zagalin-User", user.UserLogin)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("grafana query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var queryResp GrafanaQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &queryResp, nil
}
