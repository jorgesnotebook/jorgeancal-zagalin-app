/**
 * Context service for extracting Grafana context
 * This service provides context about the current dashboard, panel, time range, etc.
 */

import { config, getBackendSrv, locationService } from '@grafana/runtime';
import type {
  GrafanaContext,
  DashboardContext,
  PanelContext,
  TimeRange,
  TemplateVariable,
  AdhocFilter,
} from './contextTypes';
import type { GrafanaDashboardResponse, GrafanaPanel, GrafanaTarget } from '../types/grafana';
import { executePanelQueries, type PanelDataAnalysis } from './panelDataService';

export class ContextService {
  /**
   * Get the current Grafana context
   */
  static async getContext(): Promise<GrafanaContext> {
    const context: GrafanaContext = {};

    const location = locationService.getLocation();
    const pathname = location.pathname;

    const dashboardMatch = pathname.match(/\/d(?:ashboard)?\/([^/]+)/);
    const dashboardUid = dashboardMatch?.[1];

    if (dashboardUid) {
      context.dashboard = await this.getDashboardContext(dashboardUid);
      context.templateVariables = this.getTemplateVariables();
      context.adhocFilters = this.getAdhocFilters();
      context.timeRange = this.getTimeRange();
    }

    const panelMatch = location.search.match(/[?&]viewPanel=(\d+)/);
    const panelId = panelMatch ? parseInt(panelMatch[1], 10) : undefined;

    if (panelId && context.dashboard) {
      context.panel = context.dashboard.panels?.find((p) => p.id === panelId);
    }

    context.user = this.getUserContext();

    return context;
  }

  /**
   * Get dashboard context by UID
   */
  private static async getDashboardContext(uid: string): Promise<DashboardContext | undefined> {
    try {
      const response = await getBackendSrv().get<GrafanaDashboardResponse>(`/api/dashboards/uid/${uid}`);
      const dashboard = response.dashboard;

      const context: DashboardContext = {
        uid: dashboard.uid,
        title: dashboard.title,
        tags: dashboard.tags,
        timezone: dashboard.timezone,
        panels: dashboard.panels?.map((panel) => this.extractPanelContext(panel)) || [],
      };

      return context;
    } catch (error) {
      console.error('Failed to fetch dashboard context:', error);
      return undefined;
    }
  }

  /**
   * Extract panel context from panel model
   */
  private static extractPanelContext(panel: GrafanaPanel): PanelContext {
    return {
      id: panel.id,
      title: panel.title,
      type: panel.type,
      description: panel.description,
      targets:
        panel.targets?.map((target: GrafanaTarget) => ({
          refId: target.refId,
          datasource:
            target.datasource?.type && target.datasource?.uid
              ? {
                  type: target.datasource.type,
                  uid: target.datasource.uid,
                }
              : undefined,
          expr: target.expr,
          query: target.query,
          queryType: target.queryType,
        })) || [],
      fieldConfig: panel.fieldConfig,
      transformations: panel.transformations,
    };
  }

  /**
   * Get current time range from URL
   */
  private static getTimeRange(): TimeRange | undefined {
    const location = locationService.getLocation();
    const searchParams = new URLSearchParams(location.search);

    const from = searchParams.get('from');
    const to = searchParams.get('to');

    if (from && to) {
      return {
        from,
        to,
        raw: { from, to },
      };
    }

    return undefined;
  }

  /**
   * Get template variables from URL
   */
  private static getTemplateVariables(): TemplateVariable[] | undefined {
    const location = locationService.getLocation();
    const searchParams = new URLSearchParams(location.search);
    const variables: TemplateVariable[] = [];
    const processedVars = new Set<string>();

    for (const [key] of searchParams.entries()) {
      if (key.startsWith('var-') && !processedVars.has(key)) {
        const varName = key.substring(4);
        processedVars.add(key);

        // Get all values for this variable (supports multi-value variables)
        const values = searchParams.getAll(key);

        variables.push({
          name: varName,
          current: {
            value: values.length === 1 ? values[0] : values,
            text: values.length === 1 ? values[0] : values,
          },
        });
      }
    }

    return variables.length > 0 ? variables : undefined;
  }

  /**
   * Get current user context
   */
  private static getUserContext() {
    const user = config.bootData?.user;
    if (!user) {
      return undefined;
    }

    return {
      id: user.id,
      login: user.login,
      email: user.email,
      orgId: user.orgId,
    };
  }

  /**
   * Get ad-hoc filters from URL
   * Format: filters=key1|op1|value1&filters=key2|op2|value2
   */
  private static getAdhocFilters(): AdhocFilter[] | undefined {
    const location = locationService.getLocation();
    const searchParams = new URLSearchParams(location.search);
    const filters: AdhocFilter[] = [];

    // Get all 'filters' parameters
    const filterParams = searchParams.getAll('filters');

    for (const filterParam of filterParams) {
      // Parse format: key|operator|value
      const parts = filterParam.split('|');
      if (parts.length === 3) {
        filters.push({
          key: parts[0],
          operator: parts[1],
          value: parts[2],
        });
      }
    }

    return filters.length > 0 ? filters : undefined;
  }

  /**
   * Get context WITH panel data execution (for data-driven explanations)
   *
   * This method fixes the hallucination issue by fetching REAL data from panels.
   */
  static async getContextWithPanelData(): Promise<{
    context: GrafanaContext;
    panelData: PanelDataAnalysis[];
  }> {
    // Get basic context
    const context = await this.getContext();

    // Execute panel queries if dashboard is present
    let panelData: PanelDataAnalysis[] = [];

    if (context.dashboard && context.dashboard.panels && context.timeRange) {
      try {
        panelData = await executePanelQueries(
          context.dashboard.panels,
          context.timeRange,
          5 // Execute top 5 diagnostic panels
        );
      } catch (error) {
        console.error('Failed to execute panel queries:', error);
        // Continue without panel data rather than failing completely
      }
    }

    return { context, panelData };
  }

