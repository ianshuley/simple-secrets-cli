Let's conduct a comprehensive, production-grade testing campaign for this application. Use this structured checklist as your testing framework, but feel free to expand beyond it with any additional tests you can perform as an AI agent. I want to test the crap out of this thing. You should be inferring the nature of this semantic testing process applying it to things I haven't thought of as though that directive is itself a task on the following checklist. When you come up with new tests, please add them to the checklist.

## 🚨 **TESTING WORKFLOW NOTE** 🚨

**IMPORTANT**: This application uses development builds for daily work. Use `make dev` for all testing unless specifically testing release functionality. Development builds show `dev-abc123` format and are the normal workflow.

- **Use**: `make dev` (development builds)
- **Shows**: `dev-<git-commit>` in version
- **Don't use**: `make release` or version changes unless testing release features specifically

I want you to execute:

1. **Automated Testing**: Run all unit tests, integration tests, and any test suites
2. **Manual CLI Testing**: Execute commands by hand to verify real-world behavior
3. **Edge Case Discovery**: Look for scenarios I might have missed
4. **Regression Testing**: Ensure previously fixed bugs don't resurface
5. **Performance Testing**: Verify the system handles load and large data
6. **Security Testing**: Validate permissions, encryption, and access controls
7. **Error Path Testing**: Verify all failure modes are handled gracefully

For each test section, report:

- ✅/❌ Status with exit codes where relevant
- 🔍 Any unexpected behavior or edge cases discovered
- 🚨 Any bugs or issues found
- 💡 Suggestions for additional testing or improvements

Focus especially on:

- **Backup/Restore Integrity**: The backup key encryption after rotation issue
- **Cross-Command Integration**: How different commands interact
- **State Consistency**: Data integrity across operations
- **Production Readiness**: Real-world usage scenarios

# Simple-Secrets CLI — Comprehensive Testing Checklist v2.0

> **Version**: 2.0
> **Last Updated**: September 2025
> **Critical Focus**: Backup key encryption integrity after master key rotation

## Pre-Testing Setup

- [ ] **Environment**: Clean test environment prepared
- [ ] **Dependencies**: All build dependencies available
- [ ] **Backup**: Production data backed up if testing on real system
- [ ] **Test Data**: Testing datasets prepared
- [ ] **Automation**: Test scripts ready

---

## 1️⃣ **Build & Installation Testing**

### Basic Build

- [ ] `git pull` or clone latest code
- [ ] `make clean && make dev` - clean development build succeeds
- [ ] Binary created: `ls -la simple-secrets`
- [ ] Help displays: `./simple-secrets --help`
- [ ] Version info: `./simple-secrets version` - shows development version (dev-abc123)
- [ ] Short version: `./simple-secrets version --short` - shows dev-<commit>

### Development vs Release Builds

- [ ] **Development build**: `make dev` - shows `dev-<git-commit>`
- [ ] **Release build**: `make release VERSION=v1.0.0-test` - shows `v1.0.0-test`
- [ ] Version command shows: Git commit, build date, Go version, platform
- [ ] All builds contain proper version injection

### Installation Testing

- [ ] `sudo make install` - system install works (installs current build)
- [ ] `which simple-secrets` - binary in PATH
- [ ] Run from any directory - works globally
- [ ] Version persists: `simple-secrets version` shows same info
- [ ] `sudo make uninstall` - clean removal
- [ ] `simple-secrets` returns "command not found" after uninstall

### Cross-Platform (if applicable)

- [ ] Linux build works
- [ ] macOS build works (if applicable)
- [ ] Windows build works (if applicable)

**Exit Code Verification**: Build commands return 0 on success, non-zero on failure

### 🏷️ **Version System Testing**

**Note**: This application uses development builds (`make dev`) for daily work. Only test release builds when specifically testing release functionality.

#### Development Version Testing

- [ ] `make dev` - builds with git commit in version
- [ ] `./simple-secrets version` - shows full build info with "dev-" prefix
- [ ] `./simple-secrets version --short` - shows just "dev-abc123" format
- [ ] Version includes: git commit, build date, Go version, platform

#### Version Command Testing

- [ ] `./simple-secrets version --help` - shows version command help
- [ ] `./simple-secrets version` - shows full version info
- [ ] `./simple-secrets version --short` - shows short version only
- [ ] `./simple-secrets --version` - shows full version info (flag)
- [ ] `./simple-secrets -v` - shows full version info (short flag)
- [ ] Version appears in main help: `./simple-secrets --help | grep version`
- [ ] All version info is non-empty (no "unknown" values in normal builds)

#### Build-time Injection Testing

- [ ] Make small change, commit, rebuild - version auto-updates with new commit
- [ ] Git commit hash matches: `git rev-parse --short HEAD` vs version output
- [ ] Build date is recent (within last few minutes)

---

## 2️⃣ **Initial Setup & First Run**

### Fresh Installation

