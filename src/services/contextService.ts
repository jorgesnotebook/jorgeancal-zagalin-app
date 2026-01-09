/**
 * Context service for extracting Grafana context
 * This service provides context about the current dashboard, panel, time range, etc.
 */

import { config, getBackendSrv, locationService } from '@grafana/runtime';
import type { GrafanaContext, DashboardContext, PanelContext, TimeRange, TemplateVariable } from './contextTypes';

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
      context.timeRange = this.getTimeRange();
    }

    const panelMatch = location.search.match(/[?&]viewPanel=(\d+)/);
    const panelId = panelMatch ? parseInt(panelMatch[1], 10) : undefined;

    if (panelId && context.dashboard) {
      context.panel = context.dashboard.panels?.find(p => p.id === panelId);
    }

    context.user = this.getUserContext();

    return context;
  }

  /**
   * Get dashboard context by UID
   */
  private static async getDashboardContext(uid: string): Promise<DashboardContext | undefined> {
    try {
      const response = await getBackendSrv().get(`/api/dashboards/uid/${uid}`);
      const dashboard = response.dashboard;

      const context: DashboardContext = {
        uid: dashboard.uid,
        title: dashboard.title,
        tags: dashboard.tags,
        timezone: dashboard.timezone,
        panels: dashboard.panels?.map((panel: any) => this.extractPanelContext(panel)) || [],
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
  private static extractPanelContext(panel: any): PanelContext {
    return {
      id: panel.id,
      title: panel.title,
      type: panel.type,
      description: panel.description,
      targets: panel.targets?.map((target: any) => ({
        refId: target.refId,
        datasource: target.datasource,
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

    for (const [key, value] of searchParams.entries()) {
      if (key.startsWith('var-')) {
        const varName = key.substring(4);
        variables.push({
          name: varName,
          current: {
            value,
            text: value,
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
        context.panel.targets.forEach(target => {
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
      context.templateVariables.forEach(v => {
        parts.push(`  $${v.name} = ${v.current.value}`);
      });
    }

    if (parts.length === 0) {
      return '';
    }

    return `# Current Grafana Context\n\n${parts.join('\n')}\n\nUse this context to provide specific, relevant answers about this dashboard and panel.`;
  }
}
