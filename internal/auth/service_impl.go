/*package auth

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

package auth

import (
	"context"
	"crypto/subtle"

	"simple-secrets/pkg/auth"
	"simple-secrets/pkg/crypto"
	"simple-secrets/pkg/users"
)

// ServiceImpl implements the auth.AuthService interface
type ServiceImpl struct {
	userStore       users.Store
	rolePermissions auth.RolePermissions
}

// NewService creates a new auth service with the provided user store and role permissions
func NewService(userStore users.Store, rolePermissions auth.RolePermissions) auth.AuthService {
	return &ServiceImpl{
		userStore:       userStore,
		rolePermissions: rolePermissions,
	}
}

// NewServiceWithDefaults creates a new auth service with default role permissions
func NewServiceWithDefaults(userStore users.Store) auth.AuthService {
	return NewService(userStore, auth.NewDefaultRolePermissions())
}

// ValidateToken validates a token and returns the associated user context
func (s *ServiceImpl) ValidateToken(ctx context.Context, token string) (*auth.UserContext, error) {
	if token == "" {
		return nil, auth.NewInvalidTokenError("token cannot be empty")
	}

	// Get user by token value
	user, err := s.userStore.GetByToken(ctx, token)
	if err != nil {
		return nil, auth.NewInvalidTokenError("token not found or invalid")
	}

	// Check if user is disabled
	if user.Disabled {
		return nil, auth.NewInvalidTokenError("user account is disabled")
	}

	// Parse user role
	role, err := auth.ParseRole(user.Role)
	if err != nil {
		return nil, auth.NewInvalidRoleError(user.Role)
	}

	// Create token hash for user context
	tokenHash := s.HashToken(token)

	// Create user context with permissions
	userContext := auth.NewUserContext(user.Username, role, tokenHash, s.rolePermissions)
	
	return userContext, nil
}

// ValidateTokenHash validates a token hash and returns the associated user context  
func (s *ServiceImpl) ValidateTokenHash(ctx context.Context, tokenHash string) (*auth.UserContext, error) {
	if tokenHash == "" {
		return nil, auth.NewInvalidTokenError("token hash cannot be empty")
	}

	// Find user by iterating through all users and checking token hashes
	allUsers, err := s.userStore.List(ctx)
	if err != nil {
		return nil, auth.NewInvalidTokenError("failed to lookup user")
	}

	for _, user := range allUsers {
		// Check if user is disabled first
		if user.Disabled {
			continue
		}

		// Check each token's hash
		for _, token := range user.Tokens {
			if token.Disabled {
				continue
			}

			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(token.Hash)) == 1 {
				// Parse user role
				role, err := auth.ParseRole(user.Role)
				if err != nil {
					return nil, auth.NewInvalidRoleError(user.Role)
				}

				// Create user context with permissions
				userContext := auth.NewUserContext(user.Username, role, tokenHash, s.rolePermissions)
				
				return userContext, nil
			}
		}
	}

	return nil, auth.NewInvalidTokenError("token hash not found or invalid")
}

// HasPermission checks if a user has a specific permission
func (s *ServiceImpl) HasPermission(ctx context.Context, user *auth.UserContext, permission auth.Permission) bool {
	return s.rolePermissions.Has(user.Role, permission)
}

// RequirePermission checks permission and returns error if not granted
func (s *ServiceImpl) RequirePermission(ctx context.Context, user *auth.UserContext, permission auth.Permission) error {
	if !s.HasPermission(ctx, user, permission) {
		return auth.NewPermissionDeniedError(permission, "user does not have required permission")
	}
	return nil
}

// GetPermissions returns all permissions for a role
func (s *ServiceImpl) GetPermissions(ctx context.Context, role auth.Role) []auth.Permission {
	return s.rolePermissions.GetPermissions(role)
}

// ValidateRole checks if a role is valid
func (s *ServiceImpl) ValidateRole(ctx context.Context, role auth.Role) error {
	return s.rolePermissions.ValidateRole(role)
}

// GetRolePermissions returns the permissions for a specific role
func (s *ServiceImpl) GetRolePermissions(ctx context.Context, role auth.Role) ([]auth.Permission, error) {
	if err := s.ValidateRole(ctx, role); err != nil {
		return nil, err
	}
	return s.GetPermissions(ctx, role), nil
}

// ListRoles returns all available roles
func (s *ServiceImpl) ListRoles(ctx context.Context) ([]auth.Role, error) {
	return s.rolePermissions.GetAllRoles(), nil
}

// Authenticate validates credentials and returns user context
func (s *ServiceImpl) Authenticate(ctx context.Context, token string) (*auth.UserContext, error) {
	return s.ValidateToken(ctx, token)
}

// Authorize checks if user can perform specific action
func (s *ServiceImpl) Authorize(ctx context.Context, user *auth.UserContext, permission auth.Permission) error {
	return s.RequirePermission(ctx, user, permission)
}

// HashToken creates a secure hash of a token for storage
func (s *ServiceImpl) HashToken(token string) string {
	return crypto.HashToken(token)
}