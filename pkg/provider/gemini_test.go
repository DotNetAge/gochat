package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGeminiProvider(t *testing.T) {
	clientID := "test-client-id"
	clientSecret := "test-client-secret"
	callbackURL := "http://localhost:8080/callback"
	listenAddr := ":8080"

	provider := NewGeminiProvider(clientID, clientSecret, callbackURL, listenAddr)

	assert.Equal(t, clientID, provider.ClientID)
	assert.Equal(t, clientSecret, provider.ClientSecret)
	assert.Equal(t, callbackURL, provider.CallbackURL)
	assert.Equal(t, listenAddr, provider.ListenAddr)
	assert.NotNil(t, provider.HTTPClient)
}

func TestGeminiProvider_GetProviderName(t *testing.T) {
	provider := NewGeminiProvider("", "", "", "")
	assert.Equal(t, "Gemini", provider.GetProviderName())
}
