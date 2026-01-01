import { getBackendSrv } from '@grafana/runtime';

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

export class DatasourceService {
  static async listDatasources(): Promise<DatasourceListResponse> {
    // Always use backend to get:
    // - Datasource allowlist enforcement
    // - Default datasource configuration
    // - Consistent filtering across organization
    return this.listViaBackend();
  }

  // Backend mode (always used for datasource listing)
  private static async listViaBackend(): Promise<DatasourceListResponse> {
    try {
      const response = await getBackendSrv().get<DatasourceListResponse>(
        '/api/plugins/jorgeancal-zagalin-app/resources/datasources'
      );
      return response;
    } catch (error: any) {
      console.error('Failed to fetch datasources (backend):', error);
      // Return empty response instead of throwing
      return {
        datasources: [],
        allowedDatasources: [],
        defaultDatasource: '',
      };
    }
  }
}
