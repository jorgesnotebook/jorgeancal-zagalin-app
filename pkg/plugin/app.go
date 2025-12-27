package plugin

import (
	"context"
	"net/http"
	"os"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	contextmgr "github.com/jorgeancal/zagalin/pkg/plugin/context"
)

var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
)

type App struct {
	backend.CallResourceHandler
	settings        *Settings
	contextManager  *contextmgr.Manager
	guardrails      *Guardrails
	storage         *UserStorage
	datasourceCache *datasourceCache
	queryValidator  *QueryValidator
}

func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	var app App

	dataDir := os.Getenv("GF_PLUGIN_APP_DATA_PATH")
	if dataDir == "" {
		dataDir = "./data" // Fallback for development
	}
	app.storage = NewUserStorage(dataDir)
	backend.Logger.Info("User storage initialized", "dataDir", dataDir)

	app.contextManager = contextmgr.NewManager()
	app.datasourceCache = newDatasourceCache()
	backend.Logger.Debug("Datasource cache initialized")

	settings, err := LoadSettings(appSettings.JSONData, appSettings.DecryptedSecureJSONData)

	maxRequestsPerMinute := 60 // Default
	if settings != nil {
		maxRequestsPerMinute = settings.MaxRequestsPerMinute
	}
	app.guardrails = NewGuardrails(maxRequestsPerMinute)

	if err != nil {
		backend.Logger.Error("Failed to load settings", "error", err)
		app.settings = nil
	} else {
		app.settings = settings
		backend.Logger.Info("Settings loaded successfully",
			"maxRequestsPerMinute", settings.MaxRequestsPerMinute,
			"monthlyBudgetUSD", settings.MonthlyBudgetUSD,
		)

		app.contextManager.Start(ctx)
		backend.Logger.Info("Context manager started")

		// Initialize query validator
		app.queryValidator = NewQueryValidator(&settings.QueryValidation, &app)
		backend.Logger.Info("Query validator initialized",
			"enabled", settings.QueryValidation.Enabled,
			"strictMode", settings.QueryValidation.StrictMode,
			"llmValidation", settings.QueryValidation.EnableLLMValidation,
		)
	}

	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	return &app, nil
}

func (a *App) Dispose() {
	if a.contextManager != nil {
		a.contextManager.Stop()
		backend.Logger.Info("Context manager stopped")
	}

	if a.guardrails != nil && a.guardrails.rateLimiter != nil {
		a.guardrails.rateLimiter.Stop()
		backend.Logger.Info("Guardrails stopped")
	}
}

func (a *App) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
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
		message += " Context refresh: every " + strconv.Itoa(settings.ContextRefreshMinutes) + " minutes."
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: message,
	}, nil
}
