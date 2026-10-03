package auth

import (
	"net/http"
	"time"
)

// MiniMax 端点与公开 client_id（照抄原实现）
const (
	// minimaxClientID MiniMax 公开 client_id（PKCE 公共客户端）
	minimaxClientID = "78257093-7e40-4613-99e0-527b14b39113"

	// minimaxBaseURLCN 国内站端点
	minimaxBaseURLCN = "https://api.minimaxi.com"

	// minimaxBaseURLIntl 国际站端点
	minimaxBaseURLIntl = "https://api.minimax.io"
)

// MiniMaxProvider MiniMax 提供商——用户码两步协议的薄包装，
// 流程逻辑已迁移至 ConfigDrivenProvider，本结构体仅装配参数转调。
type MiniMaxProvider struct {
	HTTPClient    *http.Client
	TokenHelper   *TokenHelper
	PKCEHelper    *PKCEHelper
	Region        string
	BaseURL       string
	ClientID      string
	OAuthCodeURL  string // For testing override
	OAuthTokenURL string // For testing override
}

// NewMiniMaxProvider 创建 MiniMax 提供商
func NewMiniMaxProvider(region string) *MiniMaxProvider {
	var baseURL string
	if region == "cn" {
		baseURL = minimaxBaseURLCN
	} else {
		baseURL = minimaxBaseURLIntl
	}

	return &MiniMaxProvider{
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
		TokenHelper:   &TokenHelper{},
		PKCEHelper:    &PKCEHelper{},
		Region:        region,
		BaseURL:       baseURL,
		ClientID:      minimaxClientID,
		OAuthCodeURL:  baseURL + "/oauth/code",
		OAuthTokenURL: baseURL + "/oauth/token",
	}
}

// GetProviderName 获取提供商名称
func (p *MiniMaxProvider) GetProviderName() string {
	return "MiniMax"
}

// providerConfig 从当前字段值装配泛化配置（每次现构造，避免字段覆盖后状态失步）
func (p *MiniMaxProvider) providerConfig() ProviderConfig {
	return ProviderConfig{
		Name:         "MiniMax",
		Kind:         KindUserCode,
		AuthorizeURL: p.OAuthCodeURL,
		TokenURL:     p.OAuthTokenURL,
		ClientID:     p.ClientID,
		Scopes:       []string{"group_id", "profile", "model.completion"},
		UsePKCE:      true,
		HTTPClient:   p.HTTPClient,
	}
}

// RequestOAuthCode 请求 OAuth 授权码（转调泛化实现）
func (p *MiniMaxProvider) RequestOAuthCode() (*MiniMaxOAuthAuthorization, string, string, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, "", "", err
	}
	return provider.RequestUserCodeAuthorization()
}

// PollOAuthToken 轮询 OAuth 令牌（转调泛化实现）
func (p *MiniMaxProvider) PollOAuthToken(userCode, verifier string) (status string, token *OAuthToken, errorMsg string) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return "error", nil, err.Error()
	}
	return provider.PollUserCodeToken(userCode, verifier)
}

// Authenticate 执行认证流程（转调泛化实现）
func (p *MiniMaxProvider) Authenticate() (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.Authenticate()
}

// RefreshToken 刷新令牌（转调泛化实现）
func (p *MiniMaxProvider) RefreshToken(refreshToken string) (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.RefreshToken(refreshToken)
}
