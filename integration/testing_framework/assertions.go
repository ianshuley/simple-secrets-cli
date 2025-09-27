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
package testing_framework

import (
	"strings"
	"testing"
)

// TestAssertions provides fluent test assertions for CLI output
type TestAssertions struct {
	t      *testing.T
	output []byte
	err    error
}

// Assert creates a new assertion helper for CLI command results
func Assert(t *testing.T, output []byte, err error) *TestAssertions {
	return &TestAssertions{
		t:      t,
		output: output,
		err:    err,
	}
}

// Success asserts that the command succeeded (no error)
func (a *TestAssertions) Success() *TestAssertions {
	a.t.Helper()
	if a.err != nil {
		a.t.Errorf("command should have succeeded but failed: %v\n%s", a.err, a.output)
	}
	return a
}

// Failure asserts that the command failed (has error)
func (a *TestAssertions) Failure() *TestAssertions {
	a.t.Helper()
	if a.err == nil {
		a.t.Errorf("command should have failed but succeeded: %s", a.output)
	}
	return a
}

// Contains asserts that output contains the expected text
func (a *TestAssertions) Contains(expected string) *TestAssertions {
	a.t.Helper()
	if !strings.Contains(string(a.output), expected) {
		a.t.Errorf("output should contain %q but got: %s", expected, a.output)
	}
	return a
}

// NotContains asserts that output does not contain the forbidden text
func (a *TestAssertions) NotContains(forbidden string) *TestAssertions {
	a.t.Helper()
	if strings.Contains(string(a.output), forbidden) {
		a.t.Errorf("output should not contain %q but got: %s", forbidden, a.output)
	}
	return a
}

// Equals asserts that output exactly equals the expected value
func (a *TestAssertions) Equals(expected string) *TestAssertions {
	a.t.Helper()
	actual := CleanOutput(string(a.output))
	if actual != expected {
		a.t.Errorf("output should equal %q but got %q", expected, actual)
	}
	return a
}

// Empty asserts that output is empty (after trimming whitespace)
func (a *TestAssertions) Empty() *TestAssertions {
	a.t.Helper()
	if CleanOutput(string(a.output)) != "" {
		a.t.Errorf("output should be empty but got: %s", a.output)
	}
	return a
}

// NotEmpty asserts that output is not empty
func (a *TestAssertions) NotEmpty() *TestAssertions {
	a.t.Helper()
	if CleanOutput(string(a.output)) == "" {
		a.t.Errorf("output should not be empty")
	}
	return a
}

// ValidToken asserts that a token can be extracted and is valid format
func (a *TestAssertions) ValidToken() *TestAssertions {
	a.t.Helper()
	token := ParseToken(string(a.output))

	if token == "" {
		a.t.Errorf("could not extract valid token from output: %s", a.output)
		return a
	}

	if !AssertTokenFormat(token) {
		a.t.Errorf("extracted token has invalid format: %q", token)
		return a
	}

	return a
}

// ExtractToken extracts and returns the token from output (for use in subsequent commands)
func (a *TestAssertions) ExtractToken() string {
	a.t.Helper()
	token := ParseToken(string(a.output))
	if token == "" {
		a.t.Fatalf("could not extract token from output: %s", a.output)
	}
	return token
}

// Output returns the raw output for custom assertions
func (a *TestAssertions) Output() string {
	return string(a.output)
}

// Error returns the error for custom assertions
func (a *TestAssertions) Error() error {
	return a.err
}
