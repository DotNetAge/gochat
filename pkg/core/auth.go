package core

// AuthManager 认证管理器
type AuthManager struct {
	provider interface {
		GetProviderName() string
		Authenticate() (*OAuthToken, error)
		RefreshToken(refreshToken string) (*OAuthToken, error)
	}
	token       *OAuthToken
	filename    string
	tokenHelper *TokenHelper
}

// NewAuthManager 创建认证管理器
func NewAuthManager(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, filename string) *AuthManager {
	return &AuthManager{
		provider:    provider,
		filename:    filename,
		tokenHelper: &TokenHelper{},
	}
}

// Login 执行登录
func (m *AuthManager) Login() error {
	token, err := m.provider.Authenticate()
	if err != nil {
		return err
	}

	m.token = token
	return m.tokenHelper.SaveToken(token, m.filename)
}

// LoadToken 加载令牌
func (m *AuthManager) LoadToken() error {
	token, err := m.tokenHelper.LoadToken(m.filename)
	if err != nil {
		return err
	}

	m.token = token
	return nil
}

// GetToken 获取令牌（自动刷新）
func (m *AuthManager) GetToken() (*OAuthToken, error) {
	if m.token == nil {
		if err := m.LoadToken(); err != nil {
			return nil, err
		}
	}

	if m.tokenHelper.IsTokenExpired(m.token) {
		newToken, err := m.provider.RefreshToken(m.token.Refresh)
		if err != nil {
			return nil, err
		}
		m.token = newToken
		if err := m.tokenHelper.SaveToken(newToken, m.filename); err != nil {
			return nil, err
		}
	}

	return m.token, nil
}
