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
	stderrors "errors"
	"os"
	"testing"

	"simple-secrets/pkg/errors"
)

func TestStoreImpl_UserCreation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_store_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	store := NewStore(repo)
	ctx := context.Background()

	// Test successful user creation
	user, token, err := store.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	if user.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", user.Role)
	}

	if len(user.Tokens) != 1 {
		t.Errorf("Expected 1 token, got %d", len(user.Tokens))
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}

	// Test that token works for retrieval
	retrievedUser, err := store.GetByToken(ctx, token)
	if err != nil {
		t.Fatalf("Failed to get user by token: %v", err)
	}

	if retrievedUser.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", retrievedUser.Username)
	}

	// Test duplicate user creation fails
	_, _, err = store.Create(ctx, "testuser", "admin")
	if err == nil {
		t.Error("Expected error for duplicate user creation")
	}

	// Check that it's the right type of error
	var structErr *errors.StructuredError
	if !stderrors.As(err, &structErr) {
		t.Errorf("Expected StructuredError, got %T", err)
		return
	}
	if structErr.Code != errors.ErrCodeAlreadyExists {
		t.Errorf("Expected ALREADY_EXISTS error, got %s", structErr.Code)
	}
}

func TestStoreImpl_TokenOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_token_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	store := NewStore(repo)
	ctx := context.Background()

	// Create user
	_, _, err = store.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// First get the initial default token
	user, err := store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if len(user.Tokens) != 1 {
		t.Fatalf("Expected 1 initial token, got %d", len(user.Tokens))
	}

	// Test token rotation (rotates the primary/default token)
	newToken, err := store.RotateToken(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to rotate token: %v", err)
	}

	if newToken == "" {
		t.Error("Expected non-empty new token")
	}

	// New token should work
	_, err = store.GetByToken(ctx, newToken)
	if err != nil {
		t.Fatalf("New token should work: %v", err)
	}

	// Test adding named token
	token, tokenValue, err := store.AddToken(ctx, "testuser", "CICD Token")
	if err != nil {
		t.Fatalf("Failed to add token: %v", err)
	}

	if token.Name != "CICD Token" {
		t.Errorf("Expected token name 'CICD Token', got '%s'", token.Name)
	}

	if tokenValue == "" {
		t.Error("Expected non-empty token value")
	}

	// Test listing tokens
	tokens, err := store.ListTokens(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to list tokens: %v", err)
	}

	if len(tokens) != 2 { // default + CICD Token
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}

	// Test revoking token
	err = store.RevokeToken(ctx, "testuser", token.ID)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Check token is gone
	tokens, err = store.ListTokens(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to list tokens after revocation: %v", err)
	}

	if len(tokens) != 1 { // Should only have default token left
		t.Errorf("Expected 1 token after revocation, got %d", len(tokens))
	}
}

func TestStoreImpl_UserManagement(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_mgmt_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	store := NewStore(repo)
	ctx := context.Background()

	// Create user
	_, _, err = store.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test getting user by username
	user, err := store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}

	// Test listing users
	users, err := store.List(ctx)
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}

	// Test disable/enable
	err = store.Disable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to disable user: %v", err)
	}

	user, err = store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to get disabled user: %v", err)
	}

	if !user.Disabled {
		t.Error("Expected user to be disabled")
	}

	err = store.Enable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to enable user: %v", err)
	}

	user, err = store.GetByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to get enabled user: %v", err)
	}

	if user.Disabled {
		t.Error("Expected user to be enabled")
	}

	// Test deletion
	err = store.Delete(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	_, err = store.GetByUsername(ctx, "testuser")
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

func TestStoreImpl_Validation(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_validation_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	store := NewStore(repo)
	ctx := context.Background()

	// Test invalid username
	_, _, err = store.Create(ctx, "", "admin")
	if err == nil {
		t.Error("Expected error for empty username")
	}

	_, _, err = store.Create(ctx, "user@invalid", "admin")
	if err == nil {
		t.Error("Expected error for invalid username characters")
	}

	// Test invalid role
	_, _, err = store.Create(ctx, "testuser", "")
	if err == nil {
		t.Error("Expected error for empty role")
	}

	// Test invalid token name
	_, _, err = store.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create valid user: %v", err)
	}

	_, _, err = store.AddToken(ctx, "testuser", "")
	if err == nil {
		t.Error("Expected error for empty token name")
	}

	_, _, err = store.AddToken(ctx, "testuser", "token@invalid")
	if err == nil {
		t.Error("Expected error for invalid token name characters")
	}
}

func TestStoreImpl_DisabledUserOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "users_disabled_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repo := NewFileRepository(tempDir)
	store := NewStore(repo)
	ctx := context.Background()

	// Create and disable user
	_, _, err = store.Create(ctx, "testuser", "admin")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	err = store.Disable(ctx, "testuser")
	if err != nil {
		t.Fatalf("Failed to disable user: %v", err)
	}

	// Test that disabled user cannot get new tokens
	_, _, err = store.AddToken(ctx, "testuser", "New Token")
	if err == nil {
		t.Error("Expected error when adding token to disabled user")
	}

	// Test that disabled user cannot rotate token
	_, err = store.RotateToken(ctx, "testuser")
	if err == nil {
		t.Error("Expected error when rotating token for disabled user")
	}
}
