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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"simple-secrets/pkg/crypto"
	"simple-secrets/pkg/rotation"
	"simple-secrets/pkg/secrets"
	"simple-secrets/pkg/users"
)

// ServiceImpl implements the rotation.Service interface
type ServiceImpl struct {
	secretsStore secrets.Store
	usersStore   users.Store
	config       *rotation.RotationConfig
	dataDir      string
}

// NewService creates a new rotation service with the provided dependencies
func NewService(secretsStore secrets.Store, usersStore users.Store, dataDir string) rotation.Service {
	return &ServiceImpl{
		secretsStore: secretsStore,
		usersStore:   usersStore,
		config:       rotation.DefaultRotationConfig(),
		dataDir:      dataDir,
	}
}

// NewServiceWithConfig creates a new rotation service with custom configuration
func NewServiceWithConfig(secretsStore secrets.Store, usersStore users.Store, dataDir string, config *rotation.RotationConfig) rotation.Service {
	return &ServiceImpl{
		secretsStore: secretsStore,
		usersStore:   usersStore,
		config:       config,
		dataDir:      dataDir,
	}
}

// RotateMasterKey creates a backup, generates a new master key, and re-encrypts all secrets
func (s *ServiceImpl) RotateMasterKey(ctx context.Context, backupDir string) error {
	// Generate backup directory name if not provided
	if backupDir == "" {
		backupDir = filepath.Join(s.dataDir, s.config.BackupDir, rotation.GenerateBackupName("rotate"))
	}

	// Create backup before rotation
	err := s.CreateBackup(ctx, backupDir)
	if err != nil {
		return fmt.Errorf("failed to create backup before rotation: %w", err)
	}

	// Perform master key rotation using the secrets store
	// This will generate a new key and re-encrypt all secrets
	err = s.secretsStore.RotateMasterKey(ctx, backupDir)
	if err != nil {
		return fmt.Errorf("secrets store master key rotation failed: %w", err)
	}

	return nil
}

// ReencryptBackups re-encrypts existing backup files with a new master key
func (s *ServiceImpl) ReencryptBackups(ctx context.Context, oldKey, newKey []byte) error {
	backupRoot := filepath.Join(s.dataDir, s.config.BackupDir)
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return nil // No backups to re-encrypt
	}

	// Find all individual secret backup files (*.bak)
	err := filepath.Walk(backupRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.bak files
		if info.IsDir() || filepath.Ext(path) != ".bak" {
			return nil
		}

		// Read the backup file (should contain encrypted data)
		encryptedData, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read backup file %s: %w", path, err)
		}

		// Decrypt with old key
		plaintext, err := crypto.Decrypt(oldKey, string(encryptedData))
		if err != nil {
			// If we can't decrypt with old key, it might already be encrypted with new key
			// or it might be corrupted. Skip this file.
			fmt.Printf("Warning: failed to decrypt backup file %s with old key, skipping: %v\n", path, err)
			return nil
		}

		// Re-encrypt with new key
		newEncrypted, err := crypto.Encrypt(newKey, plaintext)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt backup file %s: %w", path, err)
		}

		// Write back to the same file
		if err := os.WriteFile(path, []byte(newEncrypted), 0600); err != nil {
			return fmt.Errorf("failed to write re-encrypted backup file %s: %w", path, err)
		}

		return nil
	})

	return err
}

// CreateBackup creates a manual backup of the current state
func (s *ServiceImpl) CreateBackup(ctx context.Context, backupDir string) error {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Define paths for key and secrets files
	keyPath := filepath.Join(s.dataDir, "master.key")
	secretsPath := filepath.Join(s.dataDir, "secrets.json")
	usersPath := filepath.Join(s.dataDir, "users.json")

	backupKeyPath := filepath.Join(backupDir, "master.key")
	backupSecretsPath := filepath.Join(backupDir, "secrets.json")
	backupUsersPath := filepath.Join(backupDir, "users.json")

	// Copy files securely
	if err := s.copyFileSecurely(keyPath, backupKeyPath); err != nil {
		return fmt.Errorf("failed to backup master key: %w", err)
	}

	if err := s.copyFileSecurely(secretsPath, backupSecretsPath); err != nil {
		return fmt.Errorf("failed to backup secrets: %w", err)
	}

	// Users file is optional (might not exist in all setups)
	if _, err := os.Stat(usersPath); err == nil {
		if err := s.copyFileSecurely(usersPath, backupUsersPath); err != nil {
			return fmt.Errorf("failed to backup users: %w", err)
		}
	}

	return nil
}

