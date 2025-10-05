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

// Authentication-specific errors

// NewAuthRequiredError creates an authentication required error
func NewAuthRequiredError(message string) *StructuredError {
	return New(ErrCodeAuthRequired, message)
}

// NewInvalidTokenError creates an invalid token error
func NewInvalidTokenError(details string) *StructuredError {
	return New(ErrCodeInvalidToken, "Invalid authentication token", details)
}

// NewPermissionDeniedError creates a permission denied error
func NewPermissionDeniedError(operation string) *StructuredError {
	return New(ErrCodePermissionDenied, "Permission denied", "Operation '"+operation+"' requires higher privileges")
}
