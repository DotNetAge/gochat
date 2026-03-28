package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTool_Validate(t *testing.T) {
	t.Run("Valid tool", func(t *testing.T) {
		tool := Tool{
			Name:        "get_weather",
			Description: "Get the current weather",
			Parameters: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
		}
		err := tool.Validate()
		assert.NoError(t, err)
	})

	t.Run("Empty name", func(t *testing.T) {
		tool := Tool{
			Name:        "",
			Description: "Test tool",
			Parameters:  nil,
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("Invalid name starts with number", func(t *testing.T) {
		tool := Tool{
			Name:        "123tool",
			Description: "Test tool",
			Parameters:  nil,
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valid function name")
	})

	t.Run("Invalid name with special chars", func(t *testing.T) {
		tool := Tool{
			Name:        "tool-name",
			Description: "Test tool",
			Parameters:  nil,
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valid function name")
	})

	t.Run("Nil parameters", func(t *testing.T) {
		tool := Tool{
			Name:        "valid_tool",
			Description: "Test tool",
			Parameters:  nil,
		}
		err := tool.Validate()
		assert.NoError(t, err)
	})

	t.Run("Invalid JSON parameters", func(t *testing.T) {
		tool := Tool{
			Name:        "valid_tool",
			Description: "Test tool",
			Parameters:  json.RawMessage(`invalid json`),
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "valid JSON")
	})

	t.Run("Parameters missing type field", func(t *testing.T) {
		tool := Tool{
			Name:        "valid_tool",
			Description: "Test tool",
			Parameters:  json.RawMessage(`{"properties":{}}`),
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "'type' field")
	})

	t.Run("Parameters wrong type", func(t *testing.T) {
		tool := Tool{
			Name:        "valid_tool",
			Description: "Test tool",
			Parameters:  json.RawMessage(`{"type":"string"}`),
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type 'object'")
	})

	t.Run("Invalid property name", func(t *testing.T) {
		tool := Tool{
			Name:        "valid_tool",
			Description: "Test tool",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"123invalid": {"type": "string"}
				}
			}`),
		}
		err := tool.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid identifier")
	})
}

func TestValidateTools(t *testing.T) {
	t.Run("Valid tools", func(t *testing.T) {
		tools := []Tool{
			{
				Name:        "tool_one",
				Description: "First tool",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
			{
				Name:        "tool_two",
				Description: "Second tool",
				Parameters:  json.RawMessage(`{"type":"object"}`),
			},
		}
		err := ValidateTools(tools)
		assert.NoError(t, err)
	})

	t.Run("Duplicate tool names", func(t *testing.T) {
		tools := []Tool{
			{
				Name:        "duplicate",
				Description: "First tool",
				Parameters:  nil,
			},
			{
				Name:        "duplicate",
				Description: "Second tool",
				Parameters:  nil,
			},
		}
		err := ValidateTools(tools)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate")
	})

	t.Run("Invalid tool in list", func(t *testing.T) {
		tools := []Tool{
			{
				Name:        "valid_tool",
				Description: "Valid tool",
				Parameters:  nil,
			},
			{
				Name:        "",
				Description: "Invalid tool",
				Parameters:  nil,
			},
		}
		err := ValidateTools(tools)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid tool")
	})
}
