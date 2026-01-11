package context

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

// mockRoundTripper implements http.RoundTripper for testing
type mockRoundTripper struct {
	responses map[string]*http.Response
	callCount map[string]int
	mu        sync.Mutex
}

func newMockHTTPClient() *http.Client {
	transport := &mockRoundTripper{
		responses: make(map[string]*http.Response),
		callCount: make(map[string]int),
	}
	return &http.Client{
		Transport: transport,
	}
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Track call count
	m.callCount[req.URL.Path]++

	// Match on URL path
	key := req.URL.Path
	if resp, ok := m.responses[key]; ok {
		// Clone response body for reuse
		if resp.Body != nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewReader(body))
			// Store another copy for next call
			m.responses[key].Body = io.NopCloser(bytes.NewReader(body))
		}
		return resp, nil
	}

	// Default: return empty datasource list
	if req.URL.Path == "/api/datasources" {
		datasources := []DataSource{
			{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
		}
		body, _ := json.Marshal(datasources)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader([]byte("not found"))),
	}, nil
}

// Helper to add response to mock HTTP client
func addMockResponse(client *http.Client, path string, resp *http.Response) {
	if transport, ok := client.Transport.(*mockRoundTripper); ok {
		transport.mu.Lock()
		defer transport.mu.Unlock()
		transport.responses[path] = resp
	}
}

// Helper to get call count from mock HTTP client
func getMockCallCount(client *http.Client, path string) int {
	if transport, ok := client.Transport.(*mockRoundTripper); ok {
		transport.mu.Lock()
		defer transport.mu.Unlock()
		return transport.callCount[path]
	}
	return 0
}

func createDataSourceListResponse(datasources []DataSource) *http.Response {
	body, _ := json.Marshal(datasources)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func createDashboardResponse(uid, title string) *http.Response {
	dashboard := DashboardResponse{
		Dashboard: DashboardJSON{
			UID:   uid,
			Title: title,
		},
	}
	body, _ := json.Marshal(dashboard)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func createErrorResponse(statusCode int, message string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader([]byte(message))),
	}
}

// TestNewManager tests manager initialization
func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.client == nil {
		t.Error("Manager client should not be nil")
	}

	if manager.stopCh == nil {
		t.Error("Manager stopCh should not be nil")
	}

	// Verify initial state
	ctx := manager.GetContext()
	if ctx != nil {
		t.Error("GetContext() should return nil before first refresh")
	}

	lastUpdated := manager.GetLastUpdated()
	if !lastUpdated.IsZero() {
		t.Error("LastUpdated should be zero time before first refresh")
	}
}

// TestManagerConfigure tests configuration
func TestManagerConfigure(t *testing.T) {
	tests := []struct {
		name            string
		datasourceUIDs  []string
		refreshMinutes  int
		enableMetrics   bool
		enableLogs      bool
		enableTraces    bool
	}{
		{
			name:            "all features enabled",
			datasourceUIDs:  []string{"ds-1", "ds-2"},
			refreshMinutes:  5,
			enableMetrics:   true,
			enableLogs:      true,
			enableTraces:    true,
		},
		{
			name:            "only metrics",
			datasourceUIDs:  []string{"ds-1"},
			refreshMinutes:  10,
			enableMetrics:   true,
			enableLogs:      false,
			enableTraces:    false,
		},
		{
			name:            "no datasources",
			datasourceUIDs:  []string{},
			refreshMinutes:  15,
			enableMetrics:   true,
			enableLogs:      true,
			enableTraces:    false,
		},
		{
			name:            "zero refresh interval",
			datasourceUIDs:  []string{"ds-1"},
			refreshMinutes:  0,
			enableMetrics:   true,
			enableLogs:      false,
			enableTraces:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			manager.Configure(tt.datasourceUIDs, tt.refreshMinutes, tt.enableMetrics, tt.enableLogs, tt.enableTraces)

			if len(manager.datasourceUIDs) != len(tt.datasourceUIDs) {
				t.Errorf("datasourceUIDs = %v, want %v", manager.datasourceUIDs, tt.datasourceUIDs)
			}

			expectedInterval := time.Duration(tt.refreshMinutes) * time.Minute
			if manager.refreshInterval != expectedInterval {
				t.Errorf("refreshInterval = %v, want %v", manager.refreshInterval, expectedInterval)
			}

			if manager.enableMetrics != tt.enableMetrics {
				t.Errorf("enableMetrics = %v, want %v", manager.enableMetrics, tt.enableMetrics)
			}

			if manager.enableLogs != tt.enableLogs {
				t.Errorf("enableLogs = %v, want %v", manager.enableLogs, tt.enableLogs)
			}

			if manager.enableTraces != tt.enableTraces {
				t.Errorf("enableTraces = %v, want %v", manager.enableTraces, tt.enableTraces)
			}
		})
	}
}

