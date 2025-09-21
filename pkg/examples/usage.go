/*package examples

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

// Package examples shows how external tools would use the simple-secrets API interfaces
package examples

import (
	"fmt"
	"simple-secrets/pkg/api"
)

// AnsibleFactsGatherer demonstrates read-only access for ansible fact gathering
type AnsibleFactsGatherer struct {
	reader api.SecretReader
}

// NewAnsibleFactsGatherer creates a new facts gatherer with read-only access
func NewAnsibleFactsGatherer(reader api.SecretReader) *AnsibleFactsGatherer {
	return &AnsibleFactsGatherer{reader: reader}
}

// GatherDatabaseFacts gathers database-related secrets for ansible
func (afg *AnsibleFactsGatherer) GatherDatabaseFacts() (map[string]string, error) {
	facts := make(map[string]string)

	// Get all available secrets
	keys := afg.reader.List()

	// Filter for database-related secrets
	for _, key := range keys {
		if len(key) > 3 && key[:3] == "db_" {
			if afg.reader.IsEnabled(key) {
				value, err := afg.reader.Get(key)
				if err != nil {
					return nil, fmt.Errorf("failed to get %s: %w", key, err)
				}
				facts[key] = value
			}
		}
	}

	return facts, nil
}

// AnsibleSecretManager demonstrates full secret management for ansible playbooks
type AnsibleSecretManager struct {
	service api.SecretsService // Combines read and write
	auth    api.Authenticator
}

// NewAnsibleSecretManager creates a new secret manager with read/write access
func NewAnsibleSecretManager(service api.SecretsService, auth api.Authenticator) *AnsibleSecretManager {
	return &AnsibleSecretManager{service: service, auth: auth}
}

// DeploySecrets deploys a set of secrets, ensuring user has write permissions
func (asm *AnsibleSecretManager) DeploySecrets(token string, secrets map[string]string) error {
	// Authenticate user
	user, err := asm.auth.Authenticate(token)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check write permissions
	if !asm.auth.CanWrite(user) {
		return fmt.Errorf("user %s does not have write permissions", user.Username)
	}

	// Deploy secrets
	for key, value := range secrets {
		if err := asm.service.Put(key, value); err != nil {
			return fmt.Errorf("failed to deploy secret %s: %w", key, err)
		}
	}

	return nil
}

// ApiServer demonstrates how an HTTP API server would use the interfaces
type ApiServer struct {
	full api.FullService
}

// NewApiServer creates a new API server with full service access
func NewApiServer(full api.FullService) *ApiServer {
	return &ApiServer{full: full}
}

// HandleGetSecret shows how an API endpoint would handle secret retrieval
func (as *ApiServer) HandleGetSecret(token, key string) (string, error) {
	// Authenticate request
	user, err := as.full.Authenticate(token)
	if err != nil {
		return "", fmt.Errorf("unauthorized: %w", err)
	}

	// Check read permissions
	if !as.full.CanRead(user) {
		return "", fmt.Errorf("forbidden: user %s cannot read secrets", user.Username)
	}

	// Get secret
	return as.full.Get(key)
}

// HandleCreateUser shows how an API endpoint would handle user creation
func (as *ApiServer) HandleCreateUser(adminToken, username, role string) (string, error) {
	// Authenticate admin
	admin, err := as.full.Authenticate(adminToken)
	if err != nil {
		return "", fmt.Errorf("unauthorized: %w", err)
	}

	// Check admin permissions
	if !as.full.CanAdmin(admin) {
		return "", fmt.Errorf("forbidden: user %s cannot create users", admin.Username)
	}

	// Create user
	return as.full.CreateUser(username, role)
}

// HandleBackup shows how an API endpoint would handle backup operations
func (as *ApiServer) HandleBackup(adminToken, backupDir string) error {
	// Authenticate admin
	admin, err := as.full.Authenticate(adminToken)
	if err != nil {
		return fmt.Errorf("unauthorized: %w", err)
	}

	// Check admin permissions
	if !as.full.CanAdmin(admin) {
		return fmt.Errorf("forbidden: user %s cannot perform backups", admin.Username)
	}

	// Perform backup
	return as.full.Backup(backupDir)
}

// MonitoringAgent demonstrates read-only monitoring access
type MonitoringAgent struct {
	reader api.SecretReader
}

// NewMonitoringAgent creates a new monitoring agent with read-only access
func NewMonitoringAgent(reader api.SecretReader) *MonitoringAgent {
	return &MonitoringAgent{reader: reader}
}

// CheckSecretHealth checks if critical secrets are available and enabled
func (ma *MonitoringAgent) CheckSecretHealth(criticalSecrets []string) (map[string]bool, error) {
	health := make(map[string]bool)

	for _, secret := range criticalSecrets {
		enabled := ma.reader.IsEnabled(secret)
		health[secret] = enabled

		// If it's enabled, try to access it to ensure it's readable
		if enabled {
			_, err := ma.reader.Get(secret)
			if err != nil {
				health[secret] = false
			}
		}
	}

	return health, nil
}
