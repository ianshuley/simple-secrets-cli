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

// CLIRunner provides a type-safe, fluent interface for running CLI commands
// in the test environment. It automatically handles token authentication
// and provides method chaining for readable test code.
type CLIRunner struct {
	env         *TestEnvironment
	customToken string // Optional custom token (overrides environment token)
}

// runWithToken executes a command with the admin token appended as --token flag
func (c *CLIRunner) runWithToken(args []string) ([]byte, error) {
	token := c.env.AdminToken()
	if c.customToken != "" {
		token = c.customToken
	}
	argsWithToken := append(args, "--token", token)
	return c.env.RunRawCommand(argsWithToken, c.env.CleanEnvironment(), "")
}

// runWithFirstRunProtection executes a command with first-run protection enabled
func (c *CLIRunner) runWithFirstRunProtection(args []string) ([]byte, error) {
	return c.env.RunRawCommand(args, c.env.FirstRunProtectionEnvironment(), "")
}

// SecretCommands provides secret management operations
type SecretCommands struct {
	cli *CLIRunner
}

// UserCommands provides user management operations
type UserCommands struct {
	cli *CLIRunner
}

// ListCommands provides listing operations
type ListCommands struct {
	cli *CLIRunner
}

// Put stores a secret value
func (c *CLIRunner) Put(key, value string) ([]byte, error) {
	return c.runWithToken([]string{"put", key, value})
}

// Get retrieves a secret value
func (c *CLIRunner) Get(key string) ([]byte, error) {
	return c.runWithToken([]string{"get", key})
}

// GetWithClipboard retrieves a secret value and copies it to clipboard
func (c *CLIRunner) GetWithClipboard(key string) ([]byte, error) {
	return c.runWithToken([]string{"get", key, "--clipboard"})
}

// GetSilent retrieves a secret value without printing to stdout
func (c *CLIRunner) GetSilent(key string) ([]byte, error) {
	return c.runWithToken([]string{"get", key, "--silent"})
}

// GetWithClipboardSilent retrieves a secret value, copies to clipboard, and suppresses stdout
func (c *CLIRunner) GetWithClipboardSilent(key string) ([]byte, error) {
	return c.runWithToken([]string{"get", key, "--clipboard", "--silent"})
}

// Delete removes a secret
func (c *CLIRunner) Delete(key string) ([]byte, error) {
	return c.runWithToken([]string{"delete", key})
}

// Secrets returns secret management commands
func (c *CLIRunner) Secrets() *SecretCommands {
	return &SecretCommands{cli: c}
}

// Users returns user management commands
func (c *CLIRunner) Users() *UserCommands {
	return &UserCommands{cli: c}
}

// List returns listing commands
func (c *CLIRunner) List() *ListCommands {
	return &ListCommands{cli: c}
}

// Setup runs the setup command with optional input
func (c *CLIRunner) Setup(input string) ([]byte, error) {
	if input == "" {
		input = "Y\n" // Default to accepting setup
	}
	return c.env.RunRawCommand([]string{"setup"}, nil, input)
}

// Raw executes a raw command with the given arguments
func (c *CLIRunner) Raw(args ...string) ([]byte, error) {
	return c.env.RunRawCommand(args, nil, "")
}

// RawWithInput executes a raw command with stdin input
func (c *CLIRunner) RawWithInput(input string, args ...string) ([]byte, error) {
	return c.env.RunRawCommand(args, nil, input)
}

// RawWithoutToken executes a command without the admin token
func (c *CLIRunner) RawWithoutToken(args ...string) ([]byte, error) {
	env := c.env.CleanEnvironment() // No token added
	return c.env.RunRawCommand(args, env, "")
}

// RawWithFirstRunProtection executes a command with first-run protection enabled
func (c *CLIRunner) RawWithFirstRunProtection(args ...string) ([]byte, error) {
	env := c.env.FirstRunProtectionEnvironment() // No test mode
	return c.env.RunRawCommand(args, env, "")
}

// Secret Commands

