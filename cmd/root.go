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
	"simple-secrets/internal"
	"simple-secrets/pkg/version"
	"strings"

	"github.com/spf13/cobra"
)

var TokenFlag string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "simple-secrets",
	Short: "A secure, minimal secrets manager for automation and GitOps workflows.",
	Long: `simple-secrets is a lightweight secrets manager for securely storing, retrieving, and rotating secrets.

Features:
	• AES-256-GCM encryption for all secrets
	• Master key rotation with automatic backup cleanup
	• Database backup/restore from rotation snapshots
	• Role-based access control (RBAC) for users (admin/reader)
	• CLI user management (create-user, list users, token rotation)
	• Self-service token rotation for enhanced security
	• Individual secret backup/restore functionality
	• Secret lifecycle management (disable/enable secrets)
	• Token disable/enable for security management
	• Token-based authentication (flag, env, or config file)

All secrets are encrypted and stored locally in ~/.simple-secrets/.

See 'simple-secrets --help' or the README for more info.`,
	Run: handleRootCommand,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set up token generator for internal package
	internal.DefaultTokenGenerator = GenerateSecureToken

	// Persistent token flag for all commands
	rootCmd.PersistentFlags().StringVar(&TokenFlag, "token", "", "authentication token (overrides env/config)")

	// Add setup flag for manual triggering of first-run experience
	rootCmd.Flags().Bool("setup", false, "run first-time setup (use after removing ~/.simple-secrets for reset)")

	// Add standard version flags that users expect
	rootCmd.Flags().BoolP("version", "v", false, "show version information")
}

// completeSecretNames provides completion for secret key names
func completeSecretNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get the available secret keys for completion
	keys, err := getAvailableSecretKeys(cmd)
	if err != nil {
		// If we can't get keys (e.g., no auth), return no completion
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return keys, cobra.ShellCompDirectiveNoFileComp
}

// getAvailableSecretKeys retrieves all available secret keys for completion
func getAvailableSecretKeys(cmd *cobra.Command) ([]string, error) {
	// Get CLI service helper
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Resolve the token (CLI responsibility)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return nil, err
	}

	// List secrets using focused service operations
	keys, err := helper.GetService().Secrets().List(resolvedToken)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// getAvailableDisabledSecrets retrieves all disabled secret keys for completion
func getAvailableDisabledSecrets(cmd *cobra.Command) ([]string, error) {
	// Get CLI service helper
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, err
	}

	// For disabled secrets, we need to access the store directly
	user, _, err := helper.AuthenticateCommand(cmd, false)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("authentication required")
	}

	store, err := internal.LoadSecretsStore(internal.NewFilesystemBackend())
	if err != nil {
		return nil, err
	}

	disabledSecrets := store.ListDisabledSecrets()
	return disabledSecrets, nil
}

// getAvailableBackupSecrets retrieves all secret keys that have backups for completion
func getAvailableBackupSecrets(cmd *cobra.Command) ([]string, error) {
	// Get CLI service helper
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, err
	}

	// Authenticate to access backups
	user, _, err := helper.AuthenticateCommand(cmd, false)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("authentication required")
	}

	store, err := internal.LoadSecretsStore(internal.NewFilesystemBackend())
	if err != nil {
		return nil, err
	}

	// Get all secret keys and check which have backups
	secretKeys := store.ListKeys()
	var backedUpSecrets []string

	for _, key := range secretKeys {
		backupPath := store.GetBackupPath(key)
		if _, err := os.Stat(backupPath); err == nil {
			backedUpSecrets = append(backedUpSecrets, key)
		}
	}

	return backedUpSecrets, nil
}

// getAvailableBackupNames retrieves all database backup names for completion
func getAvailableBackupNames(cmd *cobra.Command) ([]string, error) {
	// Get CLI service helper
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, err
	}

	// Authenticate to access backups
	user, _, err := helper.AuthenticateCommand(cmd, false)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("authentication required")
	}

	store, err := internal.LoadSecretsStore(internal.NewFilesystemBackend())
	if err != nil {
		return nil, err
	}

	backups, err := store.ListRotationBackups()
	if err != nil {
		return nil, err
	}

	var backupNames []string
	for _, backup := range backups {
		backupNames = append(backupNames, backup.Name)
	}

	return backupNames, nil
}

// handleRootCommand is called when simple-secrets is run without any subcommands
func handleRootCommand(cmd *cobra.Command, args []string) {
	versionFlag, _ := cmd.Flags().GetBool("version")
	setupFlag, _ := cmd.Flags().GetBool("setup")

	// Handle version flag first (highest priority)
	if versionFlag {
		fmt.Println(version.BuildInfo())
		return
	}

	if setupFlag {
		runExplicitSetup()
		return
	}

	// Check if this is a fresh install and offer automatic setup
	if needsInitialization() {
		offerAutomaticSetup()
		return
	}

	// Normal case: show help
	cmd.Help()
}

