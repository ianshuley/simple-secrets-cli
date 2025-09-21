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
		// RBAC: read access
		user, _, err := RBACGuard(false, cmd)
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
				return NewSecretNotFoundError()
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
