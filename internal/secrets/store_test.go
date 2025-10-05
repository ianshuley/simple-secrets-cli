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
	"os"
	"path/filepath"
	"testing"

	secretstesting "simple-secrets/internal/secrets/testing"
)

func TestStoreWithMemoryRepository(t *testing.T) {
	// Use memory repository for testing
	repo := secretstesting.NewMemoryRepository() 
	
	// Create temporary config directory for test
	tempDir := t.TempDir()
	os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
	
	// Load store
	store, err := LoadStore(repo)
	if err != nil {
		t.Fatalf("LoadStore() failed: %v", err)
	}
	
	// Test basic operations
	testKey := "test-secret"
	testValue := "secret-value-123"
	
	// Put a secret
	err = store.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Put() failed: %v", err)
	}
	
	// Get the secret back
	retrieved, err := store.Get(testKey)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	
	if retrieved != testValue {
		t.Errorf("Get() = %v, want %v", retrieved, testValue)
	}
	
	// List keys
	keys := store.ListKeys()
	if len(keys) != 1 || keys[0] != testKey {
		t.Errorf("ListKeys() = %v, want [%s]", keys, testKey)
	}
}

func TestStoreWithFilesystemRepository(t *testing.T) {
	// Use filesystem repository
	repo := NewFilesystemRepository()
	
	// Create temporary config directory for test
	tempDir := t.TempDir()
	os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir)
	defer os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
	
	// Load store
	store, err := LoadStore(repo)
	if err != nil {
		t.Fatalf("LoadStore() failed: %v", err)
	}
	
	// Test that key and secrets files are created
	expectedKeyPath := filepath.Join(tempDir, "master.key")
	expectedSecretsPath := filepath.Join(tempDir, "secrets.json")
	
	if _, err := os.Stat(expectedKeyPath); os.IsNotExist(err) {
		t.Error("Master key file was not created")
	}
	
	// Test basic operation
	testKey := "filesystem-test"
	testValue := "filesystem-value"
	
	err = store.Put(testKey, testValue)
	if err != nil {
		t.Fatalf("Put() failed: %v", err)
	}
	
	// Verify secrets file was created
	if _, err := os.Stat(expectedSecretsPath); os.IsNotExist(err) {
		t.Error("Secrets file was not created after Put()")
	}
	
	// Verify we can retrieve the secret
	retrieved, err := store.Get(testKey)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	
	if retrieved != testValue {
		t.Errorf("Get() = %v, want %v", retrieved, testValue)
	}
}