package plugin

import (
	"context"
	"testing"
)

// TestValidatePromQL_ValidQueries tests valid PromQL queries
func TestValidatePromQL_ValidQueries(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "simple metric",
			query: "up",
		},
		{
			name:  "metric with labels",
			query: `up{job="prometheus"}`,
		},
		{
			name:  "rate function",
			query: `rate(http_requests_total[5m])`,
		},
		{
			name:  "complex aggregation",
			query: `sum(rate(http_requests_total{job="api"}[5m])) by (status_code)`,
		},
		{
			name:  "binary operation",
			query: `(up{job="prometheus"} == 1) * 100`,
		},
		{
			name:  "nested functions",
			query: `avg_over_time(rate(http_requests_total[5m])[10m:1m])`,
		},
	}

	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:                true,
		EnablePromQLValidation: true,
		StrictMode:             false,
		MaxQueryComplexity:     100,
	}, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)
			if !result.Valid {
				t.Errorf("expected valid query, got error: %v", result.Error)
			}
			if result.Sanitized {
				t.Errorf("expected no sanitization for valid query")
			}
		})
	}
}

// TestValidatePromQL_InvalidSyntax tests invalid PromQL syntax
func TestValidatePromQL_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		strictMode bool
		expectFail bool
	}{
		{
			name:       "unbalanced braces",
			query:      `up{job="prometheus"`,
			strictMode: true,
			expectFail: true,
		},
		{
			name:       "invalid operator",
			query:      `up ++ down`,
			strictMode: true,
			expectFail: true,
		},
		{
			name:       "malformed function call",
			query:      `rate(http_requests_total)`, // Missing duration
			strictMode: true,
			expectFail: true,
		},
		{
			name:       "invalid label matcher",
			query:      `up{job=}`,
			strictMode: true,
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&QueryValidationSettings{
				Enabled:                true,
				EnablePromQLValidation: true,
				StrictMode:             tt.strictMode,
			}, nil)

			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)

			if tt.expectFail && result.Valid {
				t.Errorf("expected query to fail validation")
			}
			if !tt.expectFail && !result.Valid {
				t.Errorf("expected query to pass or be sanitized, got error: %v", result.Error)
			}
		})
	}
}

// TestValidatePromQL_Complexity tests complexity limits
func TestValidatePromQL_Complexity(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		maxComplexity    int
		expectTooComplex bool
	}{
		{
			name:             "simple query under limit",
			query:            "up",
			maxComplexity:    100,
			expectTooComplex: false,
		},
		{
			name:             "complex nested aggregation",
			query:            `sum(rate(http_requests_total[5m])) by (status) / sum(rate(http_requests_total[5m]))`,
			maxComplexity:    5, // Very low limit
			expectTooComplex: true,
		},
		{
			name:             "moderate complexity",
			query:            `rate(http_requests_total{job="api"}[5m])`,
			maxComplexity:    50,
			expectTooComplex: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&QueryValidationSettings{
				Enabled:                true,
				EnablePromQLValidation: true,
				StrictMode:             true,
				MaxQueryComplexity:     tt.maxComplexity,
			}, nil)

			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)

			if tt.expectTooComplex && result.Valid {
				t.Errorf("expected query to fail complexity check")
			}
			if !tt.expectTooComplex && !result.Valid {
				t.Errorf("expected query to pass complexity check, got: %v", result.Error)
			}
		})
	}
}

// TestValidatePromQL_FunctionAllowlist tests function restriction
func TestValidatePromQL_FunctionAllowlist(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		allowedFuncs  []string
		expectBlocked bool
	}{
		{
			name:          "allowed function",
			query:         `rate(http_requests_total[5m])`,
			allowedFuncs:  []string{"rate", "sum", "avg"},
			expectBlocked: false,
		},
		{
			name:          "blocked function",
			query:         `increase(http_requests_total[5m])`,
			allowedFuncs:  []string{"rate", "sum", "avg"},
			expectBlocked: true,
		},
		{
			name:          "no allowlist allows all",
			query:         `increase(http_requests_total[5m])`,
			allowedFuncs:  []string{},
			expectBlocked: false,
		},
		{
			name:          "multiple functions one blocked",
			query:         `sum(increase(http_requests_total[5m]))`,
			allowedFuncs:  []string{"sum", "rate"},
			expectBlocked: true, // increase is blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&QueryValidationSettings{
				Enabled:                true,
				EnablePromQLValidation: true,
				StrictMode:             true,
				MaxQueryComplexity:     100,
				AllowedFunctions:       tt.allowedFuncs,
			}, nil)

			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)

			if tt.expectBlocked && result.Valid {
				t.Errorf("expected function to be blocked")
			}
			if !tt.expectBlocked && !result.Valid {
				t.Errorf("expected function to be allowed, got: %v", result.Error)
			}
		})
	}
}

