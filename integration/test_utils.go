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
package main

import "strings"

// extractTokenFromOutput parses the admin token from first-run output
// This is shared utility used by multiple test files to avoid duplication
func extractTokenFromOutput(output string) string {
	lines := strings.SplitSeq(output, "\n")
	foundTokenSection := false
	for outputLine := range lines {
		// Look for the new token section header
		if strings.Contains(outputLine, "🔑 Your authentication token:") {
			foundTokenSection = true
			continue
		}
		// If we found the token section, the next non-empty line should be the token
		if foundTokenSection && strings.TrimSpace(outputLine) != "" {
			return strings.TrimSpace(outputLine)
		}

		// Fallback: also check for old format "Token:" for backwards compatibility
		if strings.Contains(outputLine, "Token:") {
			fields := strings.Fields(outputLine)
			if len(fields) > 1 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}
