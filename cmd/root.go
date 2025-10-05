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
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	internal "simple-secrets/internal/auth"
	"simple-secrets/internal/platform"
	"simple-secrets/pkg/auth"
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

// initializePlatform sets up platform context for all CLI commands
// This runs before every command to ensure platform services are available
func initializePlatform(cmd *cobra.Command, args []string) error {
	// Skip platform initialization for version and help commands
	if cmd.Name() == "help" || cmd.Name() == "completion" {
		return nil
	}

	// Check for version flag
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		return nil
	}

	// Skip platform initialization for commands that don't need it
	if cmd.Name() == "simple-secrets" || cmd.Name() == "setup" {
		// Allow root command and setup to handle first-run scenarios
		return nil
	}

	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return fmt.Errorf("failed to initialize platform: %w", err)
	}

	// Create platform instance
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create platform: %w", err)
	}

	// Inject platform into command context
	ctx = platform.WithPlatform(ctx, app)
	cmd.SetContext(ctx)

	return nil
}

// getPlatformConfig builds platform configuration from CLI environment
func getPlatformConfig() (platform.Config, error) {
	// Get data directory
	dataDir, err := getDataDirectory()
	if err != nil {
		return platform.Config{}, fmt.Errorf("failed to get data directory: %w", err)
	}

	// Get master key
	masterKey, err := getMasterKey()
	if err != nil {
		return platform.Config{}, fmt.Errorf("failed to get master key: %w", err)
	}

	return platform.Config{
		DataDir:   dataDir,
		MasterKey: masterKey,
	}, nil
}

// getDataDirectory returns the CLI data directory
func getDataDirectory() (string, error) {
	// Check for test override
	if testDir := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR"); testDir != "" {
		return testDir, nil
	}

	// Use default ~/.simple-secrets
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".simple-secrets"), nil
}

// getMasterKey loads or creates the master key for the platform
func getMasterKey() ([]byte, error) {
	// This function provides lazy master key creation compatible with the old internal system
	dataDir, err := getDataDirectory()
	if err != nil {
		return nil, err
	}

	masterKeyPath := filepath.Join(dataDir, "master.key")

	// Check if master key file exists
	if _, err := os.Stat(masterKeyPath); os.IsNotExist(err) {
		// Check if this is a setup scenario by looking for users.json
		usersPath := filepath.Join(dataDir, "users.json")
		if _, err := os.Stat(usersPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("master key not found - run setup first")
		}

		// Setup has run but no master key exists yet - create one (lazy initialization)
		masterKey, err := generateMasterKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate master key: %w", err)
		}

		// Save master key in base64 format for compatibility with old internal system
		base64Key := base64.StdEncoding.EncodeToString(masterKey)
		if err := os.WriteFile(masterKeyPath, []byte(base64Key), 0600); err != nil {
			return nil, fmt.Errorf("failed to save master key: %w", err)
		}

		return masterKey, nil
	}

	// Read existing master key (base64 encoded)
	data, err := os.ReadFile(masterKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read master key: %w", err)
	}

	// Decode base64 to get raw key bytes
	masterKey, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	return masterKey, nil
}

// generateMasterKey creates a new 32-byte AES key and saves it in base64 format for compatibility
func generateMasterKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	return key, nil
}

func init() {
	// Set up token generator for internal package (legacy compatibility)
	internal.DefaultTokenGenerator = GenerateSecureToken

	// Persistent token flag for all commands
	rootCmd.PersistentFlags().StringVar(&TokenFlag, "token", "", "authentication token (overrides env/config)")

	// Add setup flag for manual triggering of first-run experience
	rootCmd.Flags().Bool("setup", false, "run first-time setup (use after removing ~/.simple-secrets for reset)")

	// Add standard version flags that users expect
	rootCmd.Flags().BoolP("version", "v", false, "show version information")

	// Set up platform context injection for all commands
	rootCmd.PersistentPreRunE = initializePlatform
}

// getPlatformFromCommand is a helper that initializes the platform services
func getPlatformFromCommand(cmd *cobra.Command) (*platform.Platform, error) {
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return platform.New(ctx, config)
}

// authenticateWithPlatform is a helper that handles authentication and authorization
func authenticateWithPlatform(cmd *cobra.Command, needWrite bool) (*auth.UserContext, error) {
	app, err := getPlatformFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Resolve token for authentication
	authToken, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	ctx := context.Background()
	user, err := app.Auth.Authenticate(ctx, authToken)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Check permissions
	permission := auth.PermissionRead
	if needWrite {
		permission = auth.PermissionWrite
	}

	err = app.Auth.Authorize(ctx, user, permission)
	if err != nil {
		return nil, fmt.Errorf("access denied: %w", err)
	}

	return user, nil
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
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err // No auth = no completion
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return nil, err
	}

	// List secrets for completion
	metadata, err := app.Secrets.List(ctx)
	if err != nil {
		return nil, err
	}

	// Extract keys from metadata
	keys := make([]string, len(metadata))
	for i, meta := range metadata {
		keys[i] = meta.Key
	}

	return keys, nil
}

// getAvailableDisabledSecrets retrieves all disabled secret keys for completion
func getAvailableDisabledSecrets(cmd *cobra.Command) ([]string, error) {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err // No auth = no completion
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return nil, err
	}

	// List disabled secrets for completion
	metadata, err := app.Secrets.ListDisabled(ctx)
	if err != nil {
		return nil, err
	}

	// Extract keys from metadata
	keys := make([]string, len(metadata))
	for i, meta := range metadata {
		keys[i] = meta.Key
	}

	return keys, nil
}

// getAvailableBackupSecrets retrieves all secret keys that have backups for completion
func getAvailableBackupSecrets(cmd *cobra.Command) ([]string, error) {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err // No auth = no completion
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return nil, err
	}

	// For completion of secrets with backups, we can't easily determine which
	// individual secrets have backups without reading backup contents.
	// For now, return all secrets as potentially having backups.
	metadata, err := app.Secrets.List(ctx)
	if err != nil {
		return nil, err
	}

	// Extract keys from metadata
	keys := make([]string, len(metadata))
	for i, meta := range metadata {
		keys[i] = meta.Key
	}

	return keys, nil
}

// getAvailableBackupNames retrieves all database backup names for completion
func getAvailableBackupNames(cmd *cobra.Command) ([]string, error) {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err // No auth = no completion
	}

	// Check read permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionRead)
	if err != nil {
		return nil, err
	}

	// List backup names for completion
	backups, err := app.Rotation.ListBackups(ctx)
	if err != nil {
		return nil, err
	}

	// Extract backup names
	names := make([]string, len(backups))
	for i, backup := range backups {
		names[i] = backup.Name
	}

	return names, nil
}

// getAvailableUsernames retrieves all available usernames for completion
func getAvailableUsernames(cmd *cobra.Command) ([]string, error) {
	// Get platform configuration
	config, err := getPlatformConfig()
	if err != nil {
		return nil, err
	}

	// Initialize platform services
	ctx := context.Background()
	app, err := platform.New(ctx, config)
	if err != nil {
		return nil, err
	}

	// Try to resolve token for authentication
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Authenticate user
	user, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err // No auth = no completion
	}

	// Check user management permissions
	err = app.Auth.Authorize(ctx, user, auth.PermissionManageUsers)
	if err != nil {
		return nil, err
	}

	// List users for completion
	users, err := app.Users.List(ctx)
	if err != nil {
		return nil, err
	}

	// Extract usernames
	usernames := make([]string, len(users))
	for i, u := range users {
		usernames[i] = u.Username
	}

	return usernames, nil
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
