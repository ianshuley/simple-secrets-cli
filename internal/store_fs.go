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

const disabledPrefix = "__DISABLED_"

type SecretsStore struct {
	KeyPath     string
	SecretsPath string
	masterKey   []byte
	secrets     map[string]string // key -> base64(ciphertext)
}

// LoadSecretsStore: create ~/.simple-secrets, load key + secrets
func LoadSecretsStore() (*SecretsStore, error) {
	dir, err := getConfigDirectory()
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(dir, 0700)

	s := &SecretsStore{
		KeyPath:     filepath.Join(dir, "master.key"),
		SecretsPath: filepath.Join(dir, "secrets.json"),
		secrets:     make(map[string]string),
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
	if _, err := os.Stat(s.SecretsPath); os.IsNotExist(err) {
		s.secrets = make(map[string]string)
		return nil
	}
	b, err := os.ReadFile(s.SecretsPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.secrets)
}

func (s *SecretsStore) saveSecrets() error {
	b, err := json.MarshalIndent(s.secrets, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.SecretsPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.SecretsPath)
}

// backupSecret creates an encrypted backup of a secret
func (s *SecretsStore) backupSecret(key, encryptedValue string) {
	backupPath := s.getBackupPath(key)
	s.createBackupDirectory()
	_ = os.WriteFile(backupPath, []byte(encryptedValue), 0600)
}

func (s *SecretsStore) Put(key, value string) error {
	encryptedValue, err := encrypt(s.masterKey, []byte(value))
	if err != nil {
		return err
	}

	s.ensureBackupExists(key, encryptedValue)
	s.secrets[key] = encryptedValue
	return s.saveSecrets()
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
	enc, ok := s.secrets[key]
	if !ok {
		return "", ErrNotFound
	}
	pt, err := decrypt(s.masterKey, enc)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (s *SecretsStore) ListKeys() []string {
	keys := make([]string, 0, len(s.secrets))
	for k := range s.secrets {
		if !strings.HasPrefix(k, disabledPrefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *SecretsStore) Delete(key string) error {
	prevEnc, ok := s.secrets[key]
	if !ok {
		return ErrNotFound
	}
	// Store the encrypted data as backup (not plaintext!)
	s.backupSecret(key, prevEnc)

	delete(s.secrets, key)
	return s.saveSecrets()
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
	enc, ok := s.secrets[key]
	if !ok {
		return ErrNotFound
	}

	// Create backup before disabling
	s.backupSecret(key, enc)

	// Mark as disabled using JSON encoding to handle keys with special characters
	timestamp := time.Now().UnixNano()
	keyData := map[string]interface{}{
		"timestamp": timestamp,
		"key":       key,
	}
	keyJSON, _ := json.Marshal(keyData)
	disabledKey := disabledPrefix + string(keyJSON)

	s.secrets[disabledKey] = enc
	delete(s.secrets, key)

	return s.saveSecrets()
}

// buildDisabledSecretsMap creates a map from original key names to their disabled key names
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
	var keyData map[string]interface{}
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
	const notFound = -1
	underscorePosition := strings.Index(data, "_")
	if underscorePosition == notFound {
		return ""
	}
	positionAfterUnderscore := underscorePosition + 1
	return data[positionAfterUnderscore:]
}

// EnableSecret re-enables a previously disabled secret
func (s *SecretsStore) EnableSecret(key string) error {
	// Build a map of original keys to their disabled keys for efficient lookup
	disabledMap := s.buildDisabledSecretsMap()

	disabledKey, found := disabledMap[key]
	if !found {
		return fmt.Errorf("disabled secret '%s' not found", key)
	}

	// Move back to original key
	s.secrets[key] = s.secrets[disabledKey]
	delete(s.secrets, disabledKey)

	return s.saveSecrets()
}

// ListDisabledSecrets returns a list of disabled secret keys
func (s *SecretsStore) ListDisabledSecrets() []string {
	disabledMap := s.buildDisabledSecretsMap()

	disabled := make([]string, 0, len(disabledMap))
	for originalKey := range disabledMap {
		disabled = append(disabled, originalKey)
	}

	sort.Strings(disabled)
	return disabled
}

var ErrNotFound = os.ErrNotExist

// createBackupDirectory ensures the backup directory exists with secure permissions
func (s *SecretsStore) createBackupDirectory() {
	backupDir := s.getBackupDirectory()
	_ = os.MkdirAll(backupDir, 0700)
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
