/**
 * Response Processor - Parse and process LLM responses
 *
 * Consolidates:
 * - zagalinTools.ts (687 LOC) - Function calling tool execution
 * - artifactExtractor.ts (163 LOC) - Artifact parsing from LLM
 * - reasoningParser.ts (154 LOC) - Structured reasoning extraction
 * - actionExtractor.ts (64 LOC) - Explore link generation
 *
 * Total: 1,068 LOC → ~500 LOC (eliminate duplication)
 */

import type {
  ToolCall,
  ToolResult,
  ReasoningStep,
  SourceReference,
  Artifact,
  Action,
  ExplainableResponse,
} from './types';
import type { TimeRange } from '../contextTypes';
import { generateArtifactId } from '../../utils/idGenerator';
import { ARTIFACT_VALIDATION } from '../../utils/constants';

// Tool execution will be imported from tools/index.ts (to be created)
import { executeToolCall as executeToolInternal } from '../zagalinTools';

export class ResponseProcessor {
  /**
   * Parse function calls from LLM response
   *
   * Extracts tool calls from OpenAI-format responses
   */
  extractToolCalls(response: any): ToolCall[] {
    // Handle both direct tool_calls array and nested message format
    const toolCalls = response?.tool_calls || response?.message?.tool_calls || [];

    if (!Array.isArray(toolCalls) || toolCalls.length === 0) {
      return [];
    }

    return toolCalls
      .filter((tc: any) => tc.type === 'function' && tc.function?.name)
      .map((tc: any) => ({
        id: tc.id || `call_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`,
        type: 'function',
        function: {
          name: tc.function.name,
          arguments: tc.function.arguments || '{}',
        },
      }));
  }

  /**
   * Execute tool calls and return results
   *
   * Delegates to tool execution handlers from zagalinTools
   */
  async executeTools(toolCalls: ToolCall[]): Promise<ToolResult[]> {
    const results: ToolResult[] = [];

    for (const toolCall of toolCalls) {
      try {
        const result = await executeToolInternal(toolCall);

        results.push({
          toolCallId: toolCall.id,
          toolName: toolCall.function.name,
          result,
        });
      } catch (error: any) {
        results.push({
          toolCallId: toolCall.id,
          toolName: toolCall.function.name,
          result: null,
          error: error.message || 'Tool execution failed',
        });
      }
    }

    return results;
  }

  /**
   * Extract reasoning steps from response
   *
   * Parses structured reasoning sections (Observation, Analysis, Hypothesis, etc.)
   */
  extractReasoning(response: string): ReasoningStep[] {
    const reasoning: ReasoningStep[] = [];

    // Section patterns matching markdown headers with emojis
    const sectionPatterns = {
      observation: /##\s*🔍\s*Observation([\s\S]*?)(?=##|$)/i,
      analysis: /##\s*📊\s*Analysis([\s\S]*?)(?=##|$)/i,
      hypothesis: /##\s*💡\s*Hypothesis([\s\S]*?)(?=##|$)/i,
      conclusion: /##\s*✅\s*Conclusion([\s\S]*?)(?=##|$)/i,
      verification: /##\s*🔬\s*Verification([\s\S]*?)(?=##|$)/i,
    };

    // Extract observation
    const obsMatch = response.match(sectionPatterns.observation);
    if (obsMatch) {
      const content = obsMatch[1].trim();
      const confidence = this.extractConfidence(content, 0.9);

      reasoning.push({
        id: 'obs-1',
        type: 'observation',
        content,
        confidence,
        timestamp: new Date(),
        sources: this.extractMetricNames(content),
      });
    }

    // Extract analysis
    const analysisMatch = response.match(sectionPatterns.analysis);
    if (analysisMatch) {
      reasoning.push({
        id: 'analysis-1',
        type: 'analysis',
        content: analysisMatch[1].trim(),
        confidence: 0.85,
        timestamp: new Date(),
      });
    }

    // Extract hypothesis
    const hypMatch = response.match(sectionPatterns.hypothesis);
    if (hypMatch) {
      const content = hypMatch[1].trim();
      reasoning.push({
        id: 'hyp-1',
        type: 'hypothesis',
        content,
        confidence: this.extractHighestConfidence(content),
        timestamp: new Date(),
      });
    }

    // Extract conclusion
    const conclusionMatch = response.match(sectionPatterns.conclusion);
    if (conclusionMatch) {
      const content = conclusionMatch[1].trim();
      const confidence = this.extractConfidence(content, 0.7);

      reasoning.push({
        id: 'conclusion-1',
        type: 'conclusion',
        content,
        confidence,
        timestamp: new Date(),
      });
    }

    // Extract verification
    const verifyMatch = response.match(sectionPatterns.verification);
    if (verifyMatch) {
      reasoning.push({
        id: 'verify-1',
        type: 'verification',
        content: verifyMatch[1].trim(),
        confidence: 1.0,
        timestamp: new Date(),
      });
    }

    return reasoning;
  }

