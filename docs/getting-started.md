# Getting Started with simple-secrets

This guide will help you install, initialize, and use `simple-secrets` to securely manage secrets for automation workflows.

## 1. Installation

### Prerequisites
- Go 1.20 or newer

### Build and Install

```sh
# Clone the repository
$ git clone https://github.com/yourusername/simple-secrets-cli.git
$ cd simple-secrets-cli

# Build and install using Makefile
$ make build
$ sudo make install

# Or install to custom location
$ make install PREFIX=$HOME/.local

# The binary will be available as 'simple-secrets' in your PATH
```


## 2. First Run & Initialization

On first run, `simple-secrets` will automatically initialize your secrets store in `~/.simple-secrets` and create a default admin user. The admin token will be printed to the console—store it securely!

> **🛡️ Data Protection**: The system includes protection against accidental re-initialization. If critical files like `master.key` or `secrets.json` exist but `users.json` is missing, first-run setup will be blocked to prevent making existing secrets inaccessible. If you need to recover from this state, restore `users.json` from backup or manually recreate your user configuration.


## 3. Store a Secret

Save a secret value under a key (requires authentication):

```sh
$ simple-secrets put db_password s3cr3tP@ssw0rd --token <your-token>
Secret "db_password" stored.

# Alternative: use the 'add' alias
$ simple-secrets add api_key abc123xyz --token <your-token>
Secret "api_key" stored.
```


## 4. Retrieve a Secret

Get the value for a key:

```sh
$ simple-secrets get db_password --token <your-token>
s3cr3tP@ssw0rd
```


## 5. List All Secret Keys

Show all stored keys:

```sh
$ simple-secrets list keys --token <your-token>
api_key
db_password
```


## 6. Delete a Secret

Remove a secret by key:

```sh
$ simple-secrets delete api_key --token <your-token>
Secret "api_key" deleted.
```


## 7. Rotate the Master Key

Generate a new master key and re-encrypt all secrets. A backup is created automatically. (Admin only)

```sh
$ simple-secrets rotate master-key --token <admin-token>
This will:
  • Generate a NEW master key
  • Re-encrypt ALL secrets with the new key
  • Create a backup of the old key+secrets for rollback
Proceed? (type 'yes'): yes
Rotation complete. Backup created under ~/.simple-secrets/backups/
```


## 8. Advanced: Custom Backup Directory

Specify a backup location during rotation:

```sh
$ simple-secrets rotate master-key --backup-dir /tmp/secrets-backup --token <admin-token>
```

## 9. Database Backup Management

### List Available Backups

View all rotation backups with timestamps:

```sh
$ simple-secrets list backups --token <admin-token>
Found 3 rotation backup(s):

  rotate-20240901-143022 (most recent)
    Created: 2024-09-01 14:30:22
    Status:  ✓ Valid

  rotate-20240901-120030
    Created: 2024-09-01 12:00:30
    Status:  ✓ Valid

  rotate-20240830-095505
    Created: 2024-08-30 09:55:05
    Status:  ✗ Invalid
```

### Restore Database from Backup

Restore your entire secrets database from any backup (two command styles available):

```sh
# Using the consolidated restore command
$ simple-secrets restore database --token <admin-token>
Will restore from most recent backup: rotate-20240901-143022
  Created: 2024-09-01 14:30:22
This will:
  • Create a backup of your current secrets database
  • Replace your current database with the specified backup
  • This action affects ALL secrets in your database
Proceed? (type 'yes'): yes
Database restored successfully.

# Using the dedicated restore-database command (alternative)
$ simple-secrets restore-database --token <admin-token>

# Restore from specific backup (both styles work)
$ simple-secrets restore database rotate-20240830-095505 --token <admin-token>
$ simple-secrets restore-database rotate-20240830-095505 --token <admin-token>

# Skip confirmation prompt
$ simple-secrets restore database --yes --token <admin-token>
$ simple-secrets restore-database --yes --token <admin-token>
```

**Important**: Database restore affects ALL secrets and creates a pre-restore backup for safety.

## 10. User Management

Create a new user (admin or reader):

```sh
# Interactive (prompts for missing info)
$ simple-secrets create-user --token <admin-token>
Username: alice
Role (admin/reader): reader
Generated token: <token>
User "alice" created.

# Or specify username and role directly
$ simple-secrets create-user alice reader --token <admin-token>
User "alice" created.
Generated token: <token>
```

## 11. Token Rotation

Rotate authentication tokens for security:

