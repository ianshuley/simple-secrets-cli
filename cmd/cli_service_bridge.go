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
	"errors"
	"fmt"
	"os"
	"simple-secrets/internal"
	"simple-secrets/internal/platform"

	"github.com/spf13/cobra"
)

// CLIServiceHelper provides a bridge between the platform and CLI commands
// Provides backward compatibility while using the new platform architecture
type CLIServiceHelper struct {
	// Deprecated: Legacy field for backward compatibility
	service *internal.Service
}

// NewCLIServiceHelper creates a new CLI service helper with composable operations
func NewCLIServiceHelper() (*CLIServiceHelper, error) {
	options := []internal.ServiceOption{
		internal.WithStorageBackend(internal.NewFilesystemBackend()),
	}

	// Check for test/config directory override (needed for test isolation)
	if configDir := getConfigDirOverride(); configDir != "" {
		options = append(options, internal.WithConfigDir(configDir))
	}

	service, err := internal.NewService(options...)
	if err != nil {
		// Handle first-run required with helpful message
		if errors.Is(err, internal.ErrFirstRunRequired) {
			return nil, fmt.Errorf("%s", internal.GetFirstRunMessage())
		}
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return &CLIServiceHelper{
		service: service,
	}, nil
}

// AuthenticateCommand handles authentication for cobra commands
// Returns (user, userStore, error) compatible with existing CLI code
func (csh *CLIServiceHelper) AuthenticateCommand(cmd *cobra.Command, needWrite bool) (*internal.User, *internal.UserStore, error) {
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, nil, err
	}

	// Resolve the token first (CLI responsibility, not service responsibility)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return nil, nil, err
	}

	// Use focused auth operations
	user, err := csh.service.Auth().ValidateToken(resolvedToken)
	if err != nil {
		return nil, nil, err
	}

	// Check permissions
	if err := csh.service.Auth().ValidateAccess(resolvedToken, needWrite); err != nil {
		return nil, nil, err
	}

	// For CLI compatibility, get UserStore through service layer
	userStore := csh.service.GetUserStore()
	if userStore == nil {
		return nil, nil, fmt.Errorf("failed to get user store from service")
	}

	return user, userStore, nil
}

// AuthenticateToken handles authentication with a direct token (for custom parsing like put command)
// Returns (user, userStore, error) compatible with existing CLI code
func (csh *CLIServiceHelper) AuthenticateToken(token string, needWrite bool) (*internal.User, *internal.UserStore, error) {
	// Resolve the token first (CLI responsibility)
	resolvedToken, err := internal.ResolveToken(token)
	if err != nil {
		return nil, nil, err
	}

	// Use focused auth operations
	user, err := csh.service.Auth().ValidateToken(resolvedToken)
	if err != nil {
		return nil, nil, err
	}

	// Check permissions
	if err := csh.service.Auth().ValidateAccess(resolvedToken, needWrite); err != nil {
		return nil, nil, err
	}

	// For CLI compatibility, get UserStore through service layer
	userStore := csh.service.GetUserStore()
	if userStore == nil {
		return nil, nil, fmt.Errorf("failed to get user store from service")
	}

	return user, userStore, nil
}

// GetService returns the underlying service for advanced operations
func (csh *CLIServiceHelper) GetService() *internal.Service {
	return csh.service
}

// Global CLI service helper instance (initialized once)
var globalCLIHelper *CLIServiceHelper

// GetCLIServiceHelper returns the global CLI service helper, initializing it if necessary
func GetCLIServiceHelper() (*CLIServiceHelper, error) {
	if globalCLIHelper == nil {
		helper, err := NewCLIServiceHelper()
		if err != nil {
			return nil, err
		}
		globalCLIHelper = helper
	}
	return globalCLIHelper, nil
}

// getConfigDirOverride checks for config directory override from environment
// This ensures test isolation works properly
func getConfigDirOverride() string {
	return os.Getenv("SIMPLE_SECRETS_CONFIG_DIR")
}

// getPlatformFromCommand retrieves the platform from command context
func getPlatformFromCommand(cmd *cobra.Command) (*platform.Platform, error) {
	ctx := cmd.Context()
	if ctx == nil {
		return nil, fmt.Errorf("command context not available")
	}

	app, err := platform.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("platform not available in context: %w", err)
	}

	return app, nil
}

// authenticateWithPlatform handles authentication using platform services
// Returns the UserContext from auth domain
func authenticateWithPlatform(cmd *cobra.Command, needWrite bool) (*internal.User, error) {
	// Get platform from command context
	app, err := getPlatformFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Get token from command
	token, err := resolveTokenFromCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Use platform auth service
	ctx := cmd.Context()
	userContext, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check write permissions if needed
	if needWrite {
		if !userContext.CanWrite() {
			return nil, fmt.Errorf("write permission required")
		}
	}

	// Convert to legacy internal.User for compatibility
	legacyUser := &internal.User{
		Username:  userContext.Username,
		Role:      internal.Role(userContext.Role),
		TokenHash: userContext.TokenHash,
	}

	return legacyUser, nil
}

// authenticateTokenWithPlatform handles authentication with direct token using platform services
func authenticateTokenWithPlatform(ctx context.Context, app *platform.Platform, token string, needWrite bool) (*internal.User, error) {
	// Use platform auth service
	userContext, err := app.Auth.Authenticate(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check write permissions if needed
	if needWrite {
		if !userContext.CanWrite() {
			return nil, fmt.Errorf("write permission required")
		}
	}

	// Convert to legacy internal.User for compatibility
	legacyUser := &internal.User{
		Username:  userContext.Username,
		Role:      internal.Role(userContext.Role),
		TokenHash: userContext.TokenHash,
	}

	return legacyUser, nil
}
