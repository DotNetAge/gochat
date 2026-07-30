package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// 真实卡死场景复现测试
// ============================================================
// 背景：MindX 实测时第二次对话卡死 7+ 分钟，daemon/ollama CPU 均 0%。
// 根因：ollama 流式响应异常中断（不发数据也不关连接）时，
// gochat 的 doChatStream 用 bufio.Scanner.Scan() 阻塞读取，
// 既无 ctx 取消检查也无读取超时，导致永久挂起。
//
// 本测试组用真实 ollama 服务复现该场景，并验证修复后 ctx 取消能打断阻塞。
// 需 OLLAMA_LIVE=1 环境变量运行。

// TestLive_HangRepro_LargeSystemPrompt 用 MindX 第二次对话的近似场景复现卡死。
// - 4332 字符的 system prompt（agent 定义 + 工具目录 + 行为准则）
// - 带完整工具定义（QuickSearch/Grep/Read 等）
// - 模型 minicpm-v4.6（thinking 模型，context_length=4096）
func TestLive_HangRepro_LargeSystemPrompt(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)

	// 读取抓取的真实 system prompt
	sp, err := os.ReadFile("/tmp/mindx_system_prompt.txt")
	if err != nil {
		// 如果临时文件不存在，用近似长度的占位 prompt
		sp = []byte(fmt.Sprintf("你是后端工程师。%s", string(make([]byte, 4000))))
	}

	// 构造与 MindX 第二次对话近似的消息序列
	messages := []core.Message{
		core.NewSystemMessage(string(sp)),
		core.NewUserMessage("你好，你的职责是什么？"),
		core.NewTextMessage("assistant", "我是后端工程师，负责服务端应用的设计和开发..."),
		core.NewUserMessage("你可以帮我查一下 Bega 项目的介绍吗"),
	}

	tools := buildMindxLikeTools()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	stream, err := client.ChatStream(ctx, messages,
		core.WithTools(tools...),
		core.WithMaxTokens(500),
	)
	require.NoError(t, err)
	defer stream.Close()

	var content, thinking string
	var done bool
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case core.EventContent:
			content += ev.Content
		case core.EventThinking:
			thinking += ev.Content
		case core.EventError:
			t.Fatalf("stream error: %v", ev.Err)
		case core.EventDone:
			done = true
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream ended with error: %v", err)
	}

	t.Logf("result: done=%v contentLen=%d thinkingLen=%d", done, len(content), len(thinking))
	if content == "" && thinking == "" {
		t.Log("警告：content 和 thinking 均为空，可能复现了卡死场景")
	}
}

