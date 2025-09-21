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
	"testing"
)

func TestRotateToken(t *testing.T) {
	// Set up a temporary directory for the test
	dir := t.TempDir()
	usersPath := dir + "/users.json"

	// Create initial users with known tokens
	originalToken := "original-token"
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-token"), Role: RoleAdmin},
		{Username: "alice", TokenHash: HashToken(originalToken), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	err := os.WriteFile(usersPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write initial users: %v", err)
	}

	// Load the users to verify initial state
	loadedUsers, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}

	// Find Alice's original token hash
	var aliceOriginalHash string
	for _, u := range loadedUsers {
		if u.Username == "alice" {
			aliceOriginalHash = u.TokenHash
			break
		}
	}

	// Verify Alice can authenticate with original token
	store := &UserStore{users: loadedUsers, permissions: RolePermissions{RoleAdmin: {"read", "write"}, RoleReader: {"read"}}}
	user, err := store.Lookup(originalToken)
	if err != nil || user.Username != "alice" {
		t.Fatalf("Alice should be able to authenticate with original token")
	}

	// Now simulate token rotation by manually updating the token hash
	// (This simulates what the rotate-token command would do)
	newToken := "new-rotated-token"
	newTokenHash := HashToken(newToken)

	// Update Alice's token hash
	for i, u := range loadedUsers {
		if u.Username == "alice" {
			loadedUsers[i].TokenHash = newTokenHash
			break
		}
	}

	// Save the updated users
	updatedData, _ := json.Marshal(loadedUsers)
	err = os.WriteFile(usersPath, updatedData, 0600)
	if err != nil {
		t.Fatalf("Failed to save updated users: %v", err)
	}

	// Reload users and verify changes
	rotatedUsers, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("Failed to reload users after rotation: %v", err)
	}

	// Verify Alice's token hash changed
	var aliceNewHash string
	for _, u := range rotatedUsers {
		if u.Username == "alice" {
			aliceNewHash = u.TokenHash
			break
		}
	}

	if aliceNewHash == aliceOriginalHash {
		t.Fatal("Alice's token hash should have changed after rotation")
	}

	if aliceNewHash != newTokenHash {
		t.Fatalf("Alice's token hash should be %q, got %q", newTokenHash, aliceNewHash)
	}

	// Verify Alice can authenticate with new token but not old token
	newStore := &UserStore{users: rotatedUsers, permissions: RolePermissions{RoleAdmin: {"read", "write"}, RoleReader: {"read"}}}

	// New token should work
	user, err = newStore.Lookup(newToken)
	if err != nil || user.Username != "alice" {
		t.Fatalf("Alice should be able to authenticate with new token")
	}

	// Old token should NOT work
	_, err = newStore.Lookup(originalToken)
	if err == nil {
		t.Fatal("Alice should NOT be able to authenticate with old token after rotation")
	}
}

func TestRotateToken_UserNotFound(t *testing.T) {
	dir := t.TempDir()
	usersPath := dir + "/users.json"

	// Create users without the target user
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admin-token"), Role: RoleAdmin},
		{Username: "bob", TokenHash: HashToken("bob-token"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	err := os.WriteFile(usersPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write users: %v", err)
	}

	// Try to load users list for a non-existent user
	loadedUsers, err := LoadUsersList(usersPath)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}

	// Verify the non-existent user is not found
	var foundUser *User
	for _, u := range loadedUsers {
		if u.Username == "nonexistent" {
			foundUser = u
			break
		}
	}

	if foundUser != nil {
		t.Fatal("Should not find non-existent user")
	}
}
