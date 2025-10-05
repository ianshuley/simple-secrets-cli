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
	"context"
	"fmt"
	"sort"
	"time"

	"simple-secrets/internal"
	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"

	"github.com/spf13/cobra"
)

// listNewCmd represents the new consolidated list command
var listCmd = &cobra.Command{
	Use:   "list [keys|backups|users|disabled]",
	Short: "List secrets, backups, users, or disabled secrets",
	Long: `List different types of data in the system:
  • keys     - List all stored secret keys
  • backups  - List available rotation backups
  • users    - List all users in the system
  • disabled - List all disabled secrets`,
	Example: `  simple-secrets list keys
  simple-secrets list backups
  simple-secrets list users
  simple-secrets list disabled`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "keys":
			return listKeys(cmd)
		case "backups":
			return listBackups(cmd)
		case "users":
			return listUsers(cmd)
		case "disabled":
			return listDisabledSecrets(cmd)
		default:
			return NewUnknownTypeError("list", args[0], "'keys', 'backups', 'users', or 'disabled'")
		}
	},
}

func listKeys(cmd *cobra.Command) error {
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
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Resolve the token (temporary - use old internal for now)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, resolvedToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return fmt.Errorf("read access denied: %w", err)
	}

	// List secrets using platform services
	secretsMetadata, err := app.Secrets.List(ctx)
	if err != nil {
		return err
	}

	// Extract keys from metadata
	keys := make([]string, len(secretsMetadata))
	for i, metadata := range secretsMetadata {
		keys[i] = metadata.Key
	}

	if len(keys) == 0 {
		fmt.Println("(no secrets)")
		return nil
	}
	for _, k := range keys {
		fmt.Println(k)
	}
	return nil
}

func listBackups(cmd *cobra.Command) error {
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
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Resolve the token (temporary - use old internal for now)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, resolvedToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return fmt.Errorf("read access denied: %w", err)
	}

	// List backups using platform services
	backups, err := app.Rotation.ListBackups(ctx)
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		fmt.Println("(no rotation backups available)")
		return nil
	}

	fmt.Printf("Found %d rotation backup(s):\n\n", len(backups))
	for _, backup := range backups {
		fmt.Printf("  📁 %s\n", backup.Name)
		fmt.Printf("     Created: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("     Location: %s\n\n", backup.Path)
	}

	return nil
}

func listUsers(cmd *cobra.Command) error {
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
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Resolve the token (temporary - use old internal for now)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, resolvedToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check manage users permissions (admin required)
	err = app.Auth.Authorize(ctx, user, auth.PermissionManageUsers)
	if err != nil {
		return fmt.Errorf("user management access denied: %w", err)
	}

	// List users using platform services
	users, err := app.Users.List(ctx)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d user(s):\n\n", len(users))

	// Sort users by username for consistent output
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	for _, u := range users {
		// User icon based on role
		icon := "👤"
		if u.Role == string(auth.RoleAdmin) {
			icon = "🔑"
		}

		// Current user indicator (compare with authenticated user username)
		currentUserIndicator := ""
		if u.Username == user.Username {
			currentUserIndicator = " (current user)"
		}

		fmt.Printf("  %s %s%s\n", icon, u.Username, currentUserIndicator)
		fmt.Printf("    Role: %s\n", u.Role)

		// Display token rotation timestamp or legacy user indicator
		fmt.Printf("    Token last rotated: %s\n", getTokenRotationDisplay(u.TokenRotatedAt))
		fmt.Println()
	}

	return nil
}

func listDisabledSecrets(cmd *cobra.Command) error {
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
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Resolve the token (temporary - use old internal for now)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, resolvedToken)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return fmt.Errorf("read access denied: %w", err)
	}

	// List disabled secrets using platform services
	disabledSecretsMetadata, err := app.Secrets.ListDisabled(ctx)
	if err != nil {
		return err
	}

	// Extract keys from metadata
	disabledSecrets := make([]string, len(disabledSecretsMetadata))
	for i, metadata := range disabledSecretsMetadata {
		disabledSecrets[i] = metadata.Key
	}
	if len(disabledSecrets) == 0 {
		fmt.Println("No disabled secrets found.")
		return nil
	}

	fmt.Printf("Disabled secrets (%d):\n", len(disabledSecrets))
	for _, key := range disabledSecrets {
		fmt.Printf("  🚫 %s\n", key)
	}
	fmt.Println()
	fmt.Println("Use 'enable secret <key>' to re-enable a disabled secret.")

	return nil
}

// completeListArgs provides completion for list command arguments
func completeListArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: suggest list types
		return []string{"keys", "backups", "users", "disabled"}, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add custom completion for list command
	listCmd.ValidArgsFunction = completeListArgs
}

func getTokenRotationDisplay(tokenRotatedAt *time.Time) string {
	if tokenRotatedAt == nil {
		return "Unknown (legacy user)"
	}
	return tokenRotatedAt.Format("2006-01-02 15:04:05")
}
