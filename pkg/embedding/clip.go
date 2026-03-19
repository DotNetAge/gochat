package embedding

import (
	"context"
	"fmt"
)

// CLIPProvider implements MultimodalProvider for CLIP models.
type CLIPProvider struct {
	textModel   Provider        // Embeds text
	visionModel EmbeddingModel  // Raw model for vision
	processor   *ImageProcessor
	dimension   int
}

// NewCLIPProvider creates a new CLIP provider.
func NewCLIPProvider(modelPath string) (*CLIPProvider, error) {
	info, err := NewModelInfo(modelPath)
	if err != nil {
		return nil, err
	}

	// For a real CLIP implementation, we load text_model.onnx and vision_model.onnx.
	// Since NewModel abstracts the ONNX loading, we instantiate two models here.
	tModel, err := NewModel(info.Dimension, modelPath) // text
	if err != nil {
		return nil, err
	}
	vModel, err := NewModel(info.Dimension, modelPath) // vision
	if err != nil {
		return nil, err
	}

	localTextProvider, err := New(Config{
		Model:        tModel,
		Dimension:    info.Dimension,
		MaxBatchSize: 32,
	})
	if err != nil {
		return nil, err
	}

	return &CLIPProvider{
		textModel:   localTextProvider,
		visionModel: vModel,
		processor:   NewImageProcessor(),
		dimension:   info.Dimension,
	}, nil
}

// Embed generates text embeddings using the CLIP text encoder.
func (p *CLIPProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.textModel.Embed(ctx, texts)
}

// Dimension returns the embedding dimension (e.g., 512 for ViT-B/32).
func (p *CLIPProvider) Dimension() int {
	return p.dimension
}

// EmbedImages generates image embeddings using the CLIP vision encoder.
func (p *CLIPProvider) EmbedImages(ctx context.Context, images [][]byte) ([][]float32, error) {
	if len(images) == 0 {
		return [][]float32{}, nil
	}

	pixelValues, err := p.processor.ProcessBatch(images)
	if err != nil {
		return nil, fmt.Errorf("image processing failed: %w", err)
	}

	// Run vision model
	inputs := map[string]interface{}{
		"pixel_values": pixelValues,
	}

	outputs, err := p.visionModel.Run(inputs)
	if err != nil {
		return nil, fmt.Errorf("vision model inference failed: %w", err)
	}

	// Extract embeddings
	rawOutput, ok := outputs["image_embeds"]
	if !ok {
		// Fallback to last_hidden_state for ONNX models that output raw hidden states
		rawOutput, ok = outputs["last_hidden_state"]
		if !ok {
			return nil, fmt.Errorf("vision model output missing image_embeds or last_hidden_state")
		}
	}

	switch v := rawOutput.(type) {
	case [][]float32:
		return v, nil
	case [][][]float32:
		// Extract CLS token if output is 3D (Batch, Sequence, Features)
		batchSize := len(v)
		embeddings := make([][]float32, batchSize)
		for i := 0; i < batchSize; i++ {
			if len(v[i]) > 0 {
				embeddings[i] = v[i][0] // Take first token (usually CLS)
			} else {
				embeddings[i] = make([]float32, p.dimension)
			}
		}
		return embeddings, nil
	default:
		return nil, fmt.Errorf("unexpected vision output type: %T", rawOutput)
	}
}
