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
package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	secretsmodels "simple-secrets/pkg/secrets"

	"simple-secrets/internal/auth"
	"simple-secrets/pkg/crypto"
)

const (
	// DefaultRotationBackupCount is the number of rotation backups to keep by default
	// Set to 1 to minimize attack surface - keeps only the most recent backup
	// Can be configured via config.json {"rotation_backup_count": N} for environments needing more
	DefaultRotationBackupCount = 1
)

// decryptAllSecrets decrypts all secrets with the current master key
func (s *SecretsStore) decryptAllSecrets() (map[string][]byte, error) {
	plaintexts := make(map[string][]byte, len(s.secrets))
	for key, secret := range s.secrets {
		pt, err := crypto.Decrypt(s.masterKey, string(secret.Value))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %q: %w", key, err)
		}
		plaintexts[key] = pt
	}
	return plaintexts, nil
}

// reencryptAllSecrets re-encrypts all plaintext secrets with a new key
func (s *SecretsStore) reencryptAllSecrets(plaintexts map[string][]byte, newKey []byte) (map[string]secretsmodels.Secret, error) {
	newSecrets := make(map[string]secretsmodels.Secret, len(plaintexts))
	for key, pt := range plaintexts {
		enc, err := crypto.Encrypt(newKey, pt)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret %q: %w", key, err)
		}

		// Preserve existing metadata if available, otherwise create new
		metadata := s.createRotationMetadata(key, pt)

		newSecrets[key] = secretsmodels.Secret{
			Key:      key,
			Value:    []byte(enc),
			Metadata: metadata,
		}
	}
	return newSecrets, nil
}

// performAtomicSwap swaps the new key and secrets files into place
func (s *SecretsStore) performAtomicSwap(tmpKey, tmpSecrets string, oldKey []byte) error {
	// Rename temporary files to actual locations
	if err := os.Rename(tmpKey, s.KeyPath); err != nil {
		return fmt.Errorf("failed to move new key into place: %w", err)
	}
	if err := os.Rename(tmpSecrets, s.SecretsPath); err != nil {
		// Try to restore old key if secrets move failed
		_ = os.WriteFile(s.KeyPath, oldKey, 0600)
		return fmt.Errorf("failed to move new secrets into place: %w", err)
	}
	return nil
}

// RotateMasterKey creates a backup, generates a new key,
// re-encrypts all secrets, and persists both key + secrets.
func (s *SecretsStore) RotateMasterKey(backupDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1) Backup
	if backupDir == "" {
		ts := time.Now().Format("20060102-150405")
		backupDir = filepath.Join(filepath.Dir(s.KeyPath), "backups", "rotate-"+ts)
	}
	if err := s.backupCurrent(backupDir); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	// 2) Decrypt all existing secrets with the CURRENT key
	oldKey := make([]byte, len(s.masterKey))
	copy(oldKey, s.masterKey) // Save old key before generating new one

	plaintexts, err := s.decryptAllSecrets()
	if err != nil {
		return fmt.Errorf("failed to decrypt existing secrets: %w", err)
	}

	// 3) Generate a NEW master key
	newKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("failed to generate new master key: %w", err)
	}

	// 4) Re-encrypt all secrets with the NEW key
	newSecrets, err := s.reencryptAllSecrets(plaintexts, newKey)
	if err != nil {
		return fmt.Errorf("failed to re-encrypt secrets: %w", err)
	}

	// 5) Write to temporary files first (atomic operation)
	tmpKeyPath := s.KeyPath + ".tmp"
	tmpSecretsPath := s.SecretsPath + ".tmp"

	if err := s.writeMasterKeyToPath(tmpKeyPath, newKey); err != nil {
		return fmt.Errorf("failed to write new key to temporary file: %w", err)
	}
	defer os.Remove(tmpKeyPath) // Clean up on error

	// Wrap secrets in the file format structure
	fileFormat := SecretsFileFormat{
		Secrets: newSecrets,
		Version: "",
	}

	newSecretsData, err := json.MarshalIndent(fileFormat, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal new secrets: %w", err)
	}
	if err := os.WriteFile(tmpSecretsPath, newSecretsData, 0600); err != nil {
		return fmt.Errorf("failed to write new secrets to temporary file: %w", err)
	}
	defer os.Remove(tmpSecretsPath) // Clean up on error

	// 6) Atomic swap
	if err := s.performAtomicSwap(tmpKeyPath, tmpSecretsPath, oldKey); err != nil {
		return err
	}

	// 7) Update in-memory state
	s.masterKey = newKey
	s.secrets = newSecrets

	// 8) Re-encrypt existing backups with the new key
	if err := s.reencryptBackups(oldKey, newKey); err != nil {
		fmt.Printf("Warning: failed to re-encrypt some backups: %v\n", err)
	}

	// 9) Clean up old backups
	retentionCount := getRotationBackupCount()
	if err := s.cleanupOldBackups(retentionCount); err != nil {
		fmt.Printf("Warning: failed to clean up old backups: %v\n", err)
	}

	return nil
}

