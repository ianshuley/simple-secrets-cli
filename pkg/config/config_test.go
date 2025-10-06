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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSimpleSecretsPath(t *testing.T) {
	path, err := GetSimpleSecretsPath()
	if err != nil {
		t.Fatalf("GetSimpleSecretsPath() error: %v", err)
	}

	if path == "" {
		t.Error("GetSimpleSecretsPath() returned empty path")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("GetSimpleSecretsPath() returned relative path: %s", path)
	}

	if filepath.Base(path) != ".simple-secrets" {
		t.Errorf("GetSimpleSecretsPath() should end with .simple-secrets, got: %s", path)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		envVar   string
		wantBase string
	}{
		{
			name:     "normal config file",
			filename: "test.json",
			wantBase: "test.json",
		},
		{
			name:     "with environment override",
			filename: "test.json",
			envVar:   "/tmp/test-config",
			wantBase: "test.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment if needed
			originalEnv := os.Getenv("SIMPLE_SECRETS_CONFIG_DIR")
			if tt.envVar != "" {
				os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", tt.envVar)
			}
			defer os.Setenv("SIMPLE_SECRETS_CONFIG_DIR", originalEnv)

			path, err := DefaultConfigPath(tt.filename)
			if err != nil {
				t.Fatalf("DefaultConfigPath() error: %v", err)
			}

			if filepath.Base(path) != tt.wantBase {
				t.Errorf("DefaultConfigPath() base = %v, want %v", filepath.Base(path), tt.wantBase)
			}

			if tt.envVar != "" {
				expectedPath := filepath.Join(tt.envVar, tt.filename)
				if path != expectedPath {
					t.Errorf("DefaultConfigPath() with env = %v, want %v", path, expectedPath)
				}
			}
		})
	}
}

func TestResolveConfigPaths(t *testing.T) {
	usersPath, rolesPath, err := ResolveConfigPaths()
	if err != nil {
		t.Fatalf("ResolveConfigPaths() error: %v", err)
	}

	if filepath.Base(usersPath) != "users.json" {
		t.Errorf("users path should end with users.json, got: %s", usersPath)
	}

	if filepath.Base(rolesPath) != "roles.json" {
		t.Errorf("roles path should end with roles.json, got: %s", rolesPath)
	}

	if filepath.Dir(usersPath) != filepath.Dir(rolesPath) {
		t.Error("users and roles should be in the same directory")
	}
}

func TestEnsureConfigDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "config", "test.json")

	err := EnsureConfigDirectory(configPath)
	if err != nil {
		t.Fatalf("EnsureConfigDirectory() error: %v", err)
	}

	// Check that the directory was created
	expectedDir := filepath.Dir(configPath)
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("EnsureConfigDirectory() did not create directory: %s", expectedDir)
	}
}

func TestMarshalConfigData(t *testing.T) {
	tests := []struct {
		name string
		data any
		want string
	}{
		{
			name: "simple object",
			data: map[string]string{"key": "value"},
			want: `{
  "key": "value"
}`,
		},
		{
			name: "empty object",
			data: map[string]any{},
			want: "{}",
		},
		{
			name: "array",
			data: []string{"a", "b", "c"},
			want: `[
  "a",
  "b",
  "c"
]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalConfigData(tt.data)
			if err != nil {
				t.Fatalf("MarshalConfigData() error: %v", err)
			}

			if string(got) != tt.want {
				t.Errorf("MarshalConfigData() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestReadConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	// Write test data
	testData := map[string]string{"test": "value"}
	data, _ := MarshalConfigData(testData)
	err := os.WriteFile(configFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read it back
	var result map[string]string
	err = ReadConfigFile(configFile, &result)
	if err != nil {
		t.Fatalf("ReadConfigFile() error: %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("ReadConfigFile() got %v, want %v", result["test"], "value")
	}
}

func TestWriteConfigFileSecurely(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.json")

	testData := map[string]string{"test": "value"}

	// Mock atomic write function
	atomicWriteFunc := func(path string, data []byte, perm os.FileMode) error {
		return os.WriteFile(path, data, perm)
	}

	err := WriteConfigFileSecurely(configFile, testData, atomicWriteFunc)
	if err != nil {
		t.Fatalf("WriteConfigFileSecurely() error: %v", err)
	}

	// Verify file was written correctly
	var result map[string]string
	err = ReadConfigFile(configFile, &result)
	if err != nil {
		t.Fatalf("Failed to read back config file: %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("Written config got %v, want %v", result["test"], "value")
	}
}
