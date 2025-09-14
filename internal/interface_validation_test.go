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
	"sync"
	"testing"
)

// newTempStoreForInterface creates a temporary store for interface testing
func newTempStoreForInterface(t *testing.T) *SecretsStore {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmp+"/.simple-secrets")
	s, err := LoadSecretsStore()
	if err != nil {
		t.Fatalf("LoadSecretsStore: %v", err)
	}
	return s
}

// TestSecretsManagerInterface validates that SecretsStore implements SecretsManager correctly
func TestSecretsManagerInterface(t *testing.T) {
	store := newTempStoreForInterface(t)

	// Test through interface
	var secretsMgr SecretsManager = store

	// Test CRUD operations through interface
	key := "test/secret"
	value := "secret-value"

	if err := secretsMgr.Put(key, value); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	retrieved, err := secretsMgr.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved != value {
		t.Errorf("Get returned %q, expected %q", retrieved, value)
	}

	keys := secretsMgr.ListKeys()
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("ListKeys returned %v, expected [%s]", keys, key)
	}

	// Test state management
	if err := secretsMgr.DisableSecret(key); err != nil {
		t.Fatalf("DisableSecret failed: %v", err)
	}

	if secretsMgr.IsEnabled(key) {
		t.Error("Secret should be disabled")
	}

	if err := secretsMgr.EnableSecret(key); err != nil {
		t.Fatalf("EnableSecret failed: %v", err)
	}

	if !secretsMgr.IsEnabled(key) {
		t.Error("Secret should be enabled")
	}

	// Test deletion through interface
	if err := secretsMgr.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = secretsMgr.Get(key)
	if err == nil {
		t.Error("Get should fail after delete")
	}
}

// TestUserManagerInterface validates that UserStore implements UserManager correctly
func TestUserManagerInterface(t *testing.T) {
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-token"), Role: RoleAdmin},
	}
	permissions := RolePermissions{
		RoleAdmin:  {"read", "write", "manage-users"},
		RoleReader: {"read"},
	}

	userStore := &UserStore{
		users:       users,
		permissions: permissions,
	}

	// Test through interface
	var userMgr UserManager = userStore

	// Test lookup through interface
	user, err := userMgr.Lookup("admin-token")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("Lookup returned user %q, expected admin", user.Username)
	}

	// Test user creation through interface
	token, err := userMgr.CreateUser("reader", "reader")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify user was created
	readerUser, err := userMgr.Lookup(token)
	if err != nil {
		t.Fatalf("Lookup for created user failed: %v", err)
	}
	if readerUser.Username != "reader" || readerUser.Role != RoleReader {
		t.Errorf("Created user has incorrect data: %+v", readerUser)
	}

	// Test role update through interface
	if err := userMgr.UpdateUserRole("reader", "admin"); err != nil {
		t.Fatalf("UpdateUserRole failed: %v", err)
	}

	// Verify role was updated
	updatedUser, err := userMgr.Lookup(token)
	if err != nil {
		t.Fatalf("Lookup after role update failed: %v", err)
	}
	if updatedUser.Role != RoleAdmin {
		t.Errorf("Role update failed: expected %v, got %v", RoleAdmin, updatedUser.Role)
	}

	// Test user deletion through interface
	if err := userMgr.DeleteUser("reader"); err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// Verify user was deleted
	_, err = userMgr.Lookup(token)
	if err == nil {
		t.Error("User should not exist after deletion")
	}
}

