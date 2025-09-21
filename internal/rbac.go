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
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"time"
)

// UserStore manages users, their permissions, and authentication
type UserStore struct {
	users       []*User
	permissions RolePermissions
	mu          sync.RWMutex // protects users slice and permissions
}

// Users returns the list of users (for first-run detection)
func (us *UserStore) Users() []*User {
	us.mu.RLock()
	defer us.mu.RUnlock()
	return us.users
}

// LoadUsersList loads the user list from users.json (for CLI user creation).
func LoadUsersList(path string) ([]*User, error) {
	return loadUsers(path)
}

// LoadUsers loads users and roles. Returns (store, firstRun, token, error).
// Token is only set when firstRun is true.
func LoadUsers() (*UserStore, bool, string, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, false, "", err
	}

	users, err := loadUsers(usersPath)
	if os.IsNotExist(err) {
		// Check first-run eligibility
		if err := validateFirstRunEligibility(); err != nil {
			return nil, false, "", err
		}
		store, token, err := handleFirstRunWithToken(usersPath, rolesPath)
		if err != nil {
			return nil, false, "", err
		}
		return store, true, token, nil
	}
	if err != nil {
		return nil, false, "", err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, false, "", fmt.Errorf("load roles.json: %w", err)
	}

	store := createUserStore(users, permissions)
	return store, false, "", nil
}

// LoadUsersForAuth loads users for authentication purposes without triggering first-run setup.
// Returns (store, error). If users.json doesn't exist, returns a context-aware auth error.
func LoadUsersForAuth() (*UserStore, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, err
	}

	users, err := loadUsers(usersPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("authentication failed: invalid token or no users configured")
	}
	if err != nil {
		return nil, err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, fmt.Errorf("load roles.json: %w", err)
	}

	store := createUserStore(users, permissions)
	return store, nil
}

// UserStore methods for runtime operations

// Lookup finds a user by token
func (us *UserStore) Lookup(token string) (*User, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	tokenHash := HashToken(token)

	us.mu.RLock()
	defer us.mu.RUnlock()
	for _, u := range us.users {
		if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(u.TokenHash)) == 1 {
			return u, nil
		}
	}
	return nil, errors.New("invalid token")
}

// Permissions returns the role permissions
func (us *UserStore) Permissions() RolePermissions {
	us.mu.RLock()
	defer us.mu.RUnlock()
	return us.permissions
}

// CreateUser adds a new user to the store and returns a generated token
func (us *UserStore) CreateUser(username, role string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	// Check for duplicate username
	for _, u := range us.users {
		if u.Username == username {
			return "", fmt.Errorf("user %q already exists", username)
		}
	}

	// Parse role string
	var userRole Role
	switch role {
	case "admin":
		userRole = RoleAdmin
	case "reader":
		userRole = RoleReader
	default:
		return "", fmt.Errorf("invalid role %q: must be 'admin' or 'reader'", role)
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	tokenHash := HashToken(token)
	now := time.Now()

	newUser := &User{
		Username:       username,
		TokenHash:      tokenHash,
		Role:           userRole,
		TokenRotatedAt: &now,
	}

	us.users = append(us.users, newUser)
	return token, nil
}

// DeleteUser removes a user from the store
func (us *UserStore) DeleteUser(username string) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	for i, u := range us.users {
		if u.Username == username {
			// Prevent deleting the last admin user
			if u.Role == RoleAdmin && us.countAdminUsers() <= 1 {
				return fmt.Errorf("cannot delete the last admin user")
			}

			// Remove user from slice
			us.users = slices.Delete(us.users, i, i+1)
			return nil
		}
	}

	return fmt.Errorf("user %q not found", username)
}

// UpdateUserRole changes a user's role
func (us *UserStore) UpdateUserRole(username, newRole string) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	// Parse role string
	var role Role
	switch newRole {
	case "admin":
		role = RoleAdmin
	case "reader":
		role = RoleReader
	default:
		return fmt.Errorf("invalid role %q: must be 'admin' or 'reader'", newRole)
	}

	for _, u := range us.users {
		if u.Username == username {
			// Prevent changing the last admin user to a non-admin role
			if u.Role == RoleAdmin && role != RoleAdmin && us.countAdminUsers() <= 1 {
				return fmt.Errorf("cannot change role of the last admin user")
			}

			u.Role = role
			return nil
		}
	}

	return fmt.Errorf("user %q not found", username)
}

// Private helper functions

// createUserStore constructs a UserStore with the given users and permissions
func createUserStore(users []*User, permissions RolePermissions) *UserStore {
	return &UserStore{
		users:       users,
		permissions: permissions,
	}
}

// countAdminUsers returns the number of admin users (helper for validation)
func (us *UserStore) countAdminUsers() int {
	count := 0
	for _, u := range us.users {
		if u.Role == RoleAdmin {
			count++
		}
	}
	return count
}

// File I/O and configuration management
