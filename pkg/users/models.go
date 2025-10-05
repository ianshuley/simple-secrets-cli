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

package users

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// User represents a user in the secrets management system
type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	Tokens         []*Token   `json:"tokens"` // Multi-token support
	Role           string     `json:"role"`   // Role is just a string here, auth domain handles validation
	CreatedAt      time.Time  `json:"created_at"`
	TokenRotatedAt *time.Time `json:"token_rotated_at,omitempty"` // For backward compatibility
	Disabled       bool       `json:"disabled,omitempty"`
}

// Token represents an authentication token for a user
type Token struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"` // Human-friendly name like "CI/CD Token", "Personal Token"
	Hash       string     `json:"hash"` // Hashed token value
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Disabled   bool       `json:"disabled,omitempty"`
}

// IsDisabled returns true if the user is currently disabled
func (u *User) IsDisabled() bool {
	return u.Disabled
}

// IsEnabled returns true if the user is currently enabled
func (u *User) IsEnabled() bool {
	return !u.Disabled
}

// Enable marks the user as enabled
func (u *User) Enable() {
	u.Disabled = false
}

// Disable marks the user as disabled
func (u *User) Disable() {
	u.Disabled = true
}

// GetPrimaryToken returns the first active token (for backward compatibility)
func (u *User) GetPrimaryToken() *Token {
	for _, token := range u.Tokens {
		if !token.IsDisabled() {
			return token
		}
	}
	return nil
}

// AddToken adds a new token to the user
func (u *User) AddToken(token *Token) {
	if u.Tokens == nil {
		u.Tokens = make([]*Token, 0)
	}
	u.Tokens = append(u.Tokens, token)
}

// RemoveToken removes a token by ID
func (u *User) RemoveToken(tokenID string) bool {
	for i, token := range u.Tokens {
		if token.ID == tokenID {
			u.Tokens = append(u.Tokens[:i], u.Tokens[i+1:]...)
			return true
		}
	}
	return false
}

// GetToken returns a token by ID
func (u *User) GetToken(tokenID string) *Token {
	for _, token := range u.Tokens {
		if token.ID == tokenID {
			return token
		}
	}
	return nil
}

// HasActiveTokens returns true if the user has at least one active token
func (u *User) HasActiveTokens() bool {
	for _, token := range u.Tokens {
		if !token.IsDisabled() {
			return true
		}
	}
	return false
}

// GetTokenRotationDisplay returns a formatted string for token rotation status (backward compatibility)
func (u *User) GetTokenRotationDisplay() string {
	if u.TokenRotatedAt == nil {
		return "Legacy user (no rotation info)"
	}
	return u.TokenRotatedAt.Format("2006-01-02 15:04:05")
}

// IsDisabled returns true if the token is currently disabled
func (t *Token) IsDisabled() bool {
	return t.Disabled
}

// IsEnabled returns true if the token is currently enabled
func (t *Token) IsEnabled() bool {
	return !t.Disabled
}

// IsExpired returns true if the token is expired
func (t *Token) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*t.ExpiresAt)
}

// IsActive returns true if the token is enabled and not expired
func (t *Token) IsActive() bool {
	return t.IsEnabled() && !t.IsExpired()
}

// Enable marks the token as enabled
func (t *Token) Enable() {
	t.Disabled = false
}

// Disable marks the token as disabled
func (t *Token) Disable() {
	t.Disabled = true
}

// UpdateLastUsed updates the last used timestamp
func (t *Token) UpdateLastUsed() {
	now := time.Now()
	t.LastUsedAt = &now
}

// NewUser creates a new user with the given username and role
func NewUser(username, role string) *User {
	now := time.Now()
	return &User{
		ID:        generateID(),
		Username:  username,
		Role:      role,
		CreatedAt: now,
		Tokens:    make([]*Token, 0),
		Disabled:  false,
	}
}

// NewToken creates a new token with the given name and hash
func NewToken(name, hash string) *Token {
	now := time.Now()
	return &Token{
		ID:        generateID(),
		Name:      name,
		Hash:      hash,
		CreatedAt: now,
		Disabled:  false,
	}
}

// generateID generates a random ID for users and tokens
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
