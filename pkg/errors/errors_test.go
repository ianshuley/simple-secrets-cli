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

package errors

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestStructuredError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *StructuredError
		want string
	}{
		{
			name: "without details",
			err:  &StructuredError{Code: ErrCodeNotFound, Message: "Resource not found"},
			want: "NOT_FOUND: Resource not found",
		},
		{
			name: "with details",
			err:  &StructuredError{Code: ErrCodeInvalidToken, Message: "Invalid token", Details: "Token expired"},
			want: "INVALID_TOKEN: Invalid token (Token expired)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("StructuredError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStructuredError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name string
		code ErrorCode
		want int
	}{
		{"auth required", ErrCodeAuthRequired, 401},
		{"invalid token", ErrCodeInvalidToken, 401},
		{"permission denied", ErrCodePermissionDenied, 403},
		{"not found", ErrCodeNotFound, 404},
		{"already exists", ErrCodeAlreadyExists, 409},
		{"invalid input", ErrCodeInvalidInput, 400},
		{"invalid key", ErrCodeInvalidKey, 400},
		{"invalid username", ErrCodeInvalidUsername, 400},
		{"first run required", ErrCodeFirstRunRequired, 412},
		{"config error", ErrCodeConfigError, 412},
		{"unknown error", ErrorCode("UNKNOWN"), 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &StructuredError{Code: tt.code}
			if got := err.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		code    ErrorCode
		message string
		details []string
		want    *StructuredError
	}{
		{
			name:    "without details",
			code:    ErrCodeNotFound,
			message: "Resource not found",
			want:    &StructuredError{Code: ErrCodeNotFound, Message: "Resource not found"},
		},
		{
			name:    "with details",
			code:    ErrCodeInvalidToken,
			message: "Invalid token",
			details: []string{"Token expired"},
			want:    &StructuredError{Code: ErrCodeInvalidToken, Message: "Invalid token", Details: "Token expired"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.code, tt.message, tt.details...)
			if got.Code != tt.want.Code || got.Message != tt.want.Message || got.Details != tt.want.Details {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrapped := Wrap(originalErr, ErrCodeStorageFailure, "Storage failed")

	if wrapped.Code != ErrCodeStorageFailure {
		t.Errorf("Wrap() code = %v, want %v", wrapped.Code, ErrCodeStorageFailure)
	}

	if wrapped.Message != "Storage failed" {
		t.Errorf("Wrap() message = %v, want %v", wrapped.Message, "Storage failed")
	}

	if wrapped.Details != "original error" {
		t.Errorf("Wrap() details = %v, want %v", wrapped.Details, "original error")
	}
}

func TestStructuredError_JSON(t *testing.T) {
	err := New(ErrCodeInvalidToken, "Invalid token", "Token expired")

	jsonData, jsonErr := json.Marshal(err)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal to JSON: %v", jsonErr)
	}

	var unmarshaled StructuredError
	if unmarshalErr := json.Unmarshal(jsonData, &unmarshaled); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal from JSON: %v", unmarshalErr)
	}

	if unmarshaled.Code != err.Code {
		t.Errorf("JSON roundtrip code = %v, want %v", unmarshaled.Code, err.Code)
	}

	if unmarshaled.Message != err.Message {
		t.Errorf("JSON roundtrip message = %v, want %v", unmarshaled.Message, err.Message)
	}

	if unmarshaled.Details != err.Details {
		t.Errorf("JSON roundtrip details = %v, want %v", unmarshaled.Details, err.Details)
	}
}

func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name     string
		errorFn  func() *StructuredError
		wantCode ErrorCode
	}{
		{
			name:     "auth required",
			errorFn:  func() *StructuredError { return NewAuthRequiredError("Auth needed") },
			wantCode: ErrCodeAuthRequired,
		},
		{
			name:     "invalid token",
			errorFn:  func() *StructuredError { return NewInvalidTokenError("Bad token") },
			wantCode: ErrCodeInvalidToken,
		},
		{
			name:     "permission denied",
			errorFn:  func() *StructuredError { return NewPermissionDeniedError("admin_operation") },
			wantCode: ErrCodePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			if err.Code != tt.wantCode {
				t.Errorf("Error code = %v, want %v", err.Code, tt.wantCode)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		errorFn  func() *StructuredError
		wantCode ErrorCode
	}{
		{
			name:     "invalid input",
			errorFn:  func() *StructuredError { return NewInvalidInputError("username", "too short") },
			wantCode: ErrCodeInvalidInput,
		},
		{
			name:     "invalid key",
			errorFn:  func() *StructuredError { return NewInvalidKeyError("secret_key", "contains invalid chars") },
			wantCode: ErrCodeInvalidKey,
		},
		{
			name:     "invalid username",
			errorFn:  func() *StructuredError { return NewInvalidUsernameError("user@", "invalid format") },
			wantCode: ErrCodeInvalidUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			if err.Code != tt.wantCode {
				t.Errorf("Error code = %v, want %v", err.Code, tt.wantCode)
			}
		})
	}
}

func TestStorageErrors(t *testing.T) {
	tests := []struct {
		name     string
		errorFn  func() *StructuredError
		wantCode ErrorCode
	}{
		{
			name:     "not found",
			errorFn:  func() *StructuredError { return NewNotFoundError("secret", "test_key") },
			wantCode: ErrCodeNotFound,
		},
		{
			name:     "already exists",
			errorFn:  func() *StructuredError { return NewAlreadyExistsError("user", "admin") },
			wantCode: ErrCodeAlreadyExists,
		},
		{
			name:     "storage failure",
			errorFn:  func() *StructuredError { return NewStorageFailureError("write", errors.New("disk full")) },
			wantCode: ErrCodeStorageFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			if err.Code != tt.wantCode {
				t.Errorf("Error code = %v, want %v", err.Code, tt.wantCode)
			}
		})
	}
}
