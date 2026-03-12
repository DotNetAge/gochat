# Embedding Models Guide

本指南详细介绍了 gochat 项目支持的各种 embedding 模型及其适用场景。

## 模型概览

| 模型名称          | 类型          | 语言 | 维度 | 大小   | 推荐场景           |
| ----------------- | ------------- | ---- | ---- | ------ | ------------------ |
| bge-small-zh-v1.5 | BGE           | 中文 | 768  | ~100MB | 中文 RAG、中文搜索 |
| bge-base-zh-v1.5  | BGE           | 中文 | 768  | ~400MB | 中文高精度应用     |
| all-MiniLM-L6-v2  | Sentence-BERT | 英文 | 384  | ~80MB  | 英文轻量级应用     |
| all-mpnet-base-v2 | Sentence-BERT | 英文 | 768  | ~400MB | 英文高精度应用     |
| bert-base-uncased | BERT          | 英文 | 768  | ~400MB | 通用英文应用       |

---

## 详细模型说明

### 1. BGE 系列模型 (Chinese)

#### bge-small-zh-v1.5
- **模型类型**: BGE (BAAI General Embedding)
- **语言**: 中文
- **维度**: 768
- **大小**: ~100MB
- **速度**: 快
- **精度**: 中高

**适用场景**:
- ✅ 中文 RAG (Retrieval-Augmented Generation)
- ✅ 中文文档搜索
- ✅ 中文相似度匹配
- ✅ 中文聚类任务
- ✅ 资源受限的部署环境

**优势**:
- 轻量级，加载快
- 中文性能优秀
- 内存占用小
- 推理速度快

---

#### bge-base-zh-v1.5
- **模型类型**: BGE (BAAI General Embedding)
- **语言**: 中文
- **维度**: 768
- **大小**: ~400MB
- **速度**: 中
- **精度**: 高

**适用场景**:
- ✅ 高精度中文 RAG
- ✅ 中文语义搜索
- ✅ 中文问答系统
- ✅ 中文文本分类
- ✅ 企业级应用

**优势**:
- 中文语义理解能力强
- 精度更高
- 适合复杂任务

---

### 2. Sentence-BERT 系列模型 (English)

#### all-MiniLM-L6-v2
- **模型类型**: Sentence-BERT
- **语言**: 英文
- **维度**: 384
- **大小**: ~80MB
- **速度**: 非常快
- **精度**: 中

**适用场景**:
- ✅ 英文轻量级 RAG
- ✅ 英文快速搜索
- ✅ 英文推荐系统
- ✅ 英文文本聚类
- ✅ 边缘设备部署
- ✅ 实时应用

**优势**:
- 超轻量级
- 推理速度极快
- 内存占用极低
- 适合大规模批量处理

---

#### all-mpnet-base-v2
- **模型类型**: Sentence-BERT
- **语言**: 英文
- **维度**: 768
- **大小**: ~400MB
- **速度**: 中
- **精度**: 很高

**适用场景**:
- ✅ 高精度英文 RAG
- ✅ 英文语义搜索
- ✅ 英文问答系统
- ✅ 英文文本分类
- ✅ 企业级应用
- ✅ 研究实验

**优势**:
- 综合性能最好
- 语义理解能力强
- 适合各种复杂任务

---

### 3. BERT 基础模型

#### bert-base-uncased
- **模型类型**: BERT
- **语言**: 英文
- **维度**: 768
- **大小**: ~400MB
- **速度**: 中
- **精度**: 中高

**适用场景**:
- ✅ 通用英文任务
- ✅ 英文基础研究
- ✅ 英文特征提取
- ✅ 作为基础模型进行微调

**优势**:
- 经典模型，广泛使用
- 社区支持好
- 资源丰富

---

## 选择指南

### 按语言选择

**中文应用**:
- 推荐 `bge-small-zh-v1.5` (轻量级，速度快)
- 或 `bge-base-zh-v1.5` (高精度)

**英文应用**:
- 轻量级需求: `all-MiniLM-L6-v2`
- 高精度需求: `all-mpnet-base-v2`
- 通用需求: `bert-base-uncased`

---

### 按性能需求选择

**追求速度**:
- 中文: `bge-small-zh-v1.5`
- 英文: `all-MiniLM-L6-v2`

**追求精度**:
- 中文: `bge-base-zh-v1.5`
- 英文: `all-mpnet-base-v2`

**平衡考虑**:
- 中文: `bge-small-zh-v1.5`
- 英文: `all-MiniLM-L6-v2`

---

### 按资源限制选择

**内存/存储受限 (< 200MB)**:
- 中文: `bge-small-zh-v1.5`
- 英文: `all-MiniLM-L6-v2`

**资源充足 (> 400MB)**:
- 中文: `bge-base-zh-v1.5`
- 英文: `all-mpnet-base-v2`

---

## 使用示例

### 下载并使用模型

```go
import (
    "github.com/DotNetAge/gochat/pkg/embedding/downloader"
    "github.com/DotNetAge/gochat/pkg/embedding/models"
)

// 创建下载器
dl := downloader.NewDownloader("")

// 下载模型
modelPath, err := dl.DownloadModel("bge-small-zh-v1.5")
if err != nil {
    log.Fatal(err)
}

// 创建 provider
provider, err := models.NewProvider(modelPath)
if err != nil {
    log.Fatal(err)
}

// 生成嵌入
embeddings, err := provider.Embed(ctx, texts)
```

---

## 性能对比

### 推理速度 (相对)
- `all-MiniLM-L6-v2`: ⭐⭐⭐⭐⭐ (最快)
- `bge-small-zh-v1.5`: ⭐⭐⭐⭐
- `bert-base-uncased`: ⭐⭐⭐
- `bge-base-zh-v1.5`: ⭐⭐
- `all-mpnet-base-v2`: ⭐⭐

### 内存占用 (相对)
- `all-MiniLM-L6-v2`: ⭐⭐⭐⭐⭐ (最小)
- `bge-small-zh-v1.5`: ⭐⭐⭐⭐
- `bert-base-uncased`: ⭐⭐
- `bge-base-zh-v1.5`: ⭐⭐
- `all-mpnet-base-v2`: ⭐⭐

### 精度 (相对)
- `all-mpnet-base-v2`: ⭐⭐⭐⭐⭐ (最高)
- `bge-base-zh-v1.5`: ⭐⭐⭐⭐
- `bert-base-uncased`: ⭐⭐⭐
- `all-MiniLM-L6-v2`: ⭐⭐⭐
- `bge-small-zh-v1.5`: ⭐⭐⭐

---

## 常见问题

### Q: 如何选择适合自己的模型？
A: 考虑三个因素：语言、精度需求、资源限制。参考上面的选择指南。

### Q: 可以在边缘设备上使用吗？
A: 可以，推荐使用 `all-MiniLM-L6-v2` 或 `bge-small-zh-v1.5`，它们资源占用小。

### Q: 模型需要定期更新吗？
A: 一般不需要，除非有特定的性能需求或新的模型版本。

### Q: 可以微调这些模型吗？
A: 可以，但需要额外的工具和数据。建议先尝试预训练模型，看是否满足需求。

---

## 更多资源

- [Hugging Face Models](https://huggingface.co/models)
- [Sentence-BERT](https://www.sbert.net/)
- [BGE Models](https://github.com/FlagOpen/FlagEmbedding)
- [BERT](https://github.com/google-research/bert)
