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
	"simple-secrets/pkg/api"
	"testing"
)

// newTempStoreForAPI creates a temporary store for API interface testing
func newTempStoreForAPI(t *testing.T) (*SecretsStore, *UserStore) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmp+"/.simple-secrets")

	store, err := LoadSecretsStore(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("LoadSecretsStore: %v", err)
	}

	// Create user store for testing
	userStore := createUserStore([]*User{}, createDefaultRoles())

	return store, userStore
}

// TestSecretReaderInterface validates SecretReader interface implementation
func TestSecretReaderInterface(t *testing.T) {
	store, userStore := newTempStoreForAPI(t)
	adapter := NewServiceAdapter(store, userStore)

	// Test through SecretReader interface
	var reader api.SecretReader = adapter

	// Put some test data first (through concrete type)
	key, value := "test-key", "test-value"
	if err := store.Put(key, value); err != nil {
		t.Fatalf("Failed to put test data: %v", err)
	}

	// Test Get through interface
	retrieved, err := reader.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved != value {
		t.Errorf("Get returned %q, expected %q", retrieved, value)
	}

	// Test List through interface
	keys := reader.List()
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("List returned %v, expected [%s]", keys, key)
	}

	// Test IsEnabled through interface
	if !reader.IsEnabled(key) {
		t.Error("IsEnabled returned false for enabled secret")
	}
}

// TestSecretWriterInterface validates SecretWriter interface implementation
func TestSecretWriterInterface(t *testing.T) {
	store, userStore := newTempStoreForAPI(t)
	adapter := NewServiceAdapter(store, userStore)

	// Test through SecretWriter interface
	var writer api.SecretWriter = adapter

	key, value := "writer-test", "writer-value"

	// Test Put through interface
	if err := writer.Put(key, value); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// Verify through concrete type
	retrieved, err := store.Get(key)
	if err != nil {
		t.Fatalf("Failed to verify Put: %v", err)
	}
	if retrieved != value {
		t.Errorf("Put stored %q, expected %q", retrieved, value)
	}

	// Test Disable through interface
	if err := writer.Disable(key); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}

	// Test Enable through interface
	if err := writer.Enable(key); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}

	// Test Delete through interface
	if err := writer.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err = store.Get(key)
	if err == nil {
		t.Error("Secret still exists after Delete")
	}
}

// TestAuthenticatorInterface validates Authenticator interface implementation
func TestAuthenticatorInterface(t *testing.T) {
	_, userStore := newTempStoreForAPI(t)

	// Create test user
	token, err := userStore.CreateUser("testuser", "reader")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	adapter := NewServiceAdapter(nil, userStore) // Don't need secrets store for auth tests

	// Test through Authenticator interface
	var auth api.Authenticator = adapter

	// Test Authenticate
	user, err := auth.Authenticate(token)
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("Authenticate returned user %q, expected %q", user.Username, "testuser")
	}
	if user.Role != "reader" {
		t.Errorf("Authenticate returned role %q, expected %q", user.Role, "reader")
	}

	// Test CanRead
	if !auth.CanRead(user) {
		t.Error("CanRead returned false for reader user")
	}

	// Test CanWrite (reader should not be able to write)
	if auth.CanWrite(user) {
		t.Error("CanWrite returned true for reader user")
	}

	// Test CanAdmin (reader should not be admin)
	if auth.CanAdmin(user) {
		t.Error("CanAdmin returned true for reader user")
	}
}

// TestNewUserManagerInterface validates UserManager interface implementation
func TestNewUserManagerInterface(t *testing.T) {
	_, userStore := newTempStoreForAPI(t)
	adapter := NewServiceAdapter(nil, userStore)

	// Test through UserManager interface
	var mgr api.UserManager = adapter

	// Test CreateUser
	token, err := mgr.CreateUser("newuser", "admin")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if token == "" {
		t.Error("CreateUser returned empty token")
	}

	// Test ListUsers
	users, err := mgr.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("ListUsers returned %d users, expected 1", len(users))
	}
	if users[0].Username != "newuser" {
		t.Errorf("ListUsers returned user %q, expected %q", users[0].Username, "newuser")
	}

	// Test RotateToken
	newToken, err := mgr.RotateToken("newuser")
	if err != nil {
		t.Fatalf("RotateToken failed: %v", err)
	}
	if newToken == "" || newToken == token {
		t.Error("RotateToken did not return a new token")
	}
}

// TestFullServiceComposition validates that ServiceAdapter implements FullService
func TestFullServiceComposition(t *testing.T) {
	store, userStore := newTempStoreForAPI(t)
	adapter := NewServiceAdapter(store, userStore)

	// Test that adapter implements all interfaces
	var _ api.SecretReader = adapter
	var _ api.SecretWriter = adapter
	var _ api.Authenticator = adapter
	var _ api.UserManager = adapter
	var _ api.AdminOperations = adapter
	var _ api.FullService = adapter

	// Test that it works as FullService
	var full api.FullService = adapter

	// Create test user
	token, err := full.CreateUser("fulluser", "admin")
	if err != nil {
		t.Fatalf("FullService CreateUser failed: %v", err)
	}

	// Authenticate
	user, err := full.Authenticate(token)
	if err != nil {
		t.Fatalf("FullService Authenticate failed: %v", err)
	}

	// Test permissions
	if !full.CanAdmin(user) {
		t.Error("FullService admin user cannot admin")
	}

	// Test secret operations
	if err := full.Put("test", "value"); err != nil {
		t.Fatalf("FullService Put failed: %v", err)
	}

	value, err := full.Get("test")
	if err != nil {
		t.Fatalf("FullService Get failed: %v", err)
	}
	if value != "value" {
		t.Errorf("FullService Get returned %q, expected %q", value, "value")
	}

	t.Log("All API interface tests passed!")
}
