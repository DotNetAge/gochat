package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProviderKind OAuth 授权方式（对应 client-provider.yml 的 kind 字段）
type ProviderKind string

const (
	// KindAuthorizationCode 授权码流：浏览器跳转授权 → 本地回调收码 → 换令牌
	KindAuthorizationCode ProviderKind = "authorization_code"
	// KindDeviceCode 设备码流（RFC 8628）：请求设备码 → 用户打开网址输码 → 轮询令牌
	KindDeviceCode ProviderKind = "device_code"
	// KindUserCode 用户码两步协议（MiniMax 风格）：请求用户码 → 用户授权 → 轮询令牌；
	// 与标准设备码流的差异：expired_in 为绝对 Unix 毫秒时间戳，响应带 status 包装
	KindUserCode ProviderKind = "user_code"
)

// 交互方式常量（由 ProviderKind 派生，冗余写出供壳侧零推导）
const (
	InteractionCallback   = "callback"    // 壳拉起浏览器 + 本地监听回调
	InteractionDeviceCode = "device_code" // 壳展示设备码 + 打开网址
	InteractionUserCode   = "user_code"   // 壳展示用户码 + 轮询等待
)

// InteractionHook 授权交互通道抽象——泛化实现与用户之间的呈现接缝。
// CLI 场景注入打印实现（PrintInteractionHook），壳（GUI）场景注入 UI 实现。
type InteractionHook interface {
	// PresentAuthURL 授权码流：呈现授权链接并等待用户在浏览器完成授权
	PresentAuthURL(authorizeURL string)
	// PresentDeviceCode 设备码流：呈现用户码与验证网址（expiresIn 秒后过期）
	PresentDeviceCode(userCode, verifyURL string, expiresIn int)
	// PresentUserCode 用户码两步协议：呈现用户码与验证网址
	PresentUserCode(userCode, verifyURL string)
}

// PrintInteractionHook InteractionHook 的 CLI 打印实现
type PrintInteractionHook struct{}

// PresentAuthURL 打印授权链接
func (PrintInteractionHook) PresentAuthURL(authorizeURL string) {
	fmt.Println("=== OAuth2 Authorization ===")
	fmt.Printf("Please open this URL in your browser to authorize:\n\n%s\n\n", authorizeURL)
}

// PresentDeviceCode 打印设备码与验证网址
func (PrintInteractionHook) PresentDeviceCode(userCode, verifyURL string, expiresIn int) {
	fmt.Printf("[OAuth] Open %s to approve access.\n", verifyURL)
	fmt.Printf("[OAuth] If prompted, enter the code %s.\n", userCode)
	fmt.Printf("[OAuth] Waiting for approval... (expires in %d seconds)\n", expiresIn)
}

// PresentUserCode 打印用户码与验证网址
func (PrintInteractionHook) PresentUserCode(userCode, verifyURL string) {
	fmt.Printf("[OAuth] Open %s to approve access.\n", verifyURL)
	fmt.Printf("[OAuth] If prompted, enter the code %s.\n", userCode)
	fmt.Printf("[OAuth] Waiting for approval...\n")
}

