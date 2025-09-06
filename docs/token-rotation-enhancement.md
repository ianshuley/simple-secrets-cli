# Token Rotation Enhancement Proposal

## Current State ✅
- Admin-managed token rotation: `simple-secrets rotate token <username>`
- Immediate token invalidation (no grace period)
- Comprehensive RBAC enforcement
- Full audit trail with timestamps
- Excellent UX with clear warnings and instructions

## Proposed Enhancement: Self-Service Token Rotation

### New Command
```bash
simple-secrets rotate my-token
# or
simple-secrets rotate token --self
```

### Implementation Plan

#### 1. New Permission: "rotate-own-token"
```go
// Update createDefaultRoles() in internal/rbac.go
RolePermissions{
    RoleAdmin:  {"read", "write", "rotate-tokens", "manage-users", "rotate-own-token"},
    RoleReader: {"read", "rotate-own-token"},  // Add self-rotation capability
}
```

#### 2. Command Logic Enhancement
```go
// In cmd/rotate.go - modify rotateToken function
func rotateToken(username string) error {
    user, store, err := RBACGuard(true, TokenFlag)
    if err != nil {
        return err
    }

    // Check if user is rotating their own token
    isSelfRotation := (username == user.Username)

    if isSelfRotation {
        // Self-rotation: check "rotate-own-token" permission
        if !user.Can("rotate-own-token", store.Permissions()) {
            return fmt.Errorf("permission denied: cannot rotate own token")
        }
    } else {
        // Admin rotation: check "rotate-tokens" permission (existing logic)
        if !user.Can("rotate-tokens", store.Permissions()) {
            return fmt.Errorf("permission denied: need 'rotate-tokens' permission")
        }
    }

    // Continue with existing token rotation logic...
}
```

#### 3. Enhanced User Experience
```bash
# Self-rotation command
./simple-secrets rotate my-token

# Output:
✅ Your token has been rotated successfully!
New token: abc123def456...

⚠️  IMPORTANT:
• Your old token is now invalid
• Update your local config: ~/.simple-secrets/config.json
• Or use: export SIMPLE_SECRETS_TOKEN=abc123def456...
```

### Benefits
1. **Immediate Security Response**: Users can rotate compromised tokens instantly
2. **Reduced Admin Overhead**: Routine rotations don't require admin intervention
3. **Better Security Posture**: Encourages regular token rotation
4. **Enterprise Readiness**: Meets security team expectations for self-service capabilities

### Security Considerations
- ✅ No privilege escalation (users can only rotate their own tokens)
- ✅ Immediate invalidation (no grace period needed)
- ✅ Audit trail maintained (TokenRotatedAt still tracked)
- ✅ RBAC still enforced (new permission required)

### Testing Requirements
- User can rotate own token with proper permission
- User cannot rotate other users' tokens without admin permission
- Old token immediately invalid after self-rotation
- Admin token rotation still works as before
- Reader with "rotate-own-token" permission can self-rotate
- Proper error messages for permission denied scenarios
