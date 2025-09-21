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
	"testing"
	"time"
)

// TestAtomicMasterKeyRotation tests that master key rotation is atomic and interruption-safe
func TestAtomicMasterKeyRotation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir)

	// Create a secrets store using proper constructor
	store, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Initialize the store with some test data
	originalKey := make([]byte, AES256KeySize) // AES-256
	copy(originalKey, []byte("original-master-key-12345678901234567890123456789012")[:32])

	// Set the master key directly for testing
	err = store.writeMasterKeyToPath(store.KeyPath, originalKey)
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
	newStore, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("Failed to create new store: %v", err)
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
	tempStore, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("Failed to create temp store: %v", err)
	}

	for i := 0; i < 5; i++ {
		newKey := []byte("new-key-" + string(rune('0'+i)))
		err = tempStore.writeMasterKeyToPath(masterKeyPath, newKey)
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
	tempStore, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("Failed to create temp store: %v", err)
	}

	newKey := []byte("new-key-with-permissions-test")
	err = tempStore.writeMasterKeyToPath(masterKeyPath, newKey)
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

	// Run concurrent atomic operations on separate files
	done := make(chan bool, 3)
	errors := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine works on its own file
			masterKeyPath := filepath.Join(tmpDir, fmt.Sprintf("master-%d.key", id))

			// Create original file
			originalKey := []byte(fmt.Sprintf("original-key-%d", id))
			err := os.WriteFile(masterKeyPath, originalKey, 0600)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: failed to create test file: %w", id, err)
				return
			}

			for j := 0; j < 10; j++ {
				// Create temp store for this operation
				tempStore, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: failed to create temp store: %w", id, err)
					return
				}

				key := []byte(fmt.Sprintf("key-from-goroutine-%d-iteration-%d", id, j))
				err = tempStore.writeMasterKeyToPath(masterKeyPath, key)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: %w", id, err)
					return
				}
				time.Sleep(time.Millisecond) // Small delay to increase chance of race conditions
			} // Verify final file is in a valid state (can be read)
			finalKey, err := os.ReadFile(masterKeyPath)
			if err != nil {
				errors <- fmt.Errorf("goroutine %d: failed to read final key: %w", id, err)
				return
			}

			if len(finalKey) == 0 {
				errors <- fmt.Errorf("goroutine %d: final key is empty, indicating possible corruption", id)
				return
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for range 3 {
		select {
		case <-done:
			// Goroutine completed successfully
		case err := <-errors:
			t.Fatalf("Concurrent operation failed: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// Verify no temp files remain in the entire tmpDir
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read tmpDir: %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".tmp" {
			t.Errorf("Temporary file left behind: %s", file.Name())
		}
	}
}
