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
	"context"
)

// Service defines the public interface for rotation and backup operations.
// This interface is designed for platform extension and provides all rotation-related functionality.
type Service interface {
	// MasterKeyRotator provides master key rotation operations
	MasterKeyRotator

	// BackupManager provides backup and restore operations
	BackupManager

	// TokenRotator provides token rotation operations
	TokenRotator
}

// MasterKeyRotator handles master encryption key rotation operations.
// Master key rotation re-encrypts all secrets with a new key and creates backups.
type MasterKeyRotator interface {
	// RotateMasterKey creates a backup, generates a new master key, and re-encrypts all secrets
	// If backupDir is empty, a timestamped backup directory will be created automatically
	RotateMasterKey(ctx context.Context, backupDir string) error

	// ReencryptBackups re-encrypts existing backup files with a new master key
	ReencryptBackups(ctx context.Context, oldKey, newKey []byte) error
}

// BackupManager handles backup creation, listing, and restoration operations.
// Supports both automatic rotation backups and manual backup operations.
type BackupManager interface {
	// CreateBackup creates a manual backup of the current state
	CreateBackup(ctx context.Context, backupDir string) error

	// ListBackups returns information about all available backups
	ListBackups(ctx context.Context) ([]*BackupInfo, error)

	// RestoreFromBackup restores the system from a specified backup
	// If backupName is empty, uses the most recent valid backup
	RestoreFromBackup(ctx context.Context, backupName string) error

	// ValidateBackup checks if a backup is valid and contains all required files
	ValidateBackup(ctx context.Context, backupPath string) error

	// CleanupOldBackups removes old backup directories, keeping only the specified number
	CleanupOldBackups(ctx context.Context, keepCount int) error
}

// TokenRotator handles user authentication token rotation operations.
// Provides both self-rotation and admin-managed token rotation.
type TokenRotator interface {
	// RotateUserToken generates a new authentication token for a specific user
	// Requires appropriate permissions to rotate other users' tokens
	RotateUserToken(ctx context.Context, username string) (newToken string, err error)

	// RotateSelfToken generates a new token for the currently authenticated user
	RotateSelfToken(ctx context.Context, currentUsername string) (newToken string, err error)
}

// Repository defines the storage interface for rotation domain persistence.
// Different implementations can use files, databases, cloud storage, etc.
type Repository interface {
	// StoreBackup persists backup metadata and files
	StoreBackup(ctx context.Context, backup *BackupInfo) error

	// RetrieveBackup gets backup information by name
	RetrieveBackup(ctx context.Context, backupName string) (*BackupInfo, error)

	// ListBackups returns all available backups
	ListBackups(ctx context.Context) ([]*BackupInfo, error)

	// DeleteBackup removes a backup permanently
	DeleteBackup(ctx context.Context, backupName string) error

	// ValidateBackupIntegrity checks if backup files are present and valid
	ValidateBackupIntegrity(ctx context.Context, backupPath string) error
}
