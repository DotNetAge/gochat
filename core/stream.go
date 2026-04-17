package core

import (
	"io"
	"runtime"
	"strings"
	"sync"
	"time"
)

// streamCloseTimeout is the maximum time Close() will wait for the
// stream channel to drain before giving up.
const streamCloseTimeout = 5 * time.Second

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
	// Type indicates what kind of event this is (content, thinking, done, error).
	Type StreamEventType

	// Content contains the text data for content or thinking events.
	// This is empty for done or error events.
	Content string

	// Usage contains token usage statistics when available.
	// This is typically populated when the stream completes.
	Usage *Usage

	// Err is non-nil if an error occurred during streaming.
	// When this is set, the stream has terminated abnormally.
	Err error
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
	ch       <-chan StreamEvent
	current  StreamEvent
	done     bool
	closed   bool
	closer   io.Closer
	usage    *Usage
	mu       sync.Mutex
	doneCh   chan struct{}
	once     sync.Once
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
		ch:     ch,
		closer: closer,
		doneCh: make(chan struct{}),
	}
	if closer != nil {
		runtime.SetFinalizer(s, func(obj *Stream) {
			obj.Close()
		})
	}
	return s
}

// Next advances the stream to the next event. Returns false if the
// stream has ended (either normally or due to an error) or if
// Close was called.
//
// This method is safe for concurrent use. Only one goroutine
// should call Next at a time, but Close may be called concurrently.
//
// Returns true if there is a new event available to process
func (s *Stream) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done || s.closed {
		return false
	}
	select {
	case <-s.doneCh:
		s.done = true
		s.autoClose()
		return false
	case ev, ok := <-s.ch:
		if !ok {
			s.done = true
			s.autoClose()
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
		if ev.Type == EventDone {
			s.done = true
			s.closeInternal()
		}
		return true
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

// autoClose attempts to close the underlying closer if the stream is done.
// It does not wait for the channel to drain like Close() does.
func (s *Stream) autoClose() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeInternal()
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
// Close will wait up to 5 seconds for the event channel to drain.
// If the producer goroutine has exited without closing the channel,
// Close will return after the timeout rather than blocking forever.
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

	timer := time.NewTimer(streamCloseTimeout)
	defer timer.Stop()

	for {
		select {
		case _, ok := <-s.ch:
			if !ok {
				goto closeCloser
			}
		case <-timer.C:
			goto closeCloser
		}
	}

closeCloser:
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
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
			if ev.Type == EventContent {
				buf.WriteString(ev.Content)
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
			if ev.Type == EventThinking {
				buf.WriteString(ev.Content)
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
