package core

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableError(t *testing.T) {
	// 测试可重试的错误
	retryableErrors := []string{
		"Rate limit exceeded",
		"Request timeout",
		"Temporary error",
		"Connection refused",
		"Service unavailable",
		"Deadline exceeded",
		"Server error",
		"Connection error",
	}

	for _, errMsg := range retryableErrors {
		err := errors.New(errMsg)
		assert.True(t, IsRetryableError(err), "Error should be retryable: %s", errMsg)
	}

	// 测试不可重试的错误
	nonRetryableErrors := []string{
		"Invalid API key",
		"Not found",
		"Bad request",
		"Unauthorized",
	}

	for _, errMsg := range nonRetryableErrors {
		err := errors.New(errMsg)
		assert.False(t, IsRetryableError(err), "Error should not be retryable: %s", errMsg)
	}

	// 测试 nil 错误
	assert.False(t, IsRetryableError(nil))
}

func TestExponentialBackoff(t *testing.T) {
	baseDelay := 100 * time.Millisecond

	// 测试不同尝试次数的退避时间
	for attempt := 0; attempt < 5; attempt++ {
		delay := ExponentialBackoff(attempt, baseDelay)
		assert.Greater(t, delay, 0*time.Millisecond)
		assert.LessOrEqual(t, delay, 75*time.Second) // 最大延迟 + 最大抖动
	}

	// 测试最大延迟
	largeAttempt := 10
	delay := ExponentialBackoff(largeAttempt, baseDelay)
	assert.LessOrEqual(t, delay, 75*time.Second)
	assert.GreaterOrEqual(t, delay, 60*time.Second)
}
