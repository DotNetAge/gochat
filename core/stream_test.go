package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStream_Next_Event(t *testing.T) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventContent, Content: " world"}
	ch <- StreamEvent{Type: EventDone, Usage: &Usage{TotalTokens: 10}}
	close(ch)

	stream := NewStream(ch, nil)

	// 测试第一个事件
	assert.True(t, stream.Next())
	event := stream.Event()
	assert.Equal(t, EventContent, event.Type)
	assert.Equal(t, "Hello", event.Content)

	// 测试第二个事件
	assert.True(t, stream.Next())
	event = stream.Event()
	assert.Equal(t, EventContent, event.Type)
	assert.Equal(t, " world", event.Content)

	// 测试第三个事件
	assert.True(t, stream.Next())
	event = stream.Event()
	assert.Equal(t, EventDone, event.Type)
	assert.NotNil(t, event.Usage)
	assert.Equal(t, 10, event.Usage.TotalTokens)

	// 测试流结束
	assert.False(t, stream.Next())
}

func TestStream_Close(t *testing.T) {
	ch := make(chan StreamEvent, 1)
	ch <- StreamEvent{Type: EventContent, Content: "test"}
	close(ch)

	stream := NewStream(ch, nil)

	// 测试关闭流
	err := stream.Close()
	assert.NoError(t, err)

	// 测试重复关闭
	err = stream.Close()
	assert.NoError(t, err)

	// 测试关闭后 Next 返回 false
	assert.False(t, stream.Next())
}

func TestStream_Err(t *testing.T) {
	testErr := errors.New("test error")
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventError, Err: testErr}
	close(ch)

	stream := NewStream(ch, nil)

	// 测试第一个事件
	assert.True(t, stream.Next())
	assert.Nil(t, stream.Err())

	// 测试错误事件
	assert.True(t, stream.Next())
	assert.Equal(t, testErr, stream.Err())

	// 测试流结束
	assert.False(t, stream.Next())
}

func TestStream_Usage(t *testing.T) {
	usage := &Usage{TotalTokens: 100}
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventDone, Usage: usage}
	close(ch)

	stream := NewStream(ch, nil)

	// 测试流开始时 Usage 为 nil
	assert.Nil(t, stream.Usage())

	// 测试第一个事件
	assert.True(t, stream.Next())
	assert.Nil(t, stream.Usage())

	// 测试第二个事件（包含 Usage）
	assert.True(t, stream.Next())
	assert.Equal(t, usage, stream.Usage())
}

func TestStream_Text(t *testing.T) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventThinking, Content: "I'm thinking"}
	ch <- StreamEvent{Type: EventContent, Content: " world"}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.Text()
	assert.NoError(t, err)
	assert.Equal(t, "Hello world", text)
}

func TestStream_Text_Error(t *testing.T) {
	testErr := errors.New("test error")
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventError, Err: testErr}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.Text()
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, "Hello", text)
}

func TestStream_ReasoningText(t *testing.T) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventContent, Content: "Hello"}
	ch <- StreamEvent{Type: EventThinking, Content: "I'm thinking"}
	ch <- StreamEvent{Type: EventContent, Content: " world"}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.ReasoningText()
	assert.NoError(t, err)
	assert.Equal(t, "I'm thinking", text)
}

func TestStream_ReasoningText_Error(t *testing.T) {
	testErr := errors.New("test error")
	ch := make(chan StreamEvent, 2)
	ch <- StreamEvent{Type: EventThinking, Content: "I'm thinking"}
	ch <- StreamEvent{Type: EventError, Err: testErr}
	close(ch)

	stream := NewStream(ch, nil)

	text, err := stream.ReasoningText()
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, "I'm thinking", text)
}
