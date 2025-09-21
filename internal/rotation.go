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
package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultBackupRetentionCount is the number of rotation backups to keep by default
	DefaultBackupRetentionCount = 5
)

// decryptAllSecrets decrypts all secrets with the current master key
func (s *SecretsStore) decryptAllSecrets() (map[string][]byte, error) {
	plaintexts := make(map[string][]byte, len(s.secrets))
	for key, encValue := range s.secrets {
		pt, err := decrypt(s.masterKey, encValue)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %q: %w", key, err)
		}
		plaintexts[key] = pt
	}
	return plaintexts, nil
}

// reencryptAllSecrets re-encrypts all plaintext secrets with a new key
func (s *SecretsStore) reencryptAllSecrets(plaintexts map[string][]byte, newKey []byte) (map[string]string, error) {
	newSecrets := make(map[string]string, len(plaintexts))
	for key, pt := range plaintexts {
		enc, err := encrypt(newKey, pt)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret %q: %w", key, err)
		}
		newSecrets[key] = enc
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
	newKey := make([]byte, AES256KeySize)
	if _, err := randRead(newKey); err != nil {
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

	newSecretsData, err := json.MarshalIndent(newSecrets, "", "  ")
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
	if err := s.cleanupOldBackups(DefaultBackupRetentionCount); err != nil {
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
		plaintext, err := decrypt(oldKey, string(encryptedData))
		if err != nil {
			// If we can't decrypt with old key, it might already be encrypted with new key
			// or it might be corrupted. Skip this file.
			fmt.Printf("Warning: failed to decrypt backup file %s with old key, skipping: %v\n", path, err)
			return nil
		}

		// Re-encrypt with new key
		newEncrypted, err := encrypt(newKey, plaintext)
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
	for i := keep; i < len(rotationDirs); i++ {
		oldDir := filepath.Join(backupRoot, rotationDirs[i])
		if err := os.RemoveAll(oldDir); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", rotationDirs[i], err)
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