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
	for line := range lines {
		if strings.Contains(line, "Token:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[len(fields)-1]
			}
		}
	}
	return ""
}
