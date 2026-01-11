import {
  detectGrafanaVersion,
  checkVersionCompatibility,
  getGrafanaVersion,
  clearVersionCache,
} from './versionDetector';
import { config } from '@grafana/runtime';

// Mock @grafana/runtime
jest.mock('@grafana/runtime', () => ({
  config: {
    buildInfo: {
      version: '10.4.0',
    },
  },
}));

describe('versionDetector', () => {
  beforeEach(() => {
    // Clear cache before each test
    clearVersionCache();

    // Ensure config.buildInfo is always initialized as an object
    (config as any).buildInfo = {
      version: '10.4.0',
    };
  });

  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('detectGrafanaVersion', () => {
    it('should detect standard version', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version = detectGrafanaVersion();

      expect(version.full).toBe('10.4.0');
      expect(version.major).toBe(10);
      expect(version.minor).toBe(4);
      expect(version.patch).toBe(0);
      expect(version.isSupported).toBe(true);
      expect(version.isAvailable).toBe(true);
    });

    it('should detect version with prerelease suffix', () => {
      (config.buildInfo as any).version = '11.0.0-beta1';

      const version = detectGrafanaVersion();

      expect(version.full).toBe('11.0.0-beta1');
      expect(version.major).toBe(11);
      expect(version.minor).toBe(0);
      expect(version.patch).toBe(0);
      expect(version.isSupported).toBe(true);
      expect(version.isAvailable).toBe(true);
    });

    it('should detect version with prerelease and metadata', () => {
      (config.buildInfo as any).version = '12.1.3-alpha.2+build.123';

      const version = detectGrafanaVersion();

      expect(version.full).toBe('12.1.3-alpha.2+build.123');
      expect(version.major).toBe(12);
      expect(version.minor).toBe(1);
      expect(version.patch).toBe(3);
      expect(version.isSupported).toBe(true);
      expect(version.isAvailable).toBe(true);
    });

    it('should return unavailable version when buildInfo is missing', () => {
      (config as any).buildInfo = undefined;

      const version = detectGrafanaVersion();

      expect(version.full).toBe('unknown');
      expect(version.major).toBe(0);
      expect(version.minor).toBe(0);
      expect(version.patch).toBe(0);
      expect(version.isSupported).toBe(true); // Graceful fallback
      expect(version.isAvailable).toBe(false);
    });

    it('should return unavailable version when version is missing', () => {
      (config.buildInfo as any).version = undefined;

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true); // Graceful fallback
    });

    it('should return unavailable version when version is null', () => {
      (config.buildInfo as any).version = null;

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true);
    });

    it('should handle invalid version format gracefully', () => {
      (config.buildInfo as any).version = 'invalid-version';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true); // Graceful fallback
    });

    it('should handle version with only major.minor', () => {
      (config.buildInfo as any).version = '10.4';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true); // Graceful fallback
    });

    it('should handle version with letters in patch', () => {
      (config.buildInfo as any).version = '10.4.abc';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true); // Graceful fallback
    });
  });

  describe('version support check', () => {
    it('should support exact minimum version', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
    });

    it('should support version above minimum major', () => {
      (config.buildInfo as any).version = '11.0.0';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
    });

    it('should support version above minimum minor', () => {
      (config.buildInfo as any).version = '10.5.0';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
    });

    it('should support version above minimum patch', () => {
      (config.buildInfo as any).version = '10.4.1';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
    });

    it('should not support version below minimum major', () => {
      (config.buildInfo as any).version = '9.5.0';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(false);
    });

    it('should not support version below minimum minor', () => {
      (config.buildInfo as any).version = '10.3.9';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(false);
    });

    it('should support much newer versions', () => {
      (config.buildInfo as any).version = '15.2.3';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
    });
  });

  describe('checkVersionCompatibility', () => {
    it('should return no warnings for supported version', () => {
      (config.buildInfo as any).version = '10.4.0';

      const result = checkVersionCompatibility();

      expect(result.warnings).toHaveLength(0);
      expect(result.version.isSupported).toBe(true);
      expect(result.minimumVersion).toBe('10.4.0');
    });

    it('should return no warnings for newer version', () => {
      (config.buildInfo as any).version = '11.0.0';

      const result = checkVersionCompatibility();

      expect(result.warnings).toHaveLength(0);
      expect(result.version.isSupported).toBe(true);
    });

    it('should return warning when version unavailable', () => {
      (config as any).buildInfo = undefined;

      const result = checkVersionCompatibility();

      expect(result.warnings.length).toBeGreaterThan(0);
      expect(result.warnings[0]).toContain('could not be detected');
      expect(result.version.isAvailable).toBe(false);
    });

    it('should return warning for unsupported version', () => {
      (config.buildInfo as any).version = '10.3.0';

      const result = checkVersionCompatibility();

      expect(result.warnings.length).toBeGreaterThan(0);
      expect(result.warnings[0]).toContain('below');
      expect(result.warnings[0]).toContain('10.4.0');
      expect(result.version.isSupported).toBe(false);
    });

    it('should return warning for very old version', () => {
      (config.buildInfo as any).version = '9.0.0';

      const result = checkVersionCompatibility();

      expect(result.warnings.length).toBeGreaterThan(0);
      expect(result.warnings[0]).toContain('below');
      expect(result.version.isSupported).toBe(false);
    });
  });

  describe('getGrafanaVersion (cached)', () => {
    it('should cache version after first call', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version1 = getGrafanaVersion();
      const version2 = getGrafanaVersion();

      expect(version1).toBe(version2); // Same object reference
    });

    it('should use cached version even if config changes', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version1 = getGrafanaVersion();

      // Change config
      (config.buildInfo as any).version = '11.0.0';

      const version2 = getGrafanaVersion();

      expect(version1).toBe(version2); // Still uses cached version
      expect(version2.full).toBe('10.4.0'); // Original cached value
    });

    it('should refresh cache after clearVersionCache', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version1 = getGrafanaVersion();
      expect(version1.full).toBe('10.4.0');

      // Clear cache and change config
      clearVersionCache();
      (config.buildInfo as any).version = '11.0.0';

      const version2 = getGrafanaVersion();

      expect(version2.full).toBe('11.0.0'); // Uses new value
      expect(version1).not.toBe(version2); // Different objects
    });
  });

  describe('edge cases', () => {
    it('should handle empty string version', () => {
      (config.buildInfo as any).version = '';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false);
      expect(version.isSupported).toBe(true); // Graceful fallback
    });

    it('should handle version with extra whitespace', () => {
      (config.buildInfo as any).version = '  10.4.0  ';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false); // Regex won't match with leading whitespace
    });

    it('should handle version with v prefix', () => {
      (config.buildInfo as any).version = 'v10.4.0';

      const version = detectGrafanaVersion();

      expect(version.isAvailable).toBe(false); // Regex expects numeric start
    });

    it('should handle minimum version with patch zero', () => {
      (config.buildInfo as any).version = '10.4.0';

      const version = detectGrafanaVersion();

      expect(version.isSupported).toBe(true);
      expect(version.major).toBe(10);
      expect(version.minor).toBe(4);
      expect(version.patch).toBe(0);
    });
  });
});
