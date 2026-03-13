package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockModelForTest is a mock model for testing
type MockModelForTest struct {
}

func (m *MockModelForTest) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *MockModelForTest) Close() error {
	return nil
}

func TestNewProvider(t *testing.T) {
	// Test with invalid configuration
	_, err := New(Config{
		Model:     nil,
		Dimension: 768,
	})
	assert.Error(t, err)

	_, err = New(Config{
		Model:     &MockModelForTest{},
		Dimension: 0,
	})
	assert.Error(t, err)

	// Test with valid configuration
	_, err = New(Config{
		Model:     &MockModelForTest{},
		Dimension: 768,
	})
	assert.NoError(t, err)
}

func TestTokenizer(t *testing.T) {
	tokenizer, err := NewTokenizer()
	assert.NoError(t, err)

	texts := []string{
		"Hello world",
		"This is a test",
	}

	inputIDs, attentionMask, err := tokenizer.TokenizeBatch(texts)
	assert.NoError(t, err)
	assert.Len(t, inputIDs, 2)
	assert.Len(t, attentionMask, 2)
	assert.Greater(t, len(inputIDs[0]), 0)
	assert.Greater(t, len(attentionMask[0]), 0)
}