// ListBackups returns information about all available backups
func (s *ServiceImpl) ListBackups(ctx context.Context) ([]*rotation.BackupInfo, error) {
	backupRoot := filepath.Join(s.dataDir, s.config.BackupDir)

	// Check if backup directory exists
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return []*rotation.BackupInfo{}, nil // No backups available
	}

	// Get all backup directories
	backupDirs, err := s.scanBackupDirectories(backupRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to scan backup directories: %w", err)
	}

	var backups []*rotation.BackupInfo
	for _, dirName := range backupDirs {
		backupPath := filepath.Join(backupRoot, dirName)

		// Get directory info
		dirInfo, err := os.Stat(backupPath)
		if err != nil {
			continue // Skip if we can't stat the directory
		}

		// Parse timestamp from directory name
		timestamp, err := rotation.ParseBackupTimestamp(dirName)
		if err != nil {
			// If we can't parse timestamp, use directory modification time
			timestamp = dirInfo.ModTime()
		}

		// Calculate directory size
		size, err := s.calculateDirectorySize(backupPath)
		if err != nil {
			size = 0 // Continue even if we can't calculate size
		}

		// Validate backup integrity
		isValid := s.validateBackupIntegrity(backupPath)

		backup := rotation.NewBackupInfo(dirName, backupPath, timestamp, size, isValid)
		backups = append(backups, backup)
	}

	// Sort backups by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreFromBackup restores the system from a specified backup
func (s *ServiceImpl) RestoreFromBackup(ctx context.Context, backupName string) error {
	backupPath, err := s.determineBackupPath(ctx, backupName)
	if err != nil {
		return err
	}

	// Create a backup of current state before restoring
	currentBackupDir := filepath.Join(s.dataDir, s.config.BackupDir, rotation.GenerateBackupName("pre-restore"))
	if err := s.CreateBackup(ctx, currentBackupDir); err != nil {
		return fmt.Errorf("failed to backup current state: %w", err)
	}

	// Define file paths
	keyPath := filepath.Join(s.dataDir, "master.key")
	secretsPath := filepath.Join(s.dataDir, "secrets.json")
	usersPath := filepath.Join(s.dataDir, "users.json")

	backupKeyPath := filepath.Join(backupPath, "master.key")
	backupSecretsPath := filepath.Join(backupPath, "secrets.json")
	backupUsersPath := filepath.Join(backupPath, "users.json")

	// Restore files
	if err := s.copyFileSecurely(backupKeyPath, keyPath); err != nil {
		return fmt.Errorf("failed to restore master key: %w", err)
	}

	if err := s.copyFileSecurely(backupSecretsPath, secretsPath); err != nil {
		return fmt.Errorf("failed to restore secrets: %w", err)
	}

	// Restore users file if it exists in backup
	if _, err := os.Stat(backupUsersPath); err == nil {
		if err := s.copyFileSecurely(backupUsersPath, usersPath); err != nil {
			return fmt.Errorf("failed to restore users: %w", err)
		}
	}

	return nil
}

// ValidateBackup checks if a backup is valid and contains all required files
func (s *ServiceImpl) ValidateBackup(ctx context.Context, backupPath string) error {
	if !s.validateBackupIntegrity(backupPath) {
		return fmt.Errorf("backup validation failed: missing required files in %s", backupPath)
	}
	return nil
}

