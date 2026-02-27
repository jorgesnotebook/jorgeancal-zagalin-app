import {
  classifyErrorMessage,
  makeZagalinError,
  toZagalinError,
  isZagalinError,
  type ZagalinErrorType,
} from './errors';

describe('classifyErrorMessage', () => {
  const cases: Array<[string, ZagalinErrorType]> = [
    ['Error: 401 - authentication required', 'auth_failure'],
    ['auth failed: invalid token', 'auth_failure'],
    ['unauthorized access', 'auth_failure'],
    ['Error: 429 - rate limit exceeded', 'rate_limit'],
    ['too many requests', 'rate_limit'],
    ['context_length_exceeded', 'context_window'],
    ['maximum context length exceeded', 'context_window'],
    ['too many tokens', 'context_window'],
    ['request timed out after 60 seconds', 'timeout'],
    ['deadline exceeded', 'timeout'],
    ['Error: 503 - service unavailable', 'llm_unavailable'],
    ['grafana-llm is not responding', 'llm_unavailable'],
    ['connection refused', 'datasource_unreachable'],
    ['no such host: prometheus.example.com', 'datasource_unreachable'],
    ['query invalid: unmatched brace', 'query_invalid'],
    ['validation failed for PromQL', 'query_invalid'],
    ['something completely unexpected', 'unknown'],
  ];

  it.each(cases)('classifies "%s" as %s', (msg, expected) => {
    expect(classifyErrorMessage(msg)).toBe(expected);
  });
});

describe('makeZagalinError', () => {
  it('creates an Error with ZagalinError fields', () => {
    const err = makeZagalinError('rate_limit');
    expect(err).toBeInstanceOf(Error);
    expect(err.zagalinType).toBe('rate_limit');
    expect(err.retryable).toBe(true);
    expect(typeof err.userMessage).toBe('string');
    expect(typeof err.hint).toBe('string');
    expect(err.hint.length).toBeGreaterThan(0);
  });

  it('uses provided raw message as Error.message', () => {
    const err = makeZagalinError('timeout', 'custom raw message');
    expect(err.message).toBe('custom raw message');
  });

  it('auth_failure is not retryable', () => {
    expect(makeZagalinError('auth_failure').retryable).toBe(false);
  });

  it('context_window is not retryable', () => {
    expect(makeZagalinError('context_window').retryable).toBe(false);
  });

  it('llm_unavailable is retryable', () => {
    expect(makeZagalinError('llm_unavailable').retryable).toBe(true);
  });
});

describe('toZagalinError', () => {
  it('passes through an existing ZagalinError unchanged', () => {
    const original = makeZagalinError('timeout');
    expect(toZagalinError(original)).toBe(original);
  });

  it('classifies a plain Error by its message', () => {
    const err = toZagalinError(new Error('Error: 401 - auth required'));
    expect(err.zagalinType).toBe('auth_failure');
  });

  it('uses provided backendType over message classification', () => {
    const err = toZagalinError(new Error('something vague'), 'rate_limit');
    expect(err.zagalinType).toBe('rate_limit');
  });

  it('converts a non-Error value', () => {
    const err = toZagalinError('plain string error');
    expect(err).toBeInstanceOf(Error);
    expect(err.zagalinType).toBe('unknown');
  });
});

describe('isZagalinError', () => {
  it('returns true for a ZagalinError', () => {
    expect(isZagalinError(makeZagalinError('rate_limit'))).toBe(true);
  });

  it('returns false for a plain Error', () => {
    expect(isZagalinError(new Error('oops'))).toBe(false);
  });

  it('returns false for non-Error values', () => {
    expect(isZagalinError('string')).toBe(false);
    expect(isZagalinError(null)).toBe(false);
    expect(isZagalinError(42)).toBe(false);
  });
});
