package rotation

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultRotationConfig(t *testing.T) {
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

func TestBackupTypes(t *testing.T) {
	tests := []struct {
		name       string
		backupType BackupType
		expected   string
	}{
		{"rotation backup", BackupTypeRotation, "rotation"},
		{"manual backup", BackupTypeManual, "manual"},
		{"pre-restore backup", BackupTypePreRestore, "pre-restore"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.backupType) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(tt.backupType))
			}
		})
	}
}

func TestBackupInfo_String(t *testing.T) {
	tests := []struct {
		name            string
		backup          BackupInfo
		expectsContains []string
	}{
		{
			name: "valid backup",
			backup: BackupInfo{
				Name:      "secrets_20240101_120000.json",
				Path:      "/path/to/backup/secrets_20240101_120000.json",
				Size:      1024,
				Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				Type:      BackupTypeManual,
			},
			expectsContains: []string{"secrets_20240101_120000.json", "2024-01-01", "manual"},
		},
		{
			name: "invalid backup",
			backup: BackupInfo{
				Name: "invalid",
			},
			expectsContains: []string{"Invalid backup info"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.backup.String()
			for _, expected := range tt.expectsContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected string %q to contain %q", result, expected)
				}
			}
		})
	}
}

func TestBackupInfo_Age(t *testing.T) {
	now := time.Now()
	backup := BackupInfo{
		Timestamp: now.Add(-2 * time.Hour),
	}

	age := backup.Age()
	if age != "2 hours" {
		t.Errorf("Expected age='2 hours', got %q", age)
	}
}

func TestBackupInfo_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		timestamp time.Time
		maxAge    time.Duration
		expected  bool
	}{
		{"recent backup within limit", time.Now().Add(-30 * time.Minute), time.Hour, false},
		{"old backup beyond limit", time.Now().Add(-2 * time.Hour), time.Hour, true},
		{"exact boundary", time.Now().Add(-1 * time.Hour), time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := BackupInfo{Timestamp: tt.timestamp}
			if backup.IsExpired(tt.maxAge) != tt.expected {
				t.Errorf("Expected IsExpired(%v)=%v for %s", tt.maxAge, tt.expected, tt.name)
			}
		})
	}
}

func TestGetBackupType(t *testing.T) {
	tests := []struct {
		name       string
		backupName string
		expected   BackupType
	}{
		{"rotation backup", "rotate-20240101-120000", BackupTypeRotation},
		{"pre-restore backup", "pre-restore-20240101-120000", BackupTypePreRestore},
		{"manual backup", "manual-backup-20240101", BackupTypeManual},
		{"other backup", "some-other-backup", BackupTypeManual},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBackupType(tt.backupName)
			if result != tt.expected {
				t.Errorf("Expected GetBackupType(%q)=%q, got %q", tt.backupName, tt.expected, result)
			}
		})
	}
}

func TestParseBackupTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		backupName  string
		expectError bool
	}{
		{"valid rotate backup", "rotate-20240101-120000", false},
		{"valid pre-restore backup", "pre-restore-20240101-120000", false},
		{"invalid format", "invalid-backup", true},
		{"no timestamp", "backup", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBackupTimestamp(tt.backupName)
			hasError := err != nil
			if hasError != tt.expectError {
				t.Errorf("Expected error=%v, got error=%v (%v)", tt.expectError, hasError, err)
			}
		})
	}
}

func TestGenerateBackupName(t *testing.T) {
	name := GenerateBackupName("rotate")
	if !strings.Contains(name, "rotate-") {
		t.Errorf("Expected generated name to contain 'rotate-', got %q", name)
	}

	// Test that it parses back correctly
	_, err := ParseBackupTimestamp(name)
	if err != nil {
		t.Errorf("Generated backup name should be parseable, got error: %v", err)
	}
}