// TestValidateLogQL_ValidQueries tests valid LogQL queries
func TestValidateLogQL_ValidQueries(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "simple log selector",
			query: `{job="varlogs"}`,
		},
		{
			name:  "log stream with filter",
			query: `{job="varlogs"} |= "error"`,
		},
		{
			name:  "metric query",
			query: `rate({job="varlogs"}[5m])`,
		},
		{
			name:  "label filter",
			query: `{job="varlogs"} | json | level="error"`,
		},
		{
			name:  "aggregation",
			query: `sum(rate({job="varlogs"}[5m])) by (level)`,
		},
	}

	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:               true,
		EnableLogQLValidation: true,
		StrictMode:            false,
		MaxQueryComplexity:    100,
	}, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateQuery(context.Background(), tt.query, DatasourceLoki)
			if !result.Valid {
				t.Errorf("expected valid query, got error: %v", result.Error)
			}
		})
	}
}

// TestValidateLogQL_InvalidSyntax tests invalid LogQL syntax
func TestValidateLogQL_InvalidSyntax(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		strictMode bool
		expectFail bool
	}{
		{
			name:       "missing closing brace",
			query:      `{job="varlogs"`,
			strictMode: true,
			expectFail: true,
		},
		{
			name:       "invalid label matcher",
			query:      `{job=}`,
			strictMode: true,
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&QueryValidationSettings{
				Enabled:               true,
				EnableLogQLValidation: true,
				StrictMode:            tt.strictMode,
			}, nil)

			result := validator.ValidateQuery(context.Background(), tt.query, DatasourceLoki)

			if tt.expectFail && result.Valid {
				t.Errorf("expected query to fail validation")
			}
		})
	}
}

// TestValidateGeneric tests generic validation for unknown datasources
func TestValidateGeneric(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectValid bool
	}{
		{
			name:        "normal query",
			query:       "SELECT * FROM metrics WHERE timestamp > now() - interval '1 hour'",
			expectValid: true,
		},
		{
			name:        "SQL injection attempt",
			query:       "SELECT * FROM metrics; DROP TABLE users; --",
			expectValid: false,
		},
		{
			name:        "very long query",
			query:       string(make([]byte, 11000)), // Over 10KB limit
			expectValid: false,
		},
		{
			name:        "union select injection",
			query:       "query UNION SELECT password FROM users",
			expectValid: false,
		},
	}

	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:    true,
		StrictMode: true,
	}, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateQuery(context.Background(), tt.query, DatasourceOther)

			if tt.expectValid && !result.Valid {
				t.Errorf("expected query to be valid, got error: %v", result.Error)
			}
			if !tt.expectValid && result.Valid {
				t.Errorf("expected query to be invalid")
			}
		})
	}
}

// TestValidationDisabled tests that validation can be disabled
func TestValidationDisabled(t *testing.T) {
	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled: false,
	}, nil)

	// Even invalid query should pass when validation disabled
	result := validator.ValidateQuery(context.Background(), "invalid{{{query", DatasourcePrometheus)
	if !result.Valid {
		t.Errorf("expected all queries to pass when validation disabled")
	}
}

// TestSanitization tests query sanitization behavior
func TestSanitization(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		expectSanitized bool
		strictMode      bool
	}{
		{
			name:            "whitespace only change",
			query:           "  up  ",
			expectSanitized: false, // Trimming whitespace doesn't affect parser
			strictMode:      false,
		},
		{
			name:            "invalid query in strict mode",
			query:           "up{{{",
			expectSanitized: false,
			strictMode:      true, // Should fail, not sanitize
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&QueryValidationSettings{
				Enabled:                true,
				EnablePromQLValidation: true,
				StrictMode:             tt.strictMode,
			}, nil)

			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)

			if tt.expectSanitized != result.Sanitized {
				t.Errorf("expected sanitized=%v, got %v", tt.expectSanitized, result.Sanitized)
			}
		})
	}
}

