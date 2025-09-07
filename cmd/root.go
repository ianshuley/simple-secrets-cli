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
	"os"

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
	• Token-based authentication (flag, env, or config file)

All secrets are encrypted and stored locally in ~/.simple-secrets/.

See 'simple-secrets --help' or the README for more info.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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
}
