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
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type UserStore struct {
	users       []*User
	permissions RolePermissions
}

// Users returns the list of users (for first-run detection)
func (us *UserStore) Users() []*User {
	return us.users
}

// HashToken exports the hashToken function for use in CLI user creation.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// DefaultUserConfigPath exports defaultUserConfigPath for CLI use.
func DefaultUserConfigPath(filename string) (string, error) {
	// Check for test override first
	if testDir := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR"); testDir != "" {
		return filepath.Join(testDir, filename), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home dir: %w", err)
	}
	return filepath.Join(home, ".simple-secrets", filename), nil
}

// LoadUsersList loads the user list from users.json (for CLI user creation).
func LoadUsersList(path string) ([]*User, error) {
	return loadUsers(path)
}

// ResolveToken returns the token from CLI flag, env, or config file (in that order).
func ResolveToken(cliFlag string) (string, error) {
	if cliFlag != "" {
		return cliFlag, nil
	}

	if env := os.Getenv("SIMPLE_SECRETS_TOKEN"); env != "" {
		return env, nil
	}

	configPath, err := DefaultUserConfigPath("config.json")
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("authentication required: provide a token via --token, SIMPLE_SECRETS_TOKEN, or ~/.simple-secrets/config.json")
		}
		return "", fmt.Errorf("read config.json: %w", err)
	}

	var config struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("unmarshal config.json: %w", err)
	}
	if config.Token == "" {
		return "", errors.New("token not found in config.json")
	}

	return config.Token, nil
}

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleReader Role = "reader"
)

type User struct {
	Username       string     `json:"username"`
	TokenHash      string     `json:"token_hash"` // SHA-256 hash, base64-encoded
	Role           Role       `json:"role"`
	TokenRotatedAt *time.Time `json:"token_rotated_at,omitempty"` // When token was last rotated
}

type RolePermissions map[Role][]string

func (rp RolePermissions) Has(role Role, perm string) bool {
	perms := rp[role]
	return slices.Contains(perms, perm)
}

func (u *User) Can(perm string, perms RolePermissions) bool {
	return perms.Has(u.Role, perm)
}

// LoadUsers loads users and roles. Returns (store, firstRun, error).
func LoadUsers() (*UserStore, bool, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, false, err
	}

	users, err := loadUsers(usersPath)
	if os.IsNotExist(err) {
		// Only check first-run eligibility when users.json doesn't exist
		if err := validateFirstRunEligibility(); err != nil {
			return nil, false, err
		}
		return handleFirstRun(usersPath, rolesPath)
	}
	if err != nil {
		return nil, false, err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, false, fmt.Errorf("load roles.json: %w", err)
	}

	return createUserStore(users, permissions), false, nil
}

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

// handleFirstRun manages the first-run scenario when users.json doesn't exist
func handleFirstRun(usersPath, rolesPath string) (*UserStore, bool, error) {
	fmt.Println("users.json not found – creating default admin user...")
	return createDefaultUserFile(usersPath, rolesPath)
}

// validateFirstRunEligibility ensures we only run first-run setup in truly clean environments
func validateFirstRunEligibility() error {
	// Get the config directory from paths
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return err
	}
	configDir := filepath.Dir(usersPath)

	// Check for existing files that would indicate this is NOT a first run
	// Note: users.json is not checked here since this function is only called when users.json doesn't exist
	existingFiles := []string{
		rolesPath,                                // roles.json
		filepath.Join(configDir, "master.key"),   // encryption key
		filepath.Join(configDir, "secrets.json"), // secrets store
		filepath.Join(configDir, "backups"),      // backup directory
	}

	for _, file := range existingFiles {
		if _, err := os.Stat(file); err == nil {
			return fmt.Errorf("existing simple-secrets installation detected (found %s). Cannot create new admin user when installation already exists", filepath.Base(file))
		}
	}

	return nil
}

// createUserStore constructs a UserStore with the given users and permissions
func createUserStore(users []*User, permissions RolePermissions) *UserStore {
	return &UserStore{
		users:       users,
		permissions: permissions,
	}
}

