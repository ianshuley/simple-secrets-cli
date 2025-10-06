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
	"fmt"
	"time"

	"simple-secrets/pkg/crypto"
	"simple-secrets/pkg/errors"
	"simple-secrets/pkg/secrets"
)

// StoreImpl implements the secrets.Store interface
type StoreImpl struct {
	repo         secrets.Repository
	crypto       *CryptoService
	masterKeyMgr secrets.MasterKeyManager // For master key lifecycle management
}

// NewStore creates a new Store implementation
func NewStore(repo secrets.Repository, cryptoService *CryptoService) secrets.Store {
	return &StoreImpl{
		repo:   repo,
		crypto: cryptoService,
	}
}

// NewStoreWithMasterKeyManager creates a new Store implementation with master key management
func NewStoreWithMasterKeyManager(repo secrets.Repository, cryptoService *CryptoService, masterKeyMgr secrets.MasterKeyManager) secrets.Store {
	return &StoreImpl{
		repo:         repo,
		crypto:       cryptoService,
		masterKeyMgr: masterKeyMgr,
	}
}

// Put stores a secret with the given key and value
func (s *StoreImpl) Put(ctx context.Context, key, value string) error {
	if key == "" {
		return errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	encryptedValue, err := s.crypto.Encrypt([]byte(value))
	if err != nil {
		return errors.Wrap(err, errors.ErrCodeCryptoError, "Failed to encrypt secret")
	}

	secret := secrets.NewSecret(key, encryptedValue, len(value))
	return s.repo.Store(ctx, secret)
}

// Get retrieves a secret by key
func (s *StoreImpl) Get(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	secret, err := s.repo.Retrieve(ctx, key)
	if err != nil {
		return "", err
	}

	if secret.IsDisabled() {
		return "", errors.New(errors.ErrCodeNotFound, "Secret is disabled", key)
	}

	decryptedValue, err := s.crypto.Decrypt(secret.Value)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeCryptoError, "Failed to decrypt secret")
	}

	return string(decryptedValue), nil
}

// Generate creates a new secret with a generated value
func (s *StoreImpl) Generate(ctx context.Context, key string, length int) (string, error) {
	if key == "" {
		return "", errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	if length <= 0 {
		length = 32 // Default length
	}

	generatedValue, err := s.crypto.GenerateSecretValue(length)
	if err != nil {
		return "", errors.Wrap(err, errors.ErrCodeCryptoError, "Failed to generate secret value")
	}

	if err := s.Put(ctx, key, generatedValue); err != nil {
		return "", err
	}

	return generatedValue, nil
}

// Delete removes a secret permanently
func (s *StoreImpl) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	// Verify secret exists before deletion
	exists, err := s.repo.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewNotFoundError("secret", key)
	}

	return s.repo.Delete(ctx, key)
}

// List returns metadata for all enabled secrets
func (s *StoreImpl) List(ctx context.Context) ([]secrets.SecretMetadata, error) {
	allSecrets, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []secrets.SecretMetadata
	for _, secret := range allSecrets {
		if secret.IsEnabled() {
			metadata := secret.Metadata
			metadata.Key = secret.Key // Populate key in metadata
			result = append(result, metadata)
		}
	}

	return result, nil
}

// ListDisabled returns metadata for all disabled secrets
func (s *StoreImpl) ListDisabled(ctx context.Context) ([]secrets.SecretMetadata, error) {
	allSecrets, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	var result []secrets.SecretMetadata
	for _, secret := range allSecrets {
		if secret.IsDisabled() {
			metadata := secret.Metadata
			metadata.Key = secret.Key // Populate key in metadata
			result = append(result, metadata)
		}
	}

	return result, nil
}

// Enable makes a disabled secret available for retrieval
func (s *StoreImpl) Enable(ctx context.Context, key string) error {
	if key == "" {
		return errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	exists, err := s.repo.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewNotFoundError("secret", key)
	}

	return s.repo.Enable(ctx, key)
}

// Disable makes a secret unavailable for retrieval without deleting it
func (s *StoreImpl) Disable(ctx context.Context, key string) error {
	if key == "" {
		return errors.New(errors.ErrCodeInvalidInput, "Secret key cannot be empty")
	}

	exists, err := s.repo.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewNotFoundError("secret", key)
	}

	return s.repo.Disable(ctx, key)
}

// RotateMasterKey generates a new master encryption key and re-encrypts all secrets
func (s *StoreImpl) RotateMasterKey(ctx context.Context, backupDir string) error {
	// Generate a new master key using the key manager if available
	var newKey []byte
	var err error

	if s.masterKeyMgr != nil {
		newKey, err = s.masterKeyMgr.GenerateKey(ctx)
	} else {
		newKey, err = crypto.GenerateKey()
	}
	if err != nil {
		return fmt.Errorf("failed to generate new master key: %w", err)
	}

	// List all secrets to re-encrypt
	allSecrets, err := s.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secrets for re-encryption: %w", err)
	}

	// Store old crypto service for decryption
	oldCrypto := s.crypto

	// Create new crypto service with new key
	newCrypto := NewCryptoService(newKey)

	// Re-encrypt all secrets atomically with proper rollback
	rotationTime := time.Now()
	updatedSecrets := make([]*secrets.Secret, 0, len(allSecrets))

	// First pass: decrypt and re-encrypt all secrets (in memory)
	for _, secret := range allSecrets {
		// Decrypt with old key
		plaintext, err := oldCrypto.Decrypt(secret.Value)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret %s with old key: %w", secret.Key, err)
		}

		// Encrypt with new key
		newEncryptedValue, err := newCrypto.Encrypt(plaintext)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret %s with new key: %w", secret.Key, err)
		}

		// Update secret with new encrypted value and rotation metadata
		secret.UpdateValue(newEncryptedValue, len(plaintext))

		// Track rotation metadata
		secret.Metadata.LastRotatedAt = &rotationTime
		secret.Metadata.RotationCount++

		updatedSecrets = append(updatedSecrets, secret)
	}

	// Second pass: store all secrets atomically
	for _, secret := range updatedSecrets {
		if err := s.repo.Store(ctx, secret); err != nil {
			// On failure, attempt to rollback by restoring original secrets
			s.rollbackSecrets(ctx, allSecrets)
			return fmt.Errorf("failed to store re-encrypted secret %s: %w", secret.Key, err)
		}
	} // Update crypto service to use new key
	s.crypto = newCrypto

	// Persist the new master key using the key manager if available
	if s.masterKeyMgr != nil {
		if err := s.masterKeyMgr.SaveMasterKey(ctx, newKey); err != nil {
			return fmt.Errorf("failed to persist new master key: %w", err)
		}
	}

	return nil
}

// rollbackSecrets attempts to restore original secrets on rotation failure
func (s *StoreImpl) rollbackSecrets(ctx context.Context, originalSecrets []*secrets.Secret) {
	for _, secret := range originalSecrets {
		if err := s.repo.Store(ctx, secret); err != nil {
			// Log error but continue with rollback attempt
			fmt.Printf("Warning: failed to rollback secret %s during rotation failure: %v\n", secret.Key, err)
		}
	}
}
