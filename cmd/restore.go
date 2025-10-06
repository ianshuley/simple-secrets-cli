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
	"strings"

	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"

	"github.com/spf13/cobra"
)

// restoreNewCmd represents the new consolidated restore command
var restoreCmd = &cobra.Command{
	Use:   "restore [secret|database]",
	Short: "Restore secrets or database from backups",
	Long: `Restore different types of data from backups:
  • secret   - Restore a specific secret from its backup
  • database - Restore the entire secrets database from a rotation backup`,
	Example: `  simple-secrets restore secret my-key
  simple-secrets restore database backup-2025-01-01_123456`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "secret":
			if len(args) < 2 {
				return fmt.Errorf("secret restore requires a key name")
			}
			return restoreSecret(cmd, args[1])
		case "database":
			if len(args) < 2 {
				return fmt.Errorf("database restore requires a backup name")
			}
			return restoreDatabase(cmd, args[1])
		default:
			return NewUnknownTypeError("restore", args[0], "'secret' or 'database'")
		}
	},
}

func restoreSecret(cmd *cobra.Command, secretKey string) error {
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

	// Check write permissions (needed for restore operations)
	err = app.Auth.Authorize(ctx, user, auth.PermissionWrite)
	if err != nil {
		return fmt.Errorf("write access denied: %w", err)
	}

	// Get the backup containing the secret
	backups, err := app.Rotation.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups available for secret restoration")
	}

	// Use the most recent backup for secret restoration (industry standard point-in-time recovery)
	// This ensures atomic consistency - all secrets are from the same point in time
	mostRecentBackup := backups[0]
	for _, backup := range backups {
		if backup.Timestamp.After(mostRecentBackup.Timestamp) {
			mostRecentBackup = backup
		}
	}

	fmt.Printf("⚠️  Secret restoration will use the most recent backup: %s\n", mostRecentBackup.Name)
	fmt.Printf("⚠️  This will restore the entire database state from %s\n", mostRecentBackup.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Print("Continue? [y/N]: ")

	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Secret restoration cancelled.")
		return nil
	}

	// Restore from the backup
	err = app.Rotation.RestoreFromBackup(ctx, mostRecentBackup.Name)
	if err != nil {
		return fmt.Errorf("secret restoration failed: %w", err)
	}

	fmt.Printf("✅ Secret restoration completed successfully from backup %s\n", mostRecentBackup.Name)
	return nil
}

func restoreDatabase(cmd *cobra.Command, backupName string) error {
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

	// Check write permissions (needed for restore operations)
	err = app.Auth.Authorize(ctx, user, auth.PermissionWrite)
	if err != nil {
		return fmt.Errorf("write access denied: %w", err)
	}

	// Restore from the specified backup
	err = app.Rotation.RestoreFromBackup(ctx, backupName)
	if err != nil {
		return fmt.Errorf("database restoration failed: %w", err)
	}

	fmt.Printf("✅ Database restoration completed successfully from backup %s\n", backupName)
	return nil
}

// completeRestoreArgs provides completion for restore command arguments
func completeRestoreArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: suggest restore types
		return []string{"secret", "database"}, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		switch args[0] {
		case "secret":
			// Second argument for secret restore: complete with backed-up secret names
			keys, err := getAvailableBackupSecrets(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		case "database":
			// Second argument for database restore: complete with backup names
			backups, err := getAvailableBackupNames(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return backups, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	// Add custom completion for restore command
	restoreCmd.ValidArgsFunction = completeRestoreArgs
}
