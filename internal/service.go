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
	"fmt"
	"os"
)

// ServiceConfig holds configuration for service operations
type ServiceConfig struct {
	StorageBackend StorageBackend
	ConfigDir      string // Override default config directory (for testing/deployment)
}

// ServiceOption configures the service
type ServiceOption func(*ServiceConfig)

// WithStorageBackend sets the storage backend
func WithStorageBackend(backend StorageBackend) ServiceOption {
	return func(config *ServiceConfig) {
		config.StorageBackend = backend
	}
}

// WithConfigDir sets a custom configuration directory
func WithConfigDir(dir string) ServiceOption {
	return func(config *ServiceConfig) {
		config.ConfigDir = dir
	}
}

// NewServiceConfig creates a service config with functional options
func NewServiceConfig(options ...ServiceOption) *ServiceConfig {
	config := &ServiceConfig{
		StorageBackend: NewFilesystemBackend(),
		ConfigDir:      "", // Use default unless overridden
	}

	for _, option := range options {
		option(config)
	}

	return config
}

// SecretOperations defines operations for secret management
type SecretOperations interface {
	Get(token, key string) (string, error)
	Put(token, key, value string) error
	Delete(token, key string) error
	List(token string) ([]string, error)
	Enable(token, key string) error
	Disable(token, key string) error
}

// AuthOperations defines operations for authentication
type AuthOperations interface {
	ValidateToken(token string) (*User, error)
	ValidateAccess(token string, needWrite bool) error
}

// UserOperations defines operations for user management
type UserOperations interface {
	CreateUser(adminToken, username, role string) (string, error)
	DeleteUser(adminToken, username string) error
	ListUsers(adminToken string) ([]*User, error)
	RotateToken(token, username string) (string, error)
}

// Service provides composable operations for simple-secrets
// Clean separation of concerns with injectable dependencies
type Service struct {
	secrets SecretOperations
	auth    AuthOperations
	users   UserOperations
}

// NewService creates a service with functional options
func NewService(options ...ServiceOption) (*Service, error) {
	config := NewServiceConfig(options...)

	// Create auth operations FIRST to handle first-run setup
	// This ensures first-run detection happens before any files are created
	authOps, err := newAuthOperations(config)
	if err != nil {
		return nil, err
	}

	// Create other operations after auth setup
	secretOps, err := newSecretOperations(config)
	if err != nil {
		return nil, err
	}

	userOps, err := newUserOperations(config)
	if err != nil {
		return nil, err
	}

	return &Service{
		secrets: secretOps,
		auth:    authOps,
		users:   userOps,
	}, nil
}

// Secrets returns the secret operations interface
func (s *Service) Secrets() SecretOperations {
	return s.secrets
}

// Auth returns the authentication operations interface
func (s *Service) Auth() AuthOperations {
	return s.auth
}

// Users returns the user operations interface
func (s *Service) Users() UserOperations {
	return s.users
}

// Implementation structs for the focused interfaces
type secretOperations struct {
	store *SecretsStore
	auth  AuthOperations
}

type authOperations struct {
	userStore *UserStore
}

type userOperations struct {
	userStore *UserStore
	auth      AuthOperations
}

// newSecretOperations creates the secret operations implementation
func newSecretOperations(config *ServiceConfig) (SecretOperations, error) {
	var store *SecretsStore
	var err error

	if config.ConfigDir != "" {
		store, err = LoadSecretsStoreFromDir(config.StorageBackend, config.ConfigDir)
	} else {
		store, err = LoadSecretsStore(config.StorageBackend)
	}

	if err != nil {
		return nil, err
	}

	auth, err := newAuthOperations(config)
	if err != nil {
		return nil, err
	}

	return &secretOperations{
		store: store,
		auth:  auth,
	}, nil
}

