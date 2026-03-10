# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoChat is a Go library for interacting with multiple LLM (Large Language Model) providers through a unified interface. It supports OpenAI, Anthropic, Ollama, Azure OpenAI, and OpenAI-compatible services.

## Development Commands

### Testing
```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/client/openai
go test ./pkg/client/anthropic
go test ./pkg/client/ollama
go test ./pkg/core

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -v -run TestComplete ./pkg/client/openai
```

### Building
```bash
# Build the module
go build ./...

# Verify dependencies
go mod verify

# Tidy dependencies
go mod tidy
```

### Linting
```bash
# Format code
go fmt ./...

# Run go vet
go vet ./...
```

## Architecture

### Core Abstractions

The library is built around a layered architecture:

1. **Core Interface Layer** (`pkg/core/llm.go`)
   - `Client` interface: Defines `Complete()` and `CompleteStream()` methods that all providers must implement
   - `Config` struct: Common configuration shared across all providers (APIKey, AuthToken, Model, BaseURL, Temperature, MaxTokens)

2. **Base Client Layer** (`pkg/client/base/client.go`)
   - Provides shared functionality for all provider implementations
   - Handles HTTP client setup, timeout configuration, and retry logic
   - `Retry()` method implements exponential backoff with jitter

3. **Provider Implementation Layer** (`pkg/client/*/client.go`)
   - Each provider (openai, anthropic, ollama, azureopenai, compatible) implements the `core.Client` interface
   - Wraps the base client and adds provider-specific API handling
   - Handles both streaming and non-streaming completions

### Authentication

The library supports two authentication methods:

1. **API Key**: Direct API key (e.g., `APIKey: "sk-..."`)
2. **Auth Token**: OAuth-style tokens (e.g., `AuthToken: "token"`)

API keys can be provided directly or retrieved from environment variables. The `core.GetAPIKey()` function automatically checks for environment variables using uppercase naming (e.g., `OPENAI_API_KEY`).

For OAuth flows, the library includes:
- `pkg/core/auth.go`: AuthManager for token lifecycle management
- `pkg/core/token.go`: OAuthToken structure and token operations
- `pkg/core/token-helper.go`: Token persistence and expiration checking
- `pkg/core/pkce.go`: PKCE (Proof Key for Code Exchange) implementation

### Error Handling

Structured error handling via `pkg/core/errors.go`:
- `ErrorTypeAPI`: API-level errors from providers
- `ErrorTypeNetwork`: Network connectivity issues
- `ErrorTypeTimeout`: Request timeouts
- `ErrorTypeValidation`: Input validation failures
- `ErrorTypeUnknown`: Unclassified errors

All errors include the error type, message, and optional cause chain.

### Retry Mechanism

Automatic retry with exponential backoff (`pkg/core/retry.go`):
- Retries on rate limits, timeouts, temporary errors, connection issues
- Exponential backoff with jitter to prevent thundering herd
- Maximum delay capped at 60 seconds
- Configurable max retries (default: 3)

### Provider-Specific Notes

**OpenAI** (`pkg/client/openai/`):
- Default model: `gpt-3.5-turbo`
- Endpoint: `/v1/chat/completions`
- Auth header: `Authorization: Bearer <token>`

**Anthropic** (`pkg/client/anthropic/`):
- Default model: `claude-3-opus-20240229`
- Endpoint: `/v1/messages`
- Auth header: `x-api-key: <token>`
- Requires `anthropic-version: 2023-06-01` header

**Ollama** (`pkg/client/ollama/`):
- Default timeout: 60 seconds (longer than other providers)
- Default BaseURL: `http://localhost:11434`
- No authentication required for local instances

**Azure OpenAI** (`pkg/client/azureopenai/`):
- Uses Azure-specific endpoint structure
- Requires Azure subscription and deployment name

**Compatible** (`pkg/client/compatible/`):
- For OpenAI API-compatible services
- Uses OpenAI's request/response format

## Code Conventions

- All client implementations embed a `*base.Client` for shared functionality
- Streaming responses use Go channels (`<-chan string`)
- Context is passed through all API calls for cancellation and timeout control
- Configuration structs embed `base.Config` to inherit common fields
- Provider-specific config types are named `Config` within their package
