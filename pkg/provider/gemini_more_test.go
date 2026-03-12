package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProvider_Authenticate(t *testing.T) {
	// Mock Token Server
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken:  "access",
			RefreshToken: "refresh",
			ExpiresIn:    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	p := NewGeminiProvider("client", "secret", "http://localhost:8081/callback", ":8081")
	p.TokenURL = tokenServer.URL

	// We need to simulate the user clicking the link and the callback being received
	go func() {
		time.Sleep(500 * time.Millisecond)
		http.Get("http://localhost:8081/callback?code=test-code")
	}()

	token, err := p.Authenticate()
	require.NoError(t, err)
	assert.Equal(t, "access", token.Access)
}
