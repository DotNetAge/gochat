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

- **🔌 The Ultimate "Write Once, Run Anywhere" (Client Module)**
  - Completely smooths out the chaotic API structures and streaming parsing differences between OpenAI, Anthropic (Claude), DeepSeek, Qwen, and other models.
  - **Unified Tool Calling**: Define `core.Tool` once, and the framework will automatically "translate" it into the corresponding vendor's tool calling format (such as Anthropic's unique format), enabling seamless switching between underlying models.
  - **Built-in Anti-Fragility Mechanism**: Automatically captures HTTP 429 rate limits and network fluctuations, triggering exponential backoff with jitter retry, keeping your service rock-solid.
- **🧠 "Zero-Configuration" Local Vectorization (Embedding) Without Ollama**
  - **Get Rid of Bulky Dependencies**: Directly compute Embedding locally based on lightweight ONNX runtime. No need to deploy massive Ollama services or deal with complex Python environments.
  - **Built-in Geek Downloader**: Just one line `embedding.WithBEG("bge-small-zh-v1.5", "")`, automatically pulls model shards from remote mirrors when missing locally and loads them ready.
  - **Industrial-Grade Batching**: Built-in `BatchProcessor` supports dynamic concurrent batching and automatic Hash cache filtering for identical texts, maximizing CPU computing power.
- **🌊 Making Extremely Complex Logic Elegant (Generic Pipeline)**
  - Chain independent `Step`s like building blocks to elegantly orchestrate complex RAG or Agent reasoning flows.
  - **Strongly-Typed Context Transfer**: Thanks to Go 1.24+ generics, you can seamlessly pass custom strongly-typed `struct`s between Steps. Completely say goodbye to type assertion crashes and typos caused by traditional `map[string]any`.
  - **Programmable Control Flow**: Built-in `IfStep`, `LoopStep`, and AOP monitoring hooks (`Hooks`) give workflows exceptional observability and scheduling domain control.

---

## 📦 Installation

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 Quick Start

### 1. High-Performance Vectorization (Embedding)

```go
// Use BatchProcessor to optimize vector generation
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,
    MaxConcurrent: 4,
})

// Generate vectors and track progress
embeddings, err := processor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
    fmt.Printf("Progress: %d/%d\n", current, total)
    return true // Return false to cancel task
})
```

### 2. Pipeline Workflow

```go
p := pipeline.New[*pipeline.State]().
    AddStep(steps.NewTemplateStep("User question: {{.query}}", "prompt", "query")).
    AddStep(steps.NewGenerateCompletionStep(client, "prompt", "answer", "gpt-4o")).
    AddHook(myLogger) // Observe each step execution

state := pipeline.NewState()
state.Set("query", "What is GoChat?")

err := p.Execute(ctx, state)
fmt.Println(state.GetString("answer"))
```

### 3. Capturing "Chain of Thought" in Streaming Output

```go
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))
defer stream.Close()

for stream.Next() {
    ev := stream.Event()
    if ev.Type == core.EventThinking {
        fmt.Print(ev.Content) // Print reasoning process
    } else if ev.Type == core.EventContent {
        fmt.Print(ev.Content) // Print final answer
    }
}
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
| **Local / ONNX**    | BGE, Sentence-BERT  | Local Execution (No Key Required)|
| **Azure OpenAI**    | Microsoft-deployed models | API Key (Azure format)      |

---

### 4. Local Embedding Model Usage

GoChat's embedding package provides complete local embedding model support, achieving efficient text embedding generation through ONNX format models. The local solution requires no API Key, has no network dependency, and can run completely offline, making it ideal for scenarios with strict requirements for data privacy and response latency.

#### 4.1 Complete List of Implemented Providers

| Provider Type         | Model Type            | Dimension | Use Case             | Features                           |
| :-------------------- | :-------------------- | :-------- | :------------------- | :--------------------------------- |
| **BGEProvider**       | bge-small-zh-v1.5    | 512       | Chinese Semantic Sim | Efficient Chinese embedding, lightweight |
| **BGEProvider**       | bge-base-zh-v1.5     | 768       | Chinese Semantic Sim | Higher accuracy, balanced performance |
| **SentenceBERTProvider** | all-MiniLM-L6-v2 | 384       | English Semantic Search | Ultra-fast inference, real-time apps |
| **SentenceBERTProvider** | all-mpnet-base-v2 | 768       | English Semantic Search | High precision, BERT architecture |
| **CLIPProvider**     | clip-vit-base-patch32 | 512       | Multimodal Image-Text | Text & image bidirectional embedding |
| **LocalProvider**     | bert-base-uncased    | 768       | General English Tasks | Standard BERT model               |

```go
import "gochat/pkg/embedding"

