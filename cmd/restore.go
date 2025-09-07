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
	"os"

	"simple-secrets/internal"

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
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		switch args[0] {
		case "secret":
			if len(args) < 2 {
				return fmt.Errorf("secret restore requires a key name")
			}
			return restoreSecret(args[1])
		case "database":
			if len(args) < 2 {
				return fmt.Errorf("database restore requires a backup name")
			}
			return restoreDatabase(args[1])
		default:
			return fmt.Errorf("unknown restore type: %s. Use 'secret' or 'database'", args[0])
		}
	},
}

func restoreSecret(secretKey string) error {
	// RBAC: write access (restoring is a write operation)
	user, _, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	backupPath := fmt.Sprintf("%s/.simple-secrets/backups/%s.bak", home, secretKey)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("could not read backup: %w", err)
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return err
	}

	// Backup files are encrypted, so decrypt them first
	decryptedValue, err := store.DecryptBackup(string(data))
	if err != nil {
		return fmt.Errorf("failed to decrypt backup file: %w", err)
	}

	if err := store.Put(secretKey, decryptedValue); err != nil {
		return err
	}

	fmt.Printf("Secret '%s' restored from backup.\n", secretKey)
	return nil
}

func restoreDatabase(backupName string) error {
	// RBAC: write access (this is a destructive operation)
	user, _, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return err
	}

	if err := store.RestoreFromBackup(backupName); err != nil {
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

func init() {
	rootCmd.AddCommand(restoreCmd)
}
