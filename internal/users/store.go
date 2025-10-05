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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"simple-secrets/pkg/crypto"
	"simple-secrets/pkg/errors"
	"simple-secrets/pkg/users"
)

// StoreImpl implements the users.Store interface providing business logic
type StoreImpl struct {
	repo users.Repository
}

// NewStore creates a new users store with the provided repository
func NewStore(repo users.Repository) users.Store {
	return &StoreImpl{
		repo: repo,
	}
}

// Create creates a new user with the given username and role, returns user and initial token
func (s *StoreImpl) Create(ctx context.Context, username, role string) (*users.User, string, error) {
	if err := s.validateUsername(username); err != nil {
		return nil, "", err
	}

	if err := s.validateRole(role); err != nil {
		return nil, "", err
	}

	// Check if user already exists
	exists, err := s.repo.Exists(ctx, username)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check user existence: %w", err)
	}

	if exists {
		return nil, "", errors.NewAlreadyExistsError("user", username)
	}

	// Create user
	user := users.NewUser(username, role)

	// Generate initial token
	tokenValue, tokenHash, err := s.generateToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate initial token: %w", err)
	}

	// Create initial token with default name
	token := users.NewToken("default", tokenHash)
	user.AddToken(token)

	if err := s.repo.Store(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to store user: %w", err)
	}

	return user, tokenValue, nil
}

// GetByUsername retrieves a user by username
func (s *StoreImpl) GetByUsername(ctx context.Context, username string) (*users.User, error) {
	if err := s.validateUsername(username); err != nil {
		return nil, err
	}

	return s.repo.Retrieve(ctx, username)
}

// GetByToken retrieves a user by token value (not hash)
func (s *StoreImpl) GetByToken(ctx context.Context, token string) (*users.User, error) {
	if token == "" {
		return nil, errors.NewInvalidInputError("token", "cannot be empty")
	}

	tokenHash := s.hashToken(token)
	return s.repo.RetrieveByToken(ctx, tokenHash)
}

// List returns all users
func (s *StoreImpl) List(ctx context.Context) ([]*users.User, error) {
	return s.repo.List(ctx)
}

// Update updates user information
func (s *StoreImpl) Update(ctx context.Context, user *users.User) error {
	if user == nil {
		return errors.NewInvalidInputError("user", "cannot be nil")
	}

	if err := s.validateUsername(user.Username); err != nil {
		return err
	}

	if err := s.validateRole(user.Role); err != nil {
		return err
	}

	return s.repo.Store(ctx, user)
}

// Delete removes a user permanently
func (s *StoreImpl) Delete(ctx context.Context, username string) error {
	if err := s.validateUsername(username); err != nil {
		return err
	}

	return s.repo.Delete(ctx, username)
}

// Enable makes a disabled user active
func (s *StoreImpl) Enable(ctx context.Context, username string) error {
	if err := s.validateUsername(username); err != nil {
		return err
	}

	return s.repo.Enable(ctx, username)
}

// Disable makes a user inactive without deleting
func (s *StoreImpl) Disable(ctx context.Context, username string) error {
	if err := s.validateUsername(username); err != nil {
		return err
	}

	return s.repo.Disable(ctx, username)
}

// RotateToken generates a new token for a user (replaces primary token for backward compatibility)
func (s *StoreImpl) RotateToken(ctx context.Context, username string) (string, error) {
	if err := s.validateUsername(username); err != nil {
		return "", err
	}

	user, err := s.repo.Retrieve(ctx, username)
	if err != nil {
		return "", err
	}

	if user.Disabled {
		return "", errors.NewInvalidInputError("user", "cannot rotate token for disabled user")
	}

	// Generate new token
	tokenValue, tokenHash, err := s.generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Find primary token and replace it, or create if none exists
	primaryToken := user.GetPrimaryToken()
	if primaryToken != nil {
		primaryToken.Hash = tokenHash
		primaryToken.CreatedAt = time.Now()
		primaryToken.LastUsedAt = nil
	} else {
		// Create new default token
		token := users.NewToken("default", tokenHash)
		user.AddToken(token)
	}

	if err := s.repo.Store(ctx, user); err != nil {
		return "", fmt.Errorf("failed to store updated user: %w", err)
	}

	return tokenValue, nil
}

