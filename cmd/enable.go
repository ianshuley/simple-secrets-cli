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
	"bufio"
	"fmt"
	"os"
	"simple-secrets/internal"
	"strings"

	"github.com/spf13/cobra"
)

// enableCmd represents the enable command
var enableCmd = &cobra.Command{
	Use:   "enable [user|secret] [username|key]",
	Short: "Re-enable previously disabled users or secrets",
	Long: `Re-enable resources that were previously disabled:
  • user <username>   - Generate new token for disabled user (admin only)
  • secret <key>      - Re-enable a disabled secret

Re-enabled secrets become available for normal operations again.
Re-enabled users receive a new authentication token.`,
	Example: `  simple-secrets enable user alice       # Generate new token for alice
  simple-secrets enable secret api-key   # Re-enable a disabled secret`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "user":
			return enableUser(cmd, args[1])
		case "secret":
			return enableSecret(cmd, args[1])
		default:
			return NewUnknownTypeError("enable", args[0], "'user' or 'secret'")
		}
	},
}

func enableUser(cmd *cobra.Command, username string) error {
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	// Check permissions first
	currentUser, store, err := helper.AuthenticateCommand(cmd, true)
	if err != nil {
		return err
	}
	if currentUser == nil {
		return nil
	}

	if !currentUser.Can("manage-users", store.Permissions()) {
		return NewPermissionDeniedError("manage-users")
	}

	// Show help for the enable user operation
	fmt.Println("⚠️  User Enable Operation")
	fmt.Println("• This will generate a new authentication token for the specified user")
	fmt.Println("• The user will be able to authenticate with the new token immediately")
	fmt.Println("• Any previously disabled token will remain invalid")
	fmt.Printf("• Target user: '%s'\n", username)
	fmt.Println()

	// Get confirmation
	if !confirmUserEnable() {
		fmt.Println("❌ User enable operation cancelled.")
		return nil
	}

	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	// Use service layer to generate new token for user
	newToken, err := helper.GetService().Users().EnableUser(resolvedToken, username)
	if err != nil {
		return err
	}

	fmt.Printf("✅ New token generated for user '%s'\n", username)
	fmt.Printf("• New token: %s\n", newToken)
	fmt.Println("• The user can now authenticate with this token")
	fmt.Println("• Previous tokens remain permanently disabled")
	return nil
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

// confirmUserEnable prompts user to confirm user enable operation
func confirmUserEnable() bool {
	fmt.Print("Generate new token for this user? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// completeEnableArgs provides completion for enable command arguments
func completeEnableArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: suggest enable types
		return []string{"user", "secret"}, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		switch args[0] {
		case "user":
			// Complete with available usernames (requires admin access)
			usernames, err := getAvailableUsernames(cmd)
			if err != nil {
				// If we can't get usernames (no auth/permissions), return no completion
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return usernames, cobra.ShellCompDirectiveNoFileComp
		case "secret":
			// Complete with available disabled secret names
			keys, err := getAvailableDisabledSecrets(cmd)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return keys, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(enableCmd)

	// Add custom completion for enable command
	enableCmd.ValidArgsFunction = completeEnableArgs
}
