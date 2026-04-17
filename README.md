<div align="center">
<h1>🚀 GoChat</h1>

<p><b>
GoChat is a modern, enterprise-ready Go client SDK for Large Language Models (LLMs). It provides an elegant and type-safe unified interface that completely smooths out the API differences between OpenAI, Anthropic (Claude), DeepSeek, Qwen, Ollama, and other major cloud providers or local models.
</b></p>

[![Go Reference](https://pkg.go.dev/badge/github.com/DotNetAge/gochat.svg)](https://pkg.go.dev/github.com/DotNetAge/gochat)
[![Go Version](https://img.shields.io/github/go-mod/go-version/DotNetAge/gochat)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/gochat)](https://goreportcard.com/report/github.com/DotNetAge/gochat)
[![codecov](https://codecov.io/gh/DotNetAge/gochat/graph/badge.svg?token=placeholder)](https://codecov.io/gh/DotNetAge/gochat)
[![Docs](https://img.shields.io/badge/docs-gochat.rayainfo.cn-e92063.svg)](https://gochat.rayainfo.cn)


<p>

[**English**](README.md) | [简体中文](README_zh.md)

</p>
</div>



---

## ✨ Core Features (Why GoChat?)

- **🔌 Unified Interface**: Provides a consistent API interface that shields differences between different LLM providers
- **🏗️ Builder Pattern**: Chain calls, complete LLM requests in a single line of code
- **🔐 OAuth2 Support**: Built-in OAuth2 device code authentication for Qwen, Gemini, MiniMax
- **🔗 Workflow Orchestration**: Elegantly organize complex RAG or Agent reasoning flows through Pipeline

---

## 📦 Installation

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 Quick Start

### 1. Client - LLM Calls

Using the Builder pattern makes LLM calls extremely simple:

```go
import "github.com/DotNetAge/gochat"

// Create Builder and send request
resp, err := gochat.NewClientBuilder().
    Init(gochat.Config{APIKey: "your-api-key"}).
    Model("gpt-4o").
    Temperature(0.7).
    UserMessage("Hello, please introduce the Go language").
    GetResponse(gochat.OpenAIClient)

if err != nil {
    log.Fatal(err)
}
fmt.Println(resp.Content)
```

**Streaming Response:**

```go
stream, err := gochat.NewClientBuilder().
    Init(gochat.Config{APIKey: "your-api-key"}).
    Model("gpt-4o").
    UserMessage("Write a poem about spring").
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

**Supported Client Types:**

| Constant | Provider |
|----------|----------|
| `gochat.OpenAIClient` | OpenAI / Compatible API |
| `gochat.AnthropicClient` | Anthropic Claude |
| `gochat.DeepSeekClient` | DeepSeek |
| `gochat.OllamaClient` | Ollama Local Models |
| `gochat.AzureClient` | Azure OpenAI |

**More Builder Methods:**

```go
gochat.NewClientBuilder().
    Init(config).
    Model("gpt-4o").           // Set model
    Temperature(0.7).          // Set temperature
    MaxTokens(1000).           // Max tokens
    TopP(0.9).                 // Top-p sampling
    Stop("###", "END").        // Stop sequences
    EnableThinking(true).      // Enable thinking mode
    ThinkingBudget(1024).      // Thinking budget
    EnableSearch(true).        // Enable search
    SysemMessage("You are an assistant"). // System message
    UserMessage("User message").   // User message
    AssistantMessage("Assistant message"). // Assistant message
    AttachFile(attachment).    // Attach file
    AttachImage(image).        // Attach image
    Tools(tool).               // Tools
    UsageCallback(func(u core.Usage) {
        fmt.Printf("Token usage: %d\n", u.TotalTokens)
    }).
    GetResponse(gochat.OpenAIClient)
```

---

### 2. Auth - OAuth2 Authentication

GoChat has built-in OAuth2 device code authentication support for Qwen, Gemini, and MiniMax:

#### Qwen

```go
import "github.com/DotNetAge/gochat/auth"

// Create AuthManager
manager := auth.Qwen()

// Login to get token (will print verification link and code)
if err := manager.Login(); err != nil {
    log.Fatal(err)
}

// Get token for API calls
token, err := manager.GetToken()
if err != nil {
    log.Fatal(err)
}

// Use token to create client
resp, err := gochat.NewClientBuilder().
    Init(core.Config{AuthToken: token.Access}).
    Model("qwen-max").
    UserMessage("Hello").
    GetResponse(gochat.OpenAIClient)
```

#### Gemini

```go
// Need to provide OAuth2 configuration
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
// "cn" for China version, other values for international version
manager := auth.MiniMax("cn")

if err := manager.Login(); err != nil {
    log.Fatal(err)
}
```

**Token Persistence:**

```go
// Specify token file path
manager := auth.Qwen("/path/to/qwen_token.json")

// First login
manager.Login()

// Automatically load saved token on subsequent startups
token, err := manager.GetToken()
```

---

### 3. Pipeline - Workflow Orchestration

Pipeline is used to organize complex LLM workflows:

```go
import (
    "github.com/DotNetAge/gochat"
    "github.com/DotNetAge/gochat/pipeline"
    "github.com/DotNetAge/gochat/pipeline/steps"
)

// Create client
client, _ := openai.NewOpenAI(core.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4o",
})

// Create Pipeline
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep(
        "Please answer the following question: {{.question}}",
        "prompt",
        "question",
    )).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o"))

// Create state and set input
state := pipeline.NewState()
state.Set("question", "What is Go language?")

// Execute Pipeline
if err := p.Execute(ctx, state); err != nil {
    log.Fatal(err)
}

// Get result
fmt.Println(state.GetString("answer"))
```

**Built-in Step Types:**

| Step | Function |
|------|----------|
| `NewTemplateStep` | Template rendering, render state variables into prompts |
| `NewGenerateCompletionStep` | Call LLM to generate response |

**Add Hook to Monitor Execution:**

```go
type LoggerHook struct{}

func (h *LoggerHook) OnStepStart(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("[%s] Started\n", step.Name())
}

func (h *LoggerHook) OnStepComplete(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State) {
    fmt.Printf("[%s] Completed\n", step.Name())
}

func (h *LoggerHook) OnStepError(ctx context.Context, step pipeline.Step[*pipeline.State], state *pipeline.State, err error) {
    fmt.Printf("[%s] Failed: %v\n", step.Name(), err)
}

p := pipeline.New[*pipeline.State]().
    AddStep(step1).
    AddStep(step2).
    AddHook(&LoggerHook{})
```

---

## 🔌 Fully Supported Providers

| Provider            | Models              | Auth Methods                     |
| :------------------ | :------------------ | :------------------------------- |
| **OpenAI**          | GPT-4o, o1, o3-mini | API Key                          |
| **Anthropic**       | Claude 3.5/3.7      | API Key                          |
| **DeepSeek**        | V3, R1              | API Key                          |
| **Alibaba Qwen**    | Tongyi Qianwen series | API Key, OAuth2, Device Code   |
| **Google Gemini**   | 1.5 Pro/Flash       | API Key, OAuth2                  |
| **Ollama**          | Locally deployed models | Local Execution (No Key Required)|  
| **Azure OpenAI**    | Microsoft-deployed models | API Key (Azure format)      |

---

## 🏗️ Project Architecture

```
gochat/
├── gochat.go       # Builder pattern entry
├── client/         # LLM provider client implementations
├── core/           # Core interfaces and common functionality
├── auth/           # OAuth2 authentication module
├── pipeline/       # Workflow orchestration functionality
├── provider/       # Additional provider implementations
├── docs/           # Documentation
└── examples/       # Example code
```

---

## 📄 License

This project is open-sourced under the [MIT License](LICENSE). PRs are welcome!

---

## 📚 Comprehensive Documentation

Please refer to the `docs/` directory for detailed guides, architecture diagrams, and API references:
- 📖 [Project Overview](https://gochat.rayainfo.cn/)
- 🚀 [Quick Start](https://gochat.rayainfo.cn/quickstart/)
- 🧠 [Client Module (LLM & Tool Calling)](https://gochat.rayainfo.cn/modules/client/)
- ⛓️ [Pipeline Module (Workflow Orchestration)](https://gochat.rayainfo.cn/modules/pipeline/)
- 🏢 [Provider Module (OAuth2 Authentication)](https://gochat.rayainfo.cn/modules/provider/)
- 📋 [API Reference](https://gochat.rayainfo.cn/api_reference/)
