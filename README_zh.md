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
- **🏗️ Builder 模式**：链式调用，一行代码完成 LLM 请求
- **🔐 OAuth2 支持**：内置 Qwen、Gemini、MiniMax 的 OAuth2 设备码认证
- **🔗 工作流编排**：通过 Pipeline 优雅地组织复杂的 RAG 或 Agent 推理流程

---

## 📦 安装

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 快速入门

### 1. Client - LLM 调用

使用 Builder 模式，让 LLM 调用变得极其简单：

```go
import "github.com/DotNetAge/gochat"

// 创建 Builder 并发送请求
resp, err := gochat.NewClientBuilder().
    Init(gochat.Config{APIKey: "your-api-key"}).
    Model("gpt-4o").
    Temperature(0.7).
    UserMessage("你好，请介绍一下 Go 语言").
    GetResponse(gochat.OpenAIClient)

if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Content)
```

**流式响应：**

```go
stream, err := gochat.NewClientBuilder().
    Init(gochat.Config{APIKey: "your-api-key"}).
    Model("gpt-4o").
    UserMessage("写一首关于春天的诗").
    GetStream(gochat.OpenAIClient)

if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for stream.Next() {
    event := stream.Event()
    fmt.Print(event.Content)
}
```

**支持的客户端类型：**

| 常量 | 提供商 |
|------|--------|
| `gochat.OpenAIClient` | OpenAI / 兼容 API |
| `gochat.AnthropicClient` | Anthropic Claude |
| `gochat.DeepSeekClient` | DeepSeek |
| `gochat.OllamaClient` | Ollama 本地模型 |
| `gochat.AzureClient` | Azure OpenAI |

**更多 Builder 方法：**

```go
gochat.NewClientBuilder().
    Init(config).
    Model("gpt-4o").           // 设置模型
    Temperature(0.7).          // 设置温度
    MaxTokens(1000).           // 最大 token 数
    TopP(0.9).                 // Top-p 采样
    Stop("###", "END").        // 停止序列
    EnableThinking(true).      // 启用思考模式
    ThinkingBudget(1024).      // 思考预算
    EnableSearch(true).        // 启用搜索
    SysemMessage("你是一个助手"). // 系统消息
    UserMessage("用户消息").    // 用户消息
    AssistantMessage("助手消息"). // 助手消息
    AttachFile(attachment).    // 附加文件
    AttachImage(image).        // 附加图片
    Tools(tool).               // 工具
    UsageCallback(func(u core.Usage) {
        fmt.Printf("Token 使用: %d\n", u.TotalTokens)
    }).
    GetResponse(gochat.OpenAIClient)
```

---

### 2. Auth - OAuth2 认证

GoChat 内置了 Qwen、Gemini、MiniMax 的 OAuth2 设备码认证支持：

#### Qwen (通义千问)

```go
import "github.com/DotNetAge/gochat/auth"

// 创建 AuthManager
manager := auth.Qwen()

// 登录获取 token（会打印验证链接和验证码）
if err := manager.Login(); err != nil {
    log.Fatal(err)
}

// 获取 token 用于 API 调用
token, err := manager.GetToken()
if err != nil {
    log.Fatal(err)
}

// 使用 token 创建客户端
resp, err := gochat.NewClientBuilder().
    Init(core.Config{AuthToken: token.Access}).
    Model("qwen-max").
    UserMessage("你好").
    GetResponse(gochat.OpenAIClient)
```

#### Gemini

```go
// 需要提供 OAuth2 配置
manager := auth.GeminiWithConfig(
    "your-client-id",
    "your-client-secret",
    "http://localhost:8080/callback",
    ":8080",
)

if err := manager.Login(); err != nil {
    log.Fatal(err)
}
```

#### MiniMax

```go
// "cn" 表示国内版，其他值表示国际版
manager := auth.MiniMax("cn")

if err := manager.Login(); err != nil {
    log.Fatal(err)
}
```

**Token 持久化：**

```go
// 指定 token 文件路径
manager := auth.Qwen("/path/to/qwen_token.json")

// 首次登录
manager.Login()

// 后续启动时自动加载已保存的 token
token, err := manager.GetToken()
```

---

### 3. Pipeline - 工作流编排

Pipeline 用于组织复杂的 LLM 工作流：

```go
import (
    "github.com/DotNetAge/gochat"
    "github.com/DotNetAge/gochat/pipeline"
    "github.com/DotNetAge/gochat/pipeline/steps"
)

// 创建客户端
client, _ := openai.NewOpenAI(core.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4o",
})

// 创建 Pipeline
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep(
        "请回答以下问题：{{.question}}",
        "prompt",
        "question",
    )).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o"))

// 创建状态并设置输入
state := pipeline.NewState()
state.Set("question", "什么是 Go 语言？")

// 执行 Pipeline
if err := p.Execute(ctx, state); err != nil {
    log.Fatal(err)
}

// 获取结果
fmt.Println(state.GetString("answer"))
```

**内置 Step 类型：**

| Step | 功能 |
|------|------|
| `NewTemplateStep` | 模板渲染，将状态变量渲染成提示词 |
| `NewGenerateCompletionStep` | 调用 LLM 生成回复 |

**添加 Hook 监控执行：**

```go
type LoggerHook struct{}

func (h *LoggerHook) OnStepStart(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("[%s] 开始执行\n", step.Name())
}

func (h *LoggerHook) OnStepComplete(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("[%s] 执行完成\n", step.Name())
}

func (h *LoggerHook) OnStepError(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State, err error) {
    fmt.Printf("[%s] 执行失败: %v\n", step.Name(), err)
}

p := pipeline.New[*pipeline.State]().
    AddStep(step1).
    AddStep(step2).
    AddHook(&LoggerHook{})
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

```
gochat/
├── gochat.go       # Builder 模式入口
├── client/         # 各 LLM 提供商的客户端实现
├── core/           # 核心接口和通用功能
├── auth/           # OAuth2 认证模块
├── pipeline/       # 工作流编排功能
├── provider/       # 额外的提供商实现
├── docs/           # 文档
└── examples/       # 示例代码
```

---

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
