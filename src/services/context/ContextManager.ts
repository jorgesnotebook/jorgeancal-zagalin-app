/**
 * Context Manager - Extract and manage Grafana context
 *
 * Consolidates:
 * - contextService.ts (412 LOC) - Grafana context extraction
 * - dashboardReader.ts (175 LOC) - Dashboard question detection
 * - contextOptimizer.ts (90 LOC) - Context size reduction
 */

import { config, getBackendSrv, locationService } from '@grafana/runtime';
import type {
  GrafanaContext,
  DashboardContext,
  PanelContext,
  TimeRange,
  TemplateVariable,
  AdhocFilter,
  OptimizedContext,
  PanelData,
} from './types';
import type { GrafanaDashboardResponse, GrafanaPanel, GrafanaTarget } from '../../types/grafana';
import { executePanelQueries, type PanelDataAnalysis } from '../panelDataService';

export class ContextManager {
  /**
   * Extract full Grafana context from current page
   */
  async extractContext(): Promise<GrafanaContext> {
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
   * Get context WITH panel data execution (for data-driven explanations)
   */
  async extractContextWithPanelData(): Promise<{
    context: GrafanaContext;
    panelData: PanelDataAnalysis[];
  }> {
    const context = await this.extractContext();

    let panelData: PanelDataAnalysis[] = [];

    if (context.dashboard && context.dashboard.panels && context.timeRange) {
      try {
        panelData = await executePanelQueries(context.dashboard.panels, context.timeRange, 5);
      } catch (error) {
        console.error('Failed to execute panel queries:', error);
      }
    }

    return { context, panelData };
  }

  /**
   * Optimize context for token limits
   */
  optimizeForTokenLimit(context: GrafanaContext, maxTokens = 1000): OptimizedContext {
    const estimated = this.estimateTokens(context);

    if (estimated <= maxTokens) {
      return {
        essential: this.formatFullContext(context),
        supplemental: '',
        metadata: {},
      };
    }

    const essential = {
      dashboard: context.dashboard?.title,
      dashboardUid: context.dashboard?.uid,
      panel: context.panel?.title,
      timeRange: context.timeRange ? `${context.timeRange.from} to ${context.timeRange.to}` : undefined,
    };

    const supplemental = {
      dashboardTags: context.dashboard?.tags,
      panelDescription: context.panel?.description,
    };

    return {
      essential: JSON.stringify(essential, null, 2),
      supplemental: JSON.stringify(supplemental, null, 2),
      metadata: {},
    };
  }

  /**
   * Check if question is dashboard-specific
   */
  isDashboardQuestion(message: string, context: GrafanaContext): boolean {
    if (!context.dashboard) {
      return false;
    }

    const lowerMessage = message.toLowerCase();

    const dashboardQuestions = [
      'what am i seeing',
      'what do i see',
      'what does this show',
      'what is this',
      'explain this dashboard',
      'what is this dashboard',
      'describe this dashboard',
      'what are these panels',
      'what am i looking at',
      'what does this dashboard',
      'analyze this dashboard',
      "what's on this dashboard",
      'tell me about this dashboard',
      'what is this panel',
      'what is the panel',
      'what data',
      'what metrics',
      'what metric',
      "what's this showing",
      'what is displayed',
      'what are the trends',
      'what trends',
      'displayed in',
      'showing in',
      'on this dashboard',
      'on this panel',
      'in this panel',
      'in this dashboard',
      'this panel shows',
      'this dashboard shows',
    ];

    return dashboardQuestions.some((q) => lowerMessage.includes(q));
  }

  /**
   * Read dashboard panels and prepare them for analysis
   */
  readDashboardPanels(context: GrafanaContext): PanelData[] {
    const panels: PanelData[] = [];

    if (!context.dashboard || !context.dashboard.panels) {
      return panels;
    }

    for (const panel of context.dashboard.panels) {
      if (panel.targets && panel.targets.length > 0) {
        const target = panel.targets[0];
        const query = target.expr || target.query;

        if (query) {
          panels.push({
            panelTitle: panel.title,
            panelType: panel.type,
            query: query,
            datasourceType: target.datasource?.type,
            summary: this.generatePanelSummary(panel),
          });
        }
      }
    }

    return panels;
  }

  /**
   * Build a dashboard summary prompt for the LLM
   */
  buildDashboardSummaryPrompt(userMessage: string, context: GrafanaContext, panels: PanelData[]): string {
    let prompt = `The user is looking at a Grafana dashboard and asking: "${userMessage}"\n\n`;

    prompt += `Dashboard: "${context.dashboard?.title}"\n`;
    if (context.timeRange) {
      prompt += `Time Range: ${context.timeRange.from} to ${context.timeRange.to}\n`;
    }
    prompt += `\n`;

    if (panels.length > 0) {
      prompt += `The dashboard has ${panels.length} panels showing:\n\n`;

      panels.forEach((panel, idx) => {
        prompt += `Panel ${idx + 1}: "${panel.panelTitle}" (${panel.panelType})\n`;
        prompt += `  Purpose: ${panel.summary}\n`;
        prompt += `  Query: ${panel.query}\n`;
        if (panel.datasourceType) {
          prompt += `  Datasource: ${panel.datasourceType}\n`;
        }
        prompt += `\n`;
      });

      prompt += `Based on these panels and their queries, provide insights about what the user is seeing on this dashboard.\n`;
      prompt += `Focus on:\n`;
      prompt += `1. What metrics/data are being monitored\n`;
      prompt += `2. What patterns or issues the panels might reveal\n`;
      prompt += `3. How to interpret the dashboard effectively\n`;
      prompt += `4. Any potential issues to watch for based on these queries\n\n`;
      prompt += `Be specific and reference the actual panel queries and their purposes.`;
    } else {
      prompt += `The dashboard doesn't have any visible panels with queries.\n`;
      prompt += `Provide general guidance about what the user might see on a dashboard titled "${context.dashboard?.title}".`;
    }

    return prompt;
  }

  /**
   * Format context as a system prompt for the LLM
   */
  formatContextPrompt(context: GrafanaContext): string {
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
   */
  formatPanelDataPrompt(panelData: PanelDataAnalysis[]): string {
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

  /**
   * Calculate optimal context budget based on total max tokens
   */
  calculateContextBudget(maxTokens: number): number {
    return Math.floor(maxTokens * 0.25);
  }

  /**
   * Get current dashboard info
   */
  private async getDashboardContext(uid: string): Promise<DashboardContext | undefined> {
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
  private extractPanelContext(panel: GrafanaPanel): PanelContext {
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
  private getTimeRange(): TimeRange | undefined {
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
  private getTemplateVariables(): TemplateVariable[] | undefined {
    const location = locationService.getLocation();
    const searchParams = new URLSearchParams(location.search);
    const variables: TemplateVariable[] = [];
    const processedVars = new Set<string>();

    for (const [key] of searchParams.entries()) {
      if (key.startsWith('var-') && !processedVars.has(key)) {
        const varName = key.substring(4);
        processedVars.add(key);

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
  private getUserContext() {
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
   */
  private getAdhocFilters(): AdhocFilter[] | undefined {
    const location = locationService.getLocation();
    const searchParams = new URLSearchParams(location.search);
    const filters: AdhocFilter[] = [];

    const filterParams = searchParams.getAll('filters');

    for (const filterParam of filterParams) {
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
   * Estimate the number of tokens in a text/object
   */
  private estimateTokens(text: any): number {
    const str = typeof text === 'string' ? text : JSON.stringify(text);
    return Math.ceil(str.length / 4);
  }

  /**
   * Format full context as a string
   */
  private formatFullContext(context: GrafanaContext): string {
    const parts: string[] = [];

    if (context.dashboard) {
      parts.push(`Dashboard: ${context.dashboard.title}`);
      if (context.dashboard.uid) {
        parts.push(`UID: ${context.dashboard.uid}`);
      }
      if (context.dashboard.tags && context.dashboard.tags.length > 0) {
        parts.push(`Tags: ${context.dashboard.tags.join(', ')}`);
      }
    }

    if (context.panel) {
      parts.push(`\nPanel: ${context.panel.title}`);
      if (context.panel.description) {
        parts.push(`Description: ${context.panel.description}`);
      }
    }

    if (context.timeRange) {
      parts.push(`\nTime Range: ${context.timeRange.from} to ${context.timeRange.to}`);
    }

    return parts.join('\n');
  }

  /**
   * Generate a human-readable summary of what a panel shows
   */
  private generatePanelSummary(panel: PanelContext): string {
    const title = panel.title.toLowerCase();
    const type = panel.type;

    if (title.includes('error') && title.includes('rate')) {
      return 'Shows the rate of errors over time';
    }
    if (title.includes('request') && (title.includes('rate') || title.includes('throughput'))) {
      return 'Shows the request rate/throughput';
    }
    if (title.includes('latency') || title.includes('duration') || title.includes('response time')) {
      return 'Shows response time/latency metrics';
    }
    if (title.includes('cpu')) {
      return 'Shows CPU usage/utilization';
    }
    if (title.includes('memory') || title.includes('ram')) {
      return 'Shows memory usage';
    }
    if (title.includes('status') || title.includes('health')) {
      return 'Shows service health/status';
    }
    if (title.includes('log')) {
      return 'Shows log entries';
    }

    if (type === 'timeseries' || type === 'graph') {
      return 'Time-series metric visualization';
    }
    if (type === 'stat' || type === 'gauge') {
      return 'Current value indicator';
    }
    if (type === 'table') {
      return 'Tabular data display';
    }
    if (type === 'logs') {
      return 'Log entries';
    }

    return `${panel.type} panel`;
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
