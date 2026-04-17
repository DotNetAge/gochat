package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProvider_GetProviderName(t *testing.T) {
	p := NewGeminiProvider("client", "secret", "http://callback", ":8080")
	assert.Equal(t, "Gemini", p.GetProviderName())
}

func TestGeminiProvider_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-refresh", r.Form.Get("refresh_token"))

		resp := struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		}{
			AccessToken: "new-access",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("client", "secret", "http://callback", ":8080")
	p.TokenURL = server.URL

	token, err := p.RefreshToken("old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "new-access", token.Access)
	assert.Equal(t, "old-refresh", token.Refresh)
}

func TestGeminiProvider_ExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "test-code", r.Form.Get("code"))

		resp := struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("client", "secret", "http://callback", ":8080")
	p.TokenURL = server.URL

	token, err := p.exchangeCodeForToken("test-code")
	require.NoError(t, err)
	assert.Equal(t, "access", token.Access)
	assert.Equal(t, "refresh", token.Refresh)
}
