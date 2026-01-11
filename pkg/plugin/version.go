package plugin

import (
	"fmt"
	"regexp"
	"strconv"
)

// GrafanaVersion represents a parsed Grafana semantic version
type GrafanaVersion struct {
	Full        string // Full version string (e.g., "10.4.0", "11.0.0-beta1")
	Major       int    // Major version number
	Minor       int    // Minor version number
	Patch       int    // Patch version number
	IsAvailable bool   // Whether version was successfully detected
}

// MinimumSupportedVersion defines the minimum Grafana version supported by this plugin
// This matches the grafanaDependency in plugin.json
var MinimumSupportedVersion = GrafanaVersion{
	Full:        "10.4.0",
	Major:       10,
	Minor:       4,
	Patch:       0,
	IsAvailable: true,
}

// ParseVersion parses a semantic version string into a GrafanaVersion struct
// Supports standard semver format: "major.minor.patch" with optional prerelease suffix
// Examples: "10.4.0", "11.0.0-beta1", "12.1.3"
// Returns version with IsAvailable=false for empty or "unknown" strings
func ParseVersion(versionStr string) (GrafanaVersion, error) {
	if versionStr == "" || versionStr == "unknown" {
		return GrafanaVersion{
			Full:        "unknown",
			IsAvailable: false,
		}, nil
	}

	// Match major.minor.patch, ignore prerelease suffix
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)

	if len(matches) < 4 {
		return GrafanaVersion{}, fmt.Errorf("invalid version format: %s (expected major.minor.patch)", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return GrafanaVersion{
		Full:        versionStr,
		Major:       major,
		Minor:       minor,
		Patch:       patch,
		IsAvailable: true,
	}, nil
}

// IsSupported checks if this version meets the minimum supported version requirement
// Returns true if version >= MinimumSupportedVersion
// Also returns true for unavailable versions (graceful fallback)
func (v GrafanaVersion) IsSupported() bool {
	if !v.IsAvailable {
		return true // Assume supported if version unknown
	}

	min := MinimumSupportedVersion

	if v.Major > min.Major {
		return true
	}
	if v.Major < min.Major {
		return false
	}

	if v.Minor > min.Minor {
		return true
	}
	if v.Minor < min.Minor {
		return false
	}

	return v.Patch >= min.Patch
}

// Compare compares this version with another version
// Returns: -1 if v < other, 0 if v == other, 1 if v > other
func (v GrafanaVersion) Compare(other GrafanaVersion) int {
	if v.Major != other.Major {
		return compare(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compare(v.Minor, other.Minor)
	}
	return compare(v.Patch, other.Patch)
}

func compare(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// String returns a human-readable version string
func (v GrafanaVersion) String() string {
	if !v.IsAvailable {
		return "unknown"
	}
	return v.Full
}

// VersionWarnings returns a list of warnings if the version is unsupported or unavailable
// Returns empty slice if version is supported and available
func (v GrafanaVersion) VersionWarnings() []string {
	warnings := []string{}

	if !v.IsAvailable {
		warnings = append(warnings, "Grafana version could not be detected (version reporting may be disabled)")
	} else if !v.IsSupported() {
		warnings = append(warnings, fmt.Sprintf(
			"Grafana %s is below minimum supported version %s. Please upgrade for full compatibility.",
			v.Full,
			MinimumSupportedVersion.Full,
		))
	}

	return warnings
}
