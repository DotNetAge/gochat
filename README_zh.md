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


<p>

[English](README.md) | [**简体中文**](README_zh.md)

</p>
</div>



---

## ✨ 新特性与核心优势

- **🔌 一键无缝切模型**：只需改一行代码，即可在 GPT-4o、Claude 3.7、DeepSeek-R1、Qwen-Max 之间自由切换。
- **🧠 原生支持深度思考 (Reasoning)**：内置拦截器，完美兼容 DeepSeek-R1、Claude 3.7、OpenAI o1/o3 的思维链。
- **🧬 统一向量化 (Embedding) 系统**：
    - 支持 **远程 API** (OpenAI/Azure) 与 **本地模型** (ONNX/BGE/Sentence-BERT)。
    - 高性能 **批量处理 (Batch Processing)**，支持并发执行与原子化进度追踪。
    - 内置 **LRU 缓存** 机制，避免重复的向量计算，节省成本。
- **⛓️ 模块化流水线 (Pipeline) 框架**：
    - 使用清晰的 **Step 架构** 编排复杂的 LLM 工作流（如 RAG、多步推理）。
    - 线程安全的 **状态管理** 与执行 **Hooks**，实现全方位的可观测性。
- **🌐 随心开启联网搜索**：原生支持 Qwen 等具备外网检索能力的模型，一行代码即可开启。
- **🏢 企业级 OAuth2 持久化鉴权**：全自动处理 Device Code Flow / OAuth2 授权、Token 持久化存储与自动刷新。

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
p := pipeline.New().
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