// reencryptBackups re-encrypts all backup files with the new master key
func (s *SecretsStore) reencryptBackups(oldKey, newKey []byte) error {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
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

// backupCurrent copies current key+secrets to a backup dir for rotation purposes
func (s *SecretsStore) backupCurrent(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := s.copyFileSecurely(s.KeyPath, filepath.Join(dir, "master.key")); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	if err := s.copyFileSecurely(s.SecretsPath, filepath.Join(dir, "secrets.json")); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// cleanupOldBackups removes old rotation backup directories, keeping only the most recent 'keep' number
func (s *SecretsStore) cleanupOldBackups(keep int) error {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return nil // No backup directory exists
	}

	rotationDirs, err := s.scanRotationBackupDirectories(backupRoot)
	if err != nil {
		return err
	}

	// Sort by name (which includes timestamp) - newest first
	sort.Slice(rotationDirs, func(i, j int) bool {
		return rotationDirs[i] > rotationDirs[j]
	})

	// Remove old backups beyond the keep limit
	for _, dirName := range rotationDirs[keep:] {
		oldDir := filepath.Join(backupRoot, dirName)
		if err := os.RemoveAll(oldDir); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", dirName, err)
		}
	}

	return nil
}

// copyFileSecurely copies a file from src to dst with secure permissions
func (s *SecretsStore) copyFileSecurely(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// scanRotationBackupDirectories returns a list of rotation backup directory names
func (s *SecretsStore) scanRotationBackupDirectories(backupRoot string) ([]string, error) {
	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil, err
	}

	var rotationDirs []string
	for _, entry := range entries {
		if entry.IsDir() && (strings.HasPrefix(entry.Name(), "rotate-") || strings.HasPrefix(entry.Name(), "manual-")) {
			rotationDirs = append(rotationDirs, entry.Name())
		}
	}

	return rotationDirs, nil
}

// BackupInfo represents information about a rotation backup
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	IsValid   bool      `json:"is_valid"`
}

// validateBackupIntegrity checks if a backup directory contains both required files
func (s *SecretsStore) validateBackupIntegrity(backupPath string) bool {
	keyFile := filepath.Join(backupPath, "master.key")
	secretsFile := filepath.Join(backupPath, "secrets.json")

	_, keyErr := os.Stat(keyFile)
	_, secretsErr := os.Stat(secretsFile)

	return keyErr == nil && secretsErr == nil
}

