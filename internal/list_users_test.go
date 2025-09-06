package internal

import (
	"encoding/json"
	"os"
	"testing"
)

func TestListUsers(t *testing.T) {
	// Set up a temporary directory for the test
	dir := t.TempDir()
	usersPath := dir + "/users.json"

	// Create test users
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-token"), Role: RoleAdmin},
		{Username: "alice", TokenHash: HashToken("alice-token"), Role: RoleReader},
		{Username: "bob", TokenHash: HashToken("bob-token"), Role: RoleReader},
		{Username: "manager", TokenHash: HashToken("manager-token"), Role: RoleAdmin},
	}
	data, _ := json.Marshal(users)
	err := os.WriteFile(usersPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test users: %v", err)
	}

	// Load the users
	loadedUsers, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}

	// Verify we loaded all users
	if len(loadedUsers) != 4 {
		t.Fatalf("Expected 4 users, got %d", len(loadedUsers))
	}

	// Verify user details
	expectedUsers := map[string]Role{
		"admin":   RoleAdmin,
		"alice":   RoleReader,
		"bob":     RoleReader,
		"manager": RoleAdmin,
	}

	for _, user := range loadedUsers {
		expectedRole, exists := expectedUsers[user.Username]
		if !exists {
			t.Fatalf("Unexpected user: %s", user.Username)
		}
		if user.Role != expectedRole {
			t.Fatalf("User %s should have role %s, got %s", user.Username, expectedRole, user.Role)
		}
		if user.TokenHash == "" {
			t.Fatalf("User %s should have a token hash", user.Username)
		}
	}

	// Verify we have the right number of admins and readers
	adminCount := 0
	readerCount := 0
	for _, user := range loadedUsers {
		switch user.Role {
		case RoleAdmin:
			adminCount++
		case RoleReader:
			readerCount++
		}
	}

	if adminCount != 2 {
		t.Fatalf("Expected 2 admin users, got %d", adminCount)
	}
	if readerCount != 2 {
		t.Fatalf("Expected 2 reader users, got %d", readerCount)
	}
}

func TestListUsers_EmptyList(t *testing.T) {
	// Set up a temporary directory for the test
	dir := t.TempDir()
	usersPath := dir + "/users.json"

	// Create empty users list
	users := []*User{}
	data, _ := json.Marshal(users)
	err := os.WriteFile(usersPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write empty users: %v", err)
	}

	// Load the users (this should normally fail validation due to no admin,
	// but for testing purposes we're just checking the loading mechanism)
	loadedUsers, err := LoadUsersList(usersPath)

	// This should fail because there are no admin users
	if err == nil {
		t.Fatal("Expected error when loading users with no admin")
	}

	// Verify we don't get any users back when there's an error
	if loadedUsers != nil {
		t.Fatal("Should not get users back when there's an error")
	}
}

func TestListUsers_AuthenticationCheck(t *testing.T) {
	// Set up test users
	dir := t.TempDir()
	usersPath := dir + "/users.json"

	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-secret"), Role: RoleAdmin},
		{Username: "reader", TokenHash: HashToken("reader-secret"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	err := os.WriteFile(usersPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write users: %v", err)
	}

	loadedUsers, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}

	// Create a user store for authentication testing
	store := &UserStore{
		users: loadedUsers,
		permissions: RolePermissions{
			RoleAdmin:  {"read", "write", "rotate-tokens", "manage-users"},
			RoleReader: {"read"},
		},
	}

	// Test admin can access manage-users permission
	adminUser, err := store.Lookup("admin-secret")
	if err != nil {
		t.Fatalf("Admin should be able to authenticate: %v", err)
	}

	if !adminUser.Can("manage-users", store.Permissions()) {
		t.Fatal("Admin should have manage-users permission")
	}

	// Test reader cannot access manage-users permission
	readerUser, err := store.Lookup("reader-secret")
	if err != nil {
		t.Fatalf("Reader should be able to authenticate: %v", err)
	}

	if readerUser.Can("manage-users", store.Permissions()) {
		t.Fatal("Reader should NOT have manage-users permission")
	}
}
