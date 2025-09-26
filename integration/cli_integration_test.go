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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"simple-secrets/integration/testing_framework"
)

func TestFirstRunCreatesAdmin(t *testing.T) {
	// Create isolated test environment with automatic first-run setup
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Verify first-run setup created required files
	usersFile := filepath.Join(env.TempDir(), ".simple-secrets", "users.json")
	rolesFile := filepath.Join(env.TempDir(), ".simple-secrets", "roles.json")

	if _, err := os.Stat(usersFile); err != nil {
		t.Fatalf("users.json not created: %v", err)
	}
	if _, err := os.Stat(rolesFile); err != nil {
		t.Fatalf("roles.json not created: %v", err)
	}

	// Verify we can use the CLI with the admin token
	output, err := env.CLI().List().Keys()
	testing_framework.Assert(t, output, err).Success()
}

func TestPutGetListDelete(t *testing.T) {
	// Create isolated test environment
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Put a secret
	output, err := env.CLI().Put("foo", "bar")
	testing_framework.Assert(t, output, err).
		Success().
		Contains(`Secret "foo" stored`)

	// Get the secret back
	output, err = env.CLI().Get("foo")
	testing_framework.Assert(t, output, err).
		Success().
		Equals("bar")

	// List should show the secret
	output, err = env.CLI().List().Keys()
	testing_framework.Assert(t, output, err).
		Success().
		Contains("foo")

	// Delete the secret
	output, err = env.CLI().Delete("foo")
	testing_framework.Assert(t, output, err).Success()

	// Verify deletion - get should fail
	output, err = env.CLI().Get("foo")
	testing_framework.Assert(t, output, err).Failure()
}

func TestCreateUserAndLogin(t *testing.T) {
	// Create isolated test environment
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Create a new user
	output, err := env.CLI().Users().Create("testuser", "reader")
	testing_framework.Assert(t, output, err).
		Success().
		ValidToken()

	// Extract the new user's token
	userToken := testing_framework.ParseToken(string(output))

	// Verify the user can use their token for read operations
	// (We'll need to add a WithToken method to CLIRunner for this)
	out, err := env.RunRawCommand([]string{"list", "keys"},
		append(env.CleanEnvironment(), "SIMPLE_SECRETS_TOKEN="+userToken), "")
	testing_framework.Assert(t, out, err).Success()

	// Verify the user cannot write (should fail)
	out, err = env.RunRawCommand([]string{"put", "test", "value"},
		append(env.CleanEnvironment(), "SIMPLE_SECRETS_TOKEN="+userToken), "")
	testing_framework.Assert(t, out, err).Failure()
}

func TestMain(m *testing.M) {
	// Build the binary for CLI integration tests
	cmd := exec.Command("go", "build", "-o", "simple-secrets")
	cmd.Dir = ".." // backend directory
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to build CLI binary: %v", err)
	}
	os.Exit(m.Run())
}
