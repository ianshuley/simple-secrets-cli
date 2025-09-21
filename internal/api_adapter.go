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
	"simple-secrets/pkg/api"
)

// Ensure SecretsStore implements the API interfaces at compile time
var _ api.SecretReader = (*SecretsStore)(nil)
var _ api.SecretWriter = (*SecretsStore)(nil)

// List implements api.SecretReader.List by returning enabled secrets only
func (s *SecretsStore) List() []string {
	return s.ListKeys()
}

// Enable implements api.SecretWriter.Enable
func (s *SecretsStore) Enable(key string) error {
	return s.EnableSecret(key)
}

// Disable implements api.SecretWriter.Disable
func (s *SecretsStore) Disable(key string) error {
	return s.DisableSecret(key)
}

// ServiceAdapter adapts internal types to provide a complete API service
type ServiceAdapter struct {
	secrets *SecretsStore
	users   *UserStore
}

// NewServiceAdapter creates a new adapter that implements the full API interfaces
func NewServiceAdapter(secrets *SecretsStore, users *UserStore) *ServiceAdapter {
	return &ServiceAdapter{
		secrets: secrets,
		users:   users,
	}
}

// Ensure ServiceAdapter implements all the API interfaces at compile time
var _ api.SecretReader = (*ServiceAdapter)(nil)
var _ api.SecretWriter = (*ServiceAdapter)(nil)
var _ api.Authenticator = (*ServiceAdapter)(nil)
var _ api.UserManager = (*ServiceAdapter)(nil)
var _ api.AdminOperations = (*ServiceAdapter)(nil)
var _ api.FullService = (*ServiceAdapter)(nil)

// SecretReader interface implementation
func (sa *ServiceAdapter) Get(key string) (string, error) {
	return sa.secrets.Get(key)
}

func (sa *ServiceAdapter) List() []string {
	return sa.secrets.List()
}

func (sa *ServiceAdapter) IsEnabled(key string) bool {
	return sa.secrets.IsEnabled(key)
}

// SecretWriter interface implementation
func (sa *ServiceAdapter) Put(key, value string) error {
	return sa.secrets.Put(key, value)
}

func (sa *ServiceAdapter) Delete(key string) error {
	return sa.secrets.Delete(key)
}

func (sa *ServiceAdapter) Enable(key string) error {
	return sa.secrets.Enable(key)
}

func (sa *ServiceAdapter) Disable(key string) error {
	return sa.secrets.Disable(key)
}

// Authenticator interface implementation
func (sa *ServiceAdapter) Authenticate(token string) (*api.User, error) {
	user, err := sa.users.Lookup(token)
	if err != nil {
		return nil, err
	}
	return &api.User{
		Username: user.Username,
		Role:     string(user.Role),
	}, nil
}

func (sa *ServiceAdapter) CanRead(user *api.User) bool {
	// Both admin and reader roles can read
	return user.Role == "admin" || user.Role == "reader"
}

func (sa *ServiceAdapter) CanWrite(user *api.User) bool {
	// Only admin role can write
	return user.Role == "admin"
}

func (sa *ServiceAdapter) CanAdmin(user *api.User) bool {
	// Only admin role has admin permissions
	return user.Role == "admin"
}

// UserManager interface implementation
func (sa *ServiceAdapter) CreateUser(username, role string) (string, error) {
	return sa.users.CreateUser(username, role)
}

func (sa *ServiceAdapter) DeleteUser(username string) error {
	return sa.users.DeleteUser(username)
}

func (sa *ServiceAdapter) ListUsers() ([]*api.User, error) {
	internalUsers := sa.users.Users()
	apiUsers := make([]*api.User, len(internalUsers))

	for i, user := range internalUsers {
		apiUsers[i] = &api.User{
			Username: user.Username,
			Role:     string(user.Role),
		}
	}

	return apiUsers, nil
}

func (sa *ServiceAdapter) RotateToken(username string) (string, error) {
	// Find the user
	users := sa.users.Users()
	for _, user := range users {
		if user.Username == username {
			// Generate new token
			token, err := generateSecureToken()
			if err != nil {
				return "", err
			}

			// Update the user's token hash
			user.TokenHash = HashToken(token)

			return token, nil
		}
	}

	return "", fmt.Errorf("user %q not found", username)
}

// AdminOperations interface implementation
func (sa *ServiceAdapter) Backup(backupDir string) error {
	return sa.secrets.CreateBackup(backupDir)
}

func (sa *ServiceAdapter) Restore(backupDir string) error {
	return sa.secrets.RestoreFromBackup(backupDir)
}

func (sa *ServiceAdapter) RotateMasterKey(backupDir string) error {
	return sa.secrets.RotateMasterKey(backupDir)
}
