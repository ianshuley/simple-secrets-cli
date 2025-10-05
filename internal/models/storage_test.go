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

import (
	"os"
	"testing"
)

func TestNewFileLock(t *testing.T) {
	file := &os.File{}
	path := "/test/path"

	lock := NewFileLock(file, path)

	if lock == nil {
		t.Fatal("NewFileLock should not return nil")
	}

	if lock.File != file {
		t.Error("FileLock.File should match the provided file")
	}

	if lock.Path != path {
		t.Error("FileLock.Path should match the provided path")
	}
}

func TestFileLock_Fields(t *testing.T) {
	lock := &FileLock{
		File: nil,
		Path: "/some/path",
	}

	if lock.Path != "/some/path" {
		t.Errorf("Expected path '/some/path', got %s", lock.Path)
	}

	if lock.File != nil {
		t.Error("Expected File to be nil")
	}
}

// Test StorageBackend interface - this is more of a compilation test
// to ensure the interface is well-defined
func TestStorageBackend_Interface(t *testing.T) {
	// This test ensures that the StorageBackend interface
	// has all the expected methods with correct signatures
	var _ StorageBackend = (*testStorageBackend)(nil)
}

// testStorageBackend implements StorageBackend for testing
type testStorageBackend struct{}

func (t *testStorageBackend) ReadFile(path string) ([]byte, error) {
	return []byte("test data"), nil
}

func (t *testStorageBackend) WriteFile(path string, data []byte, perm FileMode) error {
	return nil
}

func (t *testStorageBackend) AtomicWriteFile(path string, data []byte, perm FileMode) error {
	return nil
}

func (t *testStorageBackend) MkdirAll(path string, perm FileMode) error {
	return nil
}

func (t *testStorageBackend) RemoveAll(path string) error {
	return nil
}

func (t *testStorageBackend) Exists(path string) bool {
	return true
}

func (t *testStorageBackend) ListDir(path string) ([]string, error) {
	return []string{"file1", "file2"}, nil
}

func TestStorageBackend_Implementation(t *testing.T) {
	backend := &testStorageBackend{}

	// Test ReadFile
	data, err := backend.ReadFile("/test/path")
	if err != nil {
		t.Errorf("ReadFile returned error: %v", err)
	}
	if string(data) != "test data" {
		t.Errorf("Expected 'test data', got %s", string(data))
	}

	// Test WriteFile
	err = backend.WriteFile("/test/path", []byte("data"), 0644)
	if err != nil {
		t.Errorf("WriteFile returned error: %v", err)
	}

	// Test AtomicWriteFile
	err = backend.AtomicWriteFile("/test/path", []byte("data"), 0644)
	if err != nil {
		t.Errorf("AtomicWriteFile returned error: %v", err)
	}

	// Test MkdirAll
	err = backend.MkdirAll("/test/dir", 0755)
	if err != nil {
		t.Errorf("MkdirAll returned error: %v", err)
	}

	// Test RemoveAll
	err = backend.RemoveAll("/test/path")
	if err != nil {
		t.Errorf("RemoveAll returned error: %v", err)
	}

	// Test Exists
	exists := backend.Exists("/test/path")
	if !exists {
		t.Error("Exists should return true")
	}

	// Test ListDir
	files, err := backend.ListDir("/test/dir")
	if err != nil {
		t.Errorf("ListDir returned error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}
