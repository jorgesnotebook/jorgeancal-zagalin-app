/**
 * Structured error handling for Zagalin.
 *
 * Classifies raw LLM / network errors into typed categories so the UI can
 * show a user-friendly message, a recovery hint, and — for transient failures
 * — a Retry button.
 */

export type ZagalinErrorType =
  | 'auth_failure'
  | 'rate_limit'
  | 'datasource_unreachable'
  | 'llm_unavailable'
  | 'query_invalid'
  | 'context_window'
  | 'timeout'
  | 'unknown';

export interface ZagalinError extends Error {
  zagalinType: ZagalinErrorType;
  userMessage: string;
  hint: string;
  retryable: boolean;
}

interface ErrorDefinition {
  userMessage: string;
  hint: string;
  retryable: boolean;
}

const ERROR_DEFINITIONS: Record<ZagalinErrorType, ErrorDefinition> = {
  auth_failure: {
    userMessage: 'Authentication failed.',
    hint: 'Check that the service account token and LLM API key are configured in Administration → Plugins → Zagalin → Settings.',
    retryable: false,
  },
  rate_limit: {
    userMessage: 'Rate limit reached.',
    hint: 'Too many requests in a short period. Wait a moment, then try again.',
    retryable: true,
  },
  datasource_unreachable: {
    userMessage: 'Datasource could not be reached.',
    hint: 'Check that the datasource is configured correctly and accessible from Grafana.',
    retryable: true,
  },
  llm_unavailable: {
    userMessage: 'LLM service is unavailable.',
    hint: 'Ensure grafana-llm-app is installed and the API key is valid. Check Administration → Plugins → LLM App.',
    retryable: true,
  },
  query_invalid: {
    userMessage: 'The generated query is invalid.',
    hint: 'Try rephrasing your request with more specific metric names or time ranges.',
    retryable: false,
  },
  context_window: {
    userMessage: 'Conversation is too long for the model.',
    hint: 'Start a new conversation, or ask a shorter question to reduce history length.',
    retryable: false,
  },
  timeout: {
    userMessage: 'Request timed out.',
    hint: 'The response took too long. Try a simpler question or check your connection.',
    retryable: true,
  },
  unknown: {
    userMessage: 'Something went wrong.',
    hint: 'An unexpected error occurred. Please try again.',
    retryable: true,
  },
};

/**
 * Classify a raw error message into a ZagalinErrorType.
 * Used as a fallback when the backend does not supply an explicit error_type.
 */
export function classifyErrorMessage(msg: string): ZagalinErrorType {
  const lower = msg.toLowerCase();

  if (/\b401\b|auth(entication)?.?fail|unauthorized|auth required/i.test(lower)) {
    return 'auth_failure';
  }
  if (/\b429\b|rate.?limit|too many requests/i.test(lower)) {
    return 'rate_limit';
  }
  if (/context.?length.?exceeded|context window|maximum context|too many tokens/i.test(lower)) {
    return 'context_window';
  }
  if (/time.?out|timed out|deadline exceeded|60 seconds/i.test(lower)) {
    return 'timeout';
  }
  if (/\b503\b|service unavailable|grafana-llm|llm.*unavail/i.test(lower)) {
    return 'llm_unavailable';
  }
  if (/connection refused|no such host|network error/i.test(lower)) {
    return 'datasource_unreachable';
  }
  if (/query.*invalid|invalid.*query|validation.*fail/i.test(lower)) {
    return 'query_invalid';
  }
  return 'unknown';
}

/** Build a ZagalinError from a type and optional raw message. */
export function makeZagalinError(type: ZagalinErrorType, rawMessage?: string): ZagalinError {
  const def = ERROR_DEFINITIONS[type] ?? ERROR_DEFINITIONS.unknown;
  const err = new Error(rawMessage || def.userMessage) as ZagalinError;
  err.zagalinType = type;
  err.userMessage = def.userMessage;
  err.hint = def.hint;
  err.retryable = def.retryable;
  return err;
}

/** Convert any thrown value to a ZagalinError, classifying if needed. */
export function toZagalinError(err: unknown, backendType?: string): ZagalinError {
  if (isZagalinError(err)) {
    return err;
  }
  const rawMsg = err instanceof Error ? err.message : String(err);
  const type = (backendType as ZagalinErrorType | undefined) ?? classifyErrorMessage(rawMsg);
  return makeZagalinError(type, rawMsg);
}

export function isZagalinError(err: unknown): err is ZagalinError {
  return err instanceof Error && 'zagalinType' in err;
}
