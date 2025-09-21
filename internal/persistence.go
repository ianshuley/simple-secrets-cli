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
)

// Configuration file management and persistence operations

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

// File reading operations

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

// File writing operations

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

// Data validation operations

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
