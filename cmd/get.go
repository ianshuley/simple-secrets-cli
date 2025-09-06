package cmd

import (
	"errors"
	"fmt"
	"os"
	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:     "get [key]",
	Short:   "Retrieve a secret.",
	Long:    "Retrieve the value for a given secret key.",
	Example: "simple-secrets get db_password",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		// RBAC: read access
		user, _, err := RBACGuard(false, TokenFlag)
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

		key := args[0]
		value, err := store.Get(key)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("secret '%s' not found", key)
			}
			return err
		}

		fmt.Println(value)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
