package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVersionDetector(t *testing.T) {
	logger := log.NewNullLogger()
	detector := NewVersionDetector(logger)

	assert.NotNil(t, detector)
	assert.False(t, detector.GetVersion().IsAvailable)
}

func TestDetectFromHeader(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		expectVersion bool
		expectedMajor int
		expectedMinor int
		expectedPatch int
	}{
		{
			name:          "valid version header",
			headerValue:   "10.4.0",
			expectVersion: true,
			expectedMajor: 10,
			expectedMinor: 4,
			expectedPatch: 0,
		},
		{
			name:          "version with prerelease",
			headerValue:   "11.0.0-beta1",
			expectVersion: true,
			expectedMajor: 11,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:          "newer version",
			headerValue:   "12.1.3",
			expectVersion: true,
			expectedMajor: 12,
			expectedMinor: 1,
			expectedPatch: 3,
		},
		{
			name:          "missing header",
			headerValue:   "",
			expectVersion: false,
		},
		{
			name:          "invalid format",
			headerValue:   "invalid-version",
			expectVersion: false,
		},
		{
			name:          "partial version",
			headerValue:   "10.4",
			expectVersion: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewNullLogger()
			detector := NewVersionDetector(logger)

			req := httptest.NewRequest("GET", "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Grafana-Version", tt.headerValue)
			}

			version := detector.DetectFromHeader(req)

			if tt.expectVersion {
				assert.True(t, version.IsAvailable, "Expected version to be available")
				assert.Equal(t, tt.expectedMajor, version.Major)
				assert.Equal(t, tt.expectedMinor, version.Minor)
				assert.Equal(t, tt.expectedPatch, version.Patch)
			} else {
				assert.False(t, version.IsAvailable, "Expected version to be unavailable")
			}
		})
	}
}

func TestVersionCaching(t *testing.T) {
	logger := log.NewNullLogger()
	detector := NewVersionDetector(logger)

	// Initially no version cached
	initialVersion := detector.GetVersion()
	assert.False(t, initialVersion.IsAvailable)

	// Detect version from header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Grafana-Version", "10.4.0")

	detectedVersion := detector.DetectFromHeader(req)
	assert.True(t, detectedVersion.IsAvailable)

	// Version should now be cached
	cachedVersion := detector.GetVersion()
	assert.True(t, cachedVersion.IsAvailable)
	assert.Equal(t, "10.4.0", cachedVersion.Full)
	assert.Equal(t, 10, cachedVersion.Major)
	assert.Equal(t, 4, cachedVersion.Minor)
	assert.Equal(t, 0, cachedVersion.Patch)
}

func TestVersionCachingDoesNotUpdateOnInvalid(t *testing.T) {
	logger := log.NewNullLogger()
	detector := NewVersionDetector(logger)

	// Set valid version first
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.Header.Set("X-Grafana-Version", "10.4.0")
	detector.DetectFromHeader(req1)

	cachedAfterValid := detector.GetVersion()
	assert.Equal(t, "10.4.0", cachedAfterValid.Full)

	// Try to detect invalid version
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Grafana-Version", "invalid")
	invalidVersion := detector.DetectFromHeader(req2)

	assert.False(t, invalidVersion.IsAvailable)

	// Cache should still have the valid version
	cachedAfterInvalid := detector.GetVersion()
	assert.Equal(t, "10.4.0", cachedAfterInvalid.Full)
	assert.True(t, cachedAfterInvalid.IsAvailable)
}

func TestLogVersionWarnings(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		expectWarning bool
	}{
		{
			name:          "supported version - no warning",
			headerValue:   "10.4.0",
			expectWarning: false,
		},
		{
			name:          "newer version - no warning",
			headerValue:   "11.0.0",
			expectWarning: false,
		},
		{
			name:          "unsupported version - warning",
			headerValue:   "10.3.0",
			expectWarning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.NewNullLogger()
			detector := NewVersionDetector(logger)

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("X-Grafana-Version", tt.headerValue)

			detector.DetectFromHeader(req)

			// Call LogVersionWarnings (in real use, this would log)
			ctx := context.Background()
			detector.LogVersionWarnings(ctx)

			// Verify warnings
			version := detector.GetVersion()
			warnings := version.VersionWarnings()

			if tt.expectWarning {
				assert.NotEmpty(t, warnings, "Expected warnings for version %s", tt.headerValue)
			} else {
				assert.Empty(t, warnings, "Expected no warnings for version %s", tt.headerValue)
			}
		})
	}
}

func TestLogVersionOnStartup(t *testing.T) {
	t.Run("with detected version", func(t *testing.T) {
		logger := log.NewNullLogger()
		detector := NewVersionDetector(logger)

		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Grafana-Version", "10.4.0")
		detector.DetectFromHeader(req)

		ctx := context.Background()
		detector.LogVersionOnStartup(ctx)

		version := detector.GetVersion()
		assert.True(t, version.IsAvailable)
	})

	t.Run("without detected version", func(t *testing.T) {
		logger := log.NewNullLogger()
		detector := NewVersionDetector(logger)

		ctx := context.Background()
		detector.LogVersionOnStartup(ctx)

		version := detector.GetVersion()
		assert.False(t, version.IsAvailable)
	})
}

func TestDetectFromHeaderWithRealRequest(t *testing.T) {
	logger := log.NewNullLogger()
	detector := NewVersionDetector(logger)

	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	req.Header.Set("X-Grafana-Version", "11.0.0")

	version := detector.DetectFromHeader(req)

	assert.True(t, version.IsAvailable)
	assert.Equal(t, "11.0.0", version.Full)
	assert.Equal(t, 11, version.Major)
	assert.True(t, version.IsSupported())
}

func TestGetVersionBeforeDetection(t *testing.T) {
	logger := log.NewNullLogger()
	detector := NewVersionDetector(logger)

	// Get version before any detection
	version := detector.GetVersion()

	assert.False(t, version.IsAvailable)
	assert.Equal(t, 0, version.Major)
	assert.Equal(t, 0, version.Minor)
	assert.Equal(t, 0, version.Patch)
}
