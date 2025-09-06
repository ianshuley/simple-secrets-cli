package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:     "put [key] [value]",
	Short:   "Store a secret securely.",
	Long:    "Store a secret value under a key. Overwrites if the key exists. Backs up previous value.",
	Example: "simple-secrets put db_password s3cr3tP@ssw0rd",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		// RBAC: write access
		user, _, err := RBACGuard(true, TokenFlag)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		// Initialize the secrets store
		store, err := internal.LoadSecretsStore()
		if err != nil {
			return err
		}

		key := args[0]
		value := args[1]

		// Validate key name
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("key name cannot be empty")
		}

		// Backup previous value if it exists
		prev, err := store.Get(key)
		if err == nil {
			// Only backup if key existed
			home, err := os.UserHomeDir()
			if err == nil {
				backupDir := filepath.Join(home, ".simple-secrets", "backups")
				_ = os.MkdirAll(backupDir, 0700)
				backupPath := filepath.Join(backupDir, key+".bak")
				_ = os.WriteFile(backupPath, []byte(prev), 0600)
			}
		}

		// Save secret (encrypted)
		if err := store.Put(key, value); err != nil {
			return err
		}

		fmt.Printf("Secret %q stored.\n", key)
		return nil
	},
}

var addCmd = &cobra.Command{
	Use:     "add [key] [value]",
	Short:   "Add a secret (alias for put).",
	Long:    "Store a secret with the given key and value. This is an alias for the 'put' command.",
	Example: "simple-secrets add db_password mypassword",
	Args:    cobra.ExactArgs(2),
	RunE:    putCmd.RunE, // Same implementation as put
}

func init() {
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(addCmd)
}
