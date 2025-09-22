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
	"sync"
	"syscall"
	"time"
)

// FileMode represents file permissions for storage operations
type FileMode os.FileMode

const (
	FileMode0755 FileMode = 0755
	FileMode0644 FileMode = 0644
	FileMode0600 FileMode = 0600
)

// StorageBackend defines the interface for storage operations used by the secrets domain
type StorageBackend interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm FileMode) error
	AtomicWriteFile(path string, data []byte, perm FileMode) error
	MkdirAll(path string, perm FileMode) error
	RemoveAll(path string) error
	Exists(path string) bool
	ListDir(path string) ([]string, error)
}

// FilesystemBackend implements StorageBackend for local filesystem operations
type FilesystemBackend struct{}

// NewFilesystemBackend creates a new filesystem storage backend
func NewFilesystemBackend() *FilesystemBackend {
	return &FilesystemBackend{}
}

// ReadFile reads data from a file
func (fs *FilesystemBackend) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file with specified permissions
func (fs *FilesystemBackend) WriteFile(path string, data []byte, perm FileMode) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

// AtomicWriteFile performs an atomic write using a temporary file and rename
func (fs *FilesystemBackend) AtomicWriteFile(path string, data []byte, perm FileMode) error {
	return AtomicWriteFile(path, data, os.FileMode(perm))
}

// MkdirAll creates directories with specified permissions
func (fs *FilesystemBackend) MkdirAll(path string, perm FileMode) error {
	return os.MkdirAll(path, os.FileMode(perm))
}

// RemoveAll removes a directory and all its contents
func (fs *FilesystemBackend) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Exists checks if a file or directory exists
func (fs *FilesystemBackend) Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ListDir lists the contents of a directory
func (fs *FilesystemBackend) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return names, nil
}

// ensureDirectoryExists creates the parent directory if it doesn't exist
func (fs *FilesystemBackend) ensureDirectoryExists(filePath string) error {
	dir := filepath.Dir(filePath)
	return fs.MkdirAll(dir, FileMode0755)
}

const (
	disabledPrefix = "__DISABLED_"
	// File permission constants for clarity
	secureFilePermissions      = 0600 // Owner read/write only
	secureDirectoryPermissions = 0700 // Owner read/write/execute only
)

type SecretsStore struct {
	KeyPath     string
	SecretsPath string
	masterKey   []byte
	secrets     map[string]string // key -> base64(ciphertext)
	mu          sync.RWMutex      // protects secrets map and masterKey
	storage     StorageBackend    // injectable storage backend
}

// LoadSecretsStore creates ~/.simple-secrets, loads key + secrets
// For testing, inject a custom backend; for production, use NewFilesystemBackend()
func LoadSecretsStore(backend StorageBackend) (*SecretsStore, error) {
	dir, err := getConfigDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine configuration directory for secrets storage: %w", err)
	}
	_ = backend.MkdirAll(dir, FileMode(secureDirectoryPermissions))

	s := &SecretsStore{
		KeyPath:     filepath.Join(dir, "master.key"),
		SecretsPath: filepath.Join(dir, "secrets.json"),
		secrets:     make(map[string]string),
		storage:     backend,
	}

	if err := s.loadOrCreateKey(); err != nil {
		return nil, err
	}
	if err := s.loadSecrets(); err != nil {
		return nil, err
	}
	return s, nil
}

// LoadSecretsStoreFromDir creates a secrets store using a custom directory
// This is useful for testing and custom deployments where the config directory needs to be specified
func LoadSecretsStoreFromDir(backend StorageBackend, configDir string) (*SecretsStore, error) {
	_ = backend.MkdirAll(configDir, FileMode(secureDirectoryPermissions))

	s := &SecretsStore{
		KeyPath:     filepath.Join(configDir, "master.key"),
		SecretsPath: filepath.Join(configDir, "secrets.json"),
		secrets:     make(map[string]string),
		storage:     backend,
	}

	if err := s.loadOrCreateKey(); err != nil {
		return nil, err
	}
	if err := s.loadSecrets(); err != nil {
		return nil, err
	}
	return s, nil
}

// getConfigDirectory returns the configuration directory, respecting test overrides
func getConfigDirectory() (string, error) {
	// For testing purposes
	if testDir := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR"); testDir != "" {
		return testDir, nil
	}

	return GetSimpleSecretsPath()
}

