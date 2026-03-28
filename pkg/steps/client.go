package steps

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/DotNetAge/gochat/pkg/pipeline"
)

type ChatStep struct {
	client    core.Client
	inputKey  string
	outputKey string
	opts      []core.Option
}

func NewChatStep(client core.Client, inputKey, outputKey string, opts ...core.Option) *ChatStep {
	return &ChatStep{
		client:    client,
		inputKey:  inputKey,
		outputKey: outputKey,
		opts:      opts,
	}
}

func (s *ChatStep) Name() string {
	return "Chat"
}

func (s *ChatStep) Execute(ctx context.Context, state *pipeline.State) error {
	prompt, ok := state.GetString(s.inputKey)
	if !ok || prompt == "" {
		return fmt.Errorf("missing or empty string at state key %q", s.inputKey)
	}

	messages := []core.Message{
		{
			Role:    core.RoleUser,
			Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: prompt}},
		},
	}

	resp, err := s.client.Chat(ctx, messages, s.opts...)
	if err != nil {
		return err
	}

	state.Set(s.outputKey, resp.Content)
	return nil
}

type ChatStreamStep struct {
	client    core.Client
	inputKey  string
	outputKey string
	opts      []core.Option
}

func NewChatStreamStep(client core.Client, inputKey, outputKey string, opts ...core.Option) *ChatStreamStep {
	return &ChatStreamStep{
		client:    client,
		inputKey:  inputKey,
		outputKey: outputKey,
		opts:      opts,
	}
}

func (s *ChatStreamStep) Name() string {
	return "ChatStream"
}

func (s *ChatStreamStep) Execute(ctx context.Context, state *pipeline.State) error {
	prompt, ok := state.GetString(s.inputKey)
	if !ok || prompt == "" {
		return fmt.Errorf("missing or empty string at state key %q", s.inputKey)
	}

	messages := []core.Message{
		{
			Role:    core.RoleUser,
			Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: prompt}},
		},
	}

	stream, err := s.client.ChatStream(ctx, messages, s.opts...)
	if err != nil {
		return err
	}

	var result string
	for stream.Next() {
		event := stream.Event()
		if event.Content != "" {
			result += event.Content
		}
	}

	if err := stream.Err(); err != nil {
		return err
	}

	state.Set(s.outputKey, result)
	return nil
}
