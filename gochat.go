package gochat

import (
	"context"
	"time"

	"github.com/DotNetAge/gochat/client/anthropic"
	"github.com/DotNetAge/gochat/client/azureopenai"
	"github.com/DotNetAge/gochat/client/deepseek"
	"github.com/DotNetAge/gochat/client/ollama"
	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

// Option defines the configuration options for ClientBuilder
type Option func(*core.Config)

// WithAPIKey sets the API key
func WithAPIKey(key string) Option {
	return func(c *core.Config) { c.APIKey = key }
}

// WithAuthToken sets the auth token
func WithAuthToken(token string) Option {
	return func(c *core.Config) { c.AuthToken = token }
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) Option {
	return func(c *core.Config) { c.BaseURL = url }
}

// WithModel sets the model name
func WithModel(model string) Option {
	return func(c *core.Config) { c.Model = model }
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *core.Config) { c.Timeout = timeout }
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(retries int) Option {
	return func(c *core.Config) { c.MaxRetries = retries }
}

type ClientType int

const (
	OpenAIClient ClientType = iota
	AnthropicClient
	DeepSeekClient
	AzureClient
	OllamaClient
	QwenClient // 阿里云通义千问（使用 DashScope API）
)

type ClientBuilder interface {
	Config(opts ...Option) ClientBuilder            // 用Option去设置 core.config
	Messages(messages ...core.Message) ClientBuilder // 批量设置消息
	Temperature(temp float64) ClientBuilder          // 采样温度，控制模型生成文本的多样性。temperature越高，生成的文本更多样，反之，生成的文本更确定。
	Model(model string) ClientBuilder                // 用于指定模型名称
	MaxTokens(max int) ClientBuilder                 // 用于限制模型输出的最大 Token 数。若生成内容超过此值，生成将提前停止，且返回的finish_reason为length。
	Stop(stop ...string) ClientBuilder               // 用于指定停止词。当模型生成的文本中出现stop 指定的字符串或token_id时，生成将立即终止。
	TopP(top float64) ClientBuilder                  // 核采样的概率阈值，控制模型生成文本的多样性。
	TopK(top int) ClientBuilder                      // 指定生成过程中用于采样的候选 Token 数量。值越大，输出越随机；值越小，输出越确定。若设为 null 或大于 100，则禁用 top_k 策略，仅 top_p 策略生效。取值必须为大于或等于 0 的整数。
	EnableThinking(think bool) ClientBuilder         // 启用扩展思考/推理功能
	ThinkingBudget(budget int) ClientBuilder
	EnableSearch(search bool) ClientBuilder          // 启用搜索功能
	IncrementalOutput(enabled bool) ClientBuilder    // 流式增量输出（DeepSeek, Qwen）
	Format(format string) ClientBuilder              // 响应格式，如 "json"（Ollama）
	KeepAlive(duration string) ClientBuilder         // 模型内存保持时长（Ollama）
	UsageCallback(fn func(core.Usage)) ClientBuilder // 获取用量的返回值
	Attach(attachments ...core.Attachment) ClientBuilder
	UserMessage(msg string) ClientBuilder
	SystemMessage(msg string) ClientBuilder    // 独立设置系统消息
	DeveloperMessage(msg string) ClientBuilder // 独立设置开发者消息 (OpenAI o1/o3)
	AssistantMessage(msg string) ClientBuilder
	Tools(tool ...core.Tool) ClientBuilder
	ToolChoice(choice interface{}) ClientBuilder  // 设置工具选择策略，支持 string (如 "auto") 或具体对象
	ParallelToolCalls(parallel bool) ClientBuilder // 是否开启并行工具调用
	PresencePenalty(penalty float64) ClientBuilder
	FrequencyPenalty(penalty float64) ClientBuilder // 频率惩罚
	ResponseFormat(format string) ClientBuilder     // 设置响应格式， "json" / "text"
	GetResponse() (*core.Response, error)           // 获取默认（OpenAI）的响应
	GetResponseFor(clientType ClientType) (*core.Response, error)
	GetStream() (*core.Stream, error) // 获取默认（OpenAI）的流
	GetStreamFor(clientType ClientType) (*core.Stream, error)
	Build() (core.Client, error) // 构建并返回客户端（openai)
	BuildFor(clientType ClientType) (core.Client, error)
}

// defaultClientBuilder 是 ClientBuilder 接口的默认实现
type defaultClientBuilder struct {
	config   core.Config
	options  []core.Option
	messages []core.Message
	tools    []core.Tool

	// Azure 特有配置
	azureEndpoint   string
	azureAPIVersion string
}

// Client 创建一个新的 ClientBuilder 实例
func Client() ClientBuilder {
	return &defaultClientBuilder{
		options:  make([]core.Option, 0),
		messages: make([]core.Message, 0),
		tools:    make([]core.Tool, 0),
	}
}

// Config 初始化客户端配置
func (b *defaultClientBuilder) Config(opts ...Option) ClientBuilder {
	for _, opt := range opts {
		opt(&b.config)
	}
	return b
}

// Messages 批量设置消息
func (b *defaultClientBuilder) Messages(messages ...core.Message) ClientBuilder {
	b.messages = append(b.messages, messages...)
	return b
}

// Temperature 设置生成温度
func (b *defaultClientBuilder) Temperature(temp float64) ClientBuilder {
	b.options = append(b.options, core.WithTemperature(temp))
	return b
}

// Model 设置模型名称
func (b *defaultClientBuilder) Model(model string) ClientBuilder {
	b.options = append(b.options, core.WithModel(model))
	return b
}

// MaxTokens 设置最大生成 token 数
func (b *defaultClientBuilder) MaxTokens(max int) ClientBuilder {
	b.options = append(b.options, core.WithMaxTokens(max))
	return b
}

// Stop 设置停止序列
func (b *defaultClientBuilder) Stop(stop ...string) ClientBuilder {
	b.options = append(b.options, core.WithStop(stop...))
	return b
}

// TopP 设置 top-p 采样参数
func (b *defaultClientBuilder) TopP(top float64) ClientBuilder {
	b.options = append(b.options, core.WithTopP(top))
	return b
}

// TopK 设置 top-k 采样参数
func (b *defaultClientBuilder) TopK(top int) ClientBuilder {
	b.options = append(b.options, core.WithTopK(top))
	return b
}

// EnableThinking 启用扩展思考/推理功能
func (b *defaultClientBuilder) EnableThinking(think bool) ClientBuilder {
	if think {
		b.options = append(b.options, core.WithThinking(0))
	}
	return b
}

// ThinkingBudget 设置思考 token 预算
func (b *defaultClientBuilder) ThinkingBudget(budget int) ClientBuilder {
	b.options = append(b.options, core.WithThinking(budget))
	return b
}

// EnableSearch 启用搜索功能
func (b *defaultClientBuilder) EnableSearch(search bool) ClientBuilder {
	b.options = append(b.options, core.WithEnableSearch(search))
	return b
}

// IncrementalOutput 启用流式增量输出（DeepSeek, Qwen）
func (b *defaultClientBuilder) IncrementalOutput(enabled bool) ClientBuilder {
	b.options = append(b.options, core.WithIncrementalOutput(enabled))
	return b
}

// Format 设置响应格式，如 "json"（Ollama）
func (b *defaultClientBuilder) Format(format string) ClientBuilder {
	b.options = append(b.options, core.WithFormat(format))
	return b
}

// KeepAlive 设置模型内存保持时长（Ollama）
func (b *defaultClientBuilder) KeepAlive(duration string) ClientBuilder {
	b.options = append(b.options, core.WithKeepAlive(duration))
	return b
}

// UsageCallback 设置用量回调函数
func (b *defaultClientBuilder) UsageCallback(fn func(core.Usage)) ClientBuilder {
	b.options = append(b.options, core.WithUsageCallback(fn))
	return b
}

// Attach 附加文件
func (b *defaultClientBuilder) Attach(attachments ...core.Attachment) ClientBuilder {
	b.options = append(b.options, core.WithAttachments(attachments...))
	return b
}

// UserMessage 添加用户消息
func (b *defaultClientBuilder) UserMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewUserMessage(msg))
	return b
}

