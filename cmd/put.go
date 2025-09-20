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
	"os"
	"path/filepath"
	"strings"

	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:                   "put [key] [value]",
	Short:                 "Store a secret securely.",
	Long:                  "Store a secret value under a key. Overwrites if the key exists. Backs up previous value.\n\nTo store values starting with dashes (like SSH keys), use -- to terminate flags:\nsimple-secrets put --token <token> ssh-key -- \"-----BEGIN RSA PRIVATE KEY-----\\n...\"",
	Example:               "simple-secrets put db_password s3cr3tP@ssw0rd\nsimple-secrets put --token <token> ssh-key -- \"-----BEGIN RSA PRIVATE KEY-----\\n...\"",
	Args:                  cobra.ExactArgs(2),
	DisableFlagsInUseLine: true,
	FParseErrWhitelist:    cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if token flag was explicitly set to empty string
		if flag := cmd.Flag("token"); flag != nil && flag.Changed && TokenFlag == "" {
			return fmt.Errorf("authentication required: token cannot be empty")
		}

		// RBAC: write access
		user, _, err := RBACGuard(true, TokenFlag)
		if err != nil {
			return err
		}
		if user == nil {
			return nil
		}

		// Initialize the secrets store
		store, err := internal.LoadSecretsStore()
		if err != nil {
			return err
		}

		key := args[0]
		value := args[1]

		// Security: Check for indicators of truncated input
		// Note: Unix systems truncate command arguments at null bytes before they reach Go
		// This is system-level behavior, not a bug in our application

		// Enhanced validation against suspicious patterns that could indicate truncation
		if strings.HasSuffix(key, "\x00") {
			return fmt.Errorf("key name cannot end with null bytes")
		}

		// Check for common attack patterns where truncation might be attempted
		suspiciousPatterns := []string{"admin", "root", "key", "secret", "token", "pass"}
		for _, pattern := range suspiciousPatterns {
			if key == pattern {
				// This could be a truncated key like "admin\x00reallylongname" -> "admin"
				// Ask for confirmation for potentially dangerous keys
				fmt.Fprintf(os.Stderr, "Warning: Key name '%s' could be the result of null byte truncation.\n", key)
				fmt.Fprintf(os.Stderr, "If you intended to use a longer key name, please verify the input.\n")
			}
		}

		// Validate key name
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("key name cannot be empty")
		}

		// Check for null bytes and other problematic characters
		if strings.Contains(key, "\x00") {
			return fmt.Errorf("key name cannot contain null bytes")
		}

		// Check for control characters (0x00-0x1F except \t, \n, \r)
		for _, r := range key {
			if r < 0x20 && r != 0x09 && r != 0x0A && r != 0x0D {
				return fmt.Errorf("key name cannot contain control characters")
			}
		}

		// Check for path traversal attempts
		if strings.Contains(key, "..") || strings.Contains(key, "/") || strings.Contains(key, "\\") {
			return fmt.Errorf("key name cannot contain path separators or path traversal sequences")
		}

		// Backup previous value if it exists
		prev, err := store.Get(key)
		if err == nil {
			// Only backup if key existed
			home, err := os.UserHomeDir()
			if err == nil {
				backupDir := filepath.Join(home, ".simple-secrets", "backups")
				_ = os.MkdirAll(backupDir, 0700)
				backupPath := filepath.Join(backupDir, key+".bak")
				_ = os.WriteFile(backupPath, []byte(prev), 0600)
			}
		}

		// Save secret (encrypted)
		if err := store.Put(key, value); err != nil {
			return err
		}

		fmt.Printf("Secret %q stored.\n", key)
		return nil
	},
}

var addCmd = &cobra.Command{
	Use:     "add [key] [value]",
	Short:   "Add a secret (alias for put).",
	Long:    "Store a secret with the given key and value. This is an alias for the 'put' command.",
	Example: "simple-secrets add db_password mypassword",
	Args:    cobra.ExactArgs(2),
	RunE:    putCmd.RunE, // Same implementation as put
}

func init() {
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(addCmd)
}
