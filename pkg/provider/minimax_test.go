package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMiniMaxProvider(t *testing.T) {
	region := "cn"

	provider := NewMiniMaxProvider(region)

	assert.NotNil(t, provider)
	assert.Equal(t, region, provider.Region)
}

func TestMiniMaxProvider_GetProviderName(t *testing.T) {
	provider := NewMiniMaxProvider("cn")
	assert.Equal(t, "MiniMax", provider.GetProviderName())
}
