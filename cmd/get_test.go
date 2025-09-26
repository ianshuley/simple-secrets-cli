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
	"testing"

	"github.com/spf13/cobra"
)

func TestGetCommandFlags(t *testing.T) {
	tests := []struct {
		name           string
		args          []string
		expectClipboard bool
		expectSilent   bool
	}{
		{
			name:           "no flags",
			args:          []string{"test-key"},
			expectClipboard: false,
			expectSilent:   false,
		},
		{
			name:           "clipboard flag short",
			args:          []string{"-c", "test-key"},
			expectClipboard: true,
			expectSilent:   false,
		},
		{
			name:           "clipboard flag long",
			args:          []string{"--clipboard", "test-key"},
			expectClipboard: true,
			expectSilent:   false,
		},
		{
			name:           "silent flag short",
			args:          []string{"-s", "test-key"},
			expectClipboard: false,
			expectSilent:   true,
		},
		{
			name:           "silent flag long",
			args:          []string{"--silent", "test-key"},
			expectClipboard: false,
			expectSilent:   true,
		},
		{
			name:           "both flags short",
			args:          []string{"-c", "-s", "test-key"},
			expectClipboard: true,
			expectSilent:   true,
		},
		{
			name:           "both flags long",
			args:          []string{"--clipboard", "--silent", "test-key"},
			expectClipboard: true,
			expectSilent:   true,
		},
		{
			name:           "combined short flags",
			args:          []string{"-cs", "test-key"},
			expectClipboard: true,
			expectSilent:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command for each test to avoid flag pollution
			cmd := &cobra.Command{
				Use:  "get [key]",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Don't actually execute - just test flag parsing
					return nil
				},
			}
			cmd.Flags().BoolP("clipboard", "c", false, "copy secret to clipboard")
			cmd.Flags().BoolP("silent", "s", false, "suppress output to stdout")

			// Set args and parse
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check flag values
			clipboard, _ := cmd.Flags().GetBool("clipboard")
			silent, _ := cmd.Flags().GetBool("silent")

			if clipboard != tt.expectClipboard {
				t.Errorf("expected clipboard=%v, got %v", tt.expectClipboard, clipboard)
			}
			if silent != tt.expectSilent {
				t.Errorf("expected silent=%v, got %v", tt.expectSilent, silent)
			}
		})
	}
}

func TestGetCommandArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "valid single argument",
			args:    []string{"test-key"},
			wantErr: false,
		},
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "too many arguments",
			args:    []string{"key1", "key2"},
			wantErr: true,
		},
		{
			name:    "valid with clipboard flag",
			args:    []string{"--clipboard", "test-key"},
			wantErr: false,
		},
		{
			name:    "valid with silent flag",
			args:    []string{"--silent", "test-key"},
			wantErr: false,
		},
		{
			name:    "valid with both flags",
			args:    []string{"--clipboard", "--silent", "test-key"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "get [key]",
				Args: cobra.ExactArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Don't actually execute - just test argument validation
					return nil
				},
			}
			cmd.Flags().BoolP("clipboard", "c", false, "copy secret to clipboard")
			cmd.Flags().BoolP("silent", "s", false, "suppress output to stdout")

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}