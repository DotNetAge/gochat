package core

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// toolNameRegex validates that tool and parameter names are valid Go/JSON identifiers.
// A valid name starts with a letter or underscore, followed by letters, digits, or underscores.
var toolNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Tool represents a callable function that an LLM can invoke during generation.
// Tools extend the model's capabilities by allowing it to perform actions
// like searching the web, executing code, or accessing external APIs.
//
// A Tool consists of a name, description, and JSON schema that defines
// the parameters the tool accepts. The model uses the description to
// determine when to call the tool, and the schema to generate valid arguments.
//
// Example:
//
//	tool := core.Tool{
//	    Name:        "get_weather",
//	    Description: "Get the current weather for a location",
//	    Parameters: json.RawMessage(`{
//	        "type": "object",
//	        "properties": {
//	            "location": {"type": "string"}
//	        }
//	    }`),
//	}
type Tool struct {
	// Name is a unique identifier for this tool.
	// Must be a valid identifier: starts with letter/underscore,
	// contains only letters, digits, and underscores.
	Name string `json:"name"`

	// Description explains what the tool does and when to use it.
	// This is used by the model to decide whether to call the tool.
	Description string `json:"description"`

	// Parameters is a JSON Schema defining the tool's input parameters.
	// Must be a valid JSON object with "type": "object".
	// Each property in "properties" defines one parameter.
	Parameters json.RawMessage `json:"parameters"`
}

// ToolCall represents a request from the model to invoke a tool.
// This is returned when the model decides to call a tool during generation.
// It contains the tool name and the arguments to pass to the tool.
type ToolCall struct {
	// ID uniquely identifies this tool call request.
	// Use this when sending the tool result back to the model.
	ID string `json:"id"`

	// Name is the name of the tool to call.
	// Must match the Name field of a Tool provided in the request.
	Name string `json:"name"`

	// Arguments is a JSON string containing the parameters to pass to the tool.
	// Parse this with json.Unmarshal to get a map or struct.
	Arguments string `json:"arguments"`
}

// Validate checks if the tool is properly configured and can be used in requests.
// Validation ensures the tool name is valid and the parameters schema is well-formed.
//
// Returns nil if the tool is valid, or an error describing the validation failure.
// Common validation errors include:
//   - Empty tool name
//   - Invalid tool name (starts with number, contains special characters)
//   - Invalid JSON in Parameters
//   - Parameters missing required "type" field
//   - Parameters with type other than "object"
//   - Invalid property names in the schema
func (t *Tool) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}

	if !toolNameRegex.MatchString(t.Name) {
		return fmt.Errorf("tool name must be a valid function name (alphanumeric and underscores, cannot start with a number)")
	}

	if t.Parameters == nil {
		return nil
	}

	var params map[string]interface{}
	if err := json.Unmarshal(t.Parameters, &params); err != nil {
		return fmt.Errorf("tool parameters must be valid JSON: %w", err)
	}

	paramType, ok := params["type"]
	if !ok {
		return fmt.Errorf("tool parameters must have a 'type' field")
	}

	typeStr, ok := paramType.(string)
	if !ok || typeStr != "object" {
		return fmt.Errorf("tool parameters must be of type 'object'")
	}

	if props, ok := params["properties"].(map[string]interface{}); ok {
		for propName := range props {
			if !toolNameRegex.MatchString(propName) {
				return fmt.Errorf("tool parameter property name '%s' is not a valid identifier", propName)
			}
		}
	}

	return nil
}

// ValidateTools validates a list of tools for use in a request.
// This is a convenience function that validates each tool and ensures
// there are no duplicate names.
//
// Use this before sending tools to the API to catch configuration errors early.
//
// Parameters:
//   - tools: Slice of Tool objects to validate
//
// Returns nil if all tools are valid, or an error describing the first failure
func ValidateTools(tools []Tool) error {
	seen := make(map[string]bool)
	for _, tool := range tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("invalid tool '%s': %w", tool.Name, err)
		}
		if seen[tool.Name] {
			return fmt.Errorf("duplicate tool name: %s", tool.Name)
		}
		seen[tool.Name] = true
	}
	return nil
}
