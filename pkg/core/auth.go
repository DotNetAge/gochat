package core

import (
	"fmt"
	"time"
)

// AuthManager 认证管理器
type AuthManager struct {
	provider interface {
		GetProviderName() string
		Authenticate() (*OAuthToken, error)
		RefreshToken(refreshToken string) (*OAuthToken, error)
	}
	token *OAuthToken
	store TokenStore
}

// NewAuthManager 创建认证管理器
// 保持向后兼容性，接收 string 并自动转换为 FileTokenStore
func NewAuthManager(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, filename string) *AuthManager {
	return &AuthManager{
		provider: provider,
		store:    NewFileTokenStore(filename),
	}
}

// NewAuthManagerWithStore 接受自定义 TokenStore (如 RedisStore, DBStore 等)
func NewAuthManagerWithStore(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, store TokenStore) *AuthManager {
	return &AuthManager{
		provider: provider,
		store:    store,
	}
}

// Login 执行登录
func (m *AuthManager) Login() error {
	token, err := m.provider.Authenticate()
	if err != nil {
		return err
	}

	m.token = token
	if m.store != nil {
		return m.store.Save(token)
	}
	return nil
}

// LoadToken 加载令牌
func (m *AuthManager) LoadToken() error {
	if m.store == nil {
		return fmt.Errorf("no token store configured")
	}

	token, err := m.store.Load()
	if err != nil {
		return err
	}

	m.token = token
	return nil
}

func (m *AuthManager) isTokenExpired(token *OAuthToken) bool {
	// 提前60秒判断过期，防止网络延迟引起的恰好失效
	return time.Now().UnixMilli() > (token.Expires - 60000)
}

// GetToken 获取令牌，如果过期会自动刷新并保存
func (m *AuthManager) GetToken() (*OAuthToken, error) {
	if m.token == nil {
		if err := m.LoadToken(); err != nil {
			return nil, err
		}
	}

	if m.token == nil {
		return nil, fmt.Errorf("no token available, please login first")
	}

	if m.isTokenExpired(m.token) {
		newToken, err := m.provider.RefreshToken(m.token.Refresh)
		if err != nil {
			return nil, err
		}
		m.token = newToken
		if m.store != nil {
			if err := m.store.Save(newToken); err != nil {
				return nil, err
			}
		}
	}

	return m.token, nil
}
