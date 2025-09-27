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
package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

// RBACGuard loads users, checks first run, resolves token, and returns (user, store, error)
func RBACGuard(needWrite bool, cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
	user, store, err := authenticateUser(cmd)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil // First run or empty token
	}

	if err := authorizeAccess(user, store, needWrite); err != nil {
		return nil, nil, err
	}

	return user, store, nil
}

// AuthenticateWithToken handles authentication for commands with custom token parsing (like put)
func AuthenticateWithToken(needWrite bool, token string) (*internal.User, *internal.UserStore, error) {
	userStore, firstRun, firstRunToken, err := loadUserStoreForAuth()
	if err != nil {
		return nil, nil, err
	}

	if firstRun {
		PrintFirstRunMessage()
		PrintTokenAtEnd(firstRunToken)
		return nil, nil, nil
	}

	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return nil, nil, err
	}

	user, err := userStore.Lookup(resolvedToken)
	if err != nil {
		return nil, nil, err
	}

	if needWrite && !user.Can("write", userStore.Permissions()) {
		return nil, nil, NewWritePermissionError()
	}

	return user, userStore, nil
}

// loadUserStoreForAuth handles the complex user store loading logic with fallback
func loadUserStoreForAuth() (*internal.UserStore, bool, string, error) {
	userStore, firstRun, firstRunToken, err := internal.LoadUsers()
	if err != nil {
		authStore, authErr := internal.LoadUsersForAuth()
		if authErr != nil {
			return nil, false, "", authErr
		}
		return authStore, false, "", nil
	}

	return userStore, firstRun, firstRunToken, nil
}

// authenticateUser handles user authentication and first-run detection
func authenticateUser(cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
	userStore, firstRun, err := loadUsersWithFirstRunCheck()
	if err != nil {
		return nil, nil, err
	}
	if firstRun {
		return nil, nil, nil // First run message already printed
	}

	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, nil, err
	}

	user, err := userStore.Lookup(token)
	if err != nil {
		return nil, nil, err
	}

	return user, userStore, nil
}

// loadUsersWithFirstRunCheck loads users and handles first-run scenarios
func loadUsersWithFirstRunCheck() (*internal.UserStore, bool, error) {
	userStore, firstRun, token, err := internal.LoadUsers()
	if err != nil {
		// If LoadUsers fails, try the auth-specific path
		authStore, authErr := internal.LoadUsersForAuth()
		if authErr != nil {
			return nil, false, authErr
		}
		return authStore, false, nil
	}
	if firstRun {
		PrintFirstRunMessage()
		PrintTokenAtEnd(token)
		return nil, true, nil
	}
	return userStore, false, nil
}

// resolveTokenFromCommand extracts and validates the token from command context
func resolveTokenFromCommand(cmd *cobra.Command) (string, error) {
	tokenFlag := TokenFlag

	// Check if token flag was explicitly set to empty string
	if flag := cmd.Flag("token"); flag != nil && flag.Changed && tokenFlag == "" {
		return "", fmt.Errorf("authentication required: token cannot be empty")
	}

	return internal.ResolveToken(tokenFlag)
}

// authorizeAccess checks if the user has the required permissions
func authorizeAccess(user *internal.User, store *internal.UserStore, needWrite bool) error {
	if needWrite && !user.Can("write", store.Permissions()) {
		return NewWritePermissionError()
	}
	return nil
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken() (string, error) {
	randToken := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, randToken); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randToken), nil
}

// ErrAuthenticationRequired returns a standard authentication required error
var ErrAuthenticationRequired = fmt.Errorf("authentication required: token cannot be empty")

// Common error constructors to reduce duplication
func NewPermissionDeniedError(permission string) error {
	return fmt.Errorf("permission denied: need '%s' permission", permission)
}

func NewWritePermissionError() error {
	return fmt.Errorf("permission denied: need 'write'")
}

func NewSecretNotFoundError() error {
	return fmt.Errorf("secret not found")
}

func NewDisabledSecretNotFoundError() error {
	return fmt.Errorf("disabled secret not found")
}

func NewUserNotFoundError(username string) error {
	return fmt.Errorf("user '%s' not found", username)
}

func NewUnknownTypeError(typeName, value, validOptions string) error {
	return fmt.Errorf("unknown %s type: %s. Use %s", typeName, value, validOptions)
}
