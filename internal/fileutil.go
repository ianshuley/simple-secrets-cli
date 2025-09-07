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
package internal

import (
	"fmt"
	"os"
)

// AtomicWriteFile writes data to a file atomically using a temporary file and rename.
// This ensures that either the entire write succeeds or fails completely, preventing
// partial writes that could corrupt the file.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"

	// Write to temporary file first
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename to final location
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically update file: %w", err)
	}

	return nil
}
