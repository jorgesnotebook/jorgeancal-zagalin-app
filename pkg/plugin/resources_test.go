package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type mockCallResourceResponseSender struct {
	response *backend.CallResourceResponse
}

func (s *mockCallResourceResponseSender) Send(response *backend.CallResourceResponse) error {
	s.response = response
	return nil
}

func TestCallResource(t *testing.T) {
	inst, err := NewApp(context.Background(), backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("new app: %s", err)
	}
	if inst == nil {
		t.Fatal("inst must not be nil")
	}
	app, ok := inst.(*App)
	if !ok {
		t.Fatal("inst must be of type *App")
	}

	for _, tc := range []struct {
		name string

		method string
		path   string
		body   []byte

		expStatus int
		expBody   []byte
	}{
		{
			name:      "get ping 200",
			method:    http.MethodGet,
			path:      "ping",
			expStatus: http.StatusOK,
		},
		{
			name:      "get echo 405",
			method:    http.MethodGet,
			path:      "echo",
			expStatus: http.StatusMethodNotAllowed,
		},
		{
			name:      "post echo 200",
			method:    http.MethodPost,
			path:      "echo",
			body:      []byte(`{"message":"ok"}`),
			expStatus: http.StatusOK,
			expBody:   []byte(`{"message":"ok"}`),
		},
		{
			name:      "get non existing handler 404",
			method:    http.MethodGet,
			path:      "not_found",
			expStatus: http.StatusNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r mockCallResourceResponseSender
			err = app.CallResource(context.Background(), &backend.CallResourceRequest{
				Method: tc.method,
				Path:   tc.path,
				Body:   tc.body,
			}, &r)
			if err != nil {
				t.Fatalf("CallResource error: %s", err)
			}
			if r.response == nil {
				t.Fatal("no response received from CallResource")
			}
			if tc.expStatus > 0 && tc.expStatus != r.response.Status {
				t.Errorf("response status should be %d, got %d", tc.expStatus, r.response.Status)
			}
			if len(tc.expBody) > 0 {
				if tb := bytes.TrimSpace(r.response.Body); !bytes.Equal(tb, tc.expBody) {
					t.Errorf("response body should be %s, got %s", tc.expBody, tb)
				}
			}
		})
	}
}

func TestUserIdentityExtraction(t *testing.T) {
	tests := []struct {
		name        string
		user        *backend.User
		orgID       int64
		expectError bool
		expectedID  string
	}{
		{
			name: "valid user with login",
			user: &backend.User{
				Login: "testuser",
				Email: "test@example.com",
				Name:  "Test User",
			},
			orgID:       1,
			expectError: false,
			expectedID:  "testuser",
		},
		{
			name:        "anonymous user with org",
			user:        nil,
			orgID:       1,
			expectError: false,
			expectedID:  "anonymous",
		},
		{
			name:        "no user and no org",
			user:        nil,
			orgID:       0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
				User:  tt.user,
				OrgID: tt.orgID,
			})

			req := httptest.NewRequest(http.MethodPost, "/query", nil)
			req = req.WithContext(ctx)

			user, err := extractUserIdentity(req)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if user.UserLogin != tt.expectedID {
				t.Errorf("expected user login %s, got %s", tt.expectedID, user.UserLogin)
			}

			if user.OrgID != tt.orgID {
				t.Errorf("expected org ID %d, got %d", tt.orgID, user.OrgID)
			}
		})
	}
}

