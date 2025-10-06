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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"simple-secrets/pkg/crypto"
	"simple-secrets/pkg/secrets"
)

// FileMasterKeyManager implements master key management using file system storage
type FileMasterKeyManager struct {
	keyFilePath string
}

// NewFileMasterKeyManager creates a new file-based master key manager
func NewFileMasterKeyManager(dataDir string) secrets.MasterKeyManager {
	return &FileMasterKeyManager{
		keyFilePath: filepath.Join(dataDir, "master.key"),
	}
}

// LoadMasterKey loads the master key from the file system
func (m *FileMasterKeyManager) LoadMasterKey(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(m.keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read master key file: %w", err)
	}

	key, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	return key, nil
}

// SaveMasterKey persists a master key to the file system atomically
func (m *FileMasterKeyManager) SaveMasterKey(ctx context.Context, key []byte) error {
	enc := base64.StdEncoding.EncodeToString(key)

	// Write to temporary file first for atomic operation
	tmpPath := m.keyFilePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(enc), 0600); err != nil {
		return fmt.Errorf("failed to write temporary key file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, m.keyFilePath); err != nil {
		_ = os.Remove(tmpPath) // Clean up on error
		return fmt.Errorf("failed to move new key file into place: %w", err)
	}

	return nil
}

// GenerateKey creates a new cryptographically secure master key
func (m *FileMasterKeyManager) GenerateKey(ctx context.Context) ([]byte, error) {
	return crypto.GenerateKey()
}