// loadSecrets loads secrets from disk and updates in-memory state
func (s *SecretsStore) loadSecrets() error {
	secrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.secrets = secrets
	s.mu.Unlock()
	return nil
}

// loadSecretsFromDisk loads secrets from disk without modifying in-memory state
func (s *SecretsStore) loadSecretsFromDisk() (map[string]string, error) {
	if !s.storage.Exists(s.SecretsPath) {
		return make(map[string]string), nil
	}
	b, err := s.storage.ReadFile(s.SecretsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets database from %s: %w", s.SecretsPath, err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(b, &secrets); err != nil {
		return nil, fmt.Errorf("secrets database appears to be corrupted (JSON parse error: %v). "+
			"Recovery options: "+
			"Restore from backup: ./simple-secrets restore-database; "+
			"List available backups: ./simple-secrets list backups; "+
			"Emergency contact: check ~/.simple-secrets/backups/ directory. "+
			"Do not delete ~/.simple-secrets/ - your backups contain recoverable data", err)
	}

	return secrets, nil
}

// saveSecretsLocked saves secrets to disk, assumes caller holds lock
func (s *SecretsStore) saveSecretsLocked() error {
	b, err := json.MarshalIndent(s.secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize secrets for saving: %w", err)
	}
	// Use AtomicWriteFile for proper race condition protection
	return s.storage.AtomicWriteFile(s.SecretsPath, b, FileMode(secureFilePermissions))
}

// mergeWithDiskState loads fresh state from disk and merges it with in-memory state.
// This ensures consistency when multiple processes might be modifying the store.
// Disk state takes precedence for conflicts.
func (s *SecretsStore) mergeWithDiskState() error {
	freshSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to reload secrets for merge: %w", err)
	}

	// Merge disk state with in-memory state (disk takes precedence for conflicts)
	for k, v := range freshSecrets {
		s.secrets[k] = v
	}
	return nil
}

// backupSecret creates an encrypted backup of a secret
func (s *SecretsStore) backupSecret(key, encryptedValue string) {
	backupPath := s.getBackupPath(key)
	s.createBackupDirectory()
	_ = s.storage.WriteFile(backupPath, []byte(encryptedValue), FileMode(secureFilePermissions))
}

// encryptWithMasterKey safely encrypts data using the current master key
// This ensures the key is not stale when used for encryption
func (s *SecretsStore) encryptWithMasterKey(data []byte) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return encrypt(s.masterKey, data)
}

// decryptWithMasterKey safely decrypts data using the current master key
// This ensures the key is not stale when used for decryption
func (s *SecretsStore) decryptWithMasterKey(encryptedData string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return decrypt(s.masterKey, encryptedData)
}

func (s *SecretsStore) Put(key, value string) error {
	// Encrypt using atomic key access to prevent stale key usage
	encryptedValue, err := s.encryptWithMasterKey([]byte(value))
	if err != nil {
		return err
	}

	// Acquire file lock to prevent concurrent writes from other processes
	lock, err := LockFile(s.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to acquire database lock: %w", err)
	}
	defer lock.Unlock()

	// For concurrent operations within the same process, we need to be more careful
	// Load fresh state but merge it with existing in-memory state
	s.mu.Lock()
	err = s.mergeWithDiskState()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to merge disk state: %w", err)
	}

	// Now perform the update with merged state
	s.ensureBackupExists(key)
	s.secrets[key] = encryptedValue
	err = s.saveSecretsLocked()
	s.mu.Unlock()

	return err
}

// ensureBackupExists stores an encrypted backup of the current value (if any) before modification.
// This allows rollback if the operation fails. Only the most recent backup is kept.
func (s *SecretsStore) ensureBackupExists(key string) {
	// Defensive validation: ensure key is not empty
	if key == "" {
		return // No backup needed for empty keys
	}

	// Store current value as backup using existing backup system
	if currentValue, exists := s.secrets[key]; exists {
		s.backupSecret(key, currentValue)
	}
}

