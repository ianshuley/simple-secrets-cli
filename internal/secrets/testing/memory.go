/*package testing

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

// Package testing provides test doubles for the secrets domain
package testing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileMode represents file permissions for secrets storage operations
type FileMode = os.FileMode

// Repository defines the storage operations needed by the secrets domain
type Repository interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte, perm FileMode) error
	AtomicWriteFile(ctx context.Context, path string, data []byte, perm FileMode) error
	MkdirAll(ctx context.Context, path string, perm FileMode) error
	RemoveAll(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) bool
	ListDir(ctx context.Context, path string) ([]string, error)
}

// fileEntry represents a file in memory
type fileEntry struct {
	data    []byte
	mode    FileMode
	modTime time.Time
	isDir   bool
}

// MemoryRepository implements Repository for in-memory operations during testing
type MemoryRepository struct {
	mu    sync.RWMutex
	files map[string]*fileEntry
}

// NewMemoryRepository creates a new in-memory secrets repository for testing
func NewMemoryRepository() Repository {
	return &MemoryRepository{
		files: make(map[string]*fileEntry),
	}
}

// ReadFile reads the entire content of a file
func (r *MemoryRepository) ReadFile(ctx context.Context, path string) ([]byte, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}

	if entry.isDir {
		return nil, fmt.Errorf("is a directory")
	}

	// Return a copy to prevent modification
	data := make([]byte, len(entry.data))
	copy(data, entry.data)
	return data, nil
}

// WriteFile writes data to a file with the specified permissions
func (r *MemoryRepository) WriteFile(ctx context.Context, path string, data []byte, perm FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create parent directories if they don't exist
	if err := r.ensureParentDirs(path); err != nil {
		return err
	}

	// Store a copy to prevent external modification
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	r.files[path] = &fileEntry{
		data:    dataCopy,
		mode:    perm,
		modTime: time.Now(),
		isDir:   false,
	}

	return nil
}

// AtomicWriteFile writes data to a file atomically (same as WriteFile in memory)
func (r *MemoryRepository) AtomicWriteFile(ctx context.Context, path string, data []byte, perm FileMode) error {
	// In memory, all writes are atomic
	return r.WriteFile(ctx, path, data, perm)
}

// MkdirAll creates a directory and all necessary parent directories
func (r *MemoryRepository) MkdirAll(ctx context.Context, path string, perm FileMode) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Create parent directories first
	if err := r.ensureParentDirs(path); err != nil {
		return err
	}

	// Create the target directory itself
	r.files[path] = &fileEntry{
		data:    nil,
		mode:    perm,
		modTime: time.Now(),
		isDir:   true,
	}

	return nil
}

// RemoveAll removes a path and all its contents
func (r *MemoryRepository) RemoveAll(ctx context.Context, path string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove the path itself and all children
	for filePath := range r.files {
		if filePath == path || strings.HasPrefix(filePath, path+"/") {
			delete(r.files, filePath)
		}
	}

	return nil
}

// Exists checks if a path exists
func (r *MemoryRepository) Exists(ctx context.Context, path string) bool {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.files[path]
	return exists
}

// ListDir lists all files and directories in the given directory
func (r *MemoryRepository) ListDir(ctx context.Context, path string) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}

	if !entry.isDir {
		return nil, fmt.Errorf("not a directory")
	}

	// Find all direct children
	children := make(map[string]bool)
	pathWithSlash := path + "/"
	for filePath := range r.files {
		if strings.HasPrefix(filePath, pathWithSlash) {
			// Get the relative path
			rel := strings.TrimPrefix(filePath, pathWithSlash)
			// Get just the first segment (direct child)
			segments := strings.Split(rel, "/")
			if len(segments) > 0 && segments[0] != "" {
				children[segments[0]] = true
			}
		}
	}

	// Convert to sorted slice
	names := make([]string, 0, len(children))
	for name := range children {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, nil
}

// ensureParentDirs creates parent directories if they don't exist (must be called with lock held)
func (r *MemoryRepository) ensureParentDirs(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "/" {
		return nil
	}

	// Create parent directories recursively
	if err := r.ensureParentDirs(dir); err != nil {
		return err
	}

	// Create this directory if it doesn't exist
	if _, exists := r.files[dir]; !exists {
		r.files[dir] = &fileEntry{
			data:    nil,
			mode:    0755,
			modTime: time.Now(),
			isDir:   true,
		}
	}

	return nil
}
