/**
 * Assistant Skills - Structured prompts for common Grafana tasks
 * These skills help Zagalin provide consistent, high-quality responses
 */

import type { PanelContext, GrafanaContext } from './contextTypes';

export interface SkillPrompt {
  name: string;
  systemPrompt: string;
  userPrompt: string;
}

/**
 * Skill: Explain this panel
 * Analyzes a panel's configuration and provides insights
 */
export function explainPanelSkill(panel: PanelContext): SkillPrompt {
  const queries = panel.targets
    .map(t => {
      const query = t.expr || t.query || '';
      return `Query ${t.refId}: ${query}`;
    })
    .join('\n');

  const systemPrompt = `You are Zagalin, a Grafana expert. Your task is to explain what a panel shows, how it works, and potential issues.

Follow this structure:
1. **What it shows**: Explain in plain English what the panel displays
2. **How it works**: Break down the queries and transformations
3. **Common pitfalls**: Warn about potential misinterpretations or issues
4. **Suggestions**: Offer 1-2 improvements if applicable

Keep it concise and actionable. Use technical terms but explain them.`;

  const userPrompt = `Explain this panel:

**Panel Title**: ${panel.title}
**Panel Type**: ${panel.type}
${panel.description ? `**Description**: ${panel.description}` : ''}

**Queries**:
${queries}

${panel.fieldConfig?.defaults?.unit ? `**Unit**: ${panel.fieldConfig.defaults.unit}` : ''}
${panel.transformations && panel.transformations.length > 0 ? `**Transformations**: ${panel.transformations.length} applied` : ''}`;

  return {
    name: 'explain_panel',
    systemPrompt,
    userPrompt,
  };
}

/**
 * Skill: Generate query from English
 * Converts natural language to PromQL/LogQL/TraceQL
 */
export function generateQuerySkill(request: string, context: GrafanaContext): SkillPrompt {
  // Detect likely query language from context
  const datasourceTypes = context.panel?.targets.map(t => t.datasource?.type).filter(Boolean) || [];
  const primaryDatasource = datasourceTypes[0] || 'prometheus';

  const queryLanguage = primaryDatasource === 'loki' ? 'LogQL' :
                       primaryDatasource === 'tempo' ? 'TraceQL' :
                       'PromQL';

  const systemPrompt = `You are Zagalin, a Grafana query expert specializing in ${queryLanguage}.

Your task is to convert English requests into valid ${queryLanguage} queries.

Guidelines:
- Generate syntactically correct ${queryLanguage}
- Use best practices (avoid high cardinality, use appropriate functions)
- Explain what the query does
- Warn about potential performance issues
- Suggest time range if not specified

Format your response as:
\`\`\`${queryLanguage.toLowerCase()}
<query>
\`\`\`

**Explanation**: <what the query does>
**Performance**: <any warnings about cardinality, time range, etc>
**Usage**: <suggested time range and resolution>`;

  const userPrompt = `Generate a ${queryLanguage} query for: "${request}"

${context.templateVariables && context.templateVariables.length > 0 ? `
Available template variables:
${context.templateVariables.map(v => `- $${v.name} = ${v.current.value}`).join('\n')}
` : ''}

${context.panel ? `
Current panel context:
${context.panel.targets.map(t => `- ${t.refId}: ${t.expr || t.query}`).join('\n')}
` : ''}`;

  return {
    name: 'generate_query',
    systemPrompt,
    userPrompt,
  };
}

/**
 * Skill: Guided troubleshooting
 * Provides a structured troubleshooting checklist
 */
export function guidedTroubleshootingSkill(issue: string, context: GrafanaContext): SkillPrompt {
  const systemPrompt = `You are Zagalin, a Grafana troubleshooting expert.

Your task is to provide a structured troubleshooting guide.

Format your response as:

**Problem Summary**: <restate the issue in one sentence>

**Quick Checks** (1-3 items):
- [ ] <simple check>
- [ ] <simple check>

**Diagnostic Queries** (3-5 queries to run):
1. **Check X**: \`<query>\` - <what to look for>
2. **Verify Y**: \`<query>\` - <what to look for>
3. **Investigate Z**: \`<query>\` - <what to look for>

**Common Causes**:
- <likely cause 1>
- <likely cause 2>

**Next Steps**: <what to do based on findings>

Keep it practical and actionable. Reference the current dashboard context when relevant.`;

  const userPrompt = `Help troubleshoot: "${issue}"

${context.dashboard ? `Current Dashboard: "${context.dashboard.title}"` : ''}
${context.panel ? `Current Panel: "${context.panel.title}" (${context.panel.type})` : ''}
${context.timeRange ? `Time Range: ${context.timeRange.from} to ${context.timeRange.to}` : ''}`;

  return {
    name: 'guided_troubleshooting',
    systemPrompt,
    userPrompt,
  };
}

