package context

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Manager struct {
	client *GrafanaClient

	mu          sync.RWMutex
	context     *ObservabilityContext
	lastUpdated time.Time

	dashboardsMu        sync.RWMutex
	dashboards          map[string]*DashboardResponse
	dashboardsFetchedAt time.Time

	datasourceUIDs  []string
	refreshInterval time.Duration
	enableMetrics   bool
	enableLogs      bool
	enableTraces bool

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewManager() *Manager {
	urlProvider := func(ctx context.Context) (string, error) {
		cfg := backend.GrafanaConfigFromContext(ctx)
		grafanaURL, err := cfg.AppURL()
		if err != nil {
			return "", fmt.Errorf("failed to get Grafana URL from context: %w", err)
		}
		return grafanaURL, nil
	}

	client := NewGrafanaClient(http.DefaultClient, urlProvider)
	return &Manager{
		client: client,
		stopCh: make(chan struct{}),
	}
}

func (m *Manager) Configure(datasourceUIDs []string, refreshMinutes int, enableMetrics, enableLogs, enableTraces bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.datasourceUIDs = datasourceUIDs
	m.refreshInterval = time.Duration(refreshMinutes) * time.Minute
	m.enableMetrics = enableMetrics
	m.enableLogs = enableLogs
	m.enableTraces = enableTraces
}

func (m *Manager) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.refreshLoop(ctx)
}

func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *Manager) refreshLoop(ctx context.Context) {
	defer m.wg.Done()

	if err := m.Refresh(ctx); err != nil {
		backend.Logger.Error("Initial context refresh failed", "error", err)
	}

	if m.refreshInterval <= 0 {
		backend.Logger.Warn("Context refresh interval not configured, periodic refresh disabled")
		<-m.stopCh 
		return
	}

	ticker := time.NewTicker(m.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.Refresh(ctx); err != nil {
				backend.Logger.Error("Context refresh failed", "error", err)
			}
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) Refresh(ctx context.Context) error {
	backend.Logger.Info("Refreshing observability context")

	newContext := &ObservabilityContext{}

	m.mu.RLock()
	datasourceUIDs := m.datasourceUIDs
	enableMetrics := m.enableMetrics
	enableLogs := m.enableLogs
	enableTraces := m.enableTraces
	m.mu.RUnlock()

	if len(datasourceUIDs) == 0 {
		listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		datasources, err := m.client.ListDatasources(listCtx)
		if err != nil {
			backend.Logger.Warn("Failed to list datasources - context features will be limited",
				"error", err,
				"hint", "Configure specific datasource UIDs in plugin settings, or ensure the plugin has access to Grafana API")
		} else {
			for _, ds := range datasources {
				datasourceUIDs = append(datasourceUIDs, ds.UID)
			}
			backend.Logger.Debug("Auto-discovered datasources", "count", len(datasourceUIDs))
		}
	}

	if enableMetrics {
		metricsCtx, err := m.fetchMetricsContext(ctx, datasourceUIDs)
		if err != nil {
			backend.Logger.Warn("Failed to fetch metrics context", "error", err)
		} else {
			newContext.Metrics = metricsCtx
		}
	}

	if enableLogs {
		logsCtx, err := m.fetchLogsContext(ctx, datasourceUIDs)
		if err != nil {
			backend.Logger.Warn("Failed to fetch logs context", "error", err)
		} else {
			newContext.Logs = logsCtx
		}
	}

	if enableTraces {
		tracesCtx, err := m.fetchTracesContext(ctx, datasourceUIDs)
		if err != nil {
			backend.Logger.Warn("Failed to fetch traces context", "error", err)
		} else {
			newContext.Traces = tracesCtx
		}
	}

	newContext.LastUpdated = time.Now()

	m.mu.Lock()
	m.context = newContext
	m.lastUpdated = newContext.LastUpdated
	m.mu.Unlock()

	backend.Logger.Info("Context refresh complete",
		"metrics", newContext.Metrics != nil,
		"logs", newContext.Logs != nil,
		"traces", newContext.Traces != nil,
	)

	return nil
}

func (m *Manager) GetContext() *ObservabilityContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.context == nil {
		return nil
	}

	contextCopy := *m.context
	return &contextCopy
}

func (m *Manager) GetContextPrompt() string {
	ctx := m.GetContext()
	if ctx == nil {
		return ""
	}
	return ctx.BuildPrompt()
}

func (m *Manager) GetLastUpdated() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdated
}

func (m *Manager) FetchReferenceDashboards(ctx context.Context, uids []string) error {
	m.dashboardsMu.Lock()
	defer m.dashboardsMu.Unlock()

	dashboards := make(map[string]*DashboardResponse)
	for _, uid := range uids {
		dashboard, err := m.client.GetDashboard(ctx, uid)
		if err != nil {
			backend.Logger.Warn("Failed to fetch reference dashboard",
				"uid", uid, "error", err)
			continue 
		}
		dashboards[uid] = dashboard
	}

	m.dashboards = dashboards
	m.dashboardsFetchedAt = time.Now()
	backend.Logger.Info("Fetched reference dashboards", "count", len(dashboards))
	return nil
}

func (m *Manager) GetReferenceDashboards() map[string]*DashboardResponse {
	m.dashboardsMu.RLock()
	defer m.dashboardsMu.RUnlock()
	return m.dashboards
}
