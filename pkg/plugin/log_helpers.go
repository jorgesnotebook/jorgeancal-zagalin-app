package plugin

import (
	"crypto/sha256"
	"fmt"
)

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
