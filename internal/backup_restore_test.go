package internal

import (
	"os"
	"testing"
)

func TestBackupAndRestoreSecret(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	s := newTempStore(t)
	key := "mykey"
	val1 := "secret1"
	val2 := "secret2"

	// Store initial value
	if err := s.Put(key, val1); err != nil {
		t.Fatalf("put1: %v", err)
	}

	// Overwrite value (should create backup)
	if err := s.Put(key, val2); err != nil {
		t.Fatalf("put2: %v", err)
	}

	home, _ := os.UserHomeDir()
	backupPath := home + "/.simple-secrets/backups/" + key + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	// Backup should now contain encrypted data, not plaintext
	if string(backupData) == val1 {
		t.Fatalf("backup should NOT contain plaintext, but it does")
	}

	// Verify we can decrypt the backup to get the original value
	decrypted, err := s.DecryptBackup(string(backupData))
	if err != nil {
		t.Fatalf("decrypt backup: %v", err)
	}
	if decrypted != val1 {
		t.Fatalf("backup should decrypt to old value %q, got %q", val1, decrypted)
	}

	// Simulate restore (decrypt backup and put it back)
	if err := s.Put(key, decrypted); err != nil {
		t.Fatalf("restore put: %v", err)
	}
	got, err := s.Get(key)
	if err != nil {
		t.Fatalf("get after restore: %v", err)
	}
	if got != val1 {
		t.Fatalf("expected restored value %q, got %q", val1, got)
	}
}

func TestBackupOnDelete(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	s := newTempStore(t)
	key := "delkey"
	val := "todelete"
	if err := s.Put(key, val); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := s.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	home, _ := os.UserHomeDir()
	backupPath := home + "/.simple-secrets/backups/" + key + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("expected backup file after delete: %v", err)
	}
	backupData, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	// Backup should now contain encrypted data, not plaintext
	if string(backupData) == val {
		t.Fatalf("backup should NOT contain plaintext, but it does")
	}

	// Verify we can decrypt the backup to get the deleted value
	decrypted, err := s.DecryptBackup(string(backupData))
	if err != nil {
		t.Fatalf("decrypt backup: %v", err)
	}
	if decrypted != val {
		t.Fatalf("backup should decrypt to deleted value %q, got %q", val, decrypted)
	}
}
