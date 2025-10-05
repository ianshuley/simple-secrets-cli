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

package secrets

import (
	"context"
	"os"
	"path/filepath"
)

// filesystemRepository implements Repository for local filesystem operations
type filesystemRepository struct{}

// NewFilesystemRepository creates a new filesystem-based secrets repository
func NewFilesystemRepository() Repository {
	return &filesystemRepository{}
}

// ReadFile reads the entire content of a file
func (r *filesystemRepository) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return os.ReadFile(path)
}

// WriteFile writes data to a file with the specified permissions
func (r *filesystemRepository) WriteFile(ctx context.Context, path string, data []byte, perm FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return os.WriteFile(path, data, perm)
}

// AtomicWriteFile writes data to a file atomically using a temporary file and rename
func (r *filesystemRepository) AtomicWriteFile(ctx context.Context, path string, data []byte, perm FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Create temporary file in the same directory as the target
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-"+filepath.Base(path))
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	// Write data to temp file
	if _, err = tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}

	// Set permissions
	if err = tmpFile.Chmod(perm); err != nil {
		tmpFile.Close()
		return err
	}

	// Close temp file
	if err = tmpFile.Close(); err != nil {
		return err
	}

	// Check context again before final rename
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Atomically rename temp file to target
	return os.Rename(tmpPath, path)
}

// MkdirAll creates a directory and all necessary parent directories
func (r *filesystemRepository) MkdirAll(ctx context.Context, path string, perm FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return os.MkdirAll(path, perm)
}

// RemoveAll removes a path and all its contents
func (r *filesystemRepository) RemoveAll(ctx context.Context, path string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	return os.RemoveAll(path)
}

// Exists checks if a path exists
func (r *filesystemRepository) Exists(ctx context.Context, path string) bool {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}

	_, err := os.Stat(path)
	return err == nil
}

// ListDir lists all files and directories in the given directory
func (r *filesystemRepository) ListDir(ctx context.Context, path string) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}

	return names, nil
}