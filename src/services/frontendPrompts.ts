/**
 * Frontend Prompts - System prompts for frontend orchestration
 *
 * These mirror the backend prompts but are used for frontend-based
 * orchestration when using grafana-llm-app mode.
 */

import type { AssistantContext } from './assistantService';

// Base system prompt - defines Zagalin's identity and behavior
export const BASE_SYSTEM_PROMPT = `You are Zagalin, a Senior Staff SRE embedded in Grafana. You've been on-call for years and have deep operational experience with observability systems.

Your Role:
- Help engineers understand their metrics, logs, and traces
- Identify reliability issues before they become incidents
- Suggest practical improvements based on SRE best practices
- Debug production issues using the dashboard context you have

How You Work:
- You have full context about the current dashboard and panels - use it directly
- Prioritize: Reliability > Performance > Features
- Explain the "why" behind recommendations (you're teaching, not dictating)
- Focus on actionable insights, not theory
- If you see potential production risks, call them out clearly

When You Don't Have Dashboard Context (Observability Workflow):
1. **Start with Metrics** - Get high-level context from available metrics (PromQL)
   - What services are affected? What's the error rate? Is latency spiking?
2. **Dive into Logs** - If metrics don't tell the full story, check logs (LogQL)
   - What are the actual error messages? What patterns do you see?
3. **Trace for Details** - Use logs to find trace IDs, then pull traces (TraceQL)
   - What's the exact failure path? Which service is the bottleneck?

This is the SRE debugging workflow - start broad, narrow down. Always follow this pattern.

Communication Style:
- Concise and practical - engineers are busy
- Use technical terms correctly - they know what they're doing
- Provide code/queries ready to use
- When suggesting changes, explain operational impact

You've seen these systems fail at 3am. Share that experience.`;

// Planning system prompt - for generating structured execution plans
export const PLANNING_SYSTEM_PROMPT = `You are Zagalin, a planning assistant for observability workflows.

When given a task, break it down into clear, actionable steps.

CRITICAL: Check if the user has dashboard context.

**If user is on a dashboard:**
1. FIRST: Analyze what's already visible on the dashboard (panels, queries, current values, patterns)
2. THEN: Only go deeper into raw queries if the dashboard doesn't answer the question
3. Use existing dashboard panels as starting points - don't reinvent the wheel

**If user has NO dashboard context:**
Follow the observability pyramid (Metrics → Logs → Traces):
1. Start with high-level metrics
2. Narrow to logs if needed
3. Trace specific requests if necessary

Your response MUST be valid JSON in this exact format:
{
  "goal": "One sentence describing the overall objective",
  "steps": [
    {
      "title": "Step 1: Analyze dashboard panels",
      "description": "Review error rate and latency panels on current dashboard"
    },
    {
      "title": "Step 2: Check for anomalies",
      "description": "Identify any spikes or drops in the visible metrics"
    },
    {
      "title": "Step 3: Investigate root cause",
      "description": "Query logs for errors if dashboard shows elevated error rate"
    }
  ],
  "estimatedDuration": "2-3 minutes"
}

Guidelines:
- Keep steps atomic (each step should take 30-90 seconds)
- Maximum 5 steps total
- Each step should produce a concrete artifact (query, link, finding)
- PRIORITIZE dashboard context over raw queries when available
- Steps should build on each other logically

DO NOT include any text outside the JSON. NO markdown code blocks. Just pure JSON.`;

/**
 * Build planning prompt with context
 */
export function buildPlanningPrompt(userMessage: string, context: AssistantContext): string {
  let prompt = `User request: ${userMessage}\n\n`;

  // Emphasize dashboard context if available
  if (context.dashboard) {
    prompt += `DASHBOARD CONTEXT AVAILABLE:\n`;
    prompt += `Dashboard: "${context.dashboard.title}"\n`;
    if (context.dashboard.tags && context.dashboard.tags.length > 0) {
      prompt += `Tags: ${context.dashboard.tags.join(', ')}\n`;
    }

    if (context.dashboard.panels && context.dashboard.panels.length > 0) {
      prompt += `Visible panels (${context.dashboard.panels.length} total):\n`;
      context.dashboard.panels.slice(0, 10).forEach((panel, idx) => {
        prompt += `  ${idx + 1}. ${panel.title} (${panel.type})\n`;
      });
      if (context.dashboard.panels.length > 10) {
        prompt += `  ... and ${context.dashboard.panels.length - 10} more panels\n`;
      }
    }

    prompt += `\nIMPORTANT: The user is LOOKING at this dashboard right now. Start by analyzing what's visible before suggesting new queries.\n\n`;
  } else {
    prompt += `NO DASHBOARD CONTEXT - User is not on a dashboard.\n`;
    prompt += `Start with high-level queries to gather context.\n\n`;
  }

  // Add panel context if available
  if (context.panel) {
    prompt += `FOCUSED PANEL: "${context.panel.title}" (${context.panel.type})\n`;
    if (context.panel.targets && context.panel.targets.length > 0) {
      prompt += 'Panel queries:\n';
      for (const target of context.panel.targets) {
        const query = target.expr || target.query;
        if (query) {
          prompt += `  - ${query}\n`;
        }
      }
    }
    prompt += `\n`;
  }

  // Add time range if available
  if (context.timeRange) {
    prompt += `Time range: ${context.timeRange.from} to ${context.timeRange.to}\n\n`;
  }

  prompt += 'Create an execution plan that uses available context efficiently.';

  return prompt;
}

/**
 * Build step execution prompt
 */
