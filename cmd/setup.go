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
	"strings"

	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run first-time setup to create admin user and authentication token",
	Long: `Run the initial setup process for simple-secrets.

This command creates:
  • Your admin user account
  • Your authentication token
  • The secure storage directory (~/.simple-secrets/)

After setup, you can use your token with other commands or set the
SIMPLE_SECRETS_TOKEN environment variable for convenience.

Examples:
  simple-secrets setup

If you've already run setup and want to reset:
  rm -rf ~/.simple-secrets && simple-secrets setup`,
	Run: handleSetupCommand,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func handleSetupCommand(cmd *cobra.Command, args []string) {
	isActualFirstRun, err := internal.IsFirstRun()
	if err != nil {
		// Error case (broken state/protection error)
		fmt.Println("\n🔐 Welcome to simple-secrets!")
		fmt.Printf("\n❌ Setup cannot proceed: %v\n", err)
		fmt.Println("\nIf you're seeing a protection error, you may have a partial installation.")
		fmt.Println("Try running: ./simple-secrets restore-database --help")
		return
	}

	if !isActualFirstRun {
		// Existing installation (users.json exists)
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
		fmt.Println("  • Nuclear option: Back up ~/.simple-secrets/, delete it, and run setup to start fresh")
		fmt.Println("  • Or check if it's saved in ~/.simple-secrets/config.json")
		fmt.Println("  • Or check your environment: echo $SIMPLE_SECRETS_TOKEN")

		fmt.Println("\n💡 Pro tip: Set the environment variable to avoid typing --token each time:")
		fmt.Println("  export SIMPLE_SECRETS_TOKEN=<your-token>")
		return
	}

	// Clean environment, eligible for setup
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
