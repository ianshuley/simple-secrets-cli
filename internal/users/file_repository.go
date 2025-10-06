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

package users

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"simple-secrets/pkg/errors"
	"simple-secrets/pkg/users"
)

const (
	usersSecureFilePermissions      = 0600 // Owner read/write only
	usersSecureDirectoryPermissions = 0700 // Owner read/write/execute only
)

// FileRepository implements the users.Repository interface using the filesystem
type FileRepository struct {
	dataDir   string
	usersFile string
}

// NewFileRepository creates a new file-based users repository
func NewFileRepository(dataDir string) users.Repository {
	return &FileRepository{
		dataDir:   dataDir,
		usersFile: filepath.Join(dataDir, "users.json"),
	}
}

// usersData represents the structure stored in users.json
type usersData struct {
	Users   map[string]*users.User `json:"users"` // username -> user
	Version string                 `json:"version"`
}

// Store persists a user to the filesystem
func (r *FileRepository) Store(ctx context.Context, user *users.User) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := r.ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := r.loadUsersData()
	if err != nil {
		return err
	}

	// Create a copy to avoid modifying the original
	userCopy := *user
	data.Users[user.Username] = &userCopy
	return r.saveUsersData(data)
}

// Retrieve gets a user by username
func (r *FileRepository) Retrieve(ctx context.Context, username string) (*users.User, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return nil, err
	}

	user, exists := data.Users[username]
	if !exists {
		return nil, errors.NewNotFoundError("user", username)
	}

	return user, nil
}

// RetrieveByToken gets a user by token hash
func (r *FileRepository) RetrieveByToken(ctx context.Context, tokenHash string) (*users.User, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return nil, err
	}

	// Search through all users for the token
	for _, user := range data.Users {
		for _, token := range user.Tokens {
			if token.Hash == tokenHash {
				return user, nil
			}
		}
	}

	return nil, errors.NewNotFoundError("user with token", tokenHash)
}

// Delete removes a user permanently
func (r *FileRepository) Delete(ctx context.Context, username string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return err
	}

	if _, exists := data.Users[username]; !exists {
		return errors.NewNotFoundError("user", username)
	}

	delete(data.Users, username)
	return r.saveUsersData(data)
}

// List returns all users
func (r *FileRepository) List(ctx context.Context) ([]*users.User, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return nil, err
	}

	result := make([]*users.User, 0, len(data.Users))
	for _, user := range data.Users {
		result = append(result, user)
	}

	return result, nil
}

// Enable marks a user as enabled
func (r *FileRepository) Enable(ctx context.Context, username string) error {
	return r.updateUserStatus(ctx, username, false)
}

// Disable marks a user as disabled
func (r *FileRepository) Disable(ctx context.Context, username string) error {
	return r.updateUserStatus(ctx, username, true)
}

// Exists checks if a user exists
func (r *FileRepository) Exists(ctx context.Context, username string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return false, err
	}

	_, exists := data.Users[username]
	return exists, nil
}

// updateUserStatus updates the disabled status of a user
func (r *FileRepository) updateUserStatus(ctx context.Context, username string, disabled bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := r.loadUsersData()
	if err != nil {
		return err
	}

	user, exists := data.Users[username]
	if !exists {
		return errors.NewNotFoundError("user", username)
	}

	user.Disabled = disabled

	return r.saveUsersData(data)
}

// loadUsersData loads users from the filesystem
func (r *FileRepository) loadUsersData() (*usersData, error) {
	if _, err := os.Stat(r.usersFile); os.IsNotExist(err) {
		return &usersData{
			Users:   make(map[string]*users.User),
			Version: "1.0",
		}, nil
	}

	data, err := os.ReadFile(r.usersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	// Try to unmarshal as new format first
	var result usersData
	if err := json.Unmarshal(data, &result); err == nil && result.Users != nil {
		return &result, nil
	}

	// If that fails, try legacy format (array of users)
	var legacyUsers []struct {
		Username       string `json:"username"`
		TokenHash      string `json:"token_hash"`
		Role           string `json:"role"`
		TokenRotatedAt string `json:"token_rotated_at"`
		Disabled       bool   `json:"disabled,omitempty"`
	}

	if err := json.Unmarshal(data, &legacyUsers); err != nil {
		return nil, fmt.Errorf("failed to parse users file in any known format: %w", err)
	}

	// Convert legacy format to new format
	result = usersData{
		Users:   make(map[string]*users.User),
		Version: "1.0",
	}

	for _, legacyUser := range legacyUsers {
		// Parse the timestamp
		var tokenRotatedAt *time.Time
		if legacyUser.TokenRotatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, legacyUser.TokenRotatedAt); err == nil {
				tokenRotatedAt = &parsed
			}
		}

		// Create a token for the legacy token hash
		legacyToken := &users.Token{
			ID:   "legacy-token", // Default ID for legacy token
			Name: "Admin Token",  // Default name for legacy token
			Hash: legacyUser.TokenHash,
		}

		user := &users.User{
			ID:             legacyUser.Username, // Use username as ID for legacy compatibility
			Username:       legacyUser.Username,
			Tokens:         []*users.Token{legacyToken},
			Role:           legacyUser.Role,
			CreatedAt:      time.Now(), // Legacy files don't have created_at
			TokenRotatedAt: tokenRotatedAt,
			Disabled:       legacyUser.Disabled,
		}
		result.Users[legacyUser.Username] = user
	}

	return &result, nil
}

// saveUsersData saves users to the filesystem atomically
func (r *FileRepository) saveUsersData(data *usersData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users data: %w", err)
	}

	return r.atomicWriteFile(r.usersFile, jsonData, usersSecureFilePermissions)
}

// ensureDataDir creates the data directory if it doesn't exist
func (r *FileRepository) ensureDataDir() error {
	return os.MkdirAll(r.dataDir, usersSecureDirectoryPermissions)
}

// atomicWriteFile writes data to a file atomically using a temporary file and rename
func (r *FileRepository) atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