// AddToken adds a new named token to a user (multi-token support)
func (s *StoreImpl) AddToken(ctx context.Context, username, tokenName string) (*users.Token, string, error) {
	if err := s.validateUsername(username); err != nil {
		return nil, "", err
	}

	if err := s.validateTokenName(tokenName); err != nil {
		return nil, "", err
	}

	user, err := s.repo.Retrieve(ctx, username)
	if err != nil {
		return nil, "", err
	}

	if user.Disabled {
		return nil, "", errors.NewInvalidInputError("user", "cannot create token for disabled user")
	}

	// Check if token name already exists for this user
	for _, token := range user.Tokens {
		if token.Name == tokenName {
			return nil, "", errors.NewAlreadyExistsError("token", tokenName)
		}
	}

	// Generate secure random token
	tokenValue, tokenHash, err := s.generateToken()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := users.NewToken(tokenName, tokenHash)
	user.AddToken(token)

	if err := s.repo.Store(ctx, user); err != nil {
		return nil, "", fmt.Errorf("failed to store updated user: %w", err)
	}

	return token, tokenValue, nil
}

// RevokeToken revokes a specific token by ID
func (s *StoreImpl) RevokeToken(ctx context.Context, username, tokenID string) error {
	if err := s.validateUsername(username); err != nil {
		return err
	}

	if tokenID == "" {
		return errors.NewInvalidInputError("tokenID", "cannot be empty")
	}

	user, err := s.repo.Retrieve(ctx, username)
	if err != nil {
		return err
	}

	if !user.RemoveToken(tokenID) {
		return errors.NewNotFoundError("token", tokenID)
	}

	return s.repo.Store(ctx, user)
}

// ListTokens returns all tokens for a user
func (s *StoreImpl) ListTokens(ctx context.Context, username string) ([]*users.Token, error) {
	if err := s.validateUsername(username); err != nil {
		return nil, err
	}

	user, err := s.repo.Retrieve(ctx, username)
	if err != nil {
		return nil, err
	}

	return user.Tokens, nil
}

// generateToken creates a secure random token and its hash
func (s *StoreImpl) generateToken() (string, string, error) {
	tokenBytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate token: %w", err)
	}

	tokenValue := hex.EncodeToString(tokenBytes)
	tokenHash := s.hashToken(tokenValue)

	return tokenValue, tokenHash, nil
}

// hashToken creates a secure hash of a token for storage
func (s *StoreImpl) hashToken(token string) string {
	return crypto.HashToken(token)
}

// validateUsername validates username format and constraints
func (s *StoreImpl) validateUsername(username string) error {
	if username == "" {
		return errors.NewInvalidInputError("username", "cannot be empty")
	}

	if len(username) > 64 {
		return errors.NewInvalidInputError("username", "too long (max 64 characters)")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, r := range username {
		if !isValidUsernameChar(r) {
			return errors.NewInvalidUsernameError(username, "contains invalid characters (allowed: a-z, A-Z, 0-9, -, _)")
		}
	}

	return nil
}

// validateRole validates role format (basic validation - auth domain handles full validation)
func (s *StoreImpl) validateRole(role string) error {
	if role == "" {
		return errors.NewInvalidInputError("role", "cannot be empty")
	}

	if len(role) > 32 {
		return errors.NewInvalidInputError("role", "too long (max 32 characters)")
	}

	// Basic format validation - auth domain will handle business rules
	role = strings.TrimSpace(role)
	if role == "" {
		return errors.NewInvalidInputError("role", "cannot be empty or whitespace only")
	}

	return nil
}

// validateTokenName validates token name format and constraints
func (s *StoreImpl) validateTokenName(tokenName string) error {
	if tokenName == "" {
		return errors.NewInvalidInputError("tokenName", "cannot be empty")
	}

	if len(tokenName) > 64 {
		return errors.NewInvalidInputError("tokenName", "too long (max 64 characters)")
	}

	// Check for valid characters (alphanumeric, dash, underscore, space)
	for _, r := range tokenName {
		if !isValidTokenNameChar(r) {
			return errors.NewInvalidInputError("tokenName", "contains invalid characters (allowed: a-z, A-Z, 0-9, -, _, space)")
		}
	}

	return nil
}

// isValidUsernameChar checks if a character is valid for usernames
func isValidUsernameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_'
}

// isValidTokenNameChar checks if a character is valid for token names
func isValidTokenNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_' || r == ' '
}