// ListRotationBackups returns information about available rotation backups
func (s *SecretsStore) ListRotationBackups() ([]BackupInfo, error) {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	rotationDirs, err := s.scanRotationBackupDirectories(backupRoot)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, dirName := range rotationDirs {
		backupPath := filepath.Join(backupRoot, dirName)

		// Parse timestamp from directory name
		timestamp := parseBackupTimestamp(dirName)

		backup := BackupInfo{
			Name:      dirName,
			Path:      backupPath,
			Timestamp: timestamp,
			IsValid:   s.validateBackupIntegrity(backupPath),
		}
		backups = append(backups, backup)
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreFromBackup restores the secrets store from a backup
func (s *SecretsStore) RestoreFromBackup(backupName string) error {
	backupPath, err := s.determineBackupPath(backupName)
	if err != nil {
		return err
	}

	// Create a backup of current state before restoring
	currentBackupDir := filepath.Join(filepath.Dir(s.KeyPath), "backups", "pre-restore-"+time.Now().Format("20060102-150405"))
	if err := s.backupCurrent(currentBackupDir); err != nil {
		return fmt.Errorf("failed to backup current state: %w", err)
	}

	// Copy backup files to current locations
	backupKeyPath := filepath.Join(backupPath, "master.key")
	backupSecretsPath := filepath.Join(backupPath, "secrets.json")

	if err := s.copyFileSecurely(backupKeyPath, s.KeyPath); err != nil {
		return fmt.Errorf("failed to restore master key: %w", err)
	}

	if err := s.copyFileSecurely(backupSecretsPath, s.SecretsPath); err != nil {
		return fmt.Errorf("failed to restore secrets: %w", err)
	}

	// Reload the store to pick up the restored data
	if err := s.loadOrCreateKey(); err != nil {
		return fmt.Errorf("failed to load restored key: %w", err)
	}

	if err := s.loadSecrets(); err != nil {
		return fmt.Errorf("failed to load restored secrets: %w", err)
	}

	return nil
}

// determineBackupPath returns the backup path for restoration, either from the most recent valid backup or a specified backup
func (s *SecretsStore) determineBackupPath(backupName string) (string, error) {
	if backupName == "" {
		return s.findMostRecentValidBackup()
	}
	return s.validateSpecifiedBackup(backupName)
}

// findMostRecentValidBackup finds the most recent valid backup
func (s *SecretsStore) findMostRecentValidBackup() (string, error) {
	backups, err := s.ListRotationBackups()
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}
	if len(backups) == 0 {
		return "", fmt.Errorf("no rotation backups found")
	}

	// Find the first valid backup
	for _, backup := range backups {
		if backup.IsValid {
			return backup.Path, nil
		}
	}

	return "", fmt.Errorf("no valid rotation backups found")
}

// validateSpecifiedBackup validates and returns the path for a specified backup
func (s *SecretsStore) validateSpecifiedBackup(backupName string) (string, error) {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	backupPath := filepath.Join(backupRoot, backupName)

	// Verify backup exists and is valid
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup '%s' not found", backupName)
	}

	if !s.validateBackupIntegrity(backupPath) {
		return "", fmt.Errorf("backup '%s' is missing required files", backupName)
	}

	return backupPath, nil
}

// getRotationBackupCount returns the configured rotation backup count or default
func getRotationBackupCount() int {
	// Try to load from config file
	configPath, err := auth.DefaultUserConfigPath("config.json")
	if err != nil {
		return DefaultRotationBackupCount
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultRotationBackupCount // Config file doesn't exist or can't be read
	}

	var config struct {
		RotationBackupCount *int `json:"rotation_backup_count,omitempty"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config.json is corrupted (%v). Using default rotation_backup_count=%d\n", err, DefaultRotationBackupCount)
		return DefaultRotationBackupCount // Invalid JSON, use default
	}

	if config.RotationBackupCount != nil && *config.RotationBackupCount > 0 {
		return *config.RotationBackupCount
	}

	return DefaultRotationBackupCount
}

// parseBackupTimestamp extracts timestamp from backup directory names
func parseBackupTimestamp(dirName string) time.Time {
	timestampStr := extractTimestampString(dirName)
	if timestampStr == "" {
		return time.Time{} // Zero time for unparseable names
	}

	timestamp, err := time.Parse("20060102-150405", timestampStr)
	if err != nil {
		return time.Time{} // Zero time for invalid timestamps
	}

	return timestamp
}

// extractTimestampString extracts the timestamp portion from backup directory names
func extractTimestampString(dirName string) string {
	if strings.HasPrefix(dirName, "rotate-") {
		return strings.TrimPrefix(dirName, "rotate-")
	}

	if strings.HasPrefix(dirName, "manual-") {
		return strings.TrimPrefix(dirName, "manual-")
	}

	return "" // Unknown format
}

// createRotationMetadata creates metadata for a secret during rotation, preserving existing data when available
func (s *SecretsStore) createRotationMetadata(key string, plaintext []byte) secretsmodels.SecretMetadata {
	now := time.Now()

	// Default metadata for new secrets (defensive - shouldn't happen during rotation)
	metadata := secretsmodels.SecretMetadata{
		Key:           key,
		CreatedAt:     now,
		ModifiedAt:    now,
		LastRotatedAt: &now,
		RotationCount: 1,
		Disabled:      false,
		Size:          len(plaintext),
	}

	// Override with existing metadata if available, preserving user modification timestamps
	if existingSecret, exists := s.secrets[key]; exists {
		metadata = existingSecret.Metadata
		// Update rotation tracking - this IS a cryptographic operation we want to track
		metadata.LastRotatedAt = &now
		metadata.RotationCount++
		// Update size in case of data corruption fixes during rotation
		metadata.Size = len(plaintext)
		// DO NOT update ModifiedAt - rotation is a cryptographic operation, not content modification
	}

	return metadata
}
