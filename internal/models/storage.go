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

package models

import "os"

// FileMode represents file permissions
type FileMode = os.FileMode

// StorageBackend defines the interface for storage operations used by the secrets domain
type StorageBackend interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm FileMode) error
	AtomicWriteFile(path string, data []byte, perm FileMode) error
	MkdirAll(path string, perm FileMode) error
	RemoveAll(path string) error
	Exists(path string) bool
	ListDir(path string) ([]string, error)
}

// FileLock represents a file lock for coordinating concurrent access
type FileLock struct {
	File *os.File
	Path string
}

// NewFileLock creates a new file lock
func NewFileLock(file *os.File, path string) *FileLock {
	return &FileLock{
		File: file,
		Path: path,
	}
}
