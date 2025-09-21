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
	"errors"
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
	Example:               "simple-secrets put api-key \"--prod-key-abc123\"\nsimple-secrets put db_url \"postgresql://user:pass@localhost:5432/db\"",
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
	var tokenExplicitlySet bool
	filteredArgs := extractArgumentsAndFlags(args, &token, &tokenExplicitlySet)

	if shouldShowHelp(args) {
		return nil, cmd.Help()
	}

	key, value, err := validatePutArguments(filteredArgs)
	if err != nil {
		return nil, err
	}

	resolvedToken, err := determineAuthTokenWithExplicitFlag(token, tokenExplicitlySet)
	if err != nil {
		return nil, err
	}

	return &putArguments{
		key:   key,
		value: value,
		token: resolvedToken,
	}, nil
}

func extractArgumentsAndFlags(args []string, token *string, tokenExplicitlySet *bool) []string {
	filteredArgs := []string{}

	for i := 0; i < len(args); i++ {
		if isTokenFlag(args, i) {
			i = processTokenFlag(args, i, token, tokenExplicitlySet)
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	return filteredArgs
}

func isTokenFlag(args []string, position int) bool {
	return args[position] == "--token" && hasTokenValue(args, position)
}

func hasTokenValue(args []string, flagPosition int) bool {
	return flagPosition+1 < len(args)
}

func processTokenFlag(args []string, flagPosition int, token *string, tokenExplicitlySet *bool) int {
	valuePosition := flagPosition + 1
	*token = args[valuePosition]
	*tokenExplicitlySet = true
	return valuePosition // Return position of token value to skip it
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

func determineAuthTokenWithExplicitFlag(parsedToken string, wasTokenFlagUsed bool) (string, error) {
	if !wasTokenFlagUsed {
		return TokenFlag, nil
	}

	if isEmptyToken(parsedToken) {
		return "", createEmptyTokenError()
	}

	return parsedToken, nil
}

func isEmptyToken(token string) bool {
	return strings.TrimSpace(token) == ""
}

func createEmptyTokenError() error {
	return errors.New(`authentication required: token cannot be empty

Use one of these methods:
    simple-secrets --token <your-token> put <key> <value>
    SIMPLE_SECRETS_TOKEN=<your-token> simple-secrets put <key> <value>

Or save your token in ~/.simple-secrets/config.json:
    {
		"token": "<your-token>"
	}`)

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
	user, _, err := AuthenticateWithToken(true, token)
	return user, err
}

func validatePutKeyName(key string) error {
	return ValidateSecureInput(key, SecretKeyValidationConfig)
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
