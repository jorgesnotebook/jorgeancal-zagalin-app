/**
 * LLMClient Unit Tests
 *
 * Tests the unified LLM client implementation including:
 * - Backend detection
 * - Backend proxy streaming
 * - Direct @grafana/llm streaming
 * - Fallback HTTP client
 * - System prompt building
 */

import { LLMClient } from './LLMClient';
import type { AssistantRequest } from './types';
import { getZagalinConfig } from '../configHelper';
import { getPluginApiUrl } from '../pluginUrl';
import { SSEParser } from '../../utils/sseParser';

// Mock dependencies
jest.mock('../configHelper');
jest.mock('../pluginUrl');
jest.mock('../../utils/sseParser');
jest.mock('@grafana/llm', () => ({
  llm: {
    enabled: jest.fn(),
    streamChatCompletions: jest.fn(),
  },
}));

const mockGetZagalinConfig = getZagalinConfig as jest.MockedFunction<typeof getZagalinConfig>;
const mockGetPluginApiUrl = getPluginApiUrl as jest.MockedFunction<typeof getPluginApiUrl>;
const mockSSEParser = SSEParser.parseStream as jest.MockedFunction<typeof SSEParser.parseStream>;

describe('LLMClient', () => {
  beforeEach(() => {
    jest.clearAllMocks();

    // Default mock implementations
    mockGetPluginApiUrl.mockReturnValue('/api/plugins/test/resources/llm/chat');
    mockGetZagalinConfig.mockReturnValue({
      llmBackend: 'backend-proxy',
      standardMode: {
        temperature: 0.7,
        maxTokens: 4000,
      },
      designMode: {
        temperature: 0.9,
        maxTokens: 8000,
      },
    } as any);
  });

  describe('Backend Detection', () => {
    it('should detect backend-proxy mode from config', () => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'backend-proxy',
      } as any);

      const client = new LLMClient();
      expect(client['backendType']).toBe('backend-proxy');
    });

    it('should detect grafana-llm mode from config', () => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'grafana-llm',
      } as any);

      const client = new LLMClient();
      expect(client['backendType']).toBe('grafana-llm');
    });

    it('should default to grafana-llm when config is missing', () => {
      mockGetZagalinConfig.mockReturnValue({} as any);

      const client = new LLMClient();
      expect(client['backendType']).toBe('grafana-llm');
    });

    it('should map direct mode to backend-proxy', () => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'direct',
      } as any);

      const client = new LLMClient();
      expect(client['backendType']).toBe('backend-proxy');
    });
  });

  describe('Backend Proxy Streaming', () => {
    let client: LLMClient;
    let fetchMock: jest.Mock;

    beforeEach(() => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'backend-proxy',
      } as any);

      client = new LLMClient();

      // Mock fetch
      fetchMock = jest.fn();
      global.fetch = fetchMock;
    });

    afterEach(() => {
      jest.restoreAllMocks();
    });

    it('should call backend proxy with correct request', async () => {
      const mockResponse = {
        ok: true,
        body: {} as ReadableStream,
      } as Response;

      fetchMock.mockResolvedValue(mockResponse);

      mockSSEParser.mockReturnValue({
        subscribe: jest.fn(),
      } as any);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);
      stream$.subscribe();

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/plugins/test/resources/llm/chat',
        expect.objectContaining({
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Accept: 'text/event-stream',
          },
          credentials: 'same-origin',
        })
      );

      const fetchBody = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
      expect(fetchBody).toMatchObject({
        message: 'Test message',
        history: [],
        context: {},
        mode: 'standard',
      });
    });

    it('should handle HTTP errors correctly', async () => {
      const mockResponse = {
        ok: false,
        status: 500,
        text: () => Promise.resolve('Internal Server Error'),
      } as Response;

      fetchMock.mockResolvedValue(mockResponse);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const errorHandler = jest.fn();
      stream$.subscribe({ error: errorHandler });

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(errorHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining('500'),
        })
      );
    });

    it('should parse SSE chunks correctly', async () => {
      const mockResponse = {
        ok: true,
        body: {} as ReadableStream,
      } as Response;

      fetchMock.mockResolvedValue(mockResponse);

      const mockObservable = {
        subscribe: jest.fn((observer) => {
          observer.next({ chunk: 'Hello', done: false });
          observer.next({ chunk: ' World', done: false });
          observer.next({ done: true });
          observer.complete();
          return { unsubscribe: jest.fn() };
        }),
      };

      mockSSEParser.mockReturnValue(mockObservable as any);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const chunks: any[] = [];
      stream$.subscribe({
        next: (chunk) => chunks.push(chunk),
      });

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(chunks).toEqual([
        { chunk: 'Hello', done: false },
        { chunk: ' World', done: false },
        { done: true },
      ]);
    });

    it('should support abort signal', async () => {
      // TODO: Implement abort signal test
      const mockResponse = {
        ok: true,
        body: {} as ReadableStream,
      } as Response;

      fetchMock.mockResolvedValue(mockResponse);

      mockSSEParser.mockReturnValue({
        subscribe: jest.fn(),
      } as any);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const subscription = client.chat(request).subscribe();

      // Unsubscribe should trigger abort
      subscription.unsubscribe();

      // Verify abort controller was used
      expect(fetchMock).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          signal: expect.any(AbortSignal),
        })
      );
    });
  });

  describe('Direct @grafana/llm Streaming', () => {
    let client: LLMClient;

    beforeEach(() => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'grafana-llm',
        standardMode: {
          temperature: 0.7,
          maxTokens: 4000,
        },
      } as any);

      client = new LLMClient();
    });

    it('should use @grafana/llm when available', async () => {
      const { llm } = await import('@grafana/llm');
      const mockLLM = llm as jest.Mocked<typeof llm>;

      mockLLM.enabled.mockResolvedValue(true);

      const mockStream = {
        subscribe: jest.fn((observer) => {
          observer.next({
            choices: [{ delta: { content: 'Test response' } }],
          });
          observer.next({
            choices: [{ finish_reason: 'stop' }],
          });
          return { unsubscribe: jest.fn() };
        }),
      };

      mockLLM.streamChatCompletions.mockReturnValue(mockStream as any);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const chunks: any[] = [];
      stream$.subscribe({
        next: (chunk) => chunks.push(chunk),
      });

      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(mockLLM.enabled).toHaveBeenCalled();
      expect(mockLLM.streamChatCompletions).toHaveBeenCalledWith(
        expect.objectContaining({
          messages: expect.arrayContaining([
            expect.objectContaining({ role: 'system' }),
            expect.objectContaining({ role: 'user', content: 'Test message' }),
          ]),
          temperature: 0.7,
          max_tokens: 4000,
        })
      );

      expect(chunks).toEqual([{ chunk: 'Test response', done: false }, { done: true }]);
    });

    it('should handle @grafana/llm not enabled', async () => {
      const { llm } = await import('@grafana/llm');
      const mockLLM = llm as jest.Mocked<typeof llm>;

      mockLLM.enabled.mockResolvedValue(false);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const errorHandler = jest.fn();
      stream$.subscribe({ error: errorHandler });

      await new Promise((resolve) => setTimeout(resolve, 50));

      expect(errorHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          message: expect.stringContaining('not enabled'),
        })
      );
    });
  });

  describe('System Prompt Building', () => {
    let client: LLMClient;

    beforeEach(() => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'grafana-llm',
      } as any);

      client = new LLMClient();
    });

    it('should include context in system prompt', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {
          dashboard: {
            uid: 'abc123',
            title: 'Test Dashboard',
            panels: [
              {
                title: 'CPU Usage',
                type: 'graph',
                targets: [
                  {
                    refId: 'A',
                    expr: 'rate(cpu_usage[5m])',
                  },
                ],
              },
            ],
          },
          timeRange: {
            from: 'now-1h',
            to: 'now',
          },
        },
      };

      const prompt = client['buildSystemPrompt'](request);

      expect(prompt).toContain('Test Dashboard');
      expect(prompt).toContain('abc123');
      expect(prompt).toContain('CPU Usage');
      expect(prompt).toContain('rate(cpu_usage[5m])');
      expect(prompt).toContain('now-1h');
    });

    it('should include attached contexts', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
        attachedContexts: [
          {
            dashboardUid: 'attached123',
            dashboardTitle: 'Attached Dashboard',
            panelId: 1,
            panelTitle: 'Attached Panel',
            timeFrom: 'now-6h',
            timeTo: 'now',
            addedAt: new Date('2026-01-11T10:00:00Z'),
          },
        ],
      };

      const prompt = client['buildSystemPrompt'](request);

      expect(prompt).toContain('ATTACHED DASHBOARDS (1)');
      expect(prompt).toContain('Attached Dashboard');
      expect(prompt).toContain('attached123');
      expect(prompt).toContain('Attached Panel');
    });

    it('should warn when no queries provided', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {
          panel: {
            title: 'Empty Panel',
            type: 'graph',
            targets: [],
          },
        },
      };

      const prompt = client['buildSystemPrompt'](request);

      expect(prompt).toContain('⚠️ NO QUERIES PROVIDED');
    });

    it('should include all required sections', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const prompt = client['buildSystemPrompt'](request);

      // Check for key sections
      expect(prompt).toContain('You are **Zagalin**');
      expect(prompt).toContain('Purpose:');
      expect(prompt).toContain('Hard rules:');
      expect(prompt).toContain('Evidence-first rules:');
      expect(prompt).toContain('Confidence Indicator');
      expect(prompt).toContain('PromQL for Prometheus');
    });
  });

  describe('Error Handling', () => {
    let client: LLMClient;
    let fetchMock: jest.Mock;

    beforeEach(() => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'backend-proxy',
      } as any);

      client = new LLMClient();
      fetchMock = jest.fn();
      global.fetch = fetchMock;
    });

    afterEach(() => {
      jest.restoreAllMocks();
    });

    it('should handle network errors', async () => {
      fetchMock.mockRejectedValue(new Error('Network error'));

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const errorHandler = jest.fn();
      stream$.subscribe({ error: errorHandler });

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(errorHandler).toHaveBeenCalledWith(expect.objectContaining({ message: 'Network error' }));
    });

    it('should handle abort errors gracefully', async () => {
      const abortError = new Error('The operation was aborted');
      abortError.name = 'AbortError';
      fetchMock.mockRejectedValue(abortError);

      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      const stream$ = client.chat(request);

      const errorHandler = jest.fn();
      const completeHandler = jest.fn();
      stream$.subscribe({
        error: errorHandler,
        complete: completeHandler,
      });

      await new Promise((resolve) => setTimeout(resolve, 10));

      // Should complete, not error on abort
      expect(errorHandler).not.toHaveBeenCalled();
      expect(completeHandler).toHaveBeenCalled();
    });
  });

  describe('Mode Handling', () => {
    let client: LLMClient;
    let fetchMock: jest.Mock;

    beforeEach(() => {
      mockGetZagalinConfig.mockReturnValue({
        llmBackend: 'backend-proxy',
        standardMode: {
          temperature: 0.7,
          maxTokens: 4000,
        },
        designMode: {
          temperature: 0.9,
          maxTokens: 8000,
        },
      } as any);

      client = new LLMClient();
      fetchMock = jest.fn();
      global.fetch = fetchMock;

      const mockResponse = {
        ok: true,
        body: {} as ReadableStream,
      } as Response;

      fetchMock.mockResolvedValue(mockResponse);

      mockSSEParser.mockReturnValue({
        subscribe: jest.fn(),
      } as any);
    });

    afterEach(() => {
      jest.restoreAllMocks();
    });

    it('should use standard mode by default', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
      };

      client.chat(request).subscribe();

      const fetchBody = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
      expect(fetchBody.mode).toBe('standard');
    });

    it('should support design mode', () => {
      const request: AssistantRequest = {
        message: 'Test message',
        history: [],
        context: {},
        mode: 'design',
      };

      client.chat(request).subscribe();

      const fetchBody = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
      expect(fetchBody.mode).toBe('design');
    });
  });
});
