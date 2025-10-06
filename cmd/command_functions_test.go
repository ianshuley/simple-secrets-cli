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
	"context"
	"os"
	"strings"
	"testing"

	"simple-secrets/internal/platform"
	"simple-secrets/pkg/crypto"

	"github.com/spf13/cobra"
)

func TestParsePutArgumentsWithToken(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupEnv    func(t *testing.T) func() // Returns cleanup function
		wantErr     bool
		errContains string
		checkResult func(t *testing.T, result *putArguments)
	}{
		{
			name: "token_from_environment",
			args: []string{"test-key", "test-value"},
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				token := createTestPlatformWithToken(t, tempDir)
				os.Setenv("SIMPLE_SECRETS_TOKEN", token)
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
					os.Unsetenv("SIMPLE_SECRETS_TOKEN")
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *putArguments) {
				if result.key != "test-key" || result.value != "test-value" {
					t.Errorf("Expected key=test-key, value=test-value, got key=%s, value=%s", result.key, result.value)
				}
				if result.token == "" {
					t.Errorf("Expected token to be resolved from environment")
				}
			},
		},
		{
			name: "token_from_explicit_flag",
			args: []string{"test-key", "test-value", "--token", "explicit-token"},
			setupEnv: func(t *testing.T) func() {
				return func() {} // No environment setup needed
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *putArguments) {
				if result.token != "explicit-token" {
					t.Errorf("Expected token=explicit-token, got %s", result.token)
				}
			},
		},
		{
			name: "no_token_available",
			args: []string{"test-key", "test-value"},
			setupEnv: func(t *testing.T) func() {
				// Clean environment - no token
				os.Unsetenv("SIMPLE_SECRETS_TOKEN")
				return func() {}
			},
			wantErr:     true,
			errContains: "authentication required",
		},
		{
			name: "generate_flag_parsing",
			args: []string{"api-key", "--generate", "--length", "64"},
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				token := createTestPlatformWithToken(t, tempDir)
				os.Setenv("SIMPLE_SECRETS_TOKEN", token)
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
					os.Unsetenv("SIMPLE_SECRETS_TOKEN")
				}
			},
			wantErr: false,
			checkResult: func(t *testing.T, result *putArguments) {
				if result.key != "api-key" || !result.generate || result.length != 64 {
					t.Errorf("Expected key=api-key, generate=true, length=64, got key=%s, generate=%v, length=%d",
						result.key, result.generate, result.length)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv(t)
			defer cleanup()

			cmd := &cobra.Command{Use: "put"}
			result, err := parsePutArguments(cmd, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parsePutArguments() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parsePutArguments() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("parsePutArguments() unexpected error = %v", err)
					return
				}
				if result == nil {
					t.Errorf("parsePutArguments() returned nil result")
					return
				}
				if tt.checkResult != nil {
					tt.checkResult(t, result)
				}
			}
		})
	}
}

