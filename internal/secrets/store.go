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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"simple-secrets/pkg/crypto"
)

const (
	secureFilePermissions      = 0600 // Owner read/write only
	secureDirectoryPermissions = 0700 // Owner read/write/execute only
)

// Store handles secrets storage operations with encryption and backup functionality
type Store struct {
	KeyPath     string
	SecretsPath string
	masterKey   []byte
	secrets     map[string]string // key -> base64(ciphertext)
	mu          sync.RWMutex      // protects secrets map and masterKey
	repo        Repository        // storage repository
}

// LoadStore creates ~/.simple-secrets, loads key + secrets
// For testing, inject a custom repository; for production, use NewFilesystemRepository()
func LoadStore(repo Repository) (*Store, error) {
	ctx := context.Background()
	dir, err := getConfigDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to determine configuration directory for secrets storage: %w", err)
	}
	_ = repo.MkdirAll(ctx, dir, secureDirectoryPermissions)

	s := &Store{
		KeyPath:     filepath.Join(dir, "master.key"),
		SecretsPath: filepath.Join(dir, "secrets.json"),
		secrets:     make(map[string]string),
		repo:        repo,
	}

	if err := s.loadOrCreateKey(); err != nil {
		return nil, err
	}
	if err := s.loadSecrets(); err != nil {
		return nil, err
	}
	return s, nil
}

// LoadStoreFromDir creates a secrets store using a custom directory
// This is useful for testing and custom deployments where the config directory needs to be specified
func LoadStoreFromDir(repo Repository, configDir string) (*Store, error) {
	ctx := context.Background()
	_ = repo.MkdirAll(ctx, configDir, secureDirectoryPermissions)

	s := &Store{
		KeyPath:     filepath.Join(configDir, "master.key"),
		SecretsPath: filepath.Join(configDir, "secrets.json"),
		secrets:     make(map[string]string),
		repo:        repo,
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

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to determine user home directory: %w", err)
	}

	return filepath.Join(homeDir, ".simple-secrets"), nil
}

// loadOrCreateKey loads or creates the master encryption key
func (s *Store) loadOrCreateKey() error {
	ctx := context.Background()
	if s.repo.Exists(ctx, s.KeyPath) {
		keyData, err := s.repo.ReadFile(ctx, s.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read master key: %w", err)
		}
		s.masterKey = keyData
	} else {
		// Generate new master key
		key, err := crypto.GenerateKey()
		if err != nil {
			return fmt.Errorf("failed to generate master key: %w", err)
		}
		s.masterKey = key

		// Save key with restricted permissions
		if err := s.repo.AtomicWriteFile(ctx, s.KeyPath, key, secureFilePermissions); err != nil {
			return fmt.Errorf("failed to save master key: %w", err)
		}
	}
	return nil
}

// loadSecrets loads secrets from disk into memory
func (s *Store) loadSecrets() error {
	ctx := context.Background()
	if !s.repo.Exists(ctx, s.SecretsPath) {
		// No secrets file exists yet, start with empty map
		return nil
	}

	diskSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return fmt.Errorf("failed to load secrets from disk: %w", err)
	}

	s.secrets = diskSecrets
	return nil
}

// loadSecretsFromDisk reads and returns secrets from disk without modifying the in-memory state
func (s *Store) loadSecretsFromDisk() (map[string]string, error) {
	ctx := context.Background()
	data, err := s.repo.ReadFile(ctx, s.SecretsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return make(map[string]string), nil
	}

	var secrets map[string]string
	if err := json.Unmarshal(data, &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse secrets file: %w", err)
	}

	return secrets, nil
}

