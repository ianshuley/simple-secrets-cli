/*package secrets

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

// Package secrets provides the public interfaces for the secrets domain.
// These interfaces are designed for platform extension and are considered stable contracts.
package secrets

import "context"

// Store is the main interface for secret operations that the platform will use.
// This interface abstracts all secret management functionality and is designed
// to be extended by the platform for audit logging, caching, etc.
type Store interface {
	// Put stores a secret with the given key and value
	Put(ctx context.Context, key, value string) error

	// Get retrieves a secret by key, returning an error if not found or disabled
	Get(ctx context.Context, key string) (string, error)

	// Generate creates a new secret with a generated value of the specified length
	Generate(ctx context.Context, key string, length int) (string, error)

	// Delete removes a secret permanently (creates backup first)
	Delete(ctx context.Context, key string) error

	// List returns metadata for all enabled secrets
	List(ctx context.Context) ([]SecretMetadata, error)

	// ListDisabled returns metadata for all disabled secrets
	ListDisabled(ctx context.Context) ([]SecretMetadata, error)

	// Enable makes a disabled secret available for retrieval
	Enable(ctx context.Context, key string) error

	// Disable makes a secret unavailable for retrieval without deleting it
	Disable(ctx context.Context, key string) error

	// RotateMasterKey generates a new master encryption key and re-encrypts all secrets
	// This is a fundamental secrets domain operation that affects all stored secrets
	RotateMasterKey(ctx context.Context, backupDir string) error
}

// Repository is the storage interface for the secrets domain.
// This interface defines business operations, not technical file operations.
// Different implementations can use files, databases, cloud storage, etc.
type Repository interface {
	// Store persists a secret
	Store(ctx context.Context, secret *Secret) error

	// Retrieve gets a secret by key
	Retrieve(ctx context.Context, key string) (*Secret, error)

	// Delete removes a secret permanently
	Delete(ctx context.Context, key string) error

	// List returns metadata for all secrets (enabled and disabled)
	List(ctx context.Context) ([]*Secret, error)

	// Enable marks a secret as enabled
	Enable(ctx context.Context, key string) error

	// Disable marks a secret as disabled
	Disable(ctx context.Context, key string) error

	// Exists checks if a secret exists (regardless of enabled/disabled state)
	Exists(ctx context.Context, key string) (bool, error)
}