// TestStorageBackendInterface validates that FilesystemBackend implements StorageBackend correctly
func TestStorageBackendInterface(t *testing.T) {
	tmpDir := t.TempDir()
	backend := NewFilesystemBackend()

	// Test through interface
	var storage StorageBackend = backend

	// Test file operations through interface
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test data")

	if err := storage.WriteFile(testFile, testData, FileMode0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if !storage.Exists(testFile) {
		t.Error("File should exist after write")
	}

	readData, err := storage.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readData) != string(testData) {
		t.Errorf("ReadFile returned %q, expected %q", readData, testData)
	}

	// Test atomic operations through interface
	atomicFile := filepath.Join(tmpDir, "atomic.txt")
	atomicData := []byte("atomic data")

	if err := storage.AtomicWriteFile(atomicFile, atomicData, FileMode0600); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	atomicRead, err := storage.ReadFile(atomicFile)
	if err != nil {
		t.Fatalf("ReadFile for atomic file failed: %v", err)
	}
	if string(atomicRead) != string(atomicData) {
		t.Errorf("AtomicWriteFile data mismatch: got %q, expected %q", atomicRead, atomicData)
	}

	// Test directory operations through interface
	testDir := filepath.Join(tmpDir, "testdir")
	if err := storage.MkdirAll(testDir, FileMode0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	if !storage.Exists(testDir) {
		t.Error("Directory should exist after MkdirAll")
	}

	// Test directory listing through interface
	if err := storage.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("1"), FileMode0644); err != nil {
		t.Fatalf("WriteFile in dir failed: %v", err)
	}
	if err := storage.WriteFile(filepath.Join(testDir, "file2.txt"), []byte("2"), FileMode0644); err != nil {
		t.Fatalf("WriteFile in dir failed: %v", err)
	}

	files, err := storage.ListDir(testDir)
	if err != nil {
		t.Fatalf("ListDir failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("ListDir returned %d files, expected 2", len(files))
	}
}

// TestConcurrentInterfaceOperations validates that interface operations are thread-safe
func TestConcurrentInterfaceOperations(t *testing.T) {
	store := newTempStoreForInterface(t)

	// Test concurrent operations through SecretsManager interface
	var secretsMgr SecretsManager = store

	const numGoroutines = 20
	const operationsPerGoroutine = 25

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent CRUD operations through interface
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range operationsPerGoroutine {
				key := fmt.Sprintf("test/%d/%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)

				// Put operation through interface
				if err := secretsMgr.Put(key, value); err != nil {
					t.Errorf("Goroutine %d: Put failed: %v", id, err)
					return
				}

				// Get operation through interface
				retrieved, err := secretsMgr.Get(key)
				if err != nil {
					t.Errorf("Goroutine %d: Get failed: %v", id, err)
					return
				}
				if retrieved != value {
					t.Errorf("Goroutine %d: Get returned %q, expected %q", id, retrieved, value)
					return
				}

				// State management through interface
				if err := secretsMgr.DisableSecret(key); err != nil {
					t.Errorf("Goroutine %d: DisableSecret failed: %v", id, err)
					return
				}

				if secretsMgr.IsEnabled(key) {
					t.Errorf("Goroutine %d: Secret should be disabled", id)
					return
				}

				if err := secretsMgr.EnableSecret(key); err != nil {
					t.Errorf("Goroutine %d: EnableSecret failed: %v", id, err)
					return
				}

				if !secretsMgr.IsEnabled(key) {
					t.Errorf("Goroutine %d: Secret should be enabled", id)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state through interface
	keys := secretsMgr.ListKeys()
	expectedKeys := numGoroutines * operationsPerGoroutine
	if len(keys) != expectedKeys {
		t.Errorf("Expected %d keys, got %d", expectedKeys, len(keys))
	}

	t.Logf("Interface concurrency test completed: %d goroutines × %d operations = %d total operations",
		numGoroutines, operationsPerGoroutine, numGoroutines*operationsPerGoroutine)
}

// TestConcurrentUserManagerOperations validates that UserManager operations are thread-safe
func TestConcurrentUserManagerOperations(t *testing.T) {
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-token"), Role: RoleAdmin},
	}
	permissions := RolePermissions{
		RoleAdmin:  {"read", "write", "manage-users"},
		RoleReader: {"read"},
	}

	userStore := &UserStore{
		users:       users,
		permissions: permissions,
	}

	// Test through interface
	var userMgr UserManager = userStore

	const numGoroutines = 10
	const usersPerGoroutine = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent user operations through interface
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range usersPerGoroutine {
				username := fmt.Sprintf("user-%d-%d", id, j)

				// Create user through interface
				token, err := userMgr.CreateUser(username, "reader")
				if err != nil {
					t.Errorf("Goroutine %d: CreateUser failed: %v", id, err)
					return
				}

				// Lookup user through interface
				user, err := userMgr.Lookup(token)
				if err != nil {
					t.Errorf("Goroutine %d: Lookup failed: %v", id, err)
					return
				}
				if user.Username != username {
					t.Errorf("Goroutine %d: Lookup returned wrong user: %q vs %q", id, user.Username, username)
					return
				}

				// Update role through interface
				if err := userMgr.UpdateUserRole(username, "admin"); err != nil {
					t.Errorf("Goroutine %d: UpdateUserRole failed: %v", id, err)
					return
				}

				// Verify role update
				updatedUser, err := userMgr.Lookup(token)
				if err != nil {
					t.Errorf("Goroutine %d: Lookup after update failed: %v", id, err)
					return
				}
				if updatedUser.Role != RoleAdmin {
					t.Errorf("Goroutine %d: Role not updated correctly", id)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify final state through interface
	expectedUsers := 1 + (numGoroutines * usersPerGoroutine) // 1 original admin + created users
	if len(userStore.Users()) != expectedUsers {
		t.Errorf("Expected %d users, got %d", expectedUsers, len(userStore.Users()))
	}

	t.Logf("UserManager concurrency test completed: %d goroutines × %d users = %d total operations",
		numGoroutines, usersPerGoroutine, numGoroutines*usersPerGoroutine)
}

// TestMasterKeyRotationThroughInterface validates master key rotation works through interfaces
func TestMasterKeyRotationThroughInterface(t *testing.T) {
	store := newTempStoreForInterface(t)

	// Test through SecretsManager interface
	var secretsMgr SecretsManager = store

	// Add test data through interface
	testSecrets := map[string]string{
		"test/key1": "value1",
		"test/key2": "value2",
		"test/key3": "value3",
	}

	for key, value := range testSecrets {
		if err := secretsMgr.Put(key, value); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Perform master key rotation through interface
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "rotation-backup")
	if err := secretsMgr.RotateMasterKey(backupDir); err != nil {
		t.Fatalf("RotateMasterKey failed: %v", err)
	}

	// Verify secrets are still accessible through interface after rotation
	for key, expectedValue := range testSecrets {
		value, err := secretsMgr.Get(key)
		if err != nil {
			t.Fatalf("Get after rotation failed for key %s: %v", key, err)
		}
		if value != expectedValue {
			t.Errorf("Value mismatch after rotation for key %s: got %q, expected %q", key, value, expectedValue)
		}
	}

	// Verify backup was created
	if _, err := os.Stat(backupDir); err != nil {
		t.Errorf("Backup directory not created: %v", err)
	}

	t.Log("Master key rotation through interface completed successfully")
}
