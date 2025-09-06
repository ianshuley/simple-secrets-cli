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