// ProviderConfig OAuth 客户端提供商配置——与 client-provider.yml 的 oauth_providers
// 条目字段一比一对应；yml 加载器构造本结构体后交 ConfigDrivenProvider 执行授权。
type ProviderConfig struct {
	// Name 提供商唯一键，技能 frontmatter 的 requires.oauth.provider 引用此名
	Name string
	// Title 展示名（授权预告 UI 用）
	Title string
	// Kind 授权方式：authorization_code | device_code | user_code
	Kind ProviderKind
	// AuthorizeURL 授权端点（authorization_code / user_code 必填）
	AuthorizeURL string
	// TokenURL 令牌端点（三类通用必填）
	TokenURL string
	// DeviceURL 设备码端点（device_code 必填）
	DeviceURL string
	// ClientID 客户端 ID；空 = 未配置态
	ClientID string
	// ClientSecret 机密客户端凭据——绝不从 yml 预置，仅运行时内存构造
	ClientSecret string
	// Scopes 授权范围
	Scopes []string
	// ExtraParams 附加授权参数（如 Google 的 access_type=offline / prompt=consent）
	ExtraParams map[string]string
	// UsePKCE 公共客户端恒 true（S256）；机密客户端可选
	UsePKCE bool
	// UseState 授权码流是否强制校验 state（防 CSRF）。yml 路径恒为 true 不可关；
	// 仅 legacy 直连包装为兼容既有行为显式关闭
	UseState bool
	// ListenAddr 授权码流本地回调监听地址（如 127.0.0.1:0，端口 0 = 随机）
	ListenAddr string
	// RedirectURL 完整回调地址；非空时优先于 ListenAddr 派生（legacy 兼容），
	// 为空时按 http://<实际监听地址><CallbackPath> 派生（支持随机端口 RFC 8252）
	RedirectURL string
	// CallbackPath 回调路径，RedirectURL 为空时使用，默认 /callback
	CallbackPath string
	// CallbackTimeout 授权码流等待回调超时，默认 5 分钟
	CallbackTimeout time.Duration
	// Interaction 交互方式（callback | device_code | user_code）；由 Kind 派生，
	// 非空时 Validate 校验一致性
	Interaction string
	// HTTPClient 自定义 HTTP 客户端（测试覆盖用）；nil 时默认 30 秒超时
	HTTPClient *http.Client
}

// 默认值常量
const (
	defaultCallbackPath    = "/callback"
	defaultCallbackTimeout = 5 * time.Minute
	defaultPollIntervalMs  = 2000  // 轮询间隔缺省 2 秒
	maxPollIntervalMs      = 10000 // slow_down 退避封顶 10 秒
)

// DeriveInteraction 由 Kind 推导交互方式
func (c *ProviderConfig) DeriveInteraction() string {
	switch c.Kind {
	case KindAuthorizationCode:
		return InteractionCallback
	case KindDeviceCode:
		return InteractionDeviceCode
	case KindUserCode:
		return InteractionUserCode
	default:
		return ""
	}
}

// Validate 校验配置完整性与一致性：
// name 非空、kind 枚举、必填端点与 kind 匹配、interaction 派生一致。
func (c *ProviderConfig) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("oauth provider config: name is required")
	}
	if strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("oauth provider %q: client_id is empty (unconfigured)", c.Name)
	}
	switch c.Kind {
	case KindAuthorizationCode:
		if c.AuthorizeURL == "" {
			return fmt.Errorf("oauth provider %q: authorize_url is required for kind %s", c.Name, c.Kind)
		}
	case KindDeviceCode:
		if c.DeviceURL == "" {
			return fmt.Errorf("oauth provider %q: device_url is required for kind %s", c.Name, c.Kind)
		}
	case KindUserCode:
		if c.AuthorizeURL == "" {
			return fmt.Errorf("oauth provider %q: authorize_url is required for kind %s", c.Name, c.Kind)
		}
	default:
		return fmt.Errorf("oauth provider %q: unknown kind %q", c.Name, c.Kind)
	}
	if c.TokenURL == "" {
		return fmt.Errorf("oauth provider %q: token_url is required", c.Name)
	}
	if c.Interaction != "" && c.Interaction != c.DeriveInteraction() {
		return fmt.Errorf("oauth provider %q: interaction %q mismatches kind %q (expect %q)",
			c.Name, c.Interaction, c.Kind, c.DeriveInteraction())
	}
	return nil
}

// httpClient 返回可用的 HTTP 客户端
func (c *ProviderConfig) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// callbackPath 返回回调路径
func (c *ProviderConfig) callbackPath() string {
	if c.CallbackPath != "" {
		return c.CallbackPath
	}
	return defaultCallbackPath
}

// callbackTimeout 返回回调等待超时
func (c *ProviderConfig) callbackTimeout() time.Duration {
	if c.CallbackTimeout > 0 {
		return c.CallbackTimeout
	}
	return defaultCallbackTimeout
}

// DeviceAuthorization RFC 8628 设备授权端点响应
type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"`
}

// UserCodeAuthorization 用户码两步协议端点响应（expired_in 为绝对 Unix 毫秒时间戳）
type UserCodeAuthorization struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiredIn       int    `json:"expired_in"`
	Interval        int    `json:"interval,omitempty"`
	State           string `json:"state"`
}

