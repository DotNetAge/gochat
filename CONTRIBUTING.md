# Contributing to GoChat

感谢你对 GoChat 的关注！我们欢迎各种形式的贡献。

## 如何贡献

### 报告 Bug

如果你发现了 bug，请创建一个 Issue 并包含：

1. **清晰的标题** - 简要描述问题
2. **复现步骤** - 详细的步骤说明如何触发 bug
3. **期望行为** - 你期望发生什么
4. **实际行为** - 实际发生了什么
5. **环境信息** - Go 版本、操作系统、GoChat 版本
6. **代码示例** - 最小可复现示例（如果可能）

### 提出新功能

在开始编码之前，请先创建一个 Issue 讨论你的想法：

1. 描述你想要的功能
2. 解释为什么需要这个功能
3. 提供使用示例
4. 等待维护者反馈

### 提交 Pull Request

1. **Fork 仓库**
   ```bash
   git clone https://github.com/YOUR_USERNAME/gochat.git
   cd gochat
   ```

2. **创建分支**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **编写代码**
   - 遵循现有的代码风格
   - 为新功能添加测试
   - 更新相关文档
   - 确保所有测试通过

4. **运行测试**
   ```bash
   go test ./...
   go vet ./...
   go fmt ./...
   ```

5. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

   提交信息格式：
   - `feat:` 新功能
   - `fix:` Bug 修复
   - `docs:` 文档更新
   - `test:` 测试相关
   - `refactor:` 代码重构
   - `chore:` 构建/工具相关

6. **推送并创建 PR**
   ```bash
   git push origin feature/your-feature-name
   ```
   然后在 GitHub 上创建 Pull Request

## 代码规范

### Go 代码风格

- 使用 `go fmt` 格式化代码
- 使用 `go vet` 检查代码
- 遵循 [Effective Go](https://golang.org/doc/effective_go.html)
- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

### 文档规范

- 所有公开的类型、函数、方法都必须有 godoc 注释
- godoc 注释以类型/函数名开头
- 复杂功能提供使用示例
- 在 `examples/` 目录添加可运行的示例

示例：
```go
// Client is an OpenAI LLM client.
//
// It implements the core.Client interface and provides access to OpenAI's
// chat completion API.
//
// Example:
//
//	client, err := openai.New(openai.Config{
//	    Config: base.Config{APIKey: "sk-..."},
//	})
type Client struct {
    // ...
}
```

### 测试规范

- 所有新功能必须有测试
- 使用 `testify/assert` 和 `testify/require`
- 使用 `httptest.NewServer` 模拟 HTTP 请求
- 测试函数命名：`TestFunctionName` 或 `TestType_Method`
- 使用表驱动测试处理多个测试用例

示例：
```go
func TestClient_Chat(t *testing.T) {
    tests := []struct {
        name    string
        input   []core.Message
        want    string
        wantErr bool
    }{
        {
            name:  "simple message",
            input: []core.Message{core.NewUserMessage("hello")},
            want:  "response",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## 项目结构

```
gochat/
├── pkg/
│   ├── core/           # 核心接口和类型
│   ├── client/         # 客户端实现
│   │   ├── base/       # 共享基础功能
│   │   ├── openai/     # OpenAI 客户端
│   │   ├── anthropic/  # Anthropic 客户端
│   │   └── ...
│   └── provider/       # OAuth 提供商
├── examples/           # 可运行的示例
└── README.md
```

## 添加新的 LLM 提供商

如果你想添加对新 LLM 提供商的支持：

1. 在 `pkg/client/` 下创建新目录
2. 实现 `core.Client` 接口
3. 使用 `base.Client` 处理通用功能（HTTP、重试等）
4. 添加完整的测试
5. 在 README 中添加使用示例
6. 在 `examples/` 中添加示例代码

参考现有的 `openai` 或 `anthropic` 实现。

## 需要帮助？

- 查看现有的 Issues 和 Pull Requests
- 阅读代码中的注释和文档
- 在 Issue 中提问

再次感谢你的贡献！
