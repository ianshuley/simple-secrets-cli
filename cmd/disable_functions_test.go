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
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConfirmTokenDisable(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "yes_confirms_disable",
			input:    "yes\n",
			expected: true,
		},
		{
			name:     "yes_with_whitespace_confirms",
			input:    "  yes  \n",
			expected: true,
		},
		{
			name:     "uppercase_yes_confirms",
			input:    "YES\n",
			expected: true,
		},
		{
			name:     "mixed_case_yes_confirms",
			input:    "Yes\n",
			expected: true,
		},
		{
			name:     "no_rejects_disable",
			input:    "no\n",
			expected: false,
		},
		{
			name:     "empty_input_rejects",
			input:    "\n",
			expected: false,
		},
		{
			name:     "random_text_rejects",
			input:    "maybe\n",
			expected: false,
		},
		{
			name:     "partial_yes_rejects",
			input:    "ye\n",
			expected: false,
		},
		{
			name:     "yes_with_extra_text_rejects",
			input:    "yes please\n",
			expected: false,
		},
		{
			name:     "y_alone_rejects",
			input:    "y\n",
			expected: false,
		},
		{
			name:     "numeric_input_rejects",
			input:    "1\n",
			expected: false,
		},
		{
			name:     "special_characters_reject",
			input:    "yes!\n",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to capture output
			oldStdout := os.Stdout
			captureR, captureW, _ := os.Pipe()
			os.Stdout = captureW

			// Channel to capture stdout
			outputChan := make(chan string)
			go func() {
				var buf bytes.Buffer
				buf.ReadFrom(captureR)
				outputChan <- buf.String()
			}()

			// Write test input to stdin
			go func() {
				w.Write([]byte(tt.input))
				w.Close()
			}()

			// Call function under test
			result := confirmTokenDisable()

			// Restore stdin and stdout
			os.Stdin = oldStdin
			captureW.Close()
			os.Stdout = oldStdout
			r.Close()

			// Get captured output
			output := <-outputChan
			captureR.Close()

			// Verify result
			if result != tt.expected {
				t.Errorf("confirmTokenDisable() with input %q expected %v, got %v", tt.input, tt.expected, result)
			}

			// Verify prompt is displayed
			if !strings.Contains(output, "Proceed? (type 'yes'):") {
				t.Errorf("confirmTokenDisable() should display confirmation prompt, got: %q", output)
			}

			// Verify abort message for false cases
			if !tt.expected && !strings.Contains(output, "Aborted.") {
				t.Errorf("confirmTokenDisable() should display 'Aborted.' for rejected input, got: %q", output)
			}
		})
	}
}

func TestConfirmTokenDisableEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "yes_followed_by_newlines",
			input:    "yes\n\n\n",
			expected: true,
		},
		{
			name:     "multiple_words_starting_with_yes",
			input:    "yes indeed\n",
			expected: false,
		},
		{
			name:     "whitespace_only_input",
			input:    "   \n",
			expected: false,
		},
		{
			name:     "tabs_and_spaces_with_yes",
			input:    "\t  yes  \t\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock stdin
			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r

			// Mock stdout to suppress output during test
			oldStdout := os.Stdout
			captureR, captureW, _ := os.Pipe()
			os.Stdout = captureW

			// Channel to capture stdout (even though we don't use it here)
			outputChan := make(chan string)
			go func() {
				var buf bytes.Buffer
				buf.ReadFrom(captureR)
				outputChan <- buf.String()
			}()

			// Write test input to stdin
			go func() {
				w.Write([]byte(tt.input))
				w.Close()
			}()

			// Call function under test
			result := confirmTokenDisable()

			// Restore stdin and stdout
			os.Stdin = oldStdin
			captureW.Close()
			os.Stdout = oldStdout
			r.Close()

			// Wait for output capture to complete
			<-outputChan
			captureR.Close()

			// Verify result
			if result != tt.expected {
				t.Errorf("confirmTokenDisable() edge case with input %q expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}
