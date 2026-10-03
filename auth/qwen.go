package auth

import (
	"net/http"
	"time"
)

// Qwen Portal 端点与公开 client_id（照抄原实现，Undocumented Qwen Portal Constants）
const (
	// qwenClientID Qwen Portal 公开 client_id（PKCE 公共客户端）
	qwenClientID = "f0304373b74a44d2b584a3fb70ca9e56"

	// qwenDeviceCodeURL 设备码端点
	qwenDeviceCodeURL = "https://chat.qwen.ai/api/v1/oauth2/device/code"

	// qwenTokenURL 令牌端点
	qwenTokenURL = "https://chat.qwen.ai/api/v1/oauth2/token"

	// QwenPortalModelCoder is the model used for coding tasks in the portal.
	QwenPortalModelCoder = "coder-model"

	// QwenPortalModelVision is the multimodal model used in the portal.
	QwenPortalModelVision = "vision-model"

	// QwenPortalDailyLimit represents the undocumented daily request limit.
	QwenPortalDailyLimit = 2000
)

// QwenProvider Qwen 提供商——设备码流的薄包装，
// 流程逻辑已迁移至 ConfigDrivenProvider，本结构体仅装配参数转调。
type QwenProvider struct {
	HTTPClient    *http.Client
	TokenHelper   *TokenHelper
	PKCEHelper    *PKCEHelper
	DeviceCodeURL string // For testing override
	TokenURL      string // For testing override
}

// NewQwenProvider 创建 Qwen 提供商
func NewQwenProvider() *QwenProvider {
	return &QwenProvider{
		HTTPClient:    &http.Client{Timeout: 30 * time.Second},
		TokenHelper:   &TokenHelper{},
		PKCEHelper:    &PKCEHelper{},
		DeviceCodeURL: qwenDeviceCodeURL,
		TokenURL:      qwenTokenURL,
	}
}

// GetProviderName 获取提供商名称
func (p *QwenProvider) GetProviderName() string {
	return "Qwen"
}

// providerConfig 从当前字段值装配泛化配置（每次现构造，避免字段覆盖后状态失步）
func (p *QwenProvider) providerConfig() ProviderConfig {
	return ProviderConfig{
		Name:       "Qwen",
		Kind:       KindDeviceCode,
		TokenURL:   p.TokenURL,
		DeviceURL:  p.DeviceCodeURL,
		ClientID:   qwenClientID,
		Scopes:     []string{"openid", "profile", "email", "model.completion"},
		UsePKCE:    true,
		HTTPClient: p.HTTPClient,
	}
}

// RequestDeviceCode 请求设备码（转调泛化实现）
func (p *QwenProvider) RequestDeviceCode() (*QwenDeviceAuthorization, string, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, "", err
	}
	return provider.RequestDeviceAuthorization()
}

// PollForToken 轮询获取令牌（转调泛化实现）
func (p *QwenProvider) PollForToken(deviceCode, verifier string) (status string, token *OAuthToken, slowDown bool, errorMsg string) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return "error", nil, false, err.Error()
	}
	return provider.PollDeviceToken(deviceCode, verifier)
}

// Authenticate 执行认证流程（转调泛化实现）
func (p *QwenProvider) Authenticate() (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.Authenticate()
}

// RefreshToken 刷新令牌（转调泛化实现）
func (p *QwenProvider) RefreshToken(refreshToken string) (*OAuthToken, error) {
	provider, err := NewConfigDrivenProvider(p.providerConfig(), PrintInteractionHook{})
	if err != nil {
		return nil, err
	}
	return provider.RefreshToken(refreshToken)
}
