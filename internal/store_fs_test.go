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
	"strings"
	"testing"
)

func newTempStore(t *testing.T) *SecretsStore {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tmp+"/.simple-secrets")
	// Use dependency injection for better testability
	s, err := LoadSecretsStoreWithBackend(NewFilesystemBackend())
	if err != nil {
		t.Fatalf("LoadSecretsStoreWithBackend: %v", err)
	}
	return s
}

func TestStore_PutGetListDelete_PersistsAcrossRestarts(t *testing.T) {
	s := newTempStore(t)

	err := s.Put("db/user", "alice")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	err = s.Put("db/pass", "p@ss")
	if err != nil {
		t.Fatalf("put2: %v", err)
	}

	val, err := s.Get("db/user")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "alice" {
		t.Fatalf("want alice, got %q", val)
	}

	keys := s.ListKeys()
	if len(keys) != 2 {
		t.Fatalf("want 2 keys, got %d", len(keys))
	}

	// Reload store to ensure on-disk read works
	s2, err := LoadSecretsStore()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	val2, err := s2.Get("db/pass")
	if err != nil {
		t.Fatalf("get2: %v", err)
	}
	if val2 != "p@ss" {
		t.Fatalf("want p@ss, got %q", val2)
	}

	// Delete and confirm
	err = s2.Delete("db/user")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = s2.Get("db/user")
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestStore_FilePermissionsAndAtomicity(t *testing.T) {
	s := newTempStore(t)
	err := s.Put("k", "v")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	// Key file perms
	st, err := os.Stat(s.KeyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("master.key perms = %v, want 0600", st.Mode().Perm())
	}

	// Secrets perms
	st2, err := os.Stat(s.SecretsPath)
	if err != nil {
		t.Fatalf("stat secrets: %v", err)
	}
	if st2.Mode().Perm() != 0o600 {
		t.Fatalf("secrets.json perms = %v, want 0600", st2.Mode().Perm())
	}

	// No lingering temp files after save (check for any .tmp files with process ID)
	files, err := os.ReadDir(filepath.Dir(s.SecretsPath))
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), ".tmp.") {
			t.Fatalf("temp file still exists: %s", file.Name())
		}
	}

	// Ensure file contents actually changed after an update (atomicity/content check)
	beforeContent, err := os.ReadFile(s.SecretsPath)
	if err != nil {
		t.Fatalf("read secrets before: %v", err)
	}
	err = s.Put("k2", "v2")
	if err != nil {
		t.Fatalf("put2: %v", err)
	}
	afterContent, err := os.ReadFile(s.SecretsPath)
	if err != nil {
		t.Fatalf("read secrets after: %v", err)
	}
	if string(beforeContent) == string(afterContent) {
		t.Fatalf("expected file contents to change after update")
	}
}

func TestStore_DisableEnableSecret(t *testing.T) {
	s := newTempStore(t)

	// Setup test secrets
	err := s.Put("test-secret", "test-value")
	if err != nil {
		t.Fatalf("put test secret: %v", err)
	}

	err = s.Put("other-secret", "other-value")
	if err != nil {
		t.Fatalf("put other secret: %v", err)
	}

	// Verify initial state
	keys := s.ListKeys()
	if len(keys) != 2 || keys[0] != "other-secret" || keys[1] != "test-secret" {
		t.Fatalf("expected 2 keys [other-secret, test-secret], got %v", keys)
	}

	// Disable secret
	err = s.DisableSecret("test-secret")
	if err != nil {
		t.Fatalf("disable secret: %v", err)
	}

	// Verify disabled secret is hidden from normal list
	keys = s.ListKeys()
	if len(keys) != 1 || keys[0] != "other-secret" {
		t.Fatalf("expected 1 key [other-secret] after disable, got %v", keys)
	}

	// Verify disabled secret appears in disabled list
	disabled := s.ListDisabledSecrets()
	if len(disabled) != 1 || disabled[0] != "test-secret" {
		t.Fatalf("expected 1 disabled secret [test-secret], got %v", disabled)
	}

	// Verify disabled secret cannot be retrieved normally
	_, err = s.Get("test-secret")
	if err == nil {
		t.Fatalf("expected error getting disabled secret")
	}

	// Enable secret
	err = s.EnableSecret("test-secret")
	if err != nil {
		t.Fatalf("enable secret: %v", err)
	}

	// Verify enabled secret is back in normal list
	keys = s.ListKeys()
	if len(keys) != 2 || keys[0] != "other-secret" || keys[1] != "test-secret" {
		t.Fatalf("expected 2 keys [other-secret, test-secret] after enable, got %v", keys)
	}

	// Verify no disabled secrets
	disabled = s.ListDisabledSecrets()
	if len(disabled) != 0 {
		t.Fatalf("expected 0 disabled secrets after enable, got %v", disabled)
	}

	// Verify secret value is preserved
	val, err := s.Get("test-secret")
	if err != nil {
		t.Fatalf("get enabled secret: %v", err)
	}
	if val != "test-value" {
		t.Fatalf("expected 'test-value', got %q", val)
	}
}

func TestStore_DisableNonexistentSecret(t *testing.T) {
	s := newTempStore(t)

	err := s.DisableSecret("nonexistent")
	if err == nil {
		t.Fatalf("expected error disabling nonexistent secret")
	}
}

func TestStore_EnableNonexistentSecret(t *testing.T) {
	s := newTempStore(t)

	err := s.EnableSecret("nonexistent")
	if err == nil {
		t.Fatalf("expected error enabling nonexistent secret")
	}
}

func TestStore_MultipleDisableEnableCycles(t *testing.T) {
	s := newTempStore(t)

	// Setup
	err := s.Put("cycle-test", "cycle-value")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	// Multiple disable/enable cycles
	for i := 0; i < 3; i++ {
		// Disable
		err = s.DisableSecret("cycle-test")
		if err != nil {
			t.Fatalf("disable cycle %d: %v", i, err)
		}

		// Verify disabled
		disabled := s.ListDisabledSecrets()
		if len(disabled) != 1 || disabled[0] != "cycle-test" {
			t.Fatalf("cycle %d: expected [cycle-test] disabled, got %v", i, disabled)
		}

		// Enable
		err = s.EnableSecret("cycle-test")
		if err != nil {
			t.Fatalf("enable cycle %d: %v", i, err)
		}

		// Verify enabled and value preserved
		val, err := s.Get("cycle-test")
		if err != nil {
			t.Fatalf("get cycle %d: %v", i, err)
		}
		if val != "cycle-value" {
			t.Fatalf("cycle %d: value not preserved, expected 'cycle-value', got %q", i, val)
		}
	}
}
