package plugin

import (
	"errors"
	"testing"
)

func TestClassifyError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantType      ZagalinErrorType
		wantRetryable bool
	}{
		{
			name:          "nil error",
			err:           nil,
			wantType:      ErrTypeUnknown,
			wantRetryable: false,
		},
		{
			name:          "401 authentication",
			err:           errors.New("authentication failed (401): invalid key"),
			wantType:      ErrTypeAuthFailure,
			wantRetryable: false,
		},
		{
			name:          "unauthorized",
			err:           errors.New("unauthorized: token expired"),
			wantType:      ErrTypeAuthFailure,
			wantRetryable: false,
		},
		{
			name:          "429 rate limit",
			err:           errors.New("429: rate limit exceeded"),
			wantType:      ErrTypeRateLimit,
			wantRetryable: true,
		},
		{
			name:          "too many requests",
			err:           errors.New("too many requests from this IP"),
			wantType:      ErrTypeRateLimit,
			wantRetryable: true,
		},
		{
			name:          "context_length_exceeded",
			err:           errors.New("context_length_exceeded: prompt too long"),
			wantType:      ErrTypeContextWindow,
			wantRetryable: false,
		},
		{
			name:          "context window",
			err:           errors.New("maximum context length is 8192 tokens"),
			wantType:      ErrTypeContextWindow,
			wantRetryable: false,
		},
		{
			name:          "timeout",
			err:           errors.New("request timeout after 30s"),
			wantType:      ErrTypeTimeout,
			wantRetryable: true,
		},
		{
			name:          "deadline exceeded",
			err:           errors.New("context deadline exceeded"),
			wantType:      ErrTypeTimeout,
			wantRetryable: true,
		},
		{
			name:          "503 service unavailable",
			err:           errors.New("503: service unavailable"),
			wantType:      ErrTypeLLMUnavailable,
			wantRetryable: true,
		},
		{
			name:          "grafana-llm unavailable",
			err:           errors.New("grafana-llm: connection refused"),
			wantType:      ErrTypeLLMUnavailable,
			wantRetryable: true,
		},
		{
			name:          "connection refused",
			err:           errors.New("dial tcp: connection refused"),
			wantType:      ErrTypeDatasourceUnreachable,
			wantRetryable: true,
		},
		{
			name:          "unknown error",
			err:           errors.New("something completely unexpected happened"),
			wantType:      ErrTypeUnknown,
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotRetryable := ClassifyError(tt.err)
			if gotType != tt.wantType {
				t.Errorf("ClassifyError() type = %v, want %v", gotType, tt.wantType)
			}
			if gotRetryable != tt.wantRetryable {
				t.Errorf("ClassifyError() retryable = %v, want %v", gotRetryable, tt.wantRetryable)
			}
		})
	}
}
