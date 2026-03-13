package steps

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/embedding"
	"github.com/DotNetAge/gochat/pkg/pipeline"
)

type EmbedStep struct {
	provider  embedding.Provider
	inputKey  string
	outputKey string
	batchSize int
}

func NewEmbedStep(provider embedding.Provider, inputKey, outputKey string, batchSize int) *EmbedStep {
	return &EmbedStep{
		provider:  provider,
		inputKey:  inputKey,
		outputKey: outputKey,
		batchSize: batchSize,
	}
}

func (s *EmbedStep) Name() string {
	return "Embed"
}

func (s *EmbedStep) Execute(ctx context.Context, state *pipeline.State) error {
	rawInput, ok := state.Get(s.inputKey)
	if !ok || rawInput == nil {
		return fmt.Errorf("missing value at state key %q", s.inputKey)
	}

	var texts []string
	switch v := rawInput.(type) {
	case string:
		texts = []string{v}
	case []string:
		texts = v
	case []interface{}:
		texts = make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string at state key %q, got %T", s.inputKey, item)
			}
			texts[i] = str
		}
	default:
		return fmt.Errorf("unsupported input type %T at state key %q", rawInput, s.inputKey)
	}

	var embeddings [][]float32
	var err error

	if s.batchSize > 0 && len(texts) > s.batchSize {
		bp := embedding.NewBatchProcessor(s.provider, embedding.BatchOptions{
			MaxBatchSize: s.batchSize,
		})
		embeddings, err = bp.Process(ctx, texts)
	} else {
		embeddings, err = s.provider.Embed(ctx, texts)
	}

	if err != nil {
		return err
	}

	state.Set(s.outputKey, embeddings)
	return nil
}

type EmbedSingleStep struct {
	provider  embedding.Provider
	inputKey  string
	outputKey string
}

func NewEmbedSingleStep(provider embedding.Provider, inputKey, outputKey string) *EmbedSingleStep {
	return &EmbedSingleStep{
		provider:  provider,
		inputKey:  inputKey,
		outputKey: outputKey,
	}
}

func (s *EmbedSingleStep) Name() string {
	return "EmbedSingle"
}

func (s *EmbedSingleStep) Execute(ctx context.Context, state *pipeline.State) error {
	text := state.GetString(s.inputKey)
	if text == "" {
		return fmt.Errorf("missing or empty string at state key %q", s.inputKey)
	}

	embeddings, err := s.provider.Embed(ctx, []string{text})
	if err != nil {
		return err
	}

	if len(embeddings) == 0 {
		return fmt.Errorf("no embedding returned for text")
	}

	state.Set(s.outputKey, embeddings[0])
	return nil
}
