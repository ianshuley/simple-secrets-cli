# TODO List

## Platform Integration Readiness (Future v2.0 API Development)

### Business Logic Extraction

- [ ] **Extract CLI-coupled logic**: Move business logic from `cmd/` package to core packages
- [ ] **Configuration abstraction**: Create injectable `Config` struct for storage paths
- [ ] **Service layer**: Create service layer that CLI and future API can both consume
- [ ] **Error abstraction**: Standardize error types for consistent API responses

### Advanced Features

- [ ] **Context integration**: Add `context.Context` to all core operations for cancellation
- [ ] **Dependency injection**: Allow injection of storage backend implementations
- [ ] **Alternative storage backends**: S3, database implementations of StorageBackend interface
- [ ] **API testing preparation**: Design test patterns that will work for future HTTP API


---

## Add Clipboard Flag

I'd like to add a --copy flag that pipes whatever information is being retrieved to the clipboard instead of to the console.

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