  /**
   * Extract artifacts from response
   *
   * Parses code blocks and inline queries for PromQL, LogQL, TraceQL
   */
  extractArtifacts(response: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const seenQueries = new Set<string>();

    // Extract code block queries first (higher priority)
    const codeBlockArtifacts = [
      ...this.extractCodeBlockPromQL(response),
      ...this.extractCodeBlockLogQL(response),
      ...this.extractCodeBlockTraceQL(response),
    ];

    // Add code block artifacts and track their content
    for (const artifact of codeBlockArtifacts) {
      artifacts.push(artifact);
      seenQueries.add(artifact.content);
    }

    // Extract inline queries (skip if already seen in code blocks)
    const inlineArtifacts = [
      ...this.extractInlinePromQL(response),
      ...this.extractInlineLogQL(response),
    ];

    for (const artifact of inlineArtifacts) {
      if (!seenQueries.has(artifact.content)) {
        artifacts.push(artifact);
        seenQueries.add(artifact.content);
      }
    }

    // Extract trace IDs (always add since they're unique by nature)
    artifacts.push(...this.extractTraceIDs(response));

    return artifacts;
  }

  /**
   * Extract action suggestions
   *
   * Generates explore links and query actions from artifacts
   */
  extractActions(response: string, context?: { timeRange?: TimeRange; datasourceUid?: string }): Action[] {
    const actions: Action[] = [];

    // Extract queries from code blocks
    const codeBlockRegex = /```(?:promql|logql|traceql|prometheus|loki|tempo)?\s*\n([\s\S]*?)\n```/gi;
    let match;
    let queryIndex = 1;

    while ((match = codeBlockRegex.exec(response)) !== null) {
      const query = match[1].trim();

      if (query) {
        actions.push({
          type: 'query',
          label: `Query ${queryIndex}`,
          data: {
            query,
            datasourceUid: context?.datasourceUid,
            timeRange: context?.timeRange,
          },
        });
        queryIndex++;
      }
    }

    return actions;
  }

  /**
   * Parse complete explainable response
   *
   * Combines reasoning, sources, and confidence into structured format
   */
  parseExplainableResponse(response: string): ExplainableResponse | null {
    try {
      const reasoning = this.extractReasoning(response);

      if (reasoning.length === 0) {
        return null;
      }

      const sources: SourceReference[] = [];
      const obsStep = reasoning.find((r) => r.type === 'observation');
      if (obsStep && obsStep.sources) {
        obsStep.sources.forEach((name, idx) => {
          sources.push({
            type: 'metric',
            id: `metric-${idx}`,
            name,
            relevance: 0.9,
          });
        });
      }

      const conclusion = reasoning.find((r) => r.type === 'conclusion');
      const overallConfidence = conclusion?.confidence || 0.7;

      return {
        answer: this.extractMainAnswer(response),
        reasoning,
        sources,
        confidence: overallConfidence,
      };
    } catch (error) {
      console.error('Failed to parse explainable response:', error);
      return null;
    }
  }

  // Private helper methods

