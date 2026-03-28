package core

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthManager_GetToken_ConcurrentAccess(t *testing.T) {
	token := &OAuthToken{
		Access:  "test-token",
		Refresh: "test-refresh",
		Expires: time.Now().Add(1 * time.Hour).UnixMilli(),
	}

	store := &mockTokenStore{
		token: token,
	}

	provider := &mockAuthProvider{
		name: "test-provider",
	}

	manager := NewAuthManagerWithStore(provider, store)
	err := manager.LoadToken()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	errors := make([]error, 0, 10)
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetToken()
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, err := range errors {
		assert.NoError(t, err)
	}
}

func TestAuthManager_GetToken_ConcurrentRefresh(t *testing.T) {
	expiredToken := &OAuthToken{
		Access:  "expired-token",
		Refresh: "test-refresh",
		Expires: time.Now().Add(-1 * time.Hour).UnixMilli(),
	}

	newToken := &OAuthToken{
		Access:  "new-token",
		Refresh: "new-refresh",
		Expires: time.Now().Add(1 * time.Hour).UnixMilli(),
	}

	store := &mockTokenStore{
		token: expiredToken,
	}

	refreshCalled := make(chan struct{}, 10)
	provider := &mockAuthProvider{
		name:         "test-provider",
		refreshToken: newToken,
	}

	manager := NewAuthManagerWithStore(provider, store)

	var wg sync.WaitGroup
	results := make([]*OAuthToken, 0, 5)
	var resultsMu sync.Mutex

	originalRefresh := provider.RefreshToken

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			refreshCalled <- struct{}{}
			token, err := manager.GetToken()
			if err == nil {
				resultsMu.Lock()
				results = append(results, token)
				resultsMu.Unlock()
			}
		}()
	}

	go func() {
		for range refreshCalled {
			originalRefresh("")
		}
	}()

	wg.Wait()

	assert.NotEmpty(t, results)
	for _, token := range results {
		assert.Equal(t, "new-token", token.Access)
	}
}

func TestAuthManager_GetToken_NilToken(t *testing.T) {
	store := &mockTokenStore{}
	provider := &mockAuthProvider{name: "test-provider"}
	manager := NewAuthManagerWithStore(provider, store)

	_, err := manager.GetToken()
	assert.Error(t, err)
}

func TestAuthManager_SetExpiryBuffer(t *testing.T) {
	provider := &mockAuthProvider{name: "test-provider"}
	manager := NewAuthManager(provider, "")

	manager.SetExpiryBuffer(2 * time.Minute)
	assert.NotNil(t, manager)
}

func TestAuthManager_ExpiryBuffer(t *testing.T) {
	provider := &mockAuthProvider{name: "test-provider"}
	manager := NewAuthManager(provider, "")

	manager.SetExpiryBuffer(30 * time.Second)

	token := &OAuthToken{
		Access:  "test-token",
		Refresh: "test-refresh",
		Expires: time.Now().Add(20 * time.Second).UnixMilli(),
	}
	manager.token = token

	assert.True(t, manager.isTokenExpired(token))

	token2 := &OAuthToken{
		Access:  "test-token",
		Refresh: "test-refresh",
		Expires: time.Now().Add(120 * time.Second).UnixMilli(),
	}
	assert.False(t, manager.isTokenExpired(token2))
}
