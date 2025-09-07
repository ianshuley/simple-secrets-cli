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
	"os/exec"
	"strings"
	"testing"
)

func TestSelfTokenRotationBothWays(t *testing.T) {
	tmp := t.TempDir()

	// First run to create admin and extract token
	cmd := exec.Command(cliBin, "list", "keys")
	cmd.Env = testEnv(tmp)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("first run failed: %v\n%s", err, out)
	}
	adminToken := extractToken(string(out))
	if adminToken == "" {
		t.Fatalf("could not extract admin token from output: %s", out)
	}

	// Create a reader user for testing
	cmd = exec.Command(cliBin, "create-user", "reader-test", "reader")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+adminToken)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create-user failed: %v\n%s", err, out)
	}
	readerToken := extractTokenFromCreateUser(string(out))
	if readerToken == "" {
		t.Fatalf("could not extract reader token from output: %s", out)
	}

	// Test 1: Reader can rotate own token without specifying username
	cmd = exec.Command(cliBin, "rotate", "token")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+readerToken)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self token rotation (no username) failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Your token has been rotated successfully") {
		t.Errorf("expected self-rotation success message, got: %s", out)
	}

	// Extract new token and verify it works
	newReaderToken := extractTokenFromSelfRotation(string(out))
	if newReaderToken == "" {
		t.Fatalf("could not extract new reader token from output: %s", out)
	}

	// Test 2: Reader can rotate own token by specifying their username
	cmd = exec.Command(cliBin, "rotate", "token", "reader-test")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+newReaderToken)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self token rotation (with username) failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Your token has been rotated successfully") {
		t.Errorf("expected self-rotation success message, got: %s", out)
	}

	// Test 3: Verify reader still cannot rotate other users' tokens
	finalReaderToken := extractTokenFromSelfRotation(string(out))
	cmd = exec.Command(cliBin, "rotate", "token", "admin")
	cmd.Env = append(testEnv(tmp), "SIMPLE_SECRETS_TOKEN="+finalReaderToken)
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected reader to fail when rotating admin token, but succeeded: %s", out)
	}
	if !strings.Contains(string(out), "permission denied") {
		t.Errorf("expected permission denied error, got: %s", out)
	}
}

// extractTokenFromCreateUser extracts the token from create-user command output
func extractTokenFromCreateUser(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Generated token: ") {
			return strings.TrimPrefix(line, "Generated token: ")
		}
	}
	return ""
}

// extractTokenFromSelfRotation extracts the token from self-rotation command output
func extractTokenFromSelfRotation(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "New token: ") {
			return strings.TrimPrefix(line, "New token: ")
		}
	}
	return ""
}