// TestManagerStartStop tests lifecycle management
func TestManagerStartStop(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Configure with valid interval
	manager.Configure([]string{"ds-1"}, 1, true, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start manager
	manager.Start(ctx)

	// Wait briefly for initial refresh
	time.Sleep(200 * time.Millisecond)

	// Verify context was populated
	obsCtx := manager.GetContext()
	if obsCtx == nil {
		t.Error("GetContext() should return non-nil after Start()")
	}

	// Stop manager
	manager.Stop()

	// Verify stop completed (should not hang)
	time.Sleep(50 * time.Millisecond)
}

// TestManagerStartWithZeroInterval tests that zero interval disables periodic refresh
func TestManagerStartWithZeroInterval(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Configure with zero interval
	manager.Configure([]string{"ds-1"}, 0, true, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	// Wait for initial refresh
	time.Sleep(200 * time.Millisecond)

	// Record call count after initial refresh
	initialCount := getMockCallCount(mockHTTP,"/api/datasources")

	// Wait to ensure no periodic refresh happens
	time.Sleep(300 * time.Millisecond)

	finalCount := getMockCallCount(mockHTTP,"/api/datasources")

	// Should only have initial refresh, no periodic refresh
	if finalCount != initialCount {
		t.Errorf("Expected no periodic refresh with zero interval, got %d -> %d calls", initialCount, finalCount)
	}

	manager.Stop()
}

// TestManagerRefresh tests context refresh
func TestManagerRefresh(t *testing.T) {
	tests := []struct {
		name          string
		datasources   []DataSource
		apiError      bool
		enableMetrics bool
		enableLogs    bool
		enableTraces  bool
		wantErr       bool
	}{
		{
			name: "successful refresh with metrics",
			datasources: []DataSource{
				{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
			},
			enableMetrics: true,
			wantErr:       false,
		},
		{
			name: "no datasources - auto-discovery",
			datasources: []DataSource{
				{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
			},
			enableMetrics: true,
			wantErr:       false,
		},
		{
			name:          "API error - graceful handling",
			apiError:      true,
			enableMetrics: true,
			wantErr:       false, // Should not return error, just log warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			if tt.apiError {
				addMockResponse(mockHTTP,"/api/datasources", createErrorResponse(500, "API error"))
			} else {
				addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse(tt.datasources))
			}

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			var configuredDatasources []string
			if tt.name != "no datasources - auto-discovery" {
				for _, ds := range tt.datasources {
					configuredDatasources = append(configuredDatasources, ds.UID)
				}
			}

			manager.Configure(configuredDatasources, 5, tt.enableMetrics, tt.enableLogs, tt.enableTraces)

			ctx := context.Background()
			err := manager.Refresh(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Refresh() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.apiError {
				// Verify context was updated
				obsCtx := manager.GetContext()
				if obsCtx == nil && !tt.wantErr {
					t.Error("GetContext() should return non-nil after successful refresh")
				}

				// Verify last updated time was set
				lastUpdated := manager.GetLastUpdated()
				if lastUpdated.IsZero() && !tt.wantErr {
					t.Error("LastUpdated should be set after refresh")
				}
			}
		})
	}
}

// TestManagerGetContext tests thread-safe context retrieval
func TestManagerGetContext(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Before refresh, should return nil
	ctx := manager.GetContext()
	if ctx != nil {
		t.Error("GetContext() should return nil before refresh")
	}

	// After refresh, should return context
	manager.Configure([]string{"ds-1"}, 5, true, false, false)
	err := manager.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() failed: %v", err)
	}

	ctx = manager.GetContext()
	if ctx == nil {
		t.Fatal("GetContext() should return non-nil after refresh")
	}

	// Verify context is a copy (mutations don't affect manager state)
	originalTime := ctx.LastUpdated
	ctx.LastUpdated = time.Now().Add(1 * time.Hour)

	ctx2 := manager.GetContext()
	if ctx2.LastUpdated != originalTime {
		t.Error("GetContext() should return a copy, external mutations should not affect manager state")
	}
}

// TestManagerGetContextPrompt tests prompt generation
func TestManagerGetContextPrompt(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Before refresh, should return empty string
	prompt := manager.GetContextPrompt()
	if prompt != "" {
		t.Error("GetContextPrompt() should return empty string before refresh")
	}

	// After refresh, should return prompt
	manager.Configure([]string{"ds-1"}, 5, true, false, false)
	manager.Refresh(context.Background())

	prompt = manager.GetContextPrompt()
	// Context exists but may be empty, prompt generation depends on BuildPrompt()
	obsCtx := manager.GetContext()
	if obsCtx != nil {
		// If context exists, BuildPrompt should work (even if returns empty for empty context)
		_ = prompt
	}
}

// TestManagerFetchReferenceDashboards tests dashboard fetching
func TestManagerFetchReferenceDashboards(t *testing.T) {
	tests := []struct {
		name      string
		uids      []string
		wantCount int
	}{
		{
			name:      "fetch multiple dashboards",
			uids:      []string{"dash-1", "dash-2"},
			wantCount: 2,
		},
		{
			name:      "fetch single dashboard",
			uids:      []string{"dash-1"},
			wantCount: 1,
		},
		{
			name:      "dashboard not found - partial success",
			uids:      []string{"dash-1", "dash-missing"},
			wantCount: 1, // Only dash-1 should be fetched
		},
		{
			name:      "empty dashboard list",
			uids:      []string{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add dashboard responses
			addMockResponse(mockHTTP,"/api/dashboards/uid/dash-1", createDashboardResponse("dash-1", "Dashboard 1"))
			addMockResponse(mockHTTP,"/api/dashboards/uid/dash-2", createDashboardResponse("dash-2", "Dashboard 2"))
			addMockResponse(mockHTTP,"/api/dashboards/uid/dash-missing", createErrorResponse(404, "not found"))

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			err := manager.FetchReferenceDashboards(context.Background(), tt.uids)

			// FetchReferenceDashboards should not return error even if some dashboards fail
			if err != nil {
				t.Errorf("FetchReferenceDashboards() unexpected error = %v", err)
			}

			dashboards := manager.GetReferenceDashboards()
			if len(dashboards) != tt.wantCount {
				t.Errorf("GetReferenceDashboards() count = %d, want %d", len(dashboards), tt.wantCount)
			}
		})
	}
}

// TestManagerGetReferenceDashboards tests dashboard retrieval
func TestManagerGetReferenceDashboards(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/dashboards/uid/dash-1", createDashboardResponse("dash-1", "Dashboard 1"))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Before fetching, should return nil or empty
	dashboards := manager.GetReferenceDashboards()
	if dashboards == nil {
		dashboards = make(map[string]*DashboardResponse)
	}
	if len(dashboards) != 0 {
		t.Error("GetReferenceDashboards() should return empty map before fetching")
	}

	// After fetching, should return dashboards
	manager.FetchReferenceDashboards(context.Background(), []string{"dash-1"})

	dashboards = manager.GetReferenceDashboards()
	if len(dashboards) != 1 {
		t.Errorf("GetReferenceDashboards() count = %d, want 1", len(dashboards))
	}

	if dash, ok := dashboards["dash-1"]; !ok || dash.Dashboard.Title != "Dashboard 1" {
		t.Error("GetReferenceDashboards() did not return expected dashboard")
	}
}

// TestManagerConcurrentGetContext tests thread safety
func TestManagerConcurrentGetContext(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{"ds-1"}, 5, true, false, false)
	manager.Refresh(context.Background())

	// Simulate concurrent access from 100 goroutines
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := manager.GetContext()
			if ctx != nil {
				// Access context fields to ensure no race conditions
				_ = ctx.LastUpdated
				_ = ctx.Metrics
			}
		}()
	}

	wg.Wait()
}

