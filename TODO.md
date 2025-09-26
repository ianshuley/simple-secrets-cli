# TODO List

## Platform Integration Readiness (Future v2.0 API Development)

### Business Logic Extraction

- [ ] **Error abstraction**: Standardize error types for consistent API responses
  - Create `pkg/errors` package with structured error types (AuthError, ValidationError, StorageError)
  - Implement error wrapping with context preservation for API responses
  - Update service layer interfaces to return structured errors instead of generic `error`
  - Add HTTP status code mapping for future API development

### Advanced Features

- [ ] **Context integration**: Add `context.Context` to all core operations for cancellation
  - Update all service layer interfaces (`SecretOperations`, `AuthOperations`, `UserOperations`) to accept `context.Context` as first parameter
  - Modify `SecretsStore` and `UserStore` methods to support context cancellation
  - Add timeout handling in CLI commands for long-running operations (backup, restore, rotation)
- [ ] **Dependency injection**: Allow injection of storage backend implementations
  - Current: `ServiceConfig` with `WithStorageBackend()` functional option exists
  - Extend: Add `WithUserStore()`, `WithSecretsStore()` options for complete DI
  - Create factory interfaces for store creation with different backends
- [ ] **Alternative storage backends**: S3, database implementations of StorageBackend interface
  - Current: `StorageBackend` interface exists with filesystem implementation
  - Add: `S3StorageBackend` implementing same interface (ReadFile, WriteFile, etc.)
  - Add: `DatabaseStorageBackend` for PostgreSQL/SQLite storage
  - Maintain encryption at application layer, storage backends handle persistence only
- [ ] **API testing preparation**: Design test patterns that will work for future HTTP API
  - Extract testing utilities from CLI integration tests for reuse in HTTP API tests
  - Create test data factories that work for both CLI and API testing
  - Design API contract testing patterns using service layer interfaces

---

## Release Preparation

### Documentation & User Experience

- [ ] **Installation instructions**: Add package manager installs (homebrew, apt, etc.)
- [ ] **Migration guide**: Document upgrading from dev builds to v1.0
- [ ] **Troubleshooting guide**: Common issues and solutions section in README
- [ ] **Security guide**: Best practices for production deployment
- [ ] **Backup/recovery guide**: How to handle disaster scenarios

### Polish & Production Readiness

- [ ] **Error message audit**: Review all error messages for clarity and helpfulness
- [ ] **Logging system**: Add optional verbose/debug logging for troubleshooting
  - Add `--verbose` and `--debug` global flags to root command
  - Implement structured logging using Go's slog package
  - Log levels: ERROR (always), WARN (default), INFO (verbose), DEBUG (debug)
  - Key logging points: auth attempts, file operations, encryption/decryption, backup/restore progress
  - Ensure no secrets are logged, even in debug mode
- [ ] **Configuration validation**: Validate config files on startup with helpful errors
  - Validate `config.json` schema on load in `internal/config.go`
  - Check `rotation_backup_count` is positive integer (currently only validated during rotation)
  - Validate file permissions on config directory and files (should be 0700/0600)
  - Provide specific error messages for common config issues with suggested fixes
  - Add `simple-secrets config validate` command for manual validation
- [ ] **Performance optimization**: Profile common operations (put/get/list) for bottlenecks
- [ ] **Memory usage**: Ensure reasonable memory footprint for large secret stores

### Release Infrastructure

- [ ] **GitHub releases**: Set up automated release builds with checksums
- [ ] **Package signing**: Code signing for binaries
- [ ] **Release notes template**: Standardized format for future releases
- [ ] **Backward compatibility policy**: Document what changes will/won't break compatibility
- [ ] **Versioning strategy**: Clarify semantic versioning approach

### Security Hardening

- [ ] **Security audit**: Run final security review using updated testing frameworks
- [ ] **Dependencies audit**: Review all Go dependencies for vulnerabilities
- [ ] **File permissions audit**: Ensure all created files have minimal necessary permissions
- [ ] **Input validation**: Final review of all user input validation

### User Feedback Integration

- [ ] **Beta testing**: Get feedback from a few real users before v1.0
- [ ] **Common workflows**: Document and test typical user journeys
- [ ] **Edge case documentation**: Document known limitations and workarounds


## API Development (v2.0)

- [ ] **HTTP API Server**: Build REST API using existing service layer
  - Create new repository: `simple-secrets-platform`
  - Use `pkg/api` interfaces from CLI as foundation
  - Implement HTTP handlers that delegate to service layer operations
  - JWT-based authentication using same RBAC system as CLI
  - OpenAPI/Swagger documentation generation
- [ ] **API Endpoints Design**:
  - `GET /secrets` (list), `GET /secrets/{key}` (get), `PUT /secrets/{key}` (put)
  - `DELETE /secrets/{key}` (delete), `POST /secrets/{key}/enable|disable`
  - `POST /auth/login` (token auth), `POST /users` (create), `GET /users` (list)
  - `POST /admin/backup`, `POST /admin/restore`, `POST /admin/rotate-master-key`

## Platform Command

- [ ] **Add platform teaser command**:
  - Create `cmd/platform.go` with subcommands: `status`, `info`, `roadmap`
  - `simple-secrets platform status`: Check if platform API is available locally
  - `simple-secrets platform info`: Show platform features and benefits
  - `simple-secrets platform roadmap`: Display development timeline and features
  - Include call-to-action: "The Simple Secrets Platform is coming soon! Visit <https://simple-secrets.io> to join the waitlist"
  - Add configuration option to disable platform commands for enterprise deployments

---
