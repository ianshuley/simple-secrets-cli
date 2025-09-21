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
	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display configuration file documentation and examples",
	Long: `Display documentation for the config.json file and available configuration options.

The config.json file is optional and allows you to customize simple-secrets behavior.
It should be placed at ~/.simple-secrets/config.json

Available configuration options:
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, err := internal.DefaultUserConfigPath("config.json")
		if err != nil {
			return fmt.Errorf("get config path: %w", err)
		}

		fmt.Printf(`Configuration File Documentation
================================

Location: %s

The config.json file is optional and allows you to customize simple-secrets behavior.

Available Configuration Options:
---------------------------------

1. token (string, optional)
   Description: Personal access token for authentication
   Example: "token": "your-personal-access-token-here"
   Note: Not recommended for security. Use --token flag or SIMPLE_SECRETS_TOKEN env var instead.

2. rotation_backup_count (integer, optional, default: 1)
   Description: Number of backup copies to keep during master key rotation
   Example: "rotation_backup_count": 3
   Range: 1-10 (recommended)
   Note: Individual secret backups are always 1 by design. This only affects master key rotation.

Example config.json:
-------------------
{
  "rotation_backup_count": 1
}

Example with token (not recommended):
------------------------------------
{
  "token": "your-token-here",
  "rotation_backup_count": 2
}

Security Best Practices:
-----------------------
- Use environment variables or CLI flags for tokens instead of config files
- Keep config.json readable only by you (chmod 600)
- Backup your config when changing rotation settings

For more help: simple-secrets help <command>
`, configPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
