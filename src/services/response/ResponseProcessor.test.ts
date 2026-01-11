/**
 * ResponseProcessor Unit Tests
 *
 * Tests for response parsing, tool execution, and artifact extraction
 */

import { ResponseProcessor } from './ResponseProcessor';
import type { ToolCall } from './types';

// Mock dependencies
jest.mock('../zagalinTools', () => ({
  executeToolCall: jest.fn(),
}));

jest.mock('../../utils/idGenerator', () => ({
  generateArtifactId: jest.fn(() => 'test-artifact-id'),
}));

jest.mock('../../utils/constants', () => ({
  ARTIFACT_VALIDATION: {
    MINIMUM_QUERY_LENGTH: 3,
    MINIMUM_TRACEQL_LENGTH: 5,
    MINIMUM_TRACE_ID_LENGTH: 16,
  },
}));

import { executeToolCall } from '../zagalinTools';

describe('ResponseProcessor', () => {
  let processor: ResponseProcessor;

  beforeEach(() => {
    processor = new ResponseProcessor();
    jest.clearAllMocks();
  });

  describe('extractToolCalls', () => {
    it('should extract tool calls from response', () => {
      const response = {
        tool_calls: [
          {
            id: 'call_123',
            type: 'function',
            function: {
              name: 'query_prometheus',
              arguments: '{"metric":"up"}',
            },
          },
        ],
      };

      const toolCalls = processor.extractToolCalls(response);

      expect(toolCalls).toHaveLength(1);
      expect(toolCalls[0].id).toBe('call_123');
      expect(toolCalls[0].function.name).toBe('query_prometheus');
      expect(toolCalls[0].function.arguments).toBe('{"metric":"up"}');
    });

    it('should handle nested message.tool_calls format', () => {
      const response = {
        message: {
          tool_calls: [
            {
              id: 'call_456',
              type: 'function',
              function: {
                name: 'create_promql_query',
                arguments: '{}',
              },
            },
          ],
        },
      };

      const toolCalls = processor.extractToolCalls(response);

      expect(toolCalls).toHaveLength(1);
      expect(toolCalls[0].id).toBe('call_456');
    });

    it('should return empty array when no tool calls', () => {
      const response = { content: 'No tools here' };

      const toolCalls = processor.extractToolCalls(response);

      expect(toolCalls).toEqual([]);
    });

    it('should filter out non-function tool calls', () => {
      const response = {
        tool_calls: [
          {
            id: 'call_1',
            type: 'function',
            function: {
              name: 'valid_tool',
              arguments: '{}',
            },
          },
          {
            id: 'call_2',
            type: 'other',
            function: {
              name: 'invalid_tool',
              arguments: '{}',
            },
          },
        ],
      };

      const toolCalls = processor.extractToolCalls(response);

      expect(toolCalls).toHaveLength(1);
      expect(toolCalls[0].function.name).toBe('valid_tool');
    });

    it('should generate ID if missing', () => {
      const response = {
        tool_calls: [
          {
            type: 'function',
            function: {
              name: 'test_tool',
              arguments: '{}',
            },
          },
        ],
      };

      const toolCalls = processor.extractToolCalls(response);

      expect(toolCalls[0].id).toMatch(/^call_/);
    });
  });

  describe('executeTools', () => {
    it('should execute tool calls and return results', async () => {
      const mockExecute = executeToolCall as jest.MockedFunction<typeof executeToolCall>;
      mockExecute.mockResolvedValue({ query: 'up' });

      const toolCalls: ToolCall[] = [
        {
          id: 'call_1',
          type: 'function',
          function: {
            name: 'test_tool',
            arguments: '{}',
          },
        },
      ];

      const results = await processor.executeTools(toolCalls);

      expect(results).toHaveLength(1);
      expect(results[0].toolCallId).toBe('call_1');
      expect(results[0].toolName).toBe('test_tool');
      expect(results[0].result).toEqual({ query: 'up' });
      expect(results[0].error).toBeUndefined();
    });

    it('should handle tool execution errors', async () => {
      const mockExecute = executeToolCall as jest.MockedFunction<typeof executeToolCall>;
      mockExecute.mockRejectedValue(new Error('Tool failed'));

      const toolCalls: ToolCall[] = [
        {
          id: 'call_1',
          type: 'function',
          function: {
            name: 'failing_tool',
            arguments: '{}',
          },
        },
      ];

      const results = await processor.executeTools(toolCalls);

      expect(results).toHaveLength(1);
      expect(results[0].error).toBe('Tool failed');
      expect(results[0].result).toBeNull();
    });

    it('should execute multiple tools sequentially', async () => {
      const mockExecute = executeToolCall as jest.MockedFunction<typeof executeToolCall>;
      mockExecute
        .mockResolvedValueOnce({ result: 'first' })
        .mockResolvedValueOnce({ result: 'second' });

      const toolCalls: ToolCall[] = [
        {
          id: 'call_1',
          type: 'function',
          function: { name: 'tool1', arguments: '{}' },
        },
        {
          id: 'call_2',
          type: 'function',
          function: { name: 'tool2', arguments: '{}' },
        },
      ];

      const results = await processor.executeTools(toolCalls);

      expect(results).toHaveLength(2);
      expect(results[0].result).toEqual({ result: 'first' });
      expect(results[1].result).toEqual({ result: 'second' });
    });
  });

  describe('extractReasoning', () => {
    it('should extract all reasoning sections', () => {
      const response = `
## 🔍 Observation
The system shows high CPU usage at 95%. Confidence: 90%

## 📊 Analysis
Load increased after deployment. Likely resource leak.

## 💡 Hypothesis
Memory leak in new feature causing high CPU. Confidence: 75%

## ✅ Conclusion
Root cause is memory leak. Overall confidence: 80%

## 🔬 Verification
Fixed and verified in staging.
      `;

      const reasoning = processor.extractReasoning(response);

      expect(reasoning).toHaveLength(5);
      expect(reasoning[0].type).toBe('observation');
      expect(reasoning[0].confidence).toBe(0.9);
      expect(reasoning[1].type).toBe('analysis');
      expect(reasoning[2].type).toBe('hypothesis');
      expect(reasoning[3].type).toBe('conclusion');
      expect(reasoning[3].confidence).toBe(0.8);
      expect(reasoning[4].type).toBe('verification');
    });

    it('should extract metric names from observation', () => {
      const response = `
## 🔍 Observation
Metrics \`cpu_usage\` and \`memory_usage\` are high.
      `;

      const reasoning = processor.extractReasoning(response);

      expect(reasoning[0].sources).toEqual(['cpu_usage', 'memory_usage']);
    });

    it('should return empty array when no reasoning sections', () => {
      const response = 'Just a regular response';

      const reasoning = processor.extractReasoning(response);

      expect(reasoning).toEqual([]);
    });

    it('should extract highest confidence from hypothesis', () => {
      const response = `
## 💡 Hypothesis
First hypothesis: confidence: 60%
Second hypothesis: confidence: 85%
      `;

      const reasoning = processor.extractReasoning(response);

      expect(reasoning[0].confidence).toBe(0.85);
    });
  });

  describe('extractArtifacts', () => {
    it('should extract PromQL queries from code blocks', () => {
      const response = '```promql\nrate(http_requests_total[5m])\n```';

      const artifacts = processor.extractArtifacts(response);

      expect(artifacts).toHaveLength(1);
      expect(artifacts[0].type).toBe('query');
      expect(artifacts[0].content).toBe('rate(http_requests_total[5m])');
      expect(artifacts[0].metadata.format).toBe('promql');
    });

    it('should extract LogQL queries from code blocks', () => {
      const response = '```logql\n{job="varlogs"} |= "error"\n```';

      const artifacts = processor.extractArtifacts(response);

      expect(artifacts).toHaveLength(1);
      expect(artifacts[0].metadata.format).toBe('logql');
    });

    it('should extract TraceQL queries from code blocks', () => {
      const response = '```traceql\n{service.name="frontend"}\n```';

      const artifacts = processor.extractArtifacts(response);

      expect(artifacts).toHaveLength(1);
      expect(artifacts[0].metadata.format).toBe('traceql');
    });

    it('should extract inline PromQL queries', () => {
      const response = 'You can use rate(http_requests[5m]) to check the rate.';

      const artifacts = processor.extractArtifacts(response);

      expect(artifacts.length).toBeGreaterThan(0);
      expect(artifacts[0].content).toBe('rate(http_requests[5m])');
    });

    it('should extract inline LogQL queries', () => {
      const response = 'Try {job="app"} | logfmt for parsing.';

      const artifacts = processor.extractArtifacts(response);

      expect(artifacts.length).toBeGreaterThan(0);
      expect(artifacts.some((a) => a.content.includes('job="app"'))).toBe(true);
    });

    it('should extract trace IDs', () => {
      const response = 'Trace ID: 1234567890abcdef1234567890abcdef';

      const artifacts = processor.extractArtifacts(response);

      const traceArtifacts = artifacts.filter((a) => a.type === 'trace_id');
      expect(traceArtifacts.length).toBeGreaterThan(0);
      expect(traceArtifacts[0].content).toBe('1234567890abcdef1234567890abcdef');
    });

    it('should deduplicate inline queries', () => {
      const response = 'rate(up[5m]) and rate(up[5m]) appear twice';

      const artifacts = processor.extractArtifacts(response);

      const promqlArtifacts = artifacts.filter((a) => a.metadata.format === 'promql');
      const uniqueQueries = new Set(promqlArtifacts.map((a) => a.content));
      expect(uniqueQueries.size).toBe(promqlArtifacts.length);
    });
  });

  describe('extractActions', () => {
    it('should extract query actions from code blocks', () => {
      const response = '```promql\nup\n```\n```logql\n{job="app"}\n```';

      const actions = processor.extractActions(response);

      expect(actions).toHaveLength(2);
      expect(actions[0].type).toBe('query');
      expect(actions[0].label).toBe('Query 1');
      expect(actions[0].data.query).toBe('up');
      expect(actions[1].label).toBe('Query 2');
    });

    it('should include context in actions', () => {
      const response = '```promql\nup\n```';
      const context = {
        datasourceUid: 'prom-1',
        timeRange: { from: 'now-1h', to: 'now' },
      };

      const actions = processor.extractActions(response, context);

      expect(actions[0].data.datasourceUid).toBe('prom-1');
      expect(actions[0].data.timeRange).toEqual(context.timeRange);
    });

    it('should return empty array when no queries', () => {
      const response = 'No queries here';

      const actions = processor.extractActions(response);

      expect(actions).toEqual([]);
    });
  });

  describe('parseExplainableResponse', () => {
    it('should parse complete explainable response', () => {
      const response = `
## 🔍 Observation
CPU at 90%. Metric: \`cpu_usage\`

## ✅ Conclusion
High CPU usage detected. Overall confidence: 85%
**Answer:** CPU is at 90%
      `;

      const explainable = processor.parseExplainableResponse(response);

      expect(explainable).not.toBeNull();
      expect(explainable!.reasoning).toHaveLength(2);
      expect(explainable!.sources).toHaveLength(1);
      expect(explainable!.sources[0].name).toBe('cpu_usage');
      expect(explainable!.confidence).toBe(0.85);
      expect(explainable!.answer).toBe('CPU is at 90%');
    });

    it('should return null when no reasoning sections', () => {
      const response = 'Just a plain response';

      const explainable = processor.parseExplainableResponse(response);

      expect(explainable).toBeNull();
    });

    it('should extract main answer from conclusion', () => {
      const response = `
## ✅ Conclusion
**Answer:** The system is healthy
Other info...
      `;

      const explainable = processor.parseExplainableResponse(response);

      expect(explainable!.answer).toBe('The system is healthy');
    });

    it('should fallback to first conclusion line if no Answer label', () => {
      const response = `
## ✅ Conclusion
System is healthy
More details...
      `;

      const explainable = processor.parseExplainableResponse(response);

      expect(explainable!.answer).toBe('System is healthy');
    });
  });
});
