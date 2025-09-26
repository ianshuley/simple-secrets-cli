# TODO List

## ✅ RESOLVED - Critical Bug Fixed

- [x] **FIXED: nil pointer panic in put command after rotation**
  - **Root Cause**: Race condition between `Get/Put` operations and master key rotation
  - **Solution 1**: Eliminated double-locking pattern in `Get()` and `DecryptBackup()` methods
  - **Solution 2**: Moved encryption inside write lock in `Put()` method to prevent stale key usage
  - **Solution 3**: Removed redundant `backupExistingSecret(nil, key)` call in CLI layer
  - **Result**: All rotation tests now pass, race conditions eliminated
  - **Validation**: `rotation_restore_test.go` uncommented and passes completely

## Platform Integration Readiness (Future v2.0 API Development)

### Business Logic Extraction

- [x] **Extract CLI-coupled logic**: ✅ COMPLETED - Service layer created with focused interfaces
- [x] **Service layer**: ✅ COMPLETED - Created `Service` with `SecretOperations`, `AuthOperations`, `UserOperations` interfaces
- [x] **Configuration abstraction**: ✅ COMPLETED - `ServiceConfig` with functional options (`WithStorageBackend`, `WithConfigDir`)
- [ ] **Migrate remaining CLI commands**: Update `put`, `delete`, `disable`, `enable`, `rotate`, `create_user`, `restore` to use new service layer
- [ ] **Error abstraction**: Standardize error types for consistent API responses

### Advanced Features

- [ ] **Context integration**: Add `context.Context` to all core operations for cancellation
- [ ] **Dependency injection**: Allow injection of storage backend implementations
- [ ] **Alternative storage backends**: S3, database implementations of StorageBackend interface
- [ ] **API testing preparation**: Design test patterns that will work for future HTTP API

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
- [ ] **Configuration validation**: Validate config files on startup with helpful errors
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

Start on API development using abstracted interfaces

## Platform Command

Add `platform` command teasing what's coming next: The Simple Secrets Platform is coming soon! Visit <https://simple-secrets.io> to join the waitlist or learn more

---