export function buildStepPrompt(
  stepIndex: number,
  stepTitle: string,
  stepDescription: string,
  userMessage: string,
  context: AssistantContext,
  previousFindings: string[]
): string {
  let prompt = `${BASE_SYSTEM_PROMPT}\n\n---\n\n`;

  prompt += `You are executing step ${stepIndex + 1} of a planned investigation.\n\n`;
  prompt += `**Current Step**: ${stepTitle}\n`;
  prompt += `**Task**: ${stepDescription}\n\n`;
  prompt += `**Original Request**: ${userMessage}\n\n`;

  // Add context
  if (context.dashboard) {
    prompt += `**Dashboard**: ${context.dashboard.title}\n`;
  }
  if (context.timeRange) {
    prompt += `**Time Range**: ${context.timeRange.from} to ${context.timeRange.to}\n`;
  }

  // Add previous findings
  if (previousFindings.length > 0) {
    prompt += `\n**Previous Findings**:\n`;
    previousFindings.forEach((finding, idx) => {
      prompt += `Step ${idx + 1}: ${finding}\n`;
    });
  }

  prompt += `\n**Instructions**:
- Execute the task described above
- Provide concrete queries (PromQL, LogQL, or TraceQL) that can be run
- Format queries in markdown code blocks with language identifier
- Explain your findings briefly
- Focus on actionable results

Execute the step now:`;

  return prompt;
}

/**
 * Build final synthesis prompt
 */
export function buildSynthesisPrompt(
  userMessage: string,
  goal: string,
  stepResults: Array<{ title: string; content: string }>,
  artifacts: any[]
): string {
  let prompt = `${BASE_SYSTEM_PROMPT}\n\n---\n\n`;

  prompt += `You have completed a structured investigation. Now synthesize the findings into a comprehensive answer.\n\n`;
  prompt += `**Original Question**: ${userMessage}\n`;
  prompt += `**Investigation Goal**: ${goal}\n\n`;

  prompt += `**Steps Completed**:\n`;
  stepResults.forEach((step, idx) => {
    prompt += `\n### ${step.title}\n${step.content}\n`;
  });

  if (artifacts.length > 0) {
    prompt += `\n**Artifacts Collected**: ${artifacts.length} queries/findings\n`;
  }

  prompt += `\n**Your Task**:
Provide a comprehensive answer to the original question using the structure:

**Goal**: [Restate what we were trying to accomplish]

**Plan**: [Brief summary of the approach taken]

**Evidence**: [Key findings from each step]

**Conclusion**: [Direct answer to the user's question with actionable recommendations]

Be concise and actionable. Focus on what the user needs to know.`;

  return prompt;
}

/**
 * Parse JSON plan from LLM response
 */
export interface ExecutionPlan {
  goal: string;
  steps: Array<{
    index: number;
    title: string;
    description: string;
    status: 'pending' | 'in_progress' | 'completed' | 'failed';
  }>;
  estimatedDuration: string;
}

export function parsePlanFromResponse(text: string): ExecutionPlan | null {
  try {
    // Try direct JSON parse first
    const plan = JSON.parse(text);
    if (plan.goal && plan.steps && Array.isArray(plan.steps) && plan.steps.length > 0) {
      // Add index and status to steps if not present
      const stepsWithMetadata = plan.steps.map((step: any, index: number) => ({
        index: step.index !== undefined ? step.index : index,
        title: step.title,
        description: step.description,
        status: step.status || 'pending',
      }));
      return {
        goal: plan.goal,
        steps: stepsWithMetadata,
        estimatedDuration: plan.estimatedDuration || 'Estimating...',
      } as ExecutionPlan;
    }
  } catch (e) {
    // Not valid JSON, try to extract from markdown
  }

  // Try to extract JSON from markdown code blocks
  const jsonBlockPattern = /```(?:json)?\s*\n?([\s\S]+?)\n?```/;
  const match = text.match(jsonBlockPattern);

  if (match && match[1]) {
    try {
      const plan = JSON.parse(match[1]);
      if (plan.goal && plan.steps && Array.isArray(plan.steps) && plan.steps.length > 0) {
        const stepsWithMetadata = plan.steps.map((step: any, index: number) => ({
          index: step.index !== undefined ? step.index : index,
          title: step.title,
          description: step.description,
          status: step.status || 'pending',
        }));
        return {
          goal: plan.goal,
          steps: stepsWithMetadata,
          estimatedDuration: plan.estimatedDuration || 'Estimating...',
        } as ExecutionPlan;
      }
    } catch (e) {
      // Failed to parse
    }
  }

  // Try to find JSON in the text (look for { "goal": pattern)
  const jsonPattern = /\{\s*"goal"[\s\S]*?\}\s*(?:\]|\})/;
  const jsonMatch = text.match(jsonPattern);

  if (jsonMatch) {
    try {
      const plan = JSON.parse(jsonMatch[0]);
      if (plan.goal && plan.steps && Array.isArray(plan.steps) && plan.steps.length > 0) {
        const stepsWithMetadata = plan.steps.map((step: any, index: number) => ({
          index: step.index !== undefined ? step.index : index,
          title: step.title,
          description: step.description,
          status: step.status || 'pending',
        }));
        return {
          goal: plan.goal,
          steps: stepsWithMetadata,
          estimatedDuration: plan.estimatedDuration || 'Estimating...',
        } as ExecutionPlan;
      }
    } catch (e) {
      // Failed to parse
    }
  }

  // Fallback: create a simple 1-step plan
  console.warn('[frontendPrompts] Failed to parse plan, using fallback');
  return {
    goal: 'Address the user\'s request',
    steps: [
      {
        index: 0,
        title: 'Investigate and provide answer',
        description: 'Analyze the request and provide a comprehensive response',
        status: 'pending',
      },
    ],
    estimatedDuration: '1-2 minutes',
  };
}
