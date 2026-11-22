package core

import (
	"errors"
	"testing"
	"time"
)

// probeBody 模拟一个「只有被 Close 才会解除 Read 阻塞」的 body，
// 与真实 http.Response.Body 在服务端不再发送数据时的行为一致。
type probeBody struct {
	unblock chan struct{}
}

func newProbeBody() *probeBody {
	return &probeBody{unblock: make(chan struct{})}
}

func (b *probeBody) Read(p []byte) (int, error) {
	<-b.unblock
	return 0, errors.New("http2: response body closed")
}

func (b *probeBody) Close() error {
	select {
	case <-b.unblock:
	default:
		close(b.unblock)
	}
	return nil
}

// TestStream_TailNeverClobbersFinishReason 锁定：finish_reason 之后到达的
// **任何**事件都不得覆盖 current。
//
// 不只是纯 usage 帧——某些服务端在 [DONE] 前还会补发 content 帧或重复的
// EventDone。一旦覆盖，FinishReason 会变成空串，goagent 依据它判断
// stop / tool_calls 的逻辑随之失效。
func TestStream_TailNeverClobbersFinishReason(t *testing.T) {
	cases := []struct {
		name  string
		trail []StreamEvent
	}{
		{"trailing usage", []StreamEvent{{Type: EventDone, Usage: &Usage{TotalTokens: 9}}}},
		{"trailing content", []StreamEvent{{Type: EventContent, Content: "late"}}},
		{"trailing extra done", []StreamEvent{{Type: EventDone, FinishReason: ""}}},
		{"mixed", []StreamEvent{
			{Type: EventContent, Content: "late"},
			{Type: EventDone, Usage: &Usage{TotalTokens: 9}},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ch := make(chan StreamEvent, len(c.trail)+1)
			ch <- StreamEvent{Type: EventDone, FinishReason: "tool_calls"}
			for _, ev := range c.trail {
				ch <- ev
			}
			close(ch)

			s := NewStream(ch, nil)
			for s.Next() {
			}

			if got := s.Event().FinishReason; got != "tool_calls" {
				t.Fatalf("FinishReason 被尾部事件覆盖，got %q", got)
			}
		})
	}
}

// TestClose_ReleasesProducerBeforeDraining 锁定 Close() 的顺序约束：
// 必须先关 body（解除生产者的 Read 阻塞），再排空通道。
//
// 反例（修复前）：先 drain 通道再关 body。生产者正卡在读 body 上，
// 不关 body 它永不退出、通道永不关闭，drain 必然等满 streamCloseTimeout（5s）。
// 这不会造成数据错误，但每次悬挂收尾都白等 5 秒。
func TestClose_ReleasesProducerBeforeDraining(t *testing.T) {
	body := newProbeBody()
	ch := make(chan StreamEvent, 10)

	// 模拟真实生产者：读 body → 出错 → 发 EventError → 关通道。
	go func() {
		defer close(ch)
		if _, err := body.Read(make([]byte, 16)); err != nil {
			ch <- StreamEvent{Type: EventError, Err: err}
		}
	}()

	s := NewStream(ch, body)
	s.done = true // 已收到 EventDone，消费循环退出后 defer Close()

	start := time.Now()
	_ = s.Close()
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("Close() 耗时 %s：说明它在等一个只有关 body 才能解除的阻塞（顺序反了）", elapsed)
	}
}

// TestStream_TailWait_IsTotalBudget 锁定 tailWait 是「总预算」而非「每次调用重置」。
//
// 尾部若夹带多个非 usage 事件，每返回一次就重置 timer 会让总等待被无限拉长。
// 这里让尾部每秒吐一个 content 事件，共 20 个，tailWait 设 150ms：
// 总耗时必须显著小于「20 × 150ms」，证明预算共用。
func TestStream_TailWait_IsTotalBudget(t *testing.T) {
	ch := make(chan StreamEvent, 4)
	s := NewStream(ch, nil)
	s.tailWait = 150 * time.Millisecond

	// EventDone 先送达（缓冲足够，此处不会阻塞）
	ch <- StreamEvent{Type: EventDone, FinishReason: "stop"}

	// 后台不断吐尾部事件（模拟异常服务端持续发送无害帧）
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		tk := time.NewTicker(10 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				select {
				case ch <- StreamEvent{Type: EventContent, Content: "x"}:
				case <-stop:
					return
				}
			}
		}
	}()

	start := time.Now()
	for s.Next() {
	}
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("尾部 drain 耗时 %s，tailWait 预算被逐次重置了", elapsed)
	}
	if elapsed < 100*time.Millisecond {
		t.Fatalf("尾部 drain 只用了 %s，没有真正等待 tailWait 预算", elapsed)
	}
}
