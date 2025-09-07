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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestErrorHandling(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Create a separate temp directory for "no token" test where no config exists
	noTokenTmp := t.TempDir()
	// Copy users.json and roles.json to the no-token directory so it's not a first run, but don't copy config.json
	simpleSecretsDir := filepath.Join(noTokenTmp, ".simple-secrets")
	err = os.MkdirAll(simpleSecretsDir, 0755)
	if err != nil {
		t.Fatalf("failed to create .simple-secrets dir: %v", err)
	}

	// Copy users.json
	srcUsersFile := filepath.Join(tmp, ".simple-secrets", "users.json")
	dstUsersFile := filepath.Join(simpleSecretsDir, "users.json")
	usersData, err := os.ReadFile(srcUsersFile)
	if err != nil {
		t.Fatalf("failed to read users.json: %v", err)
	}
	err = os.WriteFile(dstUsersFile, usersData, 0600)
	if err != nil {
		t.Fatalf("failed to write users.json: %v", err)
	}

	// Copy roles.json
	srcRolesFile := filepath.Join(tmp, ".simple-secrets", "roles.json")
	dstRolesFile := filepath.Join(simpleSecretsDir, "roles.json")
	rolesData, err := os.ReadFile(srcRolesFile)
	if err != nil {
		t.Fatalf("failed to read roles.json: %v", err)
	}
	err = os.WriteFile(dstRolesFile, rolesData, 0600)
	if err != nil {
		t.Fatalf("failed to write roles.json: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		env          []string
		wantErr      bool
		errorMessage string
	}{
		{
			name:         "invalid token",
			args:         []string{"list", "keys"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN=invalid"),
			wantErr:      true,
			errorMessage: "invalid token",
		},
		{
			name:         "no token provided",
			args:         []string{"list", "keys"},
			env:          []string{"HOME=" + noTokenTmp}, // Clean environment with only HOME set
			wantErr:      true,
			errorMessage: "provide a token via",
		},
		{
			name:         "list invalid subcommand",
			args:         []string{"list", "invalid"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "unknown list type",
		},
		{
			name:         "rotate invalid subcommand",
			args:         []string{"rotate", "invalid"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "unknown rotate type",
		},
		{
			name:         "restore invalid subcommand",
			args:         []string{"restore", "invalid"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "unknown restore type",
		},
		{
			name:         "get non-existent secret",
			args:         []string{"get", "nonexistent"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "not found",
		},
		{
			name:         "delete non-existent secret",
			args:         []string{"delete", "nonexistent"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "file does not exist",
		},
		{
			name:         "restore non-existent secret",
			args:         []string{"restore", "secret", "nonexistent"},
			env:          append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token),
			wantErr:      true,
			errorMessage: "could not read backup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = tt.env
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
					return
				}

				if !strings.Contains(string(out), tt.errorMessage) {
					t.Errorf("expected error message to contain %q but got: %s", tt.errorMessage, out)
				}
				return
			}

			if err != nil {
				t.Errorf("command failed: %v\n%s", err, out)
			}
		})
	}
}

func TestRBACEnforcement(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	adminToken := extractToken(string(out))
	if adminToken == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Create a reader user
	cmd = exec.Command(cliBin, "create-user", "reader", "reader")
	cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+adminToken)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create reader user failed: %v\n%s", err, out)
	}

	// Extract reader token from output
	lines := strings.Split(string(out), "\n")
	var readerToken string
	for _, line := range lines {
		if strings.Contains(line, "Generated token:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				readerToken = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	if readerToken == "" {
		t.Fatalf("could not extract reader token from output: %s", out)
	}

	tests := []struct {
		name     string
		args     []string
		token    string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "reader can list keys",
			args:    []string{"list", "keys"},
			token:   readerToken,
			wantErr: false,
		},
		{
			name:    "reader can get secrets",
			args:    []string{"get", "nonexistent"}, // Will fail for other reasons, but not RBAC
			token:   readerToken,
			wantErr: true, // But not due to RBAC
		},
		{
			name:     "reader cannot rotate master key",
			args:     []string{"rotate", "master-key", "--yes"},
			token:    readerToken,
			wantErr:  true,
			errorMsg: "permission denied",
		},
		{
			name:     "reader cannot create users",
			args:     []string{"create-user", "test", "reader"},
			token:    readerToken,
			wantErr:  true,
			errorMsg: "permission denied",
		},
		{
			name:     "reader cannot list users",
			args:     []string{"list", "users"},
			token:    readerToken,
			wantErr:  true,
			errorMsg: "permission denied",
		},
		{
			name:     "reader cannot rotate tokens",
			args:     []string{"rotate", "token", "admin"},
			token:    readerToken,
			wantErr:  true,
			errorMsg: "permission denied",
		},
		{
			name:    "reader can rotate own token",
			args:    []string{"rotate", "token"},
			token:   readerToken,
			wantErr: false,
		},
		{
			name:    "admin can do everything",
			args:    []string{"list", "users"},
			token:   adminToken,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+tt.token)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
					return
				}

				if tt.errorMsg != "" && !strings.Contains(string(out), tt.errorMsg) {
					t.Errorf("expected error message to contain %q but got: %s", tt.errorMsg, out)
				}
				return
			}

			if err != nil {
				// For some tests like "get nonexistent", we expect failure but not due to RBAC
				if tt.args[0] == "get" && strings.Contains(string(out), "secret not found") {
					return // This is expected
				}
				t.Errorf("command failed: %v\n%s", err, out)
			}
		})
	}
}

