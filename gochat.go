package gochat

import (
	"context"

	"github.com/DotNetAge/gochat/client/anthropic"
	"github.com/DotNetAge/gochat/client/azureopenai"
	"github.com/DotNetAge/gochat/client/deepseek"
	"github.com/DotNetAge/gochat/client/ollama"
	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

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
	Init(config core.Config) ClientBuilder
	Temperature(temp float64) ClientBuilder
	Model(model string) ClientBuilder
	MaxTokens(max int) ClientBuilder
	Stop(stop ...string) ClientBuilder
	TopP(top float64) ClientBuilder
	EnableThinking(think bool) ClientBuilder
	ThinkingBudget(budget int) ClientBuilder
	EnableSearch(search bool) ClientBuilder
	IncrementalOutput(enabled bool) ClientBuilder // 流式增量输出（DeepSeek, Qwen）
	Format(format string) ClientBuilder           // 响应格式，如 "json"（Ollama）
	KeepAlive(duration string) ClientBuilder      // 模型内存保持时长（Ollama）
	UsageCallback(fn func(core.Usage)) ClientBuilder
	Attach(attachments ...core.Attachment) ClientBuilder
	UserMessage(msg string) ClientBuilder
	SysemMessage(msg string) ClientBuilder
	AssistantMessage(msg string) ClientBuilder
	Tools(tool ...core.Tool) ClientBuilder
	GetResponse(clientType ClientType) (*core.Response, error)
	GetStream(clientType ClientType) (*core.Stream, error)
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

// Init 初始化客户端配置
func (b *defaultClientBuilder) Init(config core.Config) ClientBuilder {
	b.config = config
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

// AttachFile 附加文件
func (b *defaultClientBuilder) Attach(attachments ...core.Attachment) ClientBuilder {
	b.options = append(b.options, core.WithAttachments(attachments...))
	return b
}

// UserMessage 添加用户消息
func (b *defaultClientBuilder) UserMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewUserMessage(msg))
	return b
}

// SysemMessage 添加系统消息（注意：接口中拼写为 SysemMessage）
func (b *defaultClientBuilder) SysemMessage(msg string) ClientBuilder {
	b.messages = append(b.messages, core.NewSystemMessage(msg))
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

// GetResponse 获取非流式响应
func (b *defaultClientBuilder) GetResponse(clientType ClientType) (*core.Response, error) {
	client, err := b.BuildFor(clientType)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return client.Chat(ctx, b.messages, b.options...)
}

// GetStream 获取流式响应
func (b *defaultClientBuilder) GetStream(clientType ClientType) (*core.Stream, error) {
	client, err := b.BuildFor(clientType)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return client.ChatStream(ctx, b.messages, b.options...)
}