// Provider interface definition
type Provider interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
}

// Multimodal Provider (with image support)
type MultimodalProvider interface {
    Provider
    EmbedImages(ctx context.Context, images [][]byte) ([][]float32, error)
}
```

#### 4.2 Downloading ONNX Models with Downloader

Downloader provides functionality for downloading pre-compiled ONNX models from HuggingFace, supporting progress tracking and cache management.

```go
// Create downloader (specify cache directory)
downloader := embedding.NewDownloader("~/.gochat/models")

// View available model list
models := downloader.GetModelInfo()
for _, m := range models {
    fmt.Printf("Model: %s | Type: %s | Size: %s\n", m.Name, m.Type, m.Size)
}

// Download progress callback function
callback := func(modelName, fileName string, downloaded, total int64) {
    if total > 0 {
        percent := float64(downloaded) / float64(total) * 100
        fmt.Printf("\r[%s] %s: %.1f%%", modelName, fileName, percent)
    }
}

// Download model
modelPath, err := downloader.DownloadModel("bge-small-zh-v1.5", callback)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("\nModel downloaded to: %s\n", modelPath)
```

**Downloader Core Methods:**

| Method                      | Parameters                            | Return Value         | Description                     |
| :-------------------------- | :------------------------------------ | :------------------- | :------------------------------ |
| `NewDownloader(cacheDir)`   | Cache directory path, empty for default | `*Downloader`        | Create downloader instance      |
| `GetModelInfo()`            | None                                  | `[]DownloadModelInfo` | Return all available model info |
| `DownloadModel(name, callback)` | Model name, progress callback       | `(string, error)`    | Download specified model        |

**Supported Model File Download List:**

| Model Name             | File URL                                       | Est. Size     |
| :-------------------- | :--------------------------------------------- | :------------ |
| bge-small-zh-v1.5     | model_fp16.onnx, model_fp16.onnx_data         | ~48MB         |
| all-MiniLM-L6-v2      | model_fp16.onnx                                | ~45.3MB       |
| bert-base-uncased     | model_fp16.onnx                                | ~200MB        |
| bge-base-zh-v1.5      | model_fp16.onnx                                | ~100MB        |
| clip-vit-base-patch32 | text_model_fp16.onnx, vision_model_fp16.onnx  | ~300MB        |
| all-mpnet-base-v2     | model_fp16.onnx                                | ~218MB        |

#### 4.3 Local Embedding Model Initialization Configuration

**Method 1: Auto-Download and Initialize (Recommended)**

```go
// Use BGE model, auto-download (if not cached)
provider, err := embedding.WithBEG("bge-small-zh-v1.5", "")
if err != nil {
    log.Fatal(err)
}

// Use BERT model
bertProvider, err := embedding.WithBERT("all-mpnet-base-v2", "")
if err != nil {
    log.Fatal(err)
}

// Use CLIP multimodal model
clipProvider, err := embedding.WithCLIP("clip-vit-base-patch32", "")
if err != nil {
    log.Fatal(err)
}
```

**Method 2: Specify Local Path Initialization**

```go
// Already downloaded model, specify path directly
provider, err := embedding.WithBEG("bge-small-zh-v1.5", "/path/to/model")
if err != nil {
    log.Fatal(err)
}

// Use factory function to create
provider, err := embedding.NewProvider("/path/to/bge-small-zh-v1.5")
if err != nil {
    log.Fatal(err)
}

// Custom config to create LocalProvider
localProvider, err := embedding.New(embedding.Config{
    Model:        model,
    Dimension:    512,
    MaxBatchSize: 32,
})
```

**Method 3: Create Specific Provider Directly**

```go
// Create BGE Provider
bgeProvider, err := embedding.NewBGEProvider("/path/to/bge-model")

// Create Sentence-BERT Provider
sbProvider, err := embedding.NewSentenceBERTProvider("/path/to/sbert-model")

// Create CLIP Provider (with image-text support)
clipProvider, err := embedding.NewCLIPProvider("/path/to/clip-model")
```

#### 4.4 Usage Methods

**Basic Text Embedding Generation**

```go
ctx := context.Background()

// Single call
texts := []string{"Hello world", "你好世界"}
embeddings, err := provider.Embed(ctx, texts)
if err != nil {
    log.Fatal(err)
}

