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
		user, _, err := RBACGuard(true, cmd)
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
