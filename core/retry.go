package core

import (
	"math/rand"
	"strings"
	"time"
)

// IsRetryableError checks if an error is retryable based on common patterns
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"rate limit",
		"timeout",
		"temporary",
		"connection refused",
		"service unavailable",
		"deadline exceeded",
		"server error",
		"connection",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// ExponentialBackoff calculates the backoff time with jitter
func ExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	maxDelay := 60 * time.Second
	delay := baseDelay * (1 << uint(attempt))
	if delay > maxDelay {
		delay = maxDelay
	}
	// Add jitter to avoid thundering herd
	jitter := time.Duration(rand.Int63n(int64(delay / 4)))
	return delay + jitter
}
