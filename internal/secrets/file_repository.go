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
	"sync"
	"time"

	"simple-secrets/pkg/errors"
	"simple-secrets/pkg/secrets"
)

const (
	fileSecureFilePermissions      = 0600 // Owner read/write only
	fileSecureDirectoryPermissions = 0700 // Owner read/write/execute only
)

// FileRepository implements the secrets.Repository interface using the filesystem
type FileRepository struct {
	dataDir     string
	secretsFile string
	mu          sync.RWMutex // Protects concurrent access to secrets data
}

// NewFileRepository creates a new file-based repository
func NewFileRepository(dataDir string) secrets.Repository {
	return &FileRepository{
		dataDir:     dataDir,
		secretsFile: filepath.Join(dataDir, "secrets.json"),
	}
}

// secretsData represents the structure stored in secrets.json
type secretsData struct {
	Secrets map[string]*secrets.Secret `json:"secrets"`
	Version string                     `json:"version"`
}

// Store persists a secret to the filesystem
func (r *FileRepository) Store(ctx context.Context, secret *secrets.Secret) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := r.ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Use write lock for read-modify-write operation
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return err
	}

	data.Secrets[secret.Key] = secret
	return r.saveSecretsData(data)
}

// Retrieve gets a secret by key
func (r *FileRepository) Retrieve(ctx context.Context, key string) (*secrets.Secret, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use read lock for data access
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return nil, err
	}

	secret, exists := data.Secrets[key]
	if !exists {
		return nil, errors.NewNotFoundError("secret", key)
	}

	return secret, nil
}

// Delete removes a secret permanently
func (r *FileRepository) Delete(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Use write lock for read-modify-write operation
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return err
	}

	if _, exists := data.Secrets[key]; !exists {
		return errors.NewNotFoundError("secret", key)
	}

	delete(data.Secrets, key)
	return r.saveSecretsData(data)
}

// List returns all secrets (enabled and disabled)
func (r *FileRepository) List(ctx context.Context) ([]*secrets.Secret, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Use read lock for data access
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return nil, err
	}

	result := make([]*secrets.Secret, 0, len(data.Secrets))
	for _, secret := range data.Secrets {
		result = append(result, secret)
	}

	return result, nil
}

// Enable marks a secret as enabled
func (r *FileRepository) Enable(ctx context.Context, key string) error {
	return r.updateSecretStatus(ctx, key, false)
}

// Disable marks a secret as disabled
func (r *FileRepository) Disable(ctx context.Context, key string) error {
	return r.updateSecretStatus(ctx, key, true)
}

// Exists checks if a secret exists
func (r *FileRepository) Exists(ctx context.Context, key string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	// Use read lock for data access
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return false, err
	}

	_, exists := data.Secrets[key]
	return exists, nil
}

// updateSecretStatus updates the disabled status of a secret
func (r *FileRepository) updateSecretStatus(ctx context.Context, key string, disabled bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Use write lock for read-modify-write operation
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.loadSecretsData()
	if err != nil {
		return err
	}

	secret, exists := data.Secrets[key]
	if !exists {
		return errors.NewNotFoundError("secret", key)
	}

	secret.Metadata.Disabled = disabled
	secret.Metadata.ModifiedAt = time.Now()

	return r.saveSecretsData(data)
}

// loadSecretsData loads secrets from the filesystem
func (r *FileRepository) loadSecretsData() (*secretsData, error) {
	if _, err := os.Stat(r.secretsFile); os.IsNotExist(err) {
		return &secretsData{
			Secrets: make(map[string]*secrets.Secret),
			Version: "1.0",
		}, nil
	}

	data, err := os.ReadFile(r.secretsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	var result secretsData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse secrets file: %w", err)
	}

	if result.Secrets == nil {
		result.Secrets = make(map[string]*secrets.Secret)
	}

	return &result, nil
}

// saveSecretsData saves secrets to the filesystem atomically
func (r *FileRepository) saveSecretsData(data *secretsData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets data: %w", err)
	}

	return r.atomicWriteFile(r.secretsFile, jsonData, fileSecureFilePermissions)
}

// ensureDataDir creates the data directory if it doesn't exist
func (r *FileRepository) ensureDataDir() error {
	return os.MkdirAll(r.dataDir, fileSecureDirectoryPermissions)
}

// atomicWriteFile writes data to a file atomically using a temporary file and rename
func (r *FileRepository) atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
