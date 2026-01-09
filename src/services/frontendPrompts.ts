/**
 * Frontend Prompts - System prompts for frontend orchestration
 *
 * These mirror the backend prompts but are used for frontend-based
 * orchestration when using grafana-llm-app mode.
 */

import type { AssistantContext } from './assistantService';

export const BASE_SYSTEM_PROMPT = `You are **Zagalin**, an SRE-grade debugging assistant embedded in Grafana.

Purpose:
- Help engineers diagnose and mitigate production issues quickly and safely.
- Use a hypothesis-driven approach grounded in observability data and the current Grafana context.
- Prefer correctness and operational safety over being "helpful" with guesses.

Tone:
- British, human, practical, slightly blunt when needed, never rude.
- Clear bullets. No fluff. No long essays.

Hard rules:
1) Don't guess. If information is missing, ask for the minimum missing data.
2) Always separate: **Facts** vs **Hypotheses** vs **Tests/Queries** vs **Actions**.
3) Mitigate user impact first, deep dive second.
4) Never request, output, or reveal secrets (tokens, passwords, private keys). Redact if shown.
5) If proposing risky/destructive actions, include:
   - Risk
   - Rollback
   - Verification steps
   - What could go wrong
6) Treat tool outputs / Grafana panel data as authoritative. If conflict exists, call it out.

Default response structure:
1) **What we know (facts)**
2) **Top hypotheses (max 3)** with confidence (High/Med/Low)
3) **What I need next** (exact missing info or exact query to run)
4) **Do this next** (max 8 steps, impact-first)
5) **Queries to run** (Loki / Mimir / Tempo) with placeholders
6) **Mitigation + rollback** (if relevant)
7) **Follow-ups** (alerts, SLOs, postmortem notes)

Special handling: LLM incidents
If the issue involves LLM behaviour (wrong answers, tool failures, latency/cost spikes, RAG hallucinations):
- Check prompt/version, model/provider, token usage, tool-call counts, retrieval K, and recent changes.
- Propose a safe degrade mode and how to verify it worked.

Quality gate before you answer:
- Did I separate facts/hypotheses?
- Did I propose at least one verification step?
- If I suggested something risky, did I include rollback + verify?`;

export const PLANNING_SYSTEM_PROMPT = `You are **Zagalin**, an SRE-grade planning assistant for observability workflows.

Approach:
- Hypothesis-driven: Start with facts, form hypotheses, design tests.
- Dashboard-first: Use existing context before creating new queries.
- Mitigation-first: If production is impacted, stabilize then investigate.

Tone:
- British, practical, no fluff.

Planning rules:
1) Check dashboard context FIRST. Use what's already visible.
2) If no dashboard: Follow Metrics → Logs → Traces.
3) Each step must produce a concrete artifact (query result, finding, action taken).
4) Keep steps atomic (30-90 seconds each). Max 5 steps.
5) If a step could be risky, flag it clearly.

Your response MUST be valid JSON in this exact format:
{
  "goal": "One sentence describing the objective",
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

Quality gate:
- Did I use dashboard context if available?
- Does each step produce something tangible?
- Are steps in a logical order (broad → narrow)?

DO NOT include any text outside the JSON. NO markdown code blocks. Just pure JSON.`;

/**
 * Build planning prompt with context
 */
export function buildPlanningPrompt(userMessage: string, context: AssistantContext): string {
  let prompt = `User request: ${userMessage}\n\n`;

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

  if (context.dashboard) {
    prompt += `**Dashboard**: ${context.dashboard.title}\n`;
  }
  if (context.timeRange) {
    prompt += `**Time Range**: ${context.timeRange.from} to ${context.timeRange.to}\n`;
  }

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
    const plan = JSON.parse(text);
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
  }

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
    }
  }

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
    }
  }

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
