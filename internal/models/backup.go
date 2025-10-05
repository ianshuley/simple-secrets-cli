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

package models

import (
	"fmt"
	"strings"
	"time"
)

// BackupInfo contains metadata about a backup file
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	IsValid   bool      `json:"is_valid"`
}

// String returns a human-readable representation of the backup info
func (b *BackupInfo) String() string {
	status := "✓"
	if !b.IsValid {
		status = "✗"
	}
	return fmt.Sprintf("%s %s (%s)", status, b.Name, b.Timestamp.Format("2006-01-02 15:04:05"))
}

// Age returns how long ago the backup was created
func (b *BackupInfo) Age() time.Duration {
	return time.Since(b.Timestamp)
}

// IsRecent returns true if the backup was created within the specified duration
func (b *BackupInfo) IsRecent(within time.Duration) bool {
	return b.Age() <= within
}

// BaseName returns the backup name without the timestamp suffix
func (b *BackupInfo) BaseName() string {
	// Remove timestamp suffix like "_20250102_150405"
	parts := strings.Split(b.Name, "_")
	if len(parts) >= 3 {
		// Keep everything except the last two parts (date_time)
		return strings.Join(parts[:len(parts)-2], "_")
	}
	return b.Name
}
