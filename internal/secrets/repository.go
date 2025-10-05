/*package secrets

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

package secrets

import (
	"context"
	"os"
)

// FileMode represents file permissions for secrets storage operations
type FileMode = os.FileMode

// Repository defines the storage operations needed by the secrets domain
// All operations are context-aware to support cancellation and timeouts
type Repository interface {
	// ReadFile reads the entire content of a file
	ReadFile(ctx context.Context, path string) ([]byte, error)

	// WriteFile writes data to a file with the specified permissions
	WriteFile(ctx context.Context, path string, data []byte, perm FileMode) error

	// AtomicWriteFile writes data to a file atomically using a temporary file and rename
	// This prevents corruption if the operation is interrupted
	AtomicWriteFile(ctx context.Context, path string, data []byte, perm FileMode) error

	// MkdirAll creates a directory and all necessary parent directories
	MkdirAll(ctx context.Context, path string, perm FileMode) error

	// RemoveAll removes a path and all its contents
	RemoveAll(ctx context.Context, path string) error

	// Exists checks if a path exists
	Exists(ctx context.Context, path string) bool

	// ListDir lists all files and directories in the given directory
	ListDir(ctx context.Context, path string) ([]string, error)
}
