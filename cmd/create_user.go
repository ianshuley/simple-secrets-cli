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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"simple-secrets/internal"

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
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		user, _, err := validateUserCreationAccess()
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		userInput, err := collectUserInput(args)
		if err != nil {
			return err
		}

		newUser, token, err := createUserWithToken(userInput)
		if err != nil {
			return err
		}

		if err := persistNewUser(newUser); err != nil {
			return err
		}

		printUserCreationSuccess(userInput.Username, token)
		return nil
	},
}

// UserInput represents the input data for creating a new user
type UserInput struct {
	Username string
	Role     internal.Role
}

// validateUserCreationAccess checks RBAC permissions for user creation
func validateUserCreationAccess() (*internal.User, *internal.UserStore, error) {
	user, store, err := RBACGuard(true, TokenFlag)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, nil
	}
	if !user.Can("manage-users", store.Permissions()) {
		fmt.Fprintln(os.Stderr, "Error: insufficient permissions")
		return nil, nil, nil
	}
	return user, store, nil
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

// parseRole converts a role string to the appropriate Role type
func parseRole(roleStr string) (internal.Role, error) {
	switch roleStr {
	case "admin":
		return internal.RoleAdmin, nil
	case "reader":
		return internal.RoleReader, nil
	default:
		return "", fmt.Errorf("invalid role: must be 'admin' or 'reader'")
	}
}

// createUserWithToken creates a new user with a secure token
func createUserWithToken(userInput *UserInput) (*internal.User, string, error) {
	if err := validateUsernameAvailability(userInput.Username); err != nil {
		return nil, "", err
	}

	token, err := generateSecureUserToken()
	if err != nil {
		return nil, "", err
	}

	tokenHash := internal.HashToken(token)
	now := time.Now()

	newUser := &internal.User{
		Username:       userInput.Username,
		TokenHash:      tokenHash,
		Role:           userInput.Role,
		TokenRotatedAt: &now,
	}

	return newUser, token, nil
}

// validateUsernameAvailability checks if the username is already taken
func validateUsernameAvailability(username string) error {
	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return err
	}

	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return err
	}

	for _, u := range users {
		if u.Username == username {
			return fmt.Errorf("user %q already exists", username)
		}
	}

	return nil
}

// generateSecureUserToken creates a cryptographically secure random token
func generateSecureUserToken() (string, error) {
	randToken := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, randToken); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randToken), nil
}

// persistNewUser saves the new user to the users.json file atomically
func persistNewUser(newUser *internal.User) error {
	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		return err
	}

	users, err := internal.LoadUsersList(usersPath)
	if err != nil {
		return err
	}

	users = append(users, newUser)

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}

	return atomicWriteFile(usersPath, data, 0600)
}

// printUserCreationSuccess displays the success message with the new token
func printUserCreationSuccess(username, token string) {
	fmt.Printf("User %q created.\n", username)
	fmt.Printf("Generated token: %s\n", token)
}

func init() {
	rootCmd.AddCommand(createUserCmd)
}