// 历史类型别名：旧 provider 对外类型保持不变，实际结构由泛化定义承载
type (
	// QwenDeviceAuthorization 兼容别名
	QwenDeviceAuthorization = DeviceAuthorization
	// MiniMaxOAuthAuthorization 兼容别名
	MiniMaxOAuthAuthorization = UserCodeAuthorization
)

// ConfigDrivenProvider 配置驱动的通用 OAuth 提供商——yml 参数驱动实现
// AuthClientProvider 三方法，按 Kind 分支执行对应授权流程。
type ConfigDrivenProvider struct {
	cfg  ProviderConfig
	hook InteractionHook
}

// NewConfigDrivenProvider 创建配置驱动的提供商。hook 为 nil 时使用 CLI 打印实现。
func NewConfigDrivenProvider(cfg ProviderConfig, hook InteractionHook) (*ConfigDrivenProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if hook == nil {
		hook = PrintInteractionHook{}
	}
	return &ConfigDrivenProvider{cfg: cfg, hook: hook}, nil
}

// GetProviderName 获取提供商名称
func (p *ConfigDrivenProvider) GetProviderName() string {
	return p.cfg.Name
}

// Authenticate 按 Kind 分支执行授权流程
func (p *ConfigDrivenProvider) Authenticate() (*OAuthToken, error) {
	return p.AuthenticateContext(context.Background())
}

// AuthenticateContext 带取消的授权流程：ctx 取消时流程尽快返回 context.Canceled。
// 宿主（daemon 停机 / 用户取消授权）需要中断长时间阻塞的等待（回调监听、轮询），
// 无 ctx 的 Authenticate 是本方法的 Background 便捷形态。
func (p *ConfigDrivenProvider) AuthenticateContext(ctx context.Context) (*OAuthToken, error) {
	switch p.cfg.Kind {
	case KindAuthorizationCode:
		return p.authenticateAuthorizationCode(ctx)
	case KindDeviceCode:
		return p.authenticateDeviceCode(ctx)
	case KindUserCode:
		return p.authenticateUserCode(ctx)
	default:
		return nil, fmt.Errorf("oauth provider %q: unknown kind %q", p.cfg.Name, p.cfg.Kind)
	}
}

// sleepCtx 可取消休眠：ctx 取消时返回 false（调用方应立即返回 ctx.Err()）
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// RefreshToken 统一刷新令牌：grant_type=refresh_token + token_url；
// 响应解析按协议族区分（user_code 为 status/expired_in 包装，其余为标准 OAuth2 格式）
func (p *ConfigDrivenProvider) RefreshToken(refreshToken string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", p.cfg.ClientID)
	if p.cfg.ClientSecret != "" {
		data.Set("client_secret", p.cfg.ClientSecret)
	}
	data.Set("refresh_token", refreshToken)

	token, err := p.postTokenForm(p.cfg.TokenURL, data, p.cfg.Kind != KindUserCode)
	if err != nil {
		return nil, fmt.Errorf("oauth provider %q refresh failed: %w", p.cfg.Name, err)
	}
	if token.Refresh == "" {
		token.Refresh = refreshToken
	}
	return token, nil
}

// ── 授权码流（authorization_code）──────────────────────────────────────────

// authenticateAuthorizationCode 执行标准 OAuth2 授权码流程：
// PKCE/state 生成 → 本地监听 → 呈现授权链接 → 等回调校验 → 换令牌
func (p *ConfigDrivenProvider) authenticateAuthorizationCode(ctx context.Context) (*OAuthToken, error) {
	pkce := &PKCEHelper{}
	var verifier, challenge, state string
	var err error
	if p.cfg.UsePKCE {
		if verifier, challenge, err = pkce.GeneratePKCE(); err != nil {
			return nil, fmt.Errorf("generate pkce: %w", err)
		}
	}
	if p.cfg.UseState {
		if state, err = pkce.GenerateState(); err != nil {
			return nil, fmt.Errorf("generate state: %w", err)
		}
	}

	// 先行监听以拿到实际端口（支持随机端口），再构造授权链接
	listener, err := net.Listen("tcp", p.listenAddrOrDefault())
	if err != nil {
		return nil, fmt.Errorf("listen callback addr: %w", err)
	}
	redirectURI := p.cfg.RedirectURL
	if redirectURI == "" {
		redirectURI = "http://" + listener.Addr().String() + p.cfg.callbackPath()
	}

	authorizeURL := p.buildAuthorizeURL(redirectURI, state, challenge)

	// 先启动回调监听确保端口就绪，再向用户呈现授权链接，避免回调早于监听到达
	code, err := p.awaitAuthorizationCode(ctx, listener, authorizeURL, state)
	if err != nil {
		return nil, err
	}
	// RFC 6749：token 请求必须携带与授权请求一致的 redirect_uri
	return p.exchangeAuthorizationCode(code, verifier, redirectURI)
}

