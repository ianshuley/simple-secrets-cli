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

import "fmt"

// PrintFirstRunMessage prints a clear message for first-run admin creation.
func PrintFirstRunMessage() {
	fmt.Println("\nFirst run detected. Default admin user created.")
	fmt.Println("To use your new token, re-run this command in one of these ways:")
	fmt.Println("  --token <your-token> (as a flag)")
	fmt.Println("  SIMPLE_SECRETS_TOKEN=<your-token> ./simple-secrets ... (as an env var)")
	fmt.Println("  or place it in ~/.simple-secrets/config.json as { \"token\": \"<your-token>\" }")
	fmt.Println("\nIf creating config.json manually, ensure it has secure permissions:")
	fmt.Println("  chmod 600 ~/.simple-secrets/config.json")
}

// PrintTokenAtEnd displays the token at the end of the setup flow for better UX
func PrintTokenAtEnd(token string) {
	fmt.Printf("\n🔑 Your authentication token:\n")
	fmt.Printf("   %s\n", token)
	fmt.Println("\n📋 Please store this token securely in your password manager.")
	fmt.Println("   It will not be shown again!")
}
