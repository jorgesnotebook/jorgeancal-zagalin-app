import { getBackendSrv } from '@grafana/runtime';

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

function getBasePath(): string {
  return '/api/plugins/jorgeancal-zagalin-app/resources';
}

export class QueryService {
  static async query(request: QueryRequest): Promise<QueryResponse> {
    // Always use backend for queries to get:
    // - Rate limiting
    // - Query validation
    // - OTel enforcement
    // - Datasource allowlist
    // - Audit logging
    return this.queryViaBackend(request);
  }

  // Backend mode (always used for queries)
  private static async queryViaBackend(request: QueryRequest): Promise<QueryResponse> {
    try {
      const response = await getBackendSrv().post<QueryResponse>(
        `${getBasePath()}/query`,
        request
      );
      return response;
    } catch (error: any) {
      console.error('Query service error (backend):', error);
      if (error.status === 503 || error.status === 404) {
        throw new Error('Backend unavailable. Please check that the Zagalin backend plugin is running.');
      }
      throw new Error(`Query failed: ${error.message || 'Unknown error'}`);
    }
  }

  static async queryPrometheus(
    datasourceUid: string,
    expr: string,
    from: string,
    to: string
  ): Promise<QueryResponse> {
    return this.query({
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

  static async queryLoki(
    datasourceUid: string,
    query: string,
    from: string,
    to: string
  ): Promise<QueryResponse> {
    return this.query({
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

  static async queryTempo(
    datasourceUid: string,
    query: string,
    from: string,
    to: string
  ): Promise<QueryResponse> {
    return this.query({
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
}
