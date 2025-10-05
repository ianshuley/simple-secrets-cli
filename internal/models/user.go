/*package models

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

// Package models provides data models for simple-secrets
package models

import (
	"slices"
	"time"
)

// Role represents a user role with specific permissions
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleReader Role = "reader"
)

// User represents a user in the system with authentication and authorization info
type User struct {
	Username       string     `json:"username"`
	TokenHash      string     `json:"token_hash"` // SHA-256 hash, base64-encoded
	Role           Role       `json:"role"`
	TokenRotatedAt *time.Time `json:"token_rotated_at,omitempty"` // When token was last rotated
}

// RolePermissions maps roles to their allowed permissions
type RolePermissions map[Role][]string

// Has checks if a role has a specific permission
func (rp RolePermissions) Has(role Role, perm string) bool {
	perms := rp[role]
	return slices.Contains(perms, perm)
}

// Can checks if the user has a specific permission
func (u *User) Can(perm string, perms RolePermissions) bool {
	return perms.Has(u.Role, perm)
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsReader returns true if the user has reader role
func (u *User) IsReader() bool {
	return u.Role == RoleReader
}
