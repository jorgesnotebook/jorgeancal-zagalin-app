/**
 * Tests for ContextService
 * Covers context extraction, URL parsing, template variables, and prompt formatting
 */

import { ContextService } from './contextService';
import { getBackendSrv, locationService, config } from '@grafana/runtime';
import { executePanelQueries, type PanelDataAnalysis } from './panelDataService';

// Mock dependencies
jest.mock('@grafana/runtime', () => ({
  getBackendSrv: jest.fn(),
  locationService: {
    getLocation: jest.fn(),
  },
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
}));

jest.mock('./panelDataService', () => ({
  executePanelQueries: jest.fn(),
}));

describe('ContextService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Context Extraction', () => {
    describe('getContext()', () => {
      it('should extract dashboard UID from standard dashboard URL', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/my-dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'My Dashboard',
              tags: [],
              panels: [],
            },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.dashboard?.uid).toBe('abc123');
        expect(context.dashboard?.title).toBe('My Dashboard');
      });

      it('should extract dashboard UID from legacy dashboard URL', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/dashboard/def456/legacy-dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'def456',
              title: 'Legacy Dashboard',
              tags: [],
              panels: [],
            },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.dashboard?.uid).toBe('def456');
      });

      it('should return empty context for non-dashboard pages', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/explore',
          search: '',
        });

        const context = await ContextService.getContext();

        expect(context.dashboard).toBeUndefined();
        expect(context.timeRange).toBeUndefined();
        expect(context.templateVariables).toBeUndefined();
      });

      it('should extract panel ID from viewPanel parameter', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?viewPanel=42',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [
                { id: 42, title: 'My Panel', type: 'graph' },
                { id: 43, title: 'Other Panel', type: 'table' },
              ],
            },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.panel?.id).toBe(42);
        expect(context.panel?.title).toBe('My Panel');
      });

      it('should include user context', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/explore',
          search: '',
        });

        const context = await ContextService.getContext();

        expect(context.user).toEqual({
          id: 1,
          login: 'testuser',
          email: 'test@example.com',
          orgId: 1,
        });
      });

      it('should handle missing user context gracefully', async () => {
        const originalUser = config.bootData?.user;
        // @ts-ignore - testing undefined user
        config.bootData.user = undefined;

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/explore',
          search: '',
        });

        const context = await ContextService.getContext();

        expect(context.user).toBeUndefined();

        // Restore
        // @ts-ignore
        config.bootData.user = originalUser;
      });

      it('should handle dashboard fetch errors gracefully', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockRejectedValue(new Error('Not found')),
        });

        const context = await ContextService.getContext();

        expect(context.dashboard).toBeUndefined();
        expect(context).toHaveProperty('user');
      });
    });

    describe('getDashboardContext()', () => {
      it('should fetch and parse dashboard data', async () => {
        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Test Dashboard',
              tags: ['monitoring', 'prod'],
              timezone: 'UTC',
              panels: [
                {
                  id: 1,
                  title: 'CPU Usage',
                  type: 'graph',
                  description: 'CPU usage over time',
                  targets: [
                    {
                      refId: 'A',
                      expr: 'up',
                      datasource: { type: 'prometheus', uid: 'prom-1' },
                    },
                  ],
                },
              ],
            },
          }),
        });

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        const context = await ContextService.getContext();

        expect(context.dashboard?.uid).toBe('abc123');
        expect(context.dashboard?.title).toBe('Test Dashboard');
        expect(context.dashboard?.tags).toEqual(['monitoring', 'prod']);
        expect(context.dashboard?.timezone).toBe('UTC');
        expect(context.dashboard?.panels).toHaveLength(1);
      });

      it('should handle dashboard with no panels', async () => {
        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Empty Dashboard',
              panels: undefined,
            },
          }),
        });

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        const context = await ContextService.getContext();

        expect(context.dashboard?.panels).toEqual([]);
      });
    });

    describe('extractPanelContext()', () => {
      it('should extract panel with all properties', async () => {
        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [
                {
                  id: 1,
                  title: 'CPU Panel',
                  type: 'graph',
                  description: 'CPU metrics',
                  targets: [
                    {
                      refId: 'A',
                      expr: 'rate(cpu[5m])',
                      datasource: { type: 'prometheus', uid: 'prom-1' },
                    },
                  ],
                  fieldConfig: {
                    defaults: {
                      unit: 'percent',
                    },
                  },
                  transformations: [{ id: 'organize' }],
                },
              ],
            },
          }),
        });

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        const context = await ContextService.getContext();
        const panel = context.dashboard?.panels?.[0];

        expect(panel?.id).toBe(1);
        expect(panel?.title).toBe('CPU Panel');
        expect(panel?.type).toBe('graph');
        expect(panel?.description).toBe('CPU metrics');
        expect(panel?.targets).toHaveLength(1);
        expect(panel?.fieldConfig).toBeDefined();
        expect(panel?.transformations).toHaveLength(1);
      });

      it('should handle panel with multiple targets', async () => {
        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [
                {
                  id: 1,
                  title: 'Multi-query Panel',
                  type: 'graph',
                  targets: [
                    { refId: 'A', expr: 'query_a', datasource: { type: 'prometheus', uid: 'prom-1' } },
                    { refId: 'B', query: 'query_b', datasource: { type: 'loki', uid: 'loki-1' } },
                  ],
                },
              ],
            },
          }),
        });

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        const context = await ContextService.getContext();
        const panel = context.dashboard?.panels?.[0];

        expect(panel?.targets).toHaveLength(2);
        expect(panel?.targets?.[0].expr).toBe('query_a');
        expect(panel?.targets?.[1].query).toBe('query_b');
      });

      it('should handle panel with no targets', async () => {
        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [
                {
                  id: 1,
                  title: 'Text Panel',
                  type: 'text',
                  targets: undefined,
                },
              ],
            },
          }),
        });

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        const context = await ContextService.getContext();
        const panel = context.dashboard?.panels?.[0];

        expect(panel?.targets).toEqual([]);
      });
    });
  });

  describe('URL Parsing', () => {
    describe('getTimeRange()', () => {
      it('should extract time range from URL parameters', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h&to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.timeRange?.from).toBe('now-1h');
        expect(context.timeRange?.to).toBe('now');
        expect(context.timeRange?.raw).toEqual({ from: 'now-1h', to: 'now' });
      });

      it('should handle absolute time ranges', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=1609459200000&to=1609545600000',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.timeRange?.from).toBe('1609459200000');
        expect(context.timeRange?.to).toBe('1609545600000');
      });

      it('should return undefined when time range is missing', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.timeRange).toBeUndefined();
      });

      it('should return undefined when only from is present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.timeRange).toBeUndefined();
      });

      it('should return undefined when only to is present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.timeRange).toBeUndefined();
      });
    });

    describe('Dashboard and Panel ID extraction', () => {
      it('should extract dashboard UID with special characters', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc-123_xyz/dashboard-name',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc-123_xyz', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.dashboard?.uid).toBe('abc-123_xyz');
      });

      it('should extract panel ID from viewPanel with other parameters', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h&viewPanel=5&to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [{ id: 5, title: 'Panel 5', type: 'graph' }],
            },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.panel?.id).toBe(5);
      });

      it('should return undefined panel when viewPanel is not found', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?viewPanel=999',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [{ id: 1, title: 'Panel 1', type: 'graph' }],
            },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.panel).toBeUndefined();
      });
    });
  });

  describe('Template Variables', () => {
    describe('getTemplateVariables()', () => {
      it('should extract single-value template variables', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?var-env=prod&var-region=us-east-1',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toHaveLength(2);
        expect(context.templateVariables?.[0]).toEqual({
          name: 'env',
          current: {
            value: 'prod',
            text: 'prod',
          },
        });
        expect(context.templateVariables?.[1]).toEqual({
          name: 'region',
          current: {
            value: 'us-east-1',
            text: 'us-east-1',
          },
        });
      });

      it('should extract multi-value template variables', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?var-region=us-east-1&var-region=us-west-2&var-region=eu-west-1',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toHaveLength(1);
        expect(context.templateVariables?.[0]).toEqual({
          name: 'region',
          current: {
            value: ['us-east-1', 'us-west-2', 'eu-west-1'],
            text: ['us-east-1', 'us-west-2', 'eu-west-1'],
          },
        });
      });

      it('should handle variables with special characters', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?var-label=app%3Dapi&var-filter=status%3A200',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toHaveLength(2);
        expect(context.templateVariables?.find((v) => v.name === 'label')?.current.value).toBe('app=api');
        expect(context.templateVariables?.find((v) => v.name === 'filter')?.current.value).toBe('status:200');
      });

      it('should return undefined when no variables present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h&to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toBeUndefined();
      });

      it('should handle empty variable values', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?var-env=',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toHaveLength(1);
        expect(context.templateVariables?.[0].current.value).toBe('');
      });

      it('should handle mix of single and multi-value variables', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?var-env=prod&var-region=us-east-1&var-region=us-west-2&var-cluster=main',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.templateVariables).toHaveLength(3);

        const envVar = context.templateVariables?.find((v) => v.name === 'env');
        expect(envVar?.current.value).toBe('prod');

        const regionVar = context.templateVariables?.find((v) => v.name === 'region');
        expect(Array.isArray(regionVar?.current.value)).toBe(true);
        expect(regionVar?.current.value).toEqual(['us-east-1', 'us-west-2']);

        const clusterVar = context.templateVariables?.find((v) => v.name === 'cluster');
        expect(clusterVar?.current.value).toBe('main');
      });
    });

    describe('getAdhocFilters()', () => {
      it('should extract adhoc filters', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?filters=env|=|prod&filters=status|=~|2..',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.adhocFilters).toHaveLength(2);
        expect(context.adhocFilters?.[0]).toEqual({
          key: 'env',
          operator: '=',
          value: 'prod',
        });
        expect(context.adhocFilters?.[1]).toEqual({
          key: 'status',
          operator: '=~',
          value: '2..',
        });
      });

      it('should return undefined when no filters present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.adhocFilters).toBeUndefined();
      });

      it('should ignore malformed filter strings', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?filters=env|=|prod&filters=invalid&filters=status|=|active',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: { uid: 'abc123', title: 'Dashboard', panels: [] },
          }),
        });

        const context = await ContextService.getContext();

        expect(context.adhocFilters).toHaveLength(2);
        expect(context.adhocFilters?.find((f) => f.key === 'env')).toBeDefined();
        expect(context.adhocFilters?.find((f) => f.key === 'status')).toBeDefined();
      });
    });
  });

  describe('Panel Data Execution', () => {
    describe('getContextWithPanelData()', () => {
      it('should execute panel queries when dashboard and timeRange present', async () => {
        const mockPanelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'CPU Usage',
            panelType: 'graph',
            query: 'rate(cpu[5m])',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: false,
            currentValue: 0.85,
            trend: 'increasing',
            changePercent: 15.3,
            min: 0.1,
            max: 0.95,
            avg: 0.5,
            isSaturated: false,
            hasSpike: false,
            hasDrop: false,
            summary: 'CPU usage is increasing by 15.3%',
          },
        ];

        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h&to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [{ id: 1, title: 'CPU Usage', type: 'graph' }],
            },
          }),
        });

        (executePanelQueries as jest.Mock).mockResolvedValue(mockPanelData);

        const result = await ContextService.getContextWithPanelData();

        expect(result.context.dashboard?.uid).toBe('abc123');
        expect(result.panelData).toEqual(mockPanelData);
        expect(executePanelQueries).toHaveBeenCalledWith(expect.any(Array), expect.any(Object), 5);
      });

      it('should return empty panelData when no dashboard present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/explore',
          search: '',
        });

        const result = await ContextService.getContextWithPanelData();

        expect(result.context.dashboard).toBeUndefined();
        expect(result.panelData).toEqual([]);
        expect(executePanelQueries).not.toHaveBeenCalled();
      });

      it('should return empty panelData when no timeRange present', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [{ id: 1, title: 'Panel', type: 'graph' }],
            },
          }),
        });

        const result = await ContextService.getContextWithPanelData();

        expect(result.context.dashboard?.uid).toBe('abc123');
        expect(result.panelData).toEqual([]);
        expect(executePanelQueries).not.toHaveBeenCalled();
      });

      it('should handle panel query execution errors gracefully', async () => {
        (locationService.getLocation as jest.Mock).mockReturnValue({
          pathname: '/d/abc123/dashboard',
          search: '?from=now-1h&to=now',
        });

        (getBackendSrv as jest.Mock).mockReturnValue({
          get: jest.fn().mockResolvedValue({
            dashboard: {
              uid: 'abc123',
              title: 'Dashboard',
              panels: [{ id: 1, title: 'Panel', type: 'graph' }],
            },
          }),
        });

        (executePanelQueries as jest.Mock).mockRejectedValue(new Error('Query failed'));

        const result = await ContextService.getContextWithPanelData();

        expect(result.context.dashboard?.uid).toBe('abc123');
        expect(result.panelData).toEqual([]);
      });
    });
  });

  describe('Prompt Formatting', () => {
    describe('formatContextPrompt()', () => {
      it('should format dashboard context', () => {
        const context = {
          dashboard: {
            uid: 'abc123',
            title: 'Production Dashboard',
            tags: ['monitoring', 'prod'],
            panels: [],
          },
        };

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toContain('Production Dashboard');
        expect(prompt).toContain('abc123');
        expect(prompt).toContain('monitoring, prod');
      });

      it('should format panel context', () => {
        const context = {
          panel: {
            id: 1,
            title: 'CPU Usage',
            type: 'graph',
            description: 'Shows CPU usage over time',
            targets: [
              {
                refId: 'A',
                expr: 'rate(cpu[5m])',
                datasource: { type: 'prometheus', uid: 'prom-1' },
              },
            ],
            fieldConfig: {
              defaults: {
                unit: 'percent',
              },
            },
          },
        };

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toContain('CPU Usage');
        expect(prompt).toContain('graph');
        expect(prompt).toContain('Shows CPU usage over time');
        expect(prompt).toContain('rate(cpu[5m])');
        expect(prompt).toContain('prometheus');
        expect(prompt).toContain('percent');
      });

      it('should format time range', () => {
        const context = {
          timeRange: {
            from: 'now-1h',
            to: 'now',
            raw: { from: 'now-1h', to: 'now' },
          },
        };

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toContain('now-1h');
        expect(prompt).toContain('now');
      });

      it('should format template variables', () => {
        const context = {
          templateVariables: [
            { name: 'env', current: { value: 'prod', text: 'prod' } },
            { name: 'region', current: { value: ['us-east-1', 'us-west-2'], text: ['us-east-1', 'us-west-2'] } },
          ],
        };

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toContain('$env = prod');
        expect(prompt).toContain('$region = us-east-1,us-west-2');
      });

      it('should return empty string for empty context', () => {
        const context = {};

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toBe('');
      });

      it('should format complete context with all fields', () => {
        const context = {
          dashboard: {
            uid: 'abc123',
            title: 'Production Dashboard',
            tags: ['prod'],
            panels: [],
          },
          panel: {
            id: 1,
            title: 'CPU Panel',
            type: 'graph',
            targets: [{ refId: 'A', expr: 'up' }],
          },
          timeRange: {
            from: 'now-1h',
            to: 'now',
            raw: { from: 'now-1h', to: 'now' },
          },
          templateVariables: [{ name: 'env', current: { value: 'prod', text: 'prod' } }],
        };

        const prompt = ContextService.formatContextPrompt(context);

        expect(prompt).toContain('Production Dashboard');
        expect(prompt).toContain('CPU Panel');
        expect(prompt).toContain('now-1h');
        expect(prompt).toContain('$env = prod');
      });
    });

    describe('formatPanelDataPrompt()', () => {
      it('should format successful panel data', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'CPU Usage',
            panelType: 'graph',
            query: 'rate(cpu[5m])',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: false,
            currentValue: 0.85,
            trend: 'increasing',
            changePercent: 15.3,
            min: 0.1,
            max: 0.95,
            avg: 0.5,
            unit: 'percentunit',
            isSaturated: false,
            hasSpike: false,
            hasDrop: false,
            summary: 'CPU usage is increasing',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('CPU Usage');
        expect(prompt).toContain('rate(cpu[5m])');
        expect(prompt).toContain('prometheus');
        expect(prompt).toContain('85.0%');
        expect(prompt).toContain('increasing');
        expect(prompt).toContain('+15.3%');
      });

      it('should format panel data with anomalies', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'Error Rate',
            panelType: 'graph',
            query: 'rate(errors[5m])',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: false,
            currentValue: 0.95,
            trend: 'increasing',
            changePercent: 200,
            min: 0.1,
            max: 0.95,
            avg: 0.3,
            isSaturated: true,
            hasSpike: true,
            hasDrop: false,
            summary: 'Error rate has spiked',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('SATURATED');
        expect(prompt).toContain('SPIKE DETECTED');
      });

      it('should format failed panel query', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'Failed Panel',
            panelType: 'graph',
            query: 'invalid_query',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: false,
            hasNoData: false,
            error: 'Query execution failed',
            summary: '',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('Failed Panel');
        expect(prompt).toContain('Query failed');
        expect(prompt).toContain('Query execution failed');
      });

      it('should format panel with no data', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'Empty Panel',
            panelType: 'graph',
            query: 'metric_with_no_data',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: true,
            summary: '',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('Empty Panel');
        expect(prompt).toContain('No data available');
      });

      it('should return empty string for empty panel data', () => {
        const prompt = ContextService.formatPanelDataPrompt([]);

        expect(prompt).toBe('');
      });

      it('should format bytes unit correctly', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'Memory Usage',
            panelType: 'graph',
            query: 'memory_usage',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: false,
            currentValue: 1073741824,
            unit: 'bytes',
            trend: 'stable',
            changePercent: 0,
            min: 0,
            max: 2147483648,
            avg: 1073741824,
            summary: 'Memory usage is stable',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('1.00 GB');
      });

      it('should format time units correctly', () => {
        const panelData: PanelDataAnalysis[] = [
          {
            panelTitle: 'Response Time',
            panelType: 'graph',
            query: 'response_time',
            datasourceUid: 'prom-1',
            datasourceType: 'prometheus',
            success: true,
            hasNoData: false,
            currentValue: 250,
            unit: 'ms',
            trend: 'stable',
            changePercent: 0,
            min: 100,
            max: 500,
            avg: 300,
            summary: 'Response time is stable',
          },
        ];

        const prompt = ContextService.formatPanelDataPrompt(panelData);

        expect(prompt).toContain('250.00ms');
        expect(prompt).toContain('100.00ms');
        expect(prompt).toContain('500.00ms');
      });
    });
  });
});
