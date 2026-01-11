/**
 * Panel Data Service - Executes panel queries and analyzes real data
 *
 * This service fixes the hallucination issue where Zagalin provides generic
 * dashboard explanations without fetching actual data.
 */

import { executeQuery, type QueryRequest, type QueryResponse } from './queryService';
import type { PanelContext, TimeRange } from './contextTypes';

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
  changePercent?: number; // % change over time window

  // Anomaly detection
  isSaturated?: boolean; // Near limit (>90%)
  hasSpike?: boolean; // Sudden increase (>50% jump)
  hasDrop?: boolean; // Sudden decrease (>50% drop)
  hasNoData?: boolean; // Query returned no data

  // Stats
  min?: number;
  max?: number;
  avg?: number;

  // Raw summary
  summary: string;
}

/**
 * Execute queries for key diagnostic panels
 *
 * Fetches real data from 3-5 most important panels to ground LLM explanations
 */
export async function executePanelQueries(
  panels: PanelContext[],
  timeRange: TimeRange,
  maxPanels = 5
): Promise<PanelDataAnalysis[]> {
  // Prioritize panels for execution
  const prioritizedPanels = prioritizePanels(panels);
  const panelsToExecute = prioritizedPanels.slice(0, maxPanels);

  const results: PanelDataAnalysis[] = [];

  for (const panel of panelsToExecute) {
    try {
      const analysis = await executePanelQuery(panel, timeRange);
      results.push(analysis);
    } catch (error: any) {
      console.error(`Failed to execute query for panel ${panel.title}:`, error);
      results.push({
        panelId: panel.id,
        panelTitle: panel.title,
        panelType: panel.type,
        datasourceUid: 'unknown',
        datasourceType: 'unknown',
        query: extractQuery(panel),
        success: false,
        error: error.message || 'Query execution failed',
        summary: `Failed to fetch data: ${error.message}`,
      });
    }
  }

  return results;
}

/**
 * Execute query for a single panel and analyze the data
 */
async function executePanelQuery(panel: PanelContext, timeRange: TimeRange): Promise<PanelDataAnalysis> {
  const target = panel.targets?.[0];
  if (!target) {
    throw new Error('No query target found');
  }

  const datasourceUid = target.datasource?.uid || 'unknown';
  const datasourceType = target.datasource?.type || 'unknown';
  const query = target.expr || target.query || '';

  if (!query) {
    throw new Error('No query expression found');
  }

  // Execute query
  const queryRequest: QueryRequest = {
    datasource: datasourceUid,
    queries: [
      {
        refId: 'A',
        datasourceUid,
        queryType: datasourceType,
        expr: target.expr,
        query: target.query,
        intervalMs: 15000, // 15s interval
        maxDataPoints: 100, // Limit data points
      },
    ],
    timeRange: {
      from: timeRange.from,
      to: timeRange.to,
    },
  };

  const response = await executeQuery(queryRequest);

  // Analyze response
  const analysis = analyzeQueryResponse(response, panel, datasourceUid, datasourceType, query);

  return analysis;
}

/**
 * Analyze query response to extract insights
 */
