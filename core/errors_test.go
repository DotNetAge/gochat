package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_Error(t *testing.T) {
	// 测试带 cause 的错误
	cause := errors.New("underlying error")
	err := NewError(ErrorTypeAPI, "API error", cause)

	expected := "api_error: API error (cause: underlying error)"
	assert.Equal(t, expected, err.Error())

	// 测试不带 cause 的错误
	err2 := NewError(ErrorTypeValidation, "Validation error", nil)
	expected2 := "validation_error: Validation error"
	assert.Equal(t, expected2, err2.Error())
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError("API error message", assert.AnError)
	assert.Equal(t, ErrorTypeAPI, err.Type)
	assert.Equal(t, "API error message", err.Message)
	assert.Equal(t, assert.AnError, err.Cause)
}

func TestNewNetworkError(t *testing.T) {
	err := NewNetworkError("Network error message", assert.AnError)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "Network error message", err.Message)
	assert.Equal(t, assert.AnError, err.Cause)
}

func TestNewTimeoutError(t *testing.T) {
	err := NewTimeoutError("Timeout error message", assert.AnError)
	assert.Equal(t, ErrorTypeTimeout, err.Type)
	assert.Equal(t, "Timeout error message", err.Message)
	assert.Equal(t, assert.AnError, err.Cause)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("Validation error message", assert.AnError)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "Validation error message", err.Message)
	assert.Equal(t, assert.AnError, err.Cause)
}

func TestNewUnknownError(t *testing.T) {
	err := NewUnknownError("Unknown error message", assert.AnError)
	assert.Equal(t, ErrorTypeUnknown, err.Type)
	assert.Equal(t, "Unknown error message", err.Message)
	assert.Equal(t, assert.AnError, err.Cause)
}
