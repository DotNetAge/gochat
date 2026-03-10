package core

import (
	"io"
	"strings"
	"sync"
)

// StreamEventType identifies the type of stream event
type StreamEventType string

const (
	EventContent  StreamEventType = "content"
	EventThinking StreamEventType = "thinking" // reasoning/thinking content from the model
	EventDone     StreamEventType = "done"
	EventError    StreamEventType = "error"
)

// StreamEvent is one event from a streaming response
type StreamEvent struct {
	Type    StreamEventType
	Content string
	Usage   *Usage
	Err     error
}

// Stream represents an active streaming response.
// Callers iterate with Next() and must call Close().
type Stream struct {
	ch      <-chan StreamEvent
	current StreamEvent
	done    bool
	closed  bool
	closer  io.Closer // optional, for closing the HTTP response body
	usage   *Usage
	mu      sync.Mutex
}

// NewStream creates a Stream from a channel and an optional closer
func NewStream(ch <-chan StreamEvent, closer io.Closer) *Stream {
	return &Stream{ch: ch, closer: closer}
}

// Next advances to the next event. Returns false when the stream is exhausted.
func (s *Stream) Next() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return false
	}
	ev, ok := <-s.ch
	if !ok {
		s.done = true
		return false
	}
	s.current = ev
	if ev.Err != nil {
		s.done = true
	}
	if ev.Usage != nil {
		s.usage = ev.Usage
	}
	return true
}

// Event returns the current stream event. Only valid after Next() returns true.
func (s *Stream) Event() StreamEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// Close releases resources associated with the stream.
// Safe to call multiple times.
func (s *Stream) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.done = true
	s.mu.Unlock()

	// Drain remaining events to unblock the producer goroutine
	for range s.ch {
	}
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// Err returns the error from the stream, if any
func (s *Stream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current.Err
}

// Usage returns the usage information after the stream completes
func (s *Stream) Usage() *Usage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.usage
}

// Text consumes the entire stream and returns the concatenated content
func (s *Stream) Text() (string, error) {
	var buf strings.Builder
	for s.Next() {
		ev := s.Event()
		if ev.Err != nil {
			return buf.String(), ev.Err
		}
		if ev.Type == EventContent {
			buf.WriteString(ev.Content)
		}
	}
	return buf.String(), nil
}

// ReasoningText consumes the entire stream and returns the concatenated thinking/reasoning content
func (s *Stream) ReasoningText() (string, error) {
	var buf strings.Builder
	for s.Next() {
		ev := s.Event()
		if ev.Err != nil {
			return buf.String(), ev.Err
		}
		if ev.Type == EventThinking {
			buf.WriteString(ev.Content)
		}
	}
	return buf.String(), nil
}
