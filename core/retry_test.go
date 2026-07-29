package core

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsRetryableError(t *testing.T) {
	// 测试可重试的错误（字符串匹配兜底）
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

// TestIsRetryableError_Structured 验证结构化 *Error 的重试判定。
// 这是 P0-3 改造的核心：按错误类型和状态码判定，而非字符串匹配。
func TestIsRetryableError_Structured(t *testing.T) {
	// 按错误类型判定
	assert.True(t, IsRetryableError(NewError(ErrorTypeRateLimit, "429", nil)), "RateLimit 类型应可重试")
	assert.True(t, IsRetryableError(NewError(ErrorTypeTimeout, "408", nil)), "Timeout 类型应可重试")
	assert.True(t, IsRetryableError(NewError(ErrorTypeNetwork, "conn refused", nil)), "Network 类型应可重试")
	assert.False(t, IsRetryableError(NewError(ErrorTypeValidation, "bad input", nil)), "Validation 类型不应可重试")

	// 按状态码判定（5xx 应可重试，4xx 不可重试）
	assert.True(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 429}), "429 应可重试")
	assert.True(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 500}), "500 应可重试")
	assert.True(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 503}), "503 应可重试")
	assert.False(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 400}), "400 不应可重试")
	assert.False(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 401}), "401 不应可重试")
	assert.False(t, IsRetryableError(&Error{Type: ErrorTypeAPI, StatusCode: 404}), "404 不应可重试")

	// NewAPIErrorFromResponse 应正确分类状态码
	rateLimitErr := NewAPIErrorFromResponse(429, []byte(`{"error":{"message":"slow down"}}`))
	assert.Equal(t, ErrorTypeRateLimit, rateLimitErr.Type, "429 应分类为 RateLimit")
	assert.True(t, IsRetryableError(rateLimitErr))

	serverErr := NewAPIErrorFromResponse(503, []byte(`{"error":{"message":"unavailable"}}`))
	assert.True(t, IsRetryableError(serverErr), "503 应可重试")

	clientErr := NewAPIErrorFromResponse(400, []byte(`{"error":{"message":"bad request"}}`))
	assert.False(t, IsRetryableError(clientErr), "400 不应可重试")
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
