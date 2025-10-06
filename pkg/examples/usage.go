/*/*/*

Copyright © 2025 Ian Shuley



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.Copyright © 2025 Ian ShuleyCopyright © 2025 Ian Shuley

You may obtain a copy of the License at



	http://www.apache.org/licenses/LICENSE-2.0

Licensed under the Apache License, Version 2.0 (the "License");Licensed under the Apache License, Version 2.0 (the "License");

Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,you may not use this file except in compliance with the License.you may not use this file except in compliance with the License.

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions andYou may obtain a copy of the License atYou may obtain a copy of the License at

limitations under the License.

*/



// Package examples shows how external tools would use the simple-secrets domain interfaces	http://www.apache.org/licenses/LICENSE-2.0	http://www.apache.org/licenses/LICENSE-2.0

//

// The simple-secrets CLI now uses a domain-driven architecture with clean interfaces:

//

// - pkg/secrets.Store: Core secret storage and retrieval operationsUnless required by applicable law or agreed to in writing, softwareUnless required by applicable law or agreed to in writing, software

// - pkg/users.Store: User management and authentication operations

// - pkg/auth.AuthService: Role-based access control and token managementdistributed under the License is distributed on an "AS IS" BASIS,distributed under the License is distributed on an "AS IS" BASIS,

// - pkg/rotation.Service: Backup, restore, and rotation operations

//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

// External integrations can import these interfaces and compose them as needed.

// The internal/platform package shows how to wire these together into a complete system.See the License for the specific language governing permissions andSee the License for the specific language governing permissions and

//

// Example integration patterns:limitations under the License.limitations under the License.

// - Ansible fact gathering: Use secrets.Store for read-only secret access

// - Monitoring agents: Use secrets.Store with appropriate authentication*/*/

// - API servers: Use full platform.Platform for complete functionality

// - Backup tools: Use rotation.Service for backup/restore operations

//

// For concrete implementation examples, see:// Package examples shows how external tools would use the simple-secrets domain interfaces// Package examples shows how external tools would use the simple-secrets domain interfaces

// - internal/platform/platform.go: Shows how to compose all services

// - cmd/: Shows how to use the platform for CLI operations////

// - integration/: Shows end-to-end usage patterns

package examples// The simple-secrets CLI now uses a domain-driven architecture with clean interfaces:// The simple-secrets CLI now uses a domain-driven architecture with clean interfaces:

////

// - pkg/secrets.Store: Core secret storage and retrieval operations// - pkg/secrets.Store: Core secret storage and retrieval operations

// - pkg/users.Store: User management and authentication operations  // - pkg/users.Store: User management and authentication operations

// - pkg/auth.AuthService: Role-based access control and token management// - pkg/auth.AuthService: Role-based access control and token management

// - pkg/rotation.Service: Backup, restore, and rotation operations// - pkg/rotation.Service: Backup, restore, and rotation operations

////

// External integrations can import these interfaces and compose them as needed.// External integrations can import these interfaces and compose them as needed.

// The internal/platform package shows how to wire these together into a complete system.// The internal/platform package shows how to wire these together into a complete system.

////

// Example integration patterns:// Example integration patterns:

// - Ansible fact gathering: Use secrets.Store for read-only secret access// - Ansible fact gathering: Use secrets.Store for read-only secret access

// - Monitoring agents: Use secrets.Store with appropriate authentication// - Monitoring agents: Use secrets.Store with appropriate authentication

// - API servers: Use full platform.Platform for complete functionality// - API servers: Use full platform.Platform for complete functionality

// - Backup tools: Use rotation.Service for backup/restore operations// - Backup tools: Use rotation.Service for backup/restore operations

//package examples

// For concrete implementation examples, see:

// - internal/platform/platform.go: Shows how to compose all services// AnsibleFactsGatherer demonstrates read-only access for ansible fact gathering

// - cmd/: Shows how to use the platform for CLI operationstype AnsibleFactsGatherer struct {

// - integration/: Shows end-to-end usage patterns	reader api.SecretReader

package examples}

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
