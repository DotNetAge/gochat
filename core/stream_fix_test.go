package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream_Close_WithTimeout(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	close(ch)

	stream := NewStream(ch, nil)
	err := stream.Close()
	assert.NoError(t, err)
}

func TestStream_Close_MultipleCalls(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	close(ch)

	stream := NewStream(ch, nil)

	err1 := stream.Close()
	err2 := stream.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestStream_Close_WithoutProducer(t *testing.T) {
	ch := make(chan StreamEvent)
	stream := NewStream(ch, nil)

	done := make(chan error, 1)
	go func() {
		done <- stream.Close()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(6 * time.Second):
		t.Fatal("Close() appears to be blocking - possible deadlock")
	}
}

func TestStream_ContextCancellation(t *testing.T) {
	ch := make(chan StreamEvent, 10)
	for i := 0; i < 5; i++ {
		ch <- StreamEvent{Type: EventContent, Content: "data"}
	}

	stream := NewStream(ch, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for stream.Next() {
		if ctx.Err() != nil {
			break
		}
	}
}

func TestStream_Text_NoDoubleLocking(t *testing.T) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventContent, Content: " "}
	ch <- StreamEvent{Type: EventContent, Content: "World"}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.Text()
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", text)
}

func TestStream_ReasoningText_NoDoubleLocking(t *testing.T) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventThinking, Content: "Thinking..."}
	ch <- StreamEvent{Type: EventContent, Content: "Answer"}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.ReasoningText()
	assert.NoError(t, err)
	assert.Equal(t, "Thinking...", text)
}

func TestStream_Event_AfterClose(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	close(ch)

	stream := NewStream(ch, nil)
	stream.Close()

	event := stream.Event()
	assert.Equal(t, StreamEvent{}, event)
}

func TestStream_Next_AfterClose(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	close(ch)

	stream := NewStream(ch, nil)
	stream.Close()

	assert.False(t, stream.Next())
}

// TestStream_Next_ChannelClosedWithoutDone 回归：生产者未发送 EventDone/EventError 就关闭
// 通道（模型端静默断开 / 连接被超时打断）时，Next() 必须快速返回 false。
// 修复前：Next() 命中 `!ok` 分支调用 autoClose()，与自身持有的锁重入导致永久死锁。
func TestStream_Next_ChannelClosedWithoutDone(t *testing.T) {
	ch := make(chan StreamEvent)
	stream := NewStream(ch, nil)

	go func() {
		close(ch)
	}()

	nextReturned := make(chan bool, 1)
	go func() {
		nextReturned <- stream.Next()
	}()

	select {
	case v := <-nextReturned:
		assert.False(t, v, "流结束后 Next() 应返回 false")
	case <-time.After(3 * time.Second):
		t.Fatal("Next() 未在 3 秒内返回，存在锁重入死锁")
	}
}

func TestStream_Usage_AfterClose(t *testing.T) {
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	ch <- StreamEvent{Type: EventDone, Usage: &Usage{TotalTokens: 10}}
	close(ch)

	stream := NewStream(ch, nil)
	for stream.Next() {
	}
	stream.Close()

	usage := stream.Usage()
	assert.NotNil(t, usage)
	assert.Equal(t, 10, usage.TotalTokens)
}
