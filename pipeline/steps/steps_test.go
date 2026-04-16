package steps

import (
	"context"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/gochat/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Chat(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Response, error) {
	args := m.Called(ctx, messages, opts)
	return args.Get(0).(*core.Response), args.Error(1)
}

func (m *MockClient) ChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	args := m.Called(ctx, messages, opts)
	return args.Get(0).(*core.Stream), args.Error(1)
}

func TestGenerateCompletionStep(t *testing.T) {
	ctx := context.Background()
	state := pipeline.NewState()
	state.Set("input", "hello")

	mockClient := new(MockClient)
	resp := &core.Response{
		Content: "world",
	}

	// Expectations
	mockClient.On("Chat", ctx, mock.Anything, mock.Anything).Return(resp, nil)

	step := NewGenerateCompletionStep(mockClient, "input", "output", "test-model").WithSystemPrompt("sys prompt")
	assert.Equal(t, "GenerateCompletion", step.Name())

	err := step.Execute(ctx, state)
	assert.NoError(t, err)
	outputStr, _ := state.GetString("output")
	assert.Equal(t, "world", outputStr)

	mockClient.AssertExpectations(t)
}

func TestTemplateStep(t *testing.T) {
	ctx := context.Background()
	state := pipeline.NewState()
	state.Set("name", "Gopher")

	step := NewTemplateStep("Hello {{.name}}!", "output", "name")
	assert.Equal(t, "RenderTemplate", step.Name())

	err := step.Execute(ctx, state)
	assert.NoError(t, err)
	outputStr, _ := state.GetString("output")
	assert.Equal(t, "Hello Gopher!", outputStr)
}
