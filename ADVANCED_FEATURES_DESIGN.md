# GoChat 高级功能设计文档

## 1. 需求分析

### 1.1 功能需求

| 功能                | 描述                           | 使用场景               | 实现挑战                 |
| ------------------- | ------------------------------ | ---------------------- | ------------------------ |
| 多轮对话            | 支持连续的对话交互，保持上下文 | 聊天机器人、客服系统   | 上下文管理、token限制    |
| 深度思考            | 让LLM进行更深入的推理和分析    | 复杂问题求解、决策支持 | 特殊prompt设计、模型支持 |
| 在线搜索            | 集成外部搜索服务获取实时信息   | 实时信息查询、新闻获取 | 搜索服务集成、结果处理   |
| 附加文件            | 支持上传和分析文件内容         | 文档分析、代码审查     | 文件格式处理、大小限制   |
| System Prompt       | 设置系统级别的指令和上下文     | 定制LLM行为、角色设定  | 指令优化、上下文管理     |
| Function Call       | 支持LLM调用外部函数并处理结果  | 工具使用、API调用      | 二次调用流程、错误处理   |
| Token Usage可观测性 | 收集和监控Token使用情况        | 成本控制、使用分析     | 统一收集机制、性能影响   |

### 1.2 技术需求

1. **统一接口**：所有高级功能需要通过统一的接口暴露
2. **供应商兼容**：支持不同LLM供应商的实现差异
3. **可扩展性**：易于添加新功能和新供应商
4. **性能优化**：确保高级功能的执行效率
5. **错误处理**：完善的错误处理机制
6. **可观测性**：支持Prometheus监控集成

## 2. 设计方案

### 2.1 核心接口扩展

```go
// AdvancedClient 扩展核心Client接口，支持高级功能
type AdvancedClient interface {
    Client
    
    // 多轮对话
    Chat(ctx context.Context, messages []Message) (string, error)
    ChatStream(ctx context.Context, messages []Message) (<-chan string, error)
    
    // 带系统提示的对话
    ChatWithSystem(ctx context.Context, systemPrompt string, messages []Message) (string, error)
    ChatWithSystemStream(ctx context.Context, systemPrompt string, messages []Message) (<-chan string, error)
    
    // Function Call
    ChatWithFunctions(ctx context.Context, messages []Message, functions []Function) (ChatResponse, error)
    
    // 文件附加
    ChatWithFile(ctx context.Context, file File, messages []Message) (string, error)
    
    // 深度思考
    DeepThinking(ctx context.Context, prompt string, iterations int) (string, error)
    
    // 在线搜索
    SearchAndChat(ctx context.Context, query string, messages []Message) (string, error)
}

// Message 表示对话中的消息
type Message struct {
    Role    string // user, assistant, system, function
    Content string
    Name    string // 可选，用于function调用
}

// Function 表示可调用的函数
type Function struct {
    Name        string
    Description string
    Parameters  map[string]interface{} // JSON Schema
}

// FunctionCall 表示函数调用请求
type FunctionCall struct {
    Name      string
    Arguments string // JSON格式的参数
}

// ChatResponse 表示聊天响应
type ChatResponse struct {
    Content      string
    FunctionCall *FunctionCall
    Usage        Usage
}

// File 表示附加文件
type File struct {
    Name     string
    Content  []byte
    MimeType string
}

// Usage 表示token使用情况
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// TokenUsageCollector 定义Token使用情况收集器接口
type TokenUsageCollector interface {
    // Collect 收集Token使用情况
    // 
    // Parameters:
    // - usage: Token使用情况
    // - model: 使用的模型名称
    // - provider: LLM供应商名称
    Collect(usage Usage, model string, provider string)
}
```

### 2.2 Function Call 实现机制

```go
// Function Call 工作流程
func (c *OpenAIClient) ChatWithFunctions(ctx context.Context, messages []Message, functions []Function) (ChatResponse, error) {
    // 第一次调用：获取函数调用请求
    response, err := c.callAPIWithFunctions(ctx, messages, functions)
    if err != nil {
        return ChatResponse{}, err
    }
    
    // 检查是否需要函数调用
    if response.FunctionCall != nil {
        // 执行外部函数
        functionResult, err := executeFunction(response.FunctionCall)
        if err != nil {
            return ChatResponse{}, err
        }
        
        // 将函数执行结果添加到消息中
        messages = append(messages, Message{
            Role:    "function",
            Content: functionResult,
            Name:    response.FunctionCall.Name,
        })
        
        // 第二次调用：获取最终结果
        finalResponse, err := c.callAPI(ctx, messages)
        if err != nil {
            return ChatResponse{}, err
        }
        
        return finalResponse, nil
    }
    
    return response, nil
}

// 函数执行器接口
type FunctionExecutor interface {
    Execute(functionName string, arguments string) (string, error)
}
```

### 2.3 供应商适配层设计

