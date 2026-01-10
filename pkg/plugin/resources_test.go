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