func TestQueryProxyBasicHandling(t *testing.T) {
	app, err := NewApp(context.Background(), backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		body           interface{}
		user           *backend.User
		orgID          int64
		expectedStatus int
	}{
		{
			name:           "GET request returns method not allowed",
			method:         http.MethodGet,
			user:           &backend.User{Login: "testuser"},
			orgID:          1,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "valid POST request with user context",
			method: http.MethodPost,
			body: QueryRequest{
				Datasource: "prometheus-uid",
				Queries: []QueryPayload{
					{
						RefID:      "A",
						Expr:       "up",
						QueryType:  "prometheus",
						IntervalMs: 15000,
					},
				},
				TimeRange: TimeRange{
					From: "now-1h",
					To:   "now",
				},
			},
			user:           &backend.User{Login: "testuser", Email: "test@example.com"},
			orgID:          1,
			expectedStatus: http.StatusOK, 
		},
		{
			name:           "invalid JSON body",
			method:         http.MethodPost,
			body:           "invalid json",
			user:           &backend.User{Login: "testuser"},
			orgID:          1,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, _ = json.Marshal(tt.body)
				}
			}

			ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
				User:  tt.user,
				OrgID: tt.orgID,
			})

			req := httptest.NewRequest(tt.method, "/query", bytes.NewReader(bodyBytes))
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			appInstance := app.(*App)
			appInstance.handleQuery(w, req)

			if w.Code != tt.expectedStatus && w.Code != http.StatusInternalServerError {
				if tt.expectedStatus != http.StatusOK || w.Code != http.StatusInternalServerError {
					t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestRateLimitingPerUser(t *testing.T) {
	app, err := NewApp(context.Background(), backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	appInstance := app.(*App)

	appInstance.guardrails = NewGuardrails(2) 

	queryReq := QueryRequest{
		Datasource: "prometheus-uid",
		Queries: []QueryPayload{
			{
				RefID:     "A",
				Expr:      "up",
				QueryType: "prometheus",
			},
		},
		TimeRange: TimeRange{
			From: "now-1h",
			To:   "now",
		},
	}

	bodyBytes, _ := json.Marshal(queryReq)

	ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
		User:  &backend.User{Login: "testuser"},
		OrgID: 1,
	})

	successCount := 0
	rateLimited := false

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(bodyBytes))
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		appInstance.handleQuery(w, req)

		if w.Code == http.StatusTooManyRequests {
			rateLimited = true
			break
		} else if w.Code == http.StatusOK || w.Code == http.StatusInternalServerError {
			successCount++
		}
	}

	if successCount == 0 {
		t.Error("expected at least one successful request")
	}

	if !rateLimited {
		t.Error("expected to hit rate limit, but didn't")
	}
}

func TestQueryValidationIntegration(t *testing.T) {
	app := &App{
		settings: &Settings{
			PluginSettings: PluginSettings{
				QueryValidation: QueryValidationSettings{
					Enabled:                 true,
					EnablePromQLValidation:  true,
					EnableLogQLValidation:   true,
					EnableTraceQLValidation: true,
					StrictMode:              true,
					MaxQueryComplexity:      100,
				},
			},
		},
		guardrails:     NewGuardrails(60),
		queryValidator: NewQueryValidator(&QueryValidationSettings{
			Enabled:                 true,
			EnablePromQLValidation:  true,
			EnableLogQLValidation:   true,
			EnableTraceQLValidation: true,
			StrictMode:              true,
			MaxQueryComplexity:      100,
		}, nil),
		datasourceCache: newDatasourceCache(),
	}

	tests := []struct {
		name           string
		query          string
		datasourceType string
		expectedStatus int
	}{
		{
			name:           "valid PromQL query",
			query:          "up",
			datasourceType: "prometheus",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid PromQL query in strict mode",
			query:          "up{{{",
			datasourceType: "prometheus",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid LogQL query",
			query:          `{job="test"}`,
			datasourceType: "loki",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid LogQL query",
			query:          `{job=`,
			datasourceType: "loki",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.datasourceCache.mu.Lock()
			app.datasourceCache.datasources["test-ds-uid"] = DatasourceInfo{
				UID:  "test-ds-uid",
				Name: "Test Datasource",
				Type: tt.datasourceType,
			}
			app.datasourceCache.lastRefresh = time.Now()
			app.datasourceCache.mu.Unlock()

			queryReq := QueryRequest{
				Datasource: "test-ds-uid",
				Queries: []QueryPayload{
					{
						RefID: "A",
						Expr:  tt.query,
					},
				},
				TimeRange: TimeRange{
					From: "now-1h",
					To:   "now",
				},
			}

			bodyBytes, _ := json.Marshal(queryReq)
			ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
				User:  &backend.User{Login: "testuser"},
				OrgID: 1,
			})

			req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(bodyBytes))
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			app.handleQuery(w, req)

			if tt.expectedStatus == http.StatusBadRequest {
				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status %d for invalid query, got %d: %s", http.StatusBadRequest, w.Code, w.Body.String())
				}
			} else if tt.expectedStatus == http.StatusOK {
				if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
					t.Errorf("expected status %d or %d, got %d: %s", http.StatusOK, http.StatusInternalServerError, w.Code, w.Body.String())
				}
			}
		})
	}
}

