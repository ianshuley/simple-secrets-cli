package internal

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCreateUser_AdminAndReader(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admintok"), Role: RoleAdmin},
		{Username: "reader", TokenHash: HashToken("readtok"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	loaded, err := LoadUsersList(path)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 users, got %d", len(loaded))
	}
	if loaded[0].Username != "admin" || loaded[1].Username != "reader" {
		t.Fatalf("unexpected usernames: %+v", loaded)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admintok"), Role: RoleAdmin},
		{Username: "admin", TokenHash: HashToken("othertok"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	_, err := LoadUsersList(path)
	if err == nil {
		t.Fatal("expected error on duplicate username")
	}
}

func TestUserLoginWithToken(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/users.json"
	users := []*User{
		{Username: "admin", TokenHash: HashToken("admintok"), Role: RoleAdmin},
		{Username: "reader", TokenHash: HashToken("readtok"), Role: RoleReader},
	}
	data, _ := json.Marshal(users)
	os.WriteFile(path, data, 0600)
	loaded, err := LoadUsersList(path)
	if err != nil {
		t.Fatalf("LoadUsersList: %v", err)
	}
	store := &UserStore{users: loaded, permissions: RolePermissions{RoleAdmin: {"read", "write"}, RoleReader: {"read"}}}
	user, err := store.Lookup("admintok")
	if err != nil || user.Username != "admin" {
		t.Fatalf("admin login failed: %v", err)
	}
	user, err = store.Lookup("readtok")
	if err != nil || user.Username != "reader" {
		t.Fatalf("reader login failed: %v", err)
	}
}