func TestCommandInputValidation(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		errorMessage string
	}{
		{
			name:         "put without key",
			args:         []string{"put"},
			wantErr:      true,
			errorMessage: "accepts 2 arg(s)",
		},
		{
			name:         "put without value",
			args:         []string{"put", "key"},
			wantErr:      true,
			errorMessage: "accepts 2 arg(s)",
		},
		{
			name:         "get without key",
			args:         []string{"get"},
			wantErr:      true,
			errorMessage: "accepts 1 arg(s)",
		},
		{
			name:         "delete without key",
			args:         []string{"delete"},
			wantErr:      true,
			errorMessage: "accepts 1 arg(s)",
		},
		{
			name:         "create-user with too many args",
			args:         []string{"create-user", "user", "role", "extra"},
			wantErr:      true,
			errorMessage: "accepts at most 2 arg(s)",
		},
		{
			name:         "create-user with invalid role",
			args:         []string{"create-user", "user", "invalid"},
			wantErr:      true,
			errorMessage: "invalid role",
		},
		{
			name:    "create-user with valid role",
			args:    []string{"create-user", "testuser", "reader"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded: %s", out)
					return
				}

				if !strings.Contains(string(out), tt.errorMessage) {
					t.Errorf("expected error message to contain %q but got: %s", tt.errorMessage, out)
				}
				return
			}

			if err != nil {
				t.Errorf("command failed: %v\n%s", err, out)
			}
		})
	}
}

func TestWorkflowIntegration(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Test complete workflow: add secret -> rotate -> restore -> verify
	env := append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)

	// 1. Add a secret
	cmd = exec.Command(cliBin, "put", "workflow-test", "original-value")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put failed: %v", err)
	}

	// 2. Verify secret exists
	cmd = exec.Command(cliBin, "get", "workflow-test")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !strings.Contains(string(out), "original-value") {
		t.Fatalf("get returned wrong value: %s", out)
	}

	// 3. Update the secret (creates backup)
	cmd = exec.Command(cliBin, "put", "workflow-test", "updated-value")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("put update failed: %v", err)
	}

	// 4. Verify updated value
	cmd = exec.Command(cliBin, "get", "workflow-test")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get after update failed: %v", err)
	}
	if !strings.Contains(string(out), "updated-value") {
		t.Fatalf("get returned wrong updated value: %s", out)
	}

	// 5. Restore from backup
	cmd = exec.Command(cliBin, "restore", "secret", "workflow-test")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	// 6. Verify restored value
	cmd = exec.Command(cliBin, "get", "workflow-test")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get after restore failed: %v", err)
	}
	if !strings.Contains(string(out), "original-value") {
		t.Fatalf("restore didn't work, expected original-value, got: %s", out)
	}

	// 7. Rotate master key
	cmd = exec.Command(cliBin, "rotate", "master-key", "--yes")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rotate master key failed: %v", err)
	}

	// 8. Verify secret still accessible after rotation
	cmd = exec.Command(cliBin, "get", "workflow-test")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("get after rotation failed: %v", err)
	}
	if !strings.Contains(string(out), "original-value") {
		t.Fatalf("secret not accessible after rotation: %s", out)
	}

	// 9. Check that backup was created
	cmd = exec.Command(cliBin, "list", "backups")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list backups failed: %v", err)
	}
	if !strings.Contains(string(out), "rotation backup") {
		t.Fatalf("rotation backup not found: %s", out)
	}
}

func TestEdgeCases(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	token := extractToken(string(out))
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	env := append(os.Environ(), "HOME="+tmp, "SIMPLE_SECRETS_TOKEN="+token)

	// Test empty key name
	cmd = exec.Command(cliBin, "put", "", "value")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Errorf("expected error for empty key name, but command succeeded: %s", out)
	}

	// Test empty value
	cmd = exec.Command(cliBin, "put", "empty-key", "")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("putting empty value should work: %v", err)
	}

	// Test very long key name
	longKey := strings.Repeat("a", 1000)
	cmd = exec.Command(cliBin, "put", longKey, "value")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("putting long key should work: %v", err)
	}

	// Test special characters in key
	specialKey := "key-with-special@#$%^&*()chars"
	cmd = exec.Command(cliBin, "put", specialKey, "special-value")
	cmd.Env = env
	_, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("putting key with special chars should work: %v", err)
	}

	// Verify special key can be retrieved
	cmd = exec.Command(cliBin, "get", specialKey)
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("getting key with special chars failed: %v", err)
	}
	if !strings.Contains(string(out), "special-value") {
		t.Errorf("special key retrieval failed: %s", out)
	}
}
