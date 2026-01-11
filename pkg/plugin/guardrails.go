package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Guardrails struct {
	rateLimiter *RateLimiter

	auditLog *AuditLogger
}

func NewGuardrails(maxRequestsPerMinute int) *Guardrails {
	return &Guardrails{
		rateLimiter: NewRateLimiter(maxRequestsPerMinute),
		auditLog:    NewAuditLogger(),
	}
}

func (g *Guardrails) CheckRequest(ctx context.Context, userID string, messages []map[string]interface{}) error {
	if !g.rateLimiter.Allow(userID) {
		backend.Logger.Warn("Rate limit exceeded", "user", userID)
		return fmt.Errorf("rate limit exceeded: too many requests")
	}

	for _, msg := range messages {
		if content, ok := msg["content"].(string); ok {
			if err := g.validateContent(content); err != nil {
				return err
			}
		}
	}

	g.auditLog.LogRequest(userID, len(messages))

	return nil
}

func (g *Guardrails) validateContent(content string) error {
	if len(content) > 50000 {
		return fmt.Errorf("message content too large (max 50KB)")
	}

	return nil
}

func (g *Guardrails) LogResponse(userID string, tokens int, cost float64, latency time.Duration) {
	g.auditLog.LogResponse(userID, tokens, cost, latency)
}

type RateLimiter struct {
	mu            sync.Mutex
	buckets       map[string]*tokenBucket
	maxPerMinute  int
	cleanupTicker *time.Ticker
	cleanupDone   chan bool
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
}

func NewRateLimiter(maxPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		buckets:      make(map[string]*tokenBucket),
		maxPerMinute: maxPerMinute,
		cleanupDone:  make(chan bool),
	}

	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[userID]

	if !exists {
		bucket = &tokenBucket{
			tokens:     rl.maxPerMinute - 1,
			lastRefill: now,
		}
		rl.buckets[userID] = bucket
		return true
	}

	timePassed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(timePassed.Minutes() * float64(rl.maxPerMinute))

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.maxPerMinute {
			bucket.tokens = rl.maxPerMinute
		}
		bucket.lastRefill = now
	}

	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

func (rl *RateLimiter) cleanup() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			for userID, bucket := range rl.buckets {
				if now.Sub(bucket.lastRefill) > 10*time.Minute {
					delete(rl.buckets, userID)
				}
			}
			rl.mu.Unlock()
		case <-rl.cleanupDone:
			return
		}
	}
}

func (rl *RateLimiter) Stop() {
	rl.cleanupTicker.Stop()
	rl.cleanupDone <- true
}

type AuditLogger struct {
	mu sync.Mutex
}

func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

func (al *AuditLogger) LogRequest(userID string, messageCount int) {
	al.mu.Lock()
	defer al.mu.Unlock()

	backend.Logger.Info("LLM request",
		"user", userID,
		"messages", messageCount,
		"timestamp", time.Now().UTC(),
	)
}

func (al *AuditLogger) LogResponse(userID string, tokens int, cost float64, latency time.Duration) {
	al.mu.Lock()
	defer al.mu.Unlock()

	backend.Logger.Info("LLM response",
		"user", userID,
		"tokens", tokens,
		"cost_usd", cost,
		"latency_ms", latency.Milliseconds(),
		"timestamp", time.Now().UTC(),
	)
}

// parseRelativeDuration converts relative duration strings like "5m", "1h", "7d" to time.Duration
func parseRelativeDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	var value int64
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", s)
	}

	switch unit {
	case 's':
		return time.Duration(value) * time.Second, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'M':
		return time.Duration(value) * 30 * 24 * time.Hour, nil // Approximate
	case 'y':
		return time.Duration(value) * 365 * 24 * time.Hour, nil // Approximate
	default:
		return 0, fmt.Errorf("unknown duration unit: %c", unit)
	}
}

