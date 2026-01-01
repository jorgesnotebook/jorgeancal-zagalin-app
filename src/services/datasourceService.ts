import { getBackendSrv } from '@grafana/runtime';
import { getPluginApiUrl } from './pluginUrl';

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

export async function listDatasources(): Promise<DatasourceListResponse> {
  try {
    return await getBackendSrv().get<DatasourceListResponse>(
      getPluginApiUrl('/datasources')
    );
  } catch (error: any) {
    console.error('Failed to fetch datasources:', error);
    return {
      datasources: [],
      allowedDatasources: [],
      defaultDatasource: '',
    };
  }
}
