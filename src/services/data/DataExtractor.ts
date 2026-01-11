/**
 * Data Extractor
 *
 * Consolidates query execution, panel data analysis, log analysis, and datasource listing.
 * Provides a unified interface for data extraction operations.
 */

import { getBackendSrv } from '@grafana/runtime';
import { getPluginResourcePath, getPluginApiUrl } from '../pluginUrl';
import type { PanelContext, TimeRange } from '../contextTypes';
import type {
  QueryRequest,
  QueryResponse,
  PanelDataAnalysis,
  LogAnalysisResult,
  DatasourceListResponse,
} from './types';

export class DataExtractor {
  /**
   * Execute a generic query
   */
  async executeQuery(request: QueryRequest): Promise<QueryResponse> {
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

  /**
   * Execute a Prometheus query
   */
  async queryPrometheus(datasourceUid: string, expr: string, from: string, to: string): Promise<QueryResponse> {
    return this.executeQuery({
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

  /**
   * Execute a Loki query
   */
  async queryLoki(datasourceUid: string, query: string, from: string, to: string): Promise<QueryResponse> {
    return this.executeQuery({
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

  /**
   * Execute a Tempo query
   */
  async queryTempo(datasourceUid: string, query: string, from: string, to: string): Promise<QueryResponse> {
    return this.executeQuery({
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

  /**
   * List available datasources
   */
  async listDatasources(): Promise<DatasourceListResponse> {
    try {
      return await getBackendSrv().get<DatasourceListResponse>(getPluginApiUrl('/datasources'));
    } catch (error: any) {
      console.error('Failed to fetch datasources:', error);
      return {
        datasources: [],
        allowedDatasources: [],
        defaultDatasource: '',
      };
    }
  }

  /**
   * Analyze a single panel
   */
  async analyzePanelData(panel: PanelContext, timeRange: TimeRange): Promise<PanelDataAnalysis> {
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

    const queryRequest: QueryRequest = {
      datasource: datasourceUid,
      queries: [
        {
          refId: 'A',
          datasourceUid,
          queryType: datasourceType,
          expr: target.expr,
          query: target.query,
          intervalMs: 15000,
          maxDataPoints: 100,
        },
      ],
      timeRange: {
        from: timeRange.from,
        to: timeRange.to,
      },
    };

    const response = await this.executeQuery(queryRequest);
    return this.analyzeMetrics(response, panel, datasourceUid, datasourceType, query);
  }

  /**
   * Analyze multiple panels (batch operation)
   */
  async analyzePanelDataBatch(
    panels: PanelContext[],
    timeRange: TimeRange,
    maxPanels = 5
  ): Promise<PanelDataAnalysis[]> {
    const prioritizedPanels = this.prioritizePanels(panels);
    const panelsToExecute = prioritizedPanels.slice(0, maxPanels);

    const results: PanelDataAnalysis[] = [];

    for (const panel of panelsToExecute) {
      try {
        const analysis = await this.analyzePanelData(panel, timeRange);
        results.push(analysis);
      } catch (error: any) {
        console.error(`Failed to execute query for panel ${panel.title}:`, error);
        results.push({
          panelId: panel.id,
          panelTitle: panel.title,
          panelType: panel.type,
          datasourceUid: 'unknown',
          datasourceType: 'unknown',
          query: this.extractQuery(panel),
          success: false,
          error: error.message || 'Query execution failed',
          summary: `Failed to fetch data: ${error.message}`,
        });
      }
    }

    return results;
  }

  /**
   * Analyze logs from a panel
   */
  async analyzeLogs(panel: PanelContext, timeRange: TimeRange, maxLogLines = 1000): Promise<LogAnalysisResult> {
    if (panel.type !== 'logs') {
      return this.createEmptyLogResult('Panel is not a log panel', '', '', timeRange);
    }

    const target = panel.targets?.[0];
    if (!target || !target.expr) {
      return this.createEmptyLogResult('No Loki query found in panel', '', '', timeRange);
    }

    const query = target.expr;
    const datasourceUid = target.datasource?.uid || '';

    try {
      const logData = await this.executeLogQuery(query, datasourceUid, timeRange, maxLogLines);
      return this.analyzeLogData(logData, query, datasourceUid, timeRange);
    } catch (error) {
      console.error('Failed to analyze logs:', error);
      return this.createEmptyLogResult(
        error instanceof Error ? error.message : 'Unknown error',
        query,
        datasourceUid,
        timeRange
      );
    }
  }

  /**
   * Analyze metric data from query response
   */
  private analyzeMetrics(
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

    const frame = frames[0];
    const fields = frame.fields || [];
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

    const currentValue = values[values.length - 1];
    const min = Math.min(...values);
    const max = Math.max(...values);
    const avg = values.reduce((sum, v) => sum + v, 0) / values.length;
    const unit = panel.fieldConfig?.defaults?.unit || valueField.config?.unit;

    // Trend analysis
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

    const variance = values.reduce((sum, v) => sum + Math.pow(v - avg, 2), 0) / values.length;
    const stdDev = Math.sqrt(variance);
    if (stdDev > avg * 0.5) {
      trend = 'spiky';
    }

    // Anomaly detection
    const isSaturated = unit === 'percentunit' && currentValue > 0.9;
    const hasSpike = values.some((v, i) => {
      if (i === 0) {
        return false;
      }
      const prevValue = values[i - 1];
      return prevValue > 0 && (v - prevValue) / prevValue > 0.5;
    });
    const hasDrop = values.some((v, i) => {
      if (i === 0) {
        return false;
      }
      const prevValue = values[i - 1];
      return prevValue > 0 && (prevValue - v) / prevValue > 0.5;
    });

    const summary = this.generateDataSummary({
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
   * Generate human-readable summary from metric data
   */
  private generateDataSummary(data: {
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
        return this.formatBytes(value);
      }
      if (unit === 'ms' || unit === 'µs' || unit === 's') {
        return `${value.toFixed(2)}${unit}`;
      }
      return value.toFixed(2);
    };

    const parts: string[] = [];

    parts.push(`Current: ${formatValue(data.currentValue, data.unit)}`);

    if (data.trend === 'increasing') {
      parts.push(`trending up (+${Math.abs(data.changePercent).toFixed(1)}%)`);
    } else if (data.trend === 'decreasing') {
      parts.push(`trending down (-${Math.abs(data.changePercent).toFixed(1)}%)`);
    } else if (data.trend === 'stable') {
      parts.push('stable');
    } else if (data.trend === 'spiky') {
      parts.push('highly variable');
    }

    if (data.isSaturated) {
      parts.push('⚠️ SATURATED (>90%)');
    }
    if (data.hasSpike) {
      parts.push('⚠️ SPIKE DETECTED');
    }
    if (data.hasDrop) {
      parts.push('⚠️ DROP DETECTED');
    }

    parts.push(`Range: ${formatValue(data.min, data.unit)} - ${formatValue(data.max, data.unit)}`);
    parts.push(`Avg: ${formatValue(data.avg, data.unit)}`);

    return parts.join(' | ');
  }

  /**
   * Format bytes to human-readable format
   */
  private formatBytes(bytes: number): string {
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
   */
  private prioritizePanels(panels: PanelContext[]): PanelContext[] {
    const scored = panels.map((panel) => ({
      panel,
      score: this.calculatePanelPriority(panel),
    }));

    scored.sort((a, b) => b.score - a.score);
    return scored.map((s) => s.panel);
  }

  /**
   * Calculate panel priority score
   */
  private calculatePanelPriority(panel: PanelContext): number {
    const title = panel.title.toLowerCase();
    let score = 0;

    // Error-related panels
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

    // Panel type preference
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
  private extractQuery(panel: PanelContext): string {
    const target = panel.targets?.[0];
    if (!target) {
      return '';
    }
    return target.expr || target.query || '';
  }

  /**
   * Execute Loki log query
   */
  private async executeLogQuery(
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

  /**
   * Analyze log data and extract insights
   */
  private analyzeLogData(
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

    const logLines: string[] = [];
    const labels: Record<string, Map<string, number>> = {};
    const timestamps: number[] = [];

    for (const frame of frames) {
      const schema = frame.schema;
      if (!schema || !schema.fields) {
        continue;
      }

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

    const logLevels: Record<string, number> = {};
    let errorCount = 0;
    let warnCount = 0;

    const errorPatterns = [/error/i, /exception/i, /fail/i, /fatal/i, /panic/i];
    const warnPatterns = [/warn/i, /warning/i];

    const errorMessages: string[] = [];
    const allMessages: Map<string, number> = new Map();

    for (const line of logLines) {
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

      const count = allMessages.get(line) || 0;
      allMessages.set(line, count + 1);
    }

    const trend = this.detectLogTrend(timestamps);
    const changePercent = this.calculateLogChangePercent(timestamps);

    const topMessages = Array.from(allMessages.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 10)
      .map(([msg]) => msg);

    const topErrorMessages = errorMessages
      .filter((msg, idx, arr) => arr.indexOf(msg) === idx)
      .slice(0, 10);

    const topLabels: Record<string, string[]> = {};
    for (const [labelKey, labelCounts] of Object.entries(labels)) {
      const sorted = Array.from(labelCounts.entries())
        .sort((a, b) => b[1] - a[1])
        .slice(0, 5)
        .map(([value]) => value);
      topLabels[labelKey] = sorted;
    }

    const summary = this.generateLogSummary({
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
  private detectLogTrend(timestamps: number[]): 'increasing' | 'decreasing' | 'stable' | 'unknown' {
    if (timestamps.length < 10) {
      return 'unknown';
    }

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
  private calculateLogChangePercent(timestamps: number[]): number | undefined {
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
  private generateLogSummary(data: {
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

  /**
   * Create empty log result for error cases
   */
  private createEmptyLogResult(
    error: string,
    query: string,
    datasourceUid: string,
    timeRange: TimeRange
  ): LogAnalysisResult {
    return {
      success: false,
      error,
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

interface LogQueryResult {
  frames: any[];
  error?: string;
}
