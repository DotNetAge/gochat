package steps

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/DotNetAge/gochat/pkg/pipeline"
)

// GenerateCompletionStep is a pipeline step that calls an LLM client to generate a completion.
type GenerateCompletionStep struct {
	client    core.Client
	inputKey  string
	outputKey string
	model     string
	sysPrompt string
}

// NewGenerateCompletionStep creates a new step to invoke an LLM.
// - client: The core.Client to use (e.g., openai.Client).
// - inputKey: The state key to read the user prompt from (expects string).
// - outputKey: The state key to write the string response to.
// - model: The model name to use.
func NewGenerateCompletionStep(client core.Client, inputKey, outputKey, model string) *GenerateCompletionStep {
	return &GenerateCompletionStep{
		client:    client,
		inputKey:  inputKey,
		outputKey: outputKey,
		model:     model,
	}
}

// WithSystemPrompt sets an optional system prompt for this step.
func (s *GenerateCompletionStep) WithSystemPrompt(sysPrompt string) *GenerateCompletionStep {
	s.sysPrompt = sysPrompt
	return s
}

// Name returns the step's name.
func (s *GenerateCompletionStep) Name() string {
	return "GenerateCompletion"
}

// Execute performs the LLM call.
func (s *GenerateCompletionStep) Execute(ctx context.Context, state *pipeline.State) error {
	// Read input
	prompt := state.GetString(s.inputKey)
	if prompt == "" {
		return fmt.Errorf("missing or empty string at state key %q", s.inputKey)
	}

	// Prepare messages
	var messages []core.Message
	if s.sysPrompt != "" {
		messages = append(messages, core.Message{
			Role:    core.RoleSystem,
			Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: s.sysPrompt}},
		})
	}

	messages = append(messages, core.Message{
		Role:    core.RoleUser,
		Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: prompt}},
	})

	// Call LLM
	resp, err := s.client.Chat(ctx, messages, core.WithModel(s.model))
	if err != nil {
		return err
	}

	// Read response
	state.Set(s.outputKey, resp.Content)

	return nil
}