func (us *UserStore) Lookup(token string) (*User, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	tokenHash := HashToken(token)
	for _, u := range us.users {
		if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(u.TokenHash)) == 1 {
			return u, nil
		}
	}
	return nil, errors.New("invalid token")
}

func (us *UserStore) Permissions() RolePermissions {
	return us.permissions
}

func loadUsers(path string) ([]*User, error) {
	var users []*User
	if err := readConfigFile(path, &users); err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("users.json is corrupted or invalid: %w. Please fix or delete the file.", err)
	}

	return validateUsersList(users)
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
	return fmt.Errorf("no admin user found in users.json. Please fix the file or recreate users.")
}

func loadRoles(path string) (RolePermissions, error) {
	var perms RolePermissions
	if err := readConfigFile(path, &perms); err != nil {
		return nil, fmt.Errorf("unmarshal roles.json: %w", err)
	}
	return perms, nil
}

func createDefaultUserFile(usersPath, rolesPath string) (*UserStore, bool, error) {
	token, user, err := generateDefaultAdmin()
	if err != nil {
		return nil, false, err
	}

	defaultRoles := createDefaultRoles()

	if err := writeConfigFiles(usersPath, rolesPath, []*User{user}, defaultRoles); err != nil {
		return nil, false, err
	}

	printFirstRunSuccess(token)

	// Load the users from the specific path that was just created
	users, err := loadUsers(usersPath)
	if err != nil {
		return nil, false, err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, false, err
	}

	store := createUserStore(users, permissions)
	return store, true, nil
}

// generateDefaultAdmin creates a new admin user with a secure token
func generateDefaultAdmin() (string, *User, error) {
	token, err := generateSecureToken()
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	now := time.Now()
	user := &User{
		Username:       "admin",
		TokenHash:      HashToken(token),
		Role:           RoleAdmin,
		TokenRotatedAt: &now,
	}

	return token, user, nil
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken() (string, error) {
	rawToken := make([]byte, 20)
	if _, err := rand.Read(rawToken); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(rawToken), nil
}

// createDefaultRoles returns the default role permissions structure
func createDefaultRoles() RolePermissions {
	return RolePermissions{
		RoleAdmin:  {"read", "write", "rotate-tokens", "manage-users", "rotate-own-token"},
		RoleReader: {"read", "rotate-own-token"},
	}
}

// writeConfigFiles writes users and roles to their respective JSON files
func writeConfigFiles(usersPath, rolesPath string, users []*User, roles RolePermissions) error {
	if err := ensureConfigDirectory(usersPath); err != nil {
		return err
	}

	if err := writeUsersFile(usersPath, users); err != nil {
		return err
	}

	return writeRolesFile(rolesPath, roles)
}

// ensureConfigDirectory creates the configuration directory if it doesn't exist
func ensureConfigDirectory(usersPath string) error {
	return os.MkdirAll(filepath.Dir(usersPath), 0700)
}

// writeConfigFileSecurely marshals and writes any config data to JSON with secure permissions
func writeConfigFileSecurely(path string, data any) error {
	encoded, err := marshalConfigData(data)
	if err != nil {
		return err
	}
	return writeFileAtomically(path, encoded)
}

// marshalConfigData converts config data to formatted JSON
func marshalConfigData(data any) ([]byte, error) {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config data: %w", err)
	}
	return encoded, nil
}

// writeFileAtomically writes data to a file with secure permissions
func writeFileAtomically(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

// readConfigFile reads and unmarshals a JSON config file
func readConfigFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// writeUsersFile marshals and writes the users list to JSON
func writeUsersFile(usersPath string, users []*User) error {
	return writeConfigFileSecurely(usersPath, users)
}

// writeRolesFile marshals and writes the roles to JSON
func writeRolesFile(rolesPath string, roles RolePermissions) error {
	return writeConfigFileSecurely(rolesPath, roles)
}

// printFirstRunSuccess displays the success message with the new admin token
func printFirstRunSuccess(token string) {
	fmt.Printf("\n✅ Created default admin user!\n")
	fmt.Printf("   Username: admin\n")
	fmt.Printf("   Token:    %s\n", token)
	fmt.Println("   (Please store this token securely. It will not be shown again.)")
}
