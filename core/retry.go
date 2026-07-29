package core

import (
	"errors"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// IsRetryableError checks if an error is retryable based on structured
// error type, HTTP status code, and network error classification.
//
// Retryability is determined by (in priority order):
//  1. *Error with Type == ErrorTypeRateLimit → always retryable (429)
//  2. *Error with Type == ErrorTypeTimeout → retryable
//  3. *Error with Type == ErrorTypeNetwork → retryable
//  4. *Error with StatusCode in retryable set (429, 408, 500, 502, 503, 504)
//  5. *Error with Type == ErrorTypeValidation → never retryable
//  6. net.Error with Timeout() == true → retryable
//  7. Fallback: string matching for non-*Error types (backward compat)
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 1. 优先用结构化 *Error 判定
	var apiErr *Error
	if errors.As(err, &apiErr) {
		// 显式不可重试的类型
		if apiErr.Type == ErrorTypeValidation {
			return false
		}
		// 显式可重试的类型
		if apiErr.Type == ErrorTypeRateLimit ||
			apiErr.Type == ErrorTypeTimeout ||
			apiErr.Type == ErrorTypeNetwork {
			return true
		}
		// 用状态码判定（覆盖 5xx 等未细分但应重试的情况）
		if isRetryableStatusCode(apiErr.StatusCode) {
			return true
		}
		// 其它 API 错误（如 400/401/403/404）不可重试
		if apiErr.Type == ErrorTypeAPI {
			return false
		}
	}

	// 2. net.Error 超时判定
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// 3. 向后兼容：对非 *Error 类型用字符串匹配兜底
	return isRetryableByMessage(err.Error())
}

// isRetryableStatusCode 判定 HTTP 状态码是否可重试。
// 可重试：429（限流）、408（请求超时）、500/502/503/504（服务端临时故障）。
// 不可重试：400/401/403/404/422 等 4xx 客户端错误。
func isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,     // 429
		http.StatusRequestTimeout,        // 408
		http.StatusInternalServerError,   // 500
		http.StatusBadGateway,             // 502
		http.StatusServiceUnavailable,     // 503
		http.StatusGatewayTimeout:         // 504
		return true
	}
	return false
}

// isRetryableByMessage 是对非 *Error 类型的向后兼容兜底。
// 仅检查常见的网络/超时错误关键词，不再覆盖 provider 特定措辞
// （那些应该通过 *Error.StatusCode 或 *Error.Type 判定）。
func isRetryableByMessage(errStr string) bool {
	lower := strings.ToLower(errStr)
	retryableKeywords := []string{
		"rate limit",
		"timeout",
		"temporary",
		"connection refused",
		"connection reset",
		"connection error",
		"service unavailable",
		"deadline exceeded",
		"server error",
		"eof", // 流式连接意外关闭
	}
	for _, kw := range retryableKeywords {
		if strings.Contains(lower, kw) {
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