function analyzeQueryResponse(
  response: QueryResponse,
  panel: PanelContext,
  datasourceUid: string,
  datasourceType: string,
  query: string
): PanelDataAnalysis {
  const result = response.results?.A;

  if (!result || result.error) {
    return {
      panelId: panel.id,
      panelTitle: panel.title,
      panelType: panel.type,
      datasourceUid,
      datasourceType,
      query,
      success: false,
      error: result?.error || 'No data returned',
      summary: `Query failed: ${result?.error || 'No data'}`,
    };
  }

  const frames = result.frames || [];
  if (frames.length === 0) {
    return {
      panelId: panel.id,
      panelTitle: panel.title,
      panelType: panel.type,
      datasourceUid,
      datasourceType,
      query,
      success: true,
      hasNoData: true,
      summary: 'No data available for this time range',
    };
  }

  // Extract values from first frame
  const frame = frames[0];
  const fields = frame.fields || [];

  // Find value field (usually last field, after time)
  const valueField = fields.find((f: any) => f.type === 'number') || fields[fields.length - 1];

  if (!valueField || !valueField.values || valueField.values.length === 0) {
    return {
      panelId: panel.id,
      panelTitle: panel.title,
      panelType: panel.type,
      datasourceUid,
      datasourceType,
      query,
      success: true,
      hasNoData: true,
      summary: 'No numeric data in response',
    };
  }

  const values: number[] = valueField.values.filter((v: any) => typeof v === 'number' && !isNaN(v));

  if (values.length === 0) {
    return {
      panelId: panel.id,
      panelTitle: panel.title,
      panelType: panel.type,
      datasourceUid,
      datasourceType,
      query,
      success: true,
      hasNoData: true,
      summary: 'No valid numeric values',
    };
  }

  // Calculate statistics
  const currentValue = values[values.length - 1];
  const min = Math.min(...values);
  const max = Math.max(...values);
  const avg = values.reduce((sum, v) => sum + v, 0) / values.length;

  // Detect unit from field config
  const unit = panel.fieldConfig?.defaults?.unit || valueField.config?.unit;

  // Trend analysis (compare first half vs second half)
  const halfPoint = Math.floor(values.length / 2);
  const firstHalfAvg = values.slice(0, halfPoint).reduce((sum, v) => sum + v, 0) / halfPoint;
  const secondHalfAvg = values.slice(halfPoint).reduce((sum, v) => sum + v, 0) / (values.length - halfPoint);
  const changePercent = ((secondHalfAvg - firstHalfAvg) / firstHalfAvg) * 100;

  let trend: 'increasing' | 'decreasing' | 'stable' | 'spiky' | 'unknown' = 'unknown';
  if (Math.abs(changePercent) < 5) {
    trend = 'stable';
  } else if (changePercent > 5) {
    trend = 'increasing';
  } else if (changePercent < -5) {
    trend = 'decreasing';
  }

  // Check for spikiness (high variance)
  const variance = values.reduce((sum, v) => sum + Math.pow(v - avg, 2), 0) / values.length;
  const stdDev = Math.sqrt(variance);
  if (stdDev > avg * 0.5) {
    trend = 'spiky';
  }

  // Anomaly detection
  const isSaturated = unit === 'percentunit' && currentValue > 0.9; // >90%
  const hasSpike = values.some((v, i) => {
    if (i === 0) {
      return false;
    }
    const prevValue = values[i - 1];
    return prevValue > 0 && (v - prevValue) / prevValue > 0.5; // >50% jump
  });
  const hasDrop = values.some((v, i) => {
    if (i === 0) {
      return false;
    }
    const prevValue = values[i - 1];
    return prevValue > 0 && (prevValue - v) / prevValue > 0.5; // >50% drop
  });

  // Generate summary
  const summary = generateDataSummary({
    currentValue,
    trend,
    changePercent,
    min,
    max,
    avg,
    isSaturated,
    hasSpike,
    hasDrop,
    unit,
  });

  return {
    panelId: panel.id,
    panelTitle: panel.title,
    panelType: panel.type,
    datasourceUid,
    datasourceType,
    query,
    success: true,
    currentValue,
    unit,
    trend,
    changePercent,
    isSaturated,
    hasSpike,
    hasDrop,
    hasNoData: false,
    min,
    max,
    avg,
    summary,
  };
}

/**
 * Generate human-readable summary from data analysis
 */
