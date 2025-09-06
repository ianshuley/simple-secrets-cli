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

		// RBAC: write access (admin can create users)
		user, store, err := RBACGuard(true, TokenFlag)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}
		if !user.Can("manage-users", store.Permissions()) {
			fmt.Fprintln(os.Stderr, "Error: insufficient permissions")
			return nil
		}

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

		var role internal.Role
		switch roleStr {
		case "admin":
			role = internal.RoleAdmin
		case "reader":
			role = internal.RoleReader
		default:
			return fmt.Errorf("invalid role: must be 'admin' or 'reader'")
		}

		// Always generate a secure token
		randToken := make([]byte, 20)
		if _, err := io.ReadFull(rand.Reader, randToken); err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}
		token := base64.RawURLEncoding.EncodeToString(randToken)

		tokenHash := internal.HashToken(token)

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
		now := time.Now()
		newUser := &internal.User{
			Username:       username,
			TokenHash:      tokenHash,
			Role:           role,
			TokenRotatedAt: &now,
		}
		users = append(users, newUser)
		data, err := json.MarshalIndent(users, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(usersPath, data, 0600); err != nil {
			return err
		}

		fmt.Printf("User %q created.\n", username)
		fmt.Printf("Generated token: %s\n", token)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createUserCmd)
}
