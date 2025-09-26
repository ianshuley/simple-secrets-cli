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

import "fmt"

// AuthError represents authentication/authorization failures
type AuthError struct {
Message string
Code    int // Future HTTP status code (401, 403)
}

func (e *AuthError) Error() string {
return fmt.Sprintf("auth error: %s", e.Message)
}

// ValidationError represents input validation failures
type ValidationError struct {
Field   string
Message string
Code    int // Future HTTP status code (400)
}

func (e *ValidationError) Error() string {
if e.Field != "" {
return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}
return fmt.Sprintf("validation error: %s", e.Message)
}

// NotFoundError represents resource not found
type NotFoundError struct {
Resource string
Key      string
Code     int // Future HTTP status code (404)
}

func (e *NotFoundError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.Key)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// StorageError represents storage backend failures
type StorageError struct {
Operation string
Message   string
Code      int // Future HTTP status code (500, 503)
}

func (e *StorageError) Error() string {
return fmt.Sprintf("storage error during %s: %s", e.Operation, e.Message)
}

// Helper functions for common cases
func NewAuthError(message string) *AuthError {
return &AuthError{Message: message, Code: 401}
}

func NewPermissionError(message string) *AuthError {
return &AuthError{Message: message, Code: 403}
}

func NewValidationError(field, message string) *ValidationError {
return &ValidationError{Field: field, Message: message, Code: 400}
}

func NewNotFoundError(resource, key string) *NotFoundError {
return &NotFoundError{Resource: resource, Key: key, Code: 404}
}

func NewStorageError(operation, message string) *StorageError {
return &StorageError{Operation: operation, Message: message, Code: 500}
}