func TestExecutePutCommand(t *testing.T) {
	tests := []struct {
		name        string
		setupArgs   func(t *testing.T) (*putArguments, func()) // Returns args and cleanup
		wantErr     bool
		errContains string
	}{
		{
			name: "successful_put_with_valid_token",
			setupArgs: func(t *testing.T) (*putArguments, func()) {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				token := createTestPlatformWithToken(t, tempDir)

				args := &putArguments{
					key:      "test-key",
					value:    "test-value",
					token:    token,
					generate: false,
				}

				cleanup := func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}

				return args, cleanup
			},
			wantErr: false,
		},
		{
			name: "successful_put_with_generate",
			setupArgs: func(t *testing.T) (*putArguments, func()) {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				token := createTestPlatformWithToken(t, tempDir)

				args := &putArguments{
					key:      "generated-key",
					token:    token,
					generate: true,
					length:   32,
				}

				cleanup := func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}

				return args, cleanup
			},
			wantErr: false,
		},
		{
			name: "invalid_token",
			setupArgs: func(t *testing.T) (*putArguments, func()) {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				createTestPlatformWithToken(t, tempDir) // Create platform but use wrong token

				args := &putArguments{
					key:   "test-key",
					value: "test-value",
					token: "invalid-token",
				}

				cleanup := func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}

				return args, cleanup
			},
			wantErr:     true,
			errContains: "authentication failed",
		},
		{
			name: "invalid_key_name",
			setupArgs: func(t *testing.T) (*putArguments, func()) {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				token := createTestPlatformWithToken(t, tempDir)

				args := &putArguments{
					key:   "../invalid-key",
					value: "test-value",
					token: token,
				}

				cleanup := func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}

				return args, cleanup
			},
			wantErr:     true,
			errContains: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, cleanup := tt.setupArgs(t)
			defer cleanup()

			err := executePutCommand(args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("executePutCommand() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("executePutCommand() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("executePutCommand() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestGetMasterKey(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func(t *testing.T) func() // Returns cleanup function
		wantErr  bool
	}{
		{
			name: "existing_master_key",
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				// Create a platform to ensure master key exists
				createTestPlatformWithToken(t, tempDir)
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}
			},
			wantErr: false,
		},
		{
			name: "no_master_key_exists",
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				// Don't create platform - no master key should exist
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv(t)
			defer cleanup()

			key, err := getMasterKey()

			if tt.wantErr {
				if err == nil {
					t.Errorf("getMasterKey() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("getMasterKey() unexpected error = %v", err)
					return
				}
				if len(key) == 0 {
					t.Errorf("getMasterKey() returned empty key")
				}
			}
		})
	}
}

func TestGetPlatformConfigNew(t *testing.T) {
	tests := []struct {
		name     string
		setupEnv func(t *testing.T) func() // Returns cleanup function
		wantErr  bool
	}{
		{
			name: "default_config_path",
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tempDir+"/.simple-secrets")
				// Create a platform to ensure master key exists
				createTestPlatformWithToken(t, tempDir)
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}
			},
			wantErr: false,
		},
		{
			name: "custom_config_path",
			setupEnv: func(t *testing.T) func() {
				tempDir := t.TempDir()
				customDir := tempDir + "/custom-simple-secrets"
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", customDir)
				// Create a platform manually with the custom dir
				createTestPlatformWithCustomDir(t, customDir)
				return func() {
					os.Unsetenv("SIMPLE_SECRETS_CONFIG_DIR")
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv(t)
			defer cleanup()

			config, err := getPlatformConfig()

			if tt.wantErr {
				if err == nil {
					t.Errorf("getPlatformConfig() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("getPlatformConfig() unexpected error = %v", err)
					return
				}
				if config.DataDir == "" {
					t.Errorf("getPlatformConfig() returned empty DataDir")
				}
				if len(config.MasterKey) == 0 {
					t.Errorf("getPlatformConfig() returned empty MasterKey")
				}
			}
		})
	}
}

// Helper function to create test platform and return admin token
func createTestPlatformWithToken(t *testing.T, tempDir string) string {
	t.Helper()

	ctx := context.Background()

	// Generate master key
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Create platform config
	config := platform.Config{
		DataDir:   tempDir + "/.simple-secrets",
		MasterKey: masterKey,
	}

	// Initialize platform
	app, err := platform.New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test platform: %v", err)
	}
	defer app.Close()

	// Create admin user and get token
	_, token, err := app.Users.Create(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	return token
}

// Helper function to create test platform with custom directory
func createTestPlatformWithCustomDir(t *testing.T, configDir string) string {
	t.Helper()

	ctx := context.Background()

	// Generate master key
	masterKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Create platform config with custom directory
	config := platform.Config{
		DataDir:   configDir,
		MasterKey: masterKey,
	}

	// Initialize platform
	app, err := platform.New(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create test platform: %v", err)
	}
	defer app.Close()

	// Create admin user and get token
	_, token, err := app.Users.Create(ctx, "admin", "admin")
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	return token
}
