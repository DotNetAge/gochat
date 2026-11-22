package openai

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TestParseSSEStream_DataWithoutSpace 锁定 SSE 规范兼容性：
// WHATWG 允许 "data:" 后省略前导空格，自建代理/兼容实现常这么发。
// 修复前硬编码 "data: " 前缀，这类流会被整体解析为空（0 个 chunk）。
func TestParseSSEStream_DataWithoutSpace(t *testing.T) {
	body := `data:{"id":"1","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}` + "\n\n"
	var got []string
	for c := range ParseSSEStream(bytes.NewReader([]byte(body))) {
		if c.Error != nil {
			t.Fatalf("unexpected err: %v", c.Error)
		}
		if len(c.Choices) > 0 && c.Choices[0].Delta != nil {
			got = append(got, c.Choices[0].Delta.Content)
		}
	}
	if len(got) != 1 || got[0] != "hi" {
		t.Fatalf("data: 无空格应解析出 1 个 chunk，got %v", got)
	}
}

// TestParseSSEStream_WithSpaceStillWorks 确认带空格形态不受影响。
func TestParseSSEStream_WithSpaceStillWorks(t *testing.T) {
	body := `data: {"id":"1","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":null}]}` + "\n\n"
	n := 0
	for c := range ParseSSEStream(bytes.NewReader([]byte(body))) {
		if c.Error == nil {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("data: 带空格应解析出 1 个 chunk，got %d", n)
	}
}

// TestParseSSEStream_LargeFrame 锁定大帧不被截断：
// bufio.Scanner 默认上限 64KB，超限即 bufio.ErrTooLong、整条流以
// "token too long" 断掉。OpenAI 兼容服务的正常帧只有几百字节
// （实测 dashscope + deepseek：784 帧，最大 387 字节），但非增量返回的
// provider、图片回显、大 tool_call arguments 都可能出现 MB 级单帧。
func TestParseSSEStream_LargeFrame(t *testing.T) {
	for _, size := range []int{70 * 1024, 1 << 20} {
		t.Run(fmt.Sprintf("%dKB", size/1024), func(t *testing.T) {
			big := strings.Repeat("x", size)
			body := fmt.Sprintf(
				`data: {"id":"1","choices":[{"index":0,"delta":{"content":"%s"},"finish_reason":null}]}`+"\n\n",
				big)
			var contentLen int
			var gotErr error
			for c := range ParseSSEStream(bytes.NewReader([]byte(body))) {
				if c.Error != nil {
					gotErr = c.Error
					break
				}
				if len(c.Choices) > 0 && c.Choices[0].Delta != nil {
					contentLen = len(c.Choices[0].Delta.Content)
				}
			}
			if gotErr != nil {
				t.Fatalf("%dKB 单帧不应报错，got %v", size/1024, gotErr)
			}
			if contentLen != size {
				t.Fatalf("%dKB 单帧内容被截断，got %d", size/1024, contentLen)
			}
		})
	}
}

// TestParseSSEStream_DoneWithoutSpace 确认 "[DONE]" 在省略空格时也能识别并终止。
func TestParseSSEStream_DoneWithoutSpace(t *testing.T) {
	body := `data:{"id":"1","choices":[{"index":0,"delta":{"content":"a"},"finish_reason":null}]}` + "\n\n" +
		"data:[DONE]\n\n" +
		`data:{"id":"1","choices":[{"index":0,"delta":{"content":"MUST_NOT_APPEAR"},"finish_reason":null}]}` + "\n\n"
	var got []string
	for c := range ParseSSEStream(bytes.NewReader([]byte(body))) {
		if len(c.Choices) > 0 && c.Choices[0].Delta != nil {
			got = append(got, c.Choices[0].Delta.Content)
		}
	}
	if len(got) != 1 || got[0] != "a" {
		t.Fatalf("[DONE] 之后的内容不得出现，got %v", got)
	}
}