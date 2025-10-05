/*package config

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

// Package config provides configuration management for simple-secrets
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetSimpleSecretsPath returns the path to the .simple-secrets directory
func GetSimpleSecretsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".simple-secrets"), nil
}

// DefaultConfigPath returns the default path for a config file
func DefaultConfigPath(filename string) (string, error) {
	// Check for test override first
	if testDir := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR"); testDir != "" {
		return filepath.Join(testDir, filename), nil
	}

	configDir, err := GetSimpleSecretsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, filename), nil
}

// ResolveConfigPaths determines the file paths for users.json and roles.json
func ResolveConfigPaths() (string, string, error) {
	usersPath, err := DefaultConfigPath("users.json")
	if err != nil {
		return "", "", err
	}

	rolesPath, err := DefaultConfigPath("roles.json")
	if err != nil {
		return "", "", err
	}

	return usersPath, rolesPath, nil
}

// EnsureConfigDirectory creates the configuration directory if it doesn't exist
func EnsureConfigDirectory(configPath string) error {
	return os.MkdirAll(filepath.Dir(configPath), 0700)
}
