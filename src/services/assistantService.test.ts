import { streamAssistantChat, AssistantRequest, StreamChunk } from './assistantService';
import { SSEParser } from '../utils/sseParser';
import { of, throwError } from 'rxjs';
import { toArray } from 'rxjs/operators';

// Mock fetch globally
global.fetch = jest.fn();

// Mock SSEParser
jest.mock('../utils/sseParser', () => ({
  SSEParser: {
    parseStream: jest.fn(),
  },
}));

// Mock pluginUrl
jest.mock('./pluginUrl', () => ({
  getPluginApiUrl: jest.fn((path) => `/api/plugins/jorgeancal-zagalin-app/resources${path}`),
}));

// Mock @grafana/llm - return a module with llm.enabled() that returns true
jest.mock('@grafana/llm', () => ({
  llm: {
    enabled: jest.fn(() => Promise.resolve(true)),
    streamChatCompletions: jest.fn(() => ({
      subscribe: jest.fn(),
    })),
  },
  isLLMPluginEnabled: jest.fn(() => true),
}));

// Mock getZagalinConfig to use backend-proxy mode (so tests use the mocked fetch)
jest.mock('./configHelper', () => ({
  getZagalinConfig: jest.fn(() => ({
    llmBackend: 'backend-proxy', // Force backend proxy mode for tests
    standardMode: {
      temperature: 0.7,
      maxTokens: 2000,
    },
    designMode: {
      temperature: 0.3,
      maxTokens: 4000,
    },
  })),
}));

// Mock versionReporter
jest.mock('./versionReporter', () => ({
  addVersionHeader: jest.fn((headers) => headers),
}));

// Helper to create a properly mocked Response with ReadableStream
const createMockResponse = (ok: boolean, status: number, bodyContent?: string): Response => {
  const mockBody = {
    getReader: jest.fn(() => ({
      read: jest.fn().mockResolvedValue({ done: true, value: undefined }),
      releaseLock: jest.fn(),
    })),
    cancel: jest.fn(),
  };

  return {
    ok,
    status,
    body: mockBody as any,
    bodyUsed: false,
    text: jest.fn().mockResolvedValue(bodyContent || ''),
    headers: new Headers(),
    redirected: false,
    statusText: '',
    type: 'basic' as ResponseType,
    url: '',
    clone: jest.fn(),
    arrayBuffer: jest.fn(),
    blob: jest.fn(),
    formData: jest.fn(),
    json: jest.fn(),
    bytes: jest.fn(),
  } as Response;
};