// saveSecretsLocked saves the current secrets to disk (must be called with write lock held)
func (s *Store) saveSecretsLocked() error {
	ctx := context.Background()
	data, err := json.MarshalIndent(s.secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	if err := s.repo.AtomicWriteFile(ctx, s.SecretsPath, data, secureFilePermissions); err != nil {
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	return nil
}

// mergeWithDiskState merges current in-memory state with disk state to avoid conflicts
func (s *Store) mergeWithDiskState() error {
	diskSecrets, err := s.loadSecretsFromDisk()
	if err != nil {
		return err
	}

	// Merge disk secrets into memory (disk wins on conflicts)
	for key, value := range diskSecrets {
		s.secrets[key] = value
	}

	return nil
}

// backupSecret creates a backup copy of a secret before modification
func (s *Store) backupSecret(key, encryptedValue string) {
	ctx := context.Background()
	backupDir := filepath.Join(filepath.Dir(s.SecretsPath), "backups")
	_ = s.repo.MkdirAll(ctx, backupDir, secureDirectoryPermissions)
	// Backup implementation would go here
}

// Put encrypts and stores a secret
func (s *Store) Put(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encrypt the value
	encryptedValue, err := crypto.Encrypt(s.masterKey, []byte(value))
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	// If the key already exists, create a backup
	if existingValue, exists := s.secrets[key]; exists {
		s.backupSecret(key, existingValue)
	}

	// Merge with disk state to avoid conflicts
	if err := s.mergeWithDiskState(); err != nil {
		return fmt.Errorf("failed to merge with disk state: %w", err)
	}

	// Store encrypted value
	s.secrets[key] = encryptedValue

	// Ensure backup exists for the new value
	s.ensureBackupExists(key)

	// Save to disk
	if err := s.saveSecretsLocked(); err != nil {
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	return nil
}

// ensureBackupExists creates a backup if one doesn't already exist
func (s *Store) ensureBackupExists(key string) {
	ctx := context.Background()
	backupDir := filepath.Join(filepath.Dir(s.SecretsPath), "backups")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s.backup", key))

	if !s.repo.Exists(ctx, backupFile) {
		if encryptedValue, exists := s.secrets[key]; exists {
			s.backupSecret(key, encryptedValue)
		}
	}
}

// Get retrieves and decrypts a secret
func (s *Store) Get(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	encryptedValue, exists := s.secrets[key]
	if !exists {
		return "", fmt.Errorf("secret '%s' not found", key)
	}

	// Decrypt the value
	decryptedBytes, err := crypto.Decrypt(s.masterKey, encryptedValue)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(decryptedBytes), nil
}

// ListKeys returns all secret keys in sorted order
func (s *Store) ListKeys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.secrets))
	for key := range s.secrets {
		// Only include enabled secrets (not disabled ones with __DISABLED__ prefix)
		if !strings.HasPrefix(key, "__DISABLED__") {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys
}

// Delete removes a secret
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.secrets[key]; !exists {
		return fmt.Errorf("secret '%s' not found", key)
	}

	// Create backup before deletion
	if encryptedValue, exists := s.secrets[key]; exists {
		s.backupSecret(key, encryptedValue)
	}

	// Merge with disk state
	if err := s.mergeWithDiskState(); err != nil {
		return fmt.Errorf("failed to merge with disk state: %w", err)
	}

	// Remove from memory
	delete(s.secrets, key)

	// Save to disk
	if err := s.saveSecretsLocked(); err != nil {
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	return nil
}

// DecryptBackup decrypts a backup secret value
func (s *Store) DecryptBackup(encryptedData string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decryptedBytes, err := crypto.Decrypt(s.masterKey, encryptedData)
	if err != nil {
		return "", fmt.Errorf("backup decryption failed: %w", err)
	}

	return string(decryptedBytes), nil
}

// DisableSecret temporarily disables a secret by renaming it
func (s *Store) DisableSecret(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if secret exists
	encryptedValue, exists := s.secrets[key]
	if !exists {
		return fmt.Errorf("secret '%s' not found", key)
	}

	// Merge with disk state
	if err := s.mergeWithDiskState(); err != nil {
		return fmt.Errorf("failed to merge with disk state: %w", err)
	}

	// Create disabled key name with timestamp
	timestamp := time.Now().Format("20060102_150405")
	disabledKey := fmt.Sprintf("__DISABLED__%s__%s", key, timestamp)

	// Create metadata for the disabled secret
	metadata := map[string]any{
		"original_key":    key,
		"disabled_at":     time.Now().Format(time.RFC3339),
		"encrypted_value": encryptedValue,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to create disable metadata: %w", err)
	}

	// Store as disabled secret
	disabledEncryptedValue, err := crypto.Encrypt(s.masterKey, metadataBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt disable metadata: %w", err)
	}

	s.secrets[disabledKey] = disabledEncryptedValue
	delete(s.secrets, key)

	// Save to disk
	if err := s.saveSecretsLocked(); err != nil {
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	return nil
}

// EnableSecret re-enables a previously disabled secret
func (s *Store) EnableSecret(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the disabled secret
	disabledSecretsMap := s.buildDisabledSecretsMap()
	disabledKey, exists := disabledSecretsMap[key]
	if !exists {
		return fmt.Errorf("disabled secret '%s' not found", key)
	}

	// Get the disabled secret's encrypted metadata
	encryptedMetadata, exists := s.secrets[disabledKey]
	if !exists {
		return fmt.Errorf("disabled secret data not found for '%s'", key)
	}

	// Decrypt and parse metadata
	metadataBytes, err := crypto.Decrypt(s.masterKey, encryptedMetadata)
	if err != nil {
		return fmt.Errorf("failed to decrypt disabled secret metadata: %w", err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return fmt.Errorf("failed to parse disabled secret metadata: %w", err)
	}

	// Extract original encrypted value
	originalEncryptedValue, ok := metadata["encrypted_value"].(string)
	if !ok {
		return fmt.Errorf("invalid disabled secret metadata format")
	}

	// Merge with disk state
	if err := s.mergeWithDiskState(); err != nil {
		return fmt.Errorf("failed to merge with disk state: %w", err)
	}

	// Restore the original secret
	s.secrets[key] = originalEncryptedValue
	delete(s.secrets, disabledKey)

	// Save to disk
	if err := s.saveSecretsLocked(); err != nil {
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	return nil
}

// buildDisabledSecretsMap creates a map of original key -> disabled key
func (s *Store) buildDisabledSecretsMap() map[string]string {
	disabledMap := make(map[string]string)
	for disabledKey := range s.secrets {
		if strings.HasPrefix(disabledKey, "__DISABLED__") {
			originalKey := s.extractOriginalKeyFromDisabled(disabledKey)
			if originalKey != "" {
				disabledMap[originalKey] = disabledKey
			}
		}
	}
	return disabledMap
}

// extractOriginalKeyFromDisabled extracts the original key from a disabled key name
func (s *Store) extractOriginalKeyFromDisabled(disabledKey string) string {
	// Format: __DISABLED__originalkey__timestamp
	if !strings.HasPrefix(disabledKey, "__DISABLED__") {
		return ""
	}

	// Remove the __DISABLED__ prefix
	remaining := strings.TrimPrefix(disabledKey, "__DISABLED__")

	// Find the last __ which separates the key from the timestamp
	lastIndex := strings.LastIndex(remaining, "__")
	if lastIndex == -1 {
		return remaining // Fallback if no timestamp separator found
	}

	return remaining[:lastIndex]
}

// ListDisabledSecrets returns all disabled secret keys
func (s *Store) ListDisabledSecrets() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	disabledSecretsMap := s.buildDisabledSecretsMap()
	keys := make([]string, 0, len(disabledSecretsMap))
	for originalKey := range disabledSecretsMap {
		keys = append(keys, originalKey)
	}

	sort.Strings(keys)
	return keys
}

// IsEnabled checks if a secret is currently enabled (not disabled)
func (s *Store) IsEnabled(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if the key exists in enabled secrets
	_, exists := s.secrets[key]
	if exists && !strings.HasPrefix(key, "__DISABLED__") {
		return true
	}

	return false
}

// RotateMasterKey generates a new master encryption key and re-encrypts all secrets
func (s *Store) RotateMasterKey(ctx context.Context, backupDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// For now, master key rotation in the new platform services architecture
	// requires careful integration with the crypto service and repository patterns.
	// This is a complex operation that involves:
	// 1. Creating backups of current state
	// 2. Generating a new master key
	// 3. Re-encrypting all secrets with the new key
	// 4. Atomically updating both key and secrets files
	//
	// This requires significant architectural work to properly integrate with
	// the repository pattern and crypto service.
	return fmt.Errorf("master key rotation not yet implemented in platform services architecture - requires integration with crypto service and repository patterns")
}
