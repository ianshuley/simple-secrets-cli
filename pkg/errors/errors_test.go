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
	"testing"
)

func TestAuthError(t *testing.T) {
	err := NewAuthError("invalid token")
	
	expected := "auth error: invalid token"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
	
	if err.Code != 401 {
		t.Errorf("Expected code 401, got %d", err.Code)
	}
}

func TestPermissionError(t *testing.T) {
	err := NewPermissionError("need write permission")
	
	expected := "auth error: need write permission"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
	
	if err.Code != 403 {
		t.Errorf("Expected code 403, got %d", err.Code)
	}
}

func TestValidationError(t *testing.T) {
	// Test with field
	err := NewValidationError("username", "must not be empty")
	
	expected := "validation error on username: must not be empty"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
	
	// Test without field
	err2 := NewValidationError("", "invalid format")
	expected2 := "validation error: invalid format"
	if err2.Error() != expected2 {
		t.Errorf("Expected %q, got %q", expected2, err2.Error())
	}
	
	if err.Code != 400 {
		t.Errorf("Expected code 400, got %d", err.Code)
	}
}

func TestNotFoundError(t *testing.T) {
	// Test with key
	err := NewNotFoundError("secret", "test-key")
	
	expected := "secret not found: test-key"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
	
	// Test without key
	err2 := NewNotFoundError("secret", "")
	expected2 := "secret not found"
	if err2.Error() != expected2 {
		t.Errorf("Expected %q, got %q", expected2, err2.Error())
	}
	
	if err.Code != 404 {
		t.Errorf("Expected code 404, got %d", err.Code)
	}
}

func TestStorageError(t *testing.T) {
	err := NewStorageError("save", "disk full")
	
	expected := "storage error during save: disk full"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
	
	if err.Code != 500 {
		t.Errorf("Expected code 500, got %d", err.Code)
	}
}