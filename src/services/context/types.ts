/**
 * Type definitions for Context Management module
 * Consolidated from contextTypes.ts
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

export interface AdhocFilter {
  key: string;
  operator: string;
  value: string;
}

export interface PanelQuery {
  refId: string;
  datasource?: {
    type: string;
    uid: string;
  };
  expr?: string;
  query?: string;
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

export interface UserContext {
  id: number;
  login: string;
  email: string;
  orgId: number;
}

export interface GrafanaContext {
  dashboard?: DashboardContext;
  panel?: PanelContext;
  timeRange?: TimeRange;
  templateVariables?: TemplateVariable[];
  adhocFilters?: AdhocFilter[];
  user?: UserContext;
  dataSourceTypes?: string[];
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
  trigger: string;
  examples: string[];
}

/**
 * Optimized context structure
 */
export interface OptimizedContext {
  essential: string;
  supplemental: string;
  metadata: Record<string, any>;
}

/**
 * Panel data for dashboard reading
 */
export interface PanelData {
  panelTitle: string;
  panelType: string;
  query: string;
  datasourceType?: string;
  error?: string;
  summary?: string;
}
