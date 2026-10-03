package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHook 捕获 InteractionHook 呈现内容的测试实现
type testHook struct {
	authURL    string
	userCode   string
	verifyURL  string
	expiresIn  int
	notifyChan chan struct{}
}

func newTestHook() *testHook {
	return &testHook{notifyChan: make(chan struct{}, 4)}
}

func (h *testHook) PresentAuthURL(authorizeURL string) {
	h.authURL = authorizeURL
	h.notifyChan <- struct{}{}
}

func (h *testHook) PresentDeviceCode(userCode, verifyURL string, expiresIn int) {
	h.userCode = userCode
	h.verifyURL = verifyURL
	h.expiresIn = expiresIn
	h.notifyChan <- struct{}{}
}

func (h *testHook) PresentUserCode(userCode, verifyURL string) {
	h.userCode = userCode
	h.verifyURL = verifyURL
	h.notifyChan <- struct{}{}
}

// ── 授权码流 ─────────────────────────────────────────────────────────────

// TestConfigDrivenProvider_AuthorizationCode_FullFlow 授权码流完整流程：
// 校验授权 URL 参数（state/PKCE/redirect_uri）→ 模拟回调 → 换取令牌
func TestConfigDrivenProvider_AuthorizationCode_FullFlow(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))
		assert.Equal(t, "cfg-client", r.Form.Get("client_id"))
		assert.Equal(t, "test-code", r.Form.Get("code"))
		assert.NotEmpty(t, r.Form.Get("code_verifier"), "PKCE 开启时必须携带 code_verifier")
		assert.NotEmpty(t, r.Form.Get("redirect_uri"))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access",
			"refresh_token": "test-refresh",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:         "test-authcode",
		Kind:         KindAuthorizationCode,
		AuthorizeURL: "https://accounts.example.com/auth",
		TokenURL:     tokenServer.URL,
		ClientID:     "cfg-client",
		Scopes:       []string{"scope.a", "scope.b"},
		UsePKCE:      true,
		UseState:     true,
		ListenAddr:   "127.0.0.1:0",
		HTTPClient:   tokenServer.Client(),
	}, hook)
	require.NoError(t, err)

	// 模拟用户浏览器：从 hook 收到授权链接后解析参数并访问回调地址
	go func() {
		<-hook.notifyChan
		u, err := url.Parse(hook.authURL)
		require.NoError(t, err)
		q := u.Query()
		assert.Equal(t, "cfg-client", q.Get("client_id"))
		assert.Equal(t, "code", q.Get("response_type"))
		assert.Equal(t, "scope.a scope.b", q.Get("scope"))
		assert.NotEmpty(t, q.Get("state"))
		assert.NotEmpty(t, q.Get("code_challenge"))
		assert.Equal(t, "S256", q.Get("code_challenge_method"))

		redirect, err := url.Parse(q.Get("redirect_uri"))
		require.NoError(t, err)
		// 监听服务已就绪（PresentAuthURL 在 Serve 之后调用），直接回调
		callbackURL := "http://" + redirect.Host + redirect.Path + "?code=test-code&state=" + q.Get("state")
		resp, err := http.Get(callbackURL)
		require.NoError(t, err)
		resp.Body.Close()
	}()

	token, err := provider.Authenticate()
	require.NoError(t, err)
	assert.Equal(t, "test-access", token.Access)
	assert.Equal(t, "test-refresh", token.Refresh)
}

// TestConfigDrivenProvider_AuthorizationCode_StateMismatch 回调 state 不匹配必须拒绝（防 CSRF）
func TestConfigDrivenProvider_AuthorizationCode_StateMismatch(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("state mismatch 时不应调用令牌端点")
	}))
	defer tokenServer.Close()

	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:            "test-authcode",
		Kind:            KindAuthorizationCode,
		AuthorizeURL:    "https://accounts.example.com/auth",
		TokenURL:        tokenServer.URL,
		ClientID:        "cfg-client",
		UseState:        true,
		ListenAddr:      "127.0.0.1:0",
		CallbackTimeout: 3 * time.Second,
		HTTPClient:      tokenServer.Client(),
	}, hook)
	require.NoError(t, err)

	go func() {
		<-hook.notifyChan
		u, err := url.Parse(hook.authURL)
		require.NoError(t, err)
		q := u.Query()
		redirect, err := url.Parse(q.Get("redirect_uri"))
		require.NoError(t, err)
		// 携带错误 state 的回调
		callbackURL := "http://" + redirect.Host + redirect.Path + "?code=test-code&state=wrong-state"
		resp, err := http.Get(callbackURL)
		require.NoError(t, err)
		resp.Body.Close()
	}()

	_, err = provider.Authenticate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state mismatch")
}