// listenAddrOrDefault 返回监听地址，未配置时默认回环随机端口
func (p *ConfigDrivenProvider) listenAddrOrDefault() string {
	if p.cfg.ListenAddr != "" {
		return p.cfg.ListenAddr
	}
	return "127.0.0.1:0"
}

// buildAuthorizeURL 构造授权端点 URL
func (p *ConfigDrivenProvider) buildAuthorizeURL(redirectURI, state, challenge string) string {
	params := url.Values{}
	params.Set("client_id", p.cfg.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	if len(p.cfg.Scopes) > 0 {
		params.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	if state != "" {
		params.Set("state", state)
	}
	if challenge != "" {
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
	}
	for k, v := range p.cfg.ExtraParams {
		params.Set(k, v)
	}
	separator := "?"
	if strings.Contains(p.cfg.AuthorizeURL, "?") {
		separator = "&"
	}
	return p.cfg.AuthorizeURL + separator + params.Encode()
}

// awaitAuthorizationCode 启动回调服务并阻塞等待授权码或错误，超时由
// CallbackTimeout 控制，ctx 取消即中断。UseState 开启时校验回调 state 防 CSRF。
// 监听就绪后才向用户呈现授权链接。
func (p *ConfigDrivenProvider) awaitAuthorizationCode(ctx context.Context, listener net.Listener, authorizeURL, state string) (string, error) {
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(p.cfg.callbackPath(), func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			fmt.Fprintf(w, "Authorization failed: %s. You can close this window.", errMsg)
			errChan <- fmt.Errorf("oauth error from server: %s", errMsg)
			return
		}
		if state != "" && r.URL.Query().Get("state") != state {
			fmt.Fprint(w, "Authorization failed: state mismatch. You can close this window.")
			errChan <- fmt.Errorf("oauth state mismatch: possible CSRF attack or session corruption")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			fmt.Fprint(w, "Authorization failed: missing code. You can close this window.")
			errChan <- fmt.Errorf("oauth callback missing authorization code")
			return
		}
		fmt.Fprint(w, "Authorization successful! You can close this window and return to the terminal.")
		codeChan <- code
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("local server error: %w", err)
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	// 监听已就绪，此刻呈现授权链接
	p.hook.PresentAuthURL(authorizeURL)

	select {
	case code := <-codeChan:
		return code, nil
	case err := <-errChan:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(p.cfg.callbackTimeout()):
		return "", fmt.Errorf("authentication timed out waiting for callback")
	}
}

// ExchangeAuthorizationCode 授权码换取令牌（供薄包装与流程复用）
func (p *ConfigDrivenProvider) ExchangeAuthorizationCode(code string) (*OAuthToken, error) {
	return p.exchangeAuthorizationCode(code, "", p.cfg.RedirectURL)
}

// exchangeAuthorizationCode 授权码换取令牌；PKCE 场景附带 code_verifier，
// redirectURI 非空时一并携带（RFC 6749 要求与授权请求一致）
func (p *ConfigDrivenProvider) exchangeAuthorizationCode(code, verifier, redirectURI string) (*OAuthToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", p.cfg.ClientID)
	if p.cfg.ClientSecret != "" {
		data.Set("client_secret", p.cfg.ClientSecret)
	}
	data.Set("code", code)
	if verifier != "" {
		data.Set("code_verifier", verifier)
	}
	if redirectURI != "" {
		data.Set("redirect_uri", redirectURI)
	}

	token, err := p.postTokenForm(p.cfg.TokenURL, data, true)
	if err != nil {
		return nil, fmt.Errorf("oauth provider %q exchange code failed: %w", p.cfg.Name, err)
	}
	return token, nil
}

// ── 设备码流（device_code，RFC 8628）──────────────────────────────────────

// authenticateDeviceCode 执行设备码流程：请求设备码 → 呈现 → 按 interval 轮询
func (p *ConfigDrivenProvider) authenticateDeviceCode(ctx context.Context) (*OAuthToken, error) {
	da, verifier, err := p.RequestDeviceAuthorization()
	if err != nil {
		return nil, err
	}

	verifyURL := da.VerificationURI
	if da.VerificationURIComplete != "" {
		verifyURL = da.VerificationURIComplete
	}
	p.hook.PresentDeviceCode(da.UserCode, verifyURL, da.ExpiresIn)

	intervalMs := da.Interval * 1000
	if da.Interval <= 0 {
		intervalMs = defaultPollIntervalMs
	}
	timeoutMs := int64(da.ExpiresIn) * 1000
	start := time.Now().UnixMilli()

	for time.Now().UnixMilli()-start < timeoutMs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		status, token, slowDown, errMsg := p.PollDeviceToken(da.DeviceCode, verifier)
		switch status {
		case "success":
			return token, nil
		case "error":
			return nil, fmt.Errorf("oauth device flow failed: %s", errMsg)
		case "pending":
			if slowDown {
				intervalMs = int(float64(intervalMs) * 1.5)
				if intervalMs > maxPollIntervalMs {
					intervalMs = maxPollIntervalMs
				}
			}
			if !sleepCtx(ctx, time.Duration(intervalMs)*time.Millisecond) {
				return nil, ctx.Err()
			}
		}
	}
	return nil, fmt.Errorf("oauth device flow timed out waiting for authorization")
}

// RequestDeviceAuthorization 请求设备码（供薄包装复用）；返回 PKCE verifier
func (p *ConfigDrivenProvider) RequestDeviceAuthorization() (*DeviceAuthorization, string, error) {
	pkce := &PKCEHelper{}
	var verifier, challenge string
	var err error
	if p.cfg.UsePKCE {
		if verifier, challenge, err = pkce.GeneratePKCE(); err != nil {
			return nil, "", fmt.Errorf("generate pkce: %w", err)
		}
	}

	data := url.Values{}
	data.Set("client_id", p.cfg.ClientID)
	if len(p.cfg.Scopes) > 0 {
		data.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	if challenge != "" {
		data.Set("code_challenge", challenge)
		data.Set("code_challenge_method", "S256")
	}

	da, err := postDeviceRequest[DeviceAuthorization](&p.cfg, p.cfg.DeviceURL, data)
	if err != nil {
		return nil, "", err
	}
	if da.DeviceCode == "" || da.UserCode == "" || da.VerificationURI == "" {
		return nil, "", fmt.Errorf("oauth device authorization returned an incomplete payload")
	}
	return da, verifier, nil
}

// PollDeviceToken 单次设备码轮询（供薄包装复用）：
// status = success | pending | error；pending 时 slowDown 指示是否需要退避
func (p *ConfigDrivenProvider) PollDeviceToken(deviceCode, verifier string) (status string, token *OAuthToken, slowDown bool, errorMsg string) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	data.Set("client_id", p.cfg.ClientID)
	data.Set("device_code", deviceCode)
	if verifier != "" {
		data.Set("code_verifier", verifier)
	}

	body, resp, err := postForm(&p.cfg, p.cfg.TokenURL, data)
	if err != nil {
		return "error", nil, false, err.Error()
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			switch errResp.Error {
			case "authorization_pending":
				return "pending", nil, false, ""
			case "slow_down":
				return "pending", nil, true, ""
			default:
				msg := errResp.ErrorDescription
				if msg == "" {
					msg = errResp.Error
				}
				return "error", nil, false, msg
			}
		}
		return "error", nil, false, string(body)
	}

	var tokenResp struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken string  `json:"refresh_token"`
		ExpiresIn    int     `json:"expires_in"`
		ResourceUrl  *string `json:"resource_url,omitempty"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "error", nil, false, "failed to parse token response"
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" || tokenResp.ExpiresIn <= 0 {
		return "error", nil, false, "oauth device flow returned incomplete token payload"
	}

	token = &OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: tokenResp.RefreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn)*1000,
	}
	if tokenResp.ResourceUrl != nil && *tokenResp.ResourceUrl != "" {
		token.ResourceUrl = tokenResp.ResourceUrl
	}
	return "success", token, false, ""
}

// ── 用户码两步协议（user_code）────────────────────────────────────────────

// authenticateUserCode 执行用户码两步协议：请求用户码 → 呈现 → 轮询至绝对过期
func (p *ConfigDrivenProvider) authenticateUserCode(ctx context.Context) (*OAuthToken, error) {
	auth, verifier, _, err := p.RequestUserCodeAuthorization()
	if err != nil {
		return nil, err
	}

	p.hook.PresentUserCode(auth.UserCode, auth.VerificationURI)

	intervalMs := auth.Interval
	if intervalMs <= 0 {
		intervalMs = defaultPollIntervalMs
	}
	expireTimeMs := int64(auth.ExpiredIn)

	for time.Now().UnixMilli() < expireTimeMs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		status, token, errMsg := p.PollUserCodeToken(auth.UserCode, verifier)
		switch status {
		case "success":
			return token, nil
		case "error":
			return nil, fmt.Errorf("oauth user code flow failed: %s", errMsg)
		case "pending":
			intervalMs = int(float64(intervalMs) * 1.5)
			if intervalMs > maxPollIntervalMs {
				intervalMs = maxPollIntervalMs
			}
			if !sleepCtx(ctx, time.Duration(intervalMs)*time.Millisecond) {
				return nil, ctx.Err()
			}
		}
	}
	return nil, fmt.Errorf("oauth user code flow timed out waiting for authorization")
}

// RequestUserCodeAuthorization 请求用户码（供薄包装复用）；
// 返回 verifier 与 state，响应 state 不匹配即判定 CSRF/会话损坏
func (p *ConfigDrivenProvider) RequestUserCodeAuthorization() (*UserCodeAuthorization, string, string, error) {
	pkce := &PKCEHelper{}
	verifier, challenge, err := pkce.GeneratePKCE()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate pkce: %w", err)
	}
	state, err := pkce.GenerateState()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate state: %w", err)
	}

	data := url.Values{}
	data.Set("response_type", "code")
	data.Set("client_id", p.cfg.ClientID)
	if len(p.cfg.Scopes) > 0 {
		data.Set("scope", strings.Join(p.cfg.Scopes, " "))
	}
	data.Set("code_challenge", challenge)
	data.Set("code_challenge_method", "S256")
	data.Set("state", state)

	result, err := postDeviceRequest[UserCodeAuthorization](&p.cfg, p.cfg.AuthorizeURL, data)
	if err != nil {
		return nil, "", "", err
	}
	if result.UserCode == "" || result.VerificationURI == "" {
		return nil, "", "", fmt.Errorf("oauth user code authorization returned an incomplete payload")
	}
	if result.State != state {
		return nil, "", "", fmt.Errorf("oauth state mismatch: possible CSRF attack or session corruption")
	}
	return result, verifier, state, nil
}

// PollUserCodeToken 单次用户码轮询（供薄包装复用）：
// status = success | pending | error；expired_in 为绝对 Unix 毫秒时间戳
func (p *ConfigDrivenProvider) PollUserCodeToken(userCode, verifier string) (status string, token *OAuthToken, errorMsg string) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:user_code")
	data.Set("client_id", p.cfg.ClientID)
	data.Set("user_code", userCode)
	if verifier != "" {
		data.Set("code_verifier", verifier)
	}

	body, resp, err := postForm(&p.cfg, p.cfg.TokenURL, data)
	if err != nil {
		return "error", nil, err.Error()
	}

	var payload struct {
		Status              string  `json:"status"`
		AccessToken         string  `json:"access_token"`
		RefreshToken        string  `json:"refresh_token"`
		ExpiredIn           int64   `json:"expired_in"`
		ResourceUrl         *string `json:"resource_url,omitempty"`
		NotificationMessage *string `json:"notification_message,omitempty"`
		BaseResp            *struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "error", nil, "oauth failed to parse response"
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(body)
		if payload.BaseResp != nil && payload.BaseResp.StatusMsg != "" {
			msg = payload.BaseResp.StatusMsg
		}
		return "error", nil, msg
	}

	if payload.Status == "error" {
		return "error", nil, "an error occurred. Please try again later"
	}
	if payload.Status != "success" {
		return "pending", nil, "current user code is not authorized"
	}
	if payload.AccessToken == "" || payload.RefreshToken == "" || payload.ExpiredIn <= 0 {
		return "error", nil, "oauth user code flow returned incomplete token payload"
	}

	token = &OAuthToken{
		Access:  payload.AccessToken,
		Refresh: payload.RefreshToken,
		Expires: payload.ExpiredIn, // user_code 协议直接返回 Unix 毫秒时间戳
	}
	if payload.ResourceUrl != nil && *payload.ResourceUrl != "" {
		token.ResourceUrl = payload.ResourceUrl
	}
	if payload.NotificationMessage != nil && *payload.NotificationMessage != "" {
		token.NotificationMessage = payload.NotificationMessage
	}
	return "success", token, ""
}

// ── HTTP 工具 ─────────────────────────────────────────────────────────────

// postForm 发送 form 表单 POST 并返回响应体与响应对象
func postForm(cfg *ProviderConfig, endpoint string, data url.Values) ([]byte, *http.Response, error) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := cfg.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	return body, resp, nil
}

// postDeviceRequest 发送带 x-request-id 的表单请求并解析 JSON 响应（泛型）
func postDeviceRequest[T any](cfg *ProviderConfig, endpoint string, data url.Values) (*T, error) {
	body, resp, err := postForm(cfg, endpoint, data)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth request failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// postTokenForm 发送令牌端点请求并解析。standard=true 按标准 OAuth2 响应
// （expires_in 秒），false 按 user_code 协议族（status/expired_in 绝对毫秒）。
func (p *ConfigDrivenProvider) postTokenForm(endpoint string, data url.Values, standard bool) (*OAuthToken, error) {
	body, resp, err := postForm(&p.cfg, endpoint, data)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	if !standard {
		var payload struct {
			Status              string  `json:"status"`
			AccessToken         string  `json:"access_token"`
			RefreshToken        string  `json:"refresh_token"`
			ExpiredIn           int64   `json:"expired_in"`
			ResourceUrl         *string `json:"resource_url,omitempty"`
			NotificationMessage *string `json:"notification_message,omitempty"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		if payload.Status != "success" || payload.AccessToken == "" || payload.ExpiredIn <= 0 {
			return nil, fmt.Errorf("oauth returned incomplete refresh token payload")
		}
		token := &OAuthToken{
			Access:  payload.AccessToken,
			Refresh: payload.RefreshToken,
			Expires: payload.ExpiredIn,
		}
		if payload.ResourceUrl != nil && *payload.ResourceUrl != "" {
			token.ResourceUrl = payload.ResourceUrl
		}
		if payload.NotificationMessage != nil && *payload.NotificationMessage != "" {
			token.NotificationMessage = payload.NotificationMessage
		}
		return token, nil
	}

	var tokenResp struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken string  `json:"refresh_token"`
		ExpiresIn    int     `json:"expires_in"`
		ResourceUrl  *string `json:"resource_url,omitempty"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.AccessToken == "" || tokenResp.ExpiresIn <= 0 {
		return nil, fmt.Errorf("oauth returned incomplete token payload")
	}
	token := &OAuthToken{
		Access:  tokenResp.AccessToken,
		Refresh: tokenResp.RefreshToken,
		Expires: time.Now().UnixMilli() + int64(tokenResp.ExpiresIn)*1000,
	}
	if tokenResp.ResourceUrl != nil && *tokenResp.ResourceUrl != "" {
		token.ResourceUrl = tokenResp.ResourceUrl
	}
	return token, nil
}
