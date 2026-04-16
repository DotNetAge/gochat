package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionFunctions(t *testing.T) {
	// Test WithTemperature
	opts := []Option{WithTemperature(0.8)}
	assert.Len(t, opts, 1)

	// Test WithMaxTokens
	opts = []Option{WithMaxTokens(1000)}
	assert.Len(t, opts, 1)

	// Test WithTopP
	opts = []Option{WithTopP(0.9)}
	assert.Len(t, opts, 1)

	// Test WithStop
	opts = []Option{WithStop("stop1", "stop2")}
	assert.Len(t, opts, 1)

	// Test WithModel
	opts = []Option{WithModel("gpt-4")}
	assert.Len(t, opts, 1)

	// Test WithTools
	tool := Tool{Name: "calculator", Description: "Perform calculations"}
	opts = []Option{WithTools(tool)}
	assert.Len(t, opts, 1)

	// Test WithSystemPrompt
	opts = []Option{WithSystemPrompt("You are a helpful assistant")}
	assert.Len(t, opts, 1)

	// Test WithThinking
	opts = []Option{WithThinking(100)}
	assert.Len(t, opts, 1)

	// Test WithEnableSearch
	opts = []Option{WithEnableSearch(true)}
	assert.Len(t, opts, 1)

	// Test WithUsageCallback
	opts = []Option{WithUsageCallback(func(Usage) {})}
	assert.Len(t, opts, 1)
}

func TestApplyOptions(t *testing.T) {
	// Test applying multiple options
	opts := ApplyOptions(
		WithModel("gpt-4"),
		WithTemperature(0.7),
		WithMaxTokens(500),
		WithTopP(0.95),
		WithStop("stop1"),
		WithSystemPrompt("You are a helpful assistant"),
		WithThinking(100),
		WithEnableSearch(true),
	)

	assert.Equal(t, "gpt-4", opts.Model)
	assert.NotNil(t, opts.Temperature)
	assert.Equal(t, 0.7, *opts.Temperature)
	assert.NotNil(t, opts.MaxTokens)
	assert.Equal(t, 500, *opts.MaxTokens)
	assert.NotNil(t, opts.TopP)
	assert.Equal(t, 0.95, *opts.TopP)
	assert.Len(t, opts.Stop, 1)
	assert.Equal(t, "stop1", opts.Stop[0])
	assert.Equal(t, "You are a helpful assistant", opts.SystemPrompt)
	assert.True(t, opts.Thinking)
	assert.Equal(t, 100, opts.ThinkingBudget)
	assert.True(t, opts.EnableSearch)
}

func TestToolStructure(t *testing.T) {
	// Test Tool structure
	tool := Tool{
		Name:        "calculator",
		Description: "Perform mathematical calculations",
		Parameters: []byte(`{
			"type": "object",
			"properties": {
				"expression": {
					"type": "string"
				}
			}
		}`),
	}

	assert.Equal(t, "calculator", tool.Name)
	assert.Equal(t, "Perform mathematical calculations", tool.Description)
	assert.NotNil(t, tool.Parameters)
}
