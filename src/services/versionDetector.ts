import { config } from '@grafana/runtime';

/**
 * Grafana version information with support status
 */
export interface GrafanaVersion {
  full: string; // Full version string (e.g., "10.4.0", "11.0.0-beta1")
  major: number; // Major version number
  minor: number; // Minor version number
  patch: number; // Patch version number
  isSupported: boolean; // Whether this version meets minimum requirements
  isAvailable: boolean; // Whether version was successfully detected
}

/**
 * Version compatibility information including warnings
 */
export interface VersionCompatibility {
  version: GrafanaVersion;
  warnings: string[];
  minimumVersion: string;
}

/**
 * Minimum supported Grafana version (matches plugin.json)
 */
const MINIMUM_VERSION = {
  major: 10,
  minor: 4,
  patch: 0,
  full: '10.4.0',
};

/**
 * Parses a semantic version string into major.minor.patch components
 *
 * Supports standard semver format: "major.minor.patch" with optional prerelease suffix
 * Examples: "10.4.0", "11.0.0-beta1", "12.1.3-alpha.2+build.123"
 *
 * @param versionStr - Version string to parse
 * @returns Parsed version components
 * @throws Error if version format is invalid
 */
function parseVersion(versionStr: string): { full: string; major: number; minor: number; patch: number } {
  // Match major.minor.patch, ignore prerelease/metadata suffix
  const match = versionStr.match(/^(\d+)\.(\d+)\.(\d+)/);

  if (!match) {
    throw new Error(`Invalid version format: ${versionStr} (expected major.minor.patch)`);
  }

  return {
    full: versionStr,
    major: parseInt(match[1], 10),
    minor: parseInt(match[2], 10),
    patch: parseInt(match[3], 10),
  };
}

/**
 * Checks if a version meets the minimum supported version requirement
 *
 * @param version - Version to check
 * @returns true if version >= minimum supported version
 */
function isVersionSupported(version: { major: number; minor: number; patch: number }): boolean {
  if (version.major > MINIMUM_VERSION.major) {
    return true;
  }
  if (version.major < MINIMUM_VERSION.major) {
    return false;
  }

  if (version.minor > MINIMUM_VERSION.minor) {
    return true;
  }
  if (version.minor < MINIMUM_VERSION.minor) {
    return false;
  }

  return version.patch >= MINIMUM_VERSION.patch;
}

/**
 * Creates a GrafanaVersion object for when version is unavailable
 *
 * @returns Version object with isAvailable=false and isSupported=true (graceful fallback)
 */
function createUnavailableVersion(): GrafanaVersion {
  return {
    full: 'unknown',
    major: 0,
    minor: 0,
    patch: 0,
    isSupported: true, // Assume supported if unknown (graceful fallback)
    isAvailable: false,
  };
}

/**
 * Detects Grafana version from config.buildInfo
 *
 * Returns version object with isAvailable=false if detection fails (e.g., version reporting disabled)
 *
 * @returns Detected Grafana version
 */
export function detectGrafanaVersion(): GrafanaVersion {
  try {
    const buildInfo = config.buildInfo;

    if (!buildInfo?.version) {
      console.warn('Grafana version not available in config.buildInfo');
      return createUnavailableVersion();
    }

    const parsedVersion = parseVersion(buildInfo.version);

    return {
      ...parsedVersion,
      isSupported: isVersionSupported(parsedVersion),
      isAvailable: true,
    };
  } catch (error) {
    console.warn('Failed to detect Grafana version:', error);
    return createUnavailableVersion();
  }
}

/**
 * Checks version compatibility and returns warnings if any
 *
 * @returns Version compatibility information with warnings
 */
export function checkVersionCompatibility(): VersionCompatibility {
  const version = detectGrafanaVersion();
  const warnings: string[] = [];

  if (!version.isAvailable) {
    warnings.push(
      'Grafana version could not be detected. Some features may not work as expected on older versions. ' +
        'This may occur if version reporting is disabled in your Grafana settings.'
    );
  } else if (!version.isSupported) {
    warnings.push(
      `Grafana ${version.full} is below the minimum supported version ${MINIMUM_VERSION.full}. ` +
        'Please upgrade Grafana for full compatibility. Some features may not work correctly.'
    );
  }

  return {
    version,
    warnings,
    minimumVersion: MINIMUM_VERSION.full,
  };
}

/**
 * Singleton cache for version detection (runs once per session)
 */
let cachedVersion: GrafanaVersion | null = null;

/**
 * Gets the detected Grafana version (cached after first call)
 *
 * @returns Grafana version information
 */
export function getGrafanaVersion(): GrafanaVersion {
  if (!cachedVersion) {
    cachedVersion = detectGrafanaVersion();
  }
  return cachedVersion;
}

/**
 * Clears the cached version (useful for testing)
 */
export function clearVersionCache(): void {
  cachedVersion = null;
}
