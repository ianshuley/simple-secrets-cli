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

// Storage-specific errors

// NewNotFoundError creates a not found error
func NewNotFoundError(resource, identifier string) *StructuredError {
	return New(ErrCodeNotFound, resource+" not found", "No "+resource+" found with identifier: "+identifier)
}

// NewAlreadyExistsError creates an already exists error
func NewAlreadyExistsError(resource, identifier string) *StructuredError {
	return New(ErrCodeAlreadyExists, resource+" already exists", resource+" with identifier '"+identifier+"' already exists")
}

// NewStorageFailureError creates a storage failure error
func NewStorageFailureError(operation string, cause error) *StructuredError {
	return Wrap(cause, ErrCodeStorageFailure, "Storage operation '"+operation+"' failed")
}
