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

package users

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"simple-secrets/pkg/users"
)

func TestFileRepository_UserOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	ctx := context.Background()

	// Test user creation and retrieval
	user := users.NewUser("testuser", "admin")
	token := users.NewToken("default", "hash123")
	user.AddToken(token)

	// Store user
	err = repo.Store(ctx, user)
	if err != nil {
		t.Fatalf("Failed to store user: %v", err)
	}

	// Retrieve user
	retrieved, err := repo.Retrieve(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", retrieved.Username)
	}

	if retrieved.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", retrieved.Role)
	}

	if len(retrieved.Tokens) != 1 {
		t.Errorf("Expected 1 token, got %d", len(retrieved.Tokens))
	}

	// Test RetrieveByToken
	userByToken, err := repo.RetrieveByToken(ctx, "hash123")
	if err != nil {
		t.Fatalf("Failed to retrieve user by token: %v", err)
	}

	if userByToken.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", userByToken.Username)
	}

	// Test user listing
	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// Test user exists
	exists, err := repo.Exists(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if !exists {
		t.Error("Expected user to exist")
	}

	exists, err = repo.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if exists {
		t.Error("Expected user to not exist")
	}

	// Test enable/disable
	err = repo.Disable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to disable user: %v", err)
	}

	disabled, err := repo.Retrieve(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to retrieve disabled user: %v", err)
	}

	if !disabled.Disabled {
		t.Error("Expected user to be disabled")
	}

	err = repo.Enable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to enable user: %v", err)
	}

	enabled, err := repo.Retrieve(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to retrieve enabled user: %v", err)
	}

	if enabled.Disabled {
		t.Error("Expected user to be enabled")
	}

	// Test deletion
	err = repo.Delete(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	exists, err = repo.Exists(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to check deleted user existence: %v", err)
	}

	if exists {
		t.Error("Expected deleted user to not exist")
	}
}

func TestFileRepository_AtomicOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_atomic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	ctx := context.Background()

	// Test that the users file is created with proper permissions
	user := users.NewUser("testuser", "admin")
	err = repo.Store(ctx, user)
	if err != nil {
		t.Fatalf("Failed to store user: %v", err)
	}

	usersFile := filepath.Join(tempDir, "users.json")
	stat, err := os.Stat(usersFile)
	if err != nil {
		t.Fatalf("Failed to stat users file: %v", err)
	}

	expectedPerm := os.FileMode(0600)
	if stat.Mode().Perm() != expectedPerm {
		t.Errorf("Expected file permissions %v, got %v", expectedPerm, stat.Mode().Perm())
	}
}

func TestFileRepository_ContextCancellation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_context_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	user := users.NewUser("testuser", "admin")

	err = repo.Store(ctx, user)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	_, err = repo.Retrieve(ctx, "testuser")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	_, err = repo.RetrieveByToken(ctx, "hash123")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	err = repo.Delete(ctx, "testuser")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	_, err = repo.List(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	err = repo.Enable(ctx, "testuser")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	err = repo.Disable(ctx, "testuser")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	_, err = repo.Exists(ctx, "testuser")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}
