package context

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Manager handles observability context fetching and caching
type Manager struct {
	client *GrafanaClient

	// Cached context
	mu          sync.RWMutex
	context     *ObservabilityContext
	lastUpdated time.Time

	// Configuration
	datasourceUIDs  []string
	refreshInterval time.Duration
	enableMetrics   bool
	enableLogs      bool
	enableTraces    bool

	// Background refresh
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewManager creates a new context manager
// Note: Currently the context manager has limited functionality due to SDK constraints
// A full implementation would require access to Grafana's internal HTTP client
func NewManager() *Manager {
	// For now, create a basic client - this will have limited functionality
	// In a production setup, you'd want to pass proper authentication
	client := NewGrafanaClient(http.DefaultClient, "")
	return &Manager{
		client: client,
		stopCh: make(chan struct{}),
	}
}

// Configure updates the manager configuration
func (m *Manager) Configure(datasourceUIDs []string, refreshMinutes int, enableMetrics, enableLogs, enableTraces bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.datasourceUIDs = datasourceUIDs
	m.refreshInterval = time.Duration(refreshMinutes) * time.Minute
	m.enableMetrics = enableMetrics
	m.enableLogs = enableLogs
	m.enableTraces = enableTraces
}

// Start begins the background context refresh
func (m *Manager) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.refreshLoop(ctx)
}

// Stop stops the background refresh
func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// refreshLoop periodically refreshes the context
func (m *Manager) refreshLoop(ctx context.Context) {
	defer m.wg.Done()

	// Initial fetch
	if err := m.Refresh(ctx); err != nil {
		backend.Logger.Error("Initial context refresh failed", "error", err)
	}

	// Only set up periodic refresh if interval is positive
	if m.refreshInterval <= 0 {
		backend.Logger.Warn("Context refresh interval not configured, periodic refresh disabled")
		<-m.stopCh // Wait for stop signal
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

// Refresh fetches new context from Grafana
func (m *Manager) Refresh(ctx context.Context) error {
	backend.Logger.Info("Refreshing observability context")

	newContext := &ObservabilityContext{}

	m.mu.RLock()
	datasourceUIDs := m.datasourceUIDs
	enableMetrics := m.enableMetrics
	enableLogs := m.enableLogs
	enableTraces := m.enableTraces
	m.mu.RUnlock()

	// If no datasources configured, try to find them automatically
	if len(datasourceUIDs) == 0 {
		datasources, err := m.client.ListDatasources(ctx)
		if err != nil {
			backend.Logger.Warn("Failed to list datasources", "error", err)
		} else {
			for _, ds := range datasources {
				datasourceUIDs = append(datasourceUIDs, ds.UID)
			}
		}
	}

	// Fetch metrics context
	if enableMetrics {
		metricsCtx, err := m.fetchMetricsContext(ctx, datasourceUIDs)
		if err != nil {
			backend.Logger.Warn("Failed to fetch metrics context", "error", err)
		} else {
			newContext.Metrics = metricsCtx
		}
	}

	// Fetch logs context
	if enableLogs {
		logsCtx, err := m.fetchLogsContext(ctx, datasourceUIDs)
		if err != nil {
			backend.Logger.Warn("Failed to fetch logs context", "error", err)
		} else {
			newContext.Logs = logsCtx
		}
	}

	// Fetch traces context
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

// GetContext returns the current cached context
func (m *Manager) GetContext() *ObservabilityContext {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.context == nil {
		return nil
	}

	// Return a copy to prevent modifications
	contextCopy := *m.context
	return &contextCopy
}

// GetContextPrompt returns the context formatted as a prompt for LLM
func (m *Manager) GetContextPrompt() string {
	ctx := m.GetContext()
	if ctx == nil {
		return ""
	}
	return ctx.BuildPrompt()
}

// GetLastUpdated returns when the context was last updated
func (m *Manager) GetLastUpdated() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdated
}
