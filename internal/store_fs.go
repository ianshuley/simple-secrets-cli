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
	"os"
	"path/filepath"
	"sort"
)

type SecretsStore struct {
	KeyPath     string
	SecretsPath string
	masterKey   []byte
	secrets     map[string]string // key -> base64(ciphertext)
}

// LoadSecretsStore: create ~/.simple-secrets, load key + secrets
func LoadSecretsStore() (*SecretsStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".simple-secrets")
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
	home, _ := os.UserHomeDir()
	backupDir := filepath.Join(home, ".simple-secrets", "backups")
	_ = os.MkdirAll(backupDir, 0700)
	backupPath := filepath.Join(backupDir, key+".bak")
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

// ensureBackupExists creates a backup for the secret - either the previous value or the new value
func (s *SecretsStore) ensureBackupExists(key, newEncryptedValue string) {
	backupValue := s.determineBackupValue(key, newEncryptedValue)
	s.backupSecret(key, backupValue)
}

// determineBackupValue decides what to backup: previous value if it exists, otherwise the new value
func (s *SecretsStore) determineBackupValue(key, newEncryptedValue string) string {
	previousValue := newEncryptedValue // Default to backing up the new value
	if existingValue, hasExistingValue := s.secrets[key]; hasExistingValue {
		previousValue = existingValue // Override to backup the previous value
	}
	return previousValue
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
		keys = append(keys, k)
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

var ErrNotFound = os.ErrNotExist
