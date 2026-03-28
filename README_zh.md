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

### 4. 本地化向量模型调用

GoChat 的 embedding 包提供了完整的本地化向量模型支持，通过 ONNX 格式模型实现高效的文本嵌入生成。本地化方案无需 API Key，无网络依赖，可完全离线运行，适合对数据隐私和响应延迟有严格要求的场景。

#### 4.1 已实现的 Provider 完整列表

| Provider 类型            | 模型类型              | 向量维度 | 适用场景       | 特性说明                 |
| :----------------------- | :-------------------- | :------- | :------------- | :----------------------- |
| **BGEProvider**          | bge-small-zh-v1.5     | 512      | 中文语义相似度 | 高效中文嵌入，轻量级模型 |
| **BGEProvider**          | bge-base-zh-v1.5      | 768      | 中文语义相似度 | 更高精度，平衡性能与效果 |
| **SentenceBERTProvider** | all-MiniLM-L6-v2      | 384      | 英文语义搜索   | 极快速推理，适合实时应用 |
| **SentenceBERTProvider** | all-mpnet-base-v2     | 768      | 英文语义搜索   | 高精度 BERT 架构         |
| **CLIPProvider**         | clip-vit-base-patch32 | 512      | 多模态图文检索 | 支持文本与图像双向嵌入   |
| **LocalProvider**        | bert-base-uncased     | 768      | 通用英语任务   | 标准 BERT 模型           |

```go
import "gochat/pkg/embedding"

// Provider 接口定义
type Provider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
}

// 多模态 Provider (支持图像)
type MultimodalProvider interface {
    Provider
    EmbedImages(ctx context.Context, images [][]byte) ([][]float32, error)
}
```

#### 4.2 使用 Downloader 下载 ONNX 模型

Downloader 提供了从 HuggingFace 下载预编译 ONNX 模型的功能，支持进度跟踪和缓存管理。

```go
// 创建下载器 (指定缓存目录)
downloader := embedding.NewDownloader("~/.gochat/models")

// 查看可用模型列表
models := downloader.GetModelInfo()
for _, m := range models {
    fmt.Printf("模型: %s | 类型: %s | 大小: %s\n", m.Name, m.Type, m.Size)
}

// 下载进度回调函数
callback := func(modelName, fileName string, downloaded, total int64) {
    if total > 0 {
        percent := float64(downloaded) / float64(total) * 100
        fmt.Printf("\r[%s] %s: %.1f%%", modelName, fileName, percent)
    }
}

// 下载模型
modelPath, err := downloader.DownloadModel("bge-small-zh-v1.5", callback)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("\n模型已下载至: %s\n", modelPath)
```

**Downloader 核心方法说明：**

| 方法                            | 参数                               | 返回值                | 说明                 |
| :------------------------------ | :--------------------------------- | :-------------------- | :------------------- |
| `NewDownloader(cacheDir)`       | 缓存目录路径，空字符串使用默认目录 | `*Downloader`         | 创建下载器实例       |
| `GetModelInfo()`                | 无                                 | `[]DownloadModelInfo` | 返回所有可用模型信息 |
| `DownloadModel(name, callback)` | 模型名称、进度回调                 | `(string, error)`     | 下载指定模型         |

**支持的模型文件下载列表：**

| 模型名称              | 文件URL                                      | 预估大小 |
| :-------------------- | :------------------------------------------- | :------- |
| bge-small-zh-v1.5     | model_fp16.onnx, model_fp16.onnx_data        | ~48MB    |
| all-MiniLM-L6-v2      | model_fp16.onnx                              | ~45.3MB  |
| bert-base-uncased     | model_fp16.onnx                              | ~200MB   |
| bge-base-zh-v1.5      | model_fp16.onnx                              | ~100MB   |
| clip-vit-base-patch32 | text_model_fp16.onnx, vision_model_fp16.onnx | ~300MB   |
| all-mpnet-base-v2     | model_fp16.onnx                              | ~218MB   |

#### 4.3 本地化向量模型初始化配置

**方式一：自动下载并初始化 (推荐)**

```go
// 使用 BGE 模型，自动下载 (如未缓存)
provider, err := embedding.WithBEG("bge-small-zh-v1.5", "")
if err != nil {
    log.Fatal(err)
}

// 使用 BERT 模型
bertProvider, err := embedding.WithBERT("all-mpnet-base-v2", "")
if err != nil {
    log.Fatal(err)
}

// 使用 CLIP 多模态模型
clipProvider, err := embedding.WithCLIP("clip-vit-base-patch32", "")
if err != nil {
    log.Fatal(err)
}
```

**方式二：指定本地路径初始化**

```go
// 已下载模型，直接指定路径
provider, err := embedding.WithBEG("bge-small-zh-v1.5", "/path/to/model")
if err != nil {
    log.Fatal(err)
}

// 使用工厂函数创建
provider, err := embedding.NewProvider("/path/to/bge-small-zh-v1.5")
if err != nil {
    log.Fatal(err)
}

// 自定义配置创建 LocalProvider
localProvider, err := embedding.New(embedding.Config{
    Model:        model,
    Dimension:    512,
    MaxBatchSize: 32,
})
```

**方式三：直接创建特定 Provider**

