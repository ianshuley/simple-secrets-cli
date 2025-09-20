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
	"slices"

	"github.com/spf13/cobra"
)

// enableCmd represents the enable command
var enableCmd = &cobra.Command{
	Use:   "enable [secret] [key]",
	Short: "Re-enable previously disabled secrets",
	Long: `Re-enable resources that were previously disabled:
  • secret <key> - Re-enable a disabled secret

Re-enabled secrets become available for normal operations again.`,
	Example: `  simple-secrets enable secret api-key  # Re-enable a disabled secret`,
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		switch args[0] {
		case "secret":
			return enableSecret(cmd, args[1])
		default:
			return fmt.Errorf("unknown enable type: %s. Use 'secret'", args[0])
		}
	},
}

func enableSecret(cmd *cobra.Command, key string) error {
	context, err := prepareSecretEnableContext(cmd, key)
	if err != nil {
		return err
	}
	if context == nil {
		return nil // First run or access denied
	}

	if err := executeSecretEnable(context); err != nil {
		return err
	}

	printSecretEnableSuccess(key)
	return nil
}

// SecretEnableContext holds data needed for secret enabling
type SecretEnableContext struct {
	RequestingUser *internal.User
	SecretKey      string
	Store          *internal.SecretsStore
}

// prepareSecretEnableContext validates access and prepares context for secret enabling
func prepareSecretEnableContext(cmd *cobra.Command, key string) (*SecretEnableContext, error) {
	user, _, err := validateSecretEnableAccess(cmd)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return nil, err
	}

	// Check if disabled secret exists
	disabledSecrets := store.ListDisabledSecrets()
	found := slices.Contains(disabledSecrets, key)

	if !found {
		return nil, fmt.Errorf("disabled secret not found")
	}

	return &SecretEnableContext{
		RequestingUser: user,
		SecretKey:      key,
		Store:          store,
	}, nil
}

// validateSecretEnableAccess checks RBAC permissions for secret enabling
func validateSecretEnableAccess(cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
	user, store, err := RBACGuard(true, cmd)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil
	}

	if !user.Can("write", store.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'write' permission to enable secrets")
	}

	return user, store, nil
}

// executeSecretEnable re-enables a previously disabled secret
func executeSecretEnable(context *SecretEnableContext) error {
	return context.Store.EnableSecret(context.SecretKey)
}

// printSecretEnableSuccess displays success message for secret enabling
func printSecretEnableSuccess(key string) {
	fmt.Printf("✅ Secret '%s' has been re-enabled\n", key)
	fmt.Println("• The secret is now available for normal operations")
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
