/*package platform

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

// Package platform provides service composition for the CLI application.
// It wires together all domain services (secrets, users, auth, rotation) into
// a unified platform interface that CLI commands can use.
package platform

import (
	"context"
	"fmt"

	// Public domain interfaces
	"simple-secrets/pkg/auth"
	"simple-secrets/pkg/rotation"
	"simple-secrets/pkg/secrets"
	"simple-secrets/pkg/users"

	// Private domain implementations
	authImpl "simple-secrets/internal/auth"
	rotationImpl "simple-secrets/internal/rotation"
	secretsImpl "simple-secrets/internal/secrets"
	usersImpl "simple-secrets/internal/users"
)

// Platform represents the complete service composition for the CLI application.
// It provides access to all domain services through their public interfaces.
type Platform struct {
	// Secrets domain service for secret storage and management
	Secrets secrets.Store

	// Users domain service for user management and multi-token support
	Users users.Store

	// Auth domain service for authentication and authorization
	Auth auth.AuthService

	// Rotation domain service for backup management and key rotation
	Rotation rotation.Service
}

// Config holds configuration needed to create a Platform instance
type Config struct {
	// DataDir is the base directory for all data storage
	DataDir string

	// MasterKey is the encryption key for secret storage
	MasterKey []byte

	// BackupDir is the directory for storing backups (optional, defaults to dataDir/backups)
	BackupDir string
}

// New creates a new Platform instance with all services properly wired together.
// This is the main factory function that composes all domain services.
func New(ctx context.Context, config Config) (*Platform, error) {
	if config.DataDir == "" {
		return nil, fmt.Errorf("dataDir is required")
	}
	if len(config.MasterKey) == 0 {
		return nil, fmt.Errorf("masterKey is required")
	}

	// Create crypto service for secrets domain
	cryptoService := secretsImpl.NewCryptoService(config.MasterKey)

	// Create repositories
	secretsRepo := secretsImpl.NewFileRepository(config.DataDir)
	usersRepo := usersImpl.NewFileRepository(config.DataDir)

	// Create domain stores
	secretsStore := secretsImpl.NewStore(secretsRepo, cryptoService)
	usersStore := usersImpl.NewStore(usersRepo)

	// Create auth service (depends on users store)
	authService := authImpl.NewServiceWithDefaults(usersStore)

	// Create rotation service (depends on secrets and users stores)
	rotationService := rotationImpl.NewService(secretsStore, usersStore, config.DataDir)

	return &Platform{
		Secrets:  secretsStore,
		Users:    usersStore,
		Auth:     authService,
		Rotation: rotationService,
	}, nil
}

// NewWithOptions creates a Platform with additional configuration options.
// This allows for more advanced service configuration while maintaining
// the same service composition pattern.
func NewWithOptions(ctx context.Context, config Config, opts ...Option) (*Platform, error) {
	platform, err := New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Apply configuration options
	for _, opt := range opts {
		if err := opt(platform); err != nil {
			return nil, fmt.Errorf("failed to apply platform option: %w", err)
		}
	}

	return platform, nil
}

// Option is a function that can modify a Platform instance
type Option func(*Platform) error

// WithCustomRotationConfig replaces the default rotation service with one
// using custom configuration.
func WithCustomRotationConfig(config *rotation.RotationConfig) Option {
	return func(p *Platform) error {
		// Extract dependencies from existing rotation service
		secretsStore := p.Secrets
		usersStore := p.Users

		// Get data directory - this is a bit hacky but works for now
		// In a more sophisticated version, we'd store this in Platform
		dataDir := "" // Will use default behavior in NewServiceWithConfig

		// Replace rotation service with custom config version
		p.Rotation = rotationImpl.NewServiceWithConfig(secretsStore, usersStore, dataDir, config)
		return nil
	}
}

// Close gracefully shuts down the platform and all its services.
// Currently a no-op since our services don't require cleanup,
// but provides a hook for future resource management.
func (p *Platform) Close() error {
	// Future: close database connections, file handles, etc.
	return nil
}

// Health checks the health of all platform services.
// Returns an error if any critical service is unavailable.
func (p *Platform) Health(ctx context.Context) error {
	// Basic health check - try to list secrets (lightweight operation)
	_, err := p.Secrets.List(ctx)
	if err != nil {
		return fmt.Errorf("secrets service unhealthy: %w", err)
	}

	// Check users service
	_, err = p.Users.List(ctx)
	if err != nil {
		return fmt.Errorf("users service unhealthy: %w", err)
	}

	// Auth and rotation services don't have simple health checks,
	// but if secrets and users work, they should work too
	return nil
}
