package pipeline

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStep struct {
	mock.Mock
}

func (m *MockStep) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStep) Execute(ctx context.Context, state *State) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

type MockHook struct {
	mock.Mock
}

func (m *MockHook) OnStepStart(ctx context.Context, step Step[*State], state *State) {
	m.Called(ctx, step, state)
}

func (m *MockHook) OnStepError(ctx context.Context, step Step[*State], state *State, err error) {
	m.Called(ctx, step, state, err)
}

func (m *MockHook) OnStepComplete(ctx context.Context, step Step[*State], state *State) {
	m.Called(ctx, step, state)
}

func TestState(t *testing.T) {
	s := NewState()
	s.Set("key1", "value1")
	s.Set("key2", 123)

	val1, ok1 := s.Get("key1")
	assert.True(t, ok1)
	assert.Equal(t, "value1", val1)
	assert.Equal(t, "value1", s.GetString("key1"))

	val2, ok2 := s.Get("key2")
	assert.True(t, ok2)
	assert.Equal(t, 123, val2)
	assert.Equal(t, "", s.GetString("key2"))

	s.Delete("key1")
	_, ok1 = s.Get("key1")
	assert.False(t, ok1)

	clone := s.Clone()
	val2Clone, _ := clone.Get("key2")
	assert.Equal(t, 123, val2Clone)
	clone.Set("key2", 456)

	val2Original, _ := s.Get("key2")
	assert.Equal(t, 123, val2Original)
}

func TestPipeline_Execute(t *testing.T) {
	ctx := context.Background()
	state := NewState()
	p := New[*State]()

	step1 := new(MockStep)
	step1.On("Name").Return("step1").Maybe()
	step1.On("Execute", ctx, state).Return(nil)

	step2 := new(MockStep)
	step2.On("Name").Return("step2").Maybe()
	step2.On("Execute", ctx, state).Return(nil)

	hook := new(MockHook)
	hook.On("OnStepStart", ctx, step1, state).Return()
	hook.On("OnStepComplete", ctx, step1, state).Return()
	hook.On("OnStepStart", ctx, step2, state).Return()
	hook.On("OnStepComplete", ctx, step2, state).Return()

	p.AddStep(step1).AddStep(step2).AddHook(hook)

	err := p.Execute(ctx, state)
	assert.NoError(t, err)

	step1.AssertExpectations(t)
	step2.AssertExpectations(t)
	hook.AssertExpectations(t)
}

func TestPipeline_Error(t *testing.T) {
	ctx := context.Background()
	state := NewState()
	p := New[*State]()

	step1 := new(MockStep)
	errTest := errors.New("test error")
	step1.On("Name").Return("step1").Maybe()
	step1.On("Execute", ctx, state).Return(errTest)

	hook := new(MockHook)
	hook.On("OnStepStart", ctx, step1, state).Return()
	hook.On("OnStepError", ctx, step1, state, errTest).Return()

	p.AddStep(step1).AddHook(hook)

	err := p.Execute(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test error")

	step1.AssertExpectations(t)
	hook.AssertExpectations(t)
}

func TestPipeline_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	state := NewState()
	p := New[*State]()

	step1 := new(MockStep)
	step1.On("Name").Return("step1").Maybe()

	cancel() // Cancel before execution

	p.AddStep(step1)

	err := p.Execute(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline aborted")
}

// Ensure generic strong typing works
type MyTypedContext struct {
	Input  string
	Output string
}

type MyTypedStep struct{}

func (s *MyTypedStep) Name() string { return "typed_step" }
func (s *MyTypedStep) Execute(ctx context.Context, state *MyTypedContext) error {
	state.Output = state.Input + " processed"
	return nil
}

func TestPipeline_StronglyTyped(t *testing.T) {
	ctx := context.Background()
	state := &MyTypedContext{Input: "test"}

	p := New[*MyTypedContext]()
	p.AddStep(&MyTypedStep{})

	err := p.Execute(ctx, state)
	assert.NoError(t, err)
	assert.Equal(t, "test processed", state.Output)
}
