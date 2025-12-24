package plugin

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	contextmgr "github.com/jorgeancal/zagalin/pkg/plugin/context"
)

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
)

// App is the Zagalin app plugin.
// All LLM functionality is provided by grafana-llm-app plugin.
type App struct {
	backend.CallResourceHandler
	settings       *Settings
	contextManager *contextmgr.Manager
	guardrails     *Guardrails
}

// NewApp creates a new *App instance.
func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	var app App

	// Initialize context manager
	app.contextManager = contextmgr.NewManager()

	// Load and validate settings
	settings, err := LoadSettings(appSettings.JSONData, appSettings.DecryptedSecureJSONData)

	// Initialize guardrails
	maxRequestsPerMinute := 60 // Default
	if settings != nil {
		maxRequestsPerMinute = settings.MaxRequestsPerMinute
	}
	app.guardrails = NewGuardrails(maxRequestsPerMinute)

	if err != nil {
		backend.Logger.Error("Failed to load settings", "error", err)
		// Return app with nil settings - plugin can still work with defaults
		app.settings = nil
	} else {
		app.settings = settings
		backend.Logger.Info("Settings loaded successfully",
			"maxRequestsPerMinute", settings.MaxRequestsPerMinute,
			"monthlyBudgetUSD", settings.MonthlyBudgetUSD,
		)

		// Context manager is always enabled
		// It will auto-detect available context
		app.contextManager.Start(ctx)
		backend.Logger.Info("Context manager started")
	}

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {
	// Stop context manager
	if a.contextManager != nil {
		a.contextManager.Stop()
		backend.Logger.Info("Context manager stopped")
	}

	// Stop guardrails
	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		a.guardrails.rateLimiter.Stop()
		backend.Logger.Info("Guardrails stopped")
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (a *App) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Re-validate settings with the current request data
	settings, err := LoadSettings(req.PluginContext.AppInstanceSettings.JSONData,
		req.PluginContext.AppInstanceSettings.DecryptedSecureJSONData)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Settings validation failed: " + err.Error(),
		}, nil
	}

	message := "Zagalin plugin is ready. Using grafana-llm-app for LLM functionality."
	if settings != nil {
		// Include settings info in health check
		message += " Context refresh: every " + string(rune(settings.ContextRefreshMinutes)) + " minutes."
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: message,
	}, nil
}
