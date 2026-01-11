package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHashUserLoginDeterminism verifies that the same username always produces the same hash
func TestHashUserLoginDeterminism(t *testing.T) {
	tests := []struct {
		name  string
		login string
	}{
		{"simple username", "alice"},
		{"email address", "user@example.com"},
		{"uuid format", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
		{"username with spaces", "john doe"},
		{"username with special chars", "user+test@example.com"},
		{"unicode username", "用户名"},
		{"long username", string(make([]byte, 1000))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashUserLogin(tt.login)
			hash2 := hashUserLogin(tt.login)
			assert.Equal(t, hash1, hash2, "Hash should be deterministic for same input")
		})
	}
}

// TestHashUserLoginUniqueness verifies that different usernames produce different hashes
func TestHashUserLoginUniqueness(t *testing.T) {
	tests := []struct {
		login1 string
		login2 string
	}{
		{"alice", "bob"},
		{"admin", "administrator"},
		{"user1", "user2"},
		{"test@example.com", "test@example.org"},
		{"john", "John"}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s vs %s", tt.login1, tt.login2), func(t *testing.T) {
			hash1 := hashUserLogin(tt.login1)
			hash2 := hashUserLogin(tt.login2)
			assert.NotEqual(t, hash1, hash2, "Different usernames should produce different hashes")
		})
	}
}

// TestHashUserLoginNoCollisions tests collision resistance with 10,000 sample usernames
func TestHashUserLoginNoCollisions(t *testing.T) {
	sampleSize := 10000
	hashes := make(map[int64]string, sampleSize)

	for i := 0; i < sampleSize; i++ {
		login := fmt.Sprintf("user%d@example.com", i)
		hash := hashUserLogin(login)

		if existingLogin, exists := hashes[hash]; exists {
			t.Fatalf("Collision detected: '%s' and '%s' both hash to %d",
				login, existingLogin, hash)
		}

		hashes[hash] = login
	}

	t.Logf("Successfully generated %d unique hashes with no collisions", sampleSize)
}

// TestHashUserLoginDistribution verifies uniform distribution of hash values
func TestHashUserLoginDistribution(t *testing.T) {
	const (
		sampleSize = 10000
		numBuckets = 100
	)

	buckets := make([]int, numBuckets)

	for i := 0; i < sampleSize; i++ {
		login := fmt.Sprintf("user%d", i)
		hash := hashUserLogin(login)

		// Map hash to bucket (handle negative int64)
		bucket := int(hash % int64(numBuckets))
		if bucket < 0 {
			bucket = -bucket
		}
		buckets[bucket]++
	}

	expectedPerBucket := sampleSize / numBuckets
	// Allow 30% variance for statistical fluctuation
	minExpected := int(float64(expectedPerBucket) * 0.7)
	maxExpected := int(float64(expectedPerBucket) * 1.3)

	for i, count := range buckets {
		assert.GreaterOrEqual(t, count, minExpected,
			"Bucket %d has too few values: got %d, want >= %d", i, count, minExpected)
		assert.LessOrEqual(t, count, maxExpected,
			"Bucket %d has too many values: got %d, want <= %d", i, count, maxExpected)
	}

	t.Logf("Distribution: min=%d, max=%d, expected=%d",
		minBucket(buckets), maxBucket(buckets), expectedPerBucket)
}

// TestHashUserLoginEdgeCases tests edge cases and boundary conditions
func TestHashUserLoginEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		login string
	}{
		{"empty string", ""},
		{"single character", "a"},
		{"whitespace only", "   "},
		{"newline", "user\n"},
		{"null byte", "user\x00"},
		{"very long (10KB)", string(make([]byte, 10000))},
		{"unicode emoji", "user🚀"},
		{"mixed unicode", "user用户名🚀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			hash := hashUserLogin(tt.login)
			// Verify determinism
			hash2 := hashUserLogin(tt.login)
			assert.Equal(t, hash, hash2, "Hash should be deterministic even for edge cases")
		})
	}
}

// TestHashUserLoginNonZero verifies hash is non-zero for non-empty strings
func TestHashUserLoginNonZero(t *testing.T) {
	tests := []string{"admin", "user", "test", "alice", "bob"}

	for _, login := range tests {
		t.Run(login, func(t *testing.T) {
			hash := hashUserLogin(login)
			// SHA-256 of non-empty string should never be zero
			assert.NotEqual(t, int64(0), hash,
				"Hash of non-empty string should not be zero")
		})
	}
}

// BenchmarkHashUserLogin measures performance
func BenchmarkHashUserLogin(b *testing.B) {
	tests := []struct {
		name  string
		login string
	}{
		{"short", "alice"},
		{"email", "user@example.com"},
		{"long", string(make([]byte, 1000))},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = hashUserLogin(tt.login)
			}
		})
	}
}

// Helper functions
func minBucket(buckets []int) int {
	min := buckets[0]
	for _, count := range buckets {
		if count < min {
			min = count
		}
	}
	return min
}

func maxBucket(buckets []int) int {
	max := buckets[0]
	for _, count := range buckets {
		if count > max {
			max = count
		}
	}
	return max
}
