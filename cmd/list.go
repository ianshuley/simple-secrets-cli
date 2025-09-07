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
	"sort"
	"strconv"
	"time"

	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

// listNewCmd represents the new consolidated list command
var listCmd = &cobra.Command{
	Use:   "list [keys|backups|users|disabled]",
	Short: "List secrets, backups, users, or disabled secrets",
	Long: `List different types of data in the system:
  • keys     - List all stored secret keys
  • backups  - List available rotation backups
  • users    - List all users in the system
  • disabled - List all disabled secrets`,
	Example: `  simple-secrets list keys
  simple-secrets list backups
  simple-secrets list users
  simple-secrets list disabled`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		switch args[0] {
		case "keys":
			return listKeys()
		case "backups":
			return listBackups()
		case "users":
			return listUsers()
		case "disabled":
			return listDisabledSecrets()
		default:
			return fmt.Errorf("unknown list type: %s. Use 'keys', 'backups', 'users', or 'disabled'", args[0])
		}
	},
}

func listKeys() error {
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

	keys := store.ListKeys()
	if len(keys) == 0 {
		fmt.Println("(no secrets)")
		return nil
	}
	for _, k := range keys {
		// Escape special characters to prevent multiline display issues
		escaped := safeDisplayFormat(k)
		fmt.Println(escaped)
	}
	return nil
}

func listBackups() error {
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

	backups, err := store.ListRotationBackups()
	if err != nil {
		return err
	}

	if len(backups) == 0 {
		fmt.Println("(no rotation backups available)")
		return nil
	}

	fmt.Printf("Found %d rotation backup(s):\n\n", len(backups))
	for _, backup := range backups {
		fmt.Printf("  📁 %s\n", backup.Name)
		fmt.Printf("     Created: %s\n", backup.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("     Location: %s\n\n", backup.Path)
	}

	return nil
}

func listUsers() error {
	// RBAC: admin required for user management
	user, store, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	if !user.Can("manage-users", store.Permissions()) {
		return fmt.Errorf("permission denied: need 'manage-users' permission")
	}

	// Load current users
	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return err
	}
	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d user(s):\n\n", len(users))

	// Sort users by username for consistent output
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})

	for _, u := range users {
		// User icon based on role
		icon := "👤"
		if u.Role == internal.RoleAdmin {
			icon = "🔑"
		}

		// Current user indicator
		currentUserIndicator := ""
		if u.Username == user.Username {
			currentUserIndicator = " (current user)"
		}

		fmt.Printf("  %s %s%s\n", icon, u.Username, currentUserIndicator)
		fmt.Printf("    Role: %s\n", u.Role)

		// Display token rotation timestamp or legacy user indicator
		fmt.Printf("    Token last rotated: %s\n", getTokenRotationDisplay(u.TokenRotatedAt))
		fmt.Println()
	}

	return nil
}

func listDisabledSecrets() error {
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

	disabledSecrets := store.ListDisabledSecrets()
	if len(disabledSecrets) == 0 {
		fmt.Println("No disabled secrets found.")
		return nil
	}

	fmt.Printf("Disabled secrets (%d):\n", len(disabledSecrets))
	for _, key := range disabledSecrets {
		escaped := safeDisplayFormat(key)
		fmt.Printf("  🚫 %s\n", escaped)
	}
	fmt.Println()
	fmt.Println("Use 'enable secret <key>' to re-enable a disabled secret.")

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func getTokenRotationDisplay(tokenRotatedAt *time.Time) string {
	if tokenRotatedAt == nil {
		return "Unknown (legacy user)"
	}
	return tokenRotatedAt.Format("2006-01-02 15:04:05")
}

func safeDisplayFormat(key string) string {
	return strconv.Quote(key)
}
