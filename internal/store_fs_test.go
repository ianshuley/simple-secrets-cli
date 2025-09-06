package internal

import (
	"os"
	"testing"
)

func newTempStore(t *testing.T) *SecretsStore {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	s, err := LoadSecretsStore()
	if err != nil {
		t.Fatalf("LoadSecretsStore: %v", err)
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

	// No lingering temp file after save
	tmpPath := s.SecretsPath + ".tmp"
	_, err = os.Stat(tmpPath)
	if !os.IsNotExist(err) {
		t.Fatalf("temp file still exists: %s", tmpPath)
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
