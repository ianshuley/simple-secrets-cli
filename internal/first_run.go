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
	"path/filepath"
	"slices"
	"time"
)

// IsFirstRun checks if this is a fresh installation that needs setup
func IsFirstRun() (bool, error) {
	usersPath, err := DefaultUserConfigPath("users.json")
	if err != nil {
		return false, err
	}

	// If users.json exists, not first run
	_, err = os.Stat(usersPath)
	return os.IsNotExist(err), nil
}

// DoFirstRunSetup creates the initial admin user and returns the user store
func DoFirstRunSetup() (*UserStore, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, err
	}

	fmt.Println("Setting up simple-secrets for first use...")
	store, created, err := createDefaultUserFile(usersPath, rolesPath)
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, fmt.Errorf("setup failed: could not create default user")
	}
	return store, nil
}

// DoFirstRunSetupWithToken creates the initial admin user and returns both store and token
func DoFirstRunSetupWithToken() (*UserStore, string, error) {
	usersPath, rolesPath, err := resolveConfigPaths()
	if err != nil {
		return nil, "", err
	}

	fmt.Println("Setting up simple-secrets for first use...")
	return createDefaultUserFileWithToken(usersPath, rolesPath)
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
	return slices.Contains(declineResponses, response)
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
			return fmt.Errorf("existing simple-secrets installation detected (found %s). Cannot create new admin user when installation already exists. If this is unexpected, restore it from backup or manually investigate", filepath.Base(file))
		}
	}

	return nil
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

// createDefaultUserFile creates the default admin user and returns the store
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

	// Create default config.json with examples
	if err := createDefaultConfigFile(); err != nil {
		// Don't fail setup if config.json creation fails, just warn
		fmt.Printf("Warning: failed to create default config.json: %v\n", err)
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

// createDefaultConfigFile creates a minimal config.json with sensible defaults
// Documentation and examples are available via 'simple-secrets help config'
func createDefaultConfigFile() error {
	configContent := `{
  "rotation_backup_count": 1
}`

	configPath, err := DefaultUserConfigPath("config.json")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(configContent), 0600)
}
