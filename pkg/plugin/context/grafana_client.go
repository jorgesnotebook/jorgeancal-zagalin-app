package context

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type URLProvider func(ctx context.Context) (string, error)

type GrafanaClient struct {
	httpClient  *http.Client
	urlProvider URLProvider 
}

func NewGrafanaClient(httpClient *http.Client, urlProvider URLProvider) *GrafanaClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second, 
		}
	}

	if urlProvider == nil {
		urlProvider = func(ctx context.Context) (string, error) {
			return "http://localhost:3000", nil
		}
	}

	return &GrafanaClient{
		httpClient:  httpClient,
		urlProvider: urlProvider,
	}
}

func (gc *GrafanaClient) getBaseURL(ctx context.Context) (string, error) {
	baseURL, err := gc.urlProvider(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Grafana URL: %w", err)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL, nil
}

type DataSource struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

type QueryRequest struct {
	Queries []Query `json:"queries"`
	From    string  `json:"from"`
	To      string  `json:"to"`
}

type Query struct {
	RefID         string `json:"refId"`
	Expr          string `json:"expr,omitempty"`  
	Query         string `json:"query,omitempty"` 
	DatasourceUID string `json:"datasource,omitempty"`
}

type QueryResponse struct {
	Results map[string]QueryResult `json:"results"`
}

type QueryResult struct {
	Frames []Frame `json:"frames"`
	Error  string  `json:"error,omitempty"`
}

type Frame struct {
	Schema Schema          `json:"schema"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type Schema struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels,omitempty"`
}

func (gc *GrafanaClient) GetDatasource(ctx context.Context, uid string) (*DataSource, error) {
	baseURL, err := gc.getBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/datasources/uid/%s", baseURL, uid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get datasource: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("datasource request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ds DataSource
	if err := json.NewDecoder(resp.Body).Decode(&ds); err != nil {
		return nil, fmt.Errorf("failed to decode datasource: %w", err)
	}

	return &ds, nil
}

func (gc *GrafanaClient) QueryDatasource(ctx context.Context, dsUID string, query string, queryType string) (*QueryResponse, error) {
	now := time.Now()
	from := now.Add(-1 * time.Hour).Format(time.RFC3339)
	to := now.Format(time.RFC3339)

	q := Query{
		RefID:         "A",
		DatasourceUID: dsUID,
	}

	switch queryType {
	case "prometheus":
		q.Expr = query
	case "loki":
		q.Query = query
	default:
		q.Expr = query
		q.Query = query
	}

	reqBody := QueryRequest{
		Queries: []Query{q},
		From:    from,
		To:      to,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	baseURL, err := gc.getBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/ds/query", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var queryResp QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &queryResp, nil
}

func (gc *GrafanaClient) ListDatasources(ctx context.Context) ([]DataSource, error) {
	baseURL, err := gc.getBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/datasources", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("failed to fetch datasources: %w (timeout - this is expected if the plugin doesn't have direct Grafana API access)", err)
		}
		return nil, fmt.Errorf("failed to list datasources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list datasources failed with status %d: %s (URL: %s)", resp.StatusCode, string(body), url)
	}

	var datasources []DataSource
	if err := json.NewDecoder(resp.Body).Decode(&datasources); err != nil {
		return nil, fmt.Errorf("failed to decode datasources: %w", err)
	}

	return datasources, nil
}

type DashboardMeta struct {
	IsStarred bool   `json:"isStarred"`
	Slug      string `json:"slug"`
	FolderUID string `json:"folderUid"`
}

type DashboardJSON struct {
	UID    string        `json:"uid"`
	Title  string        `json:"title"`
	Tags   []string      `json:"tags"`
	Panels []interface{} `json:"panels"` 
}

type DashboardResponse struct {
	Meta      DashboardMeta `json:"meta"`
	Dashboard DashboardJSON `json:"dashboard"`
}

func (gc *GrafanaClient) GetDashboard(ctx context.Context, uid string) (*DashboardResponse, error) {
	baseURL, err := gc.getBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", baseURL, uid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("dashboard request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var dashboardResp DashboardResponse
	if err := json.NewDecoder(resp.Body).Decode(&dashboardResp); err != nil {
		return nil, fmt.Errorf("failed to decode dashboard: %w", err)
	}

	return &dashboardResp, nil
}