// TestConfigDrivenProvider_AuthorizationCode_ErrorCallback 回调携带 error 参数
func TestConfigDrivenProvider_AuthorizationCode_ErrorCallback(t *testing.T) {
	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:            "test-authcode",
		Kind:            KindAuthorizationCode,
		AuthorizeURL:    "https://accounts.example.com/auth",
		TokenURL:        "https://token.example.com/token",
		ClientID:        "cfg-client",
		UseState:        true,
		ListenAddr:      "127.0.0.1:0",
		CallbackTimeout: 3 * time.Second,
	}, hook)
	require.NoError(t, err)

	go func() {
		<-hook.notifyChan
		u, err := url.Parse(hook.authURL)
		require.NoError(t, err)
		q := u.Query()
		redirect, err := url.Parse(q.Get("redirect_uri"))
		require.NoError(t, err)
		resp, err := http.Get("http://" + redirect.Host + redirect.Path + "?error=access_denied")
		require.NoError(t, err)
		resp.Body.Close()
	}()

	_, err = provider.Authenticate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_denied")
}

// ── 设备码流 ─────────────────────────────────────────────────────────────

// TestConfigDrivenProvider_DeviceCode_FullFlow 设备码流完整流程：
// 请求设备码 → 呈现 → 首次轮询成功
func TestConfigDrivenProvider_DeviceCode_FullFlow(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "cfg-client", r.Form.Get("client_id"))
		assert.Equal(t, "openid", r.Form.Get("scope"))
		assert.NotEmpty(t, r.Form.Get("code_challenge"), "PKCE 开启时必须携带 challenge")
		assert.Equal(t, "S256", r.Form.Get("code_challenge_method"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code":      "test-device-code",
			"user_code":        "TEST-CODE",
			"verification_uri": "http://example.com/verify",
			"expires_in":       3600,
		})
	}))
	defer deviceServer.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:device_code", r.Form.Get("grant_type"))
		assert.Equal(t, "test-device-code", r.Form.Get("device_code"))
		assert.NotEmpty(t, r.Form.Get("code_verifier"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access",
			"refresh_token": "test-refresh",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:       "test-device",
		Kind:       KindDeviceCode,
		TokenURL:   tokenServer.URL,
		DeviceURL:  deviceServer.URL,
		ClientID:   "cfg-client",
		Scopes:     []string{"openid"},
		UsePKCE:    true,
		HTTPClient: deviceServer.Client(),
	}, hook)
	require.NoError(t, err)

	token, err := provider.Authenticate()
	require.NoError(t, err)
	assert.Equal(t, "test-access", token.Access)

	<-hook.notifyChan
	assert.Equal(t, "TEST-CODE", hook.userCode)
	assert.Equal(t, "http://example.com/verify", hook.verifyURL)
}

// TestConfigDrivenProvider_DeviceCode_PendingThenSuccess 设备码轮询处理 authorization_pending
func TestConfigDrivenProvider_DeviceCode_PendingThenSuccess(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code":      "test-device-code",
			"user_code":        "TEST-CODE",
			"verification_uri": "http://example.com/verify",
			"expires_in":       3600,
			"interval":         1,
		})
	}))
	defer deviceServer.Close()

	var pollCount int32
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&pollCount, 1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access",
			"refresh_token": "test-refresh",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:       "test-device",
		Kind:       KindDeviceCode,
		TokenURL:   tokenServer.URL,
		DeviceURL:  deviceServer.URL,
		ClientID:   "cfg-client",
		UsePKCE:    true,
		HTTPClient: deviceServer.Client(),
	}, hook)
	require.NoError(t, err)

	token, err := provider.Authenticate()
	require.NoError(t, err)
	assert.Equal(t, "test-access", token.Access)
	assert.EqualValues(t, 2, atomic.LoadInt32(&pollCount))
}

