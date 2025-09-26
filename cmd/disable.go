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

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:   "disable [token|user|secret] [username|key]",
	Short: "Disable user tokens or secrets",
	Long: `Disable different types of resources in the system:
  • token <username> - Disable a user's token (admin only)
  • user <username>  - Disable a user's token (alias for 'token')
  • secret <key>     - Mark a secret as disabled

Disabled tokens cannot be used for authentication.
Disabled secrets are hidden from normal operations but can be re-enabled.`,
	Example: `  simple-secrets disable token alice    # Disable alice's token
  simple-secrets disable user alice     # Same as above (alias)
  simple-secrets disable secret api-key  # Disable a secret`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		switch args[0] {
		case "token", "user": // Accept both "token" and "user" as aliases
			return disableToken(cmd, args[1])
		case "secret":
			return disableSecret(cmd, args[1])
		default:
			return NewUnknownTypeError("disable", args[0], "'token', 'user', or 'secret'")
		}
	},
}

func disableToken(cmd *cobra.Command, username string) error {
	// Show detailed help for the disable token operation
	fmt.Println("⚠️  User Token Disable Operation")
	fmt.Println("• This will disable the specified user's authentication token immediately")
	fmt.Println("• The user will no longer be able to authenticate until a new token is generated")
	fmt.Println("• Use 'rotate token' to generate a new token for this user")
	fmt.Printf("• Target user: '%s'\n", username)
	fmt.Println()

	// Get confirmation for token disable operation
	if !confirmTokenDisable() {
		fmt.Println("❌ Token disable operation cancelled.")
		return nil
	}

	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return err
	}

	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return err
	}

	if err := helper.GetService().Users().DisableUser(resolvedToken, username); err != nil {
		return err
	}

	fmt.Printf("✅ Token disabled for user '%s'\n", username)
	fmt.Println("• The user can no longer authenticate with their current token")
	fmt.Println("• Use 'rotate token' to generate a new token for this user")
	return nil
}

func disableSecret(cmd *cobra.Command, key string) error {
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

	// Use service layer for secret disabling
	err = helper.GetService().Secrets().Disable(resolvedToken, key)
	if err != nil {
		return err
	}

	fmt.Printf("✅ Secret '%s' has been disabled\n", key)
	fmt.Println("• The secret is hidden from normal operations")
	fmt.Println("• Use 'enable secret' to re-enable this secret")
	return nil
}

// confirmTokenDisable prompts the user for confirmation and returns their choice
func confirmTokenDisable() bool {
	fmt.Print("Proceed? (type 'yes'): ")
	in := bufio.NewReader(os.Stdin)
	line, _ := in.ReadString('\n')

	if strings.TrimSpace(strings.ToLower(line)) != "yes" {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
