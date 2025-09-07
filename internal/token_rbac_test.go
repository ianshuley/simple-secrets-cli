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
	"testing"
)

func TestTokenResolutionOrder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", dir+"/.simple-secrets")
	os.Unsetenv("SIMPLE_SECRETS_TOKEN")

	// Write config file
	configPath := dir + "/.simple-secrets/config.json"
	os.MkdirAll(dir+"/.simple-secrets", 0700)
	os.WriteFile(configPath, []byte(`{"token":"fromconfig"}`), 0600)

	// Env var wins over config
	t.Setenv("SIMPLE_SECRETS_TOKEN", "fromenv")
	tok, err := ResolveToken("")
	if err != nil || tok != "fromenv" {
		t.Fatalf("env should win: got %q, err %v", tok, err)
	}

	// Unset for next test within same function
	os.Unsetenv("SIMPLE_SECRETS_TOKEN")

	// Config wins if no env
	tok, err = ResolveToken("")
	if err != nil || tok != "fromconfig" {
		t.Fatalf("config should win: got %q, err %v", tok, err)
	}

	// CLI flag wins over all
	tok, err = ResolveToken("fromflag")
	if err != nil || tok != "fromflag" {
		t.Fatalf("flag should win: got %q, err %v", tok, err)
	}
}

func TestTokenResolutionErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", dir+"/.simple-secrets")
	os.Unsetenv("SIMPLE_SECRETS_TOKEN")
	// No config, no env, no flag
	_, err := ResolveToken("")
	if err == nil {
		t.Fatal("expected error when no token present")
	}
	// Malformed config
	os.MkdirAll(dir+"/.simple-secrets", 0700)
	os.WriteFile(dir+"/.simple-secrets/config.json", []byte("not-json"), 0600)
	_, err = ResolveToken("")
	if err == nil {
		t.Fatal("expected error on malformed config.json")
	}
}

func TestRBACEnforcement(t *testing.T) {
	perms := RolePermissions{
		RoleAdmin:  {"read", "write", "rotate-tokens", "manage-users"},
		RoleReader: {"read"},
	}
	admin := &User{Username: "admin", Role: RoleAdmin}
	reader := &User{Username: "bob", Role: RoleReader}
	if !admin.Can("write", perms) {
		t.Error("admin should have write")
	}
	if reader.Can("write", perms) {
		t.Error("reader should not have write")
	}
	if !admin.Can("manage-users", perms) {
		t.Error("admin should have manage-users")
	}
	if reader.Can("manage-users", perms) {
		t.Error("reader should not have manage-users")
	}
}
