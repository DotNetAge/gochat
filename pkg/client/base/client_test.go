package base

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// 测试默认值
	config := Config{}
	client := New(config)

	assert.Equal(t, 30*time.Second, client.config.Timeout)
	assert.Equal(t, 3, client.config.MaxRetries)
	assert.Equal(t, 0.7, client.config.Temperature)

	// 测试自定义值
	customConfig := Config{
		Timeout:     60 * time.Second,
		MaxRetries:  5,
		Temperature: 0.5,
	}
	client2 := New(customConfig)

	assert.Equal(t, 60*time.Second, client2.config.Timeout)
	assert.Equal(t, 5, client2.config.MaxRetries)
	assert.Equal(t, 0.5, client2.config.Temperature)
}

func TestClient_Retry_Success(t *testing.T) {
	client := New(Config{MaxRetries: 3})

	ctx := context.Background()
	var callCount int

	err := client.Retry(ctx, func() error {
		callCount++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestClient_Retry_Failure(t *testing.T) {
	client := New(Config{MaxRetries: 2})

	ctx := context.Background()
	var callCount int
	testErr := errors.New("test error")

	err := client.Retry(ctx, func() error {
		callCount++
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, 1, callCount) // 只尝试一次，因为错误不可重试
}

func TestClient_Retry_Retryable(t *testing.T) {
	client := New(Config{MaxRetries: 2})

	ctx := context.Background()
	var callCount int
	retryableErr := errors.New("rate limit exceeded")

	err := client.Retry(ctx, func() error {
		callCount++
		return retryableErr
	})

	assert.Error(t, err)
	assert.Equal(t, retryableErr, err)
	assert.Equal(t, 3, callCount) // 尝试 3 次（0, 1, 2）
}

func TestClient_Retry_ContextCanceled(t *testing.T) {
	client := New(Config{MaxRetries: 3})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消上下文

	var callCount int
	err := client.Retry(ctx, func() error {
		callCount++
		return errors.New("rate limit exceeded")
	})

	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 1, callCount)
}

func TestClient_HTTPClient(t *testing.T) {
	client := New(Config{})
	httpClient := client.HTTPClient()
	assert.NotNil(t, httpClient)
}

func TestClient_Config(t *testing.T) {
	expectedConfig := Config{
		APIKey:      "test-key",
		Model:       "gpt-4",
		BaseURL:     "https://api.example.com",
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	client := New(expectedConfig)
	actualConfig := client.Config()

	assert.Equal(t, expectedConfig, actualConfig)
}