- [ ] **Clean slate**: `rm -rf ~/.simple-secrets/`
- [ ] **First command**: `./simple-secrets list keys`
- [ ] **Admin creation message** displays
- [ ] **Admin token** shown and capturable
- [ ] **Directory created**: `~/.simple-secrets/` exists
- [ ] **Files created**: `users.json`, `roles.json`, `master.key`

### File Structure Validation

- [ ] `ls -la ~/.simple-secrets/` - directory permissions 700
- [ ] `ls -la ~/.simple-secrets/*` - files permissions 600
- [ ] **JSON validity**: `cat ~/.simple-secrets/users.json | jq .`
- [ ] **Roles structure**: `cat ~/.simple-secrets/roles.json | jq .`

### Initial State Testing

- [ ] **No secrets**: `./simple-secrets list keys` shows empty
- [ ] **Single admin**: `./simple-secrets list users` shows 1 admin
- [ ] **No backups**: `./simple-secrets list backups` shows empty

**Critical**: Save admin token for subsequent tests!

### ⚠️ **First-Run Behavior Note**

> **Important**: The system automatically creates a default admin user and token on first run. Just run ANY command that goes through authentication (like `put`, `get`, `list keys`, `create-user`, etc.) and the system will:
>
> 1. Detect it's a first run (no `users.json` exists)
> 2. Automatically create a default admin user
> 3. Generate and display a secure token
> 4. Print instructions on how to use the token
>
> **Common Testing Mistake**: Getting confused about initialization order. ALL commands that require RBAC will trigger first-run setup if needed, including `create-user`. The first-run message will appear, then you can use the generated token for subsequent commands.

### 🛡️ **First-Run Protection**

> **Security Enhancement**: The system now includes protection against accidental first-run initialization that could make existing secrets inaccessible.
>
> **Test the Protection**:
>
> - Create scenario: `mkdir test-dir && echo "fake-key" > test-dir/master.key`
> - Run command: `./simple-secrets list keys` (should fail with protection error)
> - Verify message mentions restoring from backup rather than running first-run
>
> **Why This Matters**: Prevents data loss if someone accidentally deletes only `users.json` but leaves encrypted secrets and keys, which would become inaccessible if first-run created new encryption keys.

---

## 3️⃣ **Authentication & Authorization**

### Authentication Methods

- [ ] **No token**: `./simple-secrets list keys` → fails with exit code 1
- [ ] **CLI flag**: `./simple-secrets list keys --token <TOKEN>` → succeeds
- [ ] **Environment**: `SIMPLE_SECRETS_TOKEN=<TOKEN> ./simple-secrets list keys` → succeeds
- [ ] **Config file**: `echo '{"token":"<TOKEN>"}' > ~/.simple-secrets/config.json` then `./simple-secrets list keys` → succeeds

### Token Validation

- [ ] **Invalid token**: `./simple-secrets list keys --token invalid` → exit code 1
- [ ] **Empty token**: `./simple-secrets list keys --token ""` → exit code 1
- [ ] **Malformed token**: `./simple-secrets list keys --token "abc123"` → exit code 1

### Token Precedence (Priority Order)

- [ ] CLI flag overrides environment: Test with different tokens
- [ ] Environment overrides config file: Test with different tokens
- [ ] Config file as fallback: Remove other sources

**Security Check**: No tokens visible in process lists or error messages

---

## 4️⃣ **Secret Management (CRUD Operations)**

### CREATE (Put) Operations

- [ ] **Basic secret**: `./simple-secrets put test-key "test value"`
- [ ] **Special characters**: `./simple-secrets put special '!@#$%^&*()~'`
- [ ] **Unicode**: `./simple-secrets put unicode "🔐 émojis ñ çhar"`
- [ ] **Empty value**: `./simple-secrets put empty ""`
- [ ] **Whitespace value**: `./simple-secrets put spaces "   "`
- [ ] **Multiline value**: `./simple-secrets put multiline "line1\nline2\nline3"`
- [ ] **Large value**: Create 1MB+ test data and store
- [ ] **JSON value**: Store valid JSON as secret value
- [ ] **Binary-like data**: Store base64 encoded data

### READ (Get) Operations

- [ ] **Retrieve basic**: `./simple-secrets get test-key` → exact value
- [ ] **Retrieve special chars**: Verify special characters intact
- [ ] **Retrieve empty**: `./simple-secrets get empty` → empty output
- [ ] **Retrieve multiline**: Verify newlines preserved
- [ ] **Non-existent key**: `./simple-secrets get nonexistent` → exit code 1
- [ ] **Case sensitivity**: `./simple-secrets get TEST-KEY` → fails if keys are case-sensitive

### LIST Operations

- [ ] **List all secrets**: `./simple-secrets list keys` → shows all keys
- [ ] **No values shown**: Verify secret values not displayed in list
- [ ] **Alphabetical order**: Check if keys are sorted
- [ ] **Large list**: With 100+ secrets, verify performance