// TestLive_CtxCancel_BreaksBlockedStream 验证 ctx 取消能打断阻塞的流。
// 这是修复验证的核心测试：
// 1. 发起一个会卡住的流式请求（大 prompt + 小 context）
// 2. 2 秒后取消 ctx
// 3. 验证 ChatStream 在合理时间内返回（而非永久阻塞）
//
// 修复前：scanner.Scan() 阻塞，ctx 取消信号无法打断 → 27 秒/7 分钟才返回
// 修复后：ctx 监听 goroutine 关闭 resp.Body → scanner.Scan() 返回错误
//
// 注意：修复后仍需几秒才返回，因为 ollama server 不监听客户端 ctx，
// body 关闭后 TCP 层面仍有缓冲数据需要排空。但远小于修复前的 27 秒。
func TestLive_CtxCancel_BreaksBlockedStream(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)

	sp, err := os.ReadFile("/tmp/mindx_system_prompt.txt")
	if err != nil {
		sp = []byte(fmt.Sprintf("你是后端工程师。%s", string(make([]byte, 4000))))
	}

	messages := []core.Message{
		core.NewSystemMessage(string(sp)),
		core.NewUserMessage("你可以帮我查一下 Bega 项目的介绍吗"),
	}
	tools := buildMindxLikeTools()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	start := time.Now()
	stream, err := client.ChatStream(ctx, messages,
		core.WithTools(tools...),
		core.WithMaxTokens(500),
	)
	require.NoError(t, err)
	defer stream.Close()

	// 2 秒后取消 ctx
	go func() {
		time.Sleep(2 * time.Second)
		t.Log("取消 ctx")
		cancel()
	}()

	// 验证流在 ctx 取消后 15 秒内返回（修复前要 27 秒以上）
	returned := make(chan struct{})
	var streamErr error
	go func() {
		defer close(returned)
		for stream.Next() {
			// 消费事件
		}
		streamErr = stream.Err()
	}()

	select {
	case <-returned:
		elapsed := time.Since(start)
		t.Logf("流在 %v 后返回（修复前需 27 秒）", elapsed)
		assert.Less(t, elapsed, 15*time.Second, "ctx 取消后应在 15 秒内返回（修复前需 27 秒）")
		if streamErr != nil {
			t.Logf("stream error: %v", streamErr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("FAIL: 流在 30 秒内未返回，复现了永久卡死 bug（修复未生效）")
	}
}

// TestLive_CtxTimeout_BreaksBlockedStream 验证 ctx 超时能打断阻塞。
// 这是 goharness 实际使用的模式（defaultLLMTimeout = 4 分钟）。
// 用 20 秒超时（足够 ollama 处理大 prompt 建立连接 + 开始流式），
// 验证超时后能快速返回而非永久阻塞。
func TestLive_CtxTimeout_BreaksBlockedStream(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)

	sp, err := os.ReadFile("/tmp/mindx_system_prompt.txt")
	if err != nil {
		sp = []byte(fmt.Sprintf("你是后端工程师。%s", string(make([]byte, 4000))))
	}

	messages := []core.Message{
		core.NewSystemMessage(string(sp)),
		core.NewUserMessage("你可以帮我查一下 Bega 项目的介绍吗"),
	}
	tools := buildMindxLikeTools()

	// 20 秒超时（足够建立连接，但若 ollama 卡死能在 20 秒后被打断）
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	start := time.Now()
	stream, err := client.ChatStream(ctx, messages,
		core.WithTools(tools...),
		core.WithMaxTokens(500),
	)
	// 若请求建立阶段就超时，也是正确行为（ctx 超时生效）
	if err != nil {
		elapsed := time.Since(start)
		t.Logf("ChatStream 在请求阶段就超时返回: %v (%v)", err, elapsed)
		assert.Less(t, elapsed, 30*time.Second, "ctx 超时应生效")
		return
	}
	defer stream.Close()

	returned := make(chan struct{})
	go func() {
		defer close(returned)
		for stream.Next() {
		}
	}()

	select {
	case <-returned:
		elapsed := time.Since(start)
		t.Logf("流在 %v 后返回", elapsed)
		// 超时 20 秒 + body 关闭排空时间，应在 35 秒内返回
		assert.Less(t, elapsed, 40*time.Second, "ctx 超时后应在 35 秒内返回")
	case <-time.After(60 * time.Second):
		t.Fatal("FAIL: 流在 60 秒内未返回，超时未生效（修复未生效）")
	}
}