// parseTimeValue converts Grafana time string to time.Time
// Supports: "now", "now-5m", RFC3339, Unix milliseconds/seconds
func parseTimeValue(timeStr string) (time.Time, error) {
	now := time.Now()

	// Handle "now" formats
	if timeStr == "now" {
		return now, nil
	}

	if len(timeStr) > 4 && timeStr[:4] == "now-" {
		// Parse "now-5m" style
		duration, err := parseRelativeDuration(timeStr[4:])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative time: %w", err)
		}
		return now.Add(-duration), nil
	}

	if len(timeStr) > 4 && timeStr[:4] == "now+" {
		// Parse "now+5m" style (future)
		duration, err := parseRelativeDuration(timeStr[4:])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid relative time: %w", err)
		}
		return now.Add(duration), nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t, nil
	}

	// Try Unix timestamp (milliseconds or seconds)
	var timestamp int64
	_, err := fmt.Sscanf(timeStr, "%d", &timestamp)
	if err == nil {
		if timestamp > 1e12 { // milliseconds (> year 2001)
			return time.UnixMilli(timestamp), nil
		}
		// seconds
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", timeStr)
}

// formatTimeValue formats time.Time back to string, preserving relative format if possible
func formatTimeValue(t time.Time, originalFormat string) string {
	// If original was "now-*", calculate new relative offset
	if len(originalFormat) > 4 && originalFormat[:4] == "now-" {
		duration := time.Since(t)
		if duration < 0 {
			// Future time, use absolute
			return t.Format(time.RFC3339)
		}

		// Convert to human-readable relative format
		if duration < time.Minute {
			return fmt.Sprintf("now-%ds", int(duration.Seconds()))
		}
		if duration < time.Hour {
			return fmt.Sprintf("now-%dm", int(duration.Minutes()))
		}
		if duration < 24*time.Hour {
			return fmt.Sprintf("now-%dh", int(duration.Hours()))
		}
		return fmt.Sprintf("now-%dd", int(duration.Hours()/24))
	}

	// For "now" or absolute times, use RFC3339
	return t.Format(time.RFC3339)
}

// ClampTimeRange limits the query time range to maxHours.
// Returns clamped from, to, and boolean indicating if clamping occurred, plus error if parsing fails.
func ClampTimeRange(from, to string, maxHours int) (string, string, bool, error) {
	// 1. Skip if disabled
	if maxHours <= 0 {
		return from, to, false, nil
	}

	// 2. Parse timestamps
	fromTime, err := parseTimeValue(from)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid from time: %w", err)
	}

	toTime, err := parseTimeValue(to)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid to time: %w", err)
	}

	// 3. Calculate duration
	duration := toTime.Sub(fromTime)
	if duration < 0 {
		return "", "", false, fmt.Errorf("from time (%s) is after to time (%s)", from, to)
	}

	// 4. Check if exceeds limit
	maxDuration := time.Duration(maxHours) * time.Hour
	// Use a small epsilon to handle floating point precision issues
	epsilon := time.Millisecond
	if duration <= maxDuration+epsilon {
		return from, to, false, nil // No clamping needed
	}

	// 5. Clamp from to (to - maxHours)
	clampedFromTime := toTime.Add(-maxDuration)

	// 6. Format back to original format
	clampedFrom := formatTimeValue(clampedFromTime, from)

	return clampedFrom, to, true, nil
}

// logTimeRangeClamping logs when time range is clamped for audit purposes
func logTimeRangeClamping(userID, orgID, datasourceUID, originalFrom, clampedFrom, to string, maxHours int) {
	backend.Logger.Info("Time range clamped",
		"event", "time_range_clamped",
		"timestamp", time.Now().UTC().Format(time.RFC3339),
		"user_id", userID,
		"org_id", orgID,
		"datasource_uid", datasourceUID,
		"original_from", originalFrom,
		"clamped_from", clampedFrom,
		"to", to,
		"max_hours", maxHours,
	)
}
