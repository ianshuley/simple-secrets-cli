# TODO List

## CRITICAL - Fix Concurrency Race Condition

**Problem:** Opus AI testing framework discovered a race condition when multiple operations execute simultaneously.

**Details:** Under high concurrency (multiple goroutines performing operations like put/get/rotate simultaneously), the system exhibits non-deterministic behavior that could lead to data corruption or inconsistent state.

**Root Cause:** Likely shared state access without proper synchronization in core operations.

**Impact:** Could affect production deployments under heavy load or when automation tools perform parallel operations.

**Investigation Needed:**

- Identify which shared resources lack proper locking (secrets store, user store, file operations)
- Add appropriate mutexes or use Go's sync package for critical sections
- Test with high-concurrency scenarios to validate fix
- Consider atomic file operations vs in-memory locking strategies

**Priority:** HIGH - This affects data integrity under concurrent access patterns.

---

## Add Clipboard Flag

I'd like to add a --copy flag that pipes whatever information is being retrieved to the clipboard instead of to the console.

---

## Platform Command

Add `platform` command teasing what's coming next: The Simple Secrets Platform is coming soon! Visit <https://simple-secrets.io> to join the waitlist or learn more

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

---

## API Development

Start on API