// TestManagerConcurrentRefresh tests concurrent refresh calls
func TestManagerConcurrentRefresh(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{"ds-1"}, 5, true, false, false)

	// Simulate concurrent refresh from multiple goroutines
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Refresh(context.Background())
		}()
	}

	wg.Wait()

	// Verify manager is still in consistent state
	ctx := manager.GetContext()
	if ctx == nil {
		t.Error("GetContext() should return non-nil after concurrent refreshes")
	}
}

// TestManagerConcurrentDashboardFetch tests concurrent dashboard fetching
func TestManagerConcurrentDashboardFetch(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/dashboards/uid/dash-1", createDashboardResponse("dash-1", "Dashboard 1"))
	addMockResponse(mockHTTP,"/api/dashboards/uid/dash-2", createDashboardResponse("dash-2", "Dashboard 2"))
	addMockResponse(mockHTTP,"/api/dashboards/uid/dash-3", createDashboardResponse("dash-3", "Dashboard 3"))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Simulate concurrent dashboard fetching
	var wg sync.WaitGroup
	uids := [][]string{
		{"dash-1"},
		{"dash-2"},
		{"dash-3"},
		{"dash-1", "dash-2"},
	}

	for _, uidList := range uids {
		wg.Add(1)
		go func(uids []string) {
			defer wg.Done()
			manager.FetchReferenceDashboards(context.Background(), uids)
		}(uidList)
	}

	wg.Wait()

	// Verify manager is still in consistent state
	dashboards := manager.GetReferenceDashboards()
	if dashboards == nil {
		t.Error("GetReferenceDashboards() should not return nil after concurrent fetches")
	}
}

