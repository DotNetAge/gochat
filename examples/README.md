# GoChat Examples

This directory contains practical examples demonstrating how to use the GoChat library.

## Running the Examples

All examples require API keys to be set as environment variables:

```bash
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"  # For Anthropic examples
```

Then run any example:

```bash
go run examples/01_basic_chat/main.go
```

## Examples

### 01_basic_chat
The simplest possible usage: send a message, get a response.

**What you'll learn:**
- Creating a client
- Sending a single message
- Getting a response
- Accessing token usage

### 02_multi_turn
Maintaining conversation history across multiple turns.

**What you'll learn:**
- Building conversation history
- Using system messages
- Context retention across turns
- Appending assistant responses to history

### 03_streaming
Getting responses token-by-token as they're generated.

**What you'll learn:**
- Using `ChatStream` instead of `Chat`
- Iterating through stream events
- Handling streaming errors
- Getting usage info after streaming completes

### 04_tool_calling
Allowing the model to call external functions.

**What you'll learn:**
- Defining tools with JSON Schema
- Detecting when the model wants to call a tool
- Executing tools and returning results
- Multi-turn tool calling flow

### 05_multiple_providers
Using different LLM providers with the same code.

**What you'll learn:**
- Provider-agnostic code using `core.Client` interface
- Switching between OpenAI, Anthropic, and Ollama
- Provider-specific configuration

### 06_image_input
Sending images to vision-capable models.

**What you'll learn:**
- Reading and encoding images to base64
- Creating multimodal messages (text + image)
- Using vision models (GPT-4 Vision, Claude 3)
- Handling different image formats

### 07_document_analysis
Analyzing text documents (code, markdown, text files).

**What you'll learn:**
- Reading file content
- Sending documents for analysis
- Structured analysis prompts
- Token usage for large inputs

### 08_multiple_images
Analyzing multiple images in one request.

**What you'll learn:**
- Sending multiple images in one message
- Batch image processing
- Comparing and contrasting images
- Building complex multimodal messages

### 09_helper_utilities
Reusable helper functions for common tasks.

**What you'll learn:**
- Loading images as ContentBlocks
- Creating multimodal messages easily
- Managing conversation history
- Adding tool results to conversations
- Best practices for code reuse

## More Examples

For more advanced examples, see the main README.md:
- Multimodal inputs (images)
- Custom retry strategies
- Error handling patterns
- Token usage monitoring
- Extended thinking (o1/o3 models)
