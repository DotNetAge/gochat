package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewQwenProvider(t *testing.T) {
	provider := NewQwenProvider()

	assert.NotNil(t, provider)
}

func TestQwenProvider_GetProviderName(t *testing.T) {
	provider := NewQwenProvider()
	assert.Equal(t, "Qwen", provider.GetProviderName())
}
