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
	"encoding/json"
	"fmt"
	"os"
	"simple-secrets/internal"
	"strings"
	"time"

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
	user, store, err := validateMasterKeyRotationAccess(cmd)
	if err != nil {
		return err
	}
	if user == nil {
		return nil // First run detected, message already printed
	}

	if !rotateNewYes && !confirmMasterKeyRotation() {
		return nil
	}

	if err := store.RotateMasterKey(rotateNewBackupDir); err != nil {
		return err
	}

	printMasterKeyRotationSuccess()
	return nil
}

// validateMasterKeyRotationAccess checks RBAC permissions for master key rotation
func validateMasterKeyRotationAccess(cmd *cobra.Command) (*internal.User, *internal.SecretsStore, error) {
	user, _, err := RBACGuard(true, cmd)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil // First run message already printed
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return nil, nil, err
	}

	return user, store, nil
}

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
	context, err := prepareTokenRotationContext(cmd, true) // self=true
	if err != nil {
		return err
	}
	if context == nil {
		return nil // First run or access denied
	}

	newToken, err := executeTokenRotation(context)
	if err != nil {
		return err
	}

	printSelfTokenRotationSuccess(context.TargetUsername, context.TargetUser.Role, newToken)
	return nil
}

func rotateToken(cmd *cobra.Command, targetUsername string) error {
	// First, check if this is actually self-rotation (user specified their own username)
	currentUser, _, err := RBACGuard(false, cmd) // Don't require write access yet
	if err != nil {
		return err
	}
	if currentUser == nil {
		return nil // First run detected
	}

	// If the target username matches the current user, treat as self-rotation
	if targetUsername == currentUser.Username {
		return rotateSelfToken(cmd)
	}

	// Otherwise, proceed with admin rotation (requires rotate-tokens permission)
	context, err := prepareTokenRotationContextForUser(cmd, targetUsername)
	if err != nil {
		return err
	}
	if context == nil {
		return nil // First run or access denied
	}

	newToken, err := executeTokenRotation(context)
	if err != nil {
		return err
	}

	printTokenRotationSuccess(context.TargetUsername, context.TargetUser.Role, newToken)
	return nil
}

func init() {
	rotateCmd.Flags().BoolVar(&rotateNewYes, "yes", false, "Skip confirmation prompt for master key rotation")
	rotateCmd.Flags().StringVar(&rotateNewBackupDir, "backup-dir", "", "Custom backup directory for master key rotation")

	rootCmd.AddCommand(rotateCmd)
}

func printBackupLocation(backupDir string) {
	if backupDir == "" {
		fmt.Println("📁 Backup created under ~/.simple-secrets/backups/")
		return
	}
	fmt.Printf("📁 Backup created at %s\n", backupDir)
}

// validateTokenRotationAccess checks permissions and loads necessary data for token rotation
func validateTokenRotationAccess(cmd *cobra.Command, targetUsername string) (*internal.User, string, []*internal.User, error) {
	currentUser, store, err := RBACGuard(true, cmd)
	if err != nil {
		return nil, "", nil, err
	}
	if currentUser == nil {
		return nil, "", nil, nil
	}

	if !currentUser.Can("rotate-tokens", store.Permissions()) {
		return nil, "", nil, NewPermissionDeniedError("rotate-tokens")
	}

	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return nil, "", nil, err
	}

	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return nil, "", nil, err
	}

	return currentUser, usersPath, users, nil
}

// validateSelfTokenRotationAccess checks permissions for self token rotation
func validateSelfTokenRotationAccess(cmd *cobra.Command) (*internal.User, string, []*internal.User, error) {
	currentUser, store, err := RBACGuard(false, cmd) // Use false - we check specific permission below
	if err != nil {
		return nil, "", nil, err
	}
	if currentUser == nil {
		return nil, "", nil, nil
	}

	if !currentUser.Can("rotate-own-token", store.Permissions()) {
		return nil, "", nil, fmt.Errorf("permission denied: cannot rotate own token")
	}

	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return nil, "", nil, err
	}

	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return nil, "", nil, err
	}

	return currentUser, usersPath, users, nil
}

