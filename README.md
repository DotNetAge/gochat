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

<p>

[**English**](README.md) | [简体中文](README_zh.md)

</p>
</div>

---

## ✨ New & Killer Features

- **🔌 Seamless Model Switching**: Change one line of initialization code to switch your application effortlessly between GPT-4o, Claude 3.7, DeepSeek-R1, and Qwen-Max.
- **🧠 Native Deep Thinking Support**: Built-in interceptors compatible with the reasoning chains of DeepSeek-R1, Claude 3.7, and OpenAI o1/o3.
- **🧬 Unified Embedding System**: 
    - Support for both **Remote APIs** (OpenAI/Azure) and **Local Models** (ONNX/BGE/Sentence-BERT).
    - High-performance **Batch Processing** with concurrent execution and atomic progress tracking.
    - Built-in **LRU Caching** to avoid redundant vector calculations.
- **⛓️ Modular Pipeline Framework**: 
    - Orchestrate complex LLM workflows (RAG, multi-step reasoning) using a clean **Step-based** architecture.
    - Thread-safe **State Management** and execution **Hooks** for full observability.
- **🌐 Web Search at Will**: Native support for models with external web retrieval (like Qwen) via `core.WithEnableSearch(true)`.
- **🏢 Enterprise OAuth2 Persistence**: Automates Device Code Flow / OAuth2 authorization with persistent Token storage and auto-refreshing.

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
p := pipeline.New().
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