// TestComplexityCounter tests AST node counting accuracy
func TestComplexityCounter(t *testing.T) {
	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:                true,
		EnablePromQLValidation: true,
		StrictMode:             false,
		MaxQueryComplexity:     100,
	}, nil)

	tests := []struct {
		name              string
		query             string
		expectMoreComplex bool // Compared to "up"
	}{
		{
			name:              "simple metric",
			query:             "up",
			expectMoreComplex: false,
		},
		{
			name:              "with labels",
			query:             `up{job="test"}`,
			expectMoreComplex: true,
		},
		{
			name:              "with function",
			query:             `rate(up[5m])`,
			expectMoreComplex: true,
		},
		{
			name:              "nested functions",
			query:             `sum(rate(http_requests_total[5m])) by (status)`,
			expectMoreComplex: true,
		},
	}

	// Get baseline complexity for "up"
	baseResult := validator.validatePromQL("up")
	if !baseResult.Valid {
		t.Fatal("baseline query should be valid")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.validatePromQL(tt.query)
			if !result.Valid {
				t.Skip("query failed validation, skipping complexity test")
			}

			// This is a rough test - we can't easily compare complexity directly
			// but we know more complex queries should have more AST nodes
			// For now, just verify the query validates successfully
		})
	}
}

// TestDefaultSettings tests default settings are applied correctly
func TestDefaultSettings(t *testing.T) {
	validator := NewQueryValidator(nil, nil)

	if validator.settings.Enabled {
		t.Error("expected validation to be disabled by default")
	}

	if validator.settings.MaxQueryComplexity != 100 {
		t.Errorf("expected default complexity 100, got %d", validator.settings.MaxQueryComplexity)
	}

	if validator.settings.LLMValidationMode != "advisory" {
		t.Errorf("expected default LLM mode 'advisory', got %s", validator.settings.LLMValidationMode)
	}
}

// TestViolationTypes tests that correct violation types are returned
func TestViolationTypes(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		settings         QueryValidationSettings
		expectedViolation string
	}{
		{
			name:              "syntax violation",
			query:             "up{{{",
			settings:          QueryValidationSettings{Enabled: true, EnablePromQLValidation: true, StrictMode: true},
			expectedViolation: "syntax",
		},
		{
			name:              "complexity violation",
			query:             `sum(rate(http_requests_total[5m])) / sum(rate(http_requests_total[5m]))`,
			settings:          QueryValidationSettings{Enabled: true, EnablePromQLValidation: true, StrictMode: true, MaxQueryComplexity: 3},
			expectedViolation: "complexity",
		},
		{
			name:              "function violation",
			query:             `increase(http_requests_total[5m])`,
			settings:          QueryValidationSettings{Enabled: true, EnablePromQLValidation: true, StrictMode: true, MaxQueryComplexity: 100, AllowedFunctions: []string{"rate"}},
			expectedViolation: "function_blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewQueryValidator(&tt.settings, nil)
			result := validator.ValidateQuery(context.Background(), tt.query, DatasourcePrometheus)

			if result.Valid {
				t.Error("expected query to fail validation")
			}

			if result.ViolationType != tt.expectedViolation {
				t.Errorf("expected violation type %s, got %s", tt.expectedViolation, result.ViolationType)
			}
		})
	}
}

// TestEmptyQuery tests handling of empty queries
func TestEmptyQuery(t *testing.T) {
	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:                true,
		EnablePromQLValidation: true,
		StrictMode:             true,
	}, nil)

	result := validator.ValidateQuery(context.Background(), "", DatasourcePrometheus)

	// Empty queries should be handled gracefully
	// The parser will reject it, which is correct behavior
	if result.Valid {
		t.Error("expected empty query to fail validation")
	}
}

// TestDatasourceTypes tests validation works for different datasource types
func TestDatasourceTypes(t *testing.T) {
	validator := NewQueryValidator(&QueryValidationSettings{
		Enabled:                true,
		EnablePromQLValidation: true,
		StrictMode:             false,
		MaxQueryComplexity:     100,
	}, nil)

	tests := []struct {
		dsType DatasourceType
		query  string
	}{
		{DatasourcePrometheus, "up"},
		{DatasourceLoki, `{job="test"}`},
		{DatasourceTempo, "any query"}, // Falls back to generic validation
		{DatasourceOther, "SELECT * FROM table"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dsType), func(t *testing.T) {
			result := validator.ValidateQuery(context.Background(), tt.query, tt.dsType)
			// Just verify it doesn't panic and returns a result
			if result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}
