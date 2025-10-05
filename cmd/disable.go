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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"

	"github.com/spf13/cobra"
)

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:   "disable [user|token|secret] [username|token|key]",
	Short: "Disable user tokens or secrets",
	Long: `Disable different types of resources in the system:
  • user <username>     - Disable a user's token by username (admin only)
  • token <token-value> - Disable a specific token by its value (admin only)
  • secret <key>        - Mark a secret as disabled

Disabled tokens cannot be used for authentication.
Disabled secrets are hidden from normal operations but can be re-enabled.`,
	Example: `  simple-secrets disable user alice            # Disable alice's token by username
  simple-secrets disable token abc123def456    # Disable specific token by value
  simple-secrets disable secret api-key        # Disable a secret`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "user":
			return disableUser(cmd, args[1])
		case "token":
			return disableTokenByValue(cmd, args[1])
		case "secret":
			return disableSecret(cmd, args[1])
		default:
			return NewUnknownTypeError("disable", args[0], "'user', 'token', or 'secret'")
		}
	},
}

func disableUser(cmd *cobra.Command, username string) error {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return fmt.Errorf("failed to get platform config: %w", err)
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize platform: %w", err)
	}

	// Resolve token for authentication
	authToken, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, authToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check manage-users permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionManageUsers)
	if err != nil {
		return fmt.Errorf("manage-users access denied: %w", err)
	}

	// Show detailed help for the disable user operation
	fmt.Println("⚠️  User Token Disable Operation")
	fmt.Println("• This will disable the specified user's authentication token immediately")
	fmt.Println("• The user will no longer be able to authenticate until a new token is generated")
	fmt.Println("• Use 'enable user' to generate a new token for this user")
	fmt.Printf("• Target user: '%s'\n", username)
	fmt.Println()

	// Get confirmation for user disable operation
	if !confirmTokenDisable() {
		fmt.Println("❌ User disable operation cancelled.")
		return nil
	}

	// Disable user using platform services
	err = app.Users.Disable(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	fmt.Printf("✅ Token disabled for user '%s'\n", username)
	fmt.Println("• The user can no longer authenticate with their current token")
	fmt.Println("• Use 'enable user' to generate a new token for this user")
	return nil
}

func disableTokenByValue(cmd *cobra.Command, tokenValue string) error {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize platform: %w", err)
	}

	// Resolve token for authentication
	authToken, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, authToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check manage-users permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionManageUsers)
	if err != nil {
		return fmt.Errorf("manage-users access denied: %w", err)
	}

	// Show detailed help for the disable token operation
	fmt.Println("⚠️  Token Disable Operation")
	fmt.Println("• This will disable the specified token immediately")
	fmt.Println("• The owner will no longer be able to authenticate with this token")
	fmt.Println("• Use 'enable user' to generate a new token for the affected user")
	fmt.Printf("• Target token: %s...\n", tokenValue[:min(8, len(tokenValue))])
	fmt.Println()

	// Get confirmation for token disable operation
	if !confirmTokenDisable() {
		fmt.Println("❌ Token disable operation cancelled.")
		return nil
	}

	// Find the user who owns this token
	targetUser, err := app.Users.GetByToken(ctx, tokenValue)
	if err != nil {
		return fmt.Errorf("failed to find user for token: %w", err)
	}

	// Disable the user (which effectively disables all their tokens)
	err = app.Users.Disable(ctx, targetUser.Username)
	if err != nil {
		return fmt.Errorf("failed to disable user: %w", err)
	}

	fmt.Printf("✅ Token disabled for user '%s'\n", targetUser.Username)
	fmt.Println("• The user can no longer authenticate with that token")
	fmt.Println("• Use 'enable user' to generate a new token for this user")
	return nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func disableSecret(cmd *cobra.Command, key string) error {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to initialize platform: %w", err)
	}

	// Resolve token for authentication
	authToken, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, authToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check write permissions (needed for disabling secrets)
	err = app.Auth.Authorize(ctx, user, auth.PermissionWrite)
	if err != nil {
		return fmt.Errorf("write access denied: %w", err)
	}

	// Disable the secret using platform services
	err = app.Secrets.Disable(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to disable secret: %w", err)
	}

	fmt.Printf("✅ Secret '%s' has been disabled\n", key)
	fmt.Println("• The secret is hidden from normal operations")
	fmt.Println("• Use 'enable secret' to re-enable this secret")
	return nil
}

// confirmTokenDisable prompts the user for confirmation and returns their choice
func confirmTokenDisable() bool {
	fmt.Print("Proceed? (type 'yes'): ")
	in := bufio.NewReader(os.Stdin)
	line, _ := in.ReadString('\n')

	if strings.TrimSpace(strings.ToLower(line)) != "yes" {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

// completeDisableArgs provides completion for disable command arguments
func completeDisableArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: suggest disable types
		// "token" now accepts both usernames and token values
		return []string{"user", "token", "secret"}, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		switch args[0] {
		case "user":
			// Complete with available usernames (requires admin access)
			usernames, err := getAvailableUsernames(cmd)
			if err != nil {
				// If we can't get usernames (no auth/permissions), return no completion
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return usernames, cobra.ShellCompDirectiveNoFileComp
		case "token":
			// No completion for token values (they are unpredictable secret values)
			return nil, cobra.ShellCompDirectiveNoFileComp
		case "secret":
			// Complete with available secret names
			keys, err := getAvailableSecretKeys(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(disableCmd)

	// Add custom completion for disable command
	disableCmd.ValidArgsFunction = completeDisableArgs
}
