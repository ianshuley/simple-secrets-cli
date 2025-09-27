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
	// RBAC: write access (restoring is a write operation)
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	user, _, err := helper.AuthenticateCommand(cmd, true)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	service := helper.GetService()
	if err := service.Admin().RestoreSecret(secretKey); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' restored from backup.\n", secretKey)
	return nil
}

func restoreDatabase(cmd *cobra.Command, backupName string) error {
	// RBAC: write access (this is a destructive operation)
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	user, _, err := helper.AuthenticateCommand(cmd, true)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	service := helper.GetService()
	if err := service.Admin().RestoreDatabase(backupName); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	fmt.Printf("✅ Database restored successfully from backup '%s'\n", backupName)
	fmt.Println()
	fmt.Println("All secrets have been restored from the backup.")
	fmt.Println("The master key has been restored to the backup state.")
	fmt.Println()
	fmt.Println("⚠️  Important:")
	fmt.Println("• Any secrets added after this backup are now lost")
	fmt.Println("• The master key is now the one from the backup")
	fmt.Println("• Consider running 'simple-secrets list' to verify restored secrets")

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
