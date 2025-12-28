/**
 * Tool/Function definitions for Zagalin
 * These allow the LLM to invoke structured actions in Grafana
 */

export interface Tool {
  type: 'function';
  function: {
    name: string;
    description: string;
    parameters: {
      type: 'object';
      properties: Record<string, any>;
      required?: string[];
    };
  };
}

/**
 * All available tools for Zagalin
 */
export const ZAGALIN_TOOLS: Tool[] = [
  {
    type: 'function',
    function: {
      name: 'navigate_to_dashboard',
      description: 'Navigate to a specific Grafana dashboard by UID',
      parameters: {
        type: 'object',
        properties: {
          dashboardUid: {
            type: 'string',
            description: 'The unique identifier (UID) of the dashboard',
          },
          panelId: {
            type: 'number',
            description: 'Optional panel ID to focus on a specific panel',
          },
        },
        required: ['dashboardUid'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'create_promql_query',
      description: 'Generate a PromQL query for Prometheus metrics',
      parameters: {
        type: 'object',
        properties: {
          metric: {
            type: 'string',
            description: 'The metric name to query (e.g., http_requests_total)',
          },
          filters: {
            type: 'object',
            description: 'Label filters as key-value pairs (e.g., {job: "api", status: "200"})',
          },
          aggregation: {
            type: 'string',
            enum: ['sum', 'avg', 'min', 'max', 'count', 'rate'],
            description: 'Aggregation function to apply',
          },
          timeRange: {
            type: 'string',
            description: 'Time range for rate calculations (e.g., "5m", "1h")',
          },
        },
        required: ['metric'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'create_logql_query',
      description: 'Generate a LogQL query for Loki logs',
      parameters: {
        type: 'object',
        properties: {
          logStream: {
            type: 'string',
            description: 'Log stream selector (e.g., {job="varlogs"})',
          },
          filter: {
            type: 'string',
            description: 'Log line filter expression',
          },
          parser: {
            type: 'string',
            enum: ['json', 'logfmt', 'pattern', 'regexp'],
            description: 'Parser to use for log lines',
          },
        },
        required: ['logStream'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_panel_data',
      description: 'Retrieve current data from a dashboard panel',
      parameters: {
        type: 'object',
        properties: {
          dashboardUid: {
            type: 'string',
            description: 'Dashboard UID',
          },
          panelId: {
            type: 'number',
            description: 'Panel ID',
          },
        },
        required: ['dashboardUid', 'panelId'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'open_explore_view',
      description: 'Open Explore view with a pre-populated query',
      parameters: {
        type: 'object',
        properties: {
          datasource: {
            type: 'string',
            description: 'Datasource name or UID',
          },
          query: {
            type: 'string',
            description: 'The query to run',
          },
          queryType: {
            type: 'string',
            enum: ['metrics', 'logs', 'traces'],
            description: 'Type of query',
          },
        },
        required: ['datasource', 'query'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'explain_error',
      description: 'Provide a detailed explanation of an error message or status code',
      parameters: {
        type: 'object',
        properties: {
          errorMessage: {
            type: 'string',
            description: 'The error message or status code to explain',
          },
          context: {
            type: 'string',
            description: 'Additional context (e.g., which service, what operation)',
          },
        },
        required: ['errorMessage'],
      },
    },
  },
];

/**
 * Tool call handler types
 */
export interface ToolCall {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string;
  };
}

/**
 * Parse tool call arguments safely
 */
export function parseToolArguments<T = any>(args: string): T | null {
  try {
    return JSON.parse(args);
  } catch (err) {
    console.error('Failed to parse tool arguments:', err);
    return null;
  }
}

/**
 * Handle tool call execution
 */
export async function executeToolCall(toolCall: ToolCall): Promise<any> {
  const args = parseToolArguments(toolCall.function.arguments);
  if (!args) {
    return { error: 'Invalid tool arguments' };
  }

  // Check for backend validation errors
  if (args._validation_error) {
    console.error('[zagalinTools] Validation failed:', {
      tool: toolCall.function.name,
      error: args._validation_error,
      type: args._validation_type,
    });

    return {
      error: `Query validation failed: ${args._validation_error}`,
      validationType: args._validation_type,
    };
  }

  // Use sanitized query if provided
  if (args._sanitized_query) {
    console.info('[zagalinTools] Using sanitized query:', args._sanitized_query);
  }

  switch (toolCall.function.name) {
    case 'navigate_to_dashboard':
      return navigateToDashboard(args.dashboardUid, args.panelId);

    case 'create_promql_query':
      return createPromQLQuery(args);

    case 'create_logql_query':
      return createLogQLQuery(args);

    case 'get_panel_data':
      return { message: 'Panel data retrieval not yet implemented', args };

    case 'open_explore_view':
      return openExploreView(args);

    case 'explain_error':
      return { message: 'Error explanation delegated to LLM', args };

    default:
      return { error: `Unknown tool: ${toolCall.function.name}` };
  }
}

// Tool implementation functions

const DASHBOARD_UID_REGEX = /^[a-zA-Z0-9_-]{1,40}$/;

function navigateToDashboard(uid: string, panelId?: number): any {
  // Validate dashboard UID format
  if (!DASHBOARD_UID_REGEX.test(uid)) {
    console.error('Invalid dashboard UID format:', uid);
    return { success: false, error: 'Invalid dashboard UID format' };
  }

  // Validate panelId is a positive integer if provided
  if (panelId !== undefined) {
    if (!Number.isInteger(panelId) || panelId < 0) {
      console.error('Invalid panel ID:', panelId);
      return { success: false, error: 'Invalid panel ID' };
    }
  }

  let url = `/d/${encodeURIComponent(uid)}`;
  if (panelId !== undefined) {
    url += `?viewPanel=${panelId}`;
  }
  window.location.assign(url);
  return { success: true, url };
}

function createPromQLQuery(params: any): any {
  // Use sanitized query if backend provided it
  if (params._sanitized_query) {
    return {
      query: params._sanitized_query,
      description: 'PromQL query (sanitized)',
      sanitized: true,
    };
  }

  const { metric, filters, aggregation, timeRange } = params;

  let query = metric;

  // Add filters
  if (filters && Object.keys(filters).length > 0) {
    const filterStr = Object.entries(filters)
      .map(([k, v]) => `${k}="${v}"`)
      .join(', ');
    query = `${metric}{${filterStr}}`;
  }

  // Add aggregation
  if (aggregation) {
    if (aggregation === 'rate') {
      query = `rate(${query}[${timeRange || '5m'}])`;
    } else {
      query = `${aggregation}(${query})`;
    }
  }

  return { query, description: `PromQL query generated` };
}

function createLogQLQuery(params: any): any {
  // Use sanitized query if backend provided it
  if (params._sanitized_query) {
    return {
      query: params._sanitized_query,
      description: 'LogQL query (sanitized)',
      sanitized: true,
    };
  }

  const { logStream, filter, parser } = params;

  let query = logStream;

  if (filter) {
    query += ` |= "${filter}"`;
  }

  if (parser) {
    query += ` | ${parser}`;
  }

  return { query, description: `LogQL query generated` };
}

function openExploreView(params: any): any {
  const { datasource, query } = params;

  // Validate datasource is a non-empty string
  if (!datasource || typeof datasource !== 'string' || datasource.length > 100) {
    console.error('Invalid datasource:', datasource);
    return { success: false, error: 'Invalid datasource' };
  }

  // Validate query
  if (!query || typeof query !== 'string' || query.length > 10000) {
    console.error('Invalid query:', query);
    return { success: false, error: 'Invalid query' };
  }

  // Use a safer URL construction
  const exploreState = {
    datasource: datasource,
    queries: [{ expr: query }]
  };

  try {
    const url = `/explore?left=${encodeURIComponent(JSON.stringify(exploreState))}`;
    window.location.assign(url);
    return { success: true, url };
  } catch (error) {
    console.error('Failed to construct explore URL:', error);
    return { success: false, error: 'Failed to construct URL' };
  }
}
