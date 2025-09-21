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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Role represents a user role with specific permissions
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleReader Role = "reader"
)

// TokenGenerator is a function type for generating secure tokens
type TokenGenerator func() (string, error)

// DefaultTokenGenerator holds the token generation function (set by cmd package)
var DefaultTokenGenerator TokenGenerator

// User represents a user in the system with authentication and authorization info
type User struct {
	Username       string     `json:"username"`
	TokenHash      string     `json:"token_hash"` // SHA-256 hash, base64-encoded
	Role           Role       `json:"role"`
	TokenRotatedAt *time.Time `json:"token_rotated_at,omitempty"` // When token was last rotated
}

// RolePermissions maps roles to their allowed permissions
type RolePermissions map[Role][]string

// Has checks if a role has a specific permission
func (rp RolePermissions) Has(role Role, perm string) bool {
	perms := rp[role]
	return slices.Contains(perms, perm)
}

// Can checks if the user has a specific permission
func (u *User) Can(perm string, perms RolePermissions) bool {
	return perms.Has(u.Role, perm)
}

// DisableToken disables the user's token by clearing the token hash and updating the timestamp
func (u *User) DisableToken() {
	u.TokenHash = ""
	now := time.Now()
	u.TokenRotatedAt = &now
}

// HashToken creates a SHA-256 hash of a token for secure storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

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

// DefaultUserConfigPath exports defaultUserConfigPath for CLI use.
func DefaultUserConfigPath(filename string) (string, error) {
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

Set your token via:
    --token <your-token> (as a flag)
    SIMPLE_SECRETS_TOKEN=<your-token> (as environment variable)
    ~/.simple-secrets/config.json with {"token": "<your-token>"}

For more config options, run: simple-secrets help config`)
		}
		return "", fmt.Errorf("read config.json: %w", err)
	}

	var config struct {
		Token               string `json:"token"`
		RotationBackupCount *int   `json:"rotation_backup_count,omitempty"` // Number of rotation backups to keep (default: 1)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("unmarshal config.json: %w", err)
	}
	if config.Token == "" {
		return "", errors.New(`authentication required: no token found

Set your token via:
    --token <your-token> (as a flag)
    SIMPLE_SECRETS_TOKEN=<your-token> (as environment variable)
    ~/.simple-secrets/config.json with {"token": "<your-token>"}

For more config options, run: simple-secrets help config`)
	}

	return config.Token, nil
}

// ====================================
// Configuration Path Utilities
// ====================================
// These functions handle configuration file paths and directory management
// for the simple-secrets application user configuration.

// GetSimpleSecretsPath returns the path to the .simple-secrets directory
func GetSimpleSecretsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ".simple-secrets"), nil
}