### UPDATE Operations

- [ ] **Overwrite secret**: `./simple-secrets put test-key "new value"` → updates
- [ ] **Verify update**: `./simple-secrets get test-key` → shows new value
- [ ] **Update preserves others**: Other secrets unchanged

### DELETE Operations

- [ ] **Delete secret**: `./simple-secrets delete test-key`
- [ ] **Verify deletion**: `./simple-secrets get test-key` → exit code 1
- [ ] **List after delete**: Key no longer in list
- [ ] **Delete non-existent**: `./simple-secrets delete nonexistent` → graceful error

### Edge Cases

- [ ] **Empty key name**: `./simple-secrets put "" "value"` → should fail
- [ ] **Whitespace key**: `./simple-secrets put "   " "value"` → should fail or trim
- [ ] **Very long key name**: 500+ character key name
- [ ] **Reserved characters**: Keys with `/`, `\`, quotes, etc.

### 🚨 **Security: Input Validation (Fixed Vulnerabilities)**

- [ ] **Null byte injection**: `./simple-secrets put $'test\x00key' "value"` → should fail with "cannot contain null bytes"
- [ ] **Control character injection**: `./simple-secrets put $'test\x01key' "value"` → should fail with "cannot contain control characters"
- [ ] **Path traversal**: `./simple-secrets put "test/path" "value"` → should fail with "cannot contain path separators"
- [ ] **Path traversal dots**: `./simple-secrets put "test..traversal" "value"` → should fail with "cannot contain path traversal sequences"
- [ ] **Backslash separator**: `./simple-secrets put "test\\key" "value"` → should fail with "cannot contain path separators"
- [ ] **Tab character allowed**: `./simple-secrets put $'test\tkey' "value"` → should succeed (tabs are allowed)
- [ ] **Newline allowed**: `./simple-secrets put $'test\nkey' "value"` → should succeed (newlines are allowed)
- [ ] **Carriage return allowed**: `./simple-secrets put $'test\rkey' "value"` → should succeed (CR is allowed)

### 🚨 **Disable/Enable Secret Management**

#### Secret Disable/Enable Workflow

- [ ] **Put test secrets**: `./simple-secrets put disable-test "test-value"` and `./simple-secrets put other-key "other-value"`
- [ ] **Disable secret**: `./simple-secrets disable secret disable-test` → success message
- [ ] **List excludes disabled**: `./simple-secrets list keys` → should not show `disable-test`, should show `other-key`
- [ ] **List disabled shows secret**: `./simple-secrets list disabled` → should show `disable-test` with 🚫 indicator
- [ ] **Get disabled fails**: `./simple-secrets get disable-test` → exit code 1, "secret not found"
- [ ] **Delete disabled fails**: `./simple-secrets delete disable-test` → exit code 1, "secret not found"
- [ ] **Enable secret**: `./simple-secrets enable secret disable-test` → success message
- [ ] **List includes enabled**: `./simple-secrets list keys` → should show both keys
- [ ] **Get enabled works**: `./simple-secrets get disable-test` → returns "test-value" (preserved)
- [ ] **Value preservation**: Verify secret value is exactly the same after disable/enable cycle

#### Multiple Disable/Enable Operations

- [ ] **Disable multiple**: Disable 3-4 different secrets
- [ ] **List disabled multiple**: `./simple-secrets list disabled` → shows all disabled with indicators
- [ ] **Enable specific**: Enable only 1 of the disabled secrets
- [ ] **Mixed state**: Verify list operations show correct enabled/disabled state
- [ ] **Enable all**: Re-enable all remaining disabled secrets
- [ ] **Empty disabled list**: `./simple-secrets list disabled` → shows "No disabled secrets found"

#### Edge Cases & Error Handling

- [ ] **Disable nonexistent**: `./simple-secrets disable secret nonexistent` → exit code 1, clear error
- [ ] **Enable nonexistent**: `./simple-secrets enable secret nonexistent` → exit code 1, clear error
- [ ] **Double disable**: Disable same secret twice → should handle gracefully
- [ ] **Double enable**: Enable same secret twice → should handle gracefully or show already enabled
- [ ] **Large volume**: Disable/enable 50+ secrets → verify performance
- [ ] **Special characters**: Disable/enable secrets with unicode, spaces, special chars
- [ ] **Very long key names**: Test with maximum length key names

#### Token Disable Operations

- [ ] **Create test user**: `./simple-secrets create-user tokentest reader` → capture token
- [ ] **Test token works**: Use captured token to list keys → succeeds
- [ ] **Disable token**: `./simple-secrets disable token tokentest` → success message
- [ ] **Disabled token fails**: Use disabled token to list keys → exit code 1, "invalid token"
- [ ] **Rotate for recovery**: `./simple-secrets rotate token tokentest` → generates new token
- [ ] **New token works**: Use new token to list keys → succeeds
- [ ] **Old token still invalid**: Use original token → still fails

#### RBAC for Disable/Enable

- [ ] **Reader cannot disable**: Reader token trying to disable secret → exit code 1, permission denied
- [ ] **Reader cannot enable**: Reader token trying to enable secret → exit code 1, permission denied
- [ ] **Reader cannot disable tokens**: Reader trying to disable user tokens → exit code 1
- [ ] **Admin can disable/enable**: Verify admin can perform all disable/enable operations
- [ ] **No token fails**: `./simple-secrets disable secret test` → authentication required

#### Integration with Other Operations

- [ ] **Put with same name**: After disabling secret, put new secret with same name → should work (creates new, not enable old)
- [ ] **Backup operations**: Verify disable/enable operations create backups
- [ ] **Master key rotation**: Disable secrets, rotate master key, verify disabled secrets remain disabled but can be enabled with preserved values

---

## 5️⃣ **User Management & RBAC**

### User Creation

- [ ] **Create reader**: `./simple-secrets create-user reader1 reader` → generates token
- [ ] **Create admin**: `./simple-secrets create-user admin2 admin` → generates token
- [ ] **Create multiple**: Create reader2, reader3, admin3
- [ ] **Invalid role**: `./simple-secrets create-user test invalid` → exit code 1
- [ ] **Duplicate username**: Try creating same username → exit code 1
- [ ] **Empty username**: `./simple-secrets create-user "" reader` → should fail

### RBAC - Reader Permissions

- [ ] **Can list keys**: `./simple-secrets list keys --token <READER_TOKEN>`
- [ ] **Can get secrets**: `./simple-secrets get test-key --token <READER_TOKEN>`
- [ ] **Cannot put**: `./simple-secrets put new "value" --token <READER_TOKEN>` → exit code 1
- [ ] **Cannot delete**: `./simple-secrets delete test-key --token <READER_TOKEN>` → exit code 1
- [ ] **Cannot create users**: `./simple-secrets create-user test reader --token <READER_TOKEN>` → exit code 1
- [ ] **Cannot list users**: `./simple-secrets list users --token <READER_TOKEN>` → exit code 1
- [ ] **Cannot rotate keys**: `./simple-secrets rotate master-key --token <READER_TOKEN>` → exit code 1

### RBAC - Admin Permissions

- [ ] **Can do everything**: Verify admin2 can perform all operations
- [ ] **Can manage users**: Admin2 creates users, rotates tokens
- [ ] **Can rotate master key**: Admin2 can rotate master key

### User Listing

- [ ] **List all users**: `./simple-secrets list users` → shows all with roles
- [ ] **Current user marked**: Verify which user is making the request
- [ ] **Token rotation dates**: Verify timestamps shown

---

## 6️⃣ **🚨 CRITICAL: Master Key Rotation & Backup Integrity**

> **This section tests the previously discovered backup key encryption bug**

### Pre-Rotation Setup

- [ ] **Create test secrets**: Put 5-10 secrets with known values
- [ ] **Note secret values**: Record what each secret contains
- [ ] **Create users**: Have some users with tokens
- [ ] **Record user tokens**: Save current user tokens

### Manual Master Key Rotation

- [ ] **Start rotation**: `./simple-secrets rotate master-key`
- [ ] **Test abort**: Type "no" → rotation aborted, no changes
- [ ] **Complete rotation**: `./simple-secrets rotate master-key` → type "yes"
- [ ] **Backup created**: Verify backup in `~/.simple-secrets/backups/rotate-YYYYMMDD-HHMMSS/`

### 🔍 **CRITICAL: Post-Rotation Verification**

- [ ] **Secrets accessible**: All secrets readable with same values
- [ ] **New secrets work**: Can put/get new secrets
- [ ] **User tokens work**: All existing user tokens still work
- [ ] **Backup contains old key**: Backup directory has old master.key
- [ ] **🚨 BACKUP KEY ENCRYPTION**: Secrets in backup are encrypted with OLD master key
- [ ] **🚨 BACKUP ACCESSIBILITY**: Can restore secrets from backup with old key

### Automated Rotation

- [ ] **Auto rotation**: `./simple-secrets rotate master-key --yes` → no prompts
- [ ] **Custom backup dir**: `./simple-secrets rotate master-key --yes --backup-dir /tmp/test-backup`
- [ ] **Verify custom location**: Backup created in specified directory

### 🚨 **CRITICAL: Atomic Operation Testing (Fixed Vulnerability)**

- [ ] **Interruption resistance**: Start rotation in background, kill process during rotation → system remains accessible
- [ ] **Temp file cleanup**: Verify no .tmp files remain after interrupted rotation
- [ ] **State consistency**: After interruption, secrets still accessible with original key
- [ ] **No partial corruption**: Master key file not corrupted during interruption
- [ ] **Backup integrity**: Backup creation is atomic and not corrupted by interruption

### 🔍 **CRITICAL: Multiple Rotation Test**

- [ ] **Rotate again**: Perform second master key rotation
- [ ] **Verify old backups**: Previous backups still accessible
- [ ] **Verify current secrets**: All secrets work with latest key
- [ ] **🚨 BACKUP CHAIN INTEGRITY**: Each backup encrypted with its corresponding master key

### Token Rotation

- [ ] **Rotate user token**: `./simple-secrets rotate token reader1`
- [ ] **Old token invalid**: Old reader1 token fails
- [ ] **New token works**: New reader1 token works
- [ ] **Other tokens unaffected**: Other user tokens still work

### 🔍 **Disaster Recovery Simulation**

- [ ] **Backup environment**: `cp -r ~/.simple-secrets ~/.simple-secrets.backup`
- [ ] **Simulate corruption**: Delete/corrupt current secrets
- [ ] **Restore from backup**: Copy backup files back
- [ ] **🚨 VERIFY RESTORATION**: Secrets restored with correct values
- [ ] **🚨 KEY COMPATIBILITY**: Restored secrets work with restored master key

---

## 7️⃣ **Backup & Restore Operations**

### Secret-Level Backup/Restore

- [ ] **Put secret**: Create test secret
- [ ] **Delete secret**: `./simple-secrets delete test-secret`
- [ ] **Restore secret**: `./simple-secrets restore secret test-secret`
- [ ] **Verify restoration**: Secret accessible with original value

### Database-Level Restore

- [ ] **List backups**: `./simple-secrets list backups` → shows rotation backups
- [ ] **Restore database**: `./simple-secrets restore database <backup-name>`
- [ ] **Verify full restore**: All secrets from that backup point restored

### Backup File Structure

- [ ] **Examine backup dir**: `ls -la ~/.simple-secrets/backups/`
- [ ] **Check backup contents**: Verify old master.key and encrypted secrets
- [ ] **File permissions**: Backup files properly secured (600)

---

## 8️⃣ **Consolidated Commands Testing**

### List Command Variants

- [ ] **`./simple-secrets list keys`** → shows secret keys
- [ ] **`./simple-secrets list users`** → shows users and roles
- [ ] **`./simple-secrets list backups`** → shows rotation backups
- [ ] **`./simple-secrets list`** → shows help or default behavior
- [ ] **`./simple-secrets list invalid`** → proper error message

### Rotate Command Variants

- [ ] **`./simple-secrets rotate master-key`** → master key rotation
- [ ] **`./simple-secrets rotate token <username>`** → user token rotation
- [ ] **`./simple-secrets rotate`** → shows help or default behavior
- [ ] **`./simple-secrets rotate invalid`** → proper error message

### Restore Command Variants

- [ ] **`./simple-secrets restore secret <key>`** → secret restoration
- [ ] **`./simple-secrets restore database <backup>`** → database restoration
- [ ] **`./simple-secrets restore`** → shows help or default behavior
- [ ] **`./simple-secrets restore invalid`** → proper error message

### Legacy Command Compatibility

- [ ] **Old commands still work**: Verify legacy syntax hasn't been broken
- [ ] **Help consistency**: Both old and new commands show consistent help

---

## 9️⃣ **Error Handling & Edge Cases**

### Invalid Commands

- [ ] **Non-existent command**: `./simple-secrets invalid-command` → exit code 1
- [ ] **Malformed arguments**: `./simple-secrets put` → missing args error
- [ ] **Wrong arg count**: `./simple-secrets get` → missing key error
- [ ] **Invalid flags**: `./simple-secrets list --invalid-flag` → error

### File System Issues

- [ ] **No write permissions**: `chmod 555 ~/.simple-secrets` → graceful error
- [ ] **Restore permissions**: `chmod 755 ~/.simple-secrets`
- [ ] **Disk full simulation**: (if possible) → graceful error
- [ ] **Corrupted files**: Manually corrupt JSON files → graceful error

### Data Corruption Scenarios

- [ ] **Corrupt users.json**: Add invalid JSON → graceful error + exit code 1
- [ ] **Corrupt roles.json**: Add invalid JSON → graceful error
- [ ] **Corrupt master.key**: Modify master key → encryption error
- [ ] **Missing files**: Delete critical files → proper error messages

### Large Data & Performance

- [ ] **Many secrets**: Create 500+ secrets → verify performance
- [ ] **Large secret value**: Store 10MB+ data (if system allows)
- [ ] **Long key names**: 1000+ character key names
- [ ] **Concurrent access**: (if possible) multiple operations

### Network & System Edge Cases

- [ ] **Full disk**: (if testable) verify graceful handling
- [ ] **Permission changes**: Modify file permissions during operation
- [ ] **Process interruption**: `Ctrl+C` during operations → clean state

---

## 🔟 **Security & Permissions Testing**

### File System Security

- [ ] **Directory permissions**: `~/.simple-secrets/` is 700 (user only)
- [ ] **File permissions**: All files are 600 (user read/write only)
- [ ] **Backup permissions**: Backup files maintain secure permissions
- [ ] **No world-readable**: `find ~/.simple-secrets -perm +004` → no results

### Encryption Verification

- [ ] **Encrypted at rest**: `cat ~/.simple-secrets/secrets.json` → encrypted data
- [ ] **Master key encrypted**: Master key file contains encrypted data
- [ ] **No plaintext leaks**: No secret values in temp files or logs

### Authentication Security

- [ ] **Token security**: Tokens not visible in process lists
- [ ] **No token leaks**: Error messages don't contain tokens
- [ ] **Session isolation**: Different tokens access different data correctly

### Process Security

- [ ] **Memory dumps**: (if possible) no secrets in memory dumps
- [ ] **Temp files**: No temporary files with secret data
- [ ] **Log files**: No secrets logged anywhere

---

## 1️⃣1️⃣ **Performance & Scalability**

### Bulk Operations

- [ ] **Bulk secret creation**: Script to create 1000+ secrets

```bash
for i in $(seq -w 001 1000); do
  ./simple-secrets put "perf-test-$i" "value-$i-$(date +%s)"
