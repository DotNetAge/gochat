package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
)

// Undocumented Qwen Portal Constants discovered via testing
const (
	// QwenPortalModelCoder is the model used for coding tasks in the portal.
	QwenPortalModelCoder = "coder-model"

	// QwenPortalModelVision is the multimodal model used in the portal.
	QwenPortalModelVision = "vision-model"

	// QwenPortalDailyLimit represents the undocumented daily request limit.
	QwenPortalDailyLimit = 2000
)

// QwenProvider Qwen 提供商
type QwenProvider struct {
	HTTPClient    *http.Client
	TokenHelper   *core.TokenHelper
	PKCEHelper    *core.PKCEHelper
	DeviceCodeURL string // For testing override
	TokenURL      string // For testing override
}

// NewQwenProvider 创建 Qwen 提供商
func NewQwenProvider() *QwenProvider {
	return &QwenProvider{
		HTTPClient:    &http.Client{Timeout: 30 * time.Second},
		TokenHelper:   &core.TokenHelper{},
		PKCEHelper:    &core.PKCEHelper{},
		DeviceCodeURL: "https://chat.qwen.ai/api/v1/oauth2/device/code",
		TokenURL:      "https://chat.qwen.ai/api/v1/oauth2/token",
	}
}

// GetProviderName 获取提供商名称
func (p *QwenProvider) GetProviderName() string {
	return "Qwen"
}

// QwenDeviceAuthorization Qwen 设备授权响应
type QwenDeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

// RequestDeviceCode 请求设备码
func (p *QwenProvider) RequestDeviceCode() (*QwenDeviceAuthorization, string, error) {
	verifier, challenge, err := p.PKCEHelper.GeneratePKCE()
	if err != nil {
		return nil, "", err
	}

	requestBody := "client_id=f0304373b74a44d2b584a3fb70ca9e56&scope=openid%20profile%20email%20model.completion&code_challenge=" + challenge + "&code_challenge_method=S256"

	req, err := http.NewRequest("POST", p.DeviceCodeURL, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	requestID, err := p.PKCEHelper.GenerateUUID()
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("x-request-id", requestID)

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("Qwen device authorization failed: %s", string(body))
	}

	var result QwenDeviceAuthorization
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", err
	}

	if result.DeviceCode == "" || result.UserCode == "" || result.VerificationURI == "" {
		return nil, "", fmt.Errorf("Qwen device authorization returned an incomplete payload")
	}

	return &result, verifier, nil
}

// PollForToken 轮询获取令牌
func (p *QwenProvider) PollForToken(deviceCode, verifier string) (status string, token *core.OAuthToken, slowDown bool, errorMsg string) {
	requestBody := "grant_type=urn:ietf:params:oauth:grant-type:device_code&client_id=f0304373b74a44d2b584a3fb70ca9e56&device_code=" + deviceCode + "&code_verifier=" + verifier

	req, err := http.NewRequest("POST", p.TokenURL, bytes.NewBufferString(requestBody))
	if err != nil {
		return "error", nil, false, err.Error()
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return "error", nil, false, err.Error()
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "error", nil, false, err.Error()
	}

	var errorResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if resp.StatusCode != http.StatusOK {
		if err := json.Unmarshal(body, &errorResp); err == nil {
			switch errorResp.Error {
			case "authorization_pending":
				return "pending", nil, false, ""
			case "slow_down":
				return "pending", nil, true, ""
			default:
				msg := errorResp.ErrorDescription
				if msg == "" {
					msg = errorResp.Error
				}
				return "error", nil, false, msg
			}
		}
		return "error", nil, false, string(body)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		ResourceUrl  string `json:"resource_url,omitempty"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "error", nil, false, "Failed to parse token response"
	}

	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" || tokenResp.ExpiresIn <= 0 {
		return "error", nil, false, "Qwen OAuth returned incomplete token payload"
	}

	token = &core.OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: tokenResp.RefreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn*1000),
	}

	if tokenResp.ResourceUrl != "" {
		token.ResourceUrl = &tokenResp.ResourceUrl
	}

	return "success", token, false, ""
}

// Authenticate 执行认证流程
func (p *QwenProvider) Authenticate() (*core.OAuthToken, error) {
	deviceCode, verifier, err := p.RequestDeviceCode()
	if err != nil {
		return nil, err
	}

	verificationUrl := deviceCode.VerificationURI
	if deviceCode.VerificationURIComplete != "" {
		verificationUrl = deviceCode.VerificationURIComplete
	}

	fmt.Printf("[Qwen] Open %s to approve access.\n", verificationUrl)
	fmt.Printf("[Qwen] If prompted, enter the code %s.\n", deviceCode.UserCode)
	fmt.Printf("[Qwen] Waiting for OAuth approval... (expires in %d seconds)\n", deviceCode.ExpiresIn)

	pollIntervalMs := deviceCode.Interval
	if pollIntervalMs == 0 {
		pollIntervalMs = 2
	} else {
		pollIntervalMs = pollIntervalMs * 1000
	}
	timeoutMs := int64(deviceCode.ExpiresIn * 1000)
	start := time.Now().UnixMilli()

	for time.Now().UnixMilli()-start < timeoutMs {
		status, token, slowDown, errorMsg := p.PollForToken(deviceCode.DeviceCode, verifier)

		switch status {
		case "success":
			fmt.Println("[Qwen] OAuth complete")
			return token, nil
		case "error":
			return nil, fmt.Errorf("Qwen OAuth failed: %s", errorMsg)
		case "pending":
			if slowDown {
				pollIntervalMs = int(float64(pollIntervalMs) * 1.5)
				if pollIntervalMs > 10000 {
					pollIntervalMs = 10000
				}
			}
			time.Sleep(time.Duration(pollIntervalMs) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("Qwen OAuth timed out waiting for authorization")
}

// RefreshToken 刷新令牌
func (p *QwenProvider) RefreshToken(refreshToken string) (*core.OAuthToken, error) {
	requestBody := "grant_type=refresh_token&client_id=f0304373b74a44d2b584a3fb70ca9e56&refresh_token=" + refreshToken

	req, err := http.NewRequest("POST", p.TokenURL, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Qwen OAuth refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	if tokenResp.AccessToken == "" || tokenResp.ExpiresIn <= 0 {
		return nil, fmt.Errorf("Qwen OAuth refresh response missing access token or expires_in")
	}

	token := &core.OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: tokenResp.RefreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn*1000),
	}

	if tokenResp.RefreshToken == "" {
		token.Refresh = refreshToken
	}

	return token, nil
}
