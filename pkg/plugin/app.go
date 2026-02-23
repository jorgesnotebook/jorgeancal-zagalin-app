package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	contextmgr "github.com/jorgeancal/zagalin/pkg/plugin/context"
	"github.com/jorgeancal/zagalin/pkg/plugin/skills"
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
	llmClient          *LLMClient
	versionDetector    *VersionDetector
	skillRegistry      *skills.SkillRegistry
	dashboardFetchStatus struct {
		sync.RWMutex
		LastAttempt  time.Time
		LastSuccess  time.Time
		LastError    error
		AttemptCount int
		Status       string // "idle", "fetching", "success", "failed"
	}
}

func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	var app App

	app.versionDetector = NewVersionDetector(backend.Logger)
	backend.Logger.Info("Zagalin plugin initializing", "minimumGrafanaVersion", MinimumSupportedVersion.String())

	app.contextManager = contextmgr.NewManager()
	app.datasourceCache = newDatasourceCache()
	app.otelRegistry = NewOTelLabelRegistry()
	backend.Logger.Debug("Datasource cache initialized")
	backend.Logger.Debug("OTel label registry initialized")

	// Initialize skill registry
	app.skillRegistry = skills.NewSkillRegistry()
	if err := app.skillRegistry.Load(); err != nil {
		backend.Logger.Error("Failed to load skills", "error", err)
	} else {
		backend.Logger.Info("Skills loaded successfully", "count", len(app.skillRegistry.ListSkills()))
	}

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

			// Initialize LLM client for query validation
			app.llmClient = NewLLMClient(grafanaURL, settings.ServiceAccountToken, http.DefaultClient, backend.Logger)
			backend.Logger.Info("LLM client initialized for query validation")
		} else {
			backend.Logger.Warn("No service account token configured, Grafana query client and LLM client not initialized")
		}
		backend.Logger.Info("Settings loaded successfully",
			"maxRequestsPerMinute", settings.MaxRequestsPerMinute,
			"monthlyBudgetUSD", settings.MonthlyBudgetUSD,
		)

		app.contextManager.Start(ctx)
		backend.Logger.Info("Context manager started")

		if len(settings.ReferenceDashboards) > 0 {
			go func() {
				maxRetries := 3
				backoff := 5 * time.Second

				for attempt := 1; attempt <= maxRetries; attempt++ {
					app.dashboardFetchStatus.Lock()
					app.dashboardFetchStatus.Status = "fetching"
					app.dashboardFetchStatus.LastAttempt = time.Now().UTC()
					app.dashboardFetchStatus.AttemptCount = attempt
					app.dashboardFetchStatus.Unlock()

					fetchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					err := app.contextManager.FetchReferenceDashboards(fetchCtx, settings.ReferenceDashboards)
					cancel()

					app.dashboardFetchStatus.Lock()
					if err != nil {
						app.dashboardFetchStatus.Status = "failed"
						app.dashboardFetchStatus.LastError = err
						app.dashboardFetchStatus.Unlock()

						backend.Logger.Warn("Failed to fetch reference dashboards",
							"attempt", attempt,
							"maxRetries", maxRetries,
							"error", err,
						)

						if attempt < maxRetries {
							time.Sleep(backoff)
							backoff *= 2 // Exponential backoff: 5s, 10s, 20s
						}
					} else {
						app.dashboardFetchStatus.Status = "success"
						app.dashboardFetchStatus.LastSuccess = time.Now().UTC()
						app.dashboardFetchStatus.Unlock()

						backend.Logger.Info("Reference dashboards fetched successfully", "attempt", attempt)
						return
					}
				}

				backend.Logger.Error("Failed to fetch reference dashboards after max retries", "maxRetries", maxRetries)
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

	// Add background job status
	jsonDetails := map[string]interface{}{
		"contextManager": map[string]interface{}{
			"lastUpdated": a.contextManager.GetLastUpdated(),
		},
		"runManager": map[string]interface{}{
			"activeRuns": a.runManager.GetRunCount(),
		},
	}

	// Dashboard fetch status
	a.dashboardFetchStatus.RLock()
	dashboardStatus := map[string]interface{}{
		"status":       a.dashboardFetchStatus.Status,
		"lastAttempt":  a.dashboardFetchStatus.LastAttempt,
		"lastSuccess":  a.dashboardFetchStatus.LastSuccess,
		"attemptCount": a.dashboardFetchStatus.AttemptCount,
	}
	if a.dashboardFetchStatus.LastError != nil {
		dashboardStatus["lastError"] = a.dashboardFetchStatus.LastError.Error()
	}
	a.dashboardFetchStatus.RUnlock()
	jsonDetails["referenceDashboards"] = dashboardStatus

	// Add Grafana version information
	detectedVersion := a.versionDetector.GetVersion()
	versionInfo := map[string]interface{}{
		"detected":       detectedVersion.String(),
		"isAvailable":    detectedVersion.IsAvailable,
		"isSupported":    detectedVersion.IsSupported(),
		"minimumVersion": MinimumSupportedVersion.String(),
		"warnings":       detectedVersion.VersionWarnings(),
	}
	jsonDetails["version"] = versionInfo

	jsonData, _ := json.Marshal(jsonDetails)

	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusOk,
		Message:     message,
		JSONDetails: jsonData,
	}, nil
}
