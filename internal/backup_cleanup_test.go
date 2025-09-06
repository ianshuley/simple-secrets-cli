package internal

import (
	"path/filepath"
	"testing"
	"time"
)

func TestListRotationBackups(t *testing.T) {
	s := newTempStore(t)

	// Initially should have no backups
	backups, err := s.ListRotationBackups()
	if err != nil {
		t.Fatalf("ListRotationBackups: %v", err)
	}
	if len(backups) != 0 {
		t.Fatalf("expected 0 backups, got %d", len(backups))
	}

	// Create some test secrets
	err = s.Put("test1", "value1")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Perform a rotation to create a backup
	err = s.RotateMasterKey("")
	if err != nil {
		t.Fatalf("RotateMasterKey: %v", err)
	}

	// Should now have one backup
	backups, err = s.ListRotationBackups()
	if err != nil {
		t.Fatalf("ListRotationBackups after rotation: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup after rotation, got %d", len(backups))
	}

	backup := backups[0]
	if !backup.IsValid {
		t.Fatalf("backup should be valid")
	}
	if backup.Name == "" {
		t.Fatalf("backup name should not be empty")
	}
	if backup.Timestamp.IsZero() {
		t.Fatalf("backup timestamp should not be zero")
	}
}

func TestRestoreFromBackup(t *testing.T) {
	s := newTempStore(t)

	// Create some initial secrets
	err := s.Put("original", "value1")
	if err != nil {
		t.Fatalf("Put original: %v", err)
	}

	// Rotate to create a backup
	err = s.RotateMasterKey("")
	if err != nil {
		t.Fatalf("RotateMasterKey: %v", err)
	}

	// Modify the secrets after rotation
	err = s.Put("original", "modified_value")
	if err != nil {
		t.Fatalf("Put modified: %v", err)
	}
	err = s.Put("new_secret", "new_value")
	if err != nil {
		t.Fatalf("Put new_secret: %v", err)
	}

	// Verify current state
	val, err := s.Get("original")
	if err != nil || val != "modified_value" {
		t.Fatalf("expected modified_value, got %q, err %v", val, err)
	}

	// Get backup info
	backups, err := s.ListRotationBackups()
	if err != nil || len(backups) == 0 {
		t.Fatalf("expected at least one backup, got %d, err %v", len(backups), err)
	}

	// Restore from the backup
	err = s.RestoreFromBackup(backups[0].Name)
	if err != nil {
		t.Fatalf("RestoreFromBackup: %v", err)
	}

	// Verify restoration worked
	val, err = s.Get("original")
	if err != nil || val != "value1" {
		t.Fatalf("after restore, expected value1, got %q, err %v", val, err)
	}

	// New secret should be gone
	_, err = s.Get("new_secret")
	if err == nil {
		t.Fatalf("new_secret should not exist after restore")
	}
}

func TestCleanupOldBackups(t *testing.T) {
	s := newTempStore(t)

	// First create some backups manually to test cleanup
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")

	// Create 7 manual backup directories
	for i := 0; i < 7; i++ {
		ts := time.Now().Add(-time.Duration(i) * time.Hour).Format("20060102-150405")
		backupDir := filepath.Join(backupRoot, "rotate-"+ts)
		err := s.backupCurrent(backupDir)
		if err != nil {
			t.Fatalf("backupCurrent %d: %v", i, err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Should have 7 backups
	backups, err := s.ListRotationBackups()
	if err != nil {
		t.Fatalf("ListRotationBackups: %v", err)
	}
	if len(backups) != 7 {
		t.Fatalf("expected 7 backups, got %d", len(backups))
	}

	// Test cleanup (should keep only 3)
	err = s.cleanupOldBackups(3)
	if err != nil {
		t.Fatalf("cleanupOldBackups: %v", err)
	}

	// Should now have only 3 backups
	backups, err = s.ListRotationBackups()
	if err != nil {
		t.Fatalf("ListRotationBackups after cleanup: %v", err)
	}
	if len(backups) != 3 {
		t.Fatalf("expected 3 backups after cleanup, got %d", len(backups))
	}
}

func TestRestoreFromMostRecentBackup(t *testing.T) {
	s := newTempStore(t)

	// Create initial state
	err := s.Put("test", "initial")
	if err != nil {
		t.Fatalf("Put initial: %v", err)
	}

	// Create a backup
	err = s.RotateMasterKey("")
	if err != nil {
		t.Fatalf("RotateMasterKey: %v", err)
	}

	// Modify current state
	err = s.Put("test", "modified")
	if err != nil {
		t.Fatalf("Put modified: %v", err)
	}

	// Restore from most recent backup (empty string means most recent)
	err = s.RestoreFromBackup("")
	if err != nil {
		t.Fatalf("RestoreFromBackup with empty name: %v", err)
	}

	// Should be back to initial state
	val, err := s.Get("test")
	if err != nil || val != "initial" {
		t.Fatalf("after restore, expected 'initial', got %q, err %v", val, err)
	}
}
