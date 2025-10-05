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

var (
	rotateNewYes       bool
	rotateNewBackupDir string
)

// rotateNewCmd represents the new consolidated rotate command
var rotateCmd = &cobra.Command{
	Use:   "rotate [master-key|token]",
	Short: "Rotate master key or user tokens",
	Long: `Rotate different types of keys in the system:
  • master-key - Rotate the master encryption key and re-encrypt all secrets
  • token      - Generate a new authentication token for a user or yourself

Token rotation options:
  • token             - Self: rotate your own token (no username needed)
  • token <username>  - Admin: rotate another user's token`,
	Example: `  simple-secrets rotate master-key --yes
  simple-secrets rotate token           # Rotate your own token
  simple-secrets rotate token alice     # Admin rotates alice's token`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "master-key":
			return rotateMasterKey(cmd)
		case "token":
			if len(args) < 2 {
				// No username provided - self-rotation
				return rotateSelfToken(cmd)
			}
			// Username provided - admin rotation
			return rotateToken(cmd, args[1])
		default:
			return NewUnknownTypeError("rotate", args[0], "'master-key' or 'token'")
		}
	},
}

func rotateMasterKey(cmd *cobra.Command) error {
	// Master key rotation requires careful re-implementation with platform services
	// This involves re-encrypting the entire secrets database and is complex
	return fmt.Errorf("master key rotation is temporarily disabled during platform migration")
}

// validateMasterKeyRotationAccess - unused, kept for potential future implementation
// func validateMasterKeyRotationAccess(cmd *cobra.Command) (*internal.User, *internal.SecretsStore, error) {
//   // Implementation removed during platform migration
// }

