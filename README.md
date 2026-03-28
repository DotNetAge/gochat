<div align="center">
<h1>🚀 GoChat</h1>

<p><b>
GoChat is a **modern, enterprise-ready Go client SDK for Large Language Models (LLMs)**. It provides an exceptionally elegant and type-safe unified interface that completely smooths out the chaotic API differences between OpenAI, Anthropic (Claude), DeepSeek, Qwen, Ollama, and other major cloud providers or local models.

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

## ✨ Core Killer Features (Why GoChat?)

- **🔌 Write Once, Run Anywhere (The Ultimate Client)**
  - Smooths out the chaotic API differences and streaming SSE formats across OpenAI, Anthropic, DeepSeek, and Qwen.
  - **Standardized Tool Calling**: Define your tools via `core.Tool` once, and let GoChat translate them into Anthropic's unique format or OpenAI's standard format automatically under the hood.
  - **Smart Resilience**: Built-in exponential backoff and jitter quietly handles HTTP 429 Rate Limits and network hiccups, preventing your app from crashing in production.
- **🧠 Zero-Dependency Local Embeddings**
  - **Break free from Ollama & Python**: Execute ONNX text embeddings locally right within your pure Go binary. No heavy external inference servers needed!
  - **Built-in Auto-Downloader**: Simply call `embedding.WithBEG("bge-small-zh-v1.5", "")`. GoChat will automatically fetch the model from HuggingFace, cache it, and load it into memory.
  - **Industrial Batching**: Features concurrent batching and a zero-cost text hashing cache designed to process massive datasets efficiently.
- **🌊 Elegant Type-Safe Pipeline (Generics)**
  - Build complex Agent and RAG workflows elegantly by chaining independent, reusable `Step`s.
  - **Type-Safe Context**: Powered by Go 1.24+ generics, safely pass your own custom `struct` between steps. Say goodbye to fragile `map[string]any` type-assertion panics.
  - **Declarative Control Flow**: Utilize `NewIf`, `NewLoop`, and AOP lifecycle `Hooks` for observable and clean orchestration without nested `if err != nil` hell.

---

## 📦 Installation

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 Quick Start

### 1. High-Performance Embedding (Local or Remote)

```go
// Use BatchProcessor for optimized vector generation
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,
    MaxConcurrent: 4,
})

// Generate embeddings with progress tracking
embeddings, err := processor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
    fmt.Printf("Progress: %d/%d\n", current, total)
    return true // Return false to cancel
})
```

### 2. Streamlined Pipeline Execution

```go
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep("User question: {{.query}}", "prompt", "query")).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o")).
    AddHook(myLogger) // Observe every step

state := pipeline.NewState()
state.Set("query", "What is GoChat?")

err := p.Execute(ctx, state)
fmt.Println(state.GetString("answer"))
```

### 3. Capturing the "Chain of Thought" in a Stream

```go
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))
defer stream.Close()

for stream.Next() {
    ev := stream.Event()
    if ev.Type == core.EventThinking {
        fmt.Print(ev.Content) // Reasoning output
    } else if ev.Type == core.EventContent {
        fmt.Print(ev.Content) // Final answer
    }
}
```

---

## 🔌 Fully Supported Providers

| Provider          | Models              | Auth Methods                 |
| :---------------- | :------------------ | :--------------------------- |
| **OpenAI**        | GPT-4o, o1, o3-mini | API Key                      |
| **Anthropic**     | Claude 3.5/3.7      | API Key                      |
| **DeepSeek**      | V3, R1              | API Key                      |
| **Alibaba Qwen**  | Qwen-Max, Qwen-Plus | API Key, OAuth2, Device Code |
| **Google Gemini** | 1.5 Pro/Flash       | API Key, OAuth2              |
| **Local / ONNX**  | BGE, Sentence-BERT  | Local Execution              |
| **Azure OpenAI**  | All GPT models      | API Key (Azure format)       |

---

## 🎯 Design Philosophy

GoChat adheres to Go's philosophy of minimalism: The core interface `core.Client` has only two methods: `Chat` and `ChatStream`. All provider-specific customizations are elegantly extended via **Functional Options**, ensuring the main interface remains clean and stable.

## 📄 License

This project is open-sourced under the [MIT License](LICENSE). PRs are welcome!

---

## 📚 Comprehensive Documentation

Check out the `docs/` folder for detailed guides, architecture diagrams, and API references:
- 📖 [Project Overview](docs/overview.md)
- 🚀 [Quick Start](docs/quickstart.md)
- 🧠 [Client Module (LLM & Tool Calling)](docs/modules/client.md)
- 🧬 [Embedding Module (Local Vectorization)](docs/modules/embedding.md)
- ⛓️ [Pipeline Module (Workflow Orchestration)](docs/modules/pipeline.md)
- 🏢 [Provider Module (OAuth2 & Providers)](docs/modules/provider.md)
- 📋 [API Reference](docs/api_reference.md)
