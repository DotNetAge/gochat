package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
)

// GeminiProvider Gemini 提供商 (标准 OAuth2 Authorization Code Flow)
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

// Authenticate 执行标准 OAuth2 授权码流程
func (p *GeminiProvider) Authenticate() (*core.OAuthToken, error) {
	// 1. 构建 Google OAuth2 授权链接
	scope := "https://www.googleapis.com/auth/generative-language.retriever"
	authURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline&prompt=consent",
		url.QueryEscape(p.ClientID),
		url.QueryEscape(p.CallbackURL),
		url.QueryEscape(scope),
	)

	fmt.Println("=== Gemini OAuth2 Authorization ===")
	fmt.Printf("Please open this URL in your browser to authorize:\n\n%s\n\n", authURL)
	fmt.Printf("Waiting for callback on %s ... (Timeout: 5 minutes)\n", p.CallbackURL)

	// 2. 启动本地临时 HTTP 服务器接收 Callback
	codeChan := make(chan string)
	errChan := make(chan error)

	mux := http.NewServeMux()

	callbackURL, err := url.Parse(p.CallbackURL)
	if err != nil {
		return nil, fmt.Errorf("invalid callback URL: %v", err)
	}

	mux.HandleFunc(callbackURL.Path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			fmt.Fprintf(w, "Authorization failed: %s. You can close this window.", errMsg)
			errChan <- fmt.Errorf("oauth error from server: %s", errMsg)
			return
		}
		fmt.Fprint(w, "Authorization successful! You can close this window and return to the terminal.")
		codeChan <- code
	})

	srv := &http.Server{Addr: p.ListenAddr, Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("local server error: %v", err)
		}
	}()

	// 确保退出时关闭服务器
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// 3. 阻塞等待授权码返回或超时
	var code string
	select {
	case code = <-codeChan:
		fmt.Println("\n[Success] Received Authorization Code!")
	case err := <-errChan:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out waiting for callback")
	}

	// 4. 使用 Code 换取 Token
	return p.exchangeCodeForToken(code)
}

// exchangeCodeForToken 授权码换取令牌
func (p *GeminiProvider) exchangeCodeForToken(code string) (*core.OAuthToken, error) {
	data := url.Values{}
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", p.CallbackURL)

	req, err := http.NewRequest("POST", p.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return nil, fmt.Errorf("failed to exchange code for token: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &core.OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: tokenResp.RefreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn*1000),
	}, nil
}

// RefreshToken 刷新令牌
func (p *GeminiProvider) RefreshToken(refreshToken string) (*core.OAuthToken, error) {
	data := url.Values{}
	data.Set("client_id", p.ClientID)
	data.Set("client_secret", p.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequest("POST", p.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return nil, fmt.Errorf("failed to refresh token: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	return &core.OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: refreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn*1000),
	}, nil
}
