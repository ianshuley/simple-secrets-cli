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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// LoadUsersOrShowFirstRunMessage loads users or returns a first-run error with helpful message
// This should be used by most commands instead of LoadUsers to avoid unexpected auto-setup
func LoadUsersOrShowFirstRunMessage() (*UserStore, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, err
	}

	users, err := loadUsers(usersPath)
	if os.IsNotExist(err) {
		// Return first-run error instead of auto-triggering setup
		return nil, ErrFirstRunRequired
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

// LoadUsers loads users and handles first-run setup (for setup command only)
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
		store, token, err := HandleFirstRunSetup(usersPath, rolesPath)
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

// RotateUserToken generates a new token for a user
func (us *UserStore) RotateUserToken(username string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	for _, u := range us.users {
		if u.Username == username {
			// Generate new token
			token, err := generateSecureToken()
			if err != nil {
				return "", fmt.Errorf("failed to generate token: %w", err)
			}

			// Update the user's token hash
			u.TokenHash = HashToken(token)
			now := time.Now()
			u.TokenRotatedAt = &now

			return token, nil
		}
	}

	return "", fmt.Errorf("user %q not found", username)
}

func (us *UserStore) DisableUserToken(username string) error {
	us.mu.Lock()
	defer us.mu.Unlock()

	for _, u := range us.users {
		if u.Username == username {
			u.DisableToken()
			return nil
		}
	}

	return fmt.Errorf("user %q not found", username)
}

// DisableUserByToken disables a user by their token value and returns the username (no authentication - used by service layer)
func (us *UserStore) DisableUserByToken(tokenValue string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	tokenHash := HashToken(tokenValue)
	for _, u := range us.users {
		if u.TokenHash == tokenHash {
			u.DisableToken()
			return u.Username, nil
		}
	}
	return "", fmt.Errorf("token not found or already disabled")
}

// EnableUserToken generates a new token for a disabled user (no authentication - used by service layer)
func (us *UserStore) EnableUserToken(username string) (string, error) {
	us.mu.Lock()
	defer us.mu.Unlock()

	// Find the target user
	var targetUser *User
	for _, u := range us.users {
		if u.Username == username {
			targetUser = u
			break
		}
	}

	if targetUser == nil {
		return "", fmt.Errorf("user '%s' not found", username)
	}

	// Check if user is actually disabled (has empty token hash)
	if targetUser.TokenHash != "" {
		return "", fmt.Errorf("user '%s' is not disabled - use 'rotate token' to generate a new token for active users", username)
	}

	// Generate new token
	newToken, err := generateSecureToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	// Update user with new token hash
	targetUser.TokenHash = HashToken(newToken)
	now := time.Now()
	targetUser.TokenRotatedAt = &now

	return newToken, nil
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

// RBAC Configuration File Management

// resolveConfigPaths determines the file paths for users.json and roles.json
func resolveConfigPaths() (string, string, error) {
	usersPath, err := DefaultUserConfigPath("users.json")
	if err != nil {
		return "", "", err
	}

	rolesPath, err := DefaultUserConfigPath("roles.json")
	if err != nil {
		return "", "", err
	}

	return usersPath, rolesPath, nil
}

// loadUsers reads and validates users from the specified JSON file
func loadUsers(path string) ([]*User, error) {
	var users []*User
	if err := readConfigFile(path, &users); err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("users.json is corrupted or invalid: %w; please fix or delete the file", err)
	}

	return validateUsersList(users)
}

// loadRoles reads role permissions from the specified JSON file
func loadRoles(path string) (RolePermissions, error) {
	var perms RolePermissions
	if err := readConfigFile(path, &perms); err != nil {
		return nil, fmt.Errorf("unmarshal roles.json: %w", err)
	}
	return perms, nil
}

// readConfigFile reads and unmarshals a JSON config file
func readConfigFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// writeConfigFiles writes users and roles to their respective JSON files
func writeConfigFiles(usersPath, rolesPath string, users []*User, roles RolePermissions) error {
	if err := ensureConfigDirectory(usersPath); err != nil {
		return err
	}

	if err := writeConfigFileSecurely(usersPath, users); err != nil {
		return err
	}

	return writeConfigFileSecurely(rolesPath, roles)
}

// writeConfigFileSecurely marshals and writes any config data to JSON with secure permissions
func writeConfigFileSecurely(path string, data any) error {
	encoded, err := marshalConfigData(data)
	if err != nil {
		return err
	}
	return AtomicWriteFile(path, encoded, 0600)
}

// ensureConfigDirectory creates the configuration directory if it doesn't exist
func ensureConfigDirectory(usersPath string) error {
	return os.MkdirAll(filepath.Dir(usersPath), 0700)
}

// marshalConfigData converts config data to formatted JSON
func marshalConfigData(data any) ([]byte, error) {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config data: %w", err)
	}
	return encoded, nil
}

// validateUsersList ensures users list meets business rules
func validateUsersList(users []*User) ([]*User, error) {
	if err := checkForDuplicateUsernames(users); err != nil {
		return nil, err
	}

	if err := ensureAdminExists(users); err != nil {
		return nil, err
	}

	return users, nil
}

// checkForDuplicateUsernames validates that all usernames are unique
func checkForDuplicateUsernames(users []*User) error {
	usernameSet := make(map[string]struct{})
	for _, u := range users {
		if _, exists := usernameSet[u.Username]; exists {
			return fmt.Errorf("duplicate username found: %s", u.Username)
		}
		usernameSet[u.Username] = struct{}{}
	}
	return nil
}

// ensureAdminExists validates that at least one admin user exists
func ensureAdminExists(users []*User) error {
	for _, u := range users {
		if u.Role == RoleAdmin {
			return nil
		}
	}
	return fmt.Errorf("no admin user found in users.json; please fix the file or recreate users")
}

// File I/O and configuration management
