/**
 * Unit tests for ContextManager
 */

import { ContextManager } from './ContextManager';
import type { GrafanaContext } from './types';
import { getBackendSrv, locationService } from '@grafana/runtime';

// Mock Grafana runtime
jest.mock('@grafana/runtime', () => ({
  config: {
    bootData: {
      user: {
        id: 1,
        login: 'testuser',
        email: 'test@example.com',
        orgId: 1,
      },
    },
  },
  getBackendSrv: jest.fn(),
  locationService: {
    getLocation: jest.fn(),
    getHistory: jest.fn(),
  },
}));

// Mock panelDataService
jest.mock('../panelDataService', () => ({
  executePanelQueries: jest.fn(),
}));

describe('ContextManager', () => {
  let manager: ContextManager;
  let mockGetBackendSrv: jest.MockedFunction<typeof getBackendSrv>;
  let mockLocationService: jest.Mocked<typeof locationService>;

  beforeEach(() => {
    manager = new ContextManager();
    mockGetBackendSrv = getBackendSrv as jest.MockedFunction<typeof getBackendSrv>;
    mockLocationService = locationService as jest.Mocked<typeof locationService>;
    jest.clearAllMocks();
  });

  describe('extractContext', () => {
    it('should extract context from dashboard page', async () => {
      // Mock location
      mockLocationService.getLocation.mockReturnValue({
        pathname: '/d/abc123/test-dashboard',
        search: '?from=now-1h&to=now&var-environment=prod',
      } as any);

      // Mock dashboard API
      mockGetBackendSrv.mockReturnValue({
        get: jest.fn().mockResolvedValue({
          dashboard: {
            uid: 'abc123',
            title: 'Test Dashboard',
            tags: ['test', 'monitoring'],
            timezone: 'utc',
            panels: [
              {
                id: 1,
                title: 'CPU Usage',
                type: 'timeseries',
                description: 'Shows CPU usage',
                targets: [
                  {
                    refId: 'A',
                    expr: 'rate(cpu_usage[5m])',
                    datasource: {
                      type: 'prometheus',
                      uid: 'prom-uid',
                    },
                  },
                ],
                fieldConfig: {
                  defaults: {
                    unit: 'percentunit',
                  },
                },
              },
            ],
          },
        }),
      } as any);

      const context = await manager.extractContext();

      expect(context.dashboard).toBeDefined();
      expect(context.dashboard?.uid).toBe('abc123');
      expect(context.dashboard?.title).toBe('Test Dashboard');
      expect(context.dashboard?.tags).toEqual(['test', 'monitoring']);
      expect(context.dashboard?.panels).toHaveLength(1);
      expect(context.timeRange).toBeDefined();
      expect(context.timeRange?.from).toBe('now-1h');
      expect(context.timeRange?.to).toBe('now');
      expect(context.templateVariables).toHaveLength(1);
      expect(context.templateVariables?.[0].name).toBe('environment');
      expect(context.user).toBeDefined();
      expect(context.user?.login).toBe('testuser');
    });

    it('should return empty context when not on dashboard page', async () => {
      mockLocationService.getLocation.mockReturnValue({
        pathname: '/explore',
        search: '',
      } as any);

      const context = await manager.extractContext();

      expect(context.dashboard).toBeUndefined();
      expect(context.panel).toBeUndefined();
      expect(context.timeRange).toBeUndefined();
      expect(context.user).toBeDefined();
    });

    it('should extract panel context when viewPanel query param present', async () => {
      mockLocationService.getLocation.mockReturnValue({
        pathname: '/d/abc123/test-dashboard',
        search: '?viewPanel=1',
      } as any);

      mockGetBackendSrv.mockReturnValue({
        get: jest.fn().mockResolvedValue({
          dashboard: {
            uid: 'abc123',
            title: 'Test Dashboard',
            panels: [
              {
                id: 1,
                title: 'CPU Usage',
                type: 'timeseries',
                targets: [],
              },
            ],
          },
        }),
      } as any);

      const context = await manager.extractContext();

      expect(context.panel).toBeDefined();
      expect(context.panel?.id).toBe(1);
      expect(context.panel?.title).toBe('CPU Usage');
    });

    it('should handle dashboard API errors gracefully', async () => {
      mockLocationService.getLocation.mockReturnValue({
        pathname: '/d/abc123/test-dashboard',
        search: '',
      } as any);

      mockGetBackendSrv.mockReturnValue({
        get: jest.fn().mockRejectedValue(new Error('API error')),
      } as any);

      const context = await manager.extractContext();

      expect(context.dashboard).toBeUndefined();
      expect(context.user).toBeDefined();
    });
  });

  describe('optimizeForTokenLimit', () => {
    it('should return full context when under token limit', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Test Dashboard',
        },
        timeRange: {
          from: 'now-1h',
          to: 'now',
        },
      };

      const optimized = manager.optimizeForTokenLimit(context, 10000);

      expect(optimized.essential).toContain('Test Dashboard');
      expect(optimized.supplemental).toBe('');
    });

    it('should optimize context when over token limit', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Very Long Dashboard Title That Would Exceed Token Limits'.repeat(100),
          tags: ['tag1', 'tag2', 'tag3'],
        },
        panel: {
          id: 1,
          title: 'Panel Title',
          type: 'timeseries',
          description: 'Very long description'.repeat(50),
          targets: [],
        },
        timeRange: {
          from: 'now-1h',
          to: 'now',
        },
      };

      const optimized = manager.optimizeForTokenLimit(context, 100);

      expect(optimized.essential).toBeDefined();
      expect(optimized.supplemental).toBeDefined();
      const essentialObj = JSON.parse(optimized.essential);
      expect(essentialObj.dashboard).toBeDefined();
      expect(essentialObj.panel).toBeDefined();
    });
  });

  describe('isDashboardQuestion', () => {
    const context: GrafanaContext = {
      dashboard: {
        uid: 'abc123',
        title: 'Test Dashboard',
      },
    };

    it('should detect dashboard questions', () => {
      expect(manager.isDashboardQuestion('what am i seeing', context)).toBe(true);
      expect(manager.isDashboardQuestion('explain this dashboard', context)).toBe(true);
      expect(manager.isDashboardQuestion('what is this panel', context)).toBe(true);
      expect(manager.isDashboardQuestion('what are the trends', context)).toBe(true);
    });

    it('should not detect non-dashboard questions', () => {
      expect(manager.isDashboardQuestion('how do I query prometheus', context)).toBe(false);
      expect(manager.isDashboardQuestion('what is grafana', context)).toBe(false);
      expect(manager.isDashboardQuestion('help me create a dashboard', context)).toBe(false);
    });

    it('should return false when no dashboard context', () => {
      const emptyContext: GrafanaContext = {};
      expect(manager.isDashboardQuestion('what am i seeing', emptyContext)).toBe(false);
    });
  });

  describe('readDashboardPanels', () => {
    it('should read panels with queries', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Test Dashboard',
          panels: [
            {
              id: 1,
              title: 'CPU Usage',
              type: 'timeseries',
              targets: [
                {
                  refId: 'A',
                  expr: 'rate(cpu_usage[5m])',
                  datasource: {
                    type: 'prometheus',
                    uid: 'prom-uid',
                  },
                },
              ],
            },
            {
              id: 2,
              title: 'Memory Usage',
              type: 'gauge',
              targets: [
                {
                  refId: 'A',
                  query: '{job="app"}',
                  datasource: {
                    type: 'loki',
                    uid: 'loki-uid',
                  },
                },
              ],
            },
          ],
        },
      };

      const panels = manager.readDashboardPanels(context);

      expect(panels).toHaveLength(2);
      expect(panels[0].panelTitle).toBe('CPU Usage');
      expect(panels[0].query).toBe('rate(cpu_usage[5m])');
      expect(panels[0].datasourceType).toBe('prometheus');
      expect(panels[1].panelTitle).toBe('Memory Usage');
      expect(panels[1].query).toBe('{job="app"}');
    });

    it('should return empty array when no dashboard', () => {
      const context: GrafanaContext = {};
      const panels = manager.readDashboardPanels(context);
      expect(panels).toEqual([]);
    });

    it('should skip panels without queries', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Test Dashboard',
          panels: [
            {
              id: 1,
              title: 'Text Panel',
              type: 'text',
              targets: [],
            },
          ],
        },
      };

      const panels = manager.readDashboardPanels(context);
      expect(panels).toEqual([]);
    });
  });

  describe('buildDashboardSummaryPrompt', () => {
    it('should build prompt with panels', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Test Dashboard',
        },
        timeRange: {
          from: 'now-1h',
          to: 'now',
        },
      };

      const panels = [
        {
          panelTitle: 'CPU Usage',
          panelType: 'timeseries',
          query: 'rate(cpu[5m])',
          datasourceType: 'prometheus',
          summary: 'Shows CPU usage',
        },
      ];

      const prompt = manager.buildDashboardSummaryPrompt('What am I seeing?', context, panels);

      expect(prompt).toContain('What am I seeing?');
      expect(prompt).toContain('Test Dashboard');
      expect(prompt).toContain('now-1h');
      expect(prompt).toContain('CPU Usage');
      expect(prompt).toContain('rate(cpu[5m])');
      expect(prompt).toContain('prometheus');
    });

    it('should handle empty panels', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Empty Dashboard',
        },
      };

      const prompt = manager.buildDashboardSummaryPrompt('What is this?', context, []);

      expect(prompt).toContain('Empty Dashboard');
      expect(prompt).toContain("doesn't have any visible panels");
    });
  });

  describe('formatContextPrompt', () => {
    it('should format context for LLM', () => {
      const context: GrafanaContext = {
        dashboard: {
          uid: 'abc123',
          title: 'Test Dashboard',
          tags: ['monitoring', 'test'],
        },
        panel: {
          id: 1,
          title: 'CPU Usage',
          type: 'timeseries',
          description: 'CPU metrics',
          targets: [
            {
              refId: 'A',
              expr: 'rate(cpu[5m])',
              datasource: {
                type: 'prometheus',
                uid: 'prom-uid',
              },
            },
          ],
          fieldConfig: {
            defaults: {
              unit: 'percentunit',
            },
          },
        },
        timeRange: {
          from: 'now-1h',
          to: 'now',
        },
        templateVariables: [
          {
            name: 'environment',
            current: {
              value: 'prod',
              text: 'prod',
            },
          },
        ],
      };

      const prompt = manager.formatContextPrompt(context);

      expect(prompt).toContain('Current Dashboard: "Test Dashboard"');
      expect(prompt).toContain('Tags: monitoring, test');
      expect(prompt).toContain('Viewing Panel: "CPU Usage"');
      expect(prompt).toContain('rate(cpu[5m])');
      expect(prompt).toContain('Time Range: now-1h to now');
      expect(prompt).toContain('$environment = prod');
      expect(prompt).toContain('Unit: percentunit');
    });

    it('should return empty string for empty context', () => {
      const context: GrafanaContext = {};
      const prompt = manager.formatContextPrompt(context);
      expect(prompt).toBe('');
    });
  });

  describe('calculateContextBudget', () => {
    it('should calculate 25% of max tokens', () => {
      expect(manager.calculateContextBudget(4000)).toBe(1000);
      expect(manager.calculateContextBudget(8000)).toBe(2000);
      expect(manager.calculateContextBudget(1000)).toBe(250);
    });
  });
});
