<div align="center">
<h1>🚀 GoChat</h1>

<p><b>
GoChat 是一个专为生产环境打造的**现代化 Go 语言大模型 (LLM) 客户端 SDK**。它通过极其优雅且统一的类型安全接口，帮您彻底抹平 OpenAI、Anthropic (Claude)、DeepSeek、Qwen、Ollama 等各大云厂商及本地模型之间繁杂的 API 差异。
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

## ✨ 核心杀手锏 (为什么选择 GoChat？)

- **🔌 终极的“一次编写，到处运行” (Client 模块)**
  - 彻底抹平 OpenAI、Anthropic (Claude)、DeepSeek、Qwen 等模型间繁杂的 API 结构与流式解析差异。
  - **大一统的 Tool Calling**：只需定义一次 `core.Tool`，框架底层会自动将其“翻译”为对应厂商（如 Anthropic 独有的）工具调用格式，无缝切换底层模型。
  - **内置防脆弱机制**：底层自动捕获 HTTP 429 限流与网络波动，触发带抖动的指数退避（Exponential Backoff）重试，让您的服务稳如泰山。
- **🧠 脱离 Ollama 的“零配置”本地向量化 (Embedding)**
  - **摆脱臃肿依赖**：直接基于轻量级 ONNX 运行时在本地计算 Embedding。无需额外部署庞大的 Ollama 服务，也无需折腾复杂的 Python 环境。
  - **内置极客下载器**：只需一行 `embedding.WithBEG("bge-small-zh-v1.5", "")`，本地缺失时自动从远端镜像分片拉取模型并加载就绪。
  - **工业级批处理**：内置 `BatchProcessor`，支持动态并发切批，自动对相同文本进行 Hash 缓存过滤，极限榨取 CPU 算力。
- **🌊 将极度复杂的逻辑变优雅 (泛型 Pipeline)**
  - 像搭积木一样串联独立的 `Step`，优雅编排复杂的 RAG 或 Agent 思考流。
  - **强类型上下文流转**：得益于 Go 1.24+ 泛型，您可在 Step 间无缝传递自定义的强类型 `struct`。彻底告别传统 `map[string]any` 带来的类型断言崩溃与拼写错误。
  - **可编程控制流**：内置 `IfStep`、`LoopStep` 与 AOP 监控钩子 (`Hooks`)，赋予工作流超强的可观测性与调度域控制力。

---

## 📦 安装

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 极速入门

### 1. 高性能向量化 (Embedding)

```go
// 使用 BatchProcessor 优化向量生成
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,
    MaxConcurrent: 4,
})

// 生成向量并追踪进度
embeddings, err := processor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
    fmt.Printf("进度: %d/%d\n", current, total)
    return true // 返回 false 可取消任务
})
```

### 2. 流水线 (Pipeline) 工作流

```go
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep("用户问题: {{.query}}", "prompt", "query")).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o")).
    AddHook(myLogger) // 观察每一步的执行

state := pipeline.NewState()
state.Set("query", "GoChat 是什么？")

err := p.Execute(ctx, state)
fmt.Println(state.GetString("answer"))
```

### 3. 在流式输出中捕获“思维链”

```go
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))
defer stream.Close()

for stream.Next() {
    ev := stream.Event()
    if ev.Type == core.EventThinking {
        fmt.Print(ev.Content) // 打印思考过程
    } else if ev.Type == core.EventContent {
        fmt.Print(ev.Content) // 打印最终答案
    }
}
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
| **Local / ONNX**  | BGE, Sentence-BERT  | 本地运行 (无需 Key)          |
| **Azure OpenAI**  | 微软云部署模型      | API Key (Azure 格式)         |

---

## 🎯 设计理念

GoChat 秉承 Go 的极简大道：核心接口 `core.Client` 只有 `Chat` 和 `ChatStream` 两个方法。所有个性化功能全部通过 **Functional Options** 优雅扩展，确保主接口长期稳定且不被污染。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎提交 PR 一起共建！

---

## 📚 详细文档 (Documentation)

请查阅 `docs/` 目录获取详细的指南、架构图与 API 参考：
- 📖 [项目概述](docs/overview.md)
- 🚀 [快速入门](docs/quickstart.md)
- 🧠 [Client 模块 (大模型与工具调用)](docs/modules/client.md)
- 🧬 [Embedding 模块 (本地向量化)](docs/modules/embedding.md)
- ⛓️ [Pipeline 模块 (工作流编排)](docs/modules/pipeline.md)
- 🏢 [Provider 模块 (OAuth2 鉴权)](docs/modules/provider.md)
- 📋 [API 参考](docs/api_reference.md)