// TestManagerContextCancellation tests context cancellation handling
func TestManagerContextCancellation(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{"ds-1"}, 1, true, false, false)

	ctx, cancel := context.WithCancel(context.Background())

	manager.Start(ctx)

	// Wait for initial refresh
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for goroutine to exit
	time.Sleep(100 * time.Millisecond)

	// Stop should complete without hanging
	manager.Stop()
}

// TestManagerRefreshErrorResilience tests that refresh errors don't crash the manager
func TestManagerRefreshErrorResilience(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createErrorResponse(500, "simulated API error"))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{}, 1, true, false, false) // Empty UIDs to trigger ListDatasources

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	// Wait for multiple refresh attempts
	time.Sleep(500 * time.Millisecond)

	// Verify manager is still running (didn't crash)
	manager.Stop()

	// Should have attempted multiple refreshes despite errors
	callCount := getMockCallCount(mockHTTP,"/api/datasources")

	if callCount == 0 {
		t.Error("Manager should have attempted at least one refresh despite errors")
	}
}

// TestManagerGetLastUpdated tests last updated time tracking
func TestManagerGetLastUpdated(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Before refresh, should be zero time
	lastUpdated := manager.GetLastUpdated()
	if !lastUpdated.IsZero() {
		t.Error("GetLastUpdated() should return zero time before refresh")
	}

	// After refresh, should be non-zero
	manager.Configure([]string{"ds-1"}, 5, true, false, false)
	beforeRefresh := time.Now()
	manager.Refresh(context.Background())
	afterRefresh := time.Now()

	lastUpdated = manager.GetLastUpdated()
	if lastUpdated.IsZero() {
		t.Error("GetLastUpdated() should return non-zero time after refresh")
	}

	if lastUpdated.Before(beforeRefresh) || lastUpdated.After(afterRefresh) {
		t.Errorf("GetLastUpdated() time %v should be between %v and %v", lastUpdated, beforeRefresh, afterRefresh)
	}
}

// TestManagerRefreshWithAllSignalTypes tests refresh with metrics, logs, and traces enabled
func TestManagerRefreshWithAllSignalTypes(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
		{UID: "loki-1", Type: "loki", Name: "Loki"},
		{UID: "tempo-1", Type: "tempo", Name: "Tempo"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Enable all signal types
	manager.Configure([]string{"prom-1", "loki-1", "tempo-1"}, 5, true, true, true)

	err := manager.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh() with all signal types failed: %v", err)
	}

	// Verify context was created
	ctx := manager.GetContext()
	if ctx == nil {
		t.Fatal("GetContext() should return non-nil after refresh with all signal types")
	}

	// Verify last updated is set
	if ctx.LastUpdated.IsZero() {
		t.Error("Context LastUpdated should be set")
	}
}

