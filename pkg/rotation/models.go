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

package rotation

import (
	"fmt"
	"strings"
	"time"
)

// BackupInfo represents metadata and information about a backup.
// Contains both filesystem information and validation status.
type BackupInfo struct {
	// Name is the backup identifier (e.g., "rotate-20240901-143022")
	Name string `json:"name"`

	// Path is the absolute filesystem path to the backup directory
	Path string `json:"path"`

	// Timestamp is when the backup was created
	Timestamp time.Time `json:"timestamp"`

	// Size is the total size of the backup in bytes
	Size int64 `json:"size"`

	// IsValid indicates whether the backup contains all required files
	IsValid bool `json:"is_valid"`

	// Type indicates the backup type (rotation, manual, pre-restore)
	Type BackupType `json:"type"`
}

// BackupType represents the different types of backups that can be created
type BackupType string

const (
	// BackupTypeRotation indicates a backup created during master key rotation
	BackupTypeRotation BackupType = "rotation"

	// BackupTypeManual indicates a manually created backup
	BackupTypeManual BackupType = "manual"

	// BackupTypePreRestore indicates a backup created before restoration
	BackupTypePreRestore BackupType = "pre-restore"
)

// String returns a formatted string representation of the backup
func (b *BackupInfo) String() string {
	if b.Name == "" || b.Timestamp.IsZero() {
		return "Invalid backup info"
	}
	return fmt.Sprintf("%s (%s, %s ago, %s)",
		b.Name,
		b.Timestamp.Format("2006-01-02 15:04:05"),
		b.Age(),
		b.Type)
}

// Age returns a human-readable age of the backup
func (b *BackupInfo) Age() string {
	duration := time.Since(b.Timestamp)

	if duration < time.Minute {
		return "less than a minute"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// IsExpired returns true if the backup is older than the specified duration
func (b *BackupInfo) IsExpired(maxAge time.Duration) bool {
	return time.Since(b.Timestamp) > maxAge
}

// GetBackupType determines the backup type from the backup name
func GetBackupType(backupName string) BackupType {
	if strings.HasPrefix(backupName, "rotate-") {
		return BackupTypeRotation
	}
	if strings.HasPrefix(backupName, "pre-restore-") {
		return BackupTypePreRestore
	}
	return BackupTypeManual
}

// NewBackupInfo creates a new BackupInfo with the specified parameters
func NewBackupInfo(name, path string, timestamp time.Time, size int64, isValid bool) *BackupInfo {
	return &BackupInfo{
		Name:      name,
		Path:      path,
		Timestamp: timestamp,
		Size:      size,
		IsValid:   isValid,
		Type:      GetBackupType(name),
	}
}

// GenerateBackupName creates a timestamped backup name with the specified prefix
func GenerateBackupName(prefix string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s", prefix, timestamp)
}

// ParseBackupTimestamp extracts the timestamp from a backup name
func ParseBackupTimestamp(backupName string) (time.Time, error) {
	// Extract timestamp part from names like "rotate-20240901-143022"
	parts := strings.Split(backupName, "-")
	if len(parts) < 3 {
		return time.Time{}, fmt.Errorf("invalid backup name format: %s", backupName)
	}

	// Reconstruct timestamp string from last two parts
	timestampStr := parts[len(parts)-2] + "-" + parts[len(parts)-1]
	timestamp, err := time.Parse("20060102-150405", timestampStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp from backup name %s: %w", backupName, err)
	}

	return timestamp, nil
}

// RotationConfig holds configuration for rotation operations
type RotationConfig struct {
	// BackupRetentionCount is the number of rotation backups to keep
	BackupRetentionCount int `json:"backup_retention_count"`

	// AutoCleanup enables automatic cleanup of old backups
	AutoCleanup bool `json:"auto_cleanup"`

	// BackupDir is the base directory for storing backups
	BackupDir string `json:"backup_dir"`
}

// DefaultRotationConfig returns the default rotation configuration
func DefaultRotationConfig() *RotationConfig {
	return &RotationConfig{
		BackupRetentionCount: 1, // Minimize attack surface by default
		AutoCleanup:          true,
		BackupDir:            "backups", // Relative to data directory
	}
}
