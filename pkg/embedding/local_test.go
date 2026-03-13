package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockModel struct {
	mock.Mock
}

func (m *MockModel) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	args := m.Called(inputs)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockModel) Close() error {
	return m.Called().Error(0)
}

func TestProvider_Embed(t *testing.T) {
	mockModel := new(MockModel)
	config := Config{
		Model:     mockModel,
		Dimension: 3,
	}

	p, err := New(config)
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test1", "test2"}

	// Mock model output
	mockModel.On("Run", mock.Anything).Return(map[string]interface{}{
		"last_hidden_state": [][]float32{
			{0.1, 0.2, 0.3},
			{0.4, 0.5, 0.6},
		},
	}, nil)

	embeddings, err := p.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	assert.Equal(t, float32(0.1), embeddings[0][0])

	mockModel.AssertExpectations(t)
}

func TestProvider_Embed_3DTensor(t *testing.T) {
	mockModel := new(MockModel)
	config := Config{
		Model:     mockModel,
		Dimension: 3,
	}

	p, err := New(config)
	require.NoError(t, err)

	// Mock 3D tensor output (batch, seq, hidden)
	mockModel.On("Run", mock.Anything).Return(map[string]interface{}{
		"last_hidden_state": [][][]float32{
			{{1, 2, 3}, {4, 5, 6}},
		},
	}, nil)

	ctx := context.Background()
	embeddings, err := p.Embed(ctx, []string{"test"})
	require.NoError(t, err)
	assert.Len(t, embeddings, 1)
	// Mean of {{1,2,3}, {4,5,6}} is {2.5, 3.5, 4.5}
	assert.Equal(t, float32(2.5), embeddings[0][0])
}