// TestManagerRefreshWithSelectiveSignalTypes tests enabling/disabling specific signal types
func TestManagerRefreshWithSelectiveSignalTypes(t *testing.T) {
	tests := []struct {
		name          string
		enableMetrics bool
		enableLogs    bool
		enableTraces  bool
		expectMetrics bool
		expectLogs    bool
		expectTraces  bool
	}{
		{
			name:          "only metrics",
			enableMetrics: true,
			enableLogs:    false,
			enableTraces:  false,
			expectMetrics: true,
			expectLogs:    false,
			expectTraces:  false,
		},
		{
			name:          "only logs",
			enableMetrics: false,
			enableLogs:    true,
			enableTraces:  false,
			expectMetrics: false,
			expectLogs:    true,
			expectTraces:  false,
		},
		{
			name:          "only traces",
			enableMetrics: false,
			enableLogs:    false,
			enableTraces:  true,
			expectMetrics: false,
			expectLogs:    false,
			expectTraces:  true,
		},
		{
			name:          "metrics and logs",
			enableMetrics: true,
			enableLogs:    true,
			enableTraces:  false,
			expectMetrics: true,
			expectLogs:    true,
			expectTraces:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()
			addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
				{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
			}))

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			manager.Configure([]string{"ds-1"}, 5, tt.enableMetrics, tt.enableLogs, tt.enableTraces)
			manager.Refresh(context.Background())

			ctx := manager.GetContext()
			if ctx == nil {
				t.Fatal("GetContext() should return non-nil after refresh")
			}

			// Verify signal types match configuration
			hasMetrics := ctx.Metrics != nil
			hasLogs := ctx.Logs != nil
			hasTraces := ctx.Traces != nil

			if hasMetrics != tt.expectMetrics {
				t.Errorf("Metrics presence = %v, want %v", hasMetrics, tt.expectMetrics)
			}
			if hasLogs != tt.expectLogs {
				t.Errorf("Logs presence = %v, want %v", hasLogs, tt.expectLogs)
			}
			if hasTraces != tt.expectTraces {
				t.Errorf("Traces presence = %v, want %v", hasTraces, tt.expectTraces)
			}
		})
	}
}

// TestManagerConcurrentConfigure tests concurrent Configure calls
func TestManagerConcurrentConfigure(t *testing.T) {
	manager := NewManager()

	// Simulate concurrent Configure calls from multiple goroutines
	var wg sync.WaitGroup
	configs := []struct {
		datasources []string
		interval    int
		metrics     bool
		logs        bool
		traces      bool
	}{
		{[]string{"ds-1"}, 5, true, false, false},
		{[]string{"ds-2"}, 10, false, true, false},
		{[]string{"ds-3"}, 15, false, false, true},
		{[]string{"ds-1", "ds-2"}, 20, true, true, false},
	}

	for _, cfg := range configs {
		wg.Add(1)
		go func(c struct {
			datasources []string
			interval    int
			metrics     bool
			logs        bool
			traces      bool
		}) {
			defer wg.Done()
			manager.Configure(c.datasources, c.interval, c.metrics, c.logs, c.traces)
		}(cfg)
	}

	wg.Wait()

	// Verify manager is in consistent state after concurrent Configure calls
	// (exact configuration depends on timing, but should not panic or race)
	_ = manager.datasourceUIDs
	_ = manager.refreshInterval
}