// SystemMessage 添加系统消息
func (b *defaultClientBuilder) SystemMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewSystemMessage(msg))
	return b
}

// DeveloperMessage 添加开发者消息
func (b *defaultClientBuilder) DeveloperMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewDeveloperMessage(msg))
	return b
}

// AssistantMessage 添加助手消息
func (b *defaultClientBuilder) AssistantMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewTextMessage(core.RoleAssistant, msg))
	return b
}

// Tools 设置可用工具
func (b *defaultClientBuilder) Tools(tool ...core.Tool) ClientBuilder {
	b.tools = append(b.tools, tool...)
	b.options = append(b.options, core.WithTools(tool...))
	return b
}

// ToolChoice 设置工具选择策略
func (b *defaultClientBuilder) ToolChoice(choice interface{}) ClientBuilder {
	b.options = append(b.options, core.WithToolChoice(choice))
	return b
}

// ParallelToolCalls 设置是否开启并行工具调用
func (b *defaultClientBuilder) ParallelToolCalls(parallel bool) ClientBuilder {
	b.options = append(b.options, core.WithParallelToolCalls(parallel))
	return b
}

// PresencePenalty 设置 presence penalty
func (b *defaultClientBuilder) PresencePenalty(penalty float64) ClientBuilder {
	b.options = append(b.options, core.WithPresencePenalty(penalty))
	return b
}

