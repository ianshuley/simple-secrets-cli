/*
Copyright © 2025 Ian Shuley

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package errors provides structured error types for simple-secrets
package errors

import (
	"fmt"
)

// ErrorCode represents a structured error code for API responses
type ErrorCode string

const (
	// Authentication errors
	ErrCodeAuthRequired     ErrorCode = "AUTH_REQUIRED"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"

	// Validation errors
	ErrCodeInvalidInput    ErrorCode = "INVALID_INPUT"
	ErrCodeInvalidKey      ErrorCode = "INVALID_KEY"
	ErrCodeInvalidUsername ErrorCode = "INVALID_USERNAME"

	// Storage errors
	ErrCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists  ErrorCode = "ALREADY_EXISTS"
	ErrCodeStorageFailure ErrorCode = "STORAGE_FAILURE"

	// System errors
	ErrCodeFirstRunRequired ErrorCode = "FIRST_RUN_REQUIRED"
	ErrCodeConfigError      ErrorCode = "CONFIG_ERROR"
	ErrCodeCryptoError      ErrorCode = "CRYPTO_ERROR"
)

// StructuredError represents an error with structured information for API responses
type StructuredError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// HTTPStatus returns the appropriate HTTP status code for this error
func (e *StructuredError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeAuthRequired, ErrCodeInvalidToken:
		return 401 // Unauthorized
	case ErrCodePermissionDenied:
		return 403 // Forbidden
	case ErrCodeNotFound:
		return 404 // Not Found
	case ErrCodeAlreadyExists:
		return 409 // Conflict
	case ErrCodeInvalidInput, ErrCodeInvalidKey, ErrCodeInvalidUsername:
		return 400 // Bad Request
	case ErrCodeFirstRunRequired, ErrCodeConfigError:
		return 412 // Precondition Failed
	default:
		return 500 // Internal Server Error
	}
}

// New creates a new structured error
func New(code ErrorCode, message string, details ...string) *StructuredError {
	err := &StructuredError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// Wrap wraps an existing error with structured error information
func Wrap(err error, code ErrorCode, message string) *StructuredError {
	return &StructuredError{
		Code:    code,
		Message: message,
		Details: err.Error(),
	}
}