// needsInitialization checks if this is a fresh installation
func needsInitialization() bool {
	eligible, err := internal.IsFirstRun()
	if err != nil {
		return false // If there's an error (like protection), it's not a fresh install
	}
	return eligible
}

// runExplicitSetup handles the --setup flag (user explicitly wants to set up)
func runExplicitSetup() {
	isActualFirstRun, err := internal.IsFirstRun()
	if err != nil {
		// Error case (broken state/protection error)
		displayFirstRunProtectionError(err)
		return
	}

	if !isActualFirstRun {
		// Existing installation (users.json exists)
		displayExistingInstallationInfo()
		return
	}

	// Clean environment, eligible for setup
	performFirstTimeSetup()
}

// offerAutomaticSetup handles when we detect a fresh installation automatically
func offerAutomaticSetup() {
	fmt.Println("\n🔐 Welcome to simple-secrets!")
	fmt.Println("\nFirst time setup detected.")
	fmt.Println("This will create your admin user and authentication token.")
	fmt.Println("\nAlternatively, you can run: ./simple-secrets --setup")
	fmt.Println()

	performFirstTimeSetup()
}

// displayFirstRunProtectionError shows protection error with helpful guidance
func displayFirstRunProtectionError(err error) {
	fmt.Println("\n🔐 Welcome to simple-secrets!")
	fmt.Printf("\n❌ Setup cannot proceed: %v\n", err)
	fmt.Println("\nIf you're seeing a protection error, you may have a partial installation.")
	fmt.Println("Try running: ./simple-secrets restore-database --help")
}

// displayExistingInstallationInfo shows info for existing installations
func displayExistingInstallationInfo() {
	fmt.Println("\n🔐 simple-secrets Setup")
	fmt.Println("\n✅ You already have simple-secrets set up!")
	fmt.Println("\n📋 What simple-secrets does:")
	fmt.Println("  • Securely stores your secrets with AES-256-GCM encryption")
	fmt.Println("  • Provides token-based authentication for secure access")
	fmt.Println("  • Supports role-based permissions (admin/reader)")
	fmt.Println("  • Stores everything locally in ~/.simple-secrets/")

	fmt.Println("\n💡 Quick Reference:")
	fmt.Println("  • Store a secret:     ./simple-secrets put --token <token> key value")
	fmt.Println("  • Retrieve a secret:  ./simple-secrets get --token <token> key")
	fmt.Println("  • List secrets:       ./simple-secrets list --token <token> keys")
	fmt.Println("  • Create new user:    ./simple-secrets create-user --token <token> username role")
	fmt.Println("  • List users:         ./simple-secrets list --token <token> users")

	fmt.Println("\n🔑 Need your token? If you've lost it:")
	fmt.Println("  • Nuclear option: Back up ~/.simple-secrets/, delete it, and run --setup to start fresh")
	fmt.Println("  • Or check if it's saved in ~/.simple-secrets/config.json")
	fmt.Println("  • Or check your environment: echo $SIMPLE_SECRETS_TOKEN")

	fmt.Println("\n💡 Pro tip: Set the environment variable to avoid typing --token each time:")
	fmt.Println("  export SIMPLE_SECRETS_TOKEN=<your-token>")
}

// performFirstTimeSetup handles the actual first-time setup process
func performFirstTimeSetup() {
	fmt.Println("\n🔐 Welcome to simple-secrets!")
	fmt.Println("\nSimple-secrets setup")
	fmt.Println("Creating admin user and generating authentication token.")

	fmt.Println("\nCreating admin user...")

	// Use the consolidated first-run setup function
	usersPath, err := internal.DefaultUserConfigPath("users.json")
	if err != nil {
		fmt.Printf("\n❌ Setup failed: %v\n", err)
		return
	}
	rolesPath, err := internal.DefaultUserConfigPath("roles.json")
	if err != nil {
		fmt.Printf("\n❌ Setup failed: %v\n", err)
		return
	}

	_, token, err := internal.HandleFirstRunSetup(usersPath, rolesPath)
	if err != nil {
		fmt.Printf("\n❌ Setup failed: %v\n", err)
		return
	}

	fmt.Println("Setup complete.")

	fmt.Println("\nUsage:")
	fmt.Println("  ./simple-secrets put --token <TOKEN> key value")
	fmt.Println("  ./simple-secrets get --token <TOKEN> key")
	fmt.Println("  ./simple-secrets list --token <TOKEN> keys")

	fmt.Println("\nOr set environment variable:")
	fmt.Println("  export SIMPLE_SECRETS_TOKEN=<TOKEN>")

	// Display the token clearly
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("TOKEN: %s\n", token)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("Save this token securely. It will not be shown again.")
}
