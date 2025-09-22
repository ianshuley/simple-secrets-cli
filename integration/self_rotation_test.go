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
	"strings"
	"testing"

	"simple-secrets/integration/testing_framework"
)

func TestSelfTokenRotationBothWays(t *testing.T) {
	env := testing_framework.NewEnvironment(t)
	defer env.Cleanup()

	// Create a reader user for testing
	output, err := env.CLI().Users().Create("reader-test", "reader")
	if err != nil {
		t.Fatalf("create-user failed: %v\n%s", err, output)
	}
	readerToken := testing_framework.ParseTokenFromCreateUser(string(output))
	if readerToken == "" {
		t.Fatalf("could not extract reader token from output: %s", output)
	}

	// Create a CLI runner for the reader user
	readerCLI := testing_framework.NewCLIRunnerWithToken(env, readerToken)

	// Test 1: Reader can rotate own token without specifying username
	output, err = readerCLI.Rotate().SelfToken()
	if err != nil {
		t.Fatalf("self token rotation (no username) failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Your token has been rotated successfully") {
		t.Errorf("expected self-rotation success message, got: %s", output)
	}

	// Extract new token and verify it works
	newReaderToken := testing_framework.ParseTokenFromSelfRotation(string(output))
	if newReaderToken == "" {
		t.Fatalf("could not extract new reader token from output: %s", output)
	}

	// Update the CLI runner with new token
	readerCLI = testing_framework.NewCLIRunnerWithToken(env, newReaderToken)

	// Test 2: Reader can rotate own token by specifying their username
	output, err = readerCLI.Rotate().Token("reader-test")
	if err != nil {
		t.Fatalf("self token rotation (with username) failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Your token has been rotated successfully") {
		t.Errorf("expected self-rotation success message, got: %s", output)
	}

	// Test 3: Verify reader still cannot rotate other users' tokens
	finalReaderToken := testing_framework.ParseTokenFromSelfRotation(string(output))
	readerCLI = testing_framework.NewCLIRunnerWithToken(env, finalReaderToken)

	output, err = readerCLI.Rotate().Token("admin")
	if err == nil {
		t.Fatalf("expected reader to fail when rotating admin token, but succeeded: %s", output)
	}
	if !strings.Contains(string(output), "permission denied") {
		t.Errorf("expected permission denied error, got: %s", output)
	}
}