// TestConfigDrivenProvider_DeviceCode_Timeout 设备码过期后轮询超时
func TestConfigDrivenProvider_DeviceCode_Timeout(t *testing.T) {
	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"device_code":      "test-device-code",
			"user_code":        "TEST-CODE",
			"verification_uri": "http://example.com/verify",
			"expires_in":       0, // 已过期 → 立即超时
		})
	}))
	defer deviceServer.Close()

	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:       "test-device",
		Kind:       KindDeviceCode,
		TokenURL:   "https://token.example.com/token",
		DeviceURL:  deviceServer.URL,
		ClientID:   "cfg-client",
		HTTPClient: deviceServer.Client(),
	}, newTestHook())
	require.NoError(t, err)

	_, err = provider.Authenticate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// ── 用户码两步协议 ───────────────────────────────────────────────────────

// TestConfigDrivenProvider_UserCode_FullFlow 用户码两步协议完整流程：
// 请求用户码（state 回显校验）→ 呈现 → 轮询成功（expired_in 绝对时间戳）
func TestConfigDrivenProvider_UserCode_FullFlow(t *testing.T) {
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "cfg-client", r.Form.Get("client_id"))
		assert.Equal(t, "code", r.Form.Get("response_type"))
		requestState := r.Form.Get("state")
		assert.NotEmpty(t, requestState)
		assert.NotEmpty(t, r.Form.Get("code_challenge"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_code":        "TEST-USER-CODE",
			"verification_uri": "http://example.com/verify",
			"expired_in":       time.Now().Add(time.Hour).UnixMilli(),
			"state":            requestState,
		})
	}))
	defer codeServer.Close()

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "urn:ietf:params:oauth:grant-type:user_code", r.Form.Get("grant_type"))
		assert.Equal(t, "TEST-USER-CODE", r.Form.Get("user_code"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "success",
			"access_token":  "test-access",
			"refresh_token": "test-refresh",
			"expired_in":    time.Now().Add(time.Hour).UnixMilli(),
			"resource_url":  "https://api.example.com",
		})
	}))
	defer tokenServer.Close()

	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:         "test-usercode",
		Kind:         KindUserCode,
		AuthorizeURL: codeServer.URL,
		TokenURL:     tokenServer.URL,
		ClientID:     "cfg-client",
		UsePKCE:      true,
		HTTPClient:   codeServer.Client(),
	}, hook)
	require.NoError(t, err)

	token, err := provider.Authenticate()
	require.NoError(t, err)
	assert.Equal(t, "test-access", token.Access)
	require.NotNil(t, token.ResourceUrl)
	assert.Equal(t, "https://api.example.com", *token.ResourceUrl)

	<-hook.notifyChan
	assert.Equal(t, "TEST-USER-CODE", hook.userCode)
}

// TestConfigDrivenProvider_UserCode_StateMismatch 响应 state 与请求 state 不一致必须拒绝
func TestConfigDrivenProvider_UserCode_StateMismatch(t *testing.T) {
	codeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_code":        "TEST-USER-CODE",
			"verification_uri": "http://example.com/verify",
			"expired_in":       time.Now().Add(time.Hour).UnixMilli(),
			"state":            "tampered-state",
		})
	}))
	defer codeServer.Close()

	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:         "test-usercode",
		Kind:         KindUserCode,
		AuthorizeURL: codeServer.URL,
		TokenURL:     "https://token.example.com/token",
		ClientID:     "cfg-client",
		HTTPClient:   codeServer.Client(),
	}, newTestHook())
	require.NoError(t, err)

	_, err = provider.Authenticate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "state mismatch")
}

// ── 刷新与校验 ───────────────────────────────────────────────────────────

// TestConfigDrivenProvider_RefreshToken_Standard 标准协议族刷新（expires_in 秒）
func TestConfigDrivenProvider_RefreshToken_Standard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "new-access",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:       "test-refresh",
		Kind:       KindDeviceCode,
		TokenURL:   server.URL,
		DeviceURL:  "https://device.example.com/code",
		ClientID:   "cfg-client",
		HTTPClient: server.Client(),
	}, nil)
	require.NoError(t, err)

	token, err := provider.RefreshToken("old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "new-access", token.Access)
	assert.Equal(t, "old-refresh", token.Refresh, "响应未含 refresh_token 时回退旧值")
}

