package plugin

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// VersionDetector handles Grafana version detection from HTTP headers
type VersionDetector struct {
	detectedVersion GrafanaVersion
	logger          log.Logger
}

// NewVersionDetector creates a new version detector
func NewVersionDetector(logger log.Logger) *VersionDetector {
	return &VersionDetector{
		detectedVersion: GrafanaVersion{IsAvailable: false},
		logger:          logger,
	}
}

// DetectFromHeader extracts Grafana version from X-Grafana-Version HTTP header
// Returns version with IsAvailable=false if header is missing or invalid
func (vd *VersionDetector) DetectFromHeader(r *http.Request) GrafanaVersion {
	versionHeader := r.Header.Get("X-Grafana-Version")
	if versionHeader == "" {
		return GrafanaVersion{IsAvailable: false}
	}

	version, err := ParseVersion(versionHeader)
	if err != nil {
		vd.logger.Warn("Failed to parse Grafana version from header",
			"header", versionHeader,
			"error", err)
		return GrafanaVersion{IsAvailable: false}
	}

	// Cache the detected version
	if version.IsAvailable {
		vd.detectedVersion = version
	}

	return version
}

// GetVersion returns the cached detected version
func (vd *VersionDetector) GetVersion() GrafanaVersion {
	return vd.detectedVersion
}

// LogVersionWarnings logs warnings if version is unsupported or unavailable
func (vd *VersionDetector) LogVersionWarnings(ctx context.Context) {
	warnings := vd.detectedVersion.VersionWarnings()
	for _, warning := range warnings {
		vd.logger.Warn(warning, "version", vd.detectedVersion.String())
	}
}

// LogVersionOnStartup logs the detected Grafana version with startup message
func (vd *VersionDetector) LogVersionOnStartup(ctx context.Context) {
	if vd.detectedVersion.IsAvailable {
		vd.logger.Info("Grafana version detected",
			"version", vd.detectedVersion.String(),
			"supported", vd.detectedVersion.IsSupported(),
			"minimumVersion", MinimumSupportedVersion.String())

		// Log warnings if any
		vd.LogVersionWarnings(ctx)
	} else {
		vd.logger.Info("Grafana version not yet detected (will detect on first request)",
			"minimumVersion", MinimumSupportedVersion.String())
	}
}
