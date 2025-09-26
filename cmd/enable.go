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
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "secret":
			return enableSecret(cmd, args[1])
		default:
			return NewUnknownTypeError("enable", args[0], "'secret'")
		}
	},
}

func enableSecret(cmd *cobra.Command, key string) error {
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	// Resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	// Resolve the token (CLI responsibility)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Use service layer for secret enabling
	err = helper.GetService().Secrets().Enable(resolvedToken, key)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Secret '%s' has been re-enabled\n", key)
	fmt.Println("• The secret is now available for normal operations")
	return nil
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
