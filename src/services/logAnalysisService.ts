/**
 * Log Analysis Service
 *
 * Executes Loki queries and analyzes log data to provide insights about:
 * - Log volume and patterns
 * - Error rates and trends
 * - Notable log messages
 * - Label distributions
 */

import { getBackendSrv } from '@grafana/runtime';
import type { PanelContext, TimeRange } from './contextTypes';

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

/**
 * Analyze logs from a panel
 */
export async function analyzeLogs(
  panel: PanelContext,
  timeRange: TimeRange,
  maxLogLines = 1000
): Promise<LogAnalysisResult> {
  // Validate panel is a log panel
  if (panel.type !== 'logs') {
    return {
      success: false,
      error: 'Panel is not a log panel',
      query: '',
      datasourceUid: '',
      timeRange,
      totalCount: 0,
      errorCount: 0,
      warnCount: 0,
      logLevels: {},
      trend: 'unknown',
      topErrorMessages: [],
      topMessages: [],
      topLabels: {},
      summary: 'Not a log panel',
    };
  }

  // Extract Loki query
  const target = panel.targets?.[0];
  if (!target || !target.expr) {
    return {
      success: false,
      error: 'No Loki query found in panel',
      query: '',
      datasourceUid: '',
      timeRange,
      totalCount: 0,
      errorCount: 0,
      warnCount: 0,
      logLevels: {},
      trend: 'unknown',
      topErrorMessages: [],
      topMessages: [],
      topLabels: {},
      summary: 'No query found',
    };
  }

  const query = target.expr;
  const datasourceUid = target.datasource?.uid || '';

  try {
    // Execute log query via backend
    const logData = await executeLogQuery(query, datasourceUid, timeRange, maxLogLines);

    // Analyze log data
    const analysis = analyzeLogData(logData, query, datasourceUid, timeRange);

    return analysis;
  } catch (error) {
    console.error('Failed to analyze logs:', error);
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Unknown error',
      query,
      datasourceUid,
      timeRange,
      totalCount: 0,
      errorCount: 0,
      warnCount: 0,
      logLevels: {},
      trend: 'unknown',
      topErrorMessages: [],
      topMessages: [],
      topLabels: {},
      summary: 'Failed to fetch logs',
    };
  }
}

/**
 * Execute Loki query via Grafana backend
 */
async function executeLogQuery(
  query: string,
  datasourceUid: string,
  timeRange: TimeRange,
  maxLogLines: number
): Promise<LogQueryResult> {
  const response = await getBackendSrv().post('/api/ds/query', {
    queries: [
      {
        refId: 'A',
        expr: query,
        queryType: 'range',
        datasource: {
          type: 'loki',
          uid: datasourceUid,
        },
        maxLines: maxLogLines,
      },
    ],
    from: timeRange.from,
    to: timeRange.to,
  });

  return {
    frames: response.results?.A?.frames || [],
    error: response.results?.A?.error,
  };
}

interface LogQueryResult {
  frames: any[];
  error?: string;
}

/**
 * Analyze log data and extract insights
 */
