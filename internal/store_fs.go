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
	"time"
)

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

// LoadSecretsStore: create ~/.simple-secrets, load key + secrets
func LoadSecretsStore() (*SecretsStore, error) {
	return LoadSecretsStoreWithBackend(NewFilesystemBackend())
}

// LoadSecretsStoreWithBackend allows injection of storage backend for better testability
func LoadSecretsStoreWithBackend(backend StorageBackend) (*SecretsStore, error) {
	dir, err := getConfigDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine configuration directory for secrets storage: %w", err)
	}
	_ = os.MkdirAll(dir, secureDirectoryPermissions)

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

// getConfigDirectory returns the configuration directory, respecting test overrides
func getConfigDirectory() (string, error) {
	// Check for test override first
	if testDir := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR"); testDir != "" {
		return testDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".simple-secrets"), nil
}

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
	if _, err := os.Stat(s.SecretsPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	b, err := os.ReadFile(s.SecretsPath)
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
	return AtomicWriteFile(s.SecretsPath, b, secureFilePermissions)
}

// backupSecret creates an encrypted backup of a secret
func (s *SecretsStore) backupSecret(key, encryptedValue string) {
	backupPath := s.getBackupPath(key)
	s.createBackupDirectory()
	_ = os.WriteFile(backupPath, []byte(encryptedValue), secureFilePermissions)
}

func (s *SecretsStore) Put(key, value string) error {
	// Encrypt outside any locks to minimize lock time
	s.mu.RLock()
	masterKey := s.masterKey
	s.mu.RUnlock()

	encryptedValue, err := encrypt(masterKey, []byte(value))
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
	freshSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to reload secrets before write: %w", err)
	}

	// Now perform the update with merged state
	s.mu.Lock()
	// Merge disk state with in-memory state (disk takes precedence for conflicts)
	for k, v := range freshSecrets {
		s.secrets[k] = v
	}
	s.ensureBackupExists(key, encryptedValue)
	s.secrets[key] = encryptedValue
	err = s.saveSecretsLocked()
	s.mu.Unlock()

	return err
}

// ensureBackupExists creates a backup for the secret using the appropriate strategy
func (s *SecretsStore) ensureBackupExists(key, newEncryptedValue string) {
	strategy := s.determineBackupStrategy(key)
	backupValue := strategy.selectBackupValue(newEncryptedValue)
	s.backupSecret(key, backupValue)
}

// BackupStrategy defines how to handle backup for a secret
type BackupStrategy struct {
	hasExistingValue bool
	existingValue    string
}

// selectBackupValue chooses what value to backup based on the strategy
func (bs *BackupStrategy) selectBackupValue(newValue string) string {
	if bs.hasExistingValue {
		return bs.existingValue // Backup the previous value
	}
	return newValue // Backup the new value if no previous value exists
}

// determineBackupStrategy decides what backup approach to use for a secret
func (s *SecretsStore) determineBackupStrategy(key string) *BackupStrategy {
	existingValue, hasExisting := s.secrets[key]
	return &BackupStrategy{
		hasExistingValue: hasExisting,
		existingValue:    existingValue,
	}
}

func (s *SecretsStore) Get(key string) (string, error) {
	s.mu.RLock()
	enc, ok := s.secrets[key]
	masterKey := s.masterKey // Copy for use outside lock
	s.mu.RUnlock()

	if !ok {
		return "", ErrNotFound
	}
	pt, err := decrypt(masterKey, enc)
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
	freshSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to reload secrets before delete: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Merge disk state with in-memory state (disk takes precedence for conflicts)
	for k, v := range freshSecrets {
		s.secrets[k] = v
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
	decrypted, err := decrypt(s.masterKey, encryptedData)
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
	freshSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to reload secrets before disable: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Merge disk state with in-memory state (disk takes precedence for conflicts)
	for k, v := range freshSecrets {
		s.secrets[k] = v
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
	freshSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to reload secrets before enable: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Merge disk state with in-memory state (disk takes precedence for conflicts)
	for k, v := range freshSecrets {
		s.secrets[k] = v
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
	_ = os.MkdirAll(backupDir, secureDirectoryPermissions)
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
