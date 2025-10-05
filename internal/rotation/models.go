/*package rotation

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

package rotation

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// BackupInfo represents information about a backup file
type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
}

// String returns a formatted string representation of the backup
func (b *BackupInfo) String() string {
	if b.Name == "" || b.Timestamp.IsZero() {
		return "Invalid backup info"
	}
	return fmt.Sprintf("%s (%s, %s ago)",
		b.Name,
		b.Timestamp.Format("2006-01-02 15:04:05"),
		b.Age())
}

// Age returns a human-readable age of the backup
func (b *BackupInfo) Age() string {
	duration := time.Since(b.Timestamp)

	if duration < time.Minute {
		return "less than a minute"
	}
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}

	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

// IsRecent returns true if the backup was created within the last hour
func (b *BackupInfo) IsRecent() bool {
	return time.Since(b.Timestamp) <= time.Hour
}

// BaseName extracts the base name from a timestamped backup filename
// Example: "secrets_backup_20240101_120000.json" -> "secrets_backup"
func (b *BackupInfo) BaseName() string {
	name := b.Name

	// Remove file extension if present
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Look for timestamp pattern at the end: _YYYYMMDD_HHMMSS
	parts := strings.Split(name, "_")
	if len(parts) >= 3 {
		// Check if the last two parts look like date and time
		dateTimePattern := len(parts) >= 2
		if dateTimePattern {
			lastPart := parts[len(parts)-1]
			secondLastPart := parts[len(parts)-2]

			// Simple check: date should be 8 digits, time should be 6 digits
			if len(secondLastPart) == 8 && len(lastPart) == 6 {
				// Remove the timestamp parts
				return strings.Join(parts[:len(parts)-2], "_")
			}
		}
	}

	// If no timestamp pattern found, return the whole name
	return name
}
