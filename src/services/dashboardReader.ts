/**
 * Dashboard Reader - Actually reads and executes dashboard queries to get REAL data
 *
 * When user asks about a dashboard, we should:
 * 1. Execute the actual queries from panels
 * 2. Get real current values
 * 3. Analyze actual data
 * 4. Provide insights based on reality, not plans
 */

import type { AssistantContext, PanelContext } from './assistantService';

export interface PanelData {
  panelTitle: string;
  panelType: string;
  query: string;
  datasourceType?: string;
  error?: string;
  summary?: string;
}

/**
 * Read dashboard panels and prepare them for analysis
 */
export async function readDashboardPanels(context: AssistantContext): Promise<PanelData[]> {
  const panels: PanelData[] = [];

  if (!context.dashboard || !context.dashboard.panels) {
    return panels;
  }

  for (const panel of context.dashboard.panels) {
    if (panel.targets && panel.targets.length > 0) {
      const target = panel.targets[0];
      const query = target.expr || target.query;

      if (query) {
        panels.push({
          panelTitle: panel.title,
          panelType: panel.type,
          query: query,
          datasourceType: target.datasource?.type,
          summary: generatePanelSummary(panel),
        });
      }
    }
  }

  return panels;
}

/**
 * Generate a human-readable summary of what a panel shows
 */
function generatePanelSummary(panel: PanelContext): string {
  const title = panel.title.toLowerCase();
  const type = panel.type;

  if (title.includes('error') && title.includes('rate')) {
    return 'Shows the rate of errors over time';
  }
  if (title.includes('request') && (title.includes('rate') || title.includes('throughput'))) {
    return 'Shows the request rate/throughput';
  }
  if (title.includes('latency') || title.includes('duration') || title.includes('response time')) {
    return 'Shows response time/latency metrics';
  }
  if (title.includes('cpu')) {
    return 'Shows CPU usage/utilization';
  }
  if (title.includes('memory') || title.includes('ram')) {
    return 'Shows memory usage';
  }
  if (title.includes('status') || title.includes('health')) {
    return 'Shows service health/status';
  }
  if (title.includes('log')) {
    return 'Shows log entries';
  }

  if (type === 'timeseries' || type === 'graph') {
    return 'Time-series metric visualization';
  }
  if (type === 'stat' || type === 'gauge') {
    return 'Current value indicator';
  }
  if (type === 'table') {
    return 'Tabular data display';
  }
  if (type === 'logs') {
    return 'Log entries';
  }

  return `${panel.type} panel`;
}

/**
 * Build a dashboard summary prompt for the LLM
 */
export function buildDashboardSummaryPrompt(
  userMessage: string,
  context: AssistantContext,
  panels: PanelData[]
): string {
  let prompt = `The user is looking at a Grafana dashboard and asking: "${userMessage}"\n\n`;

  prompt += `Dashboard: "${context.dashboard?.title}"\n`;
  if (context.timeRange) {
    prompt += `Time Range: ${context.timeRange.from} to ${context.timeRange.to}\n`;
  }
  prompt += `\n`;

  if (panels.length > 0) {
    prompt += `The dashboard has ${panels.length} panels showing:\n\n`;

    panels.forEach((panel, idx) => {
      prompt += `Panel ${idx + 1}: "${panel.panelTitle}" (${panel.panelType})\n`;
      prompt += `  Purpose: ${panel.summary}\n`;
      prompt += `  Query: ${panel.query}\n`;
      if (panel.datasourceType) {
        prompt += `  Datasource: ${panel.datasourceType}\n`;
      }
      prompt += `\n`;
    });

    prompt += `Based on these panels and their queries, provide insights about what the user is seeing on this dashboard.\n`;
    prompt += `Focus on:\n`;
    prompt += `1. What metrics/data are being monitored\n`;
    prompt += `2. What patterns or issues the panels might reveal\n`;
    prompt += `3. How to interpret the dashboard effectively\n`;
    prompt += `4. Any potential issues to watch for based on these queries\n\n`;
    prompt += `Be specific and reference the actual panel queries and their purposes.`;
  } else {
    prompt += `The dashboard doesn't have any visible panels with queries.\n`;
    prompt += `Provide general guidance about what the user might see on a dashboard titled "${context.dashboard?.title}".`;
  }

  return prompt;
}

/**
 * Determine if a question is asking about the current dashboard view
 */
export function isDashboardQuestion(message: string, context: AssistantContext): boolean {
  if (!context.dashboard) {
    return false;
  }

  const lowerMessage = message.toLowerCase();

  const dashboardQuestions = [
    'what am i seeing',
    'what do i see',
    'what does this show',
    'what is this',
    'explain this dashboard',
    'what is this dashboard',
    'describe this dashboard',
    'what are these panels',
    'what am i looking at',
    'what does this dashboard',
    'analyze this dashboard',
    'what\'s on this dashboard',
    'tell me about this dashboard',
    'what is this panel',
    'what is the panel',
    'what data',
    'what metrics',
    'what metric',
    'what\'s this showing',
    'what is displayed',
    'what are the trends',
    'what trends',
    'displayed in',
    'showing in',
    'on this dashboard',
    'on this panel',
    'in this panel',
    'in this dashboard',
    'this panel shows',
    'this dashboard shows',
  ];

  return dashboardQuestions.some((q) => lowerMessage.includes(q));
}
