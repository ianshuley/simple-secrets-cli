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
package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirstRunProtection_BlocksWhenMasterKeyExists(t *testing.T) {
	// Setup temp environment
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	configDir := filepath.Join(tmpDir, ".simple-secrets")
	os.MkdirAll(configDir, 0700)

	// Create existing master.key to simulate partial installation
	masterKeyPath := filepath.Join(configDir, "master.key")
	os.WriteFile(masterKeyPath, []byte("fake-master-key"), 0600)

	// Attempt to trigger first-run (should be blocked)
	_, firstRun, _, err := LoadUsers()

	// Should fail with protection error
	if err == nil {
		t.Fatal("expected error when master.key exists but users.json missing")
	}

	if !strings.Contains(err.Error(), "existing simple-secrets installation detected") {
		t.Fatalf("expected protection error message, got: %v", err)
	}

	if !strings.Contains(err.Error(), "master.key") {
		t.Fatalf("expected error to mention master.key, got: %v", err)
	}

	if firstRun {
		t.Fatal("firstRun should be false when protection triggers")
	}
}

func TestFirstRunProtection_BlocksWhenSecretsJsonExists(t *testing.T) {
	// Setup temp environment
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	configDir := filepath.Join(tmpDir, ".simple-secrets")
	os.MkdirAll(configDir, 0700)

	// Create existing secrets.json to simulate partial installation
	secretsPath := filepath.Join(configDir, "secrets.json")
	os.WriteFile(secretsPath, []byte(`{"secrets":[]}`), 0600)

	// Attempt to trigger first-run (should be blocked)
	_, firstRun, _, err := LoadUsers()

	// Should fail with protection error
	if err == nil {
		t.Fatal("expected error when secrets.json exists but users.json missing")
	}

	if !strings.Contains(err.Error(), "existing simple-secrets installation detected") {
		t.Fatalf("expected protection error message, got: %v", err)
	}

	if !strings.Contains(err.Error(), "secrets.json") {
		t.Fatalf("expected error to mention secrets.json, got: %v", err)
	}

	if firstRun {
		t.Fatal("firstRun should be false when protection triggers")
	}
}

func TestFirstRunProtection_AllowsCleanFirstRun(t *testing.T) {
	// Setup temp environment
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	// No existing files - should allow first-run
	store, firstRun, _, err := LoadUsers()

	if err != nil {
		t.Fatalf("unexpected error on clean first-run: %v", err)
	}

	if !firstRun {
		t.Fatal("expected firstRun to be true on clean installation")
	}

	if store == nil {
		t.Fatal("expected store to be created on first-run")
	}

	// Verify admin user was created
	users := store.Users()
	if len(users) != 1 {
		t.Fatalf("expected 1 admin user, got %d", len(users))
	}

	if users[0].Role != RoleAdmin {
		t.Fatalf("expected admin role, got %v", users[0].Role)
	}
}

func TestFirstRunProtection_AllowsWhenOnlyConfigJsonExists(t *testing.T) {
	// Setup temp environment
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	configDir := filepath.Join(tmpDir, ".simple-secrets")
	os.MkdirAll(configDir, 0700)

	// Create only config.json (not a critical file for protection)
	configPath := filepath.Join(configDir, "config.json")
	os.WriteFile(configPath, []byte(`{"token":"test"}`), 0600)

	// Should still allow first-run since no critical files exist
	store, firstRun, _, err := LoadUsers()

	if err != nil {
		t.Fatalf("unexpected error when only config.json exists: %v", err)
	}

	if !firstRun {
		t.Fatal("expected firstRun to be true when only config.json exists")
	}

	if store == nil {
		t.Fatal("expected store to be created")
	}
}

func TestFirstRunProtection_AllowsNormalOperationWithAllFiles(t *testing.T) {
	// Setup temp environment
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmpDir+"/.simple-secrets")

	// Create complete installation first
	store, firstRun, _, err := LoadUsers()
	if err != nil {
		t.Fatalf("failed to create initial installation: %v", err)
	}
	if !firstRun {
		t.Fatal("expected first run to succeed")
	}

	// Now verify that normal operation works when all files exist
	store2, firstRun2, _, err2 := LoadUsers()

	if err2 != nil {
		t.Fatalf("unexpected error on normal operation: %v", err2)
	}

	if firstRun2 {
		t.Fatal("expected firstRun to be false when installation already exists")
	}

	if store2 == nil {
		t.Fatal("expected store to be loaded")
	}

	// Should have same number of users as original
	if len(store2.Users()) != len(store.Users()) {
		t.Fatalf("user count mismatch: original %d, reloaded %d", len(store.Users()), len(store2.Users()))
	}
}