function analyzeLogData(
  result: LogQueryResult,
  query: string,
  datasourceUid: string,
  timeRange: TimeRange
): LogAnalysisResult {
  if (result.error) {
    return {
      success: false,
      error: result.error,
      query,
      datasourceUid,
      timeRange,
      totalCount: 0,
      errorCount: 0,
      warnCount: 0,
      logLevels: {},
      trend: 'unknown',
      topErrorMessages: [],
      topMessages: [],
      topLabels: {},
      summary: `Query failed: ${result.error}`,
    };
  }

  const frames = result.frames;
  if (!frames || frames.length === 0) {
    return {
      success: true,
      query,
      datasourceUid,
      timeRange,
      totalCount: 0,
      errorCount: 0,
      warnCount: 0,
      logLevels: {},
      trend: 'stable',
      topErrorMessages: [],
      topMessages: [],
      topLabels: {},
      summary: 'No logs found in time range',
    };
  }

  // Extract log lines and labels
  const logLines: string[] = [];
  const labels: Record<string, Map<string, number>> = {};
  const timestamps: number[] = [];

  for (const frame of frames) {
    const schema = frame.schema;
    if (!schema || !schema.fields) {
      continue;
    }

    // Find time, line, and labels fields
    let timeField: any;
    let lineField: any;
    let labelsField: any;

    for (const field of schema.fields) {
      if (field.name === 'Time' || field.name === 'time') {
        timeField = field;
      } else if (field.name === 'Line' || field.name === 'line') {
        lineField = field;
      } else if (field.name === 'labels') {
        labelsField = field;
      }
    }

    // Extract data
    const data = frame.data;
    if (!data || !data.values) {
      continue;
    }

    const timeValues = timeField ? data.values[schema.fields.indexOf(timeField)] : [];
    const lineValues = lineField ? data.values[schema.fields.indexOf(lineField)] : [];
    const labelValues = labelsField ? data.values[schema.fields.indexOf(labelsField)] : [];

    for (let i = 0; i < lineValues.length; i++) {
      const line = lineValues[i];
      if (line) {
        logLines.push(line);
      }

      if (timeValues[i]) {
        timestamps.push(timeValues[i]);
      }

      // Parse labels
      if (labelValues[i]) {
        const lineLabels = labelValues[i];
        for (const [key, value] of Object.entries(lineLabels)) {
          if (!labels[key]) {
            labels[key] = new Map();
          }
          const count = labels[key].get(value as string) || 0;
          labels[key].set(value as string, count + 1);
        }
      }
    }
  }

  // Analyze log levels
  const logLevels: Record<string, number> = {};
  let errorCount = 0;
  let warnCount = 0;

  const errorPatterns = [/error/i, /exception/i, /fail/i, /fatal/i, /panic/i];
  const warnPatterns = [/warn/i, /warning/i];

  const errorMessages: string[] = [];
  const allMessages: Map<string, number> = new Map();

  for (const line of logLines) {
    // Detect log level
    const lowerLine = line.toLowerCase();

    if (errorPatterns.some((pattern) => pattern.test(lowerLine))) {
      errorCount++;
      logLevels['error'] = (logLevels['error'] || 0) + 1;
      errorMessages.push(line);
    } else if (warnPatterns.some((pattern) => pattern.test(lowerLine))) {
      warnCount++;
      logLevels['warn'] = (logLevels['warn'] || 0) + 1;
    } else if (/info/i.test(lowerLine)) {
      logLevels['info'] = (logLevels['info'] || 0) + 1;
    } else if (/debug/i.test(lowerLine)) {
      logLevels['debug'] = (logLevels['debug'] || 0) + 1;
    } else {
      logLevels['other'] = (logLevels['other'] || 0) + 1;
    }

    // Count message frequencies
    const count = allMessages.get(line) || 0;
    allMessages.set(line, count + 1);
  }

  // Detect trend
  const trend = detectLogTrend(timestamps);
  const changePercent = calculateLogChangePercent(timestamps);

  // Top messages
  const topMessages = Array.from(allMessages.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .map(([msg]) => msg);

  // Top error messages
  const topErrorMessages = errorMessages
    .filter((msg, idx, arr) => arr.indexOf(msg) === idx) // unique
    .slice(0, 10);

  // Top labels
  const topLabels: Record<string, string[]> = {};
  for (const [labelKey, labelCounts] of Object.entries(labels)) {
    const sorted = Array.from(labelCounts.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
      .map(([value]) => value);
    topLabels[labelKey] = sorted;
  }

  // Generate summary
  const summary = generateLogSummary({
    totalCount: logLines.length,
    errorCount,
    warnCount,
    trend,
    changePercent,
    topLabels,
  });

  return {
    success: true,
    query,
    datasourceUid,
    timeRange,
    totalCount: logLines.length,
    errorCount,
    warnCount,
    logLevels,
    trend,
    changePercent,
    topErrorMessages,
    topMessages,
    topLabels,
    summary,
  };
}

/**
 * Detect log trend from timestamps
 */
function detectLogTrend(timestamps: number[]): 'increasing' | 'decreasing' | 'stable' | 'unknown' {
  if (timestamps.length < 10) {
    return 'unknown';
  }

  // Split into first half and second half
  const mid = Math.floor(timestamps.length / 2);
  const firstHalf = timestamps.slice(0, mid).length;
  const secondHalf = timestamps.slice(mid).length;

  const ratio = secondHalf / firstHalf;

  if (ratio > 1.2) {
    return 'increasing';
  } else if (ratio < 0.8) {
    return 'decreasing';
  } else {
    return 'stable';
  }
}

/**
 * Calculate log change percentage
 */
function calculateLogChangePercent(timestamps: number[]): number | undefined {
  if (timestamps.length < 10) {
    return undefined;
  }

  const mid = Math.floor(timestamps.length / 2);
  const firstHalf = timestamps.slice(0, mid).length;
  const secondHalf = timestamps.slice(mid).length;

  return ((secondHalf - firstHalf) / firstHalf) * 100;
}

/**
 * Generate log summary
 */
function generateLogSummary(data: {
  totalCount: number;
  errorCount: number;
  warnCount: number;
  trend: string;
  changePercent?: number;
  topLabels: Record<string, string[]>;
}): string {
  const parts: string[] = [];

  parts.push(`Found ${data.totalCount} log lines`);

  if (data.errorCount > 0) {
    const errorPercent = ((data.errorCount / data.totalCount) * 100).toFixed(1);
    parts.push(`${data.errorCount} errors (${errorPercent}%)`);
  }

  if (data.warnCount > 0) {
    const warnPercent = ((data.warnCount / data.totalCount) * 100).toFixed(1);
    parts.push(`${data.warnCount} warnings (${warnPercent}%)`);
  }

  if (data.trend !== 'unknown') {
    parts.push(`trend: ${data.trend}`);
    if (data.changePercent !== undefined) {
      const sign = data.changePercent > 0 ? '+' : '';
      parts.push(`${sign}${data.changePercent.toFixed(1)}%`);
    }
  }

  // Add top labels
  const labelKeys = Object.keys(data.topLabels);
  if (labelKeys.length > 0) {
    const topLabel = labelKeys[0];
    const topValues = data.topLabels[topLabel];
    if (topValues.length > 0) {
      parts.push(`top ${topLabel}: ${topValues[0]}`);
    }
  }

  return parts.join(', ');
}
