package core

import (
	"io"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ToolCallDelta represents an incremental tool call delta received during
// streaming. Tool calls arrive incrementally across multiple chunks; each
// chunk carries a delta for one or more tool calls identified by Index.
// The ID and Name are only set in the first delta chunk for each index;
// subsequent deltas accumulate Arguments incrementally.
type ToolCallDelta struct {
	Index     int    `json:"index"`
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// streamCloseTimeout is the maximum time Close() will wait for the
// stream channel to drain before giving up.
const streamCloseTimeout = 5 * time.Second

// defaultTailWait 是收到 finish_reason 之后等待尾部 usage 帧的上限。
// 需覆盖正常网络尾包延迟（毫秒级），又不能让「对端既不发 [DONE] 也不关连接」
// 的情形明显拖慢收尾。
const defaultTailWait = 500 * time.Millisecond

// StreamEventType classifies the type of event received during streaming.
type StreamEventType string

// Stream event type constants indicating what kind of data
// was received in a streaming response.
const (
	// EventContent indicates a content chunk was received.
	// This is the main type for text output from the model.
	EventContent StreamEventType = "content"

	// EventThinking indicates a reasoning/thinking chunk was received.
	// This is used by models that support extended thinking.
	EventThinking StreamEventType = "thinking"

	// EventToolCall indicates an incremental tool call delta was received.
	// Tool calls arrive across multiple chunks and must be accumulated by
	// the Stream consumer. The ToolCallDeltas field carries the delta data.
	EventToolCall StreamEventType = "tool_call"

	// EventDone indicates the stream has completed successfully.
	// This signals that all data has been received.
	EventDone StreamEventType = "done"

	// EventError indicates an error occurred during streaming.
	// The error details are in the Err field of StreamEvent.
	EventError StreamEventType = "error"
)

// StreamEvent represents a single unit of data received during streaming.
// Events are sent through a channel as they arrive from the server.
type StreamEvent struct {
	// Type indicates what kind of event this is (content, thinking, tool_call, done, error).
	Type StreamEventType

	// Content contains the text data for content or thinking events.
	// This is empty for done or error events.
	Content string

	// ToolCallDeltas contains incremental tool call data for tool_call events.
	// Tool calls arrive incrementally across multiple chunks and must be
	// accumulated by index to reconstruct the complete tool call.
	ToolCallDeltas []ToolCallDelta

	// Refusal is set when the content was generated from the refusal field
	// (content_filter). Non-empty when the model's response was flagged by
	// content moderation.
	Refusal string

	// FinishReason indicates why the model stopped generating.
	// Set on EventDone: "stop", "tool_calls", "length", "content_filter".
	FinishReason string

	// Usage contains token usage statistics when available.
	// This is typically populated when the stream completes.
	Usage *Usage

	// Err is non-nil if an error occurred during streaming.
	// When this is set, the stream has terminated abnormally.
	Err error
}

// streamToolCallState tracks the accumulation of a single tool call
// across multiple streaming delta chunks.
type streamToolCallState struct {
	id        string
	name      string
	arguments strings.Builder
}

// Stream provides a thread-safe interface for consuming streaming responses.
// It wraps a channel of StreamEvents and provides methods for iterating
// through the stream, collecting text, and proper resource cleanup.
//
// NOTE: Stream is a single-consumer model. While it is thread-safe for
// concurrent Close() calls, only one goroutine should call Next() at a time
// to ensure event order and consistency.
//
// The Close method should be called when the stream is no longer needed
// to ensure proper cleanup of underlying resources.
type Stream struct {
	ch      <-chan StreamEvent
	current StreamEvent
	done    bool
	closed  bool
	closer  io.Closer
	usage   *Usage
	mu      sync.Mutex
	doneCh  chan struct{}
	once    sync.Once

	// tailWait 是收到 EventDone 后等待尾部 usage 帧的上限，零值取 defaultTailWait。
	tailWait time.Duration

	// tailDeadline 是尾部 drain 的绝对截止时刻。首次进入 drainTail 时设定，
	// 之后所有 drainTail 调用共用同一个截止时刻，确保总等待严格不超过 tailWait。
	// 若用「每次调用新建 timer」，则每收到一个尾部非 usage 事件就会重新计满一轮，
	// 总等待可被无限拉长——与 tailWait 的语义不符。
	tailDeadline time.Time

	// Tool call accumulation state
	toolCallAcc map[int]*streamToolCallState // accumulation state by delta index
	toolCalls   []ToolCall                   // final accumulated tool calls
}

// NewStream creates a new Stream wrapping the provided event channel.
// The closer is used to close underlying resources (like the HTTP response)
// when Close is called. The closer may be nil if no cleanup is needed.
//
// Parameters:
//   - ch: The channel to read stream events from
//   - closer: An optional io.Closer to call when the stream is closed
//
// Returns a new Stream instance
func NewStream(ch <-chan StreamEvent, closer io.Closer) *Stream {
	s := &Stream{
		ch:       ch,
		closer:   closer,
		doneCh:   make(chan struct{}),
		tailWait: defaultTailWait,
	}
	if closer != nil {
		runtime.SetFinalizer(s, func(obj *Stream) {
			obj.Close()
		})
	}
	return s
}

// initToolCallAcc lazily initializes the tool call accumulator.
func (s *Stream) initToolCallAcc() {
	if s.toolCallAcc == nil {
		s.toolCallAcc = make(map[int]*streamToolCallState)
	}
}

// accumulateToolCalls processes ToolCallDeltas from an event, merging
// incremental arguments into the ongoing accumulator by delta index.
func (s *Stream) accumulateToolCalls(deltas []ToolCallDelta) {
	s.initToolCallAcc()
	for _, d := range deltas {
		state, ok := s.toolCallAcc[d.Index]
		if !ok {
			state = &streamToolCallState{}
			s.toolCallAcc[d.Index] = state
		}
		if d.ID != "" {
			state.id = d.ID
		}
		if d.Name != "" {
			state.name = d.Name
		}
		if d.Arguments != "" {
			state.arguments.WriteString(d.Arguments)
		}
	}
}

// finalizeToolCalls converts the accumulated tool call states into final
// ToolCall slices. This is called when the stream ends (EventDone).
func (s *Stream) finalizeToolCalls() {
	if len(s.toolCallAcc) == 0 {
		s.toolCalls = nil
		return
	}
	// Collect by index order for deterministic output
	maxIdx := 0
	for idx := range s.toolCallAcc {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	result := make([]ToolCall, 0, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		state, ok := s.toolCallAcc[i]
		if !ok {
			continue
		}
		result = append(result, ToolCall{
			ID:        state.id,
			Name:      state.name,
			Arguments: state.arguments.String(),
		})
	}
	s.toolCalls = result
	s.toolCallAcc = nil // release memory
}

// Next advances the stream to the next event. Returns false if the
// stream has ended (either normally or due to an error) or if
// Close was called.
//
// IMPORTANT: With include_usage=true, providers send a trailing usage-only
// chunk (choices:[]) AFTER the finish_reason chunk. We must continue
// draining events even after EventDone to capture this late usage data.
func (s *Stream) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// After done, still drain remaining channel events to catch
	// trailing usage-only chunks sent after finish_reason.
	if s.done || s.closed {
		return s.drainTail()
	}
	select {
	case <-s.doneCh:
		s.done = true
		s.closeInternal()
		return false
	case ev, ok := <-s.ch:
		if !ok {
			s.done = true
			s.closeInternal()
			s.finalizeToolCalls()
			return false
		}
		s.current = ev
		if ev.Err != nil {
			s.done = true
			s.closeInternal()
		}
		if ev.Usage != nil {
			s.usage = ev.Usage
		}
		if ev.Type == EventToolCall {
			s.accumulateToolCalls(ev.ToolCallDeltas)
		}
		if ev.Type == EventDone {
			// On streaming tool calls, the final chunk has finish_reason="tool_calls"
			// but the delta is empty — the actual tool call data came in prior chunks.
			// finalizeToolCalls merges all accumulated deltas into final ToolCall objects.
			s.finalizeToolCalls()
			// Capture finish reason for consumer inspection
			if ev.FinishReason != "" {
				s.current = ev
			}
			s.done = true
			// 不在此处关闭 body：finish_reason 之后通常还有 trailing usage 帧和
			// data: [DONE]，生产者 goroutine 此时仍阻塞在读 body 上。提前 Close 会让
			// 它读到 "http2: response body closed" 并被透传成 EventError，把正常的
			// 流式收尾误报成网络断流。body 由生产者的 defer 负责关闭。
		}
		return true
	}
}

// drainTail 在已收到 EventDone 之后继续从通道取事件，用于捕获 include_usage=true
// 时在 finish_reason 之后单独发送的 usage 帧。
//
// 关键约束：纯 usage 帧不覆盖 s.current。current 保存的是 finish_reason 帧，消费方
// （例如在循环结束后调用 Event() 读取 FinishReason）依赖它；若被后续 usage 帧覆盖，
// FinishReason 会变成空串。
//
// 返回 false 表示尾部已耗尽（生产者关闭通道）。等待上限由 tailWait 控制，
// 避免对端在 finish_reason 之后既不发送 [DONE] 也不关闭连接时无限阻塞。
//
// 上限是「从首次进入 drainTail 起算的总时长」，不是每次调用各自计时：
// tailDeadline 在首次进入时固定，此后每次调用都复用它。否则尾部若夹带多个
// 非 usage 事件，每返回一次就重置一次 timer，总等待会被无限拉长。
func (s *Stream) drainTail() bool {
	if s.tailDeadline.IsZero() {
		wait := s.tailWait
		if wait <= 0 {
			wait = defaultTailWait
		}
		s.tailDeadline = time.Now().Add(wait)
	}
	timer := time.NewTimer(time.Until(s.tailDeadline))
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-s.ch:
			if !ok {
				return false
			}
			if ev.Usage != nil {
				s.usage = ev.Usage
			}
			// 生产者报错必须冒泡，不能被尾部 drain 吞掉。
			if ev.Err != nil {
				s.current = ev
				return true
			}
			// current 保存的是 finish_reason 帧，消费方（例如循环结束后调用
			// Event() 读 FinishReason）依赖它。finish_reason 之后到达的任何事件
			// 都不得覆盖 current——不只是纯 usage 帧：某些服务端还会补发
			// content 帧或额外的 EventDone。覆盖会让 FinishReason 变成空串，
			// 进而破坏 goagent 对 stop / tool_calls 的判断。
			// 故这里只采集 usage，不写 current。
		case <-timer.C:
			return false
		}
	}
}

