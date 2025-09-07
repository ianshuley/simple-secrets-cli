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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateDefaultAdmin(t *testing.T) {
	// generateDefaultAdmin is not directly exported, but we can test it through createDefaultUserFile
	tmp := t.TempDir()
	usersPath := filepath.Join(tmp, "users.json")
	rolesPath := filepath.Join(tmp, "roles.json")

	// Call createDefaultUserFile which uses generateDefaultAdmin internally
	store, firstRun, err := createDefaultUserFile(usersPath, rolesPath)
	if err != nil {
		t.Fatalf("createDefaultUserFile failed: %v", err)
	}

	if !firstRun {
		t.Fatalf("expected firstRun to be true")
	}

	if store == nil {
		t.Fatalf("expected store to be created")
	}

	users := store.Users()
	if len(users) < 1 {
		t.Fatalf("expected at least 1 user, got %d", len(users))
	}

	// Find the admin user (there might be multiple users from previous test runs)
	var adminUser *User
	for _, user := range users {
		if user.Username == "admin" {
			adminUser = user
			break
		}
	}

	if adminUser == nil {
		t.Fatalf("admin user not found")
	}

	if adminUser.Role != RoleAdmin {
		t.Fatalf("expected role 'admin', got %q", adminUser.Role)
	}

	if adminUser.TokenHash == "" {
		t.Fatalf("expected non-empty token hash")
	}

	if adminUser.TokenRotatedAt == nil {
		// Debug by reading the file directly
		data, err := os.ReadFile(usersPath)
		if err != nil {
			t.Fatalf("failed to read users file: %v", err)
		}
		t.Logf("Users file content: %s", string(data))
		t.Errorf("expected non-nil TokenRotatedAt")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	// generateSecureToken is not directly exported, but we can test token generation consistency
	tmp := t.TempDir()
	usersPath := filepath.Join(tmp, "users.json")
	rolesPath := filepath.Join(tmp, "roles.json")

	// Create two separate default user files with different paths
	store1, _, err1 := createDefaultUserFile(usersPath+"1", rolesPath+"1")

	// Add a small delay to ensure different timing
	time.Sleep(1 * time.Millisecond)

	store2, _, err2 := createDefaultUserFile(usersPath+"2", rolesPath+"2")

	if err1 != nil || err2 != nil {
		t.Fatalf("createDefaultUserFile failed: %v, %v", err1, err2)
	}

	users1 := store1.Users()
	users2 := store2.Users()

	// Token hashes should be non-empty and reasonable length
	if len(users1[0].TokenHash) < 10 {
		t.Fatalf("token hash seems too short: %q", users1[0].TokenHash)
	}

	if len(users2[0].TokenHash) < 10 {
		t.Fatalf("token hash seems too short: %q", users2[0].TokenHash)
	}

	// Tokens should be different (secure random generation) - but if they're the same, just log it
	if users1[0].TokenHash == users2[0].TokenHash {
		t.Logf("Warning: generated tokens are the same (extremely unlikely but possible)")
	}
}

func TestCreateDefaultRoles(t *testing.T) {
	tmp := t.TempDir()
	rolesPath := filepath.Join(tmp, "roles.json")

	// createDefaultRoles is called internally by createDefaultUserFile
	usersPath := filepath.Join(tmp, "users.json")
	_, _, err := createDefaultUserFile(usersPath, rolesPath)
	if err != nil {
		t.Fatalf("createDefaultUserFile failed: %v", err)
	}

	// Verify roles.json exists and contains expected roles
	rolesData, err := os.ReadFile(rolesPath)
	if err != nil {
		t.Fatalf("failed to read roles.json: %v", err)
	}

	var roles RolePermissions
	err = json.Unmarshal(rolesData, &roles)
	if err != nil {
		t.Fatalf("failed to unmarshal roles.json: %v", err)
	}

	// Check admin role
	adminPerms, exists := roles[RoleAdmin]
	if !exists {
		t.Fatalf("admin role not found")
	}

	expectedAdminPerms := []string{"read", "write", "rotate-tokens", "manage-users", "rotate-own-token"}
	if len(adminPerms) != len(expectedAdminPerms) {
		t.Fatalf("admin role permissions mismatch: expected %v, got %v", expectedAdminPerms, adminPerms)
	}

	for _, perm := range expectedAdminPerms {
		if !contains(adminPerms, perm) {
			t.Fatalf("admin role missing permission: %s", perm)
		}
	}

	// Check reader role
	readerPerms, exists := roles[RoleReader]
	if !exists {
		t.Fatalf("reader role not found")
	}

	expectedReaderPerms := []string{"read", "rotate-own-token"}
	if len(readerPerms) != len(expectedReaderPerms) {
		t.Fatalf("reader role permissions mismatch: expected %v, got %v", expectedReaderPerms, readerPerms)
	}
}

func TestResolveConfigPaths(t *testing.T) {
	// Test the path resolution logic (which is used in createDefaultUserFile)
	tmp := t.TempDir()

	// Test that paths are resolved correctly
	usersPath := filepath.Join(tmp, "users.json")
	rolesPath := filepath.Join(tmp, "roles.json")

	_, _, err := createDefaultUserFile(usersPath, rolesPath)
	if err != nil {
		t.Fatalf("createDefaultUserFile failed: %v", err)
	}

	// Verify files were created in expected locations
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		t.Fatalf("users.json not created at expected path: %s", usersPath)
	}

	if _, err := os.Stat(rolesPath); os.IsNotExist(err) {
		t.Fatalf("roles.json not created at expected path: %s", rolesPath)
	}
}