  /**
   * Format context as a system prompt for the LLM
   */
  static formatContextPrompt(context: GrafanaContext): string {
    const parts: string[] = [];

    if (context.dashboard) {
      parts.push(`Current Dashboard: "${context.dashboard.title}" (UID: ${context.dashboard.uid})`);

      if (context.dashboard.tags && context.dashboard.tags.length > 0) {
        parts.push(`Tags: ${context.dashboard.tags.join(', ')}`);
      }
    }

    if (context.panel) {
      parts.push(`\nViewing Panel: "${context.panel.title}" (ID: ${context.panel.id}, Type: ${context.panel.type})`);

      if (context.panel.description) {
        parts.push(`Description: ${context.panel.description}`);
      }

      if (context.panel.targets && context.panel.targets.length > 0) {
        parts.push(`\nQueries:`);
        context.panel.targets.forEach((target) => {
          const query = target.expr || target.query;
          if (query) {
            parts.push(`  ${target.refId}: ${query}`);
          }
          if (target.datasource) {
            parts.push(`    Datasource: ${target.datasource.type} (${target.datasource.uid})`);
          }
        });
      }

      if (context.panel.fieldConfig?.defaults?.unit) {
        parts.push(`Unit: ${context.panel.fieldConfig.defaults.unit}`);
      }
    }

    if (context.timeRange) {
      parts.push(`\nTime Range: ${context.timeRange.from} to ${context.timeRange.to}`);
    }

    if (context.templateVariables && context.templateVariables.length > 0) {
      parts.push(`\nTemplate Variables:`);
      context.templateVariables.forEach((v) => {
        parts.push(`  $${v.name} = ${v.current.value}`);
      });
    }

    if (parts.length === 0) {
      return '';
    }

    return `# Current Grafana Context\n\n${parts.join(
      '\n'
    )}\n\nUse this context to provide specific, relevant answers about this dashboard and panel.`;
  }

  /**
   * Format panel data as evidence for the LLM
   *
   * This provides REAL data to prevent hallucinations
   */
  static formatPanelDataPrompt(panelData: PanelDataAnalysis[]): string {
    if (!panelData || panelData.length === 0) {
      return '';
    }

    const parts: string[] = [
      '# Dashboard Panel Data (REAL VALUES)',
      '',
      '**CRITICAL**: The following data is REAL, fetched from actual panel queries.',
      'You MUST base your explanation on THIS DATA ONLY. Do NOT invent trends or values.',
      '',
    ];

    for (let i = 0; i < panelData.length; i++) {
      const panel = panelData[i];

      parts.push(`## Panel ${i + 1}: "${panel.panelTitle}" (${panel.panelType})`);
      parts.push(`**Query**: \`${panel.query}\``);
      parts.push(`**Datasource**: ${panel.datasourceType}`);
      parts.push('');

      if (!panel.success) {
        parts.push(`⚠️ **Status**: Query failed`);
        parts.push(`**Error**: ${panel.error}`);
        parts.push('');
        continue;
      }

      if (panel.hasNoData) {
        parts.push(`⚠️ **Status**: No data available`);
        parts.push('');
        continue;
      }

      // Success - show real data
      parts.push(`✅ **Status**: Data retrieved successfully`);
      parts.push('');
      parts.push(`**Current Value**: ${formatPanelValue(panel.currentValue, panel.unit)}`);
      parts.push(`**Trend**: ${panel.trend} (${formatChangePercent(panel.changePercent)})`);
      parts.push('');

      parts.push(`**Statistics**:`);
      parts.push(`- Min: ${formatPanelValue(panel.min, panel.unit)}`);
      parts.push(`- Max: ${formatPanelValue(panel.max, panel.unit)}`);
      parts.push(`- Avg: ${formatPanelValue(panel.avg, panel.unit)}`);
      parts.push('');

      // Anomalies
      const anomalies: string[] = [];
      if (panel.isSaturated) {
        anomalies.push('SATURATED (>90%)');
      }
      if (panel.hasSpike) {
        anomalies.push('SPIKE DETECTED');
      }
      if (panel.hasDrop) {
        anomalies.push('DROP DETECTED');
      }

      if (anomalies.length > 0) {
        parts.push(`⚠️ **Anomalies**: ${anomalies.join(', ')}`);
        parts.push('');
      }

      parts.push(`**Summary**: ${panel.summary}`);
      parts.push('');
      parts.push('---');
      parts.push('');
    }

    parts.push('**REMINDER**: You MUST explain what is happening based on THIS DATA.');
    parts.push('Do NOT make generic statements. Be specific and cite the panel values above.');

    return parts.join('\n');
  }
}

/**
 * Format panel value with unit
 */
function formatPanelValue(value: number | undefined, unit?: string): string {
  if (value === undefined || isNaN(value)) {
    return 'N/A';
  }

  if (unit === 'percentunit') {
    return `${(value * 100).toFixed(1)}%`;
  }
  if (unit === 'bytes') {
    return formatBytes(value);
  }
  if (unit === 'ms' || unit === 'µs' || unit === 's') {
    return `${value.toFixed(2)}${unit}`;
  }

  return value.toFixed(2);
}

/**
 * Format change percentage
 */
function formatChangePercent(percent: number | undefined): string {
  if (percent === undefined || isNaN(percent)) {
    return 'unknown';
  }

  const sign = percent > 0 ? '+' : '';
  return `${sign}${percent.toFixed(1)}%`;
}

/**
 * Format bytes to human-readable format
 */
function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return '0 B';
  }
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}