// TestConfigDrivenProvider_RefreshToken_UserCode 用户码协议族刷新（expired_in 绝对毫秒）
func TestConfigDrivenProvider_RefreshToken_UserCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "success",
			"access_token": "new-access",
			"expired_in":   time.Now().Add(time.Hour).UnixMilli(),
		})
	}))
	defer server.Close()

	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:         "test-refresh",
		Kind:         KindUserCode,
		AuthorizeURL: "https://code.example.com/code",
		TokenURL:     server.URL,
		ClientID:     "cfg-client",
		HTTPClient:   server.Client(),
	}, nil)
	require.NoError(t, err)

	token, err := provider.RefreshToken("old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "new-access", token.Access)
	assert.Equal(t, "old-refresh", token.Refresh)
}

// TestProviderConfig_Validate 配置校验：kind 枚举、必填端点、interaction 一致性
func TestProviderConfig_Validate(t *testing.T) {
	base := ProviderConfig{
		Name:         "test",
		Kind:         KindAuthorizationCode,
		AuthorizeURL: "https://a.example.com",
		TokenURL:     "https://t.example.com",
		ClientID:     "cid",
	}

	assert.NoError(t, base.Validate())

	// name 必填
	missingName := base
	missingName.Name = ""
	assert.Error(t, missingName.Validate())

	// client_id 必填（空 = 未配置态）
	missingClient := base
	missingClient.ClientID = ""
	assert.Error(t, missingClient.Validate())

	// authorization_code 缺 authorize_url
	missingAuthorize := base
	missingAuthorize.AuthorizeURL = ""
	assert.Error(t, missingAuthorize.Validate())

	// device_code 缺 device_url
	device := base
	device.Kind = KindDeviceCode
	device.AuthorizeURL = ""
	device.DeviceURL = "https://d.example.com"
	assert.NoError(t, device.Validate())
	device.DeviceURL = ""
	assert.Error(t, device.Validate())

	// 未知 kind
	badKind := base
	badKind.Kind = ProviderKind("magic")
	assert.Error(t, badKind.Validate())

	// interaction 与 kind 不一致
	badInteraction := base
	badInteraction.Interaction = InteractionDeviceCode
	assert.Error(t, badInteraction.Validate())

	// interaction 与 kind 一致
	goodInteraction := base
	goodInteraction.Interaction = InteractionCallback
	assert.NoError(t, goodInteraction.Validate())

	// token_url 三类通用必填
	missingToken := base
	missingToken.TokenURL = ""
	assert.Error(t, missingToken.Validate())
}

// TestProviderKind_InteractionDerivation 交互方式派生
func TestProviderKind_InteractionDerivation(t *testing.T) {
	authCode := ProviderConfig{Kind: KindAuthorizationCode}
	deviceCode := ProviderConfig{Kind: KindDeviceCode}
	userCode := ProviderConfig{Kind: KindUserCode}
	assert.Equal(t, InteractionCallback, authCode.DeriveInteraction())
	assert.Equal(t, InteractionDeviceCode, deviceCode.DeriveInteraction())
	assert.Equal(t, InteractionUserCode, userCode.DeriveInteraction())
}

// TestConfigDrivenProvider_ExtraParams 附加授权参数透传（Google access_type=offline 场景）
func TestConfigDrivenProvider_ExtraParams(t *testing.T) {
	hook := newTestHook()
	provider, err := NewConfigDrivenProvider(ProviderConfig{
		Name:            "test-extra",
		Kind:            KindAuthorizationCode,
		AuthorizeURL:    "https://accounts.example.com/auth",
		TokenURL:        "https://token.example.com/token",
		ClientID:        "cfg-client",
		ExtraParams:     map[string]string{"access_type": "offline", "prompt": "consent"},
		CallbackTimeout: 3 * time.Second,
	}, hook)
	require.NoError(t, err)

	go func() {
		<-hook.notifyChan
		// 超时路径：仅验证 URL 构造，不完成授权
	}()
	_, _ = provider.Authenticate() // 等待超时

	assert.True(t, strings.Contains(hook.authURL, "access_type=offline"))
	assert.True(t, strings.Contains(hook.authURL, "prompt=consent"))
}
