package core

import "encoding/json"

// Tool defines a function/tool that the LLM can call.
//
// Tools enable the model to interact with external systems, APIs, or perform
// computations. When you provide tools to a Chat call, the model can decide
// to call one or more tools instead of (or in addition to) generating text.
//
// Example:
//
//	tool := core.Tool{
//	    Name:        "get_weather",
//	    Description: "Get the current weather for a location",
//	    Parameters: json.RawMessage(`{
//	        "type": "object",
//	        "properties": {
//	            "location": {
//	                "type": "string",
//	                "description": "City name, e.g. San Francisco"
//	            },
//	            "unit": {
//	                "type": "string",
//	                "enum": ["celsius", "fahrenheit"]
//	            }
//	        },
//	        "required": ["location"]
//	    }`),
//	}
type Tool struct {
	// Name is the unique identifier for this tool.
	// Must be a valid function name (alphanumeric and underscores).
	Name string `json:"name"`

	// Description explains what the tool does.
	// The model uses this to decide when to call the tool.
	Description string `json:"description"`

	// Parameters is a JSON Schema describing the tool's input parameters.
	// Must be a valid JSON Schema object with "type": "object".
	Parameters json.RawMessage `json:"parameters"`
}

// ToolCall represents a request from the model to invoke a tool.
//
// When the model decides to use a tool, it returns a ToolCall in the response.
// Your application should:
//  1. Execute the tool with the provided arguments
//  2. Add the result as a new message with Role=RoleTool and ToolCallID set
//  3. Send the updated conversation back to the model
//
// Example:
//
//	if len(response.ToolCalls) > 0 {
//	    for _, tc := range response.ToolCalls {
//	        result := executeMyTool(tc.Name, tc.Arguments)
//	        messages = append(messages, core.Message{
//	            Role:       core.RoleTool,
//	            ToolCallID: tc.ID,
//	            Content:    []core.ContentBlock{{Type: core.ContentTypeText, Text: result}},
//	        })
//	    }
//	    // Call Chat again with the tool results
//	    finalResponse, _ := client.Chat(ctx, messages)
//	}
type ToolCall struct {
	// ID is a unique identifier for this tool call.
	// Use this when responding with tool results.
	ID string `json:"id"`

	// Name is the name of the tool being called.
	Name string `json:"name"`

	// Arguments is a JSON string containing the tool's input parameters.
	// Parse this to extract the actual arguments.
	Arguments string `json:"arguments"`
}
