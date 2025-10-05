# TODO List

## 🚧 IN PROGRESS: Multi-Token Per User Support

### Current Status: Domain Layer Complete, CLI Commands Needed

**✅ Completed Infrastructure:**

- ✅ **Domain models**: `User.AddToken()`, `User.RemoveToken()`, `User.ListTokens()` methods implemented
- ✅ **Service interfaces**: `users.Store.AddToken()` method available in `pkg/users/interfaces.go`
- ✅ **Storage layer**: Token storage and retrieval working in file-based repository
- ✅ **Authentication**: Multi-token validation integrated into auth service

**🚧 Missing CLI Commands:**

- [ ] **`token create`**: Add new named token for current user (or other users if admin)
- [ ] **`token list`**: List all tokens for current user (or specific user if admin) with metadata (name, created date, last used)
- [ ] **`token revoke`**: Remove specific named token (own tokens, or any user's tokens if admin)
- [ ] **`token info`**: Show details about current token (name, permissions, expiry if applicable)

### Implementation Plan

1. **Create `cmd/token.go`** with subcommands:

   ```bash
   # Self-service (any user)
   simple-secrets token create --name "ci-server" --token <current-token>
   simple-secrets token list --token <current-token>
   simple-secrets token revoke --name "ci-server" --token <current-token>
   simple-secrets token info --token <current-token>

   # Admin operations (admin users only)
   simple-secrets token create --name "backup-job" --for-user "devops-user" --token <admin-token>
   simple-secrets token list --for-user "devops-user" --token <admin-token>
   simple-secrets token revoke --name "backup-job" --for-user "devops-user" --token <admin-token>
   simple-secrets token list --all-users --token <admin-token>  # Security audit view
   ```

2. **Enhanced User Experience:**
   - Named tokens for easier management ("production-deploy", "ci-server", "laptop")
   - Token metadata display (creation date, last used, permissions)
   - Bulk token operations for cleanup
   - Token rotation with name preservation

3. **Security Considerations & RBAC:**
   - **Regular users**: Can only manage their own tokens (no cross-user access)
   - **Admin users**: Can create/list/revoke tokens for any user (essential for enterprise)
   - **Security incident response**: Admins can immediately revoke compromised user tokens
   - **Employee offboarding**: Admins can revoke all tokens for departing employees
   - **Audit capabilities**: Admins can list all tokens across all users for security reviews
   - **Immediate invalidation**: Token revocation immediately blocks access across all operations
   - **Audit trail**: All token operations logged with user, timestamp, and action details

## 🚀 Platform Ready: Future v2.0 API Development

### Ready-to-Build Platform Features

**The CLI now provides a complete, production-ready foundation that makes building APIs and platforms trivial:**

```go
// Any new platform can simply import and extend:
import (
    "github.com/youruser/simple-secrets-cli/pkg/secrets"
    "github.com/youruser/simple-secrets-cli/pkg/auth"
    "github.com/youruser/simple-secrets-cli/pkg/users"
    "github.com/youruser/simple-secrets-cli/pkg/rotation"
)

// Use the same business logic in different transports:
type HTTPAPIServer struct {
    secrets  secrets.Store
    auth     auth.AuthService
    users    users.Store
    rotation rotation.Service
}

type GRPCServer struct {
    secrets  secrets.Store
    auth     auth.AuthService
    // ... same interfaces
}
```

### Future Enhancement Ideas (v2.0+)

- [ ] **HTTP/REST API Server**: JSON API using existing domain services
- [ ] **GraphQL API**: Single endpoint with full type safety
- [ ] **Web Dashboard**: React/Vue frontend for secret management
- [ ] **gRPC Services**: High-performance RPC interface
- [ ] **Ansible Plugin**: Native Ansible integration for automation
- [ ] **Terraform Provider**: Infrastructure-as-code integration
- [ ] **GitOps Integration**: Automated secret sync with version control
- [ ] **Cloud Storage Backends**: S3, Azure Blob, GCS repository implementations
- [ ] **Database Backends**: PostgreSQL, MySQL repository implementations
- [ ] **SSO/SAML Integration**: Enterprise authentication extensions
- [ ] **Multi-tenancy**: Organizational isolation and management
- [ ] **Audit Logging**: Comprehensive operation tracking
- [ ] **Key Escrow**: Enterprise key recovery mechanisms
- [ ] **Policy Engine**: Fine-grained access control rules
- [ ] **Secrets Scanning**: Detect secrets in code repositories

### Current Architecture Benefits

**What Makes Platform Development Easy:**

1. **Clean Domain Boundaries**: Each domain (`secrets`, `users`, `auth`, `rotation`) has clear responsibilities
2. **Public Interfaces**: All business operations available through `pkg/` interfaces
3. **Private Implementations**: Internal details hidden in `internal/`, easy to swap/extend
4. **Service Composition**: `internal/platform/` shows how to wire services together
5. **Context-Aware**: All operations support cancellation, timeouts, and request tracing
6. **Repository Pattern**: Storage abstracted behind interfaces, easy to add new backends
7. **Test Coverage**: Comprehensive test suite validates business logic independent of transport
8. **Proven in Production**: CLI validates all business logic works correctly

### Implementation Guidelines for Platform Extensions

**Best Practices:**

- Import only `pkg/` interfaces, never `internal/` packages
- Use `internal/platform.New()` pattern for service composition
- Implement transport-specific concerns (HTTP headers, gRPC metadata, etc.) in your platform
- Leverage existing test patterns from `integration/` directory
- Follow repository pattern for new storage backends
- Use `context.Context` for all domain operations
- Handle domain errors appropriately for your transport (HTTP status codes, gRPC codes, etc.)

---

## Production Polish (Future Enhancements)

### Documentation & User Experience

- [ ] **Installation instructions**: Add package manager installs (homebrew, apt, etc.)
- [ ] **Architecture Guide**: Detailed guide for platform extension developers
- [ ] **Integration Examples**: Sample implementations for common platforms
- [ ] **Troubleshooting guide**: Common issues and solutions section in README
- [ ] **Security guide**: Best practices for production deployment

### Performance & Monitoring

- [ ] **Performance optimization**: Profile common operations for bottlenecks
- [ ] **Memory usage optimization**: Ensure reasonable footprint for large secret stores
- [ ] **Metrics & Observability**: Prometheus metrics, structured logging
- [ ] **Health checks**: Endpoint for monitoring systems

### Enterprise Features

- [ ] **Audit Logging**: Comprehensive operation tracking and compliance
- [ ] **Backup encryption**: Encrypt backup files with separate keys
- [ ] **Compliance modes**: FIPS, Common Criteria, SOX compliance
- [ ] **Rate limiting**: Protect against abuse and DoS
- [ ] **Circuit breakers**: Resilience patterns for service calls

## Development Notes

**Current Status**: The CLI is now **platform-ready** with a solid domain-driven architecture. All major refactoring work is complete and the foundation is stable for building additional interfaces and platforms.

**Next Major Milestone**: When ready to build the platform, create a new repository that imports the `pkg/` interfaces and implements the desired transport layer (HTTP, gRPC, GraphQL, etc.)

## Platform Command

- [ ] **Add platform teaser command**:
  - Create `cmd/platform.go` with subcommands: `status`, `info`, `roadmap`
  - `simple-secrets platform status`: Check if platform API is available locally
  - `simple-secrets platform info`: Show platform features and benefits
  - `simple-secrets platform roadmap`: Display development timeline and features
  - Include call-to-action: "The Simple Secrets Platform is coming soon! Visit <https://simple-secrets.io> to join the waitlist"
  - Add configuration option to disable platform commands for enterprise deployments

---
