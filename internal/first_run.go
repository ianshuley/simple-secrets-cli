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

// IsFirstRunEligible checks if this is a fresh installation eligible for first-run setup
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
