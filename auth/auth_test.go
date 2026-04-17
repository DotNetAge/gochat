package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuthProvider 模拟认证提供者
type mockAuthProvider struct {
	name         string
	authToken    *OAuthToken
	authErr      error
	refreshToken *OAuthToken
	refreshErr   error
}

func (m *mockAuthProvider) GetProviderName() string {
	return m.name
}

func (m *mockAuthProvider) Authenticate() (*OAuthToken, error) {
	return m.authToken, m.authErr
}

func (m *mockAuthProvider) RefreshToken(refreshToken string) (*OAuthToken, error) {
	return m.refreshToken, m.refreshErr
}

// mockTokenStore 模拟令牌存储
type mockTokenStore struct {
	token   *OAuthToken
	saveErr error
	loadErr error
}

func (m *mockTokenStore) Save(token *OAuthToken) error {
	m.token = token
	return m.saveErr
}

func (m *mockTokenStore) Load() (*OAuthToken, error) {
	return m.token, m.loadErr
}

func TestAuthManager_Login(t *testing.T) {
	// 测试正常登录
	token := &OAuthToken{
		Access:  "test-token",
		Refresh: "test-refresh",
		Expires: time.Now().Add(1 * time.Hour).UnixMilli(),
	}

	provider := &mockAuthProvider{
		name:      "test-provider",
		authToken: token,
	}

	store := &mockTokenStore{}

	manager := NewAuthManagerWithStore(provider, store)
	err := manager.Login()
	require.NoError(t, err)

	assert.Equal(t, token, store.token)

	// 测试登录失败
	provider.authErr = assert.AnError
	manager2 := NewAuthManagerWithStore(provider, store)
	err = manager2.Login()
	assert.Error(t, err)
}

func TestAuthManager_LoadToken(t *testing.T) {
	// 测试加载令牌
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
	require.NoError(t, err)

	// 测试加载失败
	store.loadErr = assert.AnError
	manager2 := NewAuthManagerWithStore(provider, store)
	err = manager2.LoadToken()
	assert.Error(t, err)

	// 测试没有存储
	manager3 := NewAuthManagerWithStore(provider, nil)
	err = manager3.LoadToken()
	assert.Error(t, err)
}

func TestAuthManager_GetToken(t *testing.T) {
	// 测试获取有效令牌
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
	require.NoError(t, err)

	tokenResult, err := manager.GetToken()
	require.NoError(t, err)
	assert.Equal(t, token, tokenResult)

	// 测试令牌过期自动刷新
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

	store2 := &mockTokenStore{
		token: expiredToken,
	}

	provider2 := &mockAuthProvider{
		name:         "test-provider",
		refreshToken: newToken,
	}

	manager2 := NewAuthManagerWithStore(provider2, store2)
	tokenResult2, err := manager2.GetToken()
	require.NoError(t, err)
	assert.Equal(t, newToken, tokenResult2)
	assert.Equal(t, newToken, store2.token)

	// 测试刷新令牌失败
	expiredToken2 := &OAuthToken{
		Access:  "expired-token-2",
		Refresh: "test-refresh-2",
		Expires: time.Now().Add(-1 * time.Hour).UnixMilli(),
	}

	store3 := &mockTokenStore{
		token: expiredToken2,
	}

	provider3 := &mockAuthProvider{
		name:       "test-provider",
		refreshErr: assert.AnError,
	}

	manager3 := NewAuthManagerWithStore(provider3, store3)
	_, err = manager3.GetToken()
	assert.Error(t, err)

	// 测试没有令牌
	store4 := &mockTokenStore{
		loadErr: assert.AnError,
	}

	manager4 := NewAuthManagerWithStore(provider, store4)
	_, err = manager4.GetToken()
	assert.Error(t, err)
}

func TestAuthManager_isTokenExpired(t *testing.T) {
	// 测试令牌过期
	expiredToken := &OAuthToken{
		Expires: time.Now().Add(-1 * time.Hour).UnixMilli(),
	}

	provider := &mockAuthProvider{name: "test"}
	manager := NewAuthManager(provider, "")
	manager.token = expiredToken

	assert.True(t, manager.isTokenExpired(expiredToken))

	// 测试令牌有效
	validToken := &OAuthToken{
		Expires: time.Now().Add(1 * time.Hour).UnixMilli(),
	}

	assert.False(t, manager.isTokenExpired(validToken))
}