func (s *SecretsStore) Get(key string) (string, error) {
	s.mu.RLock()
	enc, ok := s.secrets[key]
	s.mu.RUnlock()

	if !ok {
		return "", ErrNotFound
	}

	// Use atomic key access to prevent stale key usage
	pt, err := s.decryptWithMasterKey(enc)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (s *SecretsStore) ListKeys() []string {
	s.mu.RLock()
	keys := make([]string, 0, len(s.secrets))
	for k := range s.secrets {
		if !strings.HasPrefix(k, disabledPrefix) {
			keys = append(keys, k)
		}
	}
	s.mu.RUnlock()

	sort.Strings(keys)
	return keys
}

func (s *SecretsStore) Delete(key string) error {
	// Acquire file lock to prevent concurrent writes from other processes
	lock, err := LockFile(s.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to acquire database lock: %w", err)
	}
	defer lock.Unlock()

	// Load fresh state but merge it with existing in-memory state
	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.mergeWithDiskState()
	if err != nil {
		return fmt.Errorf("failed to merge disk state: %w", err)
	}

	prevEnc, ok := s.secrets[key]
	if !ok {
		return ErrNotFound
	}
	// Store the encrypted data as backup (not plaintext!)
	s.backupSecret(key, prevEnc)

	delete(s.secrets, key)
	return s.saveSecretsLocked()
}

// DecryptBackup decrypts a backup file's encrypted content
func (s *SecretsStore) DecryptBackup(encryptedData string) (string, error) {
	// Use atomic key access to prevent stale key usage
	decrypted, err := s.decryptWithMasterKey(encryptedData)
	if err != nil {
		return "", err
	}
	return string(decrypted), nil
}

// DisableSecret marks a secret as disabled by adding a special prefix
func (s *SecretsStore) DisableSecret(key string) error {
	// Acquire file lock to prevent concurrent writes from other processes
	lock, err := LockFile(s.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to acquire database lock: %w", err)
	}
	defer lock.Unlock()

	// Load fresh state but merge it with existing in-memory state
	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.mergeWithDiskState()
	if err != nil {
		return fmt.Errorf("failed to merge disk state: %w", err)
	}

	enc, ok := s.secrets[key]
	if !ok {
		return ErrNotFound
	}

	// Create backup before disabling
	s.backupSecret(key, enc)

	// Mark as disabled using JSON encoding to handle keys with special characters
	timestamp := time.Now().UnixNano()
	keyData := map[string]any{
		"timestamp": timestamp,
		"key":       key,
	}
	keyJSON, _ := json.Marshal(keyData)
	disabledKey := disabledPrefix + string(keyJSON)

	s.secrets[disabledKey] = enc
	delete(s.secrets, key)

	return s.saveSecretsLocked()
}

// buildDisabledSecretsMap creates a map from original key names to their disabled key names
// Assumes caller holds appropriate lock
func (s *SecretsStore) buildDisabledSecretsMap() map[string]string {
	disabledMap := make(map[string]string)
	for disabledKey := range s.secrets {
		if originalKey := s.extractOriginalKeyFromDisabled(disabledKey); originalKey != "" {
			disabledMap[originalKey] = disabledKey
		}
	}
	return disabledMap
}

func (s *SecretsStore) extractOriginalKeyFromDisabled(disabledKey string) string {
	jsonData, isDisabled := strings.CutPrefix(disabledKey, disabledPrefix)
	if !isDisabled {
		return ""
	}

	if originalKey := s.parseJsonFormat(jsonData); originalKey != "" {
		return originalKey
	}

	return s.parseLegacyFormat(jsonData)
}

func (s *SecretsStore) parseJsonFormat(jsonData string) string {
	var keyData map[string]any
	if err := json.Unmarshal([]byte(jsonData), &keyData); err != nil {
		return ""
	}

	originalKey, ok := keyData["key"].(string)
	if !ok {
		return ""
	}

	return originalKey
}

func (s *SecretsStore) parseLegacyFormat(data string) string {
	const (
		notFound         = -1
		underscoreLength = 1
	)
	underscorePosition := strings.Index(data, "_")
	if underscorePosition == notFound {
		return ""
	}
	positionAfterUnderscore := underscorePosition + underscoreLength
	return data[positionAfterUnderscore:]
}

// EnableSecret re-enables a previously disabled secret
func (s *SecretsStore) EnableSecret(key string) error {
	// Acquire file lock to prevent concurrent writes from other processes
	lock, err := LockFile(s.SecretsPath)
	if err != nil {
		return fmt.Errorf("failed to acquire database lock: %w", err)
	}
	defer lock.Unlock()

	// Load fresh state but merge it with existing in-memory state
	s.mu.Lock()
	defer s.mu.Unlock()

	err = s.mergeWithDiskState()
	if err != nil {
		return fmt.Errorf("failed to merge disk state: %w", err)
	}

	// Build a map of original keys to their disabled keys for efficient lookup
	disabledMap := s.buildDisabledSecretsMap()

	disabledKey, found := disabledMap[key]
	if !found {
		return fmt.Errorf("disabled secret not found")
	}

	// Move back to original key
	s.secrets[key] = s.secrets[disabledKey]
	delete(s.secrets, disabledKey)

	return s.saveSecretsLocked()
}

// ListDisabledSecrets returns a list of disabled secret keys
func (s *SecretsStore) ListDisabledSecrets() []string {
	s.mu.RLock()
	disabledMap := s.buildDisabledSecretsMap()
	s.mu.RUnlock()

	disabled := make([]string, 0, len(disabledMap))
	for originalKey := range disabledMap {
		disabled = append(disabled, originalKey)
	}

	sort.Strings(disabled)
	return disabled
}

// IsEnabled checks if a secret is enabled (not disabled)
func (s *SecretsStore) IsEnabled(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if the key exists in enabled state
	_, exists := s.secrets[key]
	return exists
}

// CreateBackup creates a backup of the current secrets and master key
func (s *SecretsStore) CreateBackup(backupDir string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if backupDir == "" {
		ts := time.Now().Format("20060102-150405")
		backupDir = filepath.Join(filepath.Dir(s.KeyPath), "backups", "manual-"+ts)
	}

	return s.backupCurrent(backupDir)
}

var ErrNotFound = os.ErrNotExist

// createBackupDirectory ensures the backup directory exists with secure permissions
func (s *SecretsStore) createBackupDirectory() {
	backupDir := s.getBackupDirectory()
	_ = s.storage.MkdirAll(backupDir, FileMode(secureDirectoryPermissions))
}

// getBackupDirectory returns the path to the backup directory
func (s *SecretsStore) getBackupDirectory() string {
	configDir, _ := getConfigDirectory()
	return filepath.Join(configDir, "backups")
}

// getBackupPath returns the full path for a secret's backup file
func (s *SecretsStore) getBackupPath(key string) string {
	return filepath.Join(s.getBackupDirectory(), key+".bak")
}

// GetBackupPath returns the full path for a secret's backup file (public method)
func (s *SecretsStore) GetBackupPath(key string) string {
	return s.getBackupPath(key)
}

// ====================================
// File Operations and Locking
// ====================================
// These functions handle atomic file operations and file locking
// for safe concurrent access to secrets storage.

// AtomicWriteFile writes data to a file atomically using a temporary file and rename.
// This ensures that either the entire write succeeds or fails completely, preventing
// partial writes that could corrupt the file.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	// Use unique temp file name to prevent race conditions in concurrent operations
	// Include nanosecond timestamp and goroutine ID to ensure uniqueness
	tmpPath := fmt.Sprintf("%s.tmp.%d.%d", path, os.Getpid(), time.Now().UnixNano())

	// Write to temporary file first
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename to final location
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically update file: %w", err)
	}

	return nil
}

