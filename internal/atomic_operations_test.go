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
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAtomicMasterKeyRotation tests that master key rotation is atomic and interruption-safe
func TestAtomicMasterKeyRotation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create test files
	masterKeyPath := filepath.Join(tmpDir, "master.key")
	secretsPath := filepath.Join(tmpDir, "secrets.json")

	// Create a secrets store for testing
	store := &SecretsStore{
		KeyPath:     masterKeyPath,
		SecretsPath: secretsPath,
		secrets:     make(map[string]string),
	}

	// Initialize the store with some test data
	originalKey := make([]byte, 32) // 32 bytes for AES-256
	copy(originalKey, []byte("original-master-key-12345678901234567890123456789012")[:32])
	err := writeMasterKeyToPath(masterKeyPath, originalKey)
	if err != nil {
		t.Fatalf("Failed to create test master key: %v", err)
	}

	// Create some test secrets
	testSecrets := map[string]string{
		"test-key-1": "test-value-1",
		"test-key-2": "test-value-2",
	}

	store.masterKey = originalKey
	for key, value := range testSecrets {
		err := store.Put(key, value)
		if err != nil {
			t.Fatalf("Failed to create test secret %s: %v", key, err)
		}
	}

	// Perform atomic rotation
	backupDir := filepath.Join(tmpDir, "test-backup")
	err = store.RotateMasterKey(backupDir)
	if err != nil {
		t.Fatalf("Atomic rotation failed: %v", err)
	}

	// Verify secrets are still accessible after rotation
	newStore := &SecretsStore{
		KeyPath:     masterKeyPath,
		SecretsPath: secretsPath,
		secrets:     make(map[string]string),
	}

	err = newStore.loadOrCreateKey()
	if err != nil {
		t.Fatalf("Failed to load rotated key: %v", err)
	}

	err = newStore.loadSecrets()
	if err != nil {
		t.Fatalf("Failed to load secrets after rotation: %v", err)
	}

	// Verify all secrets are still accessible
	for key, expectedValue := range testSecrets {
		actualValue, err := newStore.Get(key)
		if err != nil {
			t.Errorf("Failed to get secret %s after rotation: %v", key, err)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Secret %s has wrong value after rotation. Expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Verify no temporary files remain
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tmp" {
			t.Errorf("Temporary file %s was not cleaned up", file.Name())
		}
	}

	// Verify backup was created
	if _, err := os.Stat(backupDir); err != nil {
		t.Errorf("Backup directory was not created: %v", err)
	}
}

// TestAtomicOperationTempFileCleanup tests that temporary files are properly cleaned up
func TestAtomicOperationTempFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	masterKeyPath := filepath.Join(tmpDir, "master.key")

	// Create original file
	originalKey := []byte("original-key")
	err := os.WriteFile(masterKeyPath, originalKey, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test multiple atomic operations to ensure temp files are cleaned up
	for i := 0; i < 5; i++ {
		newKey := []byte("new-key-" + string(rune('0'+i)))
		err = writeMasterKeyToPath(masterKeyPath, newKey)
		if err != nil {
			t.Fatalf("Operation %d failed: %v", i, err)
		}

		// Check no temp files exist after each operation
		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		tempFileCount := 0
		for _, file := range files {
			if filepath.Ext(file.Name()) == ".tmp" {
				tempFileCount++
			}
		}

		if tempFileCount > 0 {
			t.Errorf("After operation %d, found %d temporary files", i, tempFileCount)
		}
	}
}

// TestAtomicOperationPermissions tests that atomic operations preserve file permissions
func TestAtomicOperationPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	masterKeyPath := filepath.Join(tmpDir, "master.key")

	// Create original file with specific permissions
	originalKey := []byte("original-key")
	err := os.WriteFile(masterKeyPath, originalKey, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify initial permissions
	info, err := os.Stat(masterKeyPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	expectedPerms := os.FileMode(0600)
	if info.Mode().Perm() != expectedPerms {
		t.Fatalf("Initial permissions incorrect. Expected %v, got %v", expectedPerms, info.Mode().Perm())
	}

	// Perform atomic operation
	newKey := []byte("new-key-with-permissions-test")
	err = writeMasterKeyToPath(masterKeyPath, newKey)
	if err != nil {
		t.Fatalf("Atomic operation failed: %v", err)
	}

	// Verify permissions are preserved
	info, err = os.Stat(masterKeyPath)
	if err != nil {
		t.Fatalf("Failed to stat file after operation: %v", err)
	}

	if info.Mode().Perm() != expectedPerms {
		t.Errorf("Permissions not preserved. Expected %v, got %v", expectedPerms, info.Mode().Perm())
	}
}

// TestAtomicOperationConcurrency tests that concurrent atomic operations don't interfere
func TestAtomicOperationConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	masterKeyPath := filepath.Join(tmpDir, "master.key")

	// Create original file
	originalKey := []byte("original-key")
	err := os.WriteFile(masterKeyPath, originalKey, 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run concurrent atomic operations
	done := make(chan bool, 3)
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < 10; j++ {
				key := []byte("key-from-goroutine-" + string(rune('0'+id)) + "-iteration-" + string(rune('0'+j)))
				err := writeMasterKeyToPath(masterKeyPath, key)
				if err != nil {
					errors <- err
					return
				}
				time.Sleep(time.Millisecond) // Small delay to increase chance of race conditions
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case err := <-errors:
			t.Fatalf("Concurrent operation failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify file is in a valid state (can be read)
	finalKey, err := os.ReadFile(masterKeyPath)
	if err != nil {
		t.Fatalf("Failed to read final key: %v", err)
	}

	if len(finalKey) == 0 {
		t.Error("Final key is empty, indicating possible corruption")
	}

	// Verify no temp files remain
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tmp" {
			t.Errorf("Temporary file %s was not cleaned up after concurrent operations", file.Name())
		}
	}
}
