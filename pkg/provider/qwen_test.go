package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQwenProvider(t *testing.T) {
	provider := NewQwenProvider()
	assert.NotNil(t, provider)
	assert.Equal(t, "https://chat.qwen.ai/api/v1/oauth2/device/code", provider.DeviceCodeURL)
	assert.Equal(t, "https://chat.qwen.ai/api/v1/oauth2/token", provider.TokenURL)
}

func TestQwenProvider_GetProviderName(t *testing.T) {
	provider := NewQwenProvider()
	assert.Equal(t, "Qwen", provider.GetProviderName())
}

func TestQwenProvider_RequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		resp := QwenDeviceAuthorization{
			DeviceCode:      "test-device-code",
			UserCode:        "test-user-code",
			VerificationURI: "http://example.com/verify",
			ExpiresIn:       3600,
			Interval:        5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewQwenProvider()
	p.DeviceCodeURL = server.URL

	auth, verifier, err := p.RequestDeviceCode()
	require.NoError(t, err)
	assert.NotNil(t, auth)
	assert.Equal(t, "test-device-code", auth.DeviceCode)
	assert.NotEmpty(t, verifier)
}

func TestQwenProvider_PollForToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "test-device-code", r.Form.Get("device_code"))

		resp := struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewQwenProvider()
	p.TokenURL = server.URL

	status, token, slowDown, errorMsg := p.PollForToken("test-device-code", "test-verifier")
	assert.Equal(t, "success", status)
	assert.False(t, slowDown)
	assert.Empty(t, errorMsg)
	require.NotNil(t, token)
	assert.Equal(t, "test-access", token.Access)
}

func TestQwenProvider_PollForToken_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}{
			Error: "authorization_pending",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewQwenProvider()
	p.TokenURL = server.URL

	status, token, slowDown, errorMsg := p.PollForToken("test-device-code", "test-verifier")
	assert.Equal(t, "pending", status)
	assert.Nil(t, token)
	assert.False(t, slowDown)
	assert.Empty(t, errorMsg)
}

func TestQwenProvider_PollForToken_SlowDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := struct {
			Error string `json:"error"`
		}{
			Error: "slow_down",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewQwenProvider()
	p.TokenURL = server.URL

	status, token, slowDown, errorMsg := p.PollForToken("test-device-code", "test-verifier")
	assert.Equal(t, "pending", status)
	assert.Nil(t, token)
	assert.True(t, slowDown)
	assert.Empty(t, errorMsg)
}

func TestQwenProvider_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-refresh", r.Form.Get("refresh_token"))

		resp := struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewQwenProvider()
	p.TokenURL = server.URL

	token, err := p.RefreshToken("old-refresh")
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "new-access", token.Access)
	assert.Equal(t, "new-refresh", token.Refresh)
}

func TestQwenProvider_Authenticate(t *testing.T) {
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := QwenDeviceAuthorization{
			DeviceCode:      "test-device-code",
			UserCode:        "test-user-code",
			VerificationURI: "http://example.com/verify",
			ExpiresIn:       3600,
			Interval:        1, // 1 second
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer codeServer.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}{
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			ExpiresIn:    3600,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	p := NewQwenProvider()
	p.DeviceCodeURL = codeServer.URL
	p.TokenURL = tokenServer.URL

	token, err := p.Authenticate()
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "test-access", token.Access)
}