// closeInternal closes the underlying closer without acquiring the lock.
// This must only be called when the lock is already held.
func (s *Stream) closeInternal() {
	if s.closer != nil && !s.closed {
		s.closer.Close()
		s.closed = true
	}
}

// Event returns the most recent event received from the last call to Next.
// The returned event is valid until the next call to Next or Close.
//
// Returns the current StreamEvent
func (s *Stream) Event() StreamEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// Close releases resources associated with the stream. It is safe to call
// multiple times. After Close is called, Next will always return false.
//
// 顺序很重要：先关 closer（HTTP body）再排空通道。
// 生产者 goroutine 通常阻塞在 resp.Body.Read 上；只有关闭 body 才能让它读到
// EOF/错误并退出、进而关闭通道。若反过来先 drain 通道再关 body，drain 必然
// 等满 streamCloseTimeout（因为唤醒生产者的动作被排在它后面）。
// 先关 body 后，生产者立即解除阻塞并关通道，drain 通常在一轮内结束。
//
// 排空仍保留，是为了让生产者把已写入的尾部事件送达、避免 goroutine 泄漏；
// 有 streamCloseTimeout 兜底，不会永久阻塞。
//
// Returns an error if closing the underlying closer fails
func (s *Stream) Close() error {
	s.once.Do(func() {
		close(s.doneCh)
	})

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.done = true
	s.mu.Unlock()

	// Clear finalizer since we're closing manually
	runtime.SetFinalizer(s, nil)

	// 先释放生产者：它多半正卡在读 body 上，不关 body 就不会退出。
	var closeErr error
	if s.closer != nil {
		closeErr = s.closer.Close()
	}

	timer := time.NewTimer(streamCloseTimeout)
	defer timer.Stop()

	for {
		select {
		case _, ok := <-s.ch:
			if !ok {
				return closeErr
			}
		case <-timer.C:
			return closeErr
		}
	}
}

