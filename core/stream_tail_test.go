package core

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// trackingCloser 记录底层 body 是否被关闭，用于锁定「EventDone 不得提前关 body」。
type trackingCloser struct {
	closed bool
}

func (c *trackingCloser) Close() error {
	c.closed = true
	return nil
}

// TestStream_EventDone_DoesNotCloseBody 锁定回归：
//
// OpenAI 兼容服务端在 finish_reason 帧之后仍会发送 trailing usage 帧与 data: [DONE]。
// 生产者 goroutine 此时还在读 body，若 Next() 在 EventDone 处提前关闭 body，
// 生产者会读到 "http2: response body closed" 并被透传成 EventError，
// 把一次正常的流式收尾误报成网络断流（现象：任务会话内反复 body_closed 失败）。
func TestStream_EventDone_DoesNotCloseBody(t *testing.T) {
	closer := &trackingCloser{}
	ch := make(chan StreamEvent, 4)
	ch <- StreamEvent{Type: EventContent, Content: "hello"}
	ch <- StreamEvent{Type: EventDone, FinishReason: "tool_calls"}
	ch <- StreamEvent{Type: EventDone, Usage: &Usage{TotalTokens: 18}}
	close(ch)

	s := NewStream(ch, closer)

	assert.True(t, s.Next())
	assert.Equal(t, EventContent, s.Event().Type)

	assert.True(t, s.Next())
	assert.Equal(t, EventDone, s.Event().Type)
	assert.Equal(t, "tool_calls", s.Event().FinishReason)

	// 收到 EventDone 后 body 必须仍然打开：生产者还没读到 [DONE]。
	assert.False(t, closer.closed, "EventDone 处不得关闭 body，否则生产者读取会报 body closed")

	// 尾部 drain 结束后 FinishReason 不得被 usage 帧覆盖。
	for s.Next() {
	}
	assert.Equal(t, "tool_calls", s.Event().FinishReason, "尾部 usage 帧不得覆盖 FinishReason")
	assert.Equal(t, 18, s.Usage().TotalTokens)
}

// TestStream_TailDrain_CapturesLateUsage 锁定：finish_reason 之后单独到达的
// usage 帧必须被捕获（include_usage=true 的常规形态）。
func TestStream_TailDrain_CapturesLateUsage(t *testing.T) {
	ch := make(chan StreamEvent, 4)
	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
	ch <- StreamEvent{Type: EventDone, Usage: &Usage{TotalTokens: 42}}
	close(ch)

	s := NewStream(ch, nil)
	for s.Next() {
	}

	assert.NotNil(t, s.Usage())
	assert.Equal(t, 42, s.Usage().TotalTokens)
	assert.Equal(t, "stop", s.Event().FinishReason)
}

// TestStream_TailDrain_PropagatesProducerError 锁定：尾部 drain 期间生产者报的真实
// 错误仍然必须冒泡，不能被静默吞掉（否则网络断流会伪装成正常结束）。
func TestStream_TailDrain_PropagatesProducerError(t *testing.T) {
	wantErr := errors.New("boom")
	ch := make(chan StreamEvent, 4)
	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
	ch <- StreamEvent{Type: EventError, Err: wantErr}
	close(ch)

	s := NewStream(ch, nil)

	var got error
	for s.Next() {
		if ev := s.Event(); ev.Err != nil {
			got = ev.Err
			break
		}
	}
	assert.ErrorIs(t, got, wantErr)
}

// TestStream_TailDrain_TimeoutWhenProducerStalls 锁定：对端在 finish_reason 之后
// 既不发 [DONE] 也不关连接时，drain 必须按上限退出，不能永久阻塞。
func TestStream_TailDrain_TimeoutWhenProducerStalls(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
	// 通道保持打开且无后续事件：模拟悬挂

	s := NewStream(ch, nil)
	s.tailWait = 120 * time.Millisecond

	start := time.Now()
	for s.Next() {
	}
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 2*time.Second, "drain 必须按上限退出")
	assert.Equal(t, "stop", s.Event().FinishReason)
}

// TestStream_EventDone_BodyClosedByProducer 锁定正常收尾：生产者读到 EOF 后自己
// 关闭 body，Next 循环应正常结束且不产生错误。
func TestStream_EventDone_BodyClosedByProducer(t *testing.T) {
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}
	go func() {
		time.Sleep(10 * time.Millisecond)
		close(ch)
	}()

	s := NewStream(ch, io.NopCloser(nil))
	for s.Next() {
	}

	assert.NoError(t, s.Err())
	assert.Equal(t, "stop", s.Event().FinishReason)
}
