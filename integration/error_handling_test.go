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
package main

import (
	"testing"

	"simple-secrets/integration/testing_framework"
)

func TestErrorHandling(t *testing.T) {
	// Create isolated test environment
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	t.Run("InvalidToken", func(t *testing.T) {
		// Test with an invalid token using raw command without admin token
		output, err := env.RunRawCommand([]string{"list", "keys", "--token", "invalid"}, env.CleanEnvironment(), "")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("invalid token")
	})

	t.Run("NoTokenProvided", func(t *testing.T) {
		// Test without any token - should fail with authentication error
		output, err := env.CLI().RawWithoutToken("list", "keys")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("authentication required: no token found")
	})

	t.Run("ListInvalidSubcommand", func(t *testing.T) {
		// Test invalid list subcommand
		output, err := env.RunRawCommand([]string{"list", "invalid", "--token", env.AdminToken()}, env.CleanEnvironment(), "")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("unknown list type")
	})

	t.Run("GetNonexistentSecret", func(t *testing.T) {
		// Test getting a secret that doesn't exist
		output, err := env.CLI().Get("nonexistent")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("secret not found")
	})

	t.Run("DeleteNonexistentSecret", func(t *testing.T) {
		// Test deleting a secret that doesn't exist
		output, err := env.CLI().Delete("nonexistent")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("file does not exist")
	})

	t.Run("PutEmptyKey", func(t *testing.T) {
		// Test putting with empty key
		output, err := env.RunRawCommand([]string{"put", "", "value", "--token", env.AdminToken()}, env.CleanEnvironment(), "")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("key name cannot be empty")
	})

	t.Run("InvalidCommand", func(t *testing.T) {
		// Test completely invalid command
		output, err := env.RunRawCommand([]string{"invalid-command", "--token", env.AdminToken()}, env.CleanEnvironment(), "")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("unknown command")
	})

	t.Run("InsufficientPermissions", func(t *testing.T) {
		// Create a reader user and test write operations
		userOutput, err := env.CLI().Users().Create("reader_user", "reader")
		testing_framework.Assert(t, userOutput, err).Success()

		// Extract the reader token
		readerToken := testing_framework.ParseToken(string(userOutput))
		if readerToken == "" {
			t.Fatalf("Could not extract reader token from output: %s", string(userOutput))
		}

		// Test that reader cannot write secrets
		output, err := env.RunRawCommand([]string{"put", "test", "value", "--token", readerToken}, env.CleanEnvironment(), "")
		testing_framework.Assert(t, output, err).
			Failure().
			Contains("permission denied")
	})
}
