package plugin

import (
	"context"
	"crypto/sha256"
	"fmt"
)

type contextKey string

const correlationIDKey contextKey = "correlationId"

func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

func CorrelationIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

func hashQuery(query string) string {
	if query == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", hash[:8]) 
}

func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
