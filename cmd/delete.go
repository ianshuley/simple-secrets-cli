package cmd

import (
	"fmt"

	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:     "delete [key]",
	Aliases: []string{"del", "rm"},
	Short:   "Delete a stored secret by key.",
	Long:    "Delete a secret by key. This cannot be undone.",
	Example: "simple-secrets delete db_password",
	Args:    cobra.ExactArgs(1),
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

		key := args[0]

		store, err := internal.LoadSecretsStore()
		if err != nil {
			return err
		}

		if err := store.Delete(key); err != nil {
			return err
		}

		fmt.Printf("Secret %q deleted.\n", key)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
