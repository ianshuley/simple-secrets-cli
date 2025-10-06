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

	internal "simple-secrets/internal/auth"
	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"

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
		// Get platform configuration
		config, err := getPlatformConfig()
		if err != nil {
			return fmt.Errorf("failed to get platform config: %w", err)
		}

		// Initialize platform services
		ctx := context.Background()
		app, err := platform.New(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to initialize platform: %w", err)
		}

		// Resolve token for authentication
		token, err := resolveTokenFromCommand(cmd)
		if err != nil {
			return err
		}

		// Resolve the token (temporary - use old internal for now)
		resolvedToken, err := internal.ResolveToken(token)
		if err != nil {
			return err
		}

		// Authenticate user
		user, err := app.Auth.Authenticate(ctx, resolvedToken)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Check write permissions
		err = app.Auth.Authorize(ctx, user, auth.PermissionWrite)
		if err != nil {
			return fmt.Errorf("write access denied: %w", err)
		}

		key := args[0]

		// Delete secret using platform services
		if err := app.Secrets.Delete(ctx, key); err != nil {
			return err
		}

		fmt.Printf("Secret %q deleted.\n", key)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	// Add completion for secret names
	deleteCmd.ValidArgsFunction = completeSecretNames
}
