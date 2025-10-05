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
	restoreYes        bool
	restoreBackupName string
)

var restoreDatabaseCmd = &cobra.Command{
	Use:     "restore-database [backup-name]",
	Aliases: []string{"restore-db"},
	Short:   "Restore secrets database from a rotation backup.",
	Long:    "Restore the entire secrets database from a rotation backup. If no backup name is specified, uses the most recent valid backup. This operation creates a backup of the current state before restoring.",
	Example: "simple-secrets restore-database\nsimple-secrets restore-database rotate-20240901-143022",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Check write permissions (destructive operation)
		err = app.Auth.Authorize(ctx, user, auth.PermissionWrite)
		if err != nil {
			return fmt.Errorf("write access denied: %w", err)
		}

		// Determine which backup to restore
		if len(args) >= 1 {
			restoreBackupName = args[0]
		}

		// Show what backup will be restored
		if restoreBackupName == "" {
			backups, err := app.Rotation.ListBackups(ctx)
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}
			if len(backups) == 0 {
				fmt.Println("No rotation backups found.")
				return nil
			}

			// Find most recent backup (first one is most recent)
			mostRecent := backups[0]
			if !mostRecent.IsValid {
				fmt.Println("No valid rotation backups found.")
				return nil
			}

			fmt.Printf("Will restore from most recent backup: %s\n", mostRecent.Name)
			fmt.Printf("  Created: %s\n", mostRecent.Timestamp.Format("2006-01-02 15:04:05"))
			restoreBackupName = mostRecent.Name
		}

		// Display backup name for non-most-recent restores
		if restoreBackupName != "" {
			fmt.Printf("Will restore from backup: %s\n", restoreBackupName)
		}

		if !restoreYes {
			fmt.Println("\nThis will:")
			fmt.Println("  • Create a backup of your current secrets database")
			fmt.Println("  • Replace your current database with the specified backup")
			fmt.Println("  • This action affects ALL secrets in your database")
			fmt.Print("Proceed? (type 'yes'): ")

			in := bufio.NewReader(os.Stdin)
			line, _ := in.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(line)) != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Perform the restore
		if err := app.Rotation.RestoreFromBackup(ctx, restoreBackupName); err != nil {
			return fmt.Errorf("database restoration failed: %w", err)
		} // Display success message
		successMessage := determineRestoreSuccessMessage(restoreBackupName)
		fmt.Println(successMessage)
		fmt.Println("Your previous database was backed up before the restore operation.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(restoreDatabaseCmd)
	restoreDatabaseCmd.Flags().BoolVar(&restoreYes, "yes", false, "Skip confirmation prompt")

	// Add custom completion for restore-database command
	restoreDatabaseCmd.ValidArgsFunction = completeRestoreDatabaseArgs
}

// completeRestoreDatabaseArgs provides completion for restore-database command arguments
func completeRestoreDatabaseArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: backup name - suggest available backup names
		backups, err := getAvailableBackupNames(cmd)
		if err != nil {
			// If we can't get backup names, return no completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return backups, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

// determineRestoreSuccessMessage returns the appropriate success message
func determineRestoreSuccessMessage(backupName string) string {
	defaultMessage := "Database restored from most recent backup successfully."
	if backupName == "" {
		return defaultMessage
	}
	return fmt.Sprintf("Database restored from backup '%s' successfully.", backupName)
}
