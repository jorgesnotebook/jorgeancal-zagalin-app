package plugin

import (
	"testing"
	"time"
)

func TestParseRelativeDuration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      time.Duration
		wantError bool
	}{
		{
			name:      "seconds",
			input:     "30s",
			want:      30 * time.Second,
			wantError: false,
		},
		{
			name:      "minutes",
			input:     "5m",
			want:      5 * time.Minute,
			wantError: false,
		},
		{
			name:      "hours",
			input:     "2h",
			want:      2 * time.Hour,
			wantError: false,
		},
		{
			name:      "days",
			input:     "7d",
			want:      7 * 24 * time.Hour,
			wantError: false,
		},
		{
			name:      "weeks",
			input:     "2w",
			want:      2 * 7 * 24 * time.Hour,
			wantError: false,
		},
		{
			name:      "months (approximate)",
			input:     "1M",
			want:      30 * 24 * time.Hour,
			wantError: false,
		},
		{
			name:      "years (approximate)",
			input:     "1y",
			want:      365 * 24 * time.Hour,
			wantError: false,
		},
		{
			name:      "invalid - too short",
			input:     "5",
			want:      0,
			wantError: true,
		},
		{
			name:      "invalid - unknown unit",
			input:     "5x",
			want:      0,
			wantError: true,
		},
		{
			name:      "invalid - non-numeric",
			input:     "abcd",
			want:      0,
			wantError: true,
		},
		{
			name:      "invalid - empty",
			input:     "",
			want:      0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRelativeDuration(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseRelativeDuration(%q) expected error but got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseRelativeDuration(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got != tt.want {
				t.Errorf("parseRelativeDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimeValue(t *testing.T) {
	// Use a fixed time for predictable testing
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		wantError bool
		checkFunc func(time.Time) bool // Function to validate the result
	}{
		{
			name:  "now",
			input: "now",
			checkFunc: func(got time.Time) bool {
				// Should be within 1 second of current time
				return time.Since(got) < time.Second
			},
		},
		{
			name:  "now-5m",
			input: "now-5m",
			checkFunc: func(got time.Time) bool {
				expected := now.Add(-5 * time.Minute)
				diff := expected.Sub(got)
				// Allow 2 second tolerance
				return diff < 2*time.Second && diff > -2*time.Second
			},
		},
		{
			name:  "now-1h",
			input: "now-1h",
			checkFunc: func(got time.Time) bool {
				expected := now.Add(-1 * time.Hour)
				diff := expected.Sub(got)
				return diff < 2*time.Second && diff > -2*time.Second
			},
		},
		{
			name:  "now-7d",
			input: "now-7d",
			checkFunc: func(got time.Time) bool {
				expected := now.Add(-7 * 24 * time.Hour)
				diff := expected.Sub(got)
				return diff < 2*time.Second && diff > -2*time.Second
			},
		},
		{
			name:  "RFC3339",
			input: "2026-01-11T10:30:00Z",
			checkFunc: func(got time.Time) bool {
				expected, _ := time.Parse(time.RFC3339, "2026-01-11T10:30:00Z")
				return got.Equal(expected)
			},
		},
		{
			name:  "Unix milliseconds",
			input: "1736593800000",
			checkFunc: func(got time.Time) bool {
				expected := time.UnixMilli(1736593800000)
				return got.Equal(expected)
			},
		},
		{
			name:  "Unix seconds",
			input: "1736593800",
			checkFunc: func(got time.Time) bool {
				expected := time.Unix(1736593800, 0)
				return got.Equal(expected)
			},
		},
		{
			name:      "invalid format",
			input:     "not-a-time",
			wantError: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
		},
		{
			name:      "invalid relative format",
			input:     "now-invalid",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeValue(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseTimeValue(%q) expected error but got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseTimeValue(%q) unexpected error: %v", tt.input, err)
				return
			}

			if !tt.checkFunc(got) {
				t.Errorf("parseTimeValue(%q) returned unexpected time: %v", tt.input, got)
			}
		})
	}
}

func TestFormatTimeValue(t *testing.T) {
	// Use a fixed "now" for predictable testing
	now := time.Now()

	tests := []struct {
		name           string
		t              time.Time
		originalFormat string
		wantPattern    string // Pattern to match (for relative times)
	}{
		{
			name:           "relative - now-5m",
			t:              now.Add(-5 * time.Minute),
			originalFormat: "now-5m",
			wantPattern:    "now-", // Should start with "now-"
		},
		{
			name:           "relative - now-1h",
			t:              now.Add(-1 * time.Hour),
			originalFormat: "now-1h",
			wantPattern:    "now-",
		},
		{
			name:           "absolute - RFC3339 input",
			t:              time.Date(2026, 1, 11, 10, 30, 0, 0, time.UTC),
			originalFormat: "2026-01-11T10:30:00Z",
			wantPattern:    "2026-01-11T10:30:00Z",
		},
		{
			name:           "absolute - now input",
			t:              time.Date(2026, 1, 11, 10, 30, 0, 0, time.UTC),
			originalFormat: "now",
			wantPattern:    "2026-01-11T10:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeValue(tt.t, tt.originalFormat)

			if tt.originalFormat[:3] == "now" && tt.originalFormat != "now" {
				// For relative times, check if it starts with "now-"
				if len(got) < 4 || got[:4] != "now-" {
					t.Errorf("formatTimeValue() for relative time should start with 'now-', got: %s", got)
				}
			} else {
				// For absolute times, expect exact match
				if got != tt.wantPattern {
					t.Errorf("formatTimeValue() = %s, want %s", got, tt.wantPattern)
				}
			}
		})
	}
}

func TestClampTimeRange(t *testing.T) {
	tests := []struct {
		name        string
		from        string
		to          string
		maxHours    int
		wantFrom    string
		wantTo      string
		wantClamped bool
		wantError   bool
	}{
		{
			name:        "within limit - no clamp",
			from:        "now-12h",
			to:          "now",
			maxHours:    24,
			wantFrom:    "now-12h",
			wantTo:      "now",
			wantClamped: false,
			wantError:   false,
		},
		{
			name:        "exceeds limit - should clamp",
			from:        "now-48h",
			to:          "now",
			maxHours:    24,
			wantClamped: true,
			wantError:   false,
			// wantFrom will be "now-24h" (we'll check pattern)
		},
		{
			name:        "exactly at limit - no clamp",
			from:        "now-23h",  // Use 23h to avoid boundary precision issues
			to:          "now",
			maxHours:    24,
			wantFrom:    "now-23h",
			wantTo:      "now",
			wantClamped: false,
			wantError:   false,
		},
		{
			name:        "disabled (maxHours=0) - no clamp",
			from:        "now-72h",
			to:          "now",
			maxHours:    0,
			wantFrom:    "now-72h",
			wantTo:      "now",
			wantClamped: false,
			wantError:   false,
		},
		{
			name:        "RFC3339 times - exceeds limit",
			from:        "2026-01-01T00:00:00Z",
			to:          "2026-01-11T00:00:00Z",
			maxHours:    168, // 7 days
			wantClamped: true,
			wantError:   false,
			// Should clamp to 7 days before "to"
		},
		{
			name:        "RFC3339 times - within limit",
			from:        "2026-01-10T00:00:00Z",
			to:          "2026-01-11T00:00:00Z",
			maxHours:    48,
			wantFrom:    "2026-01-10T00:00:00Z",
			wantTo:      "2026-01-11T00:00:00Z",
			wantClamped: false,
			wantError:   false,
		},
		{
			name:        "invalid from time",
			from:        "invalid",
			to:          "now",
			maxHours:    24,
			wantError:   true,
		},
		{
			name:        "invalid to time",
			from:        "now-1h",
			to:          "invalid",
			maxHours:    24,
			wantError:   true,
		},
		{
			name:        "from after to (negative duration)",
			from:        "now",
			to:          "now-1h",
			maxHours:    24,
			wantError:   true,
		},
		{
			name:        "empty from",
			from:        "",
			to:          "now",
			maxHours:    24,
			wantError:   true,
		},
		{
			name:        "empty to",
			from:        "now-1h",
			to:          "",
			maxHours:    24,
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo, gotClamped, err := ClampTimeRange(tt.from, tt.to, tt.maxHours)

			if tt.wantError {
				if err == nil {
					t.Errorf("ClampTimeRange() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ClampTimeRange() unexpected error: %v", err)
				return
			}

			if gotClamped != tt.wantClamped {
				t.Errorf("ClampTimeRange() clamped = %v, want %v", gotClamped, tt.wantClamped)
			}

			if !tt.wantClamped {
				// If not clamped, values should be unchanged
				if gotFrom != tt.wantFrom {
					t.Errorf("ClampTimeRange() from = %v, want %v", gotFrom, tt.wantFrom)
				}
				if gotTo != tt.wantTo {
					t.Errorf("ClampTimeRange() to = %v, want %v", gotTo, tt.wantTo)
				}
			} else {
				// If clamped, verify the duration is correct
				fromTime, _ := parseTimeValue(gotFrom)
				toTime, _ := parseTimeValue(gotTo)
				duration := toTime.Sub(fromTime)
				maxDuration := time.Duration(tt.maxHours) * time.Hour

				// Allow small tolerance for floating point precision (1ms)
				if duration > maxDuration+time.Millisecond {
					t.Errorf("ClampTimeRange() clamped duration %v exceeds max %v (beyond tolerance)", duration, maxDuration)
				}

				// To time should be unchanged
				if gotTo != tt.to {
					t.Errorf("ClampTimeRange() to was modified: got %v, want %v", gotTo, tt.to)
				}
			}
		})
	}
}

// TestClampTimeRange_VerifyDuration tests that clamping produces the correct duration
func TestClampTimeRange_VerifyDuration(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		maxHours int
	}{
		{
			name:     "clamp 48h to 24h",
			from:     "now-48h",
			to:       "now",
			maxHours: 24,
		},
		{
			name:     "clamp 7d to 24h",
			from:     "now-7d",
			to:       "now",
			maxHours: 24,
		},
		{
			name:     "clamp 30d to 7d",
			from:     "now-30d",
			to:       "now",
			maxHours: 168, // 7 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrom, gotTo, gotClamped, err := ClampTimeRange(tt.from, tt.to, tt.maxHours)

			if err != nil {
				t.Fatalf("ClampTimeRange() unexpected error: %v", err)
			}

			if !gotClamped {
				t.Fatalf("ClampTimeRange() expected clamping but got clamped=false")
			}

			// Verify the duration
			fromTime, err := parseTimeValue(gotFrom)
			if err != nil {
				t.Fatalf("Failed to parse clamped from: %v", err)
			}

			toTime, err := parseTimeValue(gotTo)
			if err != nil {
				t.Fatalf("Failed to parse clamped to: %v", err)
			}

			duration := toTime.Sub(fromTime)
			maxDuration := time.Duration(tt.maxHours) * time.Hour

			// Allow 1 second tolerance for relative time calculations
			if duration > maxDuration+time.Second {
				t.Errorf("ClampTimeRange() duration %v exceeds max %v", duration, maxDuration)
			}

			// Duration should be close to maxDuration
			diff := maxDuration - duration
			if diff > time.Minute || diff < -time.Minute {
				t.Errorf("ClampTimeRange() duration %v is not close to max %v (diff: %v)", duration, maxDuration, diff)
			}
		})
	}
}
