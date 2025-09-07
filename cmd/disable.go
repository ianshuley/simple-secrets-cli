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
	"time"

	"github.com/spf13/cobra"
)

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:   "disable [token|secret] [username|key]",
	Short: "Disable user tokens or secrets",
	Long: `Disable different types of resources in the system:
  • token <username> - Disable a user's token (admin only)
  • secret <key>     - Mark a secret as disabled

Disabled tokens cannot be used for authentication.
Disabled secrets are hidden from normal operations but can be re-enabled.`,
	Example: `  simple-secrets disable token alice    # Disable alice's token
  simple-secrets disable secret api-key  # Disable a secret`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		switch args[0] {
		case "token":
			return disableToken(args[1])
		case "secret":
			return disableSecret(args[1])
		default:
			return fmt.Errorf("unknown disable type: %s. Use 'token' or 'secret'", args[0])
		}
	},
}

func disableToken(username string) error {
	context, err := prepareTokenDisableContext(username)
	if err != nil {
		return err
	}
	if context == nil {
		return nil // First run or access denied
	}

	if err := executeTokenDisable(context); err != nil {
		return err
	}

	printTokenDisableSuccess(username)
	return nil
}

func disableSecret(key string) error {
	context, err := prepareSecretDisableContext(key)
	if err != nil {
		return err
	}
	if context == nil {
		return nil // First run or access denied
	}

	if err := executeSecretDisable(context); err != nil {
		return err
	}

	printSecretDisableSuccess(key)
	return nil
}

// TokenDisableContext holds data needed for token disabling
type TokenDisableContext struct {
	RequestingUser *internal.User
	TargetUser     *internal.User
	TargetUsername string
	TargetIndex    int
	UsersPath      string
	Users          []*internal.User
}

// SecretDisableContext holds data needed for secret disabling
type SecretDisableContext struct {
	RequestingUser *internal.User
	SecretKey      string
	Store          *internal.SecretsStore
}

// prepareTokenDisableContext validates access and prepares context for token disabling
func prepareTokenDisableContext(username string) (*TokenDisableContext, error) {
	user, _, err := validateTokenDisableAccess()
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return nil, err
	}

	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return nil, err
	}

	targetIndex, err := findUserIndex(users, username)
	if err != nil {
		return nil, err
	}

	return &TokenDisableContext{
		RequestingUser: user,
		TargetUser:     users[targetIndex],
		TargetUsername: username,
		TargetIndex:    targetIndex,
		UsersPath:      usersPath,
		Users:          users,
	}, nil
}

// prepareSecretDisableContext validates access and prepares context for secret disabling
func prepareSecretDisableContext(key string) (*SecretDisableContext, error) {
	user, _, err := validateSecretDisableAccess()
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return nil, err
	}

	// Check if secret exists
	if _, err := store.Get(key); err != nil {
		return nil, fmt.Errorf("secret '%s' not found", key)
	}

	return &SecretDisableContext{
		RequestingUser: user,
		SecretKey:      key,
		Store:          store,
	}, nil
}

// validateTokenDisableAccess checks RBAC permissions for token disabling
func validateTokenDisableAccess() (*internal.User, *internal.UserStore, error) {
	user, store, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil
	}

	if !user.Can("rotate-tokens", store.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'rotate-tokens' permission to disable tokens")
	}

	return user, store, nil
}

// validateSecretDisableAccess checks RBAC permissions for secret disabling
func validateSecretDisableAccess() (*internal.User, *internal.UserStore, error) {
	user, store, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil
	}

	if !user.Can("write", store.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'write' permission to disable secrets")
	}

	return user, store, nil
}

// executeTokenDisable marks a token as disabled
func executeTokenDisable(context *TokenDisableContext) error {
	now := time.Now()
	context.TargetUser.TokenHash = ""        // Clear the token hash to disable
	context.TargetUser.TokenRotatedAt = &now // Update timestamp

	return saveUsersList(context.UsersPath, context.Users)
}

// executeSecretDisable marks a secret as disabled
func executeSecretDisable(context *SecretDisableContext) error {
	return context.Store.DisableSecret(context.SecretKey)
}

// printTokenDisableSuccess displays success message for token disabling
func printTokenDisableSuccess(username string) {
	fmt.Printf("✅ Token disabled for user '%s'\n", username)
	fmt.Println("• The user can no longer authenticate with their current token")
	fmt.Println("• Use 'rotate token' to generate a new token for this user")
}

// printSecretDisableSuccess displays success message for secret disabling
func printSecretDisableSuccess(key string) {
	fmt.Printf("✅ Secret '%s' has been disabled\n", key)
	fmt.Println("• The secret is hidden from normal operations")
	fmt.Println("• Use 'enable secret' to re-enable this secret")
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
