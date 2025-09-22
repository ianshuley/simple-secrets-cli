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
)

func TestServiceLayerIntegration(t *testing.T) {
	// Set up a temporary directory for testing
	tmpDir := t.TempDir()

	// Create test user and role files first
	usersPath := filepath.Join(tmpDir, "users.json")
	rolesPath := filepath.Join(tmpDir, "roles.json")

	testToken := "test-token"
	adminUser := &User{
		Username:  "admin",
		TokenHash: HashToken(testToken),
		Role:      RoleAdmin,
	}

	users := []*User{adminUser}
	permissions := createDefaultRoles()
	err := writeConfigFiles(usersPath, rolesPath, users, permissions)
	if err != nil {
		t.Fatalf("Failed to save test users: %v", err)
	}

	// Create a test service using functional options pattern
	service, err := NewService(
		WithStorageBackend(NewFilesystemBackend()),
		WithConfigDir(tmpDir),
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test that we can create the service without errors
	if service.Auth() == nil {
		t.Error("Auth operations not initialized")
	}
	if service.Secrets() == nil {
		t.Error("Secret operations not initialized")
	}
	if service.Users() == nil {
		t.Error("User operations not initialized")
	}
}

func TestServiceOperations(t *testing.T) {
	// Set up a clean test environment
	tmpDir := t.TempDir()

	// Create necessary directories
	err := os.MkdirAll(tmpDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// First, we need to ensure we have a first-run setup
	// Create a minimal setup to avoid first-run blocking
	usersPath := filepath.Join(tmpDir, "users.json")
	rolesPath := filepath.Join(tmpDir, "roles.json")
	// Don't create a dummy master key - let the SecretsStore create it properly

	// Create a test admin user
	testToken := "test-admin-token"
	adminUser := &User{
		Username:  "admin",
		TokenHash: HashToken(testToken),
		Role:      RoleAdmin,
	}

	// Save the user and roles using the internal functions
	users := []*User{adminUser}
	permissions := createDefaultRoles()
	err = writeConfigFiles(usersPath, rolesPath, users, permissions)
	if err != nil {
		t.Fatalf("Failed to save test users: %v", err)
	}

	// NOW create the service after user files exist
	service, err := NewService(
		WithStorageBackend(NewFilesystemBackend()),
		WithConfigDir(tmpDir),
	)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test putting a secret using focused operations
	err = service.Secrets().Put(testToken, "test-key", "test-value")
	if err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	// Test getting the secret
	value, err := service.Secrets().Get(testToken, "test-key")
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}

	// Test listing secrets
	keys, err := service.Secrets().List(testToken)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(keys) != 1 || keys[0] != "test-key" {
		t.Errorf("Expected ['test-key'], got %v", keys)
	}

	// Test authentication validation
	_, err = service.Auth().ValidateToken(testToken)
	if err != nil {
		t.Errorf("Admin token should be valid: %v", err)
	}

	// Test with invalid token
	_, err = service.Auth().ValidateToken("invalid-token")
	if err == nil {
		t.Error("Invalid token should be rejected")
	}
}
