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
	"strings"
	"testing"
)

// Path to the CLI binary (adjust if needed)
const cliBin = "../simple-secrets"

func TestFirstRunCreatesAdmin(t *testing.T) {
	// Create isolated test helper
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	// Should create .simple-secrets/users.json and roles.json
	if _, err := os.Stat(filepath.Join(helper.GetTempDir(), ".simple-secrets", "users.json")); err != nil {
		t.Fatalf("users.json not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(helper.GetTempDir(), ".simple-secrets", "roles.json")); err != nil {
		t.Fatalf("roles.json not created: %v", err)
	}

	// Verify we can use the token to list keys
	if _, err := helper.RunCommand("list", "keys"); err != nil {
		t.Fatalf("list keys with token failed: %v", err)
	}
}

func TestPutGetListDelete(t *testing.T) {
	tmp := t.TempDir()
	// First run: trigger creation and capture token
	cmd := exec.Command(cliBin, "setup")
	cmd.Env = testEnv(tmp)
	cmd.Stdin = strings.NewReader("Y\n")
	out, err := cmd.CombinedOutput()
	// Extract token from output
	token := ExtractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Put
	cmd = exec.Command(cliBin, "put", "foo", "bar", "--token", token)
	cmd.Env = testEnv(tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("put failed: %v\n%s", err, out)
	}

	// Get
	cmd = exec.Command(cliBin, "get", "foo", "--token", token)
	cmd.Env = testEnv(tmp)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != "bar" {
		t.Fatalf("expected 'bar', got '%s'", strings.TrimSpace(string(out)))
	}

	// List
	cmd = exec.Command(cliBin, "list", "keys", "--token", token)
	cmd.Env = testEnv(tmp)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "foo") {
		t.Fatalf("list output should contain 'foo': %s", out)
	}

	// Delete
	cmd = exec.Command(cliBin, "delete", "foo", "--token", token)
	cmd.Env = testEnv(tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("delete failed: %v\n%s", err, out)
	}

	// Verify deletion
	cmd = exec.Command(cliBin, "get", "foo", "--token", token)
	cmd.Env = testEnv(tmp)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("get should have failed after delete, but got: %s", out)
	}
}

func TestCreateUserAndLogin(t *testing.T) {
	tmp := t.TempDir()
	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "setup")
	cmd.Env = testEnv(tmp)
	cmd.Stdin = strings.NewReader("Y\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := ExtractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}
	// Create user (simulate input)
	cmd = exec.Command(cliBin, "create-user")
	envWithToken := append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	cmd.Env = envWithToken
	cmd.Stdin = strings.NewReader("bob\nreader\n\n")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create-user failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Generated token") {
		t.Fatalf("create-user did not print token: %s", out)
	}
}

func TestMain(m *testing.M) {
	// Build the binary for CLI integration tests - see `cliBin` at top of file
	cmd := exec.Command("go", "build", "-o", "simple-secrets")
	cmd.Dir = ".." // backend directory
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to build CLI binary: %v", err)
	}
	os.Exit(m.Run())
}
