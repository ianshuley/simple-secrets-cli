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
			cmd.Env = append(os.Environ(), "HOME="+tmp)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp)
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
	cmd.Env = append(os.Environ(), "HOME="+tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}

	// Extract token from output
	lines := strings.Split(string(out), "\n")
	var token string
	for _, line := range lines {
		if strings.Contains(line, "Token:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				token = fields[1]
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
			cmd.Env = append(os.Environ(), "HOME="+tmp)
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
