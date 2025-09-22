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
	"fmt"
	"simple-secrets/internal"

	"github.com/spf13/cobra"
)

// CLIServiceHelper provides a bridge between the service layer and CLI commands
// Uses composable operations instead of monolithic patterns
type CLIServiceHelper struct {
	service *internal.Service
}

// NewCLIServiceHelper creates a new CLI service helper with composable operations
func NewCLIServiceHelper() (*CLIServiceHelper, error) {
	// Use functional options pattern - much cleaner than config structs
	service, err := internal.NewService(
		internal.WithStorageBackend(internal.NewFilesystemBackend()),
	)
	if err != nil {
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

	// For CLI compatibility, we need to return a UserStore
	// This is a bridge concern, not a service concern
	userStore, _, _, err := internal.LoadUsers()
	if err != nil {
		return nil, nil, err
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

	// For CLI compatibility, return UserStore
	userStore, _, _, err := internal.LoadUsers()
	if err != nil {
		return nil, nil, err
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

// RBACGuardV2 is a new version of RBACGuard that uses the service layer
func RBACGuardV2(needWrite bool, cmd *cobra.Command) (*internal.User, *internal.UserStore, error) {
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize service layer: %w", err)
	}

	return helper.AuthenticateCommand(cmd, needWrite)
}

// AuthenticateWithTokenV2 is a new version of AuthenticateWithToken that uses the service layer
func AuthenticateWithTokenV2(needWrite bool, token string) (*internal.User, *internal.UserStore, error) {
	helper, err := GetCLIServiceHelper()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize service layer: %w", err)
	}

	return helper.AuthenticateToken(token, needWrite)
}
