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

func TestCorruptedUsersJson(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadUsersList(path)
	if err == nil || err.Error() == "" {
		t.Fatal("expected error on corrupted users.json")
	}
}

func TestMissingUsersJson(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	_, err := LoadUsersList(path)
	if err == nil {
		t.Fatal("expected error on missing users.json")
	}
}

func TestNoAdminInUsersJson(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	users := []*User{{Username: "bob", TokenHash: HashToken("t"), Role: RoleReader}}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	_, err := LoadUsersList(path)
	if err == nil || err.Error() == "" {
		t.Fatal("expected error on users.json with no admin")
	}
}

func TestCorruptedRolesJson(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/roles.json"
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadRoles(path)
	if err == nil || err.Error() == "" {
		t.Fatal("expected error on corrupted roles.json")
	}
}

func TestMissingRolesJson(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/roles.json"
	_, err := loadRoles(path)
	if err == nil {
		t.Fatal("expected error on missing roles.json")
	}
}