/**
 * Skill: Analyze what's on screen
 * Intelligently describes the entire dashboard and what the user is looking at
 */
export function analyzeDashboardSkill(context: GrafanaContext): SkillPrompt {
  const systemPrompt = `You are Zagalin, an intelligent Grafana assistant with visual understanding. The user is asking about what they see on their screen.

Your task is to:
1. **Describe the overall purpose** - What story does this dashboard tell? What system/service is being monitored?
2. **Summarize key panels** - What are the most important visualizations? Group related panels together
3. **Identify patterns** - What should the user focus on? Are there any red flags or interesting trends?
4. **Provide context** - Why would someone use this dashboard? What decisions does it support?

Be conversational and insightful. Imagine you're sitting next to them explaining what you see. Use plain language but be specific about metrics and panel types.`;

  let dashboardInfo = '';

  if (context.dashboard) {
    dashboardInfo += `# Dashboard: "${context.dashboard.title}"\n`;
    if (context.dashboard.tags && context.dashboard.tags.length > 0) {
      dashboardInfo += `Tags: ${context.dashboard.tags.join(', ')}\n`;
    }
    dashboardInfo += `\n`;

    // List all panels
    if (context.dashboard.panels && context.dashboard.panels.length > 0) {
      dashboardInfo += `## Panels on this dashboard (${context.dashboard.panels.length} total):\n\n`;

      context.dashboard.panels.forEach((panel, idx) => {
        dashboardInfo += `${idx + 1}. **${panel.title}** (${panel.type})\n`;
        if (panel.description) {
          dashboardInfo += `   Description: ${panel.description}\n`;
        }
        if (panel.targets && panel.targets.length > 0) {
          const queries = panel.targets.map(t => t.expr || t.query).filter(Boolean);
          if (queries.length > 0) {
            dashboardInfo += `   Queries: ${queries.length} query(ies)\n`;
          }
        }
      });
    }
  }

  // Add time range context
  if (context.timeRange) {
    dashboardInfo += `\n## Time Range\n`;
    dashboardInfo += `Looking at data from ${context.timeRange.from} to ${context.timeRange.to}\n`;
  }

  // Add panel focus if viewing specific panel
  if (context.panel) {
    dashboardInfo += `\n## Currently Focused Panel\n`;
    dashboardInfo += `The user has "${context.panel.title}" (${context.panel.type}) in focus.\n`;
  }

  const userPrompt = dashboardInfo || 'No dashboard context available. The user might not be on a dashboard page.';

  return {
    name: 'analyze_dashboard',
    systemPrompt,
    userPrompt,
  };
}

/**
 * Detect which skill to use based on user input
 */
export function detectSkill(userInput: string, context: GrafanaContext): SkillPrompt | null {
  const input = userInput.toLowerCase();

  // Dashboard analysis triggers - check these FIRST to catch broad queries
  if (context.dashboard && (
    input.includes('what do i see') ||
    input.includes('what am i looking at') ||
    input.includes('describe this dashboard') ||
    input.includes('look at this') ||
    input.includes('in front of me') ||
    input.includes('what is this dashboard') ||
    input.includes('show me this') ||
    input.includes('analyze this') ||
    input.includes('understand this') ||
    input.includes('overview') ||
    (input.includes('this dashboard') && !input.includes('query'))
  )) {
    return analyzeDashboardSkill(context);
  }

  // Explain panel triggers (more specific than dashboard analysis)
  if (context.panel && (
    input.includes('this panel') ||
    input.includes('this graph') ||
    input.includes('this chart') ||
    (input.includes('explain') && (input.includes('panel') || input.includes('graph'))) ||
    input.includes('what does this show')
  )) {
    return explainPanelSkill(context.panel);
  }

  // Generate query triggers
  if (
    input.includes('query for') ||
    input.includes('promql') ||
    input.includes('logql') ||
    input.includes('create a query') ||
    input.includes('write a query') ||
    input.match(/\b(rate|sum|avg|count|histogram)\b/)
  ) {
    return generateQuerySkill(userInput, context);
  }

  // Troubleshooting triggers
  if (
    input.includes('troubleshoot') ||
    input.includes('debug') ||
    input.includes('not working') ||
    input.includes('why') ||
    input.includes('error') ||
    input.includes('failing')
  ) {
    return guidedTroubleshootingSkill(userInput, context);
  }

  return null;
}
