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
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentSecretsOperations validates that concurrent operations on the new Store
// don't cause race conditions or data corruption
func TestConcurrentSecretsOperations(t *testing.T) {
	// Setup test environment
	tmpDir := t.TempDir()
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	// Create new domain-driven store
	repo := NewFileRepository(tmpDir)
	cryptoService := NewCryptoService(nil) // Will generate key automatically
	masterKeyMgr := NewFileMasterKeyManager(tmpDir)
	store := NewStoreWithMasterKeyManager(repo, cryptoService, masterKeyMgr)

	const numGoroutines = 5           // More realistic concurrency level
	const operationsPerGoroutine = 20 // More realistic operation count
	var wg sync.WaitGroup

	// Test 1: Concurrent Put operations
	t.Run("ConcurrentPuts", func(t *testing.T) {
		var errorCount int64
		var successCount int64

		wg.Add(numGoroutines)
		for i := range make([]struct{}, numGoroutines) {
			go func(workerID int) {
				defer wg.Done()
				for j := range operationsPerGoroutine {
					key := fmt.Sprintf("worker%d_key%d", workerID, j)
					value := fmt.Sprintf("value_%d_%d_%d", workerID, j, time.Now().UnixNano())

					if err := store.Put(key, value); err != nil {
						atomic.AddInt64(&errorCount, 1)
						t.Errorf("Worker %d: Put failed for %s: %v", workerID, key, err)
						continue
					}

					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}
		wg.Wait()

		// Report concurrency statistics
		t.Logf("Concurrency results: %d successes, %d errors", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))

		// Verify all keys were stored
		keys := store.ListKeys()
		expectedCount := numGoroutines * operationsPerGoroutine
		if len(keys) != expectedCount {
			t.Errorf("Expected %d keys, got %d (lost %d operations)", expectedCount, len(keys), expectedCount-len(keys))
		}
	})

	// Test 2: Concurrent mixed operations (Put, Get, Delete, List)
	t.Run("ConcurrentMixedOperations", func(t *testing.T) {
		// Pre-populate some data
		for i := range 100 {
			key := fmt.Sprintf("preset_key_%d", i)
			value := fmt.Sprintf("preset_value_%d", i)
			if err := store.Put(key, value); err != nil {
				t.Fatalf("Failed to preset data: %v", err)
			}
		}

		wg.Add(numGoroutines * 4) // 4 operation types

		// Concurrent Put operations
		for i := range numGoroutines {
			go func(workerID int) {
				defer wg.Done()
				for j := range operationsPerGoroutine {
					key := fmt.Sprintf("concurrent_key_%d_%d", workerID, j)
					value := fmt.Sprintf("concurrent_value_%d_%d", workerID, j)
					_ = store.Put(key, value) // Ignore errors for stress test
				}
			}(i)
		}

		// Concurrent Get operations
		for i := range numGoroutines {
			go func(workerID int) {
				defer wg.Done()
				for j := range operationsPerGoroutine {
					key := fmt.Sprintf("preset_key_%d", (workerID*j)%100)
					_, _ = store.Get(key) // Ignore errors/results for stress test
				}
			}(i)
		}

		// Concurrent List operations
		for range numGoroutines {
			go func() {
				defer wg.Done()
				for range operationsPerGoroutine {
					_ = store.ListKeys() // Ignore results for stress test
				}
			}()
		}

		// Concurrent Delete operations
		for i := range numGoroutines {
			go func(workerID int) {
				defer wg.Done()
				for j := range operationsPerGoroutine {
					key := fmt.Sprintf("preset_key_%d", (workerID*j+50)%100)
					_ = store.Delete(key) // Ignore errors for stress test
				}
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent mixed operations completed without deadlocks")
	})

	// Test 3: Concurrent Disable/Enable operations
	t.Run("ConcurrentDisableEnable", func(t *testing.T) {
		// Pre-populate data
		testKeys := make([]string, 50)
		for i := range 50 {
			key := fmt.Sprintf("disable_test_key_%d", i)
			value := fmt.Sprintf("disable_test_value_%d", i)
			testKeys[i] = key
			if err := store.Put(key, value); err != nil {
				t.Fatalf("Failed to preset data: %v", err)
			}
		}

		wg.Add(numGoroutines * 2)

		// Concurrent Disable operations
		for i := range numGoroutines {
			go func(workerID int) {
				defer wg.Done()
				for j := range 10 {
					key := testKeys[(workerID*j)%len(testKeys)]
					_ = store.DisableSecret(key) // Ignore errors for stress test
				}
			}(i)
		}

		// Concurrent Enable operations
		for i := range numGoroutines {
			go func(workerID int) {
				defer wg.Done()
				for j := range 10 {
					key := testKeys[(workerID*j+25)%len(testKeys)]
					_ = store.EnableSecret(key) // Ignore errors for stress test
				}
			}(i)
		}

		wg.Wait()
		t.Log("Concurrent disable/enable operations completed without deadlocks")
	})
}

// TestConcurrentMasterKeyRotation validates that master key rotation
// is thread-safe and doesn't corrupt data
func TestConcurrentMasterKeyRotation(t *testing.T) {
	// This test is more controlled since rotation is a critical operation
	// that should be serialized
	tmpDir := t.TempDir()
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	store, err := LoadSecretsStore(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("Failed to load secrets store: %v", err)
	}

	// Pre-populate with test data
	for i := range 100 {
		key := fmt.Sprintf("rotation_test_key_%d", i)
		value := fmt.Sprintf("rotation_test_value_%d", i)
		if err := store.Put(key, value); err != nil {
			t.Fatalf("Failed to preset data: %v", err)
		}
	}

	// Test that rotation works correctly under concurrent read operations
	var wg sync.WaitGroup
	done := make(chan bool)

	// Start concurrent read operations
	wg.Add(10)
	for i := range 10 {
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// Perform read operations during rotation
					keys := store.ListKeys()
					if len(keys) > 0 {
						_, _ = store.Get(keys[0]) // Ignore errors during rotation
					}
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	// Perform master key rotation
	backupDir := tmpDir + "/rotation-backup"
	if err := store.RotateMasterKey(backupDir); err != nil {
		t.Fatalf("Master key rotation failed: %v", err)
	}

	// Stop concurrent operations
	close(done)
	wg.Wait()

	// Verify data integrity after rotation
	keys := store.ListKeys()
	if len(keys) != 100 {
		t.Errorf("Expected 100 keys after rotation, got %d", len(keys))
	}

	// Verify we can still read all data
	for i := range 100 {
		key := fmt.Sprintf("rotation_test_key_%d", i)
		expectedValue := fmt.Sprintf("rotation_test_value_%d", i)

		value, err := store.Get(key)
		if err != nil {
			t.Errorf("Failed to read key %s after rotation: %v", key, err)
			continue
		}

		if value != expectedValue {
			t.Errorf("Data corruption detected for key %s: expected %s, got %s",
				key, expectedValue, value)
		}
	}

	t.Log("Master key rotation completed successfully with concurrent operations")
}

// TestAtomicFileOperations validates that all file operations are atomic
func TestAtomicFileOperations(t *testing.T) {
	tmpDir := t.TempDir()

	testData := [][]byte{
		[]byte(`{"test": "data1"}`),
		[]byte(`{"test": "data2"}`),
	}

	const (
		numWriters      = 10
		writesPerWriter = 50
	)

	// Concurrent atomic writes to separate files
	var wg sync.WaitGroup
	wg.Add(numWriters)

	for writerID := 0; writerID < numWriters; writerID++ {
		go func(id int) {
			defer wg.Done()

			testFile := fmt.Sprintf("%s/test-atomic-%d.json", tmpDir, id)
			data := testData[id%len(testData)]

			for range writesPerWriter {
				if err := AtomicWriteFile(testFile, data, 0600); err != nil {
					t.Errorf("Writer %d: AtomicWriteFile failed: %v", id, err)
				}
			}
		}(writerID)
	}

	wg.Wait()

	// Verify all files contain valid, uncorrupted data
	for i := range numWriters {
		testFile := fmt.Sprintf("%s/test-atomic-%d.json", tmpDir, i)
		expectedData := testData[i%len(testData)]

		data, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read test file %d: %v", i, err)
		}

		if string(data) != string(expectedData) {
			t.Errorf("File %d corrupted: expected %s, got %s", i, expectedData, data)
		}
	}

	t.Log("Atomic file operations completed successfully")
}

// selectTestToken alternates between valid and invalid tokens for testing
func selectTestToken(index int) string {
	if index%2 == 0 {
		return "admin-token" // Valid token
	}

	return "invalid-token" // Invalid token
}