// confirmMasterKeyRotation prompts the user for confirmation and returns their choice
func confirmMasterKeyRotation() bool {
	printMasterKeyRotationWarning()

	fmt.Print("Proceed? (type 'yes'): ")
	in := bufio.NewReader(os.Stdin)
	line, _ := in.ReadString('\n')

	if strings.TrimSpace(strings.ToLower(line)) != "yes" {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

// printMasterKeyRotationWarning displays the warning about what master key rotation will do
func printMasterKeyRotationWarning() {
	fmt.Println("This will:")
	fmt.Println("  • Generate a NEW master key")
	fmt.Println("  • Re-encrypt ALL secrets with the new key")
	fmt.Println("  • Create a backup of the old key+secrets for rollback")
}

// printMasterKeyRotationSuccess displays the success message after rotation
func printMasterKeyRotationSuccess() {
	fmt.Println("✅ Master key rotation completed successfully!")
	printBackupLocation(rotateNewBackupDir)
	fmt.Println()
	fmt.Println("All secrets have been re-encrypted with the new master key.")
	fmt.Println("The old master key and secrets are backed up for emergency recovery.")
}

func rotateSelfToken(cmd *cobra.Command) error {
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

	// Users can always rotate their own token (no additional permission check needed)
	newToken, err := app.Users.RotateToken(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to rotate token: %w", err)
	}

	printSelfTokenRotationSuccess(newToken)
	return nil
}

func rotateToken(cmd *cobra.Command, targetUsername string) error {
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

	// If the target username matches the current user, treat as self-rotation
	if targetUsername == user.Username {
		return rotateSelfToken(cmd)
	}

	// Otherwise, check rotate-tokens permission for admin rotation
	err = app.Auth.Authorize(ctx, user, auth.PermissionRotateTokens)
	if err != nil {
		return fmt.Errorf("rotate-tokens access denied: %w", err)
	}

	// Get the target user to check if they exist
	targetUser, err := app.Users.GetByUsername(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("failed to find target user: %w", err)
	}

	// Rotate the target user's token
	newToken, err := app.Users.RotateToken(ctx, targetUsername)
	if err != nil {
		return fmt.Errorf("failed to rotate token: %w", err)
	}

	printTokenRotationSuccess(targetUsername, targetUser.Role, newToken)
	return nil
}

// completeRotateArgs provides completion for rotate command arguments
func completeRotateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: suggest rotation types
		return []string{"master-key", "token"}, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 && args[0] == "token" {
		// Second argument for token rotation: complete with available usernames
		usernames, err := getAvailableUsernames(cmd)
		if err != nil {
			// If we can't get usernames (no auth/permissions), return no completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return usernames, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rotateCmd.Flags().BoolVar(&rotateNewYes, "yes", false, "Skip confirmation prompt for master key rotation")
	rotateCmd.Flags().StringVar(&rotateNewBackupDir, "backup-dir", "", "Custom backup directory for master key rotation")

	rootCmd.AddCommand(rotateCmd)

	// Add custom completion for rotate command
	rotateCmd.ValidArgsFunction = completeRotateArgs
}

func printBackupLocation(backupDir string) {
	if backupDir == "" {
		fmt.Println("📁 Backup created under ~/.simple-secrets/backups/")
		return
	}
	fmt.Printf("📁 Backup created at %s\n", backupDir)
}

// Legacy helper functions - unused after platform migration
/*
func validateTokenRotationAccess(cmd *cobra.Command, targetUsername string) (*internal.User, string, []*internal.User, error) {
	// Implementation removed during platform migration
	return nil, "", nil, fmt.Errorf("legacy function not implemented")
}
*/

// Legacy functions - commented out after platform migration
/*
All the helper functions for the old internal system have been replaced
by platform services in the main rotateSelfToken() and rotateToken() functions.
This includes:
- validateSelfTokenRotationAccess
- findUserIndex
- generateAndUpdateUserToken
- TokenRotationContext
- prepareTokenRotationContext
- prepareTokenRotationContextForUser
- executeTokenRotation
- saveUsersList
*/

// printTokenRotationSuccess displays the success message and instructions
func printTokenRotationSuccess(username string, role string, newToken string) {
	fmt.Printf("\nToken rotated for user \"%s\" (%s role).\n", username, role)
	fmt.Printf("New token: %s\n", newToken)
	fmt.Println()
	printTokenRotationWarnings()
	fmt.Println()
	printTokenUsageInstructions()
}

// printSelfTokenRotationSuccess displays the self-rotation success message and instructions
func printSelfTokenRotationSuccess(newToken string) {
	fmt.Printf("\n✅ Your token has been rotated successfully!\n")
	fmt.Println()
	printSelfTokenRotationWarnings()
	fmt.Println()
	printSelfTokenUsageInstructions()
	fmt.Println()
	fmt.Printf("New token: %s\n", newToken)
}

// printTokenRotationWarnings displays important warnings about the token rotation
func printTokenRotationWarnings() {
	fmt.Println("⚠️  IMPORTANT:")
	fmt.Println("• Store this token securely - it will not be shown again")
	fmt.Println("• The old token is now invalid and cannot be used")
	fmt.Println("• Update any scripts or configs that use the old token")
}

// printSelfTokenRotationWarnings displays important warnings about self token rotation
func printSelfTokenRotationWarnings() {
	fmt.Println("⚠️  IMPORTANT:")
	fmt.Println("• Store this token securely - it will not be shown again")
	fmt.Println("• Your old token is now invalid and cannot be used")
	fmt.Println("• Update your local configuration with the new token")
}

// printTokenUsageInstructions shows how to use the new token
func printTokenUsageInstructions() {
	fmt.Println("To use the new token:")
	fmt.Println("  --token <new-token> (as a flag)")
	fmt.Println("  SIMPLE_SECRETS_TOKEN=<new-token> (as an env var)")
	fmt.Println("  or update ~/.simple-secrets/config.json")
}

// printSelfTokenUsageInstructions shows how to use the new token for self-rotation
func printSelfTokenUsageInstructions() {
	fmt.Println("To use your new token:")
	fmt.Println("  export SIMPLE_SECRETS_TOKEN=<new-token>")
	fmt.Println("  or update ~/.simple-secrets/config.json")
	fmt.Println("  or use --token <new-token> flag in commands")
}
