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
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// loadOrCreateKey sets s.masterKey; creates the file if missing.
func (s *SecretsStore) loadOrCreateKey() error {
	if !s.storage.Exists(s.KeyPath) {
		key := make([]byte, 32) // AES-256
		if _, err := rand.Read(key); err != nil {
			return err
		}
		if err := s.writeMasterKey(key); err != nil {
			return err
		}
		s.masterKey = key
		return nil
	}

	data, err := s.storage.ReadFile(s.KeyPath)
	if err != nil {
		return err
	}

	key, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return fmt.Errorf("master key file appears corrupted - try restoring from backup or removing ~/.simple-secrets/ to start fresh: %w", err)
	}

	s.masterKey = key
	return nil
}

// writeMasterKey overwrites the key file (0600).
func (s *SecretsStore) writeMasterKey(newKey []byte) error {
	return s.writeMasterKeyToPath(s.KeyPath, newKey)
}

// writeMasterKeyToPath writes a master key to the specified path atomically.
func (s *SecretsStore) writeMasterKeyToPath(path string, key []byte) error {
	enc := base64.StdEncoding.EncodeToString(key)
	return s.storage.AtomicWriteFile(path, []byte(enc), FileMode(0600))
}
