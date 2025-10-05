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
	"context"
	"fmt"
	"os"
	"strings"

	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"

	"github.com/spf13/cobra"
)

var createUserCmd = &cobra.Command{
	Use:     "create-user [username] [role]",
	Short:   "Create a new user (admin or reader).",
	Long:    "Create a new user and generate a secure token. Admins can manage users and secrets; readers can only view secrets.",
	Example: "simple-secrets create-user alice reader\nsimple-secrets create-user bob admin",
	Args:    cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return ErrAuthenticationRequired
		}

		// Get platform configuration
		config, err := getPlatformConfig()
		if err != nil {
			return err
		}

		// Initialize platform services
		ctx := context.Background()
		app, err := platform.New(ctx, config)
		if err != nil {
			return fmt.Errorf("failed to initialize platform: %w", err)
		}

		// Resolve token for authentication
		authToken, err := resolveTokenFromCommand(cmd)
		if err != nil {
			return err
		}

		// Authenticate user
		user, err := app.Auth.Authenticate(ctx, authToken)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Check manage-users permissions
		err = app.Auth.Authorize(ctx, user, auth.PermissionManageUsers)
		if err != nil {
			return fmt.Errorf("manage-users access denied: %w", err)
		}

		// Collect user input
		userInput, err := collectUserInput(args)
		if err != nil {
			return err
		}

		// Create user using platform services
		newUser, newToken, err := app.Users.Create(ctx, userInput.Username, userInput.Role)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		printUserCreationSuccess(newUser.Username, newToken)
		return nil
	},
}

// UserInput represents the input data for creating a new user
type UserInput struct {
	Username string
	Role     string
}

// collectUserInput gathers username and role from args or interactive prompts
func collectUserInput(args []string) (*UserInput, error) {
	reader := bufio.NewReader(os.Stdin)
	var username string
	var roleStr string

	// Get username from args or prompt
	if len(args) >= 1 {
		username = args[0]
	}
	if username == "" {
		fmt.Print("Username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}

	// Get role from args or prompt
	if len(args) >= 2 {
		roleStr = args[1]
	}
	if roleStr == "" {
		fmt.Print("Role (admin/reader): ")
		roleStr, _ = reader.ReadString('\n')
		roleStr = strings.TrimSpace(roleStr)
	}

	role, err := parseRole(roleStr)
	if err != nil {
		return nil, err
	}

	return &UserInput{
		Username: username,
		Role:     role,
	}, nil
}

// parseRole validates a role string
func parseRole(roleStr string) (string, error) {
	switch roleStr {
	case "admin":
		return "admin", nil
	case "reader":
		return "reader", nil
	default:
		return "", fmt.Errorf("invalid role: must be 'admin' or 'reader'")
	}
}

// printUserCreationSuccess displays the success message with the new token
func printUserCreationSuccess(username, token string) {
	fmt.Printf("User %q created.\n", username)
	fmt.Printf("Generated token: %s\n", token)
}

// completeCreateUserArgs provides completion for create-user command arguments
func completeCreateUserArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		// First argument: username - no completion (user input)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if len(args) == 1 {
		// Second argument: role - suggest valid roles
		return []string{"admin", "reader"}, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(createUserCmd)

	// Add custom completion for create-user command
	createUserCmd.ValidArgsFunction = completeCreateUserArgs
}
