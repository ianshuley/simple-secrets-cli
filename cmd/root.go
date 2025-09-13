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

🚀 New to simple-secrets? Run without arguments for an interactive setup walkthrough,
   or use --setup to trigger it anytime.

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
	// Persistent token flag for all commands
	rootCmd.PersistentFlags().StringVar(&TokenFlag, "token", "", "authentication token (overrides env/config)")

	// Add setup flag for manual triggering of first-run experience
	rootCmd.Flags().Bool("setup", false, "trigger the interactive setup walkthrough")

	// Add standard version flags that users expect
	rootCmd.Flags().BoolP("version", "v", false, "show version information")
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
		// Manual setup requested - let the walkthrough determine the state
		runFirstRunWalkthrough(false) // false = not necessarily a first run
		return
	}

	// Check if this is a fresh install
	if isFirstRun() {
		runFirstRunWalkthrough(true) // true = this is a first run
		return
	}

	// Not a first run, show regular help
	cmd.Help()
}

// isFirstRun checks if this is a fresh installation with no setup done
// This is a read-only check that doesn't trigger any setup
func isFirstRun() bool {
	eligible, err := internal.IsFirstRunEligible()
	if err != nil {
		return false // If there's an error (like protection), it's not a first run
	}
	return eligible
}

// runFirstRunWalkthrough provides an interactive guide for new users
func runFirstRunWalkthrough(knownFirstRun bool) {
	var isActualFirstRun bool
	var err error

	if knownFirstRun {
		// We already know this is a first run, don't call LoadUsers yet
		isActualFirstRun = true
	} else {
		// Check if this is actually a first run without triggering setup
		var err error
		isActualFirstRun, err = internal.IsFirstRunEligible()

		if err != nil {
			// Handle error case (like first-run protection)
			fmt.Println("\n🔐 Welcome to simple-secrets!")
			fmt.Printf("\n❌ Setup cannot proceed: %v\n", err)
			fmt.Println("\nIf you're seeing a protection error, you may have a partial installation.")
			fmt.Println("Try running: ./simple-secrets restore-database --help")
			return
		}
	}

	if !isActualFirstRun {
		// This is an existing installation, show different message
		fmt.Println("\n🔐 simple-secrets Setup Walkthrough")
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

		fmt.Println("\n� Need your token? If you've lost it, you can:")
		fmt.Println("  • Create a new user with: ./simple-secrets create-user <username> <role>")
		fmt.Println("  • Or rotate an existing token: ./simple-secrets rotate token <username>")
		fmt.Println("  • Nuclear option: Back up ~/.simple-secrets/, delete it, and start fresh")

		fmt.Println("\n💡 Pro tip: Set the environment variable to avoid typing --token each time:")
		fmt.Println("  export SIMPLE_SECRETS_TOKEN=<your-token>")
		return
	}

	// This is a true first run
	fmt.Println("\n�🔐 Welcome to simple-secrets!")
	fmt.Println("\nIt looks like this is your first time using simple-secrets.")
	fmt.Println("Let me help you get started with a quick setup.")

	fmt.Println("\n📋 What simple-secrets does:")
	fmt.Println("  • Securely stores your secrets with AES-256-GCM encryption")
	fmt.Println("  • Provides token-based authentication for secure access")
	fmt.Println("  • Supports role-based permissions (admin/reader)")
	fmt.Println("  • Stores everything locally in ~/.simple-secrets/")

	fmt.Println("\n🚀 Ready to get started? [y/N]")

	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" && response != "yes" && response != "YES" {
		fmt.Println("\nNo problem! You can run this setup anytime with:")
		fmt.Println("  ./simple-secrets --setup")
		fmt.Println("\nOr trigger setup by running any command that requires authentication:")
		fmt.Println("  ./simple-secrets list keys")
		fmt.Println("  ./simple-secrets put mykey myvalue")
		fmt.Println("(These commands will first set up simple-secrets, then you can re-run them with your token)")
		return
	}

	fmt.Println("\n⚡ Initializing simple-secrets...")
	fmt.Println("\nI'll create your first admin user now. This will generate a secure token")
	fmt.Println("that you'll use to authenticate with simple-secrets.")

	// Use the clean first-run setup function that returns the token
	userStore, token, err := internal.PerformFirstRunSetupWithToken()
	if err != nil {
		fmt.Printf("\n❌ Setup failed: %v\n", err)
		return
	}

	fmt.Println("\n🎉 Setup complete!")

	fmt.Println("\n📖 Next steps:")
	fmt.Println("  1. Save your token somewhere secure (password manager recommended)")
	fmt.Println("  2. Try storing your first secret:")
	fmt.Println("     ./simple-secrets put --token <your-token> mykey myvalue")
	fmt.Println("  3. Retrieve it back:")
	fmt.Println("     ./simple-secrets get --token <your-token> mykey")
	fmt.Println("  4. List all your secrets:")
	fmt.Println("     ./simple-secrets list --token <your-token> keys")

	fmt.Println("\n💡 Pro tip: Set the environment variable to avoid typing --token each time:")
	fmt.Println("  export SIMPLE_SECRETS_TOKEN=<your-token>")

	// Show user count to confirm setup
	users := userStore.Users()
	fmt.Printf("\n📊 Current users: %d (you are the admin)\n", len(users))

	// Display the token prominently at the end
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🔑 YOUR AUTHENTICATION TOKEN")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("   %s\n", token)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("⚠️  IMPORTANT: Copy this token now - it will not be shown again!")
	fmt.Println("🔒 Store it securely in your password manager or config file.")
	fmt.Println("")
	fmt.Println("To use it:")
	fmt.Println("  • Flag: --token " + token)
	fmt.Println("  • Environment: export SIMPLE_SECRETS_TOKEN=" + token)
	fmt.Println("  • Config file: ~/.simple-secrets/config.json")
	fmt.Println(strings.Repeat("=", 60))
}
