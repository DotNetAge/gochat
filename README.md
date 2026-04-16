<div align="center">
<h1>🚀 GoChat</h1>

<p><b>
GoChat is a <b>modern, enterprise-ready Go client SDK for Large Language Models (LLMs)</b>. It provides an exceptionally elegant and type-safe unified interface that completely smooths out the chaotic API differences between OpenAI, Anthropic (Claude), DeepSeek, Qwen, Ollama, and other major cloud providers or local models.
</b></p>

[![Go Reference](https://pkg.go.dev/badge/github.com/DotNetAge/gochat.svg)](https://pkg.go.dev/github.com/DotNetAge/gochat)
[![Go Version](https://img.shields.io/github/go-mod/go-version/DotNetAge/gochat)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/DotNetAge/gochat)](https://goreportcard.com/report/github.com/DotNetAge/gochat)
[![codecov](https://codecov.io/gh/DotNetAge/gochat/graph/badge.svg?token=placeholder)](https://codecov.io/gh/DotNetAge/gochat)
[![Docs](https://img.shields.io/badge/docs-gochat.rayainfo.cn-e92063.svg)](https://gochat.rayainfo.cn)


<p>

[<b>English</b>](README.md) | [简体中文](README_zh.md)

</p>
</div>



---

## ✨ Core Features (Why GoChat?)

- **🔌 Unified Interface**: Provides a consistent API interface that shields differences between different LLM providers
- **Unified Tool Calling**: Define tools once, automatically convert to the corresponding provider's tool calling format
- **Built-in Anti-Fragility Mechanism**: Automatically captures HTTP 429 rate limits and network fluctuations, triggering exponential backoff with jitter retry
- **Workflow Orchestration**: Elegantly organize complex RAG or Agent reasoning flows through Pipeline
- **Strongly-Typed Context Transfer**: Leverage Go 1.24+ generics to seamlessly pass custom strongly-typed structs between steps

---

## 📦 Installation

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 Quick Start

### 1. Create Client

```go
import (
    "github.com/DotNetAge/gochat/core"
    "github.com/DotNetAge/gochat/client/openai"
)

// Create OpenAI client
client, err := openai.NewOpenAI(core.Config{
    APIKey: "your-api-key",
    Model:  "gpt-4o",
})
if err != nil {
    log.Fatal(err)
}
```

### 2. Send Message

```go
import "github.com/DotNetAge/gochat/core"

// Create messages
messages := []core.Message{
    core.NewSystemMessage("You are a helpful assistant."),
    core.NewUserMessage("Hello, who are you?"),
}

// Send message
response, err := client.Chat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### 3. Streaming Response

```go
// Send streaming request
stream, err := client.ChatStream(ctx, messages)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

// Process streaming response
for stream.Next() {
    event := stream.Event()
    fmt.Print(event.Content)
}

if err := stream.Err(); err != nil {
    log.Fatal(err)
}
```

### 4. Use Pipeline

```go
import (
    "github.com/DotNetAge/gochat/pipeline"
    "github.com/DotNetAge/gochat/pipeline/steps"
)

// Create Pipeline
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep("User question: {{.query}}", "prompt", "query")).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o"))

// Create state
state := pipeline.NewState()
state.Set("query", "What is GoChat?")

// Execute Pipeline
err := p.Execute(ctx, state)
if err != nil {
    log.Fatal(err)
}

fmt.Println(state.GetString("answer"))
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

GoChat adopts a modular architecture design, separating different functions into independent packages to achieve high scalability and maintainability.

### Overall Architecture

```
gochat/
├── client/         # LLM provider client implementations
│   ├── anthropic/  # Anthropic (Claude) client
│   ├── azureopenai/ # Azure OpenAI client
│   ├── deepseek/   # DeepSeek client
│   ├── ollama/     # Ollama local model client
│   └── openai/     # OpenAI client
├── core/           # Core interfaces and common functionality
├── pipeline/       # Workflow orchestration functionality
├── provider/       # Additional provider implementations
├── docs/           # Documentation
└── examples/       # Example code
```

### Core Modules

- **Core Module**: Defines core interfaces and common functionality, serving as the foundation of the library
- **Client Module**: Contains specific implementations for various LLM providers
- **Pipeline Module**: Provides workflow orchestration functionality, allowing users to combine independent steps into complex processes
- **Provider Module**: Contains additional provider implementations such as Gemini, Minimax, and Qwen

## 4. Key Classes and Functions

### 4.1 Core Module

#### Client Interface

```go
type Client interface {
    Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)
    ChatStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error)
}
```

- **Parameters**:
  - `ctx`: Context for cancellation and timeout
  - `messages`: Conversation messages
  - `opts`: Optional parameters (temperature, max tokens, tools, etc.)

- **Return Values**:
  - `Chat`: Returns complete response and error
  - `ChatStream`: Returns event stream and error

#### Message-related Functions

- **NewUserMessage(text string) Message**: Creates a user message
- **NewSystemMessage(text string) Message**: Creates a system message
- **NewTextMessage(role, text string) Message**: Creates a text message

#### Option-related Functions

- **WithTemperature(t float64) Option**: Sets temperature
- **WithMaxTokens(t int) Option**: Sets maximum tokens
- **WithTools(tools []Tool) Option**: Sets tools
- **WithThinking(level int) Option**: Enables thinking mode

### 4.2 Client Module

#### OpenAI Client

```go
func NewOpenAI(config core.Config) (*Client, error)
```

- **Parameters**:
  - `config`: Contains API key, model name, base URL, etc.

- **Return Values**:
  - OpenAI client instance and error

#### Anthropic Client

```go
func NewAnthropic(config core.Config) (*Client, error)
```

- **Parameters**:
  - `config`: Contains API key, model name, etc.

- **Return Values**:
  - Anthropic client instance and error

### 4.3 Pipeline Module

#### Pipeline-related Functions

```go
func New[T any]() *Pipeline[T]
func (p *Pipeline[T]) AddStep(step Step[T]) *Pipeline[T]
func (p *Pipeline[T]) Execute(ctx context.Context, state T) error
```

- **Parameters**:
  - `step`: Step to add
  - `ctx`: Context for cancellation
  - `state`: State object passed to each step

- **Return Values**:
  - `New`: Returns new Pipeline instance
  - `AddStep`: Returns Pipeline instance for method chaining
  - `Execute`: Returns execution error

#### Step-related Functions

```go
func NewTemplateStep(template, outputKey, inputKeys ...string) Step[*State]
func NewGenerateCompletionStep(client core.Client, inputKey, outputKey, model string) Step[*State]
```

- **Parameters**:
  - `template`: Template string
  - `outputKey`: Output key
  - `inputKeys`: Input keys
  - `client`: LLM client
  - `model`: Model name

- **Return Values**:
  - Step instance

## 5. Dependencies

GoChat's main dependencies are as follows:

| Dependency | Purpose | Source |
|------------|---------|--------|
| Go 1.24+ | Basic language environment, supports generics | [golang.org](https://golang.org/) |
| net/http | HTTP client | Standard library |
| encoding/json | JSON serialization and deserialization | Standard library |
| context | Context management | Standard library |

## 🔧 Configuration and Deployment

### Configuration Options

GoChat's configuration is primarily managed through the `core.Config` struct:

```go
type Config struct {
    APIKey     string            // API key
    AuthToken  string            // Authentication token
    Model      string            // Model name
    BaseURL    string            // API base URL
    HTTPClient *http.Client      // Custom HTTP client
    Headers    map[string]string // Custom HTTP headers
}
```

### Environment Variables

GoChat supports reading configuration from environment variables:

- `GOCHAT_API_KEY`: API key
- `GOCHAT_MODEL`: Default model name
- `GOCHAT_BASE_URL`: API base URL

### Deployment Recommendations

- **Production Environment**: It is recommended to use environment variables to store API keys to avoid hardcoding
- **High Concurrency Scenarios**: It is recommended to use a custom HTTP client with reasonable timeout and connection pool configurations
- **Fault Tolerance**: It is recommended to implement retry mechanisms and error handling to improve system stability

## 📊 Monitoring and Maintenance

### Logging

GoChat supports logging through the Pipeline's Hook mechanism:

```go
// Implement Hook interface
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

// Add Hook
p := pipeline.New[*pipeline.State]().
    AddStep(step1).
    AddHook(&LoggerHook{})
```

### Error Handling

GoChat provides detailed error types:

- `ValidationError`: Parameter validation error
- `NetworkError`: Network error
- `APIError`: Error returned by the API
- `RateLimitError`: Rate limit error

It is recommended to implement appropriate error handling when using GoChat:

```go
response, err := client.Chat(ctx, messages)
if err != nil {
    switch e := err.(type) {
    case *core.RateLimitError:
        // Handle rate limit
        time.Sleep(e.RetryAfter)
    case *core.NetworkError:
        // Handle network error
        retryCount++
    default:
        // Handle other errors
        log.Fatal(err)
    }
}
```

## 📁 Example Code

GoChat provides rich example code in the `examples/` directory:

| Example | Functionality | Path |
|---------|---------------|------|
| 01_basic_chat | Basic chat functionality | [examples/01_basic_chat/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/01_basic_chat/main.go) |
| 02_multi_turn | Multi-turn conversation | [examples/02_multi_turn/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/02_multi_turn/main.go) |
| 03_streaming | Streaming response | [examples/03_streaming/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/03_streaming/main.go) |
| 04_tool_calling | Tool calling | [examples/04_tool_calling/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/04_tool_calling/main.go) |
| 05_multiple_providers | Multiple providers | [examples/05_multiple_providers/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/05_multiple_providers/main.go) |
| 06_image_input | Image input | [examples/06_image_input/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/06_image_input/main.go) |
| 07_document_analysis | Document analysis | [examples/07_document_analysis/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/07_document_analysis/main.go) |
| 08_multiple_images | Multiple image input | [examples/08_multiple_images/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/08_multiple_images/main.go) |
| 09_helper_utilities | Helper utilities | [examples/09_helper_utilities/main.go](file:///Users/ray/workspaces/ai-ecosystem/gochat/examples/09_helper_utilities/main.go) |

## 🎯 Design Philosophy

GoChat adheres to Go's philosophy of minimalism: The core interface `core.Client` has only two methods, `Chat` and `ChatStream`. All personalization features are elegantly extended through **Functional Options**, ensuring the main interface remains long-term stable and uncontaminated.

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

## 📝 Summary and Highlights

GoChat is a powerful, elegantly designed Go language LLM client SDK with the following core advantages:

- **Unified Interface**: Shield API differences between different LLM providers, achieving "write once, run anywhere"
- **Type Safety**: Leverage Go's type system and generics to provide a type-safe API
- **Powerful Workflow Orchestration**: Elegantly organize complex logic through Pipeline
- **Built-in Anti-Fragility Mechanism**: Automatically handle network fluctuations and rate limits
- **Rich Examples**: Provide comprehensive example code to help users get started quickly

With GoChat, developers can focus more on business logic rather than dealing with API differences between different LLM providers, thereby building LLM-based applications more efficiently.