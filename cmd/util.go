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

// PrintFirstRunMessage prints a clear message for first-run admin creation.
func PrintFirstRunMessage() {
	fmt.Println("\nFirst run detected. Default admin user created.")
	fmt.Println("To use your new token, re-run this command in one of these ways:")
	fmt.Println("  --token <your-token> (as a flag)")
	fmt.Println("  SIMPLE_SECRETS_TOKEN=<your-token> ./simple-secrets ... (as an env var)")
	fmt.Println("  or place it in ~/.simple-secrets/config.json as { \"token\": \"<your-token>\" }")
	fmt.Println("(The token was printed above. Store it securely; it will not be shown again.)")
	fmt.Println("\nIf creating config.json manually, ensure it has secure permissions:")
	fmt.Println("  chmod 600 ~/.simple-secrets/config.json")
}

// RBACGuardWithCmd loads users, checks first run, resolves token, and returns (user, store, error)
func RBACGuardWithCmd(needWrite bool, cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
	userStore, firstRun, err := internal.LoadUsers()
	if err != nil {
		return nil, nil, err
	}
	if firstRun {
		PrintFirstRunMessage()
		return nil, nil, nil
	}

	// Check if token flag was explicitly set to empty string
	tokenFlag := TokenFlag
	if flag := cmd.Flag("token"); flag != nil && flag.Changed && tokenFlag == "" {
		return nil, nil, fmt.Errorf("authentication required: token cannot be empty")
	}

	token, err := internal.ResolveToken(tokenFlag)
	if err != nil {
		return nil, nil, err
	}
	user, err := userStore.Lookup(token)
	if err != nil {
		return nil, nil, err
	}
	if needWrite && !user.Can("write", userStore.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'write'")
	}
	return user, userStore, nil
}

// RBACGuard is the legacy function for backward compatibility
func RBACGuard(needWrite bool, tokenFlag string) (*internal.User, *internal.UserStore, error) {
	userStore, firstRun, err := internal.LoadUsers()
	if err != nil {
		return nil, nil, err
	}
	if firstRun {
		PrintFirstRunMessage()
		return nil, nil, nil
	}

	token, err := internal.ResolveToken(tokenFlag)
	if err != nil {
		return nil, nil, err
	}
	user, err := userStore.Lookup(token)
	if err != nil {
		return nil, nil, err
	}
	if needWrite && !user.Can("write", userStore.Permissions()) {
		return nil, nil, fmt.Errorf("permission denied: need 'write'")
	}
	return user, userStore, nil
}
