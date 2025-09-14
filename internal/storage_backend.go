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
	"path/filepath"
)

// FilesystemBackend implements StorageBackend for local filesystem operations
type FilesystemBackend struct{}

// NewFilesystemBackend creates a new filesystem storage backend
func NewFilesystemBackend() *FilesystemBackend {
	return &FilesystemBackend{}
}

// ReadFile reads data from a file
func (fs *FilesystemBackend) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file with specified permissions
func (fs *FilesystemBackend) WriteFile(path string, data []byte, perm FileMode) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

// AtomicWriteFile performs an atomic write using a temporary file and rename
func (fs *FilesystemBackend) AtomicWriteFile(path string, data []byte, perm FileMode) error {
	return AtomicWriteFile(path, data, os.FileMode(perm))
}

// MkdirAll creates directories with specified permissions
func (fs *FilesystemBackend) MkdirAll(path string, perm FileMode) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

// RemoveAll removes a directory and all its contents
func (fs *FilesystemBackend) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Exists checks if a file or directory exists
func (fs *FilesystemBackend) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ListDir lists the contents of a directory
func (fs *FilesystemBackend) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

// ensureDirectoryExists creates the parent directory if it doesn't exist
func (fs *FilesystemBackend) ensureDirectoryExists(filePath string) error {
	dir := filepath.Dir(filePath)
	return fs.MkdirAll(dir, FileMode0755)
}
