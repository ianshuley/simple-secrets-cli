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
	Long:                  "Store a secret value under a key. Overwrites if the key exists. Backs up previous value.\n\nUse quotes for values with spaces or special characters.",
	Example:               "simple-secrets put db_password s3cr3tP@ssw0rd\nsimple-secrets put db_url \"postgresql://user:pass@localhost:5432/db\"",
	DisableFlagsInUseLine: true,
	DisableFlagParsing:    true,
	RunE: func(cmd *cobra.Command, args []string) error {
		parsedArgs, err := parsePutArguments(cmd, args)
		if err != nil {
			return err
		}

		if parsedArgs == nil {
			return nil // Help was shown
		}

		return executePutCommand(parsedArgs)
	},
}

type putArguments struct {
	key   string
	value string
	token string
}

func parsePutArguments(cmd *cobra.Command, args []string) (*putArguments, error) {
	var token string
	filteredArgs := extractArgumentsAndFlags(args, &token)

	if shouldShowHelp(args) {
		return nil, cmd.Help()
	}

	key, value, err := validatePutArguments(filteredArgs)
	if err != nil {
		return nil, err
	}

	return &putArguments{
		key:   key,
		value: value,
		token: determineAuthToken(token),
	}, nil
}

func extractArgumentsAndFlags(args []string, token *string) []string {
	filteredArgs := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--token" && i+1 < len(args) {
			*token = args[i+1]
			i++ // skip the token value
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	return filteredArgs
}

func shouldShowHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func validatePutArguments(filteredArgs []string) (string, string, error) {
	if len(filteredArgs) != 2 {
		return "", "", fmt.Errorf("requires exactly 2 arguments [key] [value], got %d", len(filteredArgs))
	}
	return filteredArgs[0], filteredArgs[1], nil
}

func determineAuthToken(parsedToken string) string {
	if parsedToken != "" {
		return parsedToken
	}
	return TokenFlag
}

func executePutCommand(args *putArguments) error {
	user, err := authenticatePutUser(args.token)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	if err := validatePutKeyName(args.key); err != nil {
		return err
	}

	store, err := internal.LoadSecretsStore()
	if err != nil {
		return err
	}

	backupExistingSecret(store, args.key)

	if err := store.Put(args.key, args.value); err != nil {
		return err
	}

	fmt.Printf("Secret %q stored.\n", args.key)
	return nil
}

func authenticatePutUser(token string) (*internal.User, error) {
	user, _, err := RBACGuard(true, token)
	return user, err
}

func validatePutKeyName(key string) error {
	if err := checkForNullByteIssues(key); err != nil {
		return err
	}

	if err := validateKeyBasicRules(key); err != nil {
		return err
	}

	return validateKeySecurityRules(key)
}

func checkForNullByteIssues(key string) error {
	if strings.HasSuffix(key, "\x00") {
		return fmt.Errorf("key name cannot end with null bytes")
	}

	if strings.Contains(key, "\x00") {
		return fmt.Errorf("key name cannot contain null bytes")
	}

	warnAboutSuspiciousKeys(key)
	return nil
}

func warnAboutSuspiciousKeys(key string) {
	suspiciousPatterns := []string{"admin", "root", "key", "secret", "token", "pass"}
	for _, pattern := range suspiciousPatterns {
		if key == pattern {
			fmt.Fprintf(os.Stderr, "Warning: Key name '%s' could be the result of null byte truncation.\n", key)
			fmt.Fprintf(os.Stderr, "If you intended to use a longer key name, please verify the input.\n")
			return
		}
	}
}

func validateKeyBasicRules(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("key name cannot be empty")
	}
	return nil
}

func validateKeySecurityRules(key string) error {
	if err := checkForControlCharacters(key); err != nil {
		return err
	}

	return checkForPathTraversal(key)
}

func checkForControlCharacters(key string) error {
	for _, r := range key {
		if r < 0x20 && r != 0x09 && r != 0x0A && r != 0x0D {
			return fmt.Errorf("key name cannot contain control characters")
		}
	}
	return nil
}

func checkForPathTraversal(key string) error {
	if strings.Contains(key, "..") || strings.Contains(key, "/") || strings.Contains(key, "\\") {
		return fmt.Errorf("key name cannot contain path separators or path traversal sequences")
	}
	return nil
}

func backupExistingSecret(store *internal.SecretsStore, key string) {
	prev, err := store.Get(key)
	if err != nil {
		return // No existing value to backup
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return // Cannot determine backup location
	}

	backupDir := filepath.Join(home, ".simple-secrets", "backups")
	_ = os.MkdirAll(backupDir, 0700)
	backupPath := filepath.Join(backupDir, key+".bak")
	_ = os.WriteFile(backupPath, []byte(prev), 0600)
}

var addCmd = &cobra.Command{
	Use:                   "add [key] [value]",
	Short:                 "Add a secret (alias for put).",
	Long:                  "Store a secret with the given key and value. This is an alias for the 'put' command.\n\nUse quotes for values with spaces or special characters.",
	Example:               "simple-secrets add db_password mypassword\nsimple-secrets add db_url \"postgresql://user:pass@localhost:5432/db\"",
	DisableFlagsInUseLine: true,
	DisableFlagParsing:    true,
	RunE:                  putCmd.RunE, // Same implementation as put
}

func init() {
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(addCmd)
}
