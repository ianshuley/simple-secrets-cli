package rotation

import "testing"

func TestBasicRotationModels(t *testing.T) {
	config := DefaultRotationConfig()
	if config.BackupRetentionCount <= 0 {
		t.Error("BackupRetentionCount should be positive")
	}
	if !config.AutoCleanup {
		t.Error("AutoCleanup should be enabled by default")
	}
	if config.BackupDir == "" {
		t.Error("BackupDir should not be empty")
	}
}