// FrequencyPenalty 设置频率惩罚
func (b *defaultClientBuilder) FrequencyPenalty(penalty float64) ClientBuilder {
	b.options = append(b.options, core.WithFrequencyPenalty(penalty))
	return b
}

// ResponseFormat 设置响应格式
func (b *defaultClientBuilder) ResponseFormat(format string) ClientBuilder {
	b.options = append(b.options, core.WithResponseFormat(format))
	return b
}

// SetAzureEndpoint 设置 Azure OpenAI 端点（仅 Azure 客户端需要）
func (b *defaultClientBuilder) SetAzureEndpoint(endpoint string) ClientBuilder {
	b.azureEndpoint = endpoint
	return b
}

// SetAzureAPIVersion 设置 Azure API 版本（仅 Azure 客户端需要）
func (b *defaultClientBuilder) SetAzureAPIVersion(version string) ClientBuilder {
	b.azureAPIVersion = version
	return b
}

// buildClient 根据 ClientType 构建对应的客户端
func (b *defaultClientBuilder) buildClient(clientType ClientType) (core.Client, error) {
	switch clientType {
	case OpenAIClient:
		return openai.NewOpenAI(b.config)
	case AnthropicClient:
		return anthropic.NewAnthropic(b.config)
	case DeepSeekClient:
		return deepseek.NewDeepSeek(b.config)
	case AzureClient:
		azureConfig := azureopenai.Config{
			Config:     b.config,
			Endpoint:   b.azureEndpoint,
			APIVersion: b.azureAPIVersion,
		}
		return azureopenai.NewAzureOpenAI(azureConfig)
	case OllamaClient:
		return ollama.NewOllamaClient(b.config)
	case QwenClient:
		// Qwen uses DashScope OpenAI-compatible mode
		qwenConfig := b.config
		if qwenConfig.BaseURL == "" {
			qwenConfig.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		return openai.NewOpenAI(qwenConfig)
	default:
		return nil, core.NewValidationError("unknown client type", nil)
	}
}

// Build 构建并返回默认客户端（OpenAI）
func (b *defaultClientBuilder) Build() (core.Client, error) {
	return b.BuildFor(OpenAIClient)
}

// BuildFor 根据 ClientType 构建并返回对应的客户端
func (b *defaultClientBuilder) BuildFor(clientType ClientType) (core.Client, error) {
	return b.buildClient(clientType)
}

// GetResponse 获取默认（OpenAI）的非流式响应
func (b *defaultClientBuilder) GetResponse() (*core.Response, error) {
	return b.GetResponseFor(OpenAIClient)
}

// GetResponseFor 获取指定客户端的非流式响应
func (b *defaultClientBuilder) GetResponseFor(clientType ClientType) (*core.Response, error) {
	client, err := b.BuildFor(clientType)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return client.Chat(ctx, b.messages, b.options...)
}

// GetStream 获取默认（OpenAI）的流式响应
func (b *defaultClientBuilder) GetStream() (*core.Stream, error) {
	return b.GetStreamFor(OpenAIClient)
}

// GetStreamFor 获取指定客户端的流式响应
func (b *defaultClientBuilder) GetStreamFor(clientType ClientType) (*core.Stream, error) {
	client, err := b.BuildFor(clientType)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return client.ChatStream(ctx, b.messages, b.options...)
}
