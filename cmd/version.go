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
	"simple-secrets/pkg/version"

	"github.com/spf13/cobra"
)

var showShort bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for simple-secrets including:
	• Version number
	• Git commit hash
	• Build date
	• Go version used
	• Platform information`,
	Example: `  simple-secrets version           # Show full version info
  simple-secrets version --short   # Show short version only
  simple-secrets --version         # Show full version info (flag)
  simple-secrets -v               # Show full version info (short flag)`,
	Run: func(cmd *cobra.Command, args []string) {
		if showShort {
			fmt.Println(version.Short())
			return
		}

		fmt.Println(version.BuildInfo())
	},
}

func init() {
	versionCmd.Flags().BoolVar(&showShort, "short", false, "show short version only")
	rootCmd.AddCommand(versionCmd)
}
