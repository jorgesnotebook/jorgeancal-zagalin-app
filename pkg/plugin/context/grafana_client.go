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

// GrafanaClient handles API requests to Grafana
type GrafanaClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewGrafanaClient creates a new Grafana API client
func NewGrafanaClient(httpClient *http.Client, baseURL string) *GrafanaClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second, // Set reasonable timeout
		}
	}
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	// Remove trailing slash to prevent double slashes in URLs
	baseURL = strings.TrimRight(baseURL, "/")
	return &GrafanaClient{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// DataSource represents a Grafana datasource
type DataSource struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// QueryRequest represents a datasource query request
type QueryRequest struct {
	Queries []Query `json:"queries"`
	From    string  `json:"from"`
	To      string  `json:"to"`
}

// Query represents a single query
type Query struct {
	RefID         string `json:"refId"`
	Expr          string `json:"expr,omitempty"`  // For Prometheus
	Query         string `json:"query,omitempty"` // For Loki
	DatasourceUID string `json:"datasource,omitempty"`
}

// QueryResponse represents the response from a query
type QueryResponse struct {
	Results map[string]QueryResult `json:"results"`
}

// QueryResult represents a single query result
type QueryResult struct {
	Frames []Frame `json:"frames"`
	Error  string  `json:"error,omitempty"`
}

// Frame represents a data frame
type Frame struct {
	Schema Schema          `json:"schema"`
	Data   json.RawMessage `json:"data,omitempty"`
}

// Schema represents the frame schema
type Schema struct {
	Fields []Field `json:"fields"`
}

// Field represents a field in the schema
type Field struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels,omitempty"`
}

// GetDatasource retrieves a datasource by UID
func (gc *GrafanaClient) GetDatasource(ctx context.Context, uid string) (*DataSource, error) {
	url := fmt.Sprintf("%s/api/datasources/uid/%s", gc.baseURL, uid)

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

// QueryDatasource executes a query against a datasource
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

	url := fmt.Sprintf("%s/api/ds/query", gc.baseURL)
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

// ListDatasources retrieves all datasources
func (gc *GrafanaClient) ListDatasources(ctx context.Context) ([]DataSource, error) {
	url := fmt.Sprintf("%s/api/datasources", gc.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		// Check if it's a context timeout
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
