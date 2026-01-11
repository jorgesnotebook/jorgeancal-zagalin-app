/**
 * Data Extractor Types
 *
 * Type definitions for query execution and data analysis.
 */

import type { TimeRange } from '../contextTypes';

// Query types
export interface QueryRequest {
  datasource: string;
  queries: QueryPayload[];
  timeRange: TimeRange;
}

export interface QueryPayload {
  refId: string;
  datasourceUid?: string;
  queryType?: string;
  expr?: string;
  query?: string;
  intervalMs?: number;
  maxDataPoints?: number;
  format?: string;
  [key: string]: any;
}

export interface QueryResponse {
  results: Record<string, QueryResult>;
}

export interface QueryResult {
  refId: string;
  frames: any[];
  error?: string;
  meta?: Record<string, any>;
}

// Panel data analysis types
export interface PanelDataAnalysis {
  panelId?: number;
  panelTitle: string;
  panelType: string;
  datasourceUid: string;
  datasourceType: string;
  query: string;

  // Actual data
  success: boolean;
  error?: string;

  // Current state
  currentValue?: number;
  unit?: string;

  // Trend analysis
  trend?: 'increasing' | 'decreasing' | 'stable' | 'spiky' | 'unknown';
  changePercent?: number;

  // Anomaly detection
  isSaturated?: boolean;
  hasSpike?: boolean;
  hasDrop?: boolean;
  hasNoData?: boolean;

  // Stats
  min?: number;
  max?: number;
  avg?: number;

  // Raw summary
  summary: string;
}

// Log analysis types
export interface LogAnalysisResult {
  success: boolean;
  error?: string;

  // Query metadata
  query: string;
  datasourceUid: string;
  timeRange: TimeRange;
  labels?: Record<string, string>;

  // Analysis results
  totalCount: number;
  errorCount: number;
  warnCount: number;
  logLevels: Record<string, number>;

  // Trends
  trend: 'increasing' | 'decreasing' | 'stable' | 'unknown';
  changePercent?: number;

  // Top messages
  topErrorMessages: string[];
  topMessages: string[];

  // Label analysis
  topLabels: Record<string, string[]>;

  // Summary
  summary: string;
}

// Datasource types
export interface DatasourceInfo {
  uid: string;
  name: string;
  type: string;
}

export interface DatasourceListResponse {
  datasources: DatasourceInfo[];
  allowedDatasources: string[];
  defaultDatasource: string;
}