// findUserIndex locates the target user in the users slice and returns their index
func findUserIndex(users []*internal.User, username string) (int, error) {
	for i, u := range users {
		if u.Username == username {
			return i, nil
		}
	}
	return -1, NewUserNotFoundError(username)
}

// generateAndUpdateUserToken creates a new token and updates the user record
func generateAndUpdateUserToken(users []*internal.User, targetIndex int) (string, error) {
	newToken, err := GenerateSecureToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	newTokenHash := internal.HashToken(newToken)
	now := time.Now()

	users[targetIndex].TokenHash = newTokenHash
	users[targetIndex].TokenRotatedAt = &now

	return newToken, nil
}


// TokenRotationContext holds all the data needed for token rotation
type TokenRotationContext struct {
	RequestingUser *internal.User
	TargetUser     *internal.User
	TargetUsername string
	TargetIndex    int
	UsersPath      string
	Users          []*internal.User
	IsSelfRotation bool
}

// prepareTokenRotationContext prepares the context for self token rotation
func prepareTokenRotationContext(cmd *cobra.Command, isSelfRotation bool) (*TokenRotationContext, error) {
	currentUser, usersPath, users, err := validateSelfTokenRotationAccess(cmd)
	if err != nil {
		return nil, err
	}
	if currentUser == nil {
		return nil, nil
	}

	targetIndex, err := findUserIndex(users, currentUser.Username)
	if err != nil {
		return nil, err
	}

	return &TokenRotationContext{
		RequestingUser: currentUser,
		TargetUser:     users[targetIndex],
		TargetUsername: currentUser.Username,
		TargetIndex:    targetIndex,
		UsersPath:      usersPath,
		Users:          users,
		IsSelfRotation: isSelfRotation,
	}, nil
}

// prepareTokenRotationContextForUser prepares the context for admin token rotation
func prepareTokenRotationContextForUser(cmd *cobra.Command, targetUsername string) (*TokenRotationContext, error) {
	currentUser, usersPath, users, err := validateTokenRotationAccess(cmd, targetUsername)
	if err != nil {
		return nil, err
	}
	if currentUser == nil {
		return nil, nil
	}

	targetIndex, err := findUserIndex(users, targetUsername)
	if err != nil {
		return nil, err
	}

	return &TokenRotationContext{
		RequestingUser: currentUser,
		TargetUser:     users[targetIndex],
		TargetUsername: targetUsername,
		TargetIndex:    targetIndex,
		UsersPath:      usersPath,
		Users:          users,
		IsSelfRotation: false,
	}, nil
}

// executeTokenRotation performs the actual token rotation and persistence
func executeTokenRotation(context *TokenRotationContext) (string, error) {
	newToken, err := generateAndUpdateUserToken(context.Users, context.TargetIndex)
	if err != nil {
		return "", err
	}

	if err := saveUsersList(context.UsersPath, context.Users); err != nil {
		return "", err
	}

	return newToken, nil
}

// saveUsersList marshals and saves the users list to disk atomically
func saveUsersList(usersPath string, users []*internal.User) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	return internal.AtomicWriteFile(usersPath, data, 0600)
}

// printTokenRotationSuccess displays the success message and instructions
func printTokenRotationSuccess(username string, role internal.Role, newToken string) {
	fmt.Printf("\nToken rotated for user \"%s\" (%s role).\n", username, role)
	fmt.Printf("New token: %s\n", newToken)
	fmt.Println()
	printTokenRotationWarnings()
	fmt.Println()
	printTokenUsageInstructions()
}

// printSelfTokenRotationSuccess displays the self-rotation success message and instructions
func printSelfTokenRotationSuccess(username string, role internal.Role, newToken string) {
	fmt.Printf("\n✅ Your token has been rotated successfully!\n")
	fmt.Printf("New token: %s\n", newToken)
	fmt.Println()
	printSelfTokenRotationWarnings()
	fmt.Println()
	printSelfTokenUsageInstructions()
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