done
```

- [ ] **List performance**: `time ./simple-secrets list keys` with many secrets
- [ ] **Individual access**: `time ./simple-secrets get perf-test-500` → fast retrieval
- [ ] **Bulk deletion**: Delete many secrets → performance acceptable

### Memory & Resource Usage

- [ ] **Memory usage**: Monitor memory during operations
- [ ] **File handle usage**: No leaked file descriptors
- [ ] **CPU usage**: Operations complete efficiently

### Large Data Handling

- [ ] **Large secret storage**: 50MB+ secret (if system allows)
- [ ] **Large secret retrieval**: Retrieve large secret efficiently
- [ ] **Multiple large secrets**: Store several large secrets

---

## 1️⃣2️⃣ **Cross-Platform & Integration**

### System Integration

- [ ] **Shell integration**: Works in bash, zsh, fish (if applicable)
- [ ] **PATH integration**: Works from any directory when installed
- [ ] **Exit code standards**: Follows Unix exit code conventions
- [ ] **Signal handling**: Proper Ctrl+C handling

### Scripting & Automation

- [ ] **Scriptable**: Commands work in shell scripts
- [ ] **Exit codes**: Scripts can check `$?` for success/failure
- [ ] **Output parsing**: Command output is consistent and parseable
- [ ] **Batch operations**: Multiple commands in sequence work correctly

### Configuration Management

- [ ] **Config file handling**: Multiple config file scenarios
- [ ] **Environment isolation**: Different HOME directories work
- [ ] **Cleanup**: `make purge` removes all user data

---

## 1️⃣3️⃣ **Automated Test Suite Validation**

### Unit Tests

- [ ] **Run unit tests**: `go test ./internal -v` → all pass
- [ ] **Coverage check**: Unit test coverage acceptable
- [ ] **Test isolation**: Tests don't interfere with each other

### Integration Tests

- [ ] **Run integration tests**: `go test ./integration -v` → all pass
- [ ] **End-to-end workflows**: Complete user journeys tested
- [ ] **Cross-component testing**: Multiple components work together

### Regression Tests

- [ ] **Previous bugs**: Tests for previously fixed issues
- [ ] **🚨 Backup encryption bug**: Specific tests for backup key encryption issue
- [ ] **Exit code fixes**: Tests for authentication exit code fixes

### Test Suite Maintenance

- [ ] **Test data cleanup**: Tests clean up after themselves
- [ ] **Test performance**: Test suite runs in reasonable time
- [ ] **Test reliability**: Tests pass consistently

---

## 1️⃣3️⃣ **Manual Disable/Enable Workflow Testing**

### Quick Smoke Test for Disable/Enable

**Setup:**

```bash
# Build and prepare
make build
export TEST_TOKEN="<admin-token-from-setup>"

