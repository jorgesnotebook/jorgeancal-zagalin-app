import { vector } from '@grafana/llm';

export interface DashboardPayload {
  uid: string;
  title: string;
  description: string;
  tags: string[];
}

export interface QueryPayload {
  query: string;
  datasource: string;
  dashboard: string;
  frequency: number;
}

export interface PanelPayload {
  id: number;
  title: string;
  description: string;
  type: string;
  dashboardUid: string;
}

/**
 * Vector search service for semantic search across Grafana content
 */
export class VectorSearchService {
  /**
   * Search for similar dashboards
   */
  async searchSimilarDashboards(query: string, limit = 5): Promise<DashboardPayload[]> {
    try {
      const enabled = await vector.enabled();
      if (!enabled) {
        console.log('Zagalin: Vector search not enabled');
        return [];
      }

      const results = await vector.search<DashboardPayload>({
        collection: 'dashboards',
        query,
        topK: limit,
      });

      return results
        .filter((r) => r.score > 0.7)
        .map((r) => r.payload);
    } catch (err) {
      console.error('Zagalin: Dashboard search failed:', err);
      return [];
    }
  }

  /**
   * Search for similar queries
   */
  async searchSimilarQueries(query: string, limit = 3): Promise<QueryPayload[]> {
    try {
      const enabled = await vector.enabled();
      if (!enabled) {
        return [];
      }

      const results = await vector.search<QueryPayload>({
        collection: 'queries',
        query,
        topK: limit,
      });

      return results
        .filter((r) => r.score > 0.8)
        .map((r) => r.payload);
    } catch (err) {
      console.error('Zagalin: Query search failed:', err);
      return [];
    }
  }

  /**
   * Search for similar panels
   */
  async searchSimilarPanels(query: string, limit = 5): Promise<PanelPayload[]> {
    try {
      const enabled = await vector.enabled();
      if (!enabled) {
        return [];
      }

      const results = await vector.search<PanelPayload>({
        collection: 'panels',
        query,
        topK: limit,
      });

      return results
        .filter((r) => r.score > 0.75)
        .map((r) => r.payload);
    } catch (err) {
      console.error('Zagalin: Panel search failed:', err);
      return [];
    }
  }

  /**
   * Enhance user query with semantic context
   */
  async enhanceQueryWithContext(userQuery: string): Promise<string> {
    const [dashboards, queries] = await Promise.all([
      this.searchSimilarDashboards(userQuery, 3),
      this.searchSimilarQueries(userQuery, 2),
    ]);

    const contextParts: string[] = [];

    if (dashboards.length > 0) {
      contextParts.push('Similar dashboards:');
      dashboards.forEach((d) => {
        contextParts.push(`- ${d.title}: ${d.description || 'No description'}`);
      });
    }

    if (queries.length > 0) {
      contextParts.push('\nFrequently used queries:');
      queries.forEach((q) => {
        contextParts.push(`- ${q.query} (${q.datasource})`);
      });
    }

    return contextParts.length > 0 ? '\n\n' + contextParts.join('\n') : '';
  }
}