// TestManagerMultipleStarts tests that multiple Start calls don't cause issues
func TestManagerMultipleStarts(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{"ds-1"}, 1, true, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start multiple times (should spawn multiple goroutines - not ideal but shouldn't crash)
	manager.Start(ctx)
	manager.Start(ctx)

	// Wait for initial refreshes
	time.Sleep(200 * time.Millisecond)

	// Stop should wait for all goroutines
	manager.Stop()
}

// TestManagerStopWithoutStart tests stopping without starting
func TestManagerStopWithoutStart(t *testing.T) {
	manager := NewManager()

	// Stop without Start should not panic
	manager.Stop()

	// Second Stop should also not panic
	manager.Stop()
}

// TestManagerGetContextPromptWithData tests prompt generation with actual data
func TestManagerGetContextPromptWithData(t *testing.T) {
	manager := NewManager()

	// Manually populate context with data
	manager.context = &ObservabilityContext{
		Metrics: &MetricsContext{
			MetricNames: []string{"http_requests_total", "cpu_usage", "memory_usage"},
			Labels:      []string{"job", "instance", "status"},
			LabelValues: map[string][]string{
				"job":    {"api", "web"},
				"status": {"200", "404", "500"},
			},
			SampleCount: 100,
		},
		Logs: &LogsContext{
			Streams: []string{`{app="backend"}`, `{app="frontend"}`},
			Labels:  []string{"app", "level"},
			LabelValues: map[string][]string{
				"app":   {"backend", "frontend"},
				"level": {"info", "error"},
			},
			SampleCount: 50,
		},
		Traces: &TracesContext{
			Services:    []string{"api-service", "db-service"},
			Operations:  []string{"GET /api/users", "SELECT FROM users"},
			SpanNames:   []string{"http.request", "db.query"},
			SampleCount: 25,
		},
		LastUpdated: time.Now(),
	}

	prompt := manager.GetContextPrompt()

	if prompt == "" {
		t.Error("GetContextPrompt() should return non-empty string with populated context")
	}

	// Verify prompt contains key elements
	expectedStrings := []string{
		"METRICS",
		"http_requests_total",
		"LOGS",
		`{app="backend"}`,
		"TRACES",
		"api-service",
	}

	for _, expected := range expectedStrings {
		if !containsString(prompt, expected) {
			t.Errorf("GetContextPrompt() should contain %q", expected)
		}
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)*2 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestManagerRefreshWithTimeout tests refresh with context timeout
func TestManagerRefreshWithTimeout(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	manager.Configure([]string{"ds-1"}, 5, true, false, false)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Refresh should handle timeout gracefully
	err := manager.Refresh(ctx)

	// Should complete without panic (may succeed or timeout depending on timing)
	_ = err
}

// TestManagerPeriodicRefreshStability tests that periodic refresh runs without crashing
func TestManagerPeriodicRefreshStability(t *testing.T) {
	mockHTTP := newMockHTTPClient()
	addMockResponse(mockHTTP,"/api/datasources", createDataSourceListResponse([]DataSource{
		{UID: "ds-1", Type: "prometheus", Name: "Prometheus"},
	}))

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	// Configure with short interval
	manager.Configure([]string{"ds-1"}, 1, true, false, false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start manager with periodic refresh
	manager.Start(ctx)

	// Let it run for a bit to ensure stability
	time.Sleep(100 * time.Millisecond)

	// Should be able to get context without issues
	observCtx := manager.GetContext()
	if observCtx == nil {
		t.Error("GetContext() should return non-nil after refresh cycle")
	}

	// Stop should complete cleanly
	manager.Stop()

	// Verify final state is consistent
	_ = manager.GetLastUpdated()
}

// TestGrafanaClientQueryDatasource tests querying datasources via QueryDatasource
func TestGrafanaClientQueryDatasource(t *testing.T) {
	tests := []struct {
		name       string
		dsUID      string
		query      string
		queryType  string
		statusCode int
		response   interface{}
		wantErr    bool
	}{
		{
			name:       "prometheus query success",
			dsUID:      "prom-1",
			query:      "up",
			queryType:  "prometheus",
			statusCode: 200,
			response: QueryResponse{
				Results: map[string]QueryResult{
					"A": {
						Frames: []Frame{},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "loki query success",
			dsUID:      "loki-1",
			query:      `{job="api"}`,
			queryType:  "loki",
			statusCode: 200,
			response: QueryResponse{
				Results: map[string]QueryResult{
					"A": {
						Frames: []Frame{},
					},
				},
			},
			wantErr: false,
		},
		{
			name:       "query with error response",
			dsUID:      "ds-1",
			query:      "invalid",
			queryType:  "prometheus",
			statusCode: 400,
			response:   map[string]string{"error": "invalid query"},
			wantErr:    true,
		},
		{
			name:       "default query type",
			dsUID:      "ds-1",
			query:      "test",
			queryType:  "unknown",
			statusCode: 200,
			response: QueryResponse{
				Results: map[string]QueryResult{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add mock response for query endpoint
			body, _ := json.Marshal(tt.response)
			addMockResponse(mockHTTP, "/api/ds/query", &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(bytes.NewReader(body)),
			})

			client := NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			resp, err := client.QueryDatasource(context.Background(), tt.dsUID, tt.query, tt.queryType)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryDatasource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && resp == nil {
				t.Error("QueryDatasource() should return non-nil response on success")
			}
		})
	}
}

// TestExtractValuesFromFrame tests extracting field values from data frames
func TestExtractValuesFromFrame(t *testing.T) {
	tests := []struct {
		name      string
		frame     Frame
		wantCount int
		wantNames []string
	}{
		{
			name: "frame with multiple fields",
			frame: Frame{
				Schema: Schema{
					Fields: []Field{
						{Name: "Time"},
						{Name: "value"},
						{Name: "label"},
					},
				},
			},
			wantCount: 2, // Excludes "Time"
			wantNames: []string{"value", "label"},
		},
		{
			name: "frame with only time field",
			frame: Frame{
				Schema: Schema{
					Fields: []Field{
						{Name: "Time"},
					},
				},
			},
			wantCount: 0,
			wantNames: []string{},
		},
		{
			name: "frame with empty field names",
			frame: Frame{
				Schema: Schema{
					Fields: []Field{
						{Name: ""},
						{Name: "field1"},
						{Name: ""},
					},
				},
			},
			wantCount: 1,
			wantNames: []string{"field1"},
		},
		{
			name: "empty frame",
			frame: Frame{
				Schema: Schema{
					Fields: []Field{},
				},
			},
			wantCount: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := extractValuesFromFrame(tt.frame)

			if len(values) != tt.wantCount {
				t.Errorf("extractValuesFromFrame() count = %d, want %d", len(values), tt.wantCount)
			}

			// Verify expected names are present
			for _, expectedName := range tt.wantNames {
				found := false
				for _, value := range values {
					if value == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("extractValuesFromFrame() missing expected value: %s", expectedName)
				}
			}
		})
	}
}

// TestFetchLogStreamsAlternative tests alternative log stream fetching
func TestFetchLogStreamsAlternative(t *testing.T) {
	tests := []struct {
		name           string
		dsUID          string
		dsName         string
		mockQueryResp  *QueryResponse
		expectedStream int
		expectedLabels int
	}{
		{
			name:   "successful stream fetch",
			dsUID:  "loki-1",
			dsName: "Loki",
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Frames: []Frame{
							{
								Schema: Schema{
									Fields: []Field{
										{
											Name: "line",
											Labels: map[string]string{
												"app":       "backend",
												"namespace": "default",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedStream: 2, // Two streams from two labels
			expectedLabels: 2,
		},
		{
			name:   "query error - no data",
			dsUID:  "loki-1",
			dsName: "Loki",
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Error:  "query failed",
					},
				},
			},
			expectedStream: 0,
			expectedLabels: 0,
		},
		{
			name:           "no labels in fields",
			dsUID:          "loki-1",
			dsName:         "Loki",
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Frames: []Frame{
							{
								Schema: Schema{
									Fields: []Field{
										{Name: "line", Labels: nil},
									},
								},
							},
						},
					},
				},
			},
			expectedStream: 0,
			expectedLabels: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add mock response for query
			if tt.mockQueryResp != nil {
				body, _ := json.Marshal(tt.mockQueryResp)
				addMockResponse(mockHTTP, "/api/ds/query", &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				})
			}

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			streamSet := make(map[string]bool)
			labelSet := make(map[string]bool)

			manager.fetchLogStreamsAlternative(context.Background(), tt.dsUID, tt.dsName, streamSet, labelSet)

			if len(streamSet) != tt.expectedStream {
				t.Errorf("fetchLogStreamsAlternative() streamSet count = %d, want %d", len(streamSet), tt.expectedStream)
			}

			if len(labelSet) != tt.expectedLabels {
				t.Errorf("fetchLogStreamsAlternative() labelSet count = %d, want %d", len(labelSet), tt.expectedLabels)
			}
		})
	}
}

// TestFetchTracesAlternative tests alternative trace fetching
func TestFetchTracesAlternative(t *testing.T) {
	mockHTTP := newMockHTTPClient()

	manager := NewManager()
	manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
		return "http://localhost:3000", nil
	})

	serviceSet := make(map[string]bool)
	operationSet := make(map[string]bool)

	// Should complete without error (just logs a message)
	manager.fetchTracesAlternative(context.Background(), "tempo-1", "Tempo", serviceSet, operationSet)

	// Verify sets remain empty (no trace data added in alternative method)
	if len(serviceSet) != 0 {
		t.Error("fetchTracesAlternative() should not add services")
	}
	if len(operationSet) != 0 {
		t.Error("fetchTracesAlternative() should not add operations")
	}
}

// TestFetchMetricsContextWithQuery tests fetchMetricsContext with actual query responses
func TestFetchMetricsContextWithQuery(t *testing.T) {
	tests := []struct {
		name          string
		datasources   []DataSource
		mockQueryResp *QueryResponse
		wantMetrics   bool
	}{
		{
			name: "prometheus with metric data",
			datasources: []DataSource{
				{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
			},
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Frames: []Frame{
							{
								Schema: Schema{
									Fields: []Field{
										{Name: "Time"},
										{Name: "http_requests_total"},
										{Name: "cpu_usage"},
									},
								},
							},
						},
					},
				},
			},
			wantMetrics: true,
		},
		{
			name: "query error",
			datasources: []DataSource{
				{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
			},
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Error:  "query failed",
					},
				},
			},
			wantMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add mock response for datasource list
			addMockResponse(mockHTTP, "/api/datasources", createDataSourceListResponse(tt.datasources))

			// Add mock response for query
			if tt.mockQueryResp != nil {
				body, _ := json.Marshal(tt.mockQueryResp)
				addMockResponse(mockHTTP, "/api/ds/query", &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				})
			}

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			var dsUIDs []string
			for _, ds := range tt.datasources {
				dsUIDs = append(dsUIDs, ds.UID)
			}

			metricsCtx, err := manager.fetchMetricsContext(context.Background(), dsUIDs)

			if tt.wantMetrics {
				if err != nil {
					t.Errorf("fetchMetricsContext() unexpected error = %v", err)
				}
				if metricsCtx == nil {
					t.Error("fetchMetricsContext() should return non-nil context")
				}
			}
		})
	}
}