// CleanupOldBackups removes old backup directories, keeping only the specified number
func (s *ServiceImpl) CleanupOldBackups(ctx context.Context, keepCount int) error {
	if keepCount <= 0 {
		return fmt.Errorf("keepCount must be positive, got %d", keepCount)
	}

	backups, err := s.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	// Only rotation backups should be auto-cleaned
	var rotationBackups []*rotation.BackupInfo
	for _, backup := range backups {
		if backup.Type == rotation.BackupTypeRotation {
			rotationBackups = append(rotationBackups, backup)
		}
	}

	// If we have fewer than keepCount, nothing to clean
	if len(rotationBackups) <= keepCount {
		return nil
	}

	// Remove the oldest backups
	toRemove := rotationBackups[keepCount:]
	for _, backup := range toRemove {
		if err := os.RemoveAll(backup.Path); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", backup.Name, err)
		}
	}

	return nil
}

// RotateUserToken generates a new authentication token for a specific user
func (s *ServiceImpl) RotateUserToken(ctx context.Context, username string) (string, error) {
	// Use the users store to rotate the token
	return s.usersStore.RotateToken(ctx, username)
}

// RotateSelfToken generates a new token for the currently authenticated user
func (s *ServiceImpl) RotateSelfToken(ctx context.Context, currentUsername string) (string, error) {
	// Self rotation is the same as user rotation, just with the current user's username
	return s.RotateUserToken(ctx, currentUsername)
}

// Helper methods

// copyFileSecurely copies a file from src to dst with secure permissions
func (s *ServiceImpl) copyFileSecurely(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// scanBackupDirectories returns a list of backup directory names
func (s *ServiceImpl) scanBackupDirectories(backupRoot string) ([]string, error) {
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil, err
	}

	var backupDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Include directories that look like backups
			name := entry.Name()
			if strings.HasPrefix(name, "rotate-") ||
				strings.HasPrefix(name, "manual-") ||
				strings.HasPrefix(name, "pre-restore-") {
				backupDirs = append(backupDirs, name)
			}
		}
	}

	return backupDirs, nil
}

// validateBackupIntegrity checks if a backup directory contains both required files
func (s *ServiceImpl) validateBackupIntegrity(backupPath string) bool {
	keyFile := filepath.Join(backupPath, "master.key")
	secretsFile := filepath.Join(backupPath, "secrets.json")

	_, keyErr := os.Stat(keyFile)
	_, secretsErr := os.Stat(secretsFile)

	return keyErr == nil && secretsErr == nil
}

// calculateDirectorySize calculates the total size of all files in a directory
func (s *ServiceImpl) calculateDirectorySize(dirPath string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// determineBackupPath returns the backup path for restoration
func (s *ServiceImpl) determineBackupPath(ctx context.Context, backupName string) (string, error) {
	if backupName == "" {
		return s.findMostRecentValidBackup(ctx)
	}
	return s.validateSpecifiedBackup(ctx, backupName)
}

// findMostRecentValidBackup finds the most recent valid backup
func (s *ServiceImpl) findMostRecentValidBackup(ctx context.Context) (string, error) {
	backups, err := s.ListBackups(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}
	if len(backups) == 0 {
		return "", fmt.Errorf("no backups found")
	}

	// Find the first valid backup (they're already sorted by timestamp, newest first)
	for _, backup := range backups {
		if backup.IsValid {
			return backup.Path, nil
		}
	}

	return "", fmt.Errorf("no valid backups found")
}

// validateSpecifiedBackup validates and returns the path for a specified backup
func (s *ServiceImpl) validateSpecifiedBackup(ctx context.Context, backupName string) (string, error) {
	backupPath := filepath.Join(s.dataDir, s.config.BackupDir, backupName)

	// Check if backup directory exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup '%s' not found", backupName)
	}

	// Validate backup integrity
	if err := s.ValidateBackup(ctx, backupPath); err != nil {
		return "", fmt.Errorf("backup '%s' is invalid: %w", backupName, err)
	}

	return backupPath, nil
}
