package core

import (
	"os"
	"strings"
)

// GetAPIKey retrieves API key from environment variable if available
func GetAPIKey(key string) string {
	// First try to get from environment variable
	if envKey := os.Getenv(strings.ToUpper(strings.ReplaceAll(key, "-", "_"))); envKey != "" {
		return envKey
	}
	return key
}

// GetEnv gets an environment variable with a default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
