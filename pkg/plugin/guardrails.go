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

func ClampTimeRange(from, to string, maxHours int) (string, string, bool) {


	return from, to, false
}