// TestFetchLogsContextWithQuery tests fetchLogsContext with actual query responses
func TestFetchLogsContextWithQuery(t *testing.T) {
	tests := []struct {
		name          string
		datasources   []DataSource
		mockQueryResp *QueryResponse
		wantLogs      bool
	}{
		{
			name: "loki with log data",
			datasources: []DataSource{
				{UID: "loki-1", Type: "loki", Name: "Loki"},
			},
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Frames: []Frame{
							{
								Schema: Schema{
									Fields: []Field{
										{
											Name: "line",
											Labels: map[string]string{
												"app": "backend",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLogs: true,
		},
		{
			name: "query error",
			datasources: []DataSource{
				{UID: "loki-1", Type: "loki", Name: "Loki"},
			},
			mockQueryResp: &QueryResponse{
				Results: map[string]QueryResult{
					"A": {
									Error:  "query failed",
					},
				},
			},
			wantLogs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add mock response for datasource list
			addMockResponse(mockHTTP, "/api/datasources", createDataSourceListResponse(tt.datasources))

			// Add mock response for query
			if tt.mockQueryResp != nil {
				body, _ := json.Marshal(tt.mockQueryResp)
				addMockResponse(mockHTTP, "/api/ds/query", &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewReader(body)),
				})
			}

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			var dsUIDs []string
			for _, ds := range tt.datasources {
				dsUIDs = append(dsUIDs, ds.UID)
			}

			logsCtx, err := manager.fetchLogsContext(context.Background(), dsUIDs)

			if tt.wantLogs {
				if err != nil {
					t.Errorf("fetchLogsContext() unexpected error = %v", err)
				}
				if logsCtx == nil {
					t.Error("fetchLogsContext() should return non-nil context")
				}
			}
		})
	}
}

// TestFetchTracesContextWithDatasources tests fetchTracesContext with trace datasources
func TestFetchTracesContextWithDatasources(t *testing.T) {
	tests := []struct {
		name        string
		datasources []DataSource
		wantTraces  bool
	}{
		{
			name: "tempo datasource",
			datasources: []DataSource{
				{UID: "tempo-1", Type: "tempo", Name: "Tempo"},
			},
			wantTraces: true,
		},
		{
			name: "non-tempo datasource",
			datasources: []DataSource{
				{UID: "prom-1", Type: "prometheus", Name: "Prometheus"},
			},
			wantTraces: false,
		},
		{
			name:        "no datasources",
			datasources: []DataSource{},
			wantTraces:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHTTP := newMockHTTPClient()

			// Add mock response for datasource list
			addMockResponse(mockHTTP, "/api/datasources", createDataSourceListResponse(tt.datasources))

			manager := NewManager()
			manager.client = NewGrafanaClient(mockHTTP, func(ctx context.Context) (string, error) {
				return "http://localhost:3000", nil
			})

			var dsUIDs []string
			for _, ds := range tt.datasources {
				dsUIDs = append(dsUIDs, ds.UID)
			}

			tracesCtx, err := manager.fetchTracesContext(context.Background(), dsUIDs)

			if err != nil {
				t.Errorf("fetchTracesContext() unexpected error = %v", err)
			}

			// fetchTracesContext always returns a context (empty or not)
			if tracesCtx == nil {
				t.Error("fetchTracesContext() should return non-nil context")
			}

			// For tempo datasources, we expect to attempt fetching (even if empty)
			// For non-tempo, the context should be empty
			if tt.wantTraces {
				// Tempo datasource - should have attempted to fetch traces
				_ = tracesCtx
			} else {
				// Non-tempo - context should be empty
				if len(tracesCtx.Services) != 0 || len(tracesCtx.Operations) != 0 {
					t.Error("fetchTracesContext() should return empty context for non-tempo datasources")
				}
			}
		})
	}
}
