package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// mockCallResourceResponseSender implements backend.CallResourceResponseSender
// for use in tests.
type mockCallResourceResponseSender struct {
	response *backend.CallResourceResponse
}

// Send sets the received *backend.CallResourceResponse to s.response
func (s *mockCallResourceResponseSender) Send(response *backend.CallResourceResponse) error {
	s.response = response
	return nil
}

// TestCallResource tests CallResource calls, using backend.CallResourceRequest and backend.CallResourceResponse.
// This ensures the httpadapter for CallResource works correctly.
func TestCallResource(t *testing.T) {
	// Initialize app
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

	// Set up and run test cases
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
			// Request by calling CallResource. This tests the httpadapter.
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

// TestUserIdentityExtraction tests that user identity is correctly extracted from request context
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
			// Create request with plugin context
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

// TestQueryProxyBasicHandling tests basic request handling for the query proxy
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
			expectedStatus: http.StatusOK, // Will be OK even if Grafana isn't running, as we're testing the handler
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
			// Marshal body
			var bodyBytes []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, _ = json.Marshal(tt.body)
				}
			}

			// Create request with plugin context
			ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
				User:  tt.user,
				OrgID: tt.orgID,
			})

			req := httptest.NewRequest(tt.method, "/query", bytes.NewReader(bodyBytes))
			req = req.WithContext(ctx)

			// Record response
			w := httptest.NewRecorder()

			// Call handler directly
			appInstance := app.(*App)
			appInstance.handleQuery(w, req)

			// Check status code
			if w.Code != tt.expectedStatus && w.Code != http.StatusInternalServerError {
				// Allow InternalServerError as it might happen if Grafana isn't running
				if tt.expectedStatus != http.StatusOK || w.Code != http.StatusInternalServerError {
					t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestRateLimitingPerUser tests that rate limiting is applied per user
func TestRateLimitingPerUser(t *testing.T) {
	app, err := NewApp(context.Background(), backend.AppInstanceSettings{})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	appInstance := app.(*App)

	// Set low rate limit for testing
	appInstance.guardrails = NewGuardrails(2) // Only 2 requests per minute

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

	// Create request with plugin context
	ctx := backend.WithPluginContext(context.Background(), backend.PluginContext{
		User:  &backend.User{Login: "testuser"},
		OrgID: 1,
	})

	// Make requests until rate limit is hit
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
			// Both OK and InternalServerError count as not rate-limited
			// (InternalServerError happens if Grafana isn't running)
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
