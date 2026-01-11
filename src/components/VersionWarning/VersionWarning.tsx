import React from 'react';
import { Alert } from '@grafana/ui';
import { checkVersionCompatibility } from '../../services/versionDetector';

/**
 * VersionWarning Component
 *
 * Displays warnings if the detected Grafana version is unsupported
 * or if version detection failed.
 *
 * Features:
 * - Automatically detects Grafana version from config.buildInfo
 * - Shows warning for unsupported versions (< 10.4.0)
 * - Shows informational message if version unavailable
 * - Gracefully handles disabled version reporting
 * - Only renders if there are warnings to display
 */
export const VersionWarning: React.FC = () => {
  const { version, warnings, minimumVersion } = checkVersionCompatibility();

  // Don't render if no warnings
  if (warnings.length === 0) {
    return null;
  }

  // Determine severity based on version status
  const severity = version.isAvailable && !version.isSupported ? 'error' : 'warning';

  // Build title based on version availability
  const title = version.isAvailable
    ? `Grafana ${version.full} - Compatibility Warning`
    : 'Grafana Version Not Detected';

  return (
    <Alert severity={severity} title={title}>
      <div style={{ marginBottom: '8px' }}>
        {warnings.map((warning, idx) => (
          <p key={idx} style={{ margin: '4px 0' }}>
            {warning}
          </p>
        ))}
      </div>

      {version.isAvailable ? (
        <p style={{ margin: '8px 0 0 0', fontSize: '0.9em', opacity: 0.9 }}>
          <strong>Detected:</strong> Grafana {version.full}
          <br />
          <strong>Minimum Required:</strong> Grafana {minimumVersion}
        </p>
      ) : (
        <p style={{ margin: '8px 0 0 0', fontSize: '0.9em', opacity: 0.9 }}>
          <strong>Minimum Required:</strong> Grafana {minimumVersion}
          <br />
          <em>
            Note: Version detection may be disabled in your Grafana settings. The plugin will continue to function, but
            some features may not work correctly on older versions.
          </em>
        </p>
      )}
    </Alert>
  );
};