function generateDataSummary(data: {
  currentValue: number;
  trend: string;
  changePercent: number;
  min: number;
  max: number;
  avg: number;
  isSaturated?: boolean;
  hasSpike?: boolean;
  hasDrop?: boolean;
  unit?: string;
}): string {
  const formatValue = (value: number, unit?: string) => {
    if (unit === 'percentunit') {
      return `${(value * 100).toFixed(1)}%`;
    }
    if (unit === 'bytes') {
      return formatBytes(value);
    }
    if (unit === 'ms' || unit === 'µs' || unit === 's') {
      return `${value.toFixed(2)}${unit}`;
    }
    return value.toFixed(2);
  };

  const parts: string[] = [];

  // Current value
  parts.push(`Current: ${formatValue(data.currentValue, data.unit)}`);

  // Trend
  if (data.trend === 'increasing') {
    parts.push(`trending up (+${Math.abs(data.changePercent).toFixed(1)}%)`);
  } else if (data.trend === 'decreasing') {
    parts.push(`trending down (-${Math.abs(data.changePercent).toFixed(1)}%)`);
  } else if (data.trend === 'stable') {
    parts.push('stable');
  } else if (data.trend === 'spiky') {
    parts.push('highly variable');
  }

  // Anomalies
  if (data.isSaturated) {
    parts.push('⚠️ SATURATED (>90%)');
  }
  if (data.hasSpike) {
    parts.push('⚠️ SPIKE DETECTED');
  }
  if (data.hasDrop) {
    parts.push('⚠️ DROP DETECTED');
  }

  // Range
  parts.push(`Range: ${formatValue(data.min, data.unit)} - ${formatValue(data.max, data.unit)}`);
  parts.push(`Avg: ${formatValue(data.avg, data.unit)}`);

  return parts.join(' | ');
}

/**
 * Format bytes to human-readable format
 */
function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return '0 B';
  }
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
}

/**
 * Prioritize panels for execution
 *
 * Prioritizes panels that are most useful for diagnosis:
 * 1. Error rate panels
 * 2. Latency/duration panels
 * 3. Request rate/throughput panels
 * 4. Resource usage (CPU, memory)
 * 5. Status/health panels
 */
function prioritizePanels(panels: PanelContext[]): PanelContext[] {
  const scored = panels.map((panel) => ({
    panel,
    score: calculatePanelPriority(panel),
  }));

  // Sort by score (highest first)
  scored.sort((a, b) => b.score - a.score);

  return scored.map((s) => s.panel);
}

/**
 * Calculate panel priority score (higher = more important)
 */
function calculatePanelPriority(panel: PanelContext): number {
  const title = panel.title.toLowerCase();
  let score = 0;

  // Error-related panels (highest priority)
  if (title.includes('error')) {
    score += 100;
  }
  if (title.includes('failure') || title.includes('failed')) {
    score += 90;
  }
  if (title.includes('5xx') || title.includes('500')) {
    score += 85;
  }

  // Latency/performance panels
  if (title.includes('latency')) {
    score += 80;
  }
  if (title.includes('duration') || title.includes('response time')) {
    score += 75;
  }
  if (title.includes('p95') || title.includes('p99') || title.includes('percentile')) {
    score += 70;
  }

  // Request rate/throughput
  if (title.includes('request') && (title.includes('rate') || title.includes('rps'))) {
    score += 65;
  }
  if (title.includes('throughput') || title.includes('qps')) {
    score += 60;
  }

  // Resource usage
  if (title.includes('cpu')) {
    score += 55;
  }
  if (title.includes('memory') || title.includes('ram')) {
    score += 50;
  }
  if (title.includes('disk') || title.includes('storage')) {
    score += 45;
  }

  // Status/health
  if (title.includes('status')) {
    score += 40;
  }
  if (title.includes('health')) {
    score += 35;
  }
  if (title.includes('up') || title.includes('availability')) {
    score += 30;
  }

  // Connection/queue metrics
  if (title.includes('connection') || title.includes('pool')) {
    score += 25;
  }
  if (title.includes('queue')) {
    score += 20;
  }

  // Prefer timeseries panels over gauges/stats
  if (panel.type === 'timeseries' || panel.type === 'graph') {
    score += 10;
  }
  if (panel.type === 'stat' || panel.type === 'gauge') {
    score += 5;
  }

  return score;
}

/**
 * Extract query string from panel
 */
function extractQuery(panel: PanelContext): string {
  const target = panel.targets?.[0];
  if (!target) {
    return '';
  }
  return target.expr || target.query || '';
}
