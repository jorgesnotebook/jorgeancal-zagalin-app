package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Guardrails provides safety mechanisms for LLM requests
type Guardrails struct {
	// Rate limiting
	rateLimiter *RateLimiter

	// Audit logger
	auditLog *AuditLogger
}

// NewGuardrails creates a new guardrails instance
func NewGuardrails(maxRequestsPerMinute int) *Guardrails {
	return &Guardrails{
		rateLimiter: NewRateLimiter(maxRequestsPerMinute),
		auditLog:    NewAuditLogger(),
	}
}

// CheckRequest validates a request against guardrails
func (g *Guardrails) CheckRequest(ctx context.Context, userID string, messages []map[string]interface{}) error {
	// Check rate limit
	if !g.rateLimiter.Allow(userID) {
		backend.Logger.Warn("Rate limit exceeded", "user", userID)
		return fmt.Errorf("rate limit exceeded: too many requests")
	}

	// Validate time ranges in messages (look for suspicious patterns)
	for _, msg := range messages {
		if content, ok := msg["content"].(string); ok {
			if err := g.validateContent(content); err != nil {
				return err
			}
		}
	}

	// Log the request for audit
	g.auditLog.LogRequest(userID, len(messages))

	return nil
}

// validateContent checks message content for unsafe patterns
func (g *Guardrails) validateContent(content string) error {
	// Check for excessively long time ranges in queries
	// This is a simple heuristic - you can expand this
	if len(content) > 50000 {
		return fmt.Errorf("message content too large (max 50KB)")
	}

	return nil
}

// LogResponse logs the response for audit
func (g *Guardrails) LogResponse(userID string, tokens int, cost float64, latency time.Duration) {
	g.auditLog.LogResponse(userID, tokens, cost, latency)
}

// RateLimiter implements a simple token bucket rate limiter per user
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

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		buckets:      make(map[string]*tokenBucket),
		maxPerMinute: maxPerMinute,
		cleanupDone:  make(chan bool),
	}

	// Cleanup old buckets every 5 minutes
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed for the given user
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[userID]

	if !exists {
		// Create new bucket
		bucket = &tokenBucket{
			tokens:     rl.maxPerMinute - 1,
			lastRefill: now,
		}
		rl.buckets[userID] = bucket
		return true
	}

	// Refill tokens based on time passed
	timePassed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(timePassed.Minutes() * float64(rl.maxPerMinute))

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.maxPerMinute {
			bucket.tokens = rl.maxPerMinute
		}
		bucket.lastRefill = now
	}

	// Check if we have tokens available
	if bucket.tokens <= 0 {
		return false
	}

	bucket.tokens--
	return true
}

// cleanup removes old buckets
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

// Stop stops the rate limiter cleanup
func (rl *RateLimiter) Stop() {
	rl.cleanupTicker.Stop()
	rl.cleanupDone <- true
}

// AuditLogger logs all LLM requests and responses for audit purposes
type AuditLogger struct {
	mu sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// LogRequest logs an incoming request
func (al *AuditLogger) LogRequest(userID string, messageCount int) {
	al.mu.Lock()
	defer al.mu.Unlock()

	backend.Logger.Info("LLM request",
		"user", userID,
		"messages", messageCount,
		"timestamp", time.Now().UTC(),
	)
}

// LogResponse logs a response
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

// ClampTimeRange clamps a time range to safe limits
func ClampTimeRange(from, to string, maxHours int) (string, string, bool) {
	// Parse time range
	// This is a simplified version - you'd want to use Grafana's time parsing
	// For now, we'll just return the original values
	// In production, you'd parse and validate the time range

	// TODO: Implement actual time range parsing and clamping
	// For example:
	// - Parse relative times (e.g., "now-6h")
	// - Parse absolute times
	// - Calculate duration
	// - If duration > maxHours, clamp it
	// - Return clamped values and whether clamping occurred

	return from, to, false
}
