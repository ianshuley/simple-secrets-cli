/*
Copyright © 2025 Ian Shuley

Licensed und		// Get secret using platform secrets service\n		key := args[0]\n		ctx := cmd.Context()\n		value, err := app.Secrets.Get(ctx, key)he Apache License, Version 2.0 (the "License");
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

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:     "get [key]",
	Short:   "Retrieve a secret.",
	Long:    "Retrieve the value for a given secret key.",
	Example: "simple-secrets get db_password",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get platform from command context
		app, err := getPlatformFromCommand(cmd)
		if err != nil {
			return err
		}

		// Authenticate user with platform auth service\n		_, err = authenticateWithPlatform(cmd, false) // false = read access only\n		if err != nil {\n			return err\n		}		// Get secret using platform secrets service
		key := args[0]
		ctx := cmd.Context()
		value, err := app.Secrets.Get(ctx, key)
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

	// Add completion for secret names
	getCmd.ValidArgsFunction = completeSecretNames
}