describe('assistantService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('streamAssistantChat', () => {
    const createMockRequest = (): AssistantRequest => ({
      message: 'test message',
      history: [
        { role: 'user', content: 'previous message' },
        { role: 'assistant', content: 'previous response' },
      ],
      context: {
        dashboard: {
          uid: 'dash-123',
          title: 'Test Dashboard',
        },
      },
    });

    it('should stream LLM response successfully with multiple chunks', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [
        { chunk: 'Hello' },
        { chunk: ' world' },
        { chunk: '!' },
        { done: true },
      ];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request = createMockRequest();

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(4);
            expect(chunks[0].chunk).toBe('Hello');
            expect(chunks[1].chunk).toBe(' world');
            expect(chunks[2].chunk).toBe('!');
            expect(chunks[3].done).toBe(true);
            done();
          },
          error: (err) => done(err),
        });
    });

    it('should stream with tool calls', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [
        {
          chunk: 'I will query Prometheus',
        },
        {
          tool_call: {
            id: 'call_123',
            type: 'function',
            function: {
              name: 'query_prometheus',
              arguments: '{"query":"up"}',
            },
          },
        },
        { done: true },
      ];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request = createMockRequest();

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(3);
            expect(chunks[0].chunk).toBe('I will query Prometheus');
            expect(chunks[1].tool_call).toBeDefined();
            expect(chunks[1].tool_call?.function.name).toBe('query_prometheus');
            expect(chunks[2].done).toBe(true);
            done();
          },
          error: (err) => done(err),
        });
    });

    it('should handle HTTP error responses', (done) => {
      const mockResponse = createMockResponse(false, 500, 'Internal Server Error');

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => {
          expect(err.message).toContain('500');
          expect(err.message).toContain('Internal Server Error');
          done();
        },
        complete: () => done(new Error('Should not complete')),
      });
    });

    it('should handle HTTP 401 Unauthorized', (done) => {
      const mockResponse = createMockResponse(false, 401, 'Unauthorized');

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => {
          expect(err.message).toContain('401');
          expect(err.message).toContain('Unauthorized');
          done();
        },
        complete: () => done(new Error('Should not complete')),
      });
    });

    it('should handle HTTP 429 Too Many Requests', (done) => {
      const mockResponse = createMockResponse(false, 429, 'Rate limit exceeded');

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => {
          expect(err.message).toContain('429');
          expect(err.message).toContain('Rate limit exceeded');
          done();
        },
        complete: () => done(new Error('Should not complete')),
      });
    });

    it('should handle network errors', (done) => {
      (global.fetch as jest.Mock).mockRejectedValue(new Error('Network error'));

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => {
          expect(err.message).toBe('Network error');
          done();
        },
        complete: () => done(new Error('Should not complete')),
      });
    });

    it('should handle SSE parsing errors', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      (SSEParser.parseStream as jest.Mock).mockReturnValue(throwError(() => new Error('SSE parsing failed')));

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => {
          expect(err.message).toBe('SSE parsing failed');
          done();
        },
        complete: () => done(new Error('Should not complete')),
      });
    });

    it('should handle abort signal properly', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      // Mock SSEParser to emit chunks slowly
      (SSEParser.parseStream as jest.Mock).mockReturnValue(
        of({ chunk: 'test1' }, { chunk: 'test2' }, { chunk: 'test3' })
      );

      const request = createMockRequest();

      const subscription = streamAssistantChat(request).subscribe({
        next: () => {},
        error: (err) => done(err),
        complete: () => {
          // Complete is valid here since we're unsubscribing
          done();
        },
      });

      // Unsubscribe immediately (should trigger abort)
      subscription.unsubscribe();

      // Verify fetch was called with abort signal
      const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
      expect(fetchCall).toBeDefined();
      expect(fetchCall[1]).toBeDefined();
      expect(fetchCall[1].signal).toBeDefined();
      expect(fetchCall[1].signal).toBeInstanceOf(AbortSignal);

      done();
    });

    it('should handle abort errors gracefully', (done) => {
      const abortError = new Error('The operation was aborted');
      abortError.name = 'AbortError';

      (global.fetch as jest.Mock).mockRejectedValue(abortError);

      const request = createMockRequest();

      streamAssistantChat(request).subscribe({
        next: () => {},
        error: () => done(new Error('Should not error on abort')),
        complete: () => {
          // Should complete gracefully on abort
          done();
        },
      });
    });

    it('should send correct request payload', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request = createMockRequest();
      request.skillHint = 'log_analysis';
      request.enrichedMessage = 'enriched content';
      request.mode = 'design';

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          expect(fetchCall[0]).toBe('/api/plugins/jorgeancal-zagalin-app/resources/llm/chat');

          const requestOptions = fetchCall[1];
          expect(requestOptions.method).toBe('POST');
          expect(requestOptions.headers['Content-Type']).toBe('application/json');
          expect(requestOptions.headers.Accept).toBe('text/event-stream');
          expect(requestOptions.credentials).toBe('same-origin');

          const body = JSON.parse(requestOptions.body);
          expect(body.message).toBe('test message');
          expect(body.history).toHaveLength(2);
          expect(body.context).toBeDefined();
          expect(body.skillHint).toBe('log_analysis');
          expect(body.enrichedMessage).toBe('enriched content');
          expect(body.mode).toBe('design');

          done();
        },
        error: (err) => done(err),
      });
    });

    it('should default to standard mode when mode not specified', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request = createMockRequest();
      // Don't specify mode

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          const body = JSON.parse(fetchCall[1].body);
          expect(body.mode).toBe('standard');
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should handle empty context', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          const body = JSON.parse(fetchCall[1].body);
          expect(body.context).toEqual({});
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should handle stream with error chunk', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [
        { chunk: 'Starting...' },
        { error: 'LLM provider error' },
      ];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request = createMockRequest();

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(2);
            expect(chunks[0].chunk).toBe('Starting...');
            expect(chunks[1].error).toBe('LLM provider error');
            done();
          },
          error: (err) => done(err),
        });
    });
  });

  describe('SSE parsing integration', () => {
    it('should parse chunks correctly', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [
        { chunk: 'Chunk 1' },
        { chunk: 'Chunk 2' },
        { chunk: 'Chunk 3' },
      ];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(3);
            chunks.forEach((chunk, idx) => {
              expect(chunk.chunk).toBe(`Chunk ${idx + 1}`);
            });
            done();
          },
          error: (err) => done(err),
        });
    });

    it('should detect done signal', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [{ chunk: 'Final chunk' }, { done: true }];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(2);
            expect(chunks[1].done).toBe(true);
            done();
          },
          error: (err) => done(err),
        });
    });

    it('should detect error signal', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      const mockChunks: StreamChunk[] = [{ error: 'Streaming error occurred' }];

      (SSEParser.parseStream as jest.Mock).mockReturnValue(of(...mockChunks));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      streamAssistantChat(request)
        .pipe(toArray())
        .subscribe({
          next: (chunks) => {
            expect(chunks).toHaveLength(1);
            expect(chunks[0].error).toBe('Streaming error occurred');
            done();
          },
          error: (err) => done(err),
        });
    });

    it('should pass correct options to SSEParser', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const parseStreamCall = (SSEParser.parseStream as jest.Mock).mock.calls[0];
          expect(parseStreamCall).toBeDefined();
          expect(parseStreamCall[0]).toBe(mockResponse);
          expect(parseStreamCall[1]).toBeDefined();
          expect(typeof parseStreamCall[1].onChunk).toBe('function');
          expect(typeof parseStreamCall[1].shouldComplete).toBe('function');
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should complete stream on done chunk', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      // SSEParser will call shouldComplete callback
      (SSEParser.parseStream as jest.Mock).mockImplementation((response: Response, options: any) => {
        return of(
          { chunk: 'test' },
          { done: true } // This should trigger shouldComplete
        );
      });

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      let completed = false;

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          completed = true;
          expect(completed).toBe(true);
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should complete stream on error chunk', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);

      (SSEParser.parseStream as jest.Mock).mockImplementation((response: Response, options: any) => {
        return of(
          { chunk: 'test' },
          { error: 'Something went wrong' } // This should trigger shouldComplete
        );
      });

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {},
      };

      let completed = false;

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          completed = true;
          expect(completed).toBe(true);
          done();
        },
        error: (err) => done(err),
      });
    });
  });

  describe('edge cases', () => {
    it('should handle very long messages', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const longMessage = 'a'.repeat(50000); // 50KB message

      const request: AssistantRequest = {
        message: longMessage,
        history: [],
        context: {},
      };

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          const body = JSON.parse(fetchCall[1].body);
          expect(body.message).toBe(longMessage);
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should handle large context objects', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request: AssistantRequest = {
        message: 'test',
        history: [],
        context: {
          dashboard: {
            uid: 'dash-123',
            title: 'Test Dashboard',
            panels: Array.from({ length: 100 }, (_, i) => ({
              title: `Panel ${i}`,
              type: 'graph',
              targets: [
                {
                  refId: 'A',
                  expr: `metric${i}`,
                },
              ],
            })),
          },
        },
      };

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          const body = JSON.parse(fetchCall[1].body);
          expect(body.context.dashboard?.panels).toHaveLength(100);
          done();
        },
        error: (err) => done(err),
      });
    });

    it('should handle empty message', (done) => {
      const mockResponse = createMockResponse(true, 200);

      (global.fetch as jest.Mock).mockResolvedValue(mockResponse);
      (SSEParser.parseStream as jest.Mock).mockReturnValue(of({ done: true }));

      const request: AssistantRequest = {
        message: '',
        history: [],
        context: {},
      };

      streamAssistantChat(request).subscribe({
        next: () => {},
        complete: () => {
          const fetchCall = (global.fetch as jest.Mock).mock.calls[0];
          const body = JSON.parse(fetchCall[1].body);
          expect(body.message).toBe('');
          done();
        },
        error: (err) => done(err),
      });
    });
  });
});
