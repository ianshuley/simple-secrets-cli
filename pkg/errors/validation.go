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

// Validation-specific errors

// NewInvalidInputError creates an invalid input error
func NewInvalidInputError(field, reason string) *StructuredError {
	return New(ErrCodeInvalidInput, "Invalid input for field '"+field+"'", reason)
}

// NewInvalidKeyError creates an invalid key error
func NewInvalidKeyError(key, reason string) *StructuredError {
	return New(ErrCodeInvalidKey, "Invalid secret key '"+key+"'", reason)
}

// NewInvalidUsernameError creates an invalid username error
func NewInvalidUsernameError(username, reason string) *StructuredError {
	return New(ErrCodeInvalidUsername, "Invalid username '"+username+"'", reason)
}
