package models

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBGEProvider(t *testing.T) {
	// BGE small usually is 512, base is 768
	p, err := NewBGEProvider("bge-small-zh-v1.5.onnx")
	require.NoError(t, err)
	assert.Equal(t, 512, p.Dimension())
	
	p2, err := NewBGEProvider("bge-base-zh-v1.5.onnx")
	require.NoError(t, err)
	assert.Equal(t, 768, p2.Dimension())
}

func TestSentenceBERTProvider(t *testing.T) {
	p, err := NewSentenceBERTProvider("all-MiniLM-L6-v2.onnx")
	require.NoError(t, err)
	assert.Equal(t, 384, p.Dimension())
}

func TestGenericProvider(t *testing.T) {
	p, err := NewProvider("bert-base-uncased.onnx")
	require.NoError(t, err)
	assert.Equal(t, 768, p.Dimension())
	
	ctx := context.Background()
	// NewProvider uses NewCustomModel which returns mock embeddings
	embeddings, err := p.Embed(ctx, []string{"hello"})
	require.NoError(t, err)
	assert.Len(t, embeddings, 1)
	assert.Len(t, embeddings[0], 768)
}
