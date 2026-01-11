import { getBackendSrv } from '@grafana/runtime';
import { getPluginResourcePath } from './pluginUrl';

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

export interface TimeRange {
  from: string;
  to: string;
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

export async function executeQuery(request: QueryRequest): Promise<QueryResponse> {
  try {
    return await getBackendSrv().post<QueryResponse>(`${getPluginResourcePath()}/query`, request);
  } catch (error: any) {
    console.error('Query service error:', error);
    if (error.status === 503 || error.status === 404) {
      throw new Error('Backend unavailable. Please check that the Zagalin backend plugin is running.');
    }
    throw new Error(`Query failed: ${error.message || 'Unknown error'}`);
  }
}

export async function queryPrometheus(
  datasourceUid: string,
  expr: string,
  from: string,
  to: string
): Promise<QueryResponse> {
  return executeQuery({
    datasource: datasourceUid,
    queries: [
      {
        refId: 'A',
        datasourceUid,
        queryType: 'prometheus',
        expr,
      },
    ],
    timeRange: { from, to },
  });
}

export async function queryLoki(
  datasourceUid: string,
  query: string,
  from: string,
  to: string
): Promise<QueryResponse> {
  return executeQuery({
    datasource: datasourceUid,
    queries: [
      {
        refId: 'A',
        datasourceUid,
        queryType: 'loki',
        query,
      },
    ],
    timeRange: { from, to },
  });
}

export async function queryTempo(
  datasourceUid: string,
  query: string,
  from: string,
  to: string
): Promise<QueryResponse> {
  return executeQuery({
    datasource: datasourceUid,
    queries: [
      {
        refId: 'A',
        datasourceUid,
        queryType: 'tempo',
        query,
      },
    ],
    timeRange: { from, to },
  });
}