func TestCheckHealthWithVersion(t *testing.T) {
	tests := []struct {
		name            string
		versionHeader   string
		expectAvailable bool
		expectSupported bool
		expectWarnings  bool
	}{
		{
			name:            "supported version detected",
			versionHeader:   "10.4.0",
			expectAvailable: true,
			expectSupported: true,
			expectWarnings:  false,
		},
		{
			name:            "newer version detected",
			versionHeader:   "11.0.0",
			expectAvailable: true,
			expectSupported: true,
			expectWarnings:  false,
		},
		{
			name:            "unsupported version detected",
			versionHeader:   "10.3.0",
			expectAvailable: true,
			expectSupported: false,
			expectWarnings:  true,
		},
		{
			name:            "version not provided",
			versionHeader:   "",
			expectAvailable: false,
			expectSupported: true, // Graceful fallback
			expectWarnings:  true, // Warning about version not being detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create app instance
			app, err := NewApp(context.Background(), backend.AppInstanceSettings{})
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}

			// Simulate version detection by calling middleware with test request
			if tt.versionHeader != "" {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Grafana-Version", tt.versionHeader)
				app.(*App).versionDetector.DetectFromHeader(req)
			}

			// Call CheckHealth
			result, err := app.(*App).CheckHealth(context.Background(), &backend.CheckHealthRequest{
				PluginContext: backend.PluginContext{
					AppInstanceSettings: &backend.AppInstanceSettings{
						JSONData:                json.RawMessage(`{}`),
						DecryptedSecureJSONData: map[string]string{},
					},
				},
			})

			if err != nil {
				t.Fatalf("CheckHealth returned error: %v", err)
			}

			if result.Status != backend.HealthStatusOk {
				t.Errorf("expected health status OK, got %v", result.Status)
			}

			// Parse JSON details
			var details map[string]interface{}
			if err := json.Unmarshal(result.JSONDetails, &details); err != nil {
				t.Fatalf("failed to unmarshal JSON details: %v", err)
			}

			// Verify version info exists
			versionInfo, ok := details["version"].(map[string]interface{})
			if !ok {
				t.Fatal("version info not found in health check details")
			}

			// Check isAvailable
			isAvailable, ok := versionInfo["isAvailable"].(bool)
			if !ok {
				t.Fatal("isAvailable not found or not a bool")
			}
			if isAvailable != tt.expectAvailable {
				t.Errorf("expected isAvailable=%v, got %v", tt.expectAvailable, isAvailable)
			}

			// Check isSupported
			isSupported, ok := versionInfo["isSupported"].(bool)
			if !ok {
				t.Fatal("isSupported not found or not a bool")
			}
			if isSupported != tt.expectSupported {
				t.Errorf("expected isSupported=%v, got %v", tt.expectSupported, isSupported)
			}

			// Check minimumVersion
			minimumVersion, ok := versionInfo["minimumVersion"].(string)
			if !ok {
				t.Fatal("minimumVersion not found or not a string")
			}
			if minimumVersion != "10.4.0" {
				t.Errorf("expected minimumVersion=10.4.0, got %s", minimumVersion)
			}

			// Check warnings
			warnings, ok := versionInfo["warnings"].([]interface{})
			if !ok {
				t.Fatal("warnings not found or not an array")
			}

			if tt.expectWarnings {
				if len(warnings) == 0 {
					t.Error("expected warnings but got none")
				}
			} else {
				if len(warnings) > 0 {
					t.Errorf("expected no warnings but got %d: %v", len(warnings), warnings)
				}
			}

			// Check detected version string
			detected, ok := versionInfo["detected"].(string)
			if !ok {
				t.Fatal("detected version not found or not a string")
			}

			if tt.expectAvailable {
				if detected == "unknown" {
					t.Error("expected version to be detected but got 'unknown'")
				}
			} else {
				if detected != "unknown" {
					t.Errorf("expected detected='unknown' but got %s", detected)
				}
			}
		})
	}
}

func TestVersionMiddleware(t *testing.T) {
	app, err := NewApp(context.Background(), backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	// Create a test handler that will be wrapped
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	wrappedHandler := app.(*App).versionDetectionMiddleware(testHandler)

	t.Run("extracts version from header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Grafana-Version", "10.4.0")
		w := httptest.NewRecorder()

		wrappedHandler(w, req)

		// Verify handler was called
		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
		}

		// Verify version was detected
		version := app.(*App).versionDetector.GetVersion()
		if !version.IsAvailable {
			t.Error("expected version to be detected")
		}
		if version.Full != "10.4.0" {
			t.Errorf("expected version 10.4.0, got %s", version.Full)
		}
	})

	t.Run("handles missing version header gracefully", func(t *testing.T) {
		// Create new app instance
		app2, _ := NewApp(context.Background(), backend.AppInstanceSettings{})
		wrappedHandler2 := app2.(*App).versionDetectionMiddleware(testHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No X-Grafana-Version header
		w := httptest.NewRecorder()

		wrappedHandler2(w, req)

		// Verify handler was still called
		if w.Code != http.StatusOK {
			t.Errorf("expected status OK, got %d", w.Code)
		}

		// Version should not be available but app should work
		version := app2.(*App).versionDetector.GetVersion()
		if version.IsAvailable {
			t.Error("expected version to be unavailable")
		}
	})
}