// Disable disables a secret
func (s *SecretCommands) Disable(key string) ([]byte, error) {
	return s.cli.runWithToken([]string{"disable", "secret", key})
}

// Enable enables a secret
func (s *SecretCommands) Enable(key string) ([]byte, error) {
	return s.cli.runWithToken([]string{"enable", "secret", key})
}

// Restore restores a secret from backup
func (s *SecretCommands) Restore(key string) ([]byte, error) {
	return s.cli.runWithToken([]string{"restore", "secret", key})
}

// User Commands

// Create creates a new user
func (u *UserCommands) Create(username, role string) ([]byte, error) {
	return u.cli.runWithToken([]string{"create-user", username, role})
}

// Disable disables a user
func (u *UserCommands) Disable(username string) ([]byte, error) {
	return u.cli.runWithToken([]string{"disable", "user", username})
}

// Enable enables a user
func (u *UserCommands) Enable(username string) ([]byte, error) {
	return u.cli.runWithToken([]string{"enable", "user", username})
}

// List Commands

// Keys lists all secret keys
func (l *ListCommands) Keys() ([]byte, error) {
	return l.cli.runWithToken([]string{"list", "keys"})
}

// Users lists all users
func (l *ListCommands) Users() ([]byte, error) {
	return l.cli.runWithToken([]string{"list", "users"})
}

// Disabled lists disabled secrets
func (l *ListCommands) Disabled() ([]byte, error) {
	return l.cli.runWithToken([]string{"list", "disabled"})
}

// Backups lists available backups
func (l *ListCommands) Backups() ([]byte, error) {
	return l.cli.runWithToken([]string{"list", "backups"})
}

// RotateCommands provides key and token rotation operations
type RotateCommands struct {
	cli *CLIRunner
}

// Rotate returns rotation commands
func (c *CLIRunner) Rotate() *RotateCommands {
	return &RotateCommands{cli: c}
}

// MasterKey rotates the master encryption key
func (r *RotateCommands) MasterKey() ([]byte, error) {
	return r.cli.runWithToken([]string{"rotate", "master-key", "--yes"})
}

// MasterKeyWithConfirmation rotates master key with interactive confirmation
func (r *RotateCommands) MasterKeyWithConfirmation(confirm string) ([]byte, error) {
	argsWithToken := append([]string{"rotate", "master-key"}, "--token", r.cli.env.AdminToken())
	return r.cli.env.RunRawCommand(argsWithToken, r.cli.env.CleanEnvironment(), confirm)
}

// Token rotates a user's authentication token
func (r *RotateCommands) Token(username string) ([]byte, error) {
	return r.cli.runWithToken([]string{"rotate", "token", username})
}

// SelfToken rotates the current user's token
func (r *RotateCommands) SelfToken() ([]byte, error) {
	return r.cli.runWithToken([]string{"rotate", "token"})
}

// RestoreCommands provides database restoration operations
type RestoreCommands struct {
	cli *CLIRunner
}

// Restore returns restoration commands
func (c *CLIRunner) Restore() *RestoreCommands {
	return &RestoreCommands{cli: c}
}

// Database restores the entire database
func (r *RestoreCommands) Database() ([]byte, error) {
	return r.cli.runWithToken([]string{"restore", "database"})
}

// DatabaseWithConfirmation restores database with interactive confirmation
func (r *RestoreCommands) DatabaseWithConfirmation(confirm string) ([]byte, error) {
	argsWithToken := append([]string{"restore", "database"}, "--token", r.cli.env.AdminToken())
	return r.cli.env.RunRawCommand(argsWithToken, r.cli.env.CleanEnvironment(), confirm)
}

// Version shows version information
func (c *CLIRunner) Version() ([]byte, error) {
	return c.env.RunRawCommand([]string{"version"}, nil, "")
}

// Help shows help for a command
func (c *CLIRunner) Help(command ...string) ([]byte, error) {
	args := append(command, "--help")
	return c.env.RunRawCommand(args, nil, "")
}
