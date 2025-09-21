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

// TestEmptyTokenBypassVulnerability tests the specific security issue where --token "" was accepted
func TestEmptyTokenBypassVulnerability(t *testing.T) {
	tmp := t.TempDir()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "explicit_empty_token_list",
			args:    []string{"--token", "", "list", "keys"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
		{
			name:    "explicit_empty_token_put",
			args:    []string{"--token", "", "put", "key", "value"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
		{
			name:    "explicit_empty_token_delete",
			args:    []string{"--token", "", "delete", "key"},
			wantErr: true,
			errMsg:  "authentication required: token cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, tt.args...)
			cmd.Env = testEnv(tmp)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded. Output: %s", out)
					return
				}
				if !strings.Contains(string(out), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got: %s", tt.errMsg, out)
				}
			}
		})
	}
}

// TestDirectoryPermissionsVulnerability tests that directories are created with 700 permissions
func TestDirectoryPermissionsVulnerability(t *testing.T) {
	tmp := t.TempDir()

	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	_, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	configDir := filepath.Join(tmp, ".simple-secrets")
	stat, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}

	perm := stat.Mode().Perm()
	expected := os.FileMode(0700)
	if perm != expected {
		t.Errorf("config directory has insecure permissions. Expected %s, got %s", expected, perm)
	}
}

// TestInputValidationVulnerabilities tests that malicious key names are rejected
func TestInputValidationVulnerabilities(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Extract token from output
	lines := strings.Split(string(out), "\n")
	var token string
	for _, line := range lines {
		if strings.Contains(line, "🔑 Your authentication token:") {
			// Token is on the next line
			continue
		}
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "🔑") && !strings.Contains(line, "📋") && !strings.Contains(line, "Created") && !strings.Contains(line, "Username:") {
			// This might be the token line
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 20 && !strings.Contains(trimmed, " ") {
				token = trimmed
				break
			}
		}
	}
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	tests := []struct {
		name    string
		key     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "control_character_key_injection",
			key:     "test\x01key",
			wantErr: true,
			errMsg:  "key name cannot contain control characters",
		},
		{
			name:    "path_traversal_key",
			key:     "test/path",
			wantErr: true,
			errMsg:  "key name cannot contain path separators or path traversal sequences",
		},
		{
			name:    "path_traversal_dots_key",
			key:     "test..path",
			wantErr: true,
			errMsg:  "key name cannot contain path separators or path traversal sequences",
		},
		{
			name:    "valid_key_should_work",
			key:     "valid-key",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, "--token", token, "put", tt.key, "testvalue")
			cmd.Env = testEnv(tmp)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded. Output: %s", out)
					return
				}
				if !strings.Contains(string(out), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got: %s", tt.errMsg, out)
				}
			} else {
				if err != nil {
					t.Errorf("expected success but got error: %v, output: %s", err, out)
				}
			}
		})
	}
}

// TestUsernamePathTraversalVulnerability tests that usernames cannot contain path traversal sequences
func TestUsernamePathTraversalVulnerability(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Extract token from output
	lines := strings.Split(string(out), "\n")
	var token string
	for _, line := range lines {
		if strings.Contains(line, "🔑 Your authentication token:") {
			// Token is on the next line
			continue
		}
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "🔑") && !strings.Contains(line, "📋") && !strings.Contains(line, "Created") && !strings.Contains(line, "Username:") {
			// This might be the token line
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 20 && !strings.Contains(trimmed, " ") {
				token = trimmed
				break
			}
		}
	}
	if token == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	tests := []struct {
		name     string
		username string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "path_traversal_dotdot_username",
			username: "../etc/passwd",
			wantErr:  true,
			errMsg:   "username cannot contain path separators or path traversal sequences",
		},
		{
			name:     "path_traversal_slash_username",
			username: "user/with/slash",
			wantErr:  true,
			errMsg:   "username cannot contain path separators or path traversal sequences",
		},
		{
			name:     "path_traversal_backslash_username",
			username: "user\\with\\backslash",
			wantErr:  true,
			errMsg:   "username cannot contain path separators or path traversal sequences",
		},
		{
			name:     "valid_username_should_work",
			username: "valid-user",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(cliBin, "--token", token, "create-user", tt.username, "reader")
			cmd.Env = testEnv(tmp)
			out, err := cmd.CombinedOutput()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but command succeeded. Output: %s", out)
					return
				}
				if !strings.Contains(string(out), tt.errMsg) {
					t.Errorf("expected error message to contain %q, got: %s", tt.errMsg, out)
				}
			} else {
				if err != nil {
					t.Errorf("expected success but got error: %v, output: %s", err, out)
				}
			}
		})
	}
}