// Err returns any error that occurred during streaming.
// If the stream ended normally, nil is returned.
//
// Returns the error from the stream, or nil if no error occurred
func (s *Stream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current.Err
}

// Usage returns the token usage statistics if they were received.
// Usage information is typically only available after the stream
// has completed.
//
// Returns a pointer to Usage statistics, or nil if not yet received
func (s *Stream) Usage() *Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

// ToolCalls returns the accumulated tool calls from the streaming response.
// Tool calls are accumulated from EventToolCall deltas during Next() iteration
// and finalized when EventDone is received (finish_reason="tool_calls").
//
// The returned slice is nil if no tool calls were made by the model.
// Results are ordered by the delta index from the streaming response.
//
// Returns the accumulated ToolCall slice
func (s *Stream) ToolCalls() []ToolCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.toolCalls
}

// Text collects and returns all content events as a single string.
// This is a convenience method for when you only care about the
// text output and not the individual events.
//
// The method consumes the entire stream until it ends or an error
// occurs. After Text returns, the stream is fully consumed and
// should not be used further.
//
// Returns the concatenated text content and any error that occurred
func (s *Stream) Text() (string, error) {
	var buf strings.Builder
	for {
		s.mu.Lock()
		if s.done {
			s.mu.Unlock()
			break
		}
		select {
		case <-s.doneCh:
			s.done = true
			s.mu.Unlock()
			goto checkError
		case ev, ok := <-s.ch:
			if !ok {
				s.done = true
				s.mu.Unlock()
				goto checkError
			}
			s.current = ev
			if ev.Err != nil {
				s.done = true
				s.mu.Unlock()
				return buf.String(), ev.Err
			}
			if ev.Usage != nil {
				s.usage = ev.Usage
			}
			switch ev.Type {
			case EventContent:
				buf.WriteString(ev.Content)
			case EventToolCall:
				s.accumulateToolCalls(ev.ToolCallDeltas)
			case EventDone:
				s.finalizeToolCalls()
				s.done = true
				s.mu.Unlock()
				goto checkError
			}
			s.mu.Unlock()
		}
	}

checkError:
	if s.current.Err != nil {
		return buf.String(), s.current.Err
	}
	return buf.String(), nil
}

