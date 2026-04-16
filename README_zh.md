<div align="center">
<h1>🚀 GoChat</h1>

<p><b>
GoChat 是一个现代化、企业级的 Go 语言客户端 SDK，用于与大型语言模型 (LLMs) 交互。它提供了一个优雅且类型安全的统一接口，完全平滑了 OpenAI、Anthropic (Claude)、DeepSeek、Qwen、Ollama 等主要云提供商或本地模型之间的 API 差异。
</b></p>

[![Go Reference](https://pkg.go.dev/badge/github.com/DotNetAge/gochat.svg)](https://pkg.go.dev/github.com/DotNetAge/gochat)
[![Go Version](https://img.shields.io/github/go-mod/go-version/DotNetAge/gochat)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/gochat)](https://goreportcard.com/report/github.com/DotNetAge/gochat)
[![codecov](https://codecov.io/gh/DotNetAge/gochat/graph/badge.svg?token=placeholder)](https://codecov.io/gh/DotNetAge/gochat)
[![Docs](https://img.shields.io/badge/docs-gochat.rayainfo.cn-e92063.svg)](https://gochat.rayainfo.cn)


<p>

[English](README.md) | [**简体中文**](README_zh.md)

</p>
</div>



---

## ✨ 核心特性 (为什么选择 GoChat？)

- **🔌 统一接口**：提供一致的 API 接口，屏蔽不同 LLM 提供商的差异
- **统一工具调用**：定义一次工具，自动转换为对应提供商的工具调用格式
- **内置抗脆弱机制**：自动捕获 HTTP 429 速率限制和网络波动，触发指数退避和抖动重试
- **工作流编排**：通过 Pipeline 优雅地组织复杂的 RAG 或 Agent 推理流程
- **强类型上下文传递**：利用 Go 1.24+ 泛型，在步骤间无缝传递自定义强类型结构体

---

## 📦 安装

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 快速入门

### 1. 创建客户端

```go
import (
    "github.com/DotNetAge/gochat/core"
    "github.com/DotNetAge/gochat/client/openai"
)

// 创建 OpenAI 客户端
client, err := openai.NewOpenAI(core.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4o",
})
if err != nil {
    log.Fatal(err)
}
```

### 2. 发送消息

```go
import "github.com/DotNetAge/gochat/core"

// 创建消息
messages := []core.Message{
    core.NewSystemMessage("You are a helpful assistant."),
    core.NewUserMessage("Hello, who are you?"),
}

// 发送消息
response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### 3. 流式响应

```go
// 发送流式请求
stream, err := client.ChatStream(ctx, messages)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// 处理流式响应
for stream.Next() {
    event := stream.Event()
    fmt.Print(event.Content)
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

### 4. 使用 Pipeline

```go
import (
    "github.com/DotNetAge/gochat/pipeline"
    "github.com/DotNetAge/gochat/pipeline/steps"
)

// 创建 Pipeline
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep("User question: {{.query}}", "prompt", "query")).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o"))

// 创建状态
state := pipeline.NewState()
state.Set("query", "What is GoChat?")

// 执行 Pipeline
err := p.Execute(ctx, state)
if err != nil {
    log.Fatal(err)
}

fmt.Println(state.GetString("answer"))
```

---

## 🔌 全面支持的提供商

| 提供商            | 模型支持            | 鉴权方式                     |
| :---------------- | :------------------ | :--------------------------- |
| **OpenAI**        | GPT-4o, o1, o3-mini | API Key                      |
| **Anthropic**     | Claude 3.5/3.7      | API Key                      |
| **DeepSeek**      | V3, R1              | API Key                      |
| **Alibaba Qwen**  | 通义千问全系列      | API Key, OAuth2, Device Code |
| **Google Gemini** | 1.5 Pro/Flash       | API Key, OAuth2              |
| **Ollama**        | 本地部署模型        | 本地运行 (无需 Key)          |
| **Azure OpenAI**  | 微软云部署模型      | API Key (Azure 格式)         |

---

## 🏗️ 项目架构

GoChat 采用模块化的架构设计，将不同功能分离到独立的包中，实现了高度的可扩展性和可维护性。

### 整体架构

```
gochat/
├── client/         # 各 LLM 提供商的客户端实现
│   ├── anthropic/  # Anthropic (Claude) 客户端
│   ├── azureopenai/ # Azure OpenAI 客户端
│   ├── deepseek/   # DeepSeek 客户端
│   ├── ollama/     # Ollama 本地模型客户端
│   └── openai/     # OpenAI 客户端
├── core/           # 核心接口和通用功能
├── pipeline/       # 工作流编排功能
├── provider/       # 额外的提供商实现
├── docs/           # 文档
└── examples/       # 示例代码
```

### 核心模块

- **Core 模块**：定义了核心接口和通用功能，是整个库的基础
- **Client 模块**：包含了各个 LLM 提供商的具体实现
- **Pipeline 模块**：提供了工作流编排功能，允许用户将独立的步骤组合成复杂的流程
- **Provider 模块**：包含了一些额外的提供商实现，如 Gemini、Minimax 和 Qwen 等

## 4. 关键类与函数

### 4.1 Core 模块

#### Client 接口

```go
type Client interface {
    Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)
    ChatStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error)
}
```

- **参数**：
  - `ctx`：上下文，用于取消和超时
  - `messages`：对话消息
  - `opts`：可选参数（温度、最大令牌数、工具等）

- **返回值**：
  - `Chat`：返回完整响应和错误
  - `ChatStream`：返回事件流和错误

#### Message 相关函数

- **NewUserMessage(text string) Message**：创建用户消息
- **NewSystemMessage(text string) Message**：创建系统消息
- **NewTextMessage(role, text string) Message**：创建文本消息

#### Option 相关函数

- **WithTemperature(t float64) Option**：设置温度
- **WithMaxTokens(t int) Option**：设置最大令牌数
- **WithTools(tools []Tool) Option**：设置工具
- **WithThinking(level int) Option**：启用思考模式

### 4.2 Client 模块

#### OpenAI 客户端

```go
func NewOpenAI(config core.Config) (*Client, error)
```

- **参数**：
  - `config`：包含 API 密钥、模型名称、基础 URL 等配置

- **返回值**：
  - OpenAI 客户端实例和错误

#### Anthropic 客户端

```go
func NewAnthropic(config core.Config) (*Client, error)
```

- **参数**：
  - `config`：包含 API 密钥、模型名称等配置

- **返回值**：
  - Anthropic 客户端实例和错误

### 4.3 Pipeline 模块

#### Pipeline 相关函数

```go
func New[T any]() *Pipeline[T]
func (p *Pipeline[T]) AddStep(step Step[T]) *Pipeline[T]
func (p *Pipeline[T]) Execute(ctx context.Context, state T) error
```

- **参数**：
  - `step`：要添加的步骤
  - `ctx`：上下文，用于取消
  - `state`：传递给每个步骤的状态对象

- **返回值**：
  - `New`：返回新的 Pipeline 实例
  - `AddStep`：返回 Pipeline 实例，用于方法链
  - `Execute`：返回执行错误

#### Step 相关函数

```go
func NewTemplateStep(template, outputKey, inputKeys ...string) Step[*State]
func NewGenerateCompletionStep(client core.Client, inputKey, outputKey, model string) Step[*State]
```

- **参数**：
  - `template`：模板字符串
  - `outputKey`：输出键
  - `inputKeys`：输入键
  - `client`：LLM 客户端
  - `model`：模型名称

- **返回值**：
  - Step 实例

## 5. 依赖关系

GoChat 的主要依赖如下：

| 依赖项 | 用途 | 来源 |
|-------|------|------|
| Go 1.24+ | 基础语言环境，支持泛型 | [golang.org](https://golang.org/) |
| net/http | HTTP 客户端 | 标准库 |
| encoding/json | JSON 序列化与反序列化 | 标准库 |
| context | 上下文管理 | 标准库 |

## 🔧 配置与部署

### 配置选项

GoChat 的配置主要通过 `core.Config` 结构体进行：

```go
type Config struct {
    APIKey     string            // API 密钥
    AuthToken  string            // 认证令牌
    Model      string            // 模型名称
    BaseURL    string            // API 基础 URL
    HTTPClient *http.Client      // 自定义 HTTP 客户端
    Headers    map[string]string // 自定义 HTTP 头
}
```

### 环境变量

GoChat 支持从环境变量中读取配置：

- `GOCHAT_API_KEY`：API 密钥
- `GOCHAT_MODEL`：默认模型名称
- `GOCHAT_BASE_URL`：API 基础 URL

### 部署建议

- **生产环境**：建议使用环境变量存储 API 密钥，避免硬编码
- **高并发场景**：建议使用自定义 HTTP 客户端，配置合理的超时和连接池
- **容错处理**：建议实现重试机制和错误处理，提高系统稳定性

## 📊 监控与维护

### 日志

GoChat 通过 Pipeline 的 Hook 机制支持日志记录：

```go
// 实现 Hook 接口
type LoggerHook struct{}

func (h *LoggerHook) OnStepStart(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("Step %s started\n", step.Name())
}

func (h *LoggerHook) OnStepComplete(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("Step %s completed\n", step.Name())
}

func (h *LoggerHook) OnStepError(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State, err error) {
    fmt.Printf("Step %s error: %v\n", step.Name(), err)
}

// 添加 Hook
p := pipeline.New[*pipeline.State]().
    AddStep(step1).
    AddHook(&LoggerHook{})
```

### 错误处理

GoChat 提供了详细的错误类型：

- `ValidationError`：参数验证错误
- `NetworkError`：网络错误
- `APIError`：API 返回的错误
- `RateLimitError`：速率限制错误

建议在使用时进行适当的错误处理：

```go
response, err := client.Chat(ctx, messages)
if err != nil {
    switch e := err.(type) {
    case *core.RateLimitError:
        // 处理速率限制
        time.Sleep(e.RetryAfter)
    case *core.NetworkError:
        // 处理网络错误
        retryCount++
    default:
        // 处理其他错误
        log.Fatal(err)
    }
}
```

## 📁 示例代码

GoChat 提供了丰富的示例代码，位于 `examples/` 目录中：

| 示例 | 功能 | 路径 |
|------|------|------|
| 01_basic_chat | 基本聊天功能 | [examples/01_basic_chat/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/01_basic_chat/main.go) |
| 02_multi_turn | 多轮对话 | [examples/02_multi_turn/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/02_multi_turn/main.go) |
| 03_streaming | 流式响应 | [examples/03_streaming/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/03_streaming/main.go) |
| 04_tool_calling | 工具调用 | [examples/04_tool_calling/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/04_tool_calling/main.go) |
| 05_multiple_providers | 多提供商 | [examples/05_multiple_providers/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/05_multiple_providers/main.go) |
| 06_image_input | 图像输入 | [examples/06_image_input/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/06_image_input/main.go) |
| 07_document_analysis | 文档分析 | [examples/07_document_analysis/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/07_document_analysis/main.go) |
| 08_multiple_images | 多图像输入 | [examples/08_multiple_images/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/08_multiple_images/main.go) |
| 09_helper_utilities | 辅助工具 | [examples/09_helper_utilities/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/09_helper_utilities/main.go) |

## 🎯 设计理念

GoChat 秉承 Go 的极简主义：核心接口 `core.Client` 只有两个方法，`Chat` 和 `ChatStream`。所有个性化功能都通过 **Functional Options** 优雅扩展，确保主接口长期稳定且不受污染。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎提交 PR 一起共建！

---

## 📚 详细文档

请查阅 `docs/` 目录获取详细的指南、架构图与 API 参考：
- 📖 [项目概述](https://gochat.rayainfo.cn/)
- 🚀 [快速入门](https://gochat.rayainfo.cn/quickstart/)
- 🧠 [Client 模块 (大模型与工具调用)](https://gochat.rayainfo.cn/modules/client/)
- ⛓️ [Pipeline 模块 (工作流编排)](https://gochat.rayainfo.cn/modules/pipeline/)
- 🏢 [Provider 模块 (OAuth2 鉴权)](https://gochat.rayainfo.cn/modules/provider/)
- 📋 [API 参考](https://gochat.rayainfo.cn/api_reference/)

## 📝 总结与亮点回顾

GoChat 是一个功能强大、设计优雅的 Go 语言 LLM 客户端 SDK，具有以下核心优势：

- **统一接口**：屏蔽不同 LLM 提供商的 API 差异，实现"一次编写，随处运行"
- **类型安全**：利用 Go 的类型系统和泛型，提供类型安全的 API
- **强大的工作流编排**：通过 Pipeline 实现复杂逻辑的优雅组织
- **内置抗脆弱机制**：自动处理网络波动和速率限制
- **丰富的示例**：提供了全面的示例代码，帮助用户快速上手

通过 GoChat，开发者可以更专注于业务逻辑，而不是处理不同 LLM 提供商的 API 差异，从而更高效地构建基于 LLM 的应用。