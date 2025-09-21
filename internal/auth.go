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
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

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

	return GetSimpleSecretsFilePath(filename)
}

// LoadUsersList loads the user list from users.json (for CLI user creation).
func LoadUsersList(path string) ([]*User, error) {
	return loadUsers(path)
}

// ResolveToken returns the token from CLI flag, env, or config file (in that order).
func ResolveToken(cliFlag string) (string, error) {
	if cliFlag != "" {
		if strings.TrimSpace(cliFlag) == "" {
			return "", errors.New("authentication required: token cannot be empty")
		}
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
			return "", errors.New(`authentication required: no token found

Use one of these methods:
    --token <your-token> (as a flag)
    SIMPLE_SECRETS_TOKEN=<your-token> (as environment variable)
    ~/.simple-secrets/config.json with { "token": "<your-token>" }`)
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

// TokenGenerator is a function type for generating secure tokens
type TokenGenerator func() (string, error)

// DefaultTokenGenerator holds the token generation function (set by cmd package)
var DefaultTokenGenerator TokenGenerator

// generateSecureToken calls the registered token generator or uses fallback
func generateSecureToken() (string, error) {
	if DefaultTokenGenerator != nil {
		return DefaultTokenGenerator()
	}
	// Fallback for tests and direct internal package usage
	return generateSecureTokenFallback()
}

// generateSecureTokenFallback is the original implementation for fallback use
func generateSecureTokenFallback() (string, error) {
	const tokenLengthBytes = 20 // 20 bytes = 160 bits of entropy
	tokenBytes := make([]byte, tokenLengthBytes)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

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

// DisableToken disables the user's token by clearing the token hash and updating the timestamp
func (u *User) DisableToken() {
	u.TokenHash = ""
	now := time.Now()
	u.TokenRotatedAt = &now
}

// IsFirstRunEligible checks if this is a fresh installation eligible for first-run setup
// This is a read-only check that doesn't trigger any setup or create files
func IsFirstRunEligible() (bool, error) {
	usersPath, err := DefaultUserConfigPath("users.json")
	if err != nil {
		return false, err
	}

	// Check if users.json exists
	if _, err := os.Stat(usersPath); !os.IsNotExist(err) {
		return false, nil // users.json exists = not first run
	}

	// Users.json doesn't exist, check if this is a clean environment
	if err := validateFirstRunEligibility(); err != nil {
		return false, err // Broken state - not eligible for first run
	}

	return true, nil // Clean environment - eligible for first run
}

// PerformFirstRunSetup executes the first-run setup process
// Should only be called after confirming IsFirstRunEligible() returns true
func PerformFirstRunSetup() (*UserStore, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, err
	}

	// Verify we're still eligible (double-check in case of race conditions)
	if err := validateFirstRunEligibility(); err != nil {
		return nil, err
	}

	// Verify users.json still doesn't exist
	if _, err := os.Stat(usersPath); !os.IsNotExist(err) {
		return nil, fmt.Errorf("users.json was created by another process")
	}

	fmt.Println("users.json not found – creating default admin user...")
	store, firstRun, err := createDefaultUserFile(usersPath, rolesPath)
	if err != nil {
		return nil, err
	}
	if !firstRun {
		return nil, fmt.Errorf("unexpected: first run setup did not complete properly")
	}
	return store, nil
}

// PerformFirstRunSetupWithToken executes the first-run setup process and returns the admin token
// Should only be called after confirming IsFirstRunEligible() returns true
func PerformFirstRunSetupWithToken() (*UserStore, string, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, "", err
	}

	// Verify we're still eligible (double-check in case of race conditions)
	if err := validateFirstRunEligibility(); err != nil {
		return nil, "", err
	}

	// Verify users.json still doesn't exist
	if _, err := os.Stat(usersPath); !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("users.json was created by another process")
	}

	fmt.Println("users.json not found – creating default admin user...")
	store, token, err := createDefaultUserFileWithToken(usersPath, rolesPath)
	if err != nil {
		return nil, "", err
	}
	return store, token, nil
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

// handleFirstRunWithToken manages the first-run scenario and returns the generated token
func handleFirstRunWithToken(usersPath, rolesPath string) (*UserStore, string, error) {
	const (
		firstRunPrompt         = "First run detected - creating default admin user..."
		passwordManagerWarning = "⚠️  This will generate an authentication token. Have your password manager ready."
		proceedPrompt          = "\nProceed? [Y/n]"
		cancellationMessage    = "Setup cancelled. Run any command again when ready."
	)

	fmt.Println(firstRunPrompt)
	fmt.Println(passwordManagerWarning)
	fmt.Println(proceedPrompt)

	var response string
	fmt.Scanln(&response)

	if UserDeclinedSetup(response) {
		fmt.Println(cancellationMessage)
		return nil, "", fmt.Errorf("setup cancelled by user")
	}

	return createDefaultUserFileWithToken(usersPath, rolesPath)
}

// UserDeclinedSetup checks if user declined the setup prompt
// Exported for use by cmd package to avoid duplication
func UserDeclinedSetup(response string) bool {
	declineResponses := []string{"n", "N", "no", "NO"}
	for _, decline := range declineResponses {
		if response == decline {
			return true
		}
	}
	return false
} // validateFirstRunEligibility ensures we only run first-run setup in truly clean environments
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
			return fmt.Errorf("existing simple-secrets installation detected (found %s). Cannot create new admin user when installation already exists. If this is unexpected, restore it from backup or manually investigate", filepath.Base(file))
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

	us.mu.RLock()
	defer us.mu.RUnlock()
	for _, u := range us.users {
		if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(u.TokenHash)) == 1 {
			return u, nil
		}
	}
	return nil, errors.New("invalid token")
}

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

func loadRoles(path string) (RolePermissions, error) {
	var perms RolePermissions
	if err := readConfigFile(path, &perms); err != nil {
		return nil, fmt.Errorf("unmarshal roles.json: %w", err)
	}
	return perms, nil
}

func createDefaultUserFile(usersPath, rolesPath string) (*UserStore, bool, error) {
	_, user, err := generateDefaultAdmin()
	if err != nil {
		return nil, false, err
	}

	defaultRoles := createDefaultRoles()

	if err := writeConfigFiles(usersPath, rolesPath, []*User{user}, defaultRoles); err != nil {
		return nil, false, err
	}

	// User created successfully (no immediate printing needed)

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

// createDefaultUserFileWithToken creates the default admin user and returns the token without printing
func createDefaultUserFileWithToken(usersPath, rolesPath string) (*UserStore, string, error) {
	token, user, err := generateDefaultAdmin()
	if err != nil {
		return nil, "", err
	}

	defaultRoles := createDefaultRoles()

	if err := writeConfigFiles(usersPath, rolesPath, []*User{user}, defaultRoles); err != nil {
		return nil, "", err
	}

	// Don't print the token here - return it instead
	fmt.Printf("\n✅ Created default admin user!\n")
	fmt.Printf("   Username: admin\n")

	// Load the users from the specific path that was just created
	users, err := loadUsers(usersPath)
	if err != nil {
		return nil, "", err
	}

	permissions, err := loadRoles(rolesPath)
	if err != nil {
		return nil, "", err
	}

	store := createUserStore(users, permissions)
	return store, token, nil
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

	if err := writeConfigFileSecurely(usersPath, users); err != nil {
		return err
	}

	return writeConfigFileSecurely(rolesPath, roles)
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
	return AtomicWriteFile(path, encoded, 0600)
}

// marshalConfigData converts config data to formatted JSON
func marshalConfigData(data any) ([]byte, error) {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config data: %w", err)
	}
	return encoded, nil
}

// readConfigFile reads and unmarshals a JSON config file
func readConfigFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