```go
// 创建 BGE Provider
bgeProvider, err := embedding.NewBGEProvider("/path/to/bge-model")

// 创建 Sentence-BERT Provider
sbProvider, err := embedding.NewSentenceBERTProvider("/path/to/sbert-model")

// 创建 CLIP Provider (支持图文)
clipProvider, err := embedding.NewCLIPProvider("/path/to/clip-model")
```

#### 4.4 调用方法

**基础文本嵌入生成**

```go
ctx := context.Background()

// 单次调用
texts := []string{"你好世界", "Hello world"}
embeddings, err := provider.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}

// 批量调用 (自动分批处理)
largeTextList := make([]string, 1000)
// ... 填充文本
embeddings, err := provider.Embed(ctx, largeTextList)

// 获取向量维度
dim := provider.Dimension()
fmt.Printf("向量维度: %d\n", dim)
fmt.Printf("生成向量数量: %d\n", len(embeddings))
```

**CLIP 多模态调用 (图文嵌入)**

```go
clipProvider, err := embedding.WithCLIP("clip-vit-base-patch32", "")

// 文本嵌入
textEmbeddings, err := clipProvider.Embed(ctx, []string{"a cat", "a dog"})

// 图像嵌入
imageData, err := os.ReadFile("image.jpg")
imageEmbeddings, err := clipProvider.EmbedImages(ctx, [][]byte{imageData})

// 计算图文相似度
similarity := cosineSimilarity(textEmbeddings[0], imageEmbeddings[0])
```

**使用 BatchProcessor 优化批量处理**

```go
// 创建批处理器
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,                      // 每批最大文本数
    MaxConcurrent: 4,                        // 最大并发数
    CacheSize:     1000,                    // LRU 缓存条目数
})

// 简单批量处理
embeddings, err := processor.Process(ctx, texts)

// 带进度的批量处理 (适合大量文本)
callback := func(completed, total int) {
    fmt.Printf("进度: %d/%d (%.1f%%)\n", completed, total, float64(completed)/float64(total)*100)
}
embeddings, err := processor.ProcessWithProgress(ctx, largeTextList, callback)
```

#### 4.5 性能优化建议

**1. 批处理优化**

```go
// 根据模型和硬件调整批次大小
// GPU: MaxBatchSize = 64-128
// CPU: MaxBatchSize = 16-32
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,
    MaxConcurrent:  runtime.NumCPU(),  // 利用多核
    CacheSize:     5000,               // 增大缓存减少重复计算
})
```

**2. 缓存优化**

```go
// 重复文本会被自动缓存
// 首次调用计算并缓存，后续调用直接返回
texts := []string{"热门查询", "热门查询", "热门查询"} // 只计算一次
embeddings, _ := processor.Process(ctx, texts)
```

**3. 并发处理**

```go
// 对于大量文本，可并行处理不同批次
func parallelEmbed(ctx context.Context, provider embedding.Provider, texts []string, workers int) ([][]float32, error) {
    chunkSize := (len(texts) + workers - 1) / workers
    var wg sync.WaitGroup
    results := make([][][]float32, workers)
    errors := make([]error, workers)

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            start := idx * chunkSize
            end := start + chunkSize
            if end > len(texts) {
                end = len(texts)
            }
            results[idx], errors[idx] = provider.Embed(ctx, texts[start:end])
        }(i)
    }
    wg.Wait()

    // 合并结果
    var allEmbeddings [][]float32
    for _, emb := range results {
        allEmbeddings = append(allEmbeddings, emb...)
    }
    return allEmbeddings, nil
}
```

**4. 资源清理**

```go
// 使用完毕后关闭 Provider 释放资源
defer func() {
    if provider != nil {
        provider.Close()
    }
}()
```

**性能基准参考：**

| 模型                  | 硬件      | 批次大小 | 平均延迟 | 吞吐量        |
| :-------------------- | :-------- | :------- | :------- | :------------ |
| bge-small-zh-v1.5     | CPU (4核) | 32       | ~50ms/批 | ~600 文本/秒  |
| all-MiniLM-L6-v2      | CPU (4核) | 32       | ~30ms/批 | ~1000 文本/秒 |
| clip-vit-base-patch32 | CPU (4核) | 16       | ~80ms/批 | ~200 图/秒    |

## 🎯 设计理念

GoChat 秉承 Go 的极简大道：核心接口 `core.Client` 只有 `Chat` 和 `ChatStream` 两个方法。所有个性化功能全部通过 **Functional Options** 优雅扩展，确保主接口长期稳定且不被污染。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎提交 PR 一起共建！

---

## 📚 详细文档 (Documentation)

请查阅 `docs/` 目录获取详细的指南、架构图与 API 参考：
- 📖 [项目概述](https://gochat.rayainfo.cn/)
- 🚀 [快速入门](https://gochat.rayainfo.cn/quickstart/)
- 🧠 [Client 模块 (大模型与工具调用)](https://gochat.rayainfo.cn/modules/embedding/)
- 🧬 [Embedding 模块 (本地向量化)](https://gochat.rayainfo.cn/modules/embedding/)
- ⛓️ [Pipeline 模块 (工作流编排)](https://gochat.rayainfo.cn/modules/pipeline/)
- 🏢 [Provider 模块 (OAuth2 鉴权)](https://gochat.rayainfo.cn/modules/provider/)
- 📋 [API 参考](https://gochat.rayainfo.cn/api_reference/)