  private extractCodeBlockPromQL(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /```(?:promql|prometheus)\s*\n([\s\S]+?)\n```/gi;
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH) {
        artifacts.push({
          id: generateArtifactId(),
          type: 'query',
          content: query,
          metadata: {
            signal: 'metrics',
            format: 'promql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractCodeBlockLogQL(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /```(?:logql|loki)\s*\n([\s\S]+?)\n```/gi;
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.includes('=') && query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH) {
        artifacts.push({
          id: generateArtifactId(),
          type: 'query',
          content: query,
          metadata: {
            signal: 'logs',
            format: 'logql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractCodeBlockTraceQL(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /```(?:traceql|tempo)\s*\n([\s\S]+?)\n```/gi;
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const query = match[1].trim();
      if (query.length > ARTIFACT_VALIDATION.MINIMUM_TRACEQL_LENGTH) {
        artifacts.push({
          id: generateArtifactId(),
          type: 'query',
          content: query,
          metadata: {
            signal: 'traces',
            format: 'traceql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractInlinePromQL(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /\b(rate|sum|avg|count|histogram_quantile|increase)\([^)]+\)(?:\{[^}]+\})?(?:\[[^\]]+\])?/g;
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const query = match[0].trim();
      if (query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH && !artifacts.some((a) => a.content === query)) {
        artifacts.push({
          id: generateArtifactId(),
          type: 'query',
          content: query,
          metadata: {
            signal: 'metrics',
            format: 'promql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractInlineLogQL(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /\{[^}]+\}\s*(?:\|[^|\n]+)*/g;
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const query = match[0].trim();
      if (
        query.includes('=') &&
        query.length > ARTIFACT_VALIDATION.MINIMUM_QUERY_LENGTH &&
        !artifacts.some((a) => a.content === query)
      ) {
        artifacts.push({
          id: generateArtifactId(),
          type: 'query',
          content: query,
          metadata: {
            signal: 'logs',
            format: 'logql',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractTraceIDs(text: string): Artifact[] {
    const artifacts: Artifact[] = [];
    const pattern = /\b[0-9a-f]{16,32}\b/gi;
    const seenTraceIDs = new Set<string>();
    let match;

    while ((match = pattern.exec(text)) !== null) {
      const traceID = match[0].toLowerCase();
      if (traceID.length >= ARTIFACT_VALIDATION.MINIMUM_TRACE_ID_LENGTH && !seenTraceIDs.has(traceID)) {
        seenTraceIDs.add(traceID);
        artifacts.push({
          id: generateArtifactId(),
          type: 'trace_id',
          content: traceID,
          metadata: {
            signal: 'traces',
          },
          timestamp: new Date().toISOString(),
        });
      }
    }

    return artifacts;
  }

  private extractConfidence(text: string, defaultValue: number): number {
    const confMatch = text.match(/(?:overall\s+)?confidence[:\s]+(\d+)%/i);
    if (confMatch) {
      return parseInt(confMatch[1], 10) / 100;
    }
    return defaultValue;
  }

  private extractHighestConfidence(text: string): number {
    const pattern = /confidence[:\s]+(\d+)%/gi;
    const matches = Array.from(text.matchAll(pattern));

    if (matches.length === 0) {
      return 0.7;
    }

    const confidences = matches.map((m) => {
      const num = m[1];
      return num ? parseInt(num, 10) / 100 : 0.7;
    });

    return Math.max(...confidences);
  }

  private extractMetricNames(text: string): string[] {
    const pattern = /`([a-z_][a-z0-9_]*)`/gi;
    const matches = text.matchAll(pattern);
    return Array.from(new Set(Array.from(matches, (m) => m[1])));
  }

  private extractMainAnswer(markdown: string): string {
    const conclusionPattern = /##\s*✅\s*Conclusion([\s\S]*?)(?=##|$)/i;
    const conclusionMatch = markdown.match(conclusionPattern);

    if (!conclusionMatch) {
      return markdown;
    }

    const lines = conclusionMatch[1].split('\n').filter((l) => l.trim());
    const answerLine = lines.find((l) => l.includes('Answer:'));

    if (answerLine) {
      return answerLine.replace(/\*\*Answer:\s*\*\*\s*/i, '').trim();
    }

    return lines[0] || markdown;
  }
}