| 供应商     | 多轮对话 | System Prompt | Function Call | 文件附加     | 深度思考 | 在线搜索 |
| ---------- | -------- | ------------- | ------------- | ------------ | -------- | -------- |
| OpenAI     | ✅        | ✅             | ✅             | ✅            | ✅        | ✅        |
| Anthropic  | ✅        | ✅             | ⚠️ (有限支持)  | ⚠️ (有限支持) | ✅        | ✅        |
| Ollama     | ✅        | ✅             | ❌             | ❌            | ✅        | ❌        |
| Compatible | ✅        | ✅             | ✅             | ✅            | ✅        | ✅        |

### 2.4 实现优先级

1. **第一阶段**：基础高级功能
   - 多轮对话 (Chat, ChatStream)
   - System Prompt (ChatWithSystem)

2. **第二阶段**：核心高级功能
   - Function Call (ChatWithFunctions)
   - 文件附加 (ChatWithFile)

3. **第三阶段**：高级功能
   - 深度思考 (DeepThinking)
   - 在线搜索 (SearchAndChat)

## 3. 技术实现细节

### 3.1 多轮对话实现

- **上下文管理**：通过Message数组维护对话历史
- **Token限制**：实现token计数和自动截断机制
- **供应商适配**：
  - OpenAI：使用`messages`参数
  - Anthropic：使用`messages`参数
  - Ollama：使用`messages`参数
  - Compatible：使用`messages`参数

### 3.2 Function Call实现

- **二次调用流程**：
  1. 第一次调用获取函数调用请求
  2. 执行外部函数
  3. 将结果添加到消息历史
  4. 第二次调用获取最终结果

- **函数执行**：
  - 支持内置函数和自定义函数
  - 提供函数执行器接口
  - 错误处理和超时机制

### 3.3 文件附加实现

- **文件处理**：
  - 支持常见文件格式 (txt, pdf, docx, json, csv)
  - 文件大小限制和处理
  - 内容提取和格式化

- **供应商适配**：
  - OpenAI：使用`files` API
  - Anthropic：使用`messages`中的`content`字段
  - Ollama：不支持，需要本地处理
  - Compatible：根据具体实现

### 3.4 深度思考实现

- **实现方式**：
  - 多步骤推理prompt设计
  - 迭代式思考过程
  - 自我批判和修正

- **供应商适配**：
  - 所有供应商都支持，通过prompt设计实现

### 3.5 在线搜索实现

- **搜索服务集成**：
  - 支持多种搜索API (Google, Bing, DuckDuckGo)
  - 搜索结果处理和格式化
  - 结果整合到对话中

- **供应商适配**：
  - 所有供应商都支持，通过外部服务集成实现

### 3.6 Token Usage可观测性实现

- **收集器接口**：
  - 定义`TokenUsageCollector`接口，支持自定义收集逻辑
  - 提供默认的简单收集器实现

- **配置扩展**：
  - 在`Config`结构中添加`TokenUsageCollector`字段
  - 支持通过配置注入自定义收集器

- **集成方式**：
  - 在每个客户端的API调用完成后，检查是否配置了收集器
  - 如果配置了收集器，调用`Collect`方法收集Token使用情况

- **实现示例**：
  ```go
  // 简单的Token使用情况收集器
  type SimpleTokenCollector struct {
      totalTokens int
      modelUsage  map[string]int
  }
  
  func NewSimpleTokenCollector() *SimpleTokenCollector {
      return &SimpleTokenCollector{
          modelUsage: make(map[string]int),
      }
  }
  
  func (c *SimpleTokenCollector) Collect(usage Usage, model string, provider string) {
      c.totalTokens += usage.TotalTokens
      c.modelUsage[model] += usage.TotalTokens
      // 可以在这里添加日志、指标上报等逻辑
      fmt.Printf("Token usage: %d (prompt: %d, completion: %d) for model %s on %s\n", 
          usage.TotalTokens, usage.PromptTokens, usage.CompletionTokens, model, provider)
  }
  ```

