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
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"simple-secrets/internal"
)

var putCmd = &cobra.Command{
	Use:                   "put [key] [value]",
	Short:                 "Store a secret securely.",
	Long:                  "Store a secret value under a key. Overwrites if the key exists. Backs up previous value.\n\nUse quotes for values with spaces or special characters.\n\n⚠️  SECURITY: Use single quotes to prevent shell command execution:\n    ✅ SAFE:      simple-secrets put key 'value with $(command)'\n    ❌ DANGEROUS: simple-secrets put key \"value with $(command)\"\n\nDouble quotes allow shell command substitution which executes before the app runs.\n\nUse --generate to automatically create a cryptographically secure secret:\n    simple-secrets put api-key --generate\n    simple-secrets put api-key --generate --length 64",
	Example:               "simple-secrets put api-key '--prod-key-abc123'\nsimple-secrets put db_url 'postgresql://user:pass@localhost:5432/db'\nsimple-secrets put script 'echo $(whoami)'  # Stores literally, not executed\n\n# Generate secure secrets automatically\nsimple-secrets put api-key --generate\nsimple-secrets put api-key -g --length 64",
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
	key      string
	value    string
	token    string
	generate bool
	length   int
}

// generateSecretValue creates a cryptographically secure random secret
func generateSecretValue(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	// Character set: A-Z, a-z, 0-9, and URL-safe symbols
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+"

	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i, b := range randomBytes {
		result[i] = charset[int(b)%len(charset)]
	}

	return string(result), nil
}

func parsePutArguments(cmd *cobra.Command, args []string) (*putArguments, error) {
	var token string
	var tokenExplicitlySet bool
	var generate bool
	var length int = 32 // Default length

	filteredArgs := extractArgumentsAndFlags(args, &token, &tokenExplicitlySet, &generate, &length)

	if shouldShowHelp(args) {
		return nil, cmd.Help()
	}

	key, value, err := validatePutArguments(filteredArgs, generate)
	if err != nil {
		return nil, err
	}

	resolvedToken, err := determineAuthTokenWithExplicitFlag(token, tokenExplicitlySet)
	if err != nil {
		return nil, err
	}

	return &putArguments{
		key:      key,
		value:    value,
		token:    resolvedToken,
		generate: generate,
		length:   length,
	}, nil
}

func extractArgumentsAndFlags(args []string, token *string, tokenExplicitlySet *bool, generate *bool, length *int) []string {
	filteredArgs := []string{}

	for i := 0; i < len(args); i++ {
		if isTokenFlag(args, i) {
			i = processTokenFlag(args, i, token, tokenExplicitlySet)
			continue
		}
		if isGenerateFlag(args[i]) {
			*generate = true
			continue
		}
		if isLengthFlag(args, i) {
			i = processLengthFlag(args, i, length)
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

func isGenerateFlag(arg string) bool {
	return arg == "--generate" || arg == "-g"
}

func isLengthFlag(args []string, position int) bool {
	return args[position] == "--length" && hasLengthValue(args, position)
}

func hasLengthValue(args []string, flagPosition int) bool {
	return flagPosition+1 < len(args)
}

func processLengthFlag(args []string, flagPosition int, length *int) int {
	valuePosition := flagPosition + 1
	lengthStr := args[valuePosition]

	parsedLength := parsePositiveInteger(lengthStr)
	if parsedLength <= 0 {
		*length = 32 // Default for invalid values
		return valuePosition
	}

	*length = parsedLength
	return valuePosition
}

func parsePositiveInteger(value string) int {
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return 0
	}
	return result
}

func shouldShowHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func validatePutArguments(filteredArgs []string, generate bool) (string, string, error) {
	if generate {
		// With --generate, we need exactly 1 argument (key only)
		if len(filteredArgs) == 0 {
			return "", "", fmt.Errorf("requires key argument when using --generate flag")
		}
		if len(filteredArgs) > 1 {
			return "", "", fmt.Errorf("cannot provide both --generate flag and manual value")
		}
		return filteredArgs[0], "", nil // value will be generated later
	}

	// Without --generate, we need exactly 2 arguments (key and value)
	if len(filteredArgs) != 2 {
		return "", "", fmt.Errorf("requires exactly 2 arguments [key] [value], got %d", len(filteredArgs))
	}
	return filteredArgs[0], filteredArgs[1], nil
}

func determineAuthTokenWithExplicitFlag(parsedToken string, wasTokenFlagUsed bool) (string, error) {
	if !wasTokenFlagUsed {
		return internal.ResolveToken("")
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
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return err
	}

	// Use direct token authentication (put handles token parsing manually)
	user, _, err := helper.AuthenticateToken(args.token, true)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	if err := validatePutKeyName(args.key); err != nil {
		return err
	}

	// Generate secret value if needed
	value := args.value
	if args.generate {
		generatedValue, err := generateSecretValue(args.length)
		if err != nil {
			return fmt.Errorf("failed to generate secret: %w", err)
		}
		value = generatedValue
	}

	service := helper.GetService()

	// Backup is handled automatically by the service layer

	err = service.Secrets().Put(args.token, args.key, value)
	if err != nil {
		return err
	}

	fmt.Printf("Secret %q stored.\n", args.key)

	// Print generated value to stdout if it was generated
	if args.generate {
		fmt.Println(value)
	}

	return nil
}

// validatePutKeyName ensures secret keys meet security and usability requirements:
// - Non-empty
// - No control characters (except tab, LF, CR)
// - No path traversal sequences
// - No shell metacharacters
func validatePutKeyName(key string) error {
	return ValidateSecureInput(key, SecretKeyValidationConfig)
}

var addCmd = &cobra.Command{
	Use:                   "add [key] [value]",
	Short:                 "Add a secret (alias for put).",
	Long:                  "Store a secret with the given key and value. This is an alias for the 'put' command.\n\nUse quotes for values with spaces or special characters.\n\n⚠️  SECURITY: Use single quotes to prevent shell command execution:\n    ✅ SAFE:      simple-secrets add key 'value with $(command)'\n    ❌ DANGEROUS: simple-secrets add key \"value with $(command)\"\n\nDouble quotes allow shell command substitution which executes before the app runs.",
	Example:               "simple-secrets add db_password mypassword\nsimple-secrets add db_url \"postgresql://user:pass@localhost:5432/db\"",
	DisableFlagsInUseLine: true,
	DisableFlagParsing:    true,
	RunE:                  putCmd.RunE, // Same implementation as put
}

func init() {
	rootCmd.AddCommand(putCmd)
	rootCmd.AddCommand(addCmd)
}
