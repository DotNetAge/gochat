package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
)

// MiniMaxProvider MiniMax 提供商
type MiniMaxProvider struct {
	HTTPClient  *http.Client
	TokenHelper *core.TokenHelper
	PKCEHelper  *core.PKCEHelper
	Region      string
	BaseURL     string
	ClientID    string
}

// NewMiniMaxProvider 创建 MiniMax 提供商
func NewMiniMaxProvider(region string) *MiniMaxProvider {
	var baseURL, clientID string
	if region == "cn" {
		baseURL = "https://api.minimaxi.com"
	} else {
		baseURL = "https://api.minimax.io"
	}
	clientID = "78257093-7e40-4613-99e0-527b14b39113"

	return &MiniMaxProvider{
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
		TokenHelper: &core.TokenHelper{},
		PKCEHelper:  &core.PKCEHelper{},
		Region:      region,
		BaseURL:     baseURL,
		ClientID:    clientID,
	}
}

// GetProviderName 获取提供商名称
func (p *MiniMaxProvider) GetProviderName() string {
	return "MiniMax"
}

// MiniMaxOAuthAuthorization MiniMax OAuth 授权响应
type MiniMaxOAuthAuthorization struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiredIn       int    `json:"expired_in"`
	Interval        int    `json:"interval,omitempty"`
	State           string `json:"state"`
}

// RequestOAuthCode 请求 OAuth 授权码
func (p *MiniMaxProvider) RequestOAuthCode() (*MiniMaxOAuthAuthorization, string, string, error) {
	verifier, challenge, err := p.PKCEHelper.GeneratePKCE()
	if err != nil {
		return nil, "", "", err
	}

	state, err := p.PKCEHelper.GenerateState()
	if err != nil {
		return nil, "", "", err
	}

	requestBody := "response_type=code&client_id=" + p.ClientID + "&scope=group_id%20profile%20model.completion&code_challenge=" + challenge + "&code_challenge_method=S256&state=" + state

	req, err := http.NewRequest("POST", p.BaseURL+"/oauth/code", bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-request-id", p.PKCEHelper.GenerateUUID())

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("MiniMax OAuth authorization failed: %s", string(body))
	}

	var result MiniMaxOAuthAuthorization
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", "", err
	}

	if result.UserCode == "" || result.VerificationURI == "" {
		return nil, "", "", fmt.Errorf("MiniMax OAuth authorization returned an incomplete payload")
	}

	if result.State != state {
		return nil, "", "", fmt.Errorf("MiniMax OAuth state mismatch: possible CSRF attack or session corruption")
	}

	return &result, verifier, state, nil
}

// PollOAuthToken 轮询 OAuth 令牌
func (p *MiniMaxProvider) PollOAuthToken(userCode, verifier string) (status string, token *core.OAuthToken, errorMsg string) {
	requestBody := "grant_type=urn:ietf:params:oauth:grant-type:user_code&client_id=" + p.ClientID + "&user_code=" + userCode + "&code_verifier=" + verifier

	req, err := http.NewRequest("POST", p.BaseURL+"/oauth/token", bytes.NewBufferString(requestBody))
	if err != nil {
		return "error", nil, err.Error()
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return "error", nil, err.Error()
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "error", nil, err.Error()
	}

	var payload struct {
		Status              string `json:"status"`
		AccessToken         string `json:"access_token"`
		RefreshToken        string `json:"refresh_token"`
		ExpiredIn           int64  `json:"expired_in"`
		ResourceUrl         string `json:"resource_url,omitempty"`
		NotificationMessage string `json:"notification_message,omitempty"`
		BaseResp            *struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return "error", nil, "MiniMax OAuth failed to parse response"
	}

	if resp.StatusCode != http.StatusOK {
		msg := payload.BaseResp.StatusMsg
		if msg == "" {
			msg = string(body)
		}
		return "error", nil, msg
	}

	if payload.Status == "error" {
		return "error", nil, "An error occurred. Please try again later"
	}

	if payload.Status != "success" {
		return "pending", nil, "current user code is not authorized"
	}

	if payload.AccessToken == "" || payload.RefreshToken == "" || payload.ExpiredIn <= 0 {
		return "error", nil, "MiniMax OAuth returned incomplete token payload"
	}

	token = &core.OAuthToken{
		Access:  payload.AccessToken,
		Refresh: payload.RefreshToken,
		Expires: payload.ExpiredIn, // MiniMax 直接返回 Unix 时间戳
	}

	if payload.ResourceUrl != "" {
		token.ResourceUrl = &payload.ResourceUrl
	}

	if payload.NotificationMessage != "" {
		token.NotificationMessage = &payload.NotificationMessage
	}

	return "success", token, ""
}

// Authenticate 执行认证流程
func (p *MiniMaxProvider) Authenticate() (*core.OAuthToken, error) {
	auth, verifier, _, err := p.RequestOAuthCode()
	if err != nil {
		return nil, err
	}

	fmt.Printf("[MiniMax] Open %s to approve access.\n", auth.VerificationURI)
	fmt.Printf("[MiniMax] If prompted, enter the code %s.\n", auth.UserCode)
	fmt.Printf("[MiniMax] Interval: %dms, Expires at: %d unix timestamp\n", auth.Interval, auth.ExpiredIn)
	fmt.Printf("[MiniMax] Waiting for OAuth approval...\n")

	pollIntervalMs := auth.Interval
	if pollIntervalMs == 0 {
		pollIntervalMs = 2000
	}
	expireTimeMs := int64(auth.ExpiredIn)

	for time.Now().UnixMilli() < expireTimeMs {
		status, token, errorMsg := p.PollOAuthToken(auth.UserCode, verifier)

		switch status {
		case "success":
			fmt.Println("[MiniMax] OAuth complete")
			return token, nil
		case "error":
			return nil, fmt.Errorf("MiniMax OAuth failed: %s", errorMsg)
		case "pending":
			pollIntervalMs = int(float64(pollIntervalMs) * 1.5)
			if pollIntervalMs > 10000 {
				pollIntervalMs = 10000
			}
			time.Sleep(time.Duration(pollIntervalMs) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("MiniMax OAuth timed out waiting for authorization")
}

// RefreshToken 刷新令牌
func (p *MiniMaxProvider) RefreshToken(refreshToken string) (*core.OAuthToken, error) {
	requestBody := "grant_type=refresh_token&client_id=" + p.ClientID + "&refresh_token=" + refreshToken

	req, err := http.NewRequest("POST", p.BaseURL+"/oauth/token", bytes.NewBufferString(requestBody))
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MiniMax refresh token failed: %s", string(body))
	}

	var payload struct {
		Status              string `json:"status"`
		AccessToken         string `json:"access_token"`
		RefreshToken        string `json:"refresh_token"`
		ExpiredIn           int64  `json:"expired_in"`
		ResourceUrl         string `json:"resource_url,omitempty"`
		NotificationMessage string `json:"notification_message,omitempty"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if payload.Status != "success" || payload.AccessToken == "" || payload.ExpiredIn <= 0 {
		return nil, fmt.Errorf("MiniMax OAuth returned incomplete refresh token payload")
	}

	token := &core.OAuthToken{
		Access:  payload.AccessToken,
		Refresh: payload.RefreshToken,
		Expires: payload.ExpiredIn,
	}

	if payload.ResourceUrl != "" {
		token.ResourceUrl = &payload.ResourceUrl
	}

	if payload.NotificationMessage != "" {
		token.NotificationMessage = &payload.NotificationMessage
	}

	return token, nil
}
