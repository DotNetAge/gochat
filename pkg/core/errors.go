package core

import "fmt"

// ErrorType defines the type of error
type ErrorType string

const (
	ErrorTypeAPI        ErrorType = "api_error"
	ErrorTypeNetwork    ErrorType = "network_error"
	ErrorTypeTimeout    ErrorType = "timeout_error"
	ErrorTypeValidation ErrorType = "validation_error"
	ErrorTypeUnknown    ErrorType = "unknown_error"
)

// Error represents a structured error
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewError creates a new structured error
func NewError(errType ErrorType, message string, cause error) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Cause:   cause,
	}
}

// NewAPIError creates a new API error
func NewAPIError(message string, cause error) *Error {
	return NewError(ErrorTypeAPI, message, cause)
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *Error {
	return NewError(ErrorTypeNetwork, message, cause)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string, cause error) *Error {
	return NewError(ErrorTypeTimeout, message, cause)
}

// NewValidationError creates a new validation error
func NewValidationError(message string, cause error) *Error {
	return NewError(ErrorTypeValidation, message, cause)
}

// NewUnknownError creates a new unknown error
func NewUnknownError(message string, cause error) *Error {
	return NewError(ErrorTypeUnknown, message, cause)
}