// FileLock represents a file lock
type FileLock struct {
	file *os.File
	path string
}

// LockFile creates an exclusive file lock for coordinating access to a resource.
// This prevents multiple processes from concurrently modifying the same data.
func LockFile(path string) (*FileLock, error) {
	lockPath := path + ".lock"

	// Create lock file if it doesn't exist
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock with timeout
	const maxLockAttempts = 100 // 10 seconds total with 100ms intervals (increased for high concurrency)
	for attempt := 0; attempt < maxLockAttempts; attempt++ {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			return &FileLock{file: file, path: lockPath}, nil
		}

		if err != syscall.EWOULDBLOCK {
			// Real error, not just lock busy
			file.Close()
			return nil, fmt.Errorf("failed to acquire file lock: %w", err)
		}

		// Lock is busy, wait and retry with exponential backoff
		backoffMs := 10 + (attempt * 2) // Start at 10ms, increase by 2ms each attempt
		if backoffMs > 100 {
			backoffMs = 100 // Cap at 100ms
		}
		time.Sleep(time.Duration(backoffMs) * time.Millisecond)
	}

	file.Close()
	return nil, fmt.Errorf("timeout acquiring file lock after %d attempts", maxLockAttempts)
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	// Release the lock
	err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN)

	// Close the file
	closeErr := fl.file.Close()
	fl.file = nil

	// Clean up lock file
	_ = os.Remove(fl.path)

	if err != nil {
		return fmt.Errorf("failed to release file lock: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close lock file: %w", closeErr)
	}

	return nil
}
