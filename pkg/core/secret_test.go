package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAPIKey(t *testing.T) {
	// 测试从环境变量获取
	testKey := "test-api-key"
	envKey := "TEST_API_KEY"

	// 设置环境变量
	err := os.Setenv(envKey, testKey)
	assert.NoError(t, err)
	defer os.Unsetenv(envKey)

	// 测试获取环境变量中的值
	result := GetAPIKey("test-api-key")
	assert.Equal(t, testKey, result)

	// 测试当环境变量不存在时返回原始值
	result2 := GetAPIKey("fallback-key")
	assert.Equal(t, "fallback-key", result2)
}

func TestGetEnv(t *testing.T) {
	// 测试获取存在的环境变量
	testValue := "test-value"
	testKey := "TEST_KEY"

	// 设置环境变量
	err := os.Setenv(testKey, testValue)
	assert.NoError(t, err)
	defer os.Unsetenv(testKey)

	result := GetEnv(testKey, "default")
	assert.Equal(t, testValue, result)

	// 测试获取不存在的环境变量，返回默认值
	result2 := GetEnv("NON_EXISTENT_KEY", "default-value")
	assert.Equal(t, "default-value", result2)
}