```sh
# Rotate your own token (both admin and reader users can do this)
$ simple-secrets rotate token --token <your-current-token>
✅ Your token has been rotated successfully!
New token: <new-token>

⚠️  IMPORTANT:
• Store this token securely - it will not be shown again
• Your old token is now invalid and cannot be used
• Update your local configuration with the new token

# Admins can rotate other users' tokens
$ simple-secrets rotate token alice --token <admin-token>
Token rotated for user "alice" (reader role).
New token: <new-token>
```

**Security Note**: Old tokens are immediately invalidated after rotation. Update your configuration files, environment variables, or scripts with the new token.

## 12. RBAC & Permissions

- **admin**: Can read, write, rotate keys, manage users, rotate own token.
- **reader**: Can only read/list secrets and rotate own token.

## 13. Individual Secret Backup & Restore

Previous secret values are backed up automatically on overwrite or delete. Restore a secret from backup:

```sh
$ simple-secrets restore secret db_password --token <admin-token>
Secret "db_password" restored from backup.
```

## 14. Secret Lifecycle Management

### Disable Secrets

Temporarily hide secrets from normal operations without deleting them:

```sh
# Disable a secret (admin only)
$ simple-secrets disable secret sensitive_key --token <admin-token>
✅ Secret 'sensitive_key' has been disabled
• The secret is hidden from normal operations
• Use 'enable secret' to re-enable this secret

# Verify secret is hidden from normal list
$ simple-secrets list keys --token <token>
# (sensitive_key will not appear)

# List disabled secrets specifically
$ simple-secrets list disabled --token <token>
Disabled secrets (1):
  🚫 sensitive_key

Use 'enable secret <key>' to re-enable a disabled secret.
```

### Enable Secrets

Re-enable previously disabled secrets:

```sh
# Re-enable a disabled secret
$ simple-secrets enable secret sensitive_key --token <admin-token>
✅ Secret 'sensitive_key' has been re-enabled
• The secret is now available for normal operations

# Verify secret value is preserved
$ simple-secrets get sensitive_key --token <token>
# (original value returned intact)
```

### Disable User Tokens

Disable user authentication tokens for security purposes:

```sh
# Disable a user's token (admin only)
$ simple-secrets disable token alice --token <admin-token>
✅ Token disabled for user 'alice'
• The user can no longer authenticate with their current token
• Use 'rotate token' to generate a new token for this user

# Generate new token for user (recovery method)
$ simple-secrets rotate token alice --token <admin-token>
Token rotated for user "alice".
New token: <new-secure-token>
```

**Use Cases for Disable/Enable:**
- **Secret Rotation Planning**: Disable secrets before updating dependent systems
- **Security Incidents**: Quickly disable compromised tokens without deleting users
- **Maintenance Windows**: Temporarily hide secrets during system updates
- **Audit Compliance**: Demonstrate controlled access to sensitive secrets

## 15. Shell Completion

Enable shell autocompletion for better CLI experience:

```sh
# Bash completion
$ simple-secrets completion bash > /etc/bash_completion.d/simple-secrets

# Zsh completion
$ simple-secrets completion zsh > "${fpath[1]}/_simple-secrets"

# Fish completion
$ simple-secrets completion fish > ~/.config/fish/completions/simple-secrets.fish

# PowerShell completion
$ simple-secrets completion powershell > simple-secrets.ps1
```

## 16. Token Authentication

Authenticate using:

- `--token <token>` flag
- `SIMPLE_SECRETS_TOKEN` environment variable
- `~/.simple-secrets/config.json` with `{ "token": "<token>" }`

## Security Notes

- Secrets are encrypted with AES-256-GCM.
- Master key is stored in `~/.simple-secrets/master.key` (protected 0600).
- Rotation backups are automatically cleaned up (keeps last 5).
- Always keep backups safe and rotate keys regularly.
- Database restores create pre-restore backups for safety.

## Next Steps

- Integrate with Ansible or GitOps workflows
- Explore key protection backends (passphrase, keyring, KMS)
- Set up regular key rotation schedule
- Use shell completion for better CLI experience
- See CLI help: `simple-secrets --help`
- View all available commands: `simple-secrets completion --help`

### Available Commands Summary

**Core Operations:**
- `put` / `add` - Store secrets
- `get` - Retrieve secrets
- `list` - List keys, backups, users, or disabled secrets
- `delete` - Remove secrets
- `disable` - Disable secrets or user tokens
- `enable` - Re-enable disabled secrets

**User Management:**
- `create-user` - Create admin/reader users
- `rotate token` - Rotate authentication tokens

**Backup & Restore:**
- `rotate master-key` - Rotate encryption key with backup
- `restore secret` - Restore individual secrets
- `restore database` / `restore-database` - Restore entire database

**Utilities:**
- `completion` - Generate shell autocompletion scripts