// ReasoningText collects and returns all thinking events as a single string.
// This is used for models that provide extended thinking/reasoning output
// separately from the main content.
//
// The method consumes the entire stream until it ends or an error
// occurs. After ReasoningText returns, the stream is fully consumed.
//
// Returns the concatenated thinking content and any error that occurred
func (s *Stream) ReasoningText() (string, error) {
	var buf strings.Builder
	for {
		s.mu.Lock()
		if s.done {
			s.mu.Unlock()
			break
		}
		select {
		case <-s.doneCh:
			s.done = true
			s.mu.Unlock()
			goto checkError
		case ev, ok := <-s.ch:
			if !ok {
				s.done = true
				s.mu.Unlock()
				goto checkError
			}
			s.current = ev
			if ev.Err != nil {
				s.done = true
				s.mu.Unlock()
				return buf.String(), ev.Err
			}
			if ev.Usage != nil {
				s.usage = ev.Usage
			}
			switch ev.Type {
			case EventThinking:
				buf.WriteString(ev.Content)
			case EventToolCall:
				s.accumulateToolCalls(ev.ToolCallDeltas)
			case EventDone:
				s.finalizeToolCalls()
				s.done = true
				s.mu.Unlock()
				goto checkError
			}
			s.mu.Unlock()
		}
	}

checkError:
	if s.current.Err != nil {
		return buf.String(), s.current.Err
	}
	return buf.String(), nil
}

// RefusalText collects and returns all refusal content as a single string.
// This is used when the model's response was flagged by content moderation
// (finish_reason="content_filter"). Refusal text arrives incrementally via
// delta.refusal in the streaming response.
//
// The method consumes the entire stream until it ends or an error
// occurs. After RefusalText returns, the stream is fully consumed.
//
// Returns the concatenated refusal text and any error that occurred
func (s *Stream) RefusalText() (string, error) {
	var buf strings.Builder
	for {
		s.mu.Lock()
		if s.done {
			s.mu.Unlock()
			break
		}
		select {
		case <-s.doneCh:
			s.done = true
			s.mu.Unlock()
			goto checkErrorRefusal
		case ev, ok := <-s.ch:
			if !ok {
				s.done = true
				s.mu.Unlock()
				goto checkErrorRefusal
			}
			s.current = ev
			if ev.Err != nil {
				s.done = true
				s.mu.Unlock()
				return buf.String(), ev.Err
			}
			if ev.Usage != nil {
				s.usage = ev.Usage
			}
			switch ev.Type {
			case EventContent:
				if ev.Refusal != "" {
					buf.WriteString(ev.Content)
				}
			case EventToolCall:
				s.accumulateToolCalls(ev.ToolCallDeltas)
			case EventDone:
				s.finalizeToolCalls()
				s.done = true
				s.mu.Unlock()
				goto checkErrorRefusal
			}
			s.mu.Unlock()
		}
	}

checkErrorRefusal:
	if s.current.Err != nil {
		return buf.String(), s.current.Err
	}
	return buf.String(), nil
}