- **Prometheus收集器实现**：
  ```go
  import (
      "github.com/prometheus/client_golang/prometheus"
      "github.com/prometheus/client_golang/prometheus/promauto"
  )
  
  // PrometheusTokenCollector 实现基于Prometheus的Token使用情况收集器
  type PrometheusTokenCollector struct {
      promptTokens     *prometheus.CounterVec
      completionTokens *prometheus.CounterVec
      totalTokens      *prometheus.CounterVec
  }
  
  func NewPrometheusTokenCollector(registry *prometheus.Registry) *PrometheusTokenCollector {
      if registry == nil {
          registry = prometheus.DefaultRegisterer
      }
      
      promptTokens := promauto.With(registry).NewCounterVec(
          prometheus.CounterOpts{
              Name: "llm_prompt_tokens_total",
              Help: "Total number of prompt tokens used",
          },
          []string{"model", "provider"},
      )
      
      completionTokens := promauto.With(registry).NewCounterVec(
          prometheus.CounterOpts{
              Name: "llm_completion_tokens_total",
              Help: "Total number of completion tokens used",
          },
          []string{"model", "provider"},
      )
      
      totalTokens := promauto.With(registry).NewCounterVec(
          prometheus.CounterOpts{
              Name: "llm_total_tokens_total",
              Help: "Total number of tokens used",
          },
          []string{"model", "provider"},
      )
      
      return &PrometheusTokenCollector{
          promptTokens:     promptTokens,
          completionTokens: completionTokens,
          totalTokens:      totalTokens,
      }
  }
  
  func (c *PrometheusTokenCollector) Collect(usage Usage, model string, provider string) {
      c.promptTokens.WithLabelValues(model, provider).Add(float64(usage.PromptTokens))
      c.completionTokens.WithLabelValues(model, provider).Add(float64(usage.CompletionTokens))
      c.totalTokens.WithLabelValues(model, provider).Add(float64(usage.TotalTokens))
  }
  ```

- **使用方式**：
  ```go
  // 创建简单收集器
  collector := NewSimpleTokenCollector()
  
  // 配置客户端
  config := core.Config{
      APIKey:              "your-api-key",
      Model:               "gpt-4",
      TokenUsageCollector: collector,
  }
  
  // 创建客户端
  client, err := openai.New(openai.Config{Config: config})
  
  // 使用客户端
  response, err := client.Complete(context.Background(), "Hello, who are you?")
  // Token使用情况会自动被收集
  ```

- **Prometheus集成示例**：
  ```go
  import (
      "github.com/prometheus/client_golang/prometheus"
      "github.com/prometheus/client_golang/prometheus/promhttp"
      "net/http"
  )
  
  // 创建Prometheus注册表
  registry := prometheus.NewRegistry()
  
  // 创建Prometheus收集器
  promCollector := NewPrometheusTokenCollector(registry)
  
  // 配置客户端
  config := core.Config{
      APIKey:              "your-api-key",
      Model:               "gpt-4",
      TokenUsageCollector: promCollector,
  }
  
  // 创建客户端
  client, err := openai.New(openai.Config{Config: config})
  
  // 启动Prometheus metrics endpoint
  http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
  go func() {
      http.ListenAndServe(":8080", nil)
  }()
  
  // 使用客户端
  response, err := client.Complete(context.Background(), "Hello, who are you?")
  // Token使用情况会自动被收集到Prometheus
  
  // 现在可以通过 http://localhost:8080/metrics 访问Token使用指标
  ```

## 4. 挑战与解决方案

### 4.1 供应商差异

**挑战**：不同供应商的API结构和功能支持差异很大

**解决方案**：
- 设计抽象适配层
- 针对不同供应商实现具体适配器
- 提供功能支持检测机制

### 4.2 Function Call复杂性

**挑战**：Function Call需要二次调用，执行外部函数后再反馈给LLM

**解决方案**：
- 设计清晰的二次调用流程
- 提供函数执行器接口
- 实现错误处理和重试机制

### 4.3 性能问题

**挑战**：多轮对话和Function Call可能增加延迟

**解决方案**：
- 实现缓存机制
- 优化API调用
- 并行处理搜索和文件分析

### 4.4 兼容性

**挑战**：确保新功能不会破坏现有API

**解决方案**：
- 保持向后兼容
- 渐进式功能添加
- 详细的文档和迁移指南

### 4.5 Token Usage可观测性挑战

**挑战**：确保Token使用情况收集不会影响性能，同时提供足够的信息

**解决方案**：
- 设计轻量级的收集接口
- 支持异步收集机制
- 提供可配置的收集级别
- 确保收集器实现的灵活性

## 5. 测试计划

### 5.1 单元测试
- 核心接口测试
- 各供应商适配器测试
- Function Call流程测试
- 文件处理测试
- Token Usage收集器测试
- Prometheus收集器测试

### 5.2 集成测试
- 完整对话流程测试
- 多供应商兼容性测试
- 性能测试
- 错误处理测试

### 5.3 端到端测试
- 完整功能测试
- 实际使用场景测试
- 边界情况测试

## 6. 文档和示例

### 6.1 文档
- API文档
- 功能指南
- 最佳实践
- 迁移指南
- Prometheus集成指南

### 6.2 示例
- 多轮对话示例
- Function Call示例
- 文件附加示例
- 深度思考示例
- 在线搜索示例
- Prometheus集成示例

## 7. 结论

通过本设计方案，GoChat库将能够支持丰富的高级功能，包括多轮对话、深度思考、在线搜索、附加文件、System Prompt和Function Call。同时，通过统一的接口设计和供应商适配层，确保了跨供应商的一致性和可扩展性。

实现将按照优先级逐步进行，确保每个功能的质量和稳定性。挑战将通过合理的设计和实现策略来解决，最终为用户提供强大而灵活的LLM交互能力。