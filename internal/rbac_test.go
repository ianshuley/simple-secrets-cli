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

func TestDefaultUserConfigPath(t *testing.T) {
	path, err := DefaultUserConfigPath("test.json")
	if err != nil {
		t.Fatalf("DefaultUserConfigPath: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestHashToken_Deterministic(t *testing.T) {
	h1 := HashToken("secret")
	h2 := HashToken("secret")
	if h1 != h2 {
		t.Fatal("HashToken should be deterministic")
	}
}

func TestLoadUsersList_DuplicateUsername(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	users := []*User{{Username: "admin", TokenHash: HashToken("a"), Role: RoleAdmin}, {Username: "admin", TokenHash: HashToken("b"), Role: RoleAdmin}}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	_, err := LoadUsersList(path)
	if err == nil {
		t.Fatal("expected error on duplicate usernames")
	}
}

func TestLoadUsersList_MultipleAdminsAllowed(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	// Multiple admin users with different usernames should be allowed
	users := []*User{
		{Username: "admin1", TokenHash: HashToken("token1"), Role: RoleAdmin},
		{Username: "admin2", TokenHash: HashToken("token2"), Role: RoleAdmin},
		{Username: "reader", TokenHash: HashToken("token3"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	loadedUsers, err := LoadUsersList(path)
	if err != nil {
		t.Fatalf("multiple admin users should be allowed: %v", err)
	}
	if len(loadedUsers) != 3 {
		t.Fatalf("expected 3 users, got %d", len(loadedUsers))
	}
	// Count admin users
	adminCount := 0
	for _, u := range loadedUsers {
		if u.Role == RoleAdmin {
			adminCount++
		}
	}
	if adminCount != 2 {
		t.Fatalf("expected 2 admin users, got %d", adminCount)
	}
}

func TestResolveToken_Order(t *testing.T) {
	os.Setenv("SIMPLE_SECRETS_TOKEN", "envtoken")
	defer os.Unsetenv("SIMPLE_SECRETS_TOKEN")
	// CLI flag wins
	tok, err := ResolveToken("flagtoken")
	if err != nil || tok != "flagtoken" {
		t.Fatalf("flag token should win: %v %v", tok, err)
	}
	// Env wins if no flag
	tok, err = ResolveToken("")
	if err != nil || tok != "envtoken" {
		t.Fatalf("env token should win: %v %v", tok, err)
	}
}

func TestRBAC_Permissions(t *testing.T) {
	perms := RolePermissions{
		RoleAdmin:  {"read", "write", "rotate-tokens", "manage-users"},
		RoleReader: {"read"},
	}
	admin := &User{Username: "a", Role: RoleAdmin}
	reader := &User{Username: "b", Role: RoleReader}
	if !admin.Can("write", perms) {
		t.Error("admin should have write")
	}
	if reader.Can("write", perms) {
		t.Error("reader should not have write")
	}
}

func TestFirstRunCreatesDefaultAdmin(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", dir+"/.simple-secrets")
	os.RemoveAll(dir + "/.simple-secrets")
	_, firstRun, err := LoadUsers()
	if err != nil {
		t.Fatalf("LoadUsers: %v", err)
	}
	if !firstRun {
		t.Error("expected firstRun true")
	}
	// Should have users.json and roles.json
	if _, err := os.Stat(dir + "/.simple-secrets/users.json"); err != nil {
		t.Error("users.json not created")
	}
	if _, err := os.Stat(dir + "/.simple-secrets/roles.json"); err != nil {
		t.Error("roles.json not created")
	}
}
