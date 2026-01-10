package plugin

import (
	"context"
	"net/http"
	"strconv"
	"time"

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
	settings           *Settings
	contextManager     *contextmgr.Manager
	guardrails         *Guardrails
	datasourceCache    *datasourceCache
	queryValidator     *QueryValidator
	runManager         *RunManager
	otelRegistry       *OTelLabelRegistry
	grafanaQueryClient *GrafanaQueryClient
}

func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	var app App

	app.contextManager = contextmgr.NewManager()
	app.datasourceCache = newDatasourceCache()
	app.otelRegistry = NewOTelLabelRegistry()
	backend.Logger.Debug("Datasource cache initialized")
	backend.Logger.Debug("OTel label registry initialized")

	app.runManager = NewRunManager(backend.Logger)
	backend.Logger.Info("Run manager initialized")

	settings, err := LoadSettings(appSettings.JSONData, appSettings.DecryptedSecureJSONData)

	maxRequestsPerMinute := 60
	if settings != nil {
		maxRequestsPerMinute = settings.MaxRequestsPerMinute
	}
	app.guardrails = NewGuardrails(maxRequestsPerMinute)

	grafanaCfg := backend.GrafanaConfigFromContext(ctx)
	grafanaURL, urlErr := grafanaCfg.AppURL()
	if urlErr != nil {
		backend.Logger.Warn("Failed to get Grafana URL, using default", "error", urlErr)
		grafanaURL = "http://localhost:3000"
	}

	if err != nil {
		backend.Logger.Error("Failed to load settings", "error", err)
		app.settings = nil
	} else {
		app.settings = settings
		app.settings.GrafanaURL = grafanaURL

		if settings.ServiceAccountToken != "" {
			app.grafanaQueryClient = NewGrafanaQueryClient(grafanaURL, settings.ServiceAccountToken)
			backend.Logger.Info("Grafana query client initialized", "grafanaURL", grafanaURL)
		} else {
			backend.Logger.Warn("No service account token configured, Grafana query client not initialized")
		}
		backend.Logger.Info("Settings loaded successfully",
			"maxRequestsPerMinute", settings.MaxRequestsPerMinute,
			"monthlyBudgetUSD", settings.MonthlyBudgetUSD,
		)

		app.contextManager.Start(ctx)
		backend.Logger.Info("Context manager started")

		if len(settings.ReferenceDashboards) > 0 {
			go func() {
				fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := app.contextManager.FetchReferenceDashboards(fetchCtx, settings.ReferenceDashboards); err != nil {
					backend.Logger.Warn("Failed to fetch reference dashboards", "error", err)
				}
			}()
		}

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

	if a.runManager != nil {
		a.runManager.Stop()
		backend.Logger.Info("Run manager stopped")
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
