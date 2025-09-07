package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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

// backupCurrent copies current key+secrets to a backup dir.
func (s *SecretsStore) backupCurrent(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	cp := func(src, dst string) error {
		data, err := os.ReadFile(src)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		return os.WriteFile(dst, data, 0600)
	}
	if err := cp(s.KeyPath, filepath.Join(dir, "master.key")); err != nil {
		return err
	}
	if err := cp(s.SecretsPath, filepath.Join(dir, "secrets.json")); err != nil {
		return err
	}
	return nil
}

// RotateMasterKey creates a backup, generates a new key,
// re-encrypts all secrets, and persists both key + secrets.
func (s *SecretsStore) RotateMasterKey(backupDir string) error {
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

	plaintexts := make(map[string][]byte, len(s.secrets))
	for k, enc := range s.secrets {
		pt, err := decrypt(s.masterKey, enc)
		if err != nil {
			return fmt.Errorf("decrypt %q with old key failed: %w", k, err)
		}
		plaintexts[k] = pt
	}

	// 3) Generate a NEW key
	newKey := make([]byte, 32)
	if _, err := randRead(newKey); err != nil {
		return fmt.Errorf("generate new key: %w", err)
	}

	// 4) Re-encrypt all plaintexts under the NEW key (in memory)
	reenc := make(map[string]string, len(plaintexts))
	for k, pt := range plaintexts {
		enc2, err := encrypt(newKey, pt)
		if err != nil {
			return fmt.Errorf("encrypt %q with new key failed: %w", k, err)
		}
		reenc[k] = enc2
	}

	// 5) Write new secrets.json to a temp file first
	tmpSecrets := s.SecretsPath + ".tmp"
	data, err := json.MarshalIndent(reenc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpSecrets, data, 0600); err != nil {
		return err
	}

	// 6) Write the NEW master key to a temp file
	tmpKey := s.KeyPath + ".tmp"
	if err := writeMasterKeyToPath(tmpKey, newKey); err != nil {
		_ = os.Remove(tmpSecrets) // cleanup temp files on failure
		return fmt.Errorf("write new master key to temp failed: %w", err)
	}

	// 6.5) Re-encrypt individual backup files with the new key
	if err := s.reencryptBackups(oldKey, newKey); err != nil {
		// Non-fatal - log but continue
		fmt.Printf("Warning: failed to re-encrypt some backup files: %v\n", err)
	}

	// 7) ATOMIC SWAP: Rename both files simultaneously
	// If either fails, the old state is preserved
	if err := os.Rename(tmpKey, s.KeyPath); err != nil {
		_ = os.Remove(tmpSecrets) // cleanup on failure
		_ = os.Remove(tmpKey)
		return fmt.Errorf("atomic swap of master key failed: %w", err)
	}
	if err := os.Rename(tmpSecrets, s.SecretsPath); err != nil {
		// Try to restore old key if secrets rename fails
		_ = s.writeMasterKey(oldKey)
		_ = os.Remove(tmpSecrets)
		return fmt.Errorf("atomic swap of secrets failed: %w", err)
	}
	s.masterKey = newKey
	s.secrets = reenc

	// 8) Clean up old rotation backups (keep only last N)
	if err := s.cleanupOldBackups(DefaultBackupRetentionCount); err != nil {
		// Non-fatal - log but continue
		fmt.Printf("Warning: failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// reencryptBackups re-encrypts all individual backup files from old key to new key
func (s *SecretsStore) reencryptBackups(oldKey, newKey []byte) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	backupDir := filepath.Join(home, ".simple-secrets", "backups")

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil // No backups to re-encrypt
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".bak") {
			continue // Skip non-backup files
		}

		backupPath := filepath.Join(backupDir, entry.Name())

		// Read encrypted backup
		oldEncrypted, err := os.ReadFile(backupPath)
		if err != nil {
			continue // Skip files we can't read
		}

		// Decrypt with old key
		plaintext, err := decrypt(oldKey, string(oldEncrypted))
		if err != nil {
			continue // Skip files we can't decrypt (might be from different rotation)
		}

		// Re-encrypt with new key
		newEncrypted, err := encrypt(newKey, plaintext)
		if err != nil {
			continue // Skip files we can't re-encrypt
		}

		// Write back to same file
		if err := os.WriteFile(backupPath, []byte(newEncrypted), 0600); err != nil {
			continue // Skip files we can't write
		}
	}

	return nil
}

// tiny indirection so it's easy to mock later, if desired
var randRead = func(b []byte) (int, error) { return randReadImpl(b) }

// cleanupOldBackups removes old rotation backup directories, keeping only the most recent 'keep' number
func (s *SecretsStore) cleanupOldBackups(keep int) error {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return nil // No backup directory exists
	}

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return err
	}

	// Filter for rotation backup directories (those starting with "rotate-")
	var rotationDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "rotate-") {
			rotationDirs = append(rotationDirs, entry.Name())
		}
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

// BackupInfo represents information about a rotation backup
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	IsValid   bool      `json:"is_valid"`
}

// ListRotationBackups returns information about available rotation backups
func (s *SecretsStore) ListRotationBackups() ([]BackupInfo, error) {
	backupRoot := filepath.Join(filepath.Dir(s.KeyPath), "backups")
	if _, err := os.Stat(backupRoot); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(backupRoot)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "rotate-") {
			continue
		}

		backupPath := filepath.Join(backupRoot, entry.Name())

		// Parse timestamp from directory name (format: rotate-20060102-150405)
		timestampStr := strings.TrimPrefix(entry.Name(), "rotate-")
		timestamp, err := time.Parse("20060102-150405", timestampStr)
		if err != nil {
			// Skip directories that don't match expected format
			continue
		}

		// Check if backup is valid (contains both master.key and secrets.json)
		keyPath := filepath.Join(backupPath, "master.key")
		secretsPath := filepath.Join(backupPath, "secrets.json")
		isValid := true
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			isValid = false
		}
		if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
			isValid = false
		}

		backups = append(backups, BackupInfo{
			Name:      entry.Name(),
			Path:      backupPath,
			Timestamp: timestamp,
			IsValid:   isValid,
		})
	}

	// Sort by timestamp, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// RestoreFromBackup restores the secrets store from a specific backup directory
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

	// Copy backup files to active location
	backupKeyPath := filepath.Join(backupPath, "master.key")
	backupSecretsPath := filepath.Join(backupPath, "secrets.json")

	if err := copyFile(backupKeyPath, s.KeyPath); err != nil {
		return fmt.Errorf("failed to restore master.key: %w", err)
	}

	if err := copyFile(backupSecretsPath, s.SecretsPath); err != nil {
		return fmt.Errorf("failed to restore secrets.json: %w", err)
	}

	// Reload the store to reflect restored state
	if err := s.loadOrCreateKey(); err != nil {
		return fmt.Errorf("failed to load restored master key: %w", err)
	}

	if err := s.loadSecrets(); err != nil {
		return fmt.Errorf("failed to load restored secrets: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
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
	keyPath := filepath.Join(backupPath, "master.key")
	secretsPath := filepath.Join(backupPath, "secrets.json")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup '%s' not found", backupName)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup '%s' is missing master.key", backupName)
	}
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("backup '%s' is missing secrets.json", backupName)
	}

	return backupPath, nil
}
