package core

import (
	"encoding/json"
	"fmt"
)

// ErrorType defines the classification of errors that can occur
// during API operations. Use these types to programmatically handle
// specific error conditions.
type ErrorType string

// Error classification constants for structured error handling.
// These values are used by the Error struct to categorize failures.
const (
	// ErrorTypeAPI indicates an error returned by the AI provider's API.
	// This includes invalid requests, authentication failures, or rate limits.
	ErrorTypeAPI ErrorType = "api_error"

	// ErrorTypeNetwork indicates a network connectivity issue.
	// This includes connection timeouts, DNS failures, or refused connections.
	ErrorTypeNetwork ErrorType = "network_error"

	// ErrorTypeTimeout indicates an operation exceeded its time limit.
	// This occurs when a request takes longer than the configured timeout.
	ErrorTypeTimeout ErrorType = "timeout_error"

	// ErrorTypeValidation indicates invalid input parameters.
	// This occurs when request data fails validation checks.
	ErrorTypeValidation ErrorType = "validation_error"

	// ErrorTypeUnknown indicates an unexpected error without classification.
	// This is used as a fallback for un categorized failures.
	ErrorTypeUnknown ErrorType = "unknown_error"
)

// Error represents a structured error with type classification,
// a human-readable message, and an optional underlying cause.
// This design enables both programmatic error handling and
// human-readable error reporting.
//
// Example:
//
//	if err != nil {
//	    if apiErr, ok := err.(*core.Error); ok {
//	        if apiErr.Type == core.ErrorTypeRateLimit {
//	            // handle rate limit specifically
//	        }
//	    }
//	}
type Error struct {
	// Type categorizes the error for programmatic handling.
	Type ErrorType

	// Message is a human-readable description of what went wrong.
	Message string

	// Cause is the underlying error that triggered this one, if available.
	// This may be nil if the error originated without a specific cause.
	Cause error
}

// Error implements the error interface and returns a formatted string
// representation of the error, including the error type, message, and
// optionally the underlying cause if present.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause of the error, allowing it to work
// with errors.Is and errors.As from the standard errors package.
func (e *Error) Unwrap() error {
	return e.Cause
}

// NewError creates a new structured error with the specified type,
// message, and optional cause. This is the low-level constructor for
// creating Error instances; most callers should use one of the
// specialized constructors like NewAPIError or NewNetworkError.
//
// Parameters:
//   - errType: The classification of the error
//   - message: A human-readable description of the error
//   - cause: The underlying error that triggered this one, or nil
//
// Returns a pointer to the newly created Error
func NewError(errType ErrorType, message string, cause error) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// NewAPIError creates a new error for API-level failures such as
// invalid requests, authentication errors, or provider-side issues.
//
// Parameters:
//   - message: A description of the API error
//   - cause: The underlying error, or nil if not applicable
//
// Returns an Error with Type set to ErrorTypeAPI
func NewAPIError(message string, cause error) *Error {
	return NewError(ErrorTypeAPI, message, cause)
}

// NewNetworkError creates a new error for network-related failures
// such as connection refused, DNS resolution failures, or connectivity
// issues that prevent communication with the API server.
//
// Parameters:
//   - message: A description of the network error
//   - cause: The underlying error, or nil if not applicable
//
// Returns an Error with Type set to ErrorTypeNetwork
func NewNetworkError(message string, cause error) *Error {
	return NewError(ErrorTypeNetwork, message, cause)
}

// NewTimeoutError creates a new error for operations that exceed
// their configured time limit. This includes both connect timeouts
// and read/write timeouts.
//
// Parameters:
//   - message: A description of what timed out
//   - cause: The underlying error, or nil if not applicable
//
// Returns an Error with Type set to ErrorTypeTimeout
func NewTimeoutError(message string, cause error) *Error {
	return NewError(ErrorTypeTimeout, message, cause)
}

// NewValidationError creates a new error for input validation
// failures. This includes invalid request parameters, malformed
// data, or values that fail business logic validation.
//
// Parameters:
//   - message: A description of the validation failure
//   - cause: The underlying error, or nil if not applicable
//
// Returns an Error with Type set to ErrorTypeValidation
func NewValidationError(message string, cause error) *Error {
	return NewError(ErrorTypeValidation, message, cause)
}

// NewUnknownError creates a new error for unexpected failures that
// don't fit into other categories. Use this as a fallback when
// the error source doesn't provide more specific information.
//
// Parameters:
//   - message: A description of the unexpected error
//   - cause: The underlying error, or nil if not applicable
//
// Returns an Error with Type set to ErrorTypeUnknown
func NewUnknownError(message string, cause error) *Error {
	return NewError(ErrorTypeUnknown, message, cause)
}

// NewAPIErrorFromResponse creates a new API error from HTTP response status and body.
// This is a convenience method for creating standardized error responses.
// It attempts to parse JSON error responses common to AI providers.
func NewAPIErrorFromResponse(statusCode int, body []byte) *Error {
	var errData struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"` // fallback for some providers
		Code    string `json:"code"`    // fallback for some providers
	}

	message := string(body)
	if err := json.Unmarshal(body, &errData); err == nil {
		if errData.Error.Message != "" {
			message = errData.Error.Message
		} else if errData.Message != "" {
			message = errData.Message
		}
	}

	return NewAPIError(fmt.Sprintf("request failed with status %d: %s", statusCode, message), nil)
}