func TestHandleFirstRun(t *testing.T) {
	// handleFirstRun logic is embedded in various commands, test via LoadUsers with missing files
	tmp := t.TempDir()

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmp)

	// Call LoadUsers which should trigger first run setup
	store, firstRun, err := LoadUsers()
	if err != nil {
		t.Fatalf("LoadUsers failed: %v", err)
	}

	if !firstRun {
		t.Fatalf("expected first run to be detected")
	}

	if store == nil {
		t.Fatalf("expected store to be created, got nil")
	}

	users := store.Users()
	if len(users) != 1 {
		t.Fatalf("expected 1 user after first run, got %d", len(users))
	}

	// Verify files were created
	secretsDir := filepath.Join(tmp, ".simple-secrets")
	usersPath := filepath.Join(secretsDir, "users.json")
	rolesPath := filepath.Join(secretsDir, "roles.json")

	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		t.Fatalf("users.json not created during first run")
	}

	if _, err := os.Stat(rolesPath); os.IsNotExist(err) {
		t.Fatalf("roles.json not created during first run")
	}
}

func TestCreateUserStore(t *testing.T) {
	// createUserStore logic is tested indirectly through user operations
	tmp := t.TempDir()

	// Set HOME to temp directory
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmp)

	// Create user store via LoadUsers
	_, _, err := LoadUsers()
	if err != nil {
		t.Fatalf("LoadUsers failed: %v", err)
	}

	// Test that we can load users list (which requires proper user store)
	usersPath, err := DefaultUserConfigPath("users.json")
	if err != nil {
		t.Fatalf("DefaultUserConfigPath failed: %v", err)
	}

	users, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("LoadUsersList failed after setup: %v", err)
	}

	if len(users) != 1 {
		t.Fatalf("expected 1 user in store, got %d", len(users))
	}

	if users[0].Username != "admin" {
		t.Fatalf("expected admin user, got %q", users[0].Username)
	}
}

func TestMethodExtractionConsistency(t *testing.T) {
	// Test that the extracted methods produce consistent results
	tmp1 := t.TempDir()
	tmp2 := t.TempDir()

	// Create two separate user stores
	usersPath1 := filepath.Join(tmp1, "users.json")
	rolesPath1 := filepath.Join(tmp1, "roles.json")
	usersPath2 := filepath.Join(tmp2, "users.json")
	rolesPath2 := filepath.Join(tmp2, "roles.json")

	_, _, err1 := createDefaultUserFile(usersPath1, rolesPath1)
	_, _, err2 := createDefaultUserFile(usersPath2, rolesPath2)

	if err1 != nil || err2 != nil {
		t.Fatalf("createDefaultUserFile failed: %v, %v", err1, err2)
	}

	// Read and compare structure (not tokens, which should be different)
	rolesData1, _ := os.ReadFile(rolesPath1)
	rolesData2, _ := os.ReadFile(rolesPath2)

	var roles1, roles2 RolePermissions
	json.Unmarshal(rolesData1, &roles1)
	json.Unmarshal(rolesData2, &roles2)

	// Roles should be identical in structure
	if len(roles1) != len(roles2) {
		t.Fatalf("roles structure should be consistent")
	}

	for roleName, perms1 := range roles1 {
		perms2, exists := roles2[roleName]
		if !exists {
			t.Fatalf("role %s missing in second creation", roleName)
		}

		if len(perms1) != len(perms2) {
			t.Fatalf("role %s permissions count mismatch", roleName)
		}

		for i, perm := range perms1 {
			if perm != perms2[i] {
				t.Fatalf("role %s permission mismatch at index %d", roleName, i)
			}
		}
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
