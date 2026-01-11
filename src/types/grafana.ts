export interface GrafanaDashboard {
  uid: string;
  title: string;
  tags?: string[];
  timezone?: string;
  panels?: GrafanaPanel[];
}

export interface GrafanaPanel {
  id: number;
  title: string;
  type: string;
  description?: string;
  targets?: GrafanaTarget[];
  fieldConfig?: GrafanaFieldConfig;
  transformations?: unknown[];
}

export interface GrafanaTarget {
  refId: string;
  datasource?: {
    type?: string;
    uid?: string;
  };
  expr?: string;
  query?: string;
  queryType?: string;
}

export interface GrafanaFieldConfig {
  defaults?: {
    unit?: string;
    decimals?: number;
    min?: number;
    max?: number;
    color?: {
      mode?: string;
    };
  };
  overrides?: unknown[];
}

export interface GrafanaDashboardResponse {
  dashboard: GrafanaDashboard;
  meta?: {
    isStarred?: boolean;
    slug?: string;
    url?: string;
    version?: number;
  };
}
