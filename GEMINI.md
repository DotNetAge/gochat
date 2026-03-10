# GoChat - Project Context

## Project Overview

**GoChat** is a modern, type-safe Go library that provides a unified interface for interacting with multiple Large Language Model (LLM) providers. It abstracts away the complexities of different provider APIs (such as OpenAI, Anthropic, Ollama, Azure OpenAI, Deepseek, Minimax, and Qwen) behind a single, consistent abstraction layer. 

Key capabilities include:
- **Multi-turn Conversations**: Native support for managing chat history.
- **Multi-modal Support**: Handling text, images, and file inputs.
- **Tool/Function Calling**: Structured support for LLM tool invocation.
- **Streaming**: Robust stream handling with event iteration and error tracking.
- **Smart Retries**: Built-in exponential backoff and jitter for network/API resilience.
- **Functional Options**: Flexible and extensible request configuration.

## Architecture & Directory Structure

The repository is built around a layered architecture emphasizing small interfaces and type safety:

- **`pkg/core/`**: Contains the core domain models and interfaces. 
  - `Client` interface (`Chat` and `ChatStream`).
  - Types for `Message`, `ContentBlock`, `Tool`, `Stream`, and `Option`.
  - Shared logic for errors, authentication, token management, and retry mechanics.
- **`pkg/client/base/`**: Base HTTP client implementation handling common concerns like timeouts, request execution, and retry strategies.
- **`pkg/client/*/`**: Provider-specific implementations (e.g., `openai`, `anthropic`, `ollama`, `azureopenai`). Each implements the `core.Client` interface and maps provider-specific API formats to the unified core types.
- **`examples/`**: A comprehensive suite of runnable examples demonstrating features from basic chat to tool calling and document analysis.

## Building and Testing

As a Go library, standard Go toolchain commands apply.

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific provider
go test ./pkg/client/openai
```

### Building & Dependency Management
```bash
# Verify the code compiles
go build ./...

# Manage dependencies
go mod tidy
go mod verify
```

### Running Examples
You can run the provided examples to test specific features (make sure to set necessary environment variables like `OPENAI_API_KEY` or `ANTHROPIC_API_KEY`):
```bash
go run examples/01_basic_chat/main.go
```

## Development Conventions

When contributing or modifying code in this repository, adhere to the following conventions:

1. **Small Interfaces**: Adhere to the core interface design (`Client` only has `Chat` and `ChatStream`). Advanced behavior should be injected via `Message` structures or `Option` functions rather than expanding the interface.
2. **Functional Options Pattern**: Use `core.Option` (e.g., `core.WithTemperature()`, `core.WithTools()`) for optional configurations in requests.
3. **Type Safety**: Avoid unstructured data (like `map[string]interface{}`) or raw strings for prompts where possible. Use structured types like `core.Message` and `core.ContentBlock`.
4. **Context-Aware**: All API and network operations must take a `context.Context` as the first parameter to allow for strict timeout and cancellation control.
5. **Streaming Paradigms**: Use the structured `Stream` object and its iterator pattern (`stream.Next()`, `stream.Event()`) instead of naked Go channels. This ensures proper propagation of errors and usage metrics.
6. **Error Handling**: Use the structured error types defined in `pkg/core/errors.go` (e.g., `ErrorTypeAPI`, `ErrorTypeNetwork`) rather than generic Go errors to facilitate the built-in retry mechanisms.
7. **Embedding**: Provider clients should embed `*base.Client` to inherit standard HTTP and retry behaviors. Config structs should embed `base.Config`.