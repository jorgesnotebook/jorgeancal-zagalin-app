/**
 * Grafana context types for the AI assistant
 * These types represent the context we extract from Grafana to make AI responses more relevant
 */

export interface TimeRange {
  from: string;
  to: string;
  raw?: {
    from: string;
    to: string;
  };
}

export interface TemplateVariable {
  name: string;
  current: {
    value: string | string[];
    text: string | string[];
  };
}

export interface PanelQuery {
  refId: string;
  datasource?: {
    type: string;
    uid: string;
  };
  expr?: string; // PromQL
  query?: string; // LogQL or other
  queryType?: string;
}

export interface PanelContext {
  id: number;
  title: string;
  type: string;
  description?: string;
  targets: PanelQuery[];
  fieldConfig?: {
    defaults?: {
      unit?: string;
      decimals?: number;
    };
  };
  transformations?: any[];
}

export interface DashboardContext {
  uid: string;
  title: string;
  tags?: string[];
  timezone?: string;
  panels?: PanelContext[];
}

export interface GrafanaContext {
  dashboard?: DashboardContext;
  panel?: PanelContext;
  timeRange?: TimeRange;
  templateVariables?: TemplateVariable[];
  user?: {
    id: number;
    login: string;
    email: string;
    orgId: number;
  };
  dataSourceTypes?: string[]; // Available datasource types
}

/**
 * Structured actions that the AI can return for the UI to render as buttons
 */
export interface AssistantAction {
  type: 'explore' | 'copy' | 'dashboard' | 'panel' | 'query';
  label: string;
  data: {
    query?: string;
    datasourceUid?: string;
    dashboardUid?: string;
    panelId?: number;
    timeRange?: TimeRange;
  };
}

export interface AssistantSkill {
  name: string;
  description: string;
  trigger: string; // e.g., "explain panel", "generate query"
  examples: string[];
}
