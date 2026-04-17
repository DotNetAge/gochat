package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMiniMaxProvider(t *testing.T) {
	region := "cn"
	provider := NewMiniMaxProvider(region)
	assert.NotNil(t, provider)
	assert.Equal(t, region, provider.Region)
	assert.Equal(t, "https://api.minimaxi.com", provider.BaseURL)
	assert.Equal(t, "https://api.minimaxi.com/oauth/code", provider.OAuthCodeURL)
	assert.Equal(t, "https://api.minimaxi.com/oauth/token", provider.OAuthTokenURL)

	providerIntl := NewMiniMaxProvider("intl")
	assert.Equal(t, "https://api.minimax.io", providerIntl.BaseURL)
}

func TestMiniMaxProvider_GetProviderName(t *testing.T) {
	provider := NewMiniMaxProvider("cn")
	assert.Equal(t, "MiniMax", provider.GetProviderName())
}

func TestMiniMaxProvider_RequestOAuthCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		// We need to capture the state from the request to return it in the response
		r.ParseForm()
		state := r.Form.Get("state")

		resp := MiniMaxOAuthAuthorization{
			UserCode:        "test-user-code",
			VerificationURI: "http://example.com/verify",
			ExpiredIn:       int(time.Now().Add(time.Hour).UnixMilli()),
			State:           state,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthCodeURL = server.URL

	auth, verifier, state, err := p.RequestOAuthCode()
	require.NoError(t, err)
	assert.NotNil(t, auth)
	assert.Equal(t, "test-user-code", auth.UserCode)
	assert.NotEmpty(t, verifier)
	assert.Equal(t, state, auth.State)
}

func TestMiniMaxProvider_PollOAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "test-user-code", r.Form.Get("user_code"))

		resp := struct {
			Status       string `json:"status"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiredIn    int64  `json:"expired_in"`
		}{
			Status:       "success",
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			ExpiredIn:    time.Now().Add(time.Hour).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthTokenURL = server.URL

	status, token, errMsg := p.PollOAuthToken("test-user-code", "test-verifier")
	assert.Equal(t, "success", status)
	assert.Empty(t, errMsg)
	require.NotNil(t, token)
	assert.Equal(t, "test-access", token.Access)
	assert.Equal(t, "test-refresh", token.Refresh)
}

func TestMiniMaxProvider_RefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-refresh", r.Form.Get("refresh_token"))

		resp := struct {
			Status       string `json:"status"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiredIn    int64  `json:"expired_in"`
		}{
			Status:       "success",
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiredIn:    time.Now().Add(time.Hour).UnixMilli(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthTokenURL = server.URL

	token, err := p.RefreshToken("old-refresh")
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "new-access", token.Access)
	assert.Equal(t, "new-refresh", token.Refresh)
}

func TestMiniMaxProvider_Authenticate(t *testing.T) {
	// Mock both Code and Token endpoints
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		state := r.Form.Get("state")
		resp := MiniMaxOAuthAuthorization{
			UserCode:        "test-user-code",
			VerificationURI: "http://example.com/verify",
			ExpiredIn:       int(time.Now().Add(time.Hour).UnixMilli()),
			State:           state,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer codeServer.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Status       string `json:"status"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiredIn    int64  `json:"expired_in"`
		}{
			Status:       "success",
			AccessToken:  "test-access",
			RefreshToken: "test-refresh",
			ExpiredIn:    time.Now().Add(time.Hour).UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer tokenServer.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthCodeURL = codeServer.URL
	p.OAuthTokenURL = tokenServer.URL

	token, err := p.Authenticate()
	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, "test-access", token.Access)
}

func TestMiniMaxProvider_PollOAuthToken_Pending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Status string `json:"status"`
		}{
			Status: "pending",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthTokenURL = server.URL

	status, token, errMsg := p.PollOAuthToken("test-user-code", "test-verifier")
	assert.Equal(t, "pending", status)
	assert.Nil(t, token)
	assert.Equal(t, "current user code is not authorized", errMsg)
}

func TestMiniMaxProvider_PollOAuthToken_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := struct {
			BaseResp struct {
				StatusMsg string `json:"status_msg"`
			} `json:"base_resp"`
		}{}
		resp.BaseResp.StatusMsg = "Invalid request"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewMiniMaxProvider("cn")
	p.OAuthTokenURL = server.URL

	status, token, errMsg := p.PollOAuthToken("test-user-code", "test-verifier")
	assert.Equal(t, "error", status)
	assert.Nil(t, token)
	assert.Equal(t, "Invalid request", errMsg)
}