// buildMindxLikeTools 构造与 MindX backend-engineer agent 近似的工具集
func buildMindxLikeTools() []core.Tool {
	return []core.Tool{
		{
			Name:        "QuickSearch",
			Description: "高效语义搜索 — 按含义查找本项目内的代码和文档",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"搜索查询"}},"required":["query"]}`),
		},
		{
			Name:        "Grep",
			Description: "本地全文搜索",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string"}},"required":["pattern"]}`),
		},
		{
			Name:        "Read",
			Description: "从本地文件系统读取文件",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		},
		{
			Name:        "Glob",
			Description: "查找文件",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"}},"required":["pattern"]}`),
		},
		{
			Name:        "WebSearch",
			Description: "搜索网络以获取实时信息",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`),
		},
	}
}

// ============================================================
// Mock 测试：ctx 取消能打断模拟的阻塞流（CI 友好）
// ============================================================
// 不依赖真实 ollama，用 mock server 模拟"发一个 chunk 后永久阻塞"
// 的场景，验证 ctx 取消能打断 scanner.Scan() 阻塞。

// TestMock_CtxCancel_BreaksBlockedStream 验证修复后的 ctx 响应能力。
// mock server 发一个 chunk 后阻塞，模拟 ollama 卡死。
// ctx 取消后 stream.Next() 应在 5 秒内返回 false。
func TestMock_CtxCancel_BreaksBlockedStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)

		// 发一个 chunk 后永久阻塞（模拟 ollama 卡死）
		_, _ = w.Write([]byte(ndjsonLine(&ollamaMessage{Role: "assistant", Content: "hi"}, false, "", 0, 0) + "\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// 阻塞直到 ctx 取消或连接断开
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client, err := NewOllamaClient(core.Config{
		BaseURL: server.URL,
		Model:   "test-model",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.ChatStream(ctx, []core.Message{core.NewUserMessage("test")})
	require.NoError(t, err)
	defer stream.Close()

	// 消费第一个 chunk
	require.True(t, stream.Next(), "应收到第一个 chunk")
	ev := stream.Event()
	assert.Equal(t, core.EventContent, ev.Type)

	// 1 秒后取消 ctx
	go func() {
		time.Sleep(1 * time.Second)
		cancel()
	}()

	// 验证 stream.Next() 在 ctx 取消后 5 秒内返回 false
	start := time.Now()
	returned := false
	for {
		if !stream.Next() {
			returned = true
			break
		}
		if time.Since(start) > 10*time.Second {
			break
		}
	}

	elapsed := time.Since(start)
	assert.True(t, returned, "stream.Next() 应在 ctx 取消后返回 false")
	assert.Less(t, elapsed, 5*time.Second, "应在 ctx 取消后 5 秒内返回，而非永久阻塞")
	t.Logf("stream 在 %v 后返回", elapsed)
}

// ============================================================
// 完整请求体真实复现测试
// ============================================================
// 以下测试把 MindX 实测中导致卡死的完整请求体原样复制到 gochat 的 ollama 测试中，
// 连接真实 ollama server (minicpm-v4.6:latest)，验证是否能正确执行。
//
// 出问题场景（来自 MindX session 01KYPVXA8TVBBAPYNSVN4H49QY）：
//   - 模型: minicpm-v4.6:latest（thinking 模型，context_length=4096）
//   - 第一轮对话成功（prompt_tokens=2831, completion_tokens=71, 耗时 28s）
//   - 第二轮对话卡死（"你可以帮我查一下 Bega 项目的介绍吗"）
//
// 根因：ollama client 的 buildRequest 没有把 assistant 消息的 ReasoningContent
// 转成 ollamaMessage.Thinking 字段。minicpm-v4.6 是 thinking 模型，
// 多轮对话时 ollama 期望历史 assistant 消息带 thinking，缺失会导致卡死。

// newLiveClientWithModel 创建指向真实 ollama server 的 client，支持指定模型。
func newLiveClientWithModel(t *testing.T, model string) *Client {
	t.Helper()
	client, err := NewOllamaClient(core.Config{
		BaseURL: "http://localhost:11434",
		Model:   model,
	})
	require.NoError(t, err)
	return client
}

// loadMindxSystemPrompt 从 testdata 加载真实的 MindX system prompt。
// 这个 9478 字节的 prompt 来自 MindX backend-engineer agent 的真实会话，
// 包含身份、技能目录、行为准则、环境信息、工具目录等完整段落。
func loadMindxSystemPrompt(t *testing.T) string {
	t.Helper()
	sp, err := os.ReadFile("testdata/mindx_system_prompt.txt")
	require.NoError(t, err, "需要 testdata/mindx_system_prompt.txt")
	return string(sp)
}

// buildMindxReproMessages 构造与 MindX 第二次对话完全相同的消息序列。
//
// 消息序列（来自 session 01KYPVXA8TVBBAPYNSVN4H49QY 的真实对话历史）：
//  1. user: "你好，你的职责是什么？"
//  2. assistant: content="我的职责是帮助设计与维护服务端应用..." + reasoning_content="用户想了解我的职责..."
//  3. user: "你可以帮我查一下 Bega 项目的介绍吗"（这一轮卡死）
//
// 关键：assistant 消息必须带 ReasoningContent，这是 thinking 模型多轮对话的必要字段。
func buildMindxReproMessages() []core.Message {
	assistantMsg := core.NewTextMessage("assistant",
		"我的职责是帮助设计与维护服务端应用，具体包括研发服务端代码、构建API接口以及管理数据库等相关工作。")
	assistantMsg.ReasoningContent = "用户想了解我的职责。我需要基于角色描述来回答。\n\n" +
		"根据之前的对话设置，我是后端工程师，负责服务端开发。我应该告知用户的职责是设计、开发和维护服务端应用、API 和数据管道。\n"

	return []core.Message{
		core.NewUserMessage("你好，你的职责是什么？"),
		assistantMsg,
		core.NewUserMessage("你可以帮我查一下 Bega 项目的介绍吗"),
	}
}

// TestLive_MindxRepro_FullRequestBody 用 MindX 完整请求体做真实测试。
//
// 这是核心验证测试：把出问题的请求体原样复制到 gochat 测试中，
// 连接真实 ollama server (minicpm-v4.6:latest)，验证是否能正确执行。
//
// 步骤：
//  1. 加载真实 system prompt（9478 字节，来自 testdata/mindx_system_prompt.txt）
//  2. 构造 MindX 第二次对话的完整消息序列（含 assistant 的 reasoning_content）
//  3. dump 请求体 JSON，验证 thinking 字段是否存在
//  4. 发送给真实 ollama server，设置 90 秒超时
//  5. 验证是否在合理时间内返回（修复前卡死，修复后应正常返回）
//
// 运行：OLLAMA_LIVE=1 go test ./client/ollama/ -run TestLive_MindxRepro_FullRequestBody -v -timeout 120s
func TestLive_MindxRepro_FullRequestBody(t *testing.T) {
	skipUnlessLive(t)

	systemPrompt := loadMindxSystemPrompt(t)
	messages := buildMindxReproMessages()

	// 使用与 MindX 完全相同的模型配置
	client := newLiveClientWithModel(t, "minicpm-v4.6:latest")

	// dump 请求体 JSON，验证 thinking 字段是否存在
	// 这一步用于诊断：buildRequest 是否正确处理了 ReasoningContent
	t.Run("dump_request_body", func(t *testing.T) {
		options := core.ApplyOptions(
			core.WithSystemPrompt(systemPrompt),
			core.WithTemperature(0.0),
			core.WithMaxTokens(500),
		)
		reqBody := client.buildRequest(messages, options, true)
		jsonData, err := json.MarshalIndent(reqBody, "", "  ")
		require.NoError(t, err)

		// 验证请求体结构
		assert.Equal(t, "minicpm-v4.6:latest", reqBody.Model)
		assert.True(t, reqBody.Stream)

		// system prompt 应作为第一条消息（通过 SystemPrompt 注入）
		require.NotEmpty(t, reqBody.Messages)
		assert.Equal(t, "system", reqBody.Messages[0].Role)

		// 找到 assistant 消息，验证 thinking 字段
		// 修复前：Thinking 为空（ReasoningContent 未被转换）
		// 修复后：Thinking 应等于 assistantMsg.ReasoningContent
		var assistantMsg *ollamaMessage
		for i := range reqBody.Messages {
			if reqBody.Messages[i].Role == "assistant" {
				assistantMsg = &reqBody.Messages[i]
				break
			}
		}
		require.NotNil(t, assistantMsg, "请求体应包含 assistant 消息")
		t.Logf("assistant 消息 thinking 字段长度: %d", len(assistantMsg.Thinking))
		if assistantMsg.Thinking == "" {
			t.Log("⚠ 诊断：assistant 消息的 thinking 字段为空 — buildRequest 未转换 ReasoningContent")
		} else {
			t.Log("✓ assistant 消息的 thinking 字段已正确设置")
		}

		// 把完整请求体写入临时文件，供离线分析
		dumpPath := filepath.Join(t.TempDir(), "mindx_request_body.json")
		require.NoError(t, os.WriteFile(dumpPath, jsonData, 0644))
		t.Logf("请求体已 dump 到: %s", dumpPath)
		t.Logf("请求体大小: %d bytes, 消息数: %d", len(jsonData), len(reqBody.Messages))
	})

	// 真实流式调用：验证是否能正确执行
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	start := time.Now()
	stream, err := client.ChatStream(ctx, messages,
		core.WithSystemPrompt(systemPrompt),
		core.WithTemperature(0.0),
		core.WithMaxTokens(500),
	)
	require.NoError(t, err)
	defer stream.Close()

	var content, thinking string
	var done bool
	returned := make(chan struct{})

	go func() {
		defer close(returned)
		for stream.Next() {
			ev := stream.Event()
			switch ev.Type {
			case core.EventContent:
				content += ev.Content
			case core.EventThinking:
				thinking += ev.Content
			case core.EventDone:
				done = true
			}
		}
	}()

	select {
	case <-returned:
		elapsed := time.Since(start)
		t.Logf("流在 %v 后返回: done=%v contentLen=%d thinkingLen=%d",
			elapsed, done, len(content), len(thinking))
		if elapsed > 60*time.Second {
			t.Logf("⚠ 耗时较长（%v），可能接近卡死边界", elapsed)
		}
		// 只要流能正常返回（不卡死），就算通过
		assert.Less(t, elapsed, 90*time.Second, "流应在 90 秒内返回（不卡死）")
	case <-time.After(120 * time.Second):
		t.Fatal("FAIL: 流在 120 秒内未返回，复现了 MindX 卡死 bug")
	}
}
