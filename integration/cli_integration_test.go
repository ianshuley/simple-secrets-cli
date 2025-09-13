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
	tmp := t.TempDir()
	// Set HOME to temp dir for isolation
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	// Should create .simple-secrets/users.json and roles.json
	if _, err := os.Stat(filepath.Join(tmp, ".simple-secrets", "users.json")); err != nil {
		t.Fatalf("users.json not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".simple-secrets", "roles.json")); err != nil {
		t.Fatalf("roles.json not created: %v", err)
	}
	// Should print first-run message
	if !strings.Contains(string(out), "First run detected") {
		t.Fatalf("missing first-run message: %s", out)
	}
}

func TestPutGetListDelete(t *testing.T) {
	tmp := t.TempDir()
	// First run: trigger creation and capture token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	// Extract token from output
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Put
	cmd = exec.Command(cliBin, "put", "foo", "bar")
	envWithToken := append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+token)
	cmd.Env = envWithToken
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put failed: %v\n%s", err, out)
	}
	// Get
	cmd = exec.Command(cliBin, "get", "foo")
	cmd.Env = envWithToken
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bar") {
		t.Fatalf("get did not return value: %s", out)
	}
	// List
	cmd = exec.Command(cliBin, "list", "keys")
	cmd.Env = envWithToken
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "foo") {
		t.Fatalf("list did not show key: %s", out)
	}
	// Delete
	cmd = exec.Command(cliBin, "delete", "foo")
	cmd.Env = envWithToken
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("delete failed: %v\n%s", err, out)
	}
}

// extractToken parses the admin token from first-run output
func extractToken(out string) string {
	return extractTokenFromOutput(out)
}

func TestCreateUserAndLogin(t *testing.T) {
	tmp := t.TempDir()
	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
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