// Batch call (automatic batching)
largeTextList := make([]string, 1000)
// ... fill texts
embeddings, err := provider.Embed(ctx, largeTextList)

// Get vector dimension
dim := provider.Dimension()
fmt.Printf("Vector dimension: %d\n", dim)
fmt.Printf("Number of vectors generated: %d\n", len(embeddings))
```

**CLIP Multimodal Usage (Image-Text Embedding)**

```go
clipProvider, err := embedding.WithCLIP("clip-vit-base-patch32", "")

// Text embedding
textEmbeddings, err := clipProvider.Embed(ctx, []string{"a cat", "a dog"})

// Image embedding
imageData, err := os.ReadFile("image.jpg")
imageEmbeddings, err := clipProvider.EmbedImages(ctx, [][]byte{imageData})

// Calculate image-text similarity
similarity := cosineSimilarity(textEmbeddings[0], imageEmbeddings[0])
```

**Optimize Batch Processing with BatchProcessor**

```go
// Create batch processor
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,                      // Max texts per batch
    MaxConcurrent: 4,                       // Max concurrent batches
    CacheSize:     1000,                    // LRU cache entries
})

// Simple batch processing
embeddings, err := processor.Process(ctx, texts)

// Batch processing with progress (suitable for large texts)
callback := func(completed, total int) {
    fmt.Printf("Progress: %d/%d (%.1f%%)\n", completed, total, float64(completed)/float64(total)*100)
}
embeddings, err := processor.ProcessWithProgress(ctx, largeTextList, callback)
```

#### 4.5 Performance Optimization Suggestions

**1. Batch Processing Optimization**

```go
// Adjust batch size based on model and hardware
// GPU: MaxBatchSize = 64-128
// CPU: MaxBatchSize = 16-32
processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
    MaxBatchSize:  32,
    MaxConcurrent: runtime.NumCPU(),  // Utilize multi-core
    CacheSize:     5000,              // Increase cache to reduce redundant computation
})
```

**2. Cache Optimization**

```go
// Duplicate texts are automatically cached
// First call computes and caches, subsequent calls return directly
texts := []string{"hot query", "hot query", "hot query"} // Only computed once
embeddings, _ := processor.Process(ctx, texts)
```

**3. Concurrent Processing**

```go
// For large volumes of text, process different batches in parallel
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

    // Merge results
    var allEmbeddings [][]float32
    for _, emb := range results {
        allEmbeddings = append(allEmbeddings, emb...)
    }
    return allEmbeddings, nil
}
```

**4. Resource Cleanup**

```go
// Close Provider to release resources after use
defer func() {
    if provider != nil {
        provider.Close()
    }
}()
```

**Performance Benchmark Reference:**

| Model                  | Hardware   | Batch Size | Avg Latency | Throughput        |
| :-------------------- | :--------- | :--------- | :---------- | :---------------- |
| bge-small-zh-v1.5     | CPU (4 core) | 32         | ~50ms/batch | ~600 texts/sec  |
| all-MiniLM-L6-v2      | CPU (4 core) | 32         | ~30ms/batch | ~1000 texts/sec |
| clip-vit-base-patch32 | CPU (4 core) | 16         | ~80ms/batch | ~200 images/sec |

## 🎯 Design Philosophy

GoChat adheres to Go's philosophy of minimalism: The core interface `core.Client` has only two methods, `Chat` and `ChatStream`. All personalization features are elegantly extended through **Functional Options**, ensuring the main interface remains long-term stable and uncontaminated.

## 📄 License

This project is open-sourced under the [MIT License](LICENSE). PRs are welcome!

---

## 📚 Comprehensive Documentation

Please refer to the `docs/` directory for detailed guides, architecture diagrams, and API references:
- 📖 [Project Overview](https://gochat.rayainfo.cn/)
- 🚀 [Quick Start](https://gochat.rayainfo.cn/quickstart/)
- 🧠 [Client Module (LLM & Tool Calling)](https://gochat.rayainfo.cn/modules/embedding/)
- 🧬 [Embedding Module (Local Vectorization)](https://gochat.rayainfo.cn/modules/embedding/)
- ⛓️ [Pipeline Module (Workflow Orchestration)](https://gochat.rayainfo.cn/modules/pipeline/)
- 🏢 [Provider Module (OAuth2 Authentication)](https://gochat.rayainfo.cn/modules/provider/)
- 📋 [API Reference](https://gochat.rayainfo.cn/api_reference/)
