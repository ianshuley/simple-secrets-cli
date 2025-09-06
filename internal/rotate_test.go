package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotateMasterKey_PreservesPlaintextsAndBacksUp(t *testing.T) {
	s := newTempStore(t)

	want := map[string]string{
		"a": "1",
		"b": "2",
		"c": strings.Repeat("x", 1024),
	}

	for k, v := range want {
		err := s.Put(k, v)
		if err != nil {
			t.Fatalf("put %s: %v", k, err)
		}
	}

	oldKey, err := os.ReadFile(s.KeyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}

	// Rotate
	err = s.RotateMasterKey("") // default backup dir
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	// Key must change
	newKey, err := os.ReadFile(s.KeyPath)
	if err != nil {
		t.Fatalf("read new key: %v", err)
	}
	if string(newKey) == string(oldKey) {
		t.Fatalf("master key did not change")
	}

	// Plaintexts preserved
	for k, v := range want {
		got, err := s.Get(k)
		if err != nil {
			t.Fatalf("get %s: %v", k, err)
		}
		if got != v {
			t.Fatalf("after rotate, want %q got %q", v, got)
		}
	}

	// Backup exists with expected files
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		t.Fatalf("readdir backups: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("no backups found in %s", backupRoot)
	}

	// Find rotation backup directory (not individual backup files)
	var rotationDir string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "rotate-") {
			rotationDir = entry.Name()
			break
		}
	}
	if rotationDir == "" {
		t.Fatalf("no rotation backup directory found")
	}

	bdir := filepath.Join(backupRoot, rotationDir)

	if _, err := os.Stat(filepath.Join(bdir, "master.key")); err != nil {
		t.Fatalf("missing backup master.key: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bdir, "secrets.json")); err != nil {
		t.Fatalf("missing backup secrets.json: %v", err)
	}
}

func TestRotateMasterKey_FailsCleanlyOnCorruption(t *testing.T) {
	s := newTempStore(t)

	err := s.Put("good", "ok")
	if err != nil {
		t.Fatalf("put: %v", err)
	}

	// Manually corrupt on-disk secrets to simulate unreadable/undecodable entry
	onDisk := make(map[string]string)
	b, err := os.ReadFile(s.SecretsPath)
	if err != nil {
		t.Fatalf("read secrets: %v", err)
	}
	err = json.Unmarshal(b, &onDisk)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	onDisk["bad"] = "!!!not-base64!!!"

	bad, _ := json.MarshalIndent(onDisk, "", "  ")
	err = os.WriteFile(s.SecretsPath, bad, 0o600)
	if err != nil {
		t.Fatalf("write corrupt: %v", err)
	}

	// Reload secrets from disk to reflect corruption in memory
	err = s.loadSecrets()
	if err != nil {
		t.Fatalf("reload secrets after corruption: %v", err)
	}

	// Capture current key file to ensure no partial changes on failure
	oldKey, err := os.ReadFile(s.KeyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}

	err = s.RotateMasterKey("")
	if err == nil {
		t.Fatal("expected rotate to fail on corrupt secret")
	}

	// Ensure key not changed on failure
	newKey, err := os.ReadFile(s.KeyPath)
	if err != nil {
		t.Fatalf("read key after: %v", err)
	}
	if string(newKey) != string(oldKey) {
		t.Fatalf("master key should remain unchanged on failed rotate")
	}

	// Ensure secrets still decryptable where valid
	got, err := s.Get("good")
	if err != nil {
		t.Fatalf("get good after failure: %v", err)
	}
	if got != "ok" {
		t.Fatalf("want ok, got %q", got)
	}
}
