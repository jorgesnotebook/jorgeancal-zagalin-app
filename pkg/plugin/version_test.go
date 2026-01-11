package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		expectedVersion GrafanaVersion
		expectError     bool
	}{
		{
			name:  "standard version",
			input: "10.4.0",
			expectedVersion: GrafanaVersion{
				Full:        "10.4.0",
				Major:       10,
				Minor:       4,
				Patch:       0,
				IsAvailable: true,
			},
			expectError: false,
		},
		{
			name:  "version with prerelease",
			input: "11.0.0-beta1",
			expectedVersion: GrafanaVersion{
				Full:        "11.0.0-beta1",
				Major:       11,
				Minor:       0,
				Patch:       0,
				IsAvailable: true,
			},
			expectError: false,
		},
		{
			name:  "version with prerelease and metadata",
			input: "12.1.3-alpha.2+build.123",
			expectedVersion: GrafanaVersion{
				Full:        "12.1.3-alpha.2+build.123",
				Major:       12,
				Minor:       1,
				Patch:       3,
				IsAvailable: true,
			},
			expectError: false,
		},
		{
			name:  "minimum supported version",
			input: "10.4.0",
			expectedVersion: GrafanaVersion{
				Full:        "10.4.0",
				Major:       10,
				Minor:       4,
				Patch:       0,
				IsAvailable: true,
			},
			expectError: false,
		},
		{
			name:  "newer version",
			input: "13.5.7",
			expectedVersion: GrafanaVersion{
				Full:        "13.5.7",
				Major:       13,
				Minor:       5,
				Patch:       7,
				IsAvailable: true,
			},
			expectError: false,
		},
		{
			name:  "empty string",
			input: "",
			expectedVersion: GrafanaVersion{
				Full:        "unknown",
				IsAvailable: false,
			},
			expectError: false,
		},
		{
			name:  "unknown string",
			input: "unknown",
			expectedVersion: GrafanaVersion{
				Full:        "unknown",
				IsAvailable: false,
			},
			expectError: false,
		},
		{
			name:        "invalid format - no dots",
			input:       "10",
			expectError: true,
		},
		{
			name:        "invalid format - missing patch",
			input:       "10.4",
			expectError: true,
		},
		{
			name:        "invalid format - not a number",
			input:       "abc.def.ghi",
			expectError: true,
		},
		{
			name:        "invalid format - partial number",
			input:       "10.4.abc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseVersion(tt.input)

			if tt.expectError {
				require.Error(t, err, "Expected error for input: %s", tt.input)
			} else {
				require.NoError(t, err, "Unexpected error for input: %s", tt.input)
				assert.Equal(t, tt.expectedVersion.Full, result.Full)
				assert.Equal(t, tt.expectedVersion.Major, result.Major)
				assert.Equal(t, tt.expectedVersion.Minor, result.Minor)
				assert.Equal(t, tt.expectedVersion.Patch, result.Patch)
				assert.Equal(t, tt.expectedVersion.IsAvailable, result.IsAvailable)
			}
		})
	}
}

func TestVersionIsSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   GrafanaVersion
		supported bool
	}{
		{
			name: "exact minimum version",
			version: GrafanaVersion{
				Major:       10,
				Minor:       4,
				Patch:       0,
				IsAvailable: true,
			},
			supported: true,
		},
		{
			name: "above minimum major",
			version: GrafanaVersion{
				Major:       11,
				Minor:       0,
				Patch:       0,
				IsAvailable: true,
			},
			supported: true,
		},
		{
			name: "above minimum minor",
			version: GrafanaVersion{
				Major:       10,
				Minor:       5,
				Patch:       0,
				IsAvailable: true,
			},
			supported: true,
		},
		{
			name: "above minimum patch",
			version: GrafanaVersion{
				Major:       10,
				Minor:       4,
				Patch:       1,
				IsAvailable: true,
			},
			supported: true,
		},
		{
			name: "below minimum major",
			version: GrafanaVersion{
				Major:       9,
				Minor:       5,
				Patch:       0,
				IsAvailable: true,
			},
			supported: false,
		},
		{
			name: "below minimum minor",
			version: GrafanaVersion{
				Major:       10,
				Minor:       3,
				Patch:       9,
				IsAvailable: true,
			},
			supported: false,
		},
		{
			name: "below minimum patch",
			version: GrafanaVersion{
				Major:       10,
				Minor:       4,
				Patch:       0,
				IsAvailable: true,
			},
			supported: true, // 10.4.0 is the minimum
		},
		{
			name: "much newer version",
			version: GrafanaVersion{
				Major:       15,
				Minor:       2,
				Patch:       3,
				IsAvailable: true,
			},
			supported: true,
		},
		{
			name: "unavailable version assumes supported",
			version: GrafanaVersion{
				Full:        "unknown",
				IsAvailable: false,
			},
			supported: true, // Graceful fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.IsSupported()
			assert.Equal(t, tt.supported, result, "Version %v should have supported=%v", tt.version, tt.supported)
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		name     string
		v1       GrafanaVersion
		v2       GrafanaVersion
		expected int // -1, 0, or 1
	}{
		{
			name:     "equal versions",
			v1:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: 0,
		},
		{
			name:     "v1 greater major",
			v1:       GrafanaVersion{Major: 11, Minor: 0, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: 1,
		},
		{
			name:     "v1 lesser major",
			v1:       GrafanaVersion{Major: 9, Minor: 5, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: -1,
		},
		{
			name:     "equal major, v1 greater minor",
			v1:       GrafanaVersion{Major: 10, Minor: 5, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: 1,
		},
		{
			name:     "equal major, v1 lesser minor",
			v1:       GrafanaVersion{Major: 10, Minor: 3, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: -1,
		},
		{
			name:     "equal major and minor, v1 greater patch",
			v1:       GrafanaVersion{Major: 10, Minor: 4, Patch: 1},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			expected: 1,
		},
		{
			name:     "equal major and minor, v1 lesser patch",
			v1:       GrafanaVersion{Major: 10, Minor: 4, Patch: 0},
			v2:       GrafanaVersion{Major: 10, Minor: 4, Patch: 1},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.v1.Compare(tt.v2)
			assert.Equal(t, tt.expected, result, "Comparison of %v and %v", tt.v1, tt.v2)
		})
	}
}

func TestVersionString(t *testing.T) {
	tests := []struct {
		name     string
		version  GrafanaVersion
		expected string
	}{
		{
			name: "available version",
			version: GrafanaVersion{
				Full:        "10.4.0",
				IsAvailable: true,
			},
			expected: "10.4.0",
		},
		{
			name: "available version with prerelease",
			version: GrafanaVersion{
				Full:        "11.0.0-beta1",
				IsAvailable: true,
			},
			expected: "11.0.0-beta1",
		},
		{
			name: "unavailable version",
			version: GrafanaVersion{
				Full:        "unknown",
				IsAvailable: false,
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVersionWarnings(t *testing.T) {
	tests := []struct {
		name            string
		version         GrafanaVersion
		expectWarnings  bool
		warningContains string
	}{
		{
			name: "supported version - no warnings",
			version: GrafanaVersion{
				Full:        "10.4.0",
				Major:       10,
				Minor:       4,
				Patch:       0,
				IsAvailable: true,
			},
			expectWarnings: false,
		},
		{
			name: "newer version - no warnings",
			version: GrafanaVersion{
				Full:        "11.0.0",
				Major:       11,
				Minor:       0,
				Patch:       0,
				IsAvailable: true,
			},
			expectWarnings: false,
		},
		{
			name: "unavailable version - has warning",
			version: GrafanaVersion{
				Full:        "unknown",
				IsAvailable: false,
			},
			expectWarnings:  true,
			warningContains: "could not be detected",
		},
		{
			name: "unsupported version - has warning",
			version: GrafanaVersion{
				Full:        "10.3.0",
				Major:       10,
				Minor:       3,
				Patch:       0,
				IsAvailable: true,
			},
			expectWarnings:  true,
			warningContains: "below minimum supported version",
		},
		{
			name: "very old version - has warning",
			version: GrafanaVersion{
				Full:        "9.0.0",
				Major:       9,
				Minor:       0,
				Patch:       0,
				IsAvailable: true,
			},
			expectWarnings:  true,
			warningContains: "below minimum supported version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.version.VersionWarnings()

			if tt.expectWarnings {
				assert.NotEmpty(t, warnings, "Expected warnings but got none")
				if tt.warningContains != "" {
					found := false
					for _, warning := range warnings {
						if assert.Contains(t, warning, tt.warningContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected warning containing '%s' but not found", tt.warningContains)
				}
			} else {
				assert.Empty(t, warnings, "Expected no warnings but got: %v", warnings)
			}
		})
	}
}

func TestMinimumSupportedVersion(t *testing.T) {
	// Verify MinimumSupportedVersion constant is correctly defined
	assert.Equal(t, "10.4.0", MinimumSupportedVersion.Full)
	assert.Equal(t, 10, MinimumSupportedVersion.Major)
	assert.Equal(t, 4, MinimumSupportedVersion.Minor)
	assert.Equal(t, 0, MinimumSupportedVersion.Patch)
	assert.True(t, MinimumSupportedVersion.IsAvailable)
	assert.True(t, MinimumSupportedVersion.IsSupported())
}