// newAuthOperations creates the auth operations implementation
func newAuthOperations(config *ServiceConfig) (AuthOperations, error) {
	var userStore *UserStore
	var err error

	if config.ConfigDir != "" {
		usersPath := config.ConfigDir + "/users.json"
		rolesPath := config.ConfigDir + "/roles.json"

		users, err := loadUsers(usersPath)
		if os.IsNotExist(err) {
			// Return first-run error instead of auto-triggering setup
			return nil, ErrFirstRunRequired
		} else if err != nil {
			return nil, err
		} else {
			permissions, err := loadRoles(rolesPath)
			if err != nil {
				return nil, err
			}
			userStore = createUserStore(users, permissions)
		}
	} else {
		userStore, err = LoadUsersOrShowFirstRunMessage()
		if err != nil {
			return nil, err
		}
	}

	return &authOperations{
		userStore: userStore,
	}, nil
}

// newUserOperations creates the user operations implementation
func newUserOperations(config *ServiceConfig) (UserOperations, error) {
	userStore, err := newAuthOperations(config)
	if err != nil {
		return nil, err
	}

	auth := userStore.(*authOperations)

	return &userOperations{
		userStore: auth.userStore,
		auth:      auth,
	}, nil
}

// Implementation of SecretOperations interface
func (s *secretOperations) Get(token, key string) (string, error) {
	if _, err := s.auth.ValidateToken(token); err != nil {
		return "", err
	}

	return s.store.Get(key)
}

func (s *secretOperations) Put(token, key, value string) error {
	if err := s.auth.ValidateAccess(token, true); err != nil {
		return err
	}

	return s.store.Put(key, value)
}

func (s *secretOperations) Delete(token, key string) error {
	if err := s.auth.ValidateAccess(token, true); err != nil {
		return err
	}

	return s.store.Delete(key)
}

func (s *secretOperations) List(token string) ([]string, error) {
	if _, err := s.auth.ValidateToken(token); err != nil {
		return nil, err
	}

	return s.store.ListKeys(), nil
}

func (s *secretOperations) Enable(token, key string) error {
	if err := s.auth.ValidateAccess(token, true); err != nil {
		return err
	}

	return s.store.EnableSecret(key)
}

func (s *secretOperations) Disable(token, key string) error {
	if err := s.auth.ValidateAccess(token, true); err != nil {
		return err
	}

	return s.store.DisableSecret(key)
}

// Implementation of AuthOperations interface
func (a *authOperations) ValidateToken(token string) (*User, error) {
	return a.userStore.Lookup(token)
}

func (a *authOperations) ValidateAccess(token string, needWrite bool) error {
	user, err := a.ValidateToken(token)
	if err != nil {
		return err
	}

	if needWrite && !user.Can("write", a.userStore.Permissions()) {
		return fmt.Errorf("permission denied: need 'write' permission")
	}

	return nil
}

// Implementation of UserOperations interface
func (u *userOperations) CreateUser(adminToken, username, role string) (string, error) {
	if err := u.auth.ValidateAccess(adminToken, true); err != nil {
		return "", err
	}

	admin, err := u.auth.ValidateToken(adminToken)
	if err != nil {
		return "", err
	}

	if admin.Role != "admin" {
		return "", fmt.Errorf("permission denied: admin role required")
	}

	return u.userStore.CreateUser(username, role)
}

func (u *userOperations) DeleteUser(adminToken, username string) error {
	if err := u.auth.ValidateAccess(adminToken, true); err != nil {
		return err
	}

	admin, err := u.auth.ValidateToken(adminToken)
	if err != nil {
		return err
	}

	if admin.Role != "admin" {
		return fmt.Errorf("permission denied: admin role required")
	}

	return u.userStore.DeleteUser(username)
}

func (u *userOperations) ListUsers(adminToken string) ([]*User, error) {
	admin, err := u.auth.ValidateToken(adminToken)
	if err != nil {
		return nil, err
	}

	if admin.Role != "admin" {
		return nil, fmt.Errorf("permission denied: admin role required")
	}

	return u.userStore.Users(), nil
}

func (u *userOperations) RotateToken(token, username string) (string, error) {
	user, err := u.auth.ValidateToken(token)
	if err != nil {
		return "", err
	}

	// Users can rotate their own tokens, admins can rotate any
	if user.Username != username && user.Role != "admin" {
		return "", fmt.Errorf("permission denied: can only rotate own token")
	}

	return u.userStore.RotateUserToken(username)
}
