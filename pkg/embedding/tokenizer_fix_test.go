package embedding

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenizer_Constants(t *testing.T) {
	assert.Equal(t, 101, CLSTokenID)
	assert.Equal(t, 102, SEITokenID)
	assert.Equal(t, 0, PadTokenID)
	assert.Equal(t, 10000, VocabStart)
}

func TestTokenizer_VocabSize(t *testing.T) {
	tokenizer, err := NewTokenizer()
	assert.NoError(t, err)

	initialSize := tokenizer.VocabSize()
	assert.Equal(t, 0, initialSize)

	texts := []string{"hello world", "foo bar"}
	_, _, err = tokenizer.TokenizeBatch(texts)
	assert.NoError(t, err)

	newSize := tokenizer.VocabSize()
	assert.Greater(t, newSize, initialSize)
}

func TestTokenizer_ConcurrentAccess(t *testing.T) {
	tokenizer, err := NewTokenizer()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	errors := make([]error, 0, 10)
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			texts := []string{"concurrent tokenization", "test text"}
			_, _, err := tokenizer.TokenizeBatch(texts)
			mu.Lock()
			errors = append(errors, err)
			mu.Unlock()
		}()
	}

	wg.Wait()

	for _, err := range errors {
		assert.NoError(t, err)
	}
}

func TestTokenizer_ConcurrentSameTexts(t *testing.T) {
	tokenizer, err := NewTokenizer()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	results := make([][][]int64, 0, 10)
	var mu sync.Mutex

	sameTexts := []string{"same text for concurrent tokenization"}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			inputIDs, _, err := tokenizer.TokenizeBatch(sameTexts)
			if err == nil {
				mu.Lock()
				results = append(results, inputIDs)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Len(t, results, 10)
	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0][0], results[i][0])
	}
}

func TestTokenizer_Padding(t *testing.T) {
	tokenizer, err := NewTokenizer()
	assert.NoError(t, err)

	texts := []string{"a", "longer text here", "short"}

	inputIDs, attentionMask, err := tokenizer.TokenizeBatch(texts)
	assert.NoError(t, err)
	assert.Len(t, inputIDs, 3)
	assert.Len(t, attentionMask, 3)

	maxLen := len(inputIDs[0])
	for i, ids := range inputIDs {
		assert.Len(t, ids, maxLen, "Text %d should have padding to max length", i)
		assert.Equal(t, int64(CLSTokenID), ids[0], "First token should be CLS")
		hasSEP := false
		for _, id := range ids {
			if id == int64(SEITokenID) {
				hasSEP = true
				break
			}
		}
		assert.True(t, hasSEP, "Should contain SEP token")
		assert.Equal(t, int64(1), attentionMask[i][0], "First attention mask should be 1")
	}
}
