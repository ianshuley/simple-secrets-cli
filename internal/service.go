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

	"simple-secrets/pkg/api"
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
	Generate(token, key string, length int) (string, error)
	Delete(token, key string) error
	List(token string) ([]string, error)
	ListDisabled(token string) ([]string, error)
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
	RotateSelfToken(currentUser *User) (string, error)
	DisableUser(token, username string) error
	DisableUserByToken(adminToken, targetToken string) (string, error)
	EnableUser(token, username string) (string, error)
}

// Service provides composable operations for simple-secrets
// Clean separation of concerns with injectable dependencies
type Service struct {
	secrets SecretOperations
	auth    AuthOperations
	users   UserOperations
	admin   api.AdminOperations
}

// NewService creates a service with functional options
func NewService(options ...ServiceOption) (*Service, error) {
	config := NewServiceConfig(options...)

	// Create shared stores first
	var secretsStore *SecretsStore
	var userStore *UserStore
	var err error

	// Create secrets store
	secretsStore, err = createSecretsStore(config)
	if err != nil {
		return nil, err
	}

	// Create user store
	userStore, err = createUserStoreFromConfig(config)
	if err != nil {
		return nil, err
	}

	// Create auth operations
	authOps := &authOperations{
		userStore: userStore,
	}

	// Create other operations using shared stores
	secretOps := &secretOperations{
		store: secretsStore,
		auth:  authOps,
	}

	// Determine config directory for file paths
	configDir := config.ConfigDir
	if configDir == "" {
		defaultDir, err := GetSimpleSecretsPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get config directory: %w", err)
		}
		configDir = defaultDir
	}

	userOps := &userOperations{
		userStore: userStore,
		auth:      authOps,
		usersPath: filepath.Join(configDir, "users.json"),
		rolesPath: filepath.Join(configDir, "roles.json"),
	}

	// Create admin operations using shared stores
	adminOps := NewServiceAdapter(secretsStore, userStore)

	return &Service{
		secrets: secretOps,
		auth:    authOps,
		users:   userOps,
		admin:   adminOps,
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

// Admin returns the admin operations interface
func (s *Service) Admin() api.AdminOperations {
	return s.admin
}

// GetUserStore returns the underlying UserStore for CLI compatibility
// This is a bridge method to avoid tight coupling in CLI commands
func (s *Service) GetUserStore() *UserStore {
	// Access the userStore through the userOperations implementation
	if userOps, ok := s.users.(*userOperations); ok {
		return userOps.userStore
	}
	return nil
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
	usersPath string
	rolesPath string
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

func (s *secretOperations) Generate(token, key string, length int) (string, error) {
	if err := s.auth.ValidateAccess(token, true); err != nil {
		return "", err
	}

	// Generate the secret value
	generatedValue, err := GenerateSecretValue(length)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Store the generated value
	err = s.store.Put(key, generatedValue)
	if err != nil {
		return "", fmt.Errorf("failed to store generated secret: %w", err)
	}

	return generatedValue, nil
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

func (s *secretOperations) ListDisabled(token string) ([]string, error) {
	if _, err := s.auth.ValidateToken(token); err != nil {
		return nil, err
	}

	return s.store.ListDisabledSecrets(), nil
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

	newToken, err := u.userStore.CreateUser(username, role)
	if err != nil {
		return "", err
	}

	// Save the updated users to disk
	if err := u.saveUsersWithError(); err != nil {
		return "", err
	}

	return newToken, nil
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

	if err := u.userStore.DeleteUser(username); err != nil {
		return err
	}

	// Save the updated users to disk
	return u.saveUsersWithError()
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

	newToken, err := u.userStore.RotateUserToken(username)
	if err != nil {
		return "", err
	}

	// Save the updated users to disk
	if err := u.saveUsersWithError(); err != nil {
		return "", err
	}

	return newToken, nil
}

func (u *userOperations) DisableUser(token, username string) error {
	user, err := u.auth.ValidateToken(token)
	if err != nil {
		return err
	}

	if !user.Can("rotate-tokens", u.userStore.Permissions()) {
		return fmt.Errorf("permission denied: require rotate-tokens permission to disable user tokens")
	}

	if err := u.userStore.DisableUserToken(username); err != nil {
		return err
	}

	// Save the updated users to disk
	return u.saveUsersWithError()
}

func (u *userOperations) DisableUserByToken(token, tokenValue string) (string, error) {
	// Verify authentication and permissions
	if err := u.auth.ValidateAccess(token, true); err != nil {
		return "", err
	}

	// Disable the user by token value
	username, err := u.userStore.DisableUserByToken(tokenValue)
	if err != nil {
		return "", err
	}

	// Save the updated users to disk
	if err := u.saveUsersWithError(); err != nil {
		return "", err
	}

	return username, nil
}

func (u *userOperations) EnableUser(token, username string) (string, error) {
	// Verify authentication and permissions
	if err := u.auth.ValidateAccess(token, true); err != nil {
		return "", err
	}

	// Generate new token for the disabled user
	newToken, err := u.userStore.EnableUserToken(username)
	if err != nil {
		return "", err
	}

	// Save the updated users to disk
	if err := u.saveUsersWithError(); err != nil {
		return "", err
	}

	return newToken, nil
}

func (u *userOperations) RotateSelfToken(currentUser *User) (string, error) {
	newToken, err := u.userStore.RotateUserToken(currentUser.Username)
	if err != nil {
		return "", err
	}

	// Save the updated users to disk
	if err := u.saveUsersWithError(); err != nil {
		return "", err
	}

	return newToken, nil
}

// saveUsers persists the current user list to disk
func (u *userOperations) saveUsers() error {
	users := u.userStore.Users()
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}
	return AtomicWriteFile(u.usersPath, data, 0600)
}

// saveUsersWithError wraps saveUsers with consistent error messaging
func (u *userOperations) saveUsersWithError() error {
	if err := u.saveUsers(); err != nil {
		return fmt.Errorf("failed to save user changes: %w", err)
	}
	return nil
}

// createSecretsStore creates a secrets store with the appropriate backend and directory
func createSecretsStore(config *ServiceConfig) (*SecretsStore, error) {
	if config.ConfigDir != "" {
		return LoadSecretsStoreFromDir(config.StorageBackend, config.ConfigDir)
	}

	return LoadSecretsStore(config.StorageBackend)
}

// createUserStoreFromConfig creates a user store based on the service configuration
func createUserStoreFromConfig(config *ServiceConfig) (*UserStore, error) {
	if config.ConfigDir != "" {
		return loadUserStoreFromConfigDir(config.ConfigDir)
	}

	return LoadUsersOrShowFirstRunMessage()
}

// loadUserStoreFromConfigDir loads user store from a specific configuration directory
func loadUserStoreFromConfigDir(configDir string) (*UserStore, error) {
	usersPath := filepath.Join(configDir, "users.json")
	rolesPath := filepath.Join(configDir, "roles.json")

	users, err := loadUsers(usersPath)
	if os.IsNotExist(err) {
		return nil, ErrFirstRunRequired
	}
	if err != nil {
		return nil, err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, err
	}

	return createUserStore(users, permissions), nil
}
