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

package platform

import (
	"context"
	"fmt"
	"time"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey int

const (
	// platformKey is the context key for storing Platform instances
	platformKey contextKey = iota
)

// WithPlatform adds a Platform instance to the context.
// This allows CLI commands to access the platform services through context.
func WithPlatform(ctx context.Context, platform *Platform) context.Context {
	return context.WithValue(ctx, platformKey, platform)
}

// FromContext retrieves the Platform instance from the context.
// Returns an error if no platform is found in the context.
func FromContext(ctx context.Context) (*Platform, error) {
	platform, ok := ctx.Value(platformKey).(*Platform)
	if !ok || platform == nil {
		return nil, fmt.Errorf("no platform found in context")
	}
	return platform, nil
}

// MustFromContext retrieves the Platform instance from the context.
// Panics if no platform is found in the context.
// Use this in CLI commands where platform presence is guaranteed.
func MustFromContext(ctx context.Context) *Platform {
	platform, err := FromContext(ctx)
	if err != nil {
		panic(fmt.Sprintf("platform not found in context: %v", err))
	}
	return platform
}

// WithTimeout creates a context with a timeout and includes the platform.
// This is a convenience function for CLI operations that need both timeout
// and platform access.
func WithTimeout(ctx context.Context, timeout string) (context.Context, context.CancelFunc, error) {
	// Parse timeout duration
	duration, err := parseDuration(timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, duration)

	// Preserve platform if it exists in the original context
	if platform, err := FromContext(ctx); err == nil {
		timeoutCtx = WithPlatform(timeoutCtx, platform)
	}

	return timeoutCtx, cancel, nil
}

// parseDuration is a helper to parse timeout strings
func parseDuration(timeout string) (time.Duration, error) {
	return time.ParseDuration(timeout)
}

// Background returns a background context with the platform attached.
// This is useful for operations that need to run independently of
// the current request context but still need platform access.
func Background(platform *Platform) context.Context {
	return WithPlatform(context.Background(), platform)
}

// TODO creates a TODO context with the platform attached.
// This is useful during development when you need a quick context
// with platform access.
func TODO(platform *Platform) context.Context {
	return WithPlatform(context.TODO(), platform)
}
