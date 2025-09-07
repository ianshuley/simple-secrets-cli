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
	"fmt"
	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

// RBACGuardWithCmd loads users, checks first run, resolves token, and returns (user, store, error)
func RBACGuardWithCmd(needWrite bool, cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
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

// RBACGuard is the legacy function for backward compatibility
func RBACGuard(needWrite bool, tokenFlag string) (*internal.User, *internal.UserStore, error) {
	userStore, firstRun, err := internal.LoadUsers()
	if err != nil {
		return nil, nil, err
	}
	if firstRun {
		PrintFirstRunMessage()
		return nil, nil, nil
	}

	token, err := internal.ResolveToken(tokenFlag)
	if err != nil {
		return nil, nil, err
	}
	user, err := userStore.Lookup(token)
	if err != nil {
		return nil, nil, err
	}
	if needWrite && !user.Can("write", userStore.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'write'")
	}
	return user, userStore, nil
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
	userStore, firstRun, err := internal.LoadUsers()
	if err != nil {
		return nil, false, err
	}
	if firstRun {
		PrintFirstRunMessage()
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
		return fmt.Errorf("permission denied: need 'write'")
	}
	return nil
}
