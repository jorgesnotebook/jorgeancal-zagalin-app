package plugin

import (
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// ZagalinErrorType categorises backend errors for structured client-side handling.
type ZagalinErrorType string

const (
	ErrTypeAuthFailure           ZagalinErrorType = "auth_failure"
	ErrTypeRateLimit             ZagalinErrorType = "rate_limit"
	ErrTypeDatasourceUnreachable ZagalinErrorType = "datasource_unreachable"
	ErrTypeLLMUnavailable        ZagalinErrorType = "llm_unavailable"
	ErrTypeQueryInvalid          ZagalinErrorType = "query_invalid"
	ErrTypeContextWindow         ZagalinErrorType = "context_window"
	ErrTypeTimeout               ZagalinErrorType = "timeout"
	ErrTypeUnknown               ZagalinErrorType = "unknown"
)

// ClassifyError inspects an error's message and returns its category and
// whether the client can safely retry the request.
func ClassifyError(err error) (ZagalinErrorType, bool) {
	if err == nil {
		return ErrTypeUnknown, false
	}
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "401") ||
		strings.Contains(msg, "authentication") ||
		strings.Contains(msg, "unauthorized"):
		return ErrTypeAuthFailure, false

	case strings.Contains(msg, "429") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "too many requests"):
		return ErrTypeRateLimit, true

	case strings.Contains(msg, "context_length_exceeded") ||
		strings.Contains(msg, "context window") ||
		strings.Contains(msg, "maximum context length") ||
		strings.Contains(msg, "too many tokens"):
		return ErrTypeContextWindow, false

	case strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context deadline"):
		return ErrTypeTimeout, true

	case strings.Contains(msg, "503") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "grafana-llm") ||
		strings.Contains(msg, "llm unavailable"):
		return ErrTypeLLMUnavailable, true

	case strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "datasource"):
		return ErrTypeDatasourceUnreachable, true

	case strings.Contains(msg, "query") && (strings.Contains(msg, "invalid") || strings.Contains(msg, "validation")):
		return ErrTypeQueryInvalid, false

	default:
		return ErrTypeUnknown, false
	}
}

func sendErrorResponse(w http.ResponseWriter, logMsg string, err error, statusCode int) {
	backend.Logger.Error(logMsg, "error", err)

	var clientMsg string
	switch statusCode {
	case http.StatusBadRequest:
		clientMsg = logMsg
	case http.StatusUnauthorized:
		clientMsg = "authentication required"
	case http.StatusForbidden:
		clientMsg = logMsg
	case http.StatusNotFound:
		clientMsg = logMsg
	case http.StatusTooManyRequests:
		clientMsg = logMsg
	case http.StatusNotImplemented:
		clientMsg = logMsg
	case http.StatusInternalServerError:
		clientMsg = "internal server error"
	default:
		clientMsg = "request failed"
	}

	http.Error(w, clientMsg, statusCode)
}
