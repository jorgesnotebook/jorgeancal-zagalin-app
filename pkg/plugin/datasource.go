package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// DatasourceInfo represents basic datasource information
type DatasourceInfo struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// DatasourceListResponse represents the response from Grafana's datasource API
type DatasourceListResponse []struct {
	ID   int64  `json:"id"`
	UID  string `json:"uid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// datasourceCache caches datasource information to avoid repeated API calls
type datasourceCache struct {
	mu          sync.RWMutex
	datasources map[string]DatasourceInfo // UID -> Info
	lastRefresh time.Time
	ttl         time.Duration
}

// newDatasourceCache creates a new datasource cache with 5-minute TTL
func newDatasourceCache() *datasourceCache {
	return &datasourceCache{
		datasources: make(map[string]DatasourceInfo),
		ttl:         5 * time.Minute,
	}
}

// fetchDatasources fetches all datasources from Grafana
func (a *App) fetchDatasources(ctx context.Context, req *http.Request) ([]DatasourceInfo, error) {
	// Get Grafana URL from context
	cfg := backend.GrafanaConfigFromContext(ctx)
	grafanaURL, err := cfg.AppURL()
	if err != nil {
		return nil, fmt.Errorf("failed to get Grafana URL: %w", err)
	}

	datasourcesURL := fmt.Sprintf("%s/api/datasources", grafanaURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, datasourcesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Forward authentication headers from the incoming request
	if authHeader := req.Header.Get("Authorization"); authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}
	if cookie := req.Header.Get("Cookie"); cookie != "" {
		httpReq.Header.Set("Cookie", cookie)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	backend.Logger.Debug("Fetching datasources", "url", datasourcesURL)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch datasources: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("datasource fetch failed with status %d: %s", resp.StatusCode, string(body))
	}

	var datasourceList DatasourceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&datasourceList); err != nil {
		return nil, fmt.Errorf("failed to parse datasource list: %w", err)
	}

	// Convert to simplified format
	result := make([]DatasourceInfo, 0, len(datasourceList))
	for _, ds := range datasourceList {
		result = append(result, DatasourceInfo{
			UID:  ds.UID,
			Name: ds.Name,
			Type: ds.Type,
		})
	}

	backend.Logger.Debug("Datasources fetched", "count", len(result))

	return result, nil
}

// isDatasourceAllowed checks if a datasource UID is in the allowlist
func (a *App) isDatasourceAllowed(datasourceUID string) bool {
	// If no allowlist configured, allow all datasources
	if a.settings == nil || len(a.settings.AllowedDatasources) == 0 {
		return true
	}

	// Check if datasource is in allowlist
	for _, allowedUID := range a.settings.AllowedDatasources {
		if allowedUID == datasourceUID {
			return true
		}
	}

	return false
}

// handleListDatasources returns the list of available datasources
func (a *App) handleListDatasources(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	backend.Logger.Debug("Listing datasources")

	datasources, err := a.fetchDatasources(req.Context(), req)
	if err != nil {
		backend.Logger.Error("Failed to fetch datasources", "error", err)
		// Return empty list instead of error to prevent UI breakage
		datasources = []DatasourceInfo{}
	}

	// Filter by allowlist if configured
	var filteredDatasources []DatasourceInfo
	if a.settings != nil && len(a.settings.AllowedDatasources) > 0 {
		for _, ds := range datasources {
			if a.isDatasourceAllowed(ds.UID) {
				filteredDatasources = append(filteredDatasources, ds)
			}
		}
	} else {
		filteredDatasources = datasources
	}

	// Build response, handling nil settings
	var allowedDatasources []string
	var defaultDatasource string
	if a.settings != nil {
		allowedDatasources = a.settings.AllowedDatasources
		defaultDatasource = a.settings.DefaultDatasource
	}

	response := map[string]interface{}{
		"datasources":        filteredDatasources,
		"allowedDatasources": allowedDatasources,
		"defaultDatasource":  defaultDatasource,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		backend.Logger.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// getDatasourceType returns the type of a datasource by UID (prometheus, loki, tempo, etc.)
// Uses caching to avoid repeated API calls
func (a *App) getDatasourceType(ctx context.Context, req *http.Request, uid string) (string, error) {
	if a.datasourceCache == nil {
		return "", fmt.Errorf("datasource cache not initialized")
	}

	// Check cache first
	a.datasourceCache.mu.RLock()
	needsRefresh := time.Since(a.datasourceCache.lastRefresh) > a.datasourceCache.ttl
	if !needsRefresh {
		if ds, ok := a.datasourceCache.datasources[uid]; ok {
			a.datasourceCache.mu.RUnlock()
			return ds.Type, nil
		}
	}
	a.datasourceCache.mu.RUnlock()

	// Cache miss or expired - fetch datasources
	datasources, err := a.fetchDatasources(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch datasources: %w", err)
	}

	// Update cache
	a.datasourceCache.mu.Lock()
	a.datasourceCache.datasources = make(map[string]DatasourceInfo)
	for _, ds := range datasources {
		a.datasourceCache.datasources[ds.UID] = ds
	}
	a.datasourceCache.lastRefresh = time.Now()
	a.datasourceCache.mu.Unlock()

	// Look up the requested UID
	a.datasourceCache.mu.RLock()
	defer a.datasourceCache.mu.RUnlock()
	if ds, ok := a.datasourceCache.datasources[uid]; ok {
		return ds.Type, nil
	}

	return "", fmt.Errorf("datasource not found: %s", uid)
}
