package plugin

import (
	"crypto/sha256"
	"fmt"
)

// hashQuery creates a correlation hash for query logging
// This allows correlating logs without exposing sensitive query content
func hashQuery(query string) string {
	if query == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", hash[:8]) // First 16 chars of hash
}

// truncateForLog safely truncates strings for logging
// Used to preview content without exposing full sensitive data
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