# Create test secrets
./simple-secrets put test-secret-1 "value-1" --token $TEST_TOKEN
./simple-secrets put test-secret-2 "value-2" --token $TEST_TOKEN
./simple-secrets put test-secret-3 "value-3" --token $TEST_TOKEN
```

**Secret Disable/Enable Test:**

- [ ] **Initial state**: `./simple-secrets list keys` → shows all 3 secrets
- [ ] **Disable one**: `./simple-secrets disable secret test-secret-2` → success message
- [ ] **List excludes disabled**: `./simple-secrets list keys` → shows test-secret-1, test-secret-3 only
- [ ] **List disabled**: `./simple-secrets list disabled` → shows test-secret-2 with 🚫 icon
- [ ] **Get disabled fails**: `./simple-secrets get test-secret-2` → "secret not found" error
- [ ] **Enable secret**: `./simple-secrets enable secret test-secret-2` → success message
- [ ] **List includes enabled**: `./simple-secrets list keys` → shows all 3 secrets again
- [ ] **Value preserved**: `./simple-secrets get test-secret-2` → returns "value-2"

**Token Disable Test:**

- [ ] **Create test user**: `./simple-secrets create-user testuser reader` → capture token
- [ ] **Test token works**: Use captured token to list keys → succeeds
- [ ] **Disable token**: `./simple-secrets disable token testuser` → success message
- [ ] **Disabled token fails**: Use disabled token → "invalid token" error
- [ ] **Generate new token**: `./simple-secrets rotate token testuser` → new token
- [ ] **New token works**: Use new token → succeeds

**Edge Cases:**

- [ ] **Double disable**: Disable same secret twice → graceful handling
- [ ] **Enable non-disabled**: Enable active secret → appropriate message
- [ ] **Nonexistent operations**: Disable/enable nonexistent resources → clear errors

---

## 1️⃣4️⃣ **Documentation & Usability**

### Help & Documentation

- [ ] **Global help**: `./simple-secrets --help` → comprehensive
- [ ] **Command help**: `./simple-secrets <cmd> --help` → specific help
- [ ] **Error messages**: Clear, actionable error messages
- [ ] **Examples**: Help includes usage examples

### User Experience

- [ ] **First-time user**: Fresh user can follow documentation
- [ ] **Error recovery**: Users can recover from mistakes
- [ ] **Feedback**: Commands provide appropriate feedback
- [ ] **Consistency**: Similar operations behave similarly

---

## 1️⃣5️⃣ **Production Readiness Checklist**

### Deployment Testing

- [ ] **Clean installation**: Fresh install on clean system
- [ ] **Upgrade testing**: Upgrade from previous version (if applicable)
- [ ] **Migration testing**: Data migration works correctly
- [ ] **Rollback testing**: Can rollback if needed

### Operational Testing

- [ ] **Monitoring**: System behaves predictably under monitoring
- [ ] **Logging**: Appropriate log levels and content
- [ ] **Debugging**: Debug information available when needed
- [ ] **Troubleshooting**: Common issues have clear solutions

### Security Audit

- [ ] **Permission review**: All file permissions appropriate
- [ ] **Encryption review**: All sensitive data encrypted
- [ ] **Authentication review**: Auth mechanisms secure
- [ ] **Attack surface**: Minimize exposed functionality

---

## 🎯 **Critical Test Results Verification**

For each test section, document:

### Exit Codes

- [ ] **Success operations**: Return 0
- [ ] **User errors**: Return 1
- [ ] **System errors**: Return appropriate non-zero codes
- [ ] **Consistency**: Same types of errors return same codes

### Error Messages

- [ ] **Clarity**: Error messages are understandable
- [ ] **Actionability**: Users know how to fix issues
- [ ] **No leaks**: No sensitive data in error messages
- [ ] **Consistency**: Similar errors have similar messages

### Data Integrity

- [ ] **🚨 CRITICAL**: Backup restoration preserves exact secret values
- [ ] **🚨 CRITICAL**: Master key rotation doesn't corrupt existing secrets
- [ ] **🚨 CRITICAL**: Backup encryption uses correct master key for each backup
- [ ] **Consistency**: Operations are atomic (all succeed or all fail)

### Performance Benchmarks

- [ ] **Response time**: Commands complete in <1 second for normal operations
- [ ] **Scalability**: System handles expected load (define your limits)
- [ ] **Resource usage**: Memory and CPU usage reasonable

---

## Test Execution Log Template

### Test Session Information

- **Date/Time**: [YYYY-MM-DD HH:MM:SS]
- **Tester**: [Name]
- **Application Version**: [Version/Commit Hash]
- **Environment**: [OS, Go Version, etc.]
- **Test Objective**: [Brief description]

### Pre-Test Setup

- [ ] Clean environment preparation
- [ ] Binary compilation verification
- [ ] Initial configuration validation
- [ ] Baseline performance metrics captured

### Critical Focus Areas Checklist

#### 1. Backup Key Encryption Integrity (HIGH PRIORITY)

**Context**: Previous bug where backup encryption used wrong master key after rotation

- [ ] **Test 1.1**: Create secret, rotate master key, verify backup decryption
  - Command: `./simple-secrets put test-key "test-value"`
  - Command: `./simple-secrets rotate master-key`
  - Verification: Backup file should decrypt with OLD master key, not new one
  - **Expected**: Backup encrypted with pre-rotation key
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 1.2**: Multiple rotations backup integrity
  - Rotate master key 3 times
  - Verify each backup uses correct historical key
  - **Expected**: Each backup decryptable with respective master key
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 1.3**: Disaster recovery scenario
  - Create secrets, rotate master key, simulate data loss
  - Restore from backup only
  - **Expected**: Complete recovery possible
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

#### 2. Authentication & Exit Codes (HIGH PRIORITY)

**Context**: Previous bug where auth failures returned exit code 0

- [ ] **Test 2.1**: Invalid token handling
  - Command: `./simple-secrets list keys --token invalid`
  - **Expected**: Exit code 1, clear error message
  - **Actual**: Exit code: __ Message: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 2.2**: No authentication provided
  - Command: `./simple-secrets put key value` (no auth)
  - **Expected**: Exit code 1, auth required message
  - **Actual**: Exit code: __ Message: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

#### 3. Core Functionality Matrix

| Feature | Command | Auth Method | Expected Result | Actual Result | Status |
|---------|---------|-------------|-----------------|---------------|---------|
| Create Secret | `put key "value"` | Token | Success | | ☐ P ☐ F |
| Read Secret | `get key` | Token | Value returned | | ☐ P ☐ F |
| List Secrets | `list keys` | Token | Keys listed | | ☐ P ☐ F |
| Delete Secret | `delete key` | Admin | Key removed | | ☐ P ☐ F |
| Create User | `create-user alice admin` | Admin | User created | | ☐ P ☐ F |
| List Users | `list users` | Admin | Users listed | | ☐ P ☐ F |
| Rotate Token | `rotate token alice` | Admin | New token | | ☐ P ☐ F |
| Master Key Rotation | `rotate master-key` | Admin | Keys rotated | | ☐ P ☐ F |
| List Backups | `list backups` | Admin | Backups shown | | ☐ P ☐ F |
| Restore Secret | `restore secret key timestamp` | Admin | Secret restored | | ☐ P ☐ F |
| Restore Database | `restore database timestamp` | Admin | DB restored | | ☐ P ☐ F |

#### 4. RBAC Verification Matrix

| User Role | Command | Expected Access | Test Result | Status |
|-----------|---------|-----------------|-------------|---------|
| Admin | All commands | Full access | | ☐ P ☐ F |
| Reader | `get`, `list keys` | Read-only access | | ☐ P ☐ F |
| Reader | `put`, `delete` | Access denied | | ☐ P ☐ F |
| Reader | `create-user` | Access denied | | ☐ P ☐ F |
| Reader | `rotate` commands | Access denied | | ☐ P ☐ F |

#### 5. Edge Cases & Error Handling

- [ ] **Test 5.1**: Large secret values (>1MB)
  - **Expected**: Graceful handling or clear limits
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 5.2**: Special characters in keys/values
  - Test with: Unicode, newlines, quotes, null bytes
  - **Expected**: Proper encoding/escaping
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 5.3**: Concurrent operations
  - Multiple simultaneous puts/gets
  - **Expected**: Data consistency maintained
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

#### 6. Performance & Resource Usage

- [ ] **Test 6.1**: Startup time measurement
  - Command: `time ./simple-secrets list keys`
  - **Baseline**: < 100ms for cold start
  - **Actual**: ________________ms
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

- [ ] **Test 6.2**: Memory usage with large datasets
  - Create 1000+ secrets, monitor memory
  - **Expected**: Linear growth, no leaks
  - **Actual**: ________________
  - **Status**: ☐ PASS ☐ FAIL ☐ SKIP

### Issues Discovered

| Issue ID | Severity | Description | Steps to Reproduce | Status |
|----------|----------|-------------|-------------------|---------|
| | | | | |
| | | | | |

### Test Summary

- **Total Tests Executed**: ___
- **Passed**: ___
- **Failed**: ___
- **Skipped**: ___
- **Critical Issues Found**: ___
- **Regression Risk**: ☐ Low ☐ Medium ☐ High

### Recommendations

- [ ] Areas requiring additional testing
- [ ] Performance optimization opportunities
- [ ] Security considerations
- [ ] Documentation updates needed

### Sign-off

- **Tester Signature**: ________________
- **Date**: ________________
- **Ready for Production**: ☐ Yes ☐ No ☐ With Conditions

---

## 🔄 **Continuous Testing Guidelines**

### When to Run Full Test Suite

- [ ] Before releases
- [ ] After major features
- [ ] After security fixes
- [ ] Weekly/monthly regression testing

### When to Run Specific Sections

- [ ] **Authentication tests**: After auth changes
- [ ] **🚨 Backup tests**: After encryption/key management changes
- [ ] **RBAC tests**: After permission changes
- [ ] **Performance tests**: After scalability changes

### Test Environment Management

- [ ] **Clean environment**: Start with fresh state
- [ ] **Test data management**: Consistent test datasets
- [ ] **Environment cleanup**: Clean up after testing
- [ ] **Documentation updates**: Keep checklist current

---

> **🚨 SPECIAL FOCUS: Backup Key Encryption Integrity**
>
> This bug was critical: after master key rotation, backup keys were not properly encrypted with their corresponding master keys, making backups inaccessible. Ensure all backup-related tests verify that:
>
> 1. Each backup is encrypted with its corresponding master key
> 2. Backups remain accessible after multiple rotations
> 3. Restore operations work with the correct backup keys
> 4. The backup chain integrity is maintained across rotations

---

**Testing Philosophy**: Test like your production system depends on it, because it does. 🚀
