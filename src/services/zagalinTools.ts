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
      name: 'create_traceql_query',
      description:
        'Generate a TraceQL query for Tempo traces. Use this to search traces by attributes, duration, or status.',
      parameters: {
        type: 'object',
        properties: {
          traceSelector: {
            type: 'string',
            description: 'Trace selector expression (e.g., {service.name="api-gateway"})',
          },
          filters: {
            type: 'object',
            description: 'Additional attribute filters (e.g., {status: "error", "http.status_code": "500"})',
          },
          duration: {
            type: 'string',
            description: 'Duration filter (e.g., "> 1s", "< 100ms")',
          },
        },
        required: ['traceSelector'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_trace_by_id',
      description:
        'Fetch and analyze a specific trace by its trace ID. Use this when the user provides a trace ID. Returns trace structure with spans, services, durations, and errors.',
      parameters: {
        type: 'object',
        properties: {
          traceId: {
            type: 'string',
            description: 'The trace ID to fetch (e.g., "abc123def456")',
          },
          datasource: {
            type: 'string',
            description: 'Tempo datasource UID or name',
          },
        },
        required: ['traceId', 'datasource'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_logs',
      description:
        'Fetch and analyze logs from a Loki datasource. Use this when the user asks about logs they are viewing or when you need to investigate log patterns, errors, or volume. Returns log analysis with trends, error rates, and notable messages.',
      parameters: {
        type: 'object',
        properties: {
          panelId: {
            type: 'number',
            description: 'Panel ID to analyze (if user is viewing a specific log panel)',
          },
          query: {
            type: 'string',
            description: 'LogQL query to execute (e.g., "{namespace=\\"production\\"} |= \\"error\\"")',
          },
          datasource: {
            type: 'string',
            description: 'Loki datasource UID or name',
          },
          maxLines: {
            type: 'number',
            description: 'Maximum number of log lines to fetch (default: 1000, max: 5000)',
          },
        },
        required: [],
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

// ---------------------------------------------------------------------------
// Tool permission gating
// ---------------------------------------------------------------------------

/**
 * Per-tool in-progress status messages shown in the chat UI while a tool is
 * executing. Displayed immediately when a tool call is detected so users see
 * feedback rather than a blank wait.
 */
export const TOOL_STATUS_MESSAGES: Record<string, string> = {
  execute_promql: 'Querying Prometheus…',
  execute_logql: 'Fetching logs from Loki…',
  execute_traceql: 'Searching traces in Tempo…',
  get_trace_by_id: 'Fetching trace details…',
  get_logs: 'Analysing logs…',
  get_firing_alerts: 'Checking firing alerts…',
  search_dashboards: 'Searching dashboards…',
  get_dashboard: 'Loading dashboard details…',
  get_annotations: 'Fetching annotations…',
  list_folders: 'Listing folders…',
  navigate_to_dashboard: 'Navigating to dashboard…',
  open_explore_view: 'Opening Explore view…',
  create_promql_query: 'Generating PromQL query…',
  create_logql_query: 'Generating LogQL query…',
  create_traceql_query: 'Generating TraceQL query…',
};

/**
 * Tools that execute queries against live production systems.
 * When permission gating is enabled, the user must approve before these run.
 */
export const TOOLS_REQUIRING_PERMISSION = new Set<string>([
  'execute_promql',
  'execute_logql',
  'execute_traceql',
  'get_logs',
  'get_trace_by_id',
]);

/**
 * Callback invoked when a sensitive tool needs user approval.
 * Should return true to allow execution, false to deny.
 */
export type PermissionChecker = (toolName: string, permissionMessage: string) => Promise<boolean>;

// ---------------------------------------------------------------------------
// Tool output sanitization
// ---------------------------------------------------------------------------

/** Maximum serialized size (in characters) allowed for a single tool result. */
export const MAX_TOOL_OUTPUT_CHARS = 20_000;

/**
 * Structural delimiters used by various LLM providers to separate conversation
 * roles. A datasource result that contains these tokens could be interpreted as
 * role-switching instructions when embedded in the conversation history.
 */
const INJECTION_DELIMITERS: string[] = [
  '<|im_start|>',
  '<|im_end|>',
  '<|system|>',
  '<|user|>',
  '<|assistant|>',
  '[INST]',
  '[/INST]',
  '<<SYS>>',
  '<</SYS>>',
  '<system>',
  '</system>',
];

// Build a single case-insensitive regex from all delimiters.
const INJECTION_DELIMITER_RE = new RegExp(
  INJECTION_DELIMITERS.map((d) => d.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|'),
  'gi'
);

/**
 * Recursively strip injection delimiters from all string values in an unknown
 * value (object, array, or primitive).
 */
function sanitizeValue(value: unknown): unknown {
  if (typeof value === 'string') {
    return value.replace(INJECTION_DELIMITER_RE, '');
  }
  if (Array.isArray(value)) {
    return value.map(sanitizeValue);
  }
  if (value !== null && typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      out[k] = sanitizeValue(v);
    }
    return out;
  }
  return value;
}

/**
 * Sanitize a tool result before it is serialised into the conversation history:
 *  1. Strip LLM prompt-injection delimiters from all string fields.
 *  2. Truncate the serialised output to MAX_TOOL_OUTPUT_CHARS with a clear
 *     notice so the LLM knows the data was cut.
 *
 * Exported for unit testing.
 */
export function sanitizeToolOutput(toolName: string, result: unknown): unknown {
  const sanitized = sanitizeValue(result);

  const serialized = JSON.stringify(sanitized);
  if (serialized.length > MAX_TOOL_OUTPUT_CHARS) {
    console.warn(
      `[zagalinTools] Output for "${toolName}" truncated: ${serialized.length} → ${MAX_TOOL_OUTPUT_CHARS} chars`
    );
    return {
      _truncated: true,
      _originalSize: serialized.length,
      _notice: `Result truncated to ${MAX_TOOL_OUTPUT_CHARS} characters. Summarise from the partial data below.`,
      data: serialized.slice(0, MAX_TOOL_OUTPUT_CHARS),
    };
  }

  return sanitized;
}

/**
 * Handle tool call execution.
 *
 * @param toolCall          The tool call to execute.
 * @param permissionChecker Optional callback for sensitive tools. When provided,
 *                          tools in TOOLS_REQUIRING_PERMISSION pause and await
 *                          approval before executing. Returning false cancels
 *                          execution with an "execution denied" result.
 * @param permissionMessage Custom message shown to the user during approval.
 *                          Defaults to a generic prompt when omitted.
 */
export async function executeToolCall(
  toolCall: ToolCall,
  permissionChecker?: PermissionChecker,
  permissionMessage?: string
): Promise<any> {
  const args = parseToolArguments(toolCall.function.arguments);
  if (!args) {
    return { error: 'Invalid tool arguments' };
  }

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

  if (args._sanitized_query) {
    console.info('[zagalinTools] Using sanitized query:', args._sanitized_query);
  }

  // Gate sensitive tools behind user approval when a checker is provided.
  if (permissionChecker && TOOLS_REQUIRING_PERMISSION.has(toolCall.function.name)) {
    const message =
      permissionMessage ||
      `Allow tool "${toolCall.function.name}" to execute a live query against a production datasource?`;
    const allowed = await permissionChecker(toolCall.function.name, message);
    if (!allowed) {
      console.info(`[zagalinTools] Execution denied by user: ${toolCall.function.name}`);
      return {
        error: 'Execution denied by user',
        toolName: toolCall.function.name,
      };
    }
  }

  const rawResult = await dispatchToolCall(toolCall.function.name, args);
  return sanitizeToolOutput(toolCall.function.name, rawResult);
}

async function dispatchToolCall(name: string, args: any): Promise<any> {
  switch (name) {
    case 'navigate_to_dashboard':
      return navigateToDashboard(args.dashboardUid, args.panelId);

    case 'create_promql_query':
      return createPromQLQuery(args);

    case 'create_logql_query':
      return createLogQLQuery(args);

    case 'create_traceql_query':
      return createTraceQLQuery(args);

    case 'get_trace_by_id':
      return getTraceById(args);

    case 'get_logs':
      return getLogs(args);

    case 'open_explore_view':
      return openExploreView(args);

    case 'execute_promql':
      return executePromQL(args);

    case 'execute_logql':
      return executeLogQL(args);

    case 'execute_traceql':
      return executeTraceQL(args);

    case 'get_firing_alerts':
      return getFiringAlerts(args);

    case 'search_dashboards':
      return searchDashboards(args);

    case 'get_dashboard':
      return getDashboard(args);

    case 'get_annotations':
      return getAnnotations(args);

    case 'list_folders':
      return listFolders(args);

    default:
      return { error: `Unknown tool: ${name}` };
  }
}

const DASHBOARD_UID_REGEX = /^[a-zA-Z0-9_-]{1,40}$/;

interface NavigationResult {
  success: boolean;
  url?: string;
  error?: string;
}

interface QueryResult {
  query: string;
  description: string;
  sanitized?: boolean;
}

interface PromQLParams {
  _sanitized_query?: string;
  metric?: string;
  filters?: Record<string, string>;
  aggregation?: string;
  timeRange?: string;
}

interface LogQLParams {
  _sanitized_query?: string;
  logStream?: string;
  filter?: string;
  parser?: string;
}

interface TraceQLParams {
  _sanitized_query?: string;
  traceSelector?: string;
  filters?: Record<string, string>;
  duration?: string;
}

interface GetTraceParams {
  traceId: string;
  datasource: string;
}

interface ExploreParams {
  datasource: string;
  query: string;
}

interface GetLogsParams {
  panelId?: number;
  query?: string;
  datasource?: string;
  maxLines?: number;
}

function navigateToDashboard(uid: string, panelId?: number): NavigationResult {
  if (!DASHBOARD_UID_REGEX.test(uid)) {
    console.error('Invalid dashboard UID format:', uid);
    return { success: false, error: 'Invalid dashboard UID format' };
  }

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

function createPromQLQuery(params: PromQLParams): QueryResult {
  if (params._sanitized_query) {
    return {
      query: params._sanitized_query,
      description: 'PromQL query (sanitized)',
      sanitized: true,
    };
  }

  const { metric, filters, aggregation, timeRange } = params;

  let query = metric || '';

  if (filters && Object.keys(filters).length > 0) {
    const filterStr = Object.entries(filters)
      .map(([k, v]) => `${k}="${v}"`)
      .join(', ');
    query = `${metric || ''}{${filterStr}}`;
  }

  if (aggregation) {
    if (aggregation === 'rate') {
      query = `rate(${query}[${timeRange || '5m'}])`;
    } else {
      query = `${aggregation}(${query})`;
    }
  }

  return { query, description: `PromQL query generated` };
}

function createLogQLQuery(params: LogQLParams): QueryResult {
  if (params._sanitized_query) {
    return {
      query: params._sanitized_query,
      description: 'LogQL query (sanitized)',
      sanitized: true,
    };
  }

  const { logStream, filter, parser } = params;

  let query = logStream || '';

  if (filter) {
    query += ` |= "${filter}"`;
  }

  if (parser) {
    query += ` | ${parser}`;
  }

  return { query, description: `LogQL query generated` };
}

function createTraceQLQuery(params: TraceQLParams): QueryResult {
  if (params._sanitized_query) {
    return {
      query: params._sanitized_query,
      description: 'TraceQL query (sanitized)',
      sanitized: true,
    };
  }

  const { traceSelector, filters, duration } = params;

  let query = traceSelector || '{}';

  // Add attribute filters
  if (filters && Object.keys(filters).length > 0) {
    const filterParts = Object.entries(filters)
      .map(([key, value]) => `${key}="${value}"`)
      .join(' && ');

    // Combine with trace selector
    if (query === '{}') {
      query = `{${filterParts}}`;
    } else {
      // Remove closing brace, add filters, re-add closing brace
      query = query.slice(0, -1) + ` && ${filterParts}}`;
    }
  }

  // Add duration filter
  if (duration) {
    query += ` && duration ${duration}`;
  }

  return { query, description: `TraceQL query generated` };
}

async function getTraceById(params: GetTraceParams): Promise<any> {
  const { traceId, datasource } = params;

  if (!traceId || !datasource) {
    return { error: 'Missing required parameters: traceId and datasource' };
  }

  // Validate trace ID format (16-32 hex characters)
  const traceIdPattern = /^[0-9a-f]{16,32}$/i;
  if (!traceIdPattern.test(traceId)) {
    return {
      success: false,
      error: `Invalid trace ID format: ${traceId}. Expected 16-32 hexadecimal characters.`,
      traceId,
      datasource,
    };
  }

  try {
    // Import query service
    const { queryTempo } = await import('./queryService');

    // Query for the specific trace ID
    // TraceQL query to fetch trace by ID
    const query = `{traceID="${traceId}"}`;

    // Use a reasonable time range (last 24 hours)
    const now = Date.now();
    const from = new Date(now - 24 * 60 * 60 * 1000).toISOString();
    const to = new Date(now).toISOString();

    const response = await queryTempo(datasource, query, from, to);

    // Extract trace structure
    if (response.results && response.results.A && response.results.A.frames) {
      const frames = response.results.A.frames;

      // Analyze trace structure
      const analysis = analyzeTraceFrames(frames, traceId);

      return {
        success: true,
        traceId,
        datasource,
        ...analysis,
      };
    } else {
      return {
        success: false,
        error: 'Trace not found or no data returned',
        traceId,
        datasource,
      };
    }
  } catch (error: any) {
    console.error('Failed to fetch trace:', error);
    return {
      success: false,
      error: `Failed to fetch trace: ${error.message || 'Unknown error'}`,
      traceId,
      datasource,
    };
  }
}

/**
 * Get and analyze logs from Loki
 */
async function getLogs(params: GetLogsParams): Promise<any> {
  try {
    // Import services
    const { ContextService } = await import('./contextService');
    const { analyzeLogs } = await import('./logAnalysisService');

    // Get context to find panel if panelId provided
    const context = await ContextService.getContext();

    // Find panel to analyze
    let panel = null;
    if (params.panelId && context.dashboard) {
      panel = context.dashboard.panels?.find((p) => p.id === params.panelId);
    } else if (context.panel) {
      // Use current focused panel if no panelId specified
      panel = context.panel;
    }

    // Get time range
    const timeRange = context.timeRange || {
      from: 'now-15m',
      to: 'now',
    };

    // If panel found and it's a log panel, analyze it
    if (panel && panel.type === 'logs') {
      const maxLogLines = Math.min(params.maxLines || 1000, 5000); // Cap at 5000
      const analysis = await analyzeLogs(panel, timeRange, maxLogLines);

      return {
        ...analysis,
      };
    }

    // If no panel but query and datasource provided, construct synthetic panel
    if (params.query && params.datasource) {
      const syntheticPanel = {
        id: 0,
        title: 'Custom Log Query',
        type: 'logs' as const,
        targets: [
          {
            refId: 'A',
            expr: params.query,
            datasource: {
              type: 'loki',
              uid: params.datasource,
            },
          },
        ],
      };

      const maxLogLines = Math.min(params.maxLines || 1000, 5000);
      const analysis = await analyzeLogs(syntheticPanel, timeRange, maxLogLines);

      return {
        ...analysis,
      };
    }

    // No valid log source found
    return {
      success: false,
      error: 'No log panel found and no query/datasource provided',
    };
  } catch (error: any) {
    console.error('Failed to analyze logs:', error);
    return {
      success: false,
      error: `Failed to analyze logs: ${error.message || 'Unknown error'}`,
    };
  }
}

function analyzeTraceFrames(frames: any[], traceId: string): any {
  // Extract span data from frames
  const spans: any[] = [];
  let rootSpan: any = null;
  const services = new Set<string>();
  const operations = new Set<string>();
  const errors: any[] = [];

  for (const frame of frames) {
    if (!frame.fields) {
      continue;
    }

    // Extract span information from frame fields
    const spanCount = frame.fields[0]?.values?.length || 0;

    for (let i = 0; i < spanCount; i++) {
      const span: any = {};

      for (const field of frame.fields) {
        const fieldName = field.name;
        const value = field.values[i];

        if (fieldName === 'spanID') {
          span.spanId = value;
        }
        if (fieldName === 'operationName') {
          span.operation = value;
        }
        if (fieldName === 'serviceName') {
          span.service = value;
          services.add(value);
        }
        if (fieldName === 'duration') {
          span.duration = value;
        }
        if (fieldName === 'startTime') {
          span.startTime = value;
        }
        if (fieldName === 'parentSpanID') {
          span.parentSpanId = value;
        }
        if (fieldName === 'statusCode' && value !== 0) {
          span.status = value;
          errors.push(span);
        }
      }

      spans.push(span);

      if (span.operation) {
        operations.add(span.operation);
      }

      // Identify root span (no parent)
      if (!span.parentSpanId || span.parentSpanId === '') {
        rootSpan = span;
      }
    }
  }

  // Find slowest spans
  const slowestSpans = [...spans]
    .sort((a, b) => (b.duration || 0) - (a.duration || 0))
    .slice(0, 5)
    .map((s) => ({
      service: s.service,
      operation: s.operation,
      duration: `${(s.duration / 1000).toFixed(2)}ms`,
    }));

  // Calculate total duration
  const totalDuration = rootSpan?.duration || 0;

  return {
    traceStructure: {
      totalSpans: spans.length,
      services: Array.from(services),
      operations: Array.from(operations),
      rootService: rootSpan?.service,
      rootOperation: rootSpan?.operation,
      totalDuration: `${(totalDuration / 1000).toFixed(2)}ms`,
      errorCount: errors.length,
    },
    slowestSpans,
    errors:
      errors.length > 0
        ? errors.slice(0, 5).map((e) => ({
            service: e.service,
            operation: e.operation,
            status: e.status,
          }))
        : [],
    summary:
      `Trace has ${spans.length} spans across ${services.size} services. ` +
      `Root service: ${rootSpan?.service || 'unknown'}. ` +
      `Total duration: ${(totalDuration / 1000).toFixed(2)}ms. ` +
      `${errors.length > 0 ? `Found ${errors.length} error spans.` : 'No errors detected.'}`,
  };
}

function openExploreView(params: ExploreParams): NavigationResult {
  const { datasource, query } = params;

  if (!datasource || typeof datasource !== 'string' || datasource.length > 100) {
    console.error('Invalid datasource:', datasource);
    return { success: false, error: 'Invalid datasource' };
  }

  if (!query || typeof query !== 'string' || query.length > 10000) {
    console.error('Invalid query:', query);
    return { success: false, error: 'Invalid query' };
  }

  const exploreState = {
    datasource: datasource,
    queries: [{ expr: query }],
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

/**
 * Execute PromQL query and return structured analytics
 */
async function executePromQL(params: any): Promise<any> {
  const { datasourceUid, query, from, to, step } = params;

  if (!datasourceUid || !query) {
    return {
      success: false,
      error: 'Missing required parameters: datasourceUid and query',
    };
  }

  try {
    const { getBackendSrv } = await import('@grafana/runtime');

    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/execute_promql',
      {
        datasourceUid,
        query,
        from: from || 'now-15m',
        to: to || 'now',
        step: step || undefined,
        serviceName: params.serviceName,
        environmentName: params.environmentName,
      }
    );

    return {
      success: true,
      ...response,
    };
  } catch (error: any) {
    console.error('Failed to execute PromQL query:', error);
    return {
      success: false,
      error: `Failed to execute PromQL query: ${error.message || 'Unknown error'}`,
      query,
      datasourceUid,
    };
  }
}

/**
 * Execute LogQL query and return log analysis
 */
async function executeLogQL(params: any): Promise<any> {
  const { datasourceUid, query, from, to, limit } = params;

  if (!datasourceUid || !query) {
    return {
      success: false,
      error: 'Missing required parameters: datasourceUid and query',
    };
  }

  try {
    const { getBackendSrv } = await import('@grafana/runtime');

    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/execute_logql',
      {
        datasourceUid,
        query,
        from: from || 'now-15m',
        to: to || 'now',
        limit: limit || 1000,
        serviceName: params.serviceName,
        environmentName: params.environmentName,
      }
    );

    return {
      success: true,
      ...response,
    };
  } catch (error: any) {
    console.error('Failed to execute LogQL query:', error);
    return {
      success: false,
      error: `Failed to execute LogQL query: ${error.message || 'Unknown error'}`,
      query,
      datasourceUid,
    };
  }
}

/**
 * Execute TraceQL query and return trace analytics
 */
async function executeTraceQL(params: any): Promise<any> {
  const { datasourceUid, query, from, to, limit } = params;

  if (!datasourceUid || !query) {
    return {
      success: false,
      error: 'Missing required parameters: datasourceUid and query',
    };
  }

  try {
    const { getBackendSrv } = await import('@grafana/runtime');

    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/execute_traceql',
      {
        datasourceUid,
        query,
        from: from || 'now-15m',
        to: to || 'now',
        limit: limit || 100,
        serviceName: params.serviceName,
        environmentName: params.environmentName,
      }
    );

    return {
      success: true,
      ...response,
    };
  } catch (error: any) {
    console.error('Failed to execute TraceQL query:', error);
    return {
      success: false,
      error: `Failed to execute TraceQL query: ${error.message || 'Unknown error'}`,
      query,
      datasourceUid,
    };
  }
}

/**
 * Fetch currently firing alerts from Grafana Alertmanager
 */
async function getFiringAlerts(params: any): Promise<any> {
  try {
    const { getBackendSrv } = await import('@grafana/runtime');
    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/get_firing_alerts',
      params || {}
    );
    return { success: true, ...response };
  } catch (error: any) {
    console.error('Failed to get firing alerts:', error);
    return { success: false, error: `Failed to get firing alerts: ${error.message || 'Unknown error'}` };
  }
}

/**
 * Search Grafana dashboards by title or tag
 */
async function searchDashboards(params: any): Promise<any> {
  try {
    const { getBackendSrv } = await import('@grafana/runtime');
    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/search_dashboards',
      params || {}
    );
    return { success: true, ...response };
  } catch (error: any) {
    console.error('Failed to search dashboards:', error);
    return { success: false, error: `Failed to search dashboards: ${error.message || 'Unknown error'}` };
  }
}

/**
 * Fetch a specific Grafana dashboard by UID
 */
async function getDashboard(params: any): Promise<any> {
  if (!params?.uid) {
    return { success: false, error: 'Missing required parameter: uid' };
  }
  try {
    const { getBackendSrv } = await import('@grafana/runtime');
    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/get_dashboard',
      params
    );
    return { success: true, ...response };
  } catch (error: any) {
    console.error('Failed to get dashboard:', error);
    return { success: false, error: `Failed to get dashboard: ${error.message || 'Unknown error'}` };
  }
}

/**
 * Fetch Grafana annotations for a given time range
 */
async function getAnnotations(params: any): Promise<any> {
  try {
    const { getBackendSrv } = await import('@grafana/runtime');
    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/get_annotations',
      params || {}
    );
    return { success: true, ...response };
  } catch (error: any) {
    console.error('Failed to get annotations:', error);
    return { success: false, error: `Failed to get annotations: ${error.message || 'Unknown error'}` };
  }
}

/**
 * List all Grafana folders
 */
async function listFolders(params: any): Promise<any> {
  try {
    const { getBackendSrv } = await import('@grafana/runtime');
    const response = await getBackendSrv().post(
      '/api/plugins/jorgeancal-zagalin-app/resources/tools/list_folders',
      params || {}
    );
    return { success: true, ...response };
  } catch (error: any) {
    console.error('Failed to list folders:', error);
    return { success: false, error: `Failed to list folders: ${error.message || 'Unknown error'}` };
  }
}
