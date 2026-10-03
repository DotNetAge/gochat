package auth

import (
	"net/http"
	"time"
)

// geminiProviderScope Gemini 授权范围（照抄原实现）
const geminiProviderScope = "https://www.googleapis.com/auth/generative-language.retriever"

// GeminiProvider Gemini 提供商——标准 OAuth2 授权码流的薄包装，
// 流程逻辑已迁移至 ConfigDrivenProvider，本结构体仅装配常量参数转调。
type GeminiProvider struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string // 例如: http://localhost:8080/oauth2/callback
	ListenAddr   string // 例如: ":8080"
	HTTPClient   *http.Client
	TokenURL     string // For testing override
}

// NewGeminiProvider 创建 Gemini 提供商
func NewGeminiProvider(clientID, clientSecret, callbackURL, listenAddr string) *GeminiProvider {
	return &GeminiProvider{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CallbackURL:  callbackURL,
		ListenAddr:   listenAddr,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		TokenURL:     "https://oauth2.googleapis.com/token",
	}
}

// GetProviderName 获取提供商名称
func (p *GeminiProvider) GetProviderName() string {
	return "Gemini"
}

// providerConfig 从当前字段值装配泛化配置（每次现构造，避免字段覆盖后状态失步）。
// UseState=false 保持既有外部行为（历史授权流程无 state 参数）。
func (p *GeminiProvider) providerConfig() ProviderConfig {
	return ProviderConfig{
		Name:         "Gemini",
		Kind:         KindAuthorizationCode,
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     p.TokenURL,
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Scopes:       []string{geminiProviderScope},
		ExtraParams:  map[string]string{"access_type": "offline", "prompt": "consent"},
		UsePKCE:      false,
		UseState:     false,
		ListenAddr:   p.ListenAddr,
		RedirectURL:  p.CallbackURL,
		HTTPClient:   p.HTTPClient,
	}
}

// Authenticate 执行标准 OAuth2 授权码流程（转调泛化实现）
func (p *GeminiProvider) Authenticate() (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.Authenticate()
}

// exchangeCodeForToken 授权码换取令牌（转调泛化实现）
func (p *GeminiProvider) exchangeCodeForToken(code string) (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.ExchangeAuthorizationCode(code)
}

// RefreshToken 刷新令牌（转调泛化实现）
func (p *GeminiProvider) RefreshToken(refreshToken string) (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.RefreshToken(refreshToken)
}
