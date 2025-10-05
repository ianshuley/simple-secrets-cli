# Simple Secrets CLI

A secure command-line tool for managing secrets with role-based access control (RBAC), designed for teams that need a simple, file-based secret management solution without complex infrastructure requirements.

## Features

- **🔐 Secure Storage**: AES-256-GCM encryption for all secrets
- **👥 Role-Based Access Control**: Admin and reader roles with granular permissions
- **🔄 Token Management**: Secure token rotation with self-service capabilities
- **💾 Backup & Restore**: Automatic backups with encrypted restore capabilities
- **🔧 Secret Lifecycle**: Disable/enable secrets for security management
- **🎯 Simple CLI**: Intuitive commands for common secret operations
- **🔑 Flexible Authentication**: Token-based auth via flag, environment, or config file
- **🚀 Zero Dependencies**: Single binary with no external service requirements

## Installation

### Build from source

```bash
git clone https://github.com/yourusername/simple-secrets-cli
cd simple-secrets-cli
make build
```

### Install globally

```bash
sudo make install
```

### Installation & Usage (with Makefile)

You can use the provided Makefile for easy build, install, and cleanup:

```bash
# Build the CLI
make build

# Install system-wide (default: /usr/local/bin)
sudo make install

# Install to a custom prefix (e.g., your home bin)
make install PREFIX=$HOME/.local

# Uninstall the CLI
sudo make uninstall

# Clean up build artifacts
make clean

# Purge everything (build, install, user data, coverage)
make purge
```

## Quick Start

### First-Time Setup

To get started with `simple-secrets`, run the setup command to create your admin user:

```bash
# Run setup to create admin user and get your token
simple-secrets setup

# Alternative: use the --setup flag
simple-secrets --setup
```

Setup process:

1. **Prompts for confirmation** to create admin user
2. **Creates admin user** and generates secure authentication token
3. **Displays token once** - save it securely as it will not be shown again
4. **You're ready to use simple-secrets** with your new token

### Basic Usage

```bash
# First run: setup to create admin user
simple-secrets setup
# (Prompts for confirmation, creates admin, displays token)

# Then use the token for actual operations:
simple-secrets list keys --token YOUR_ADMIN_TOKEN

# Store secrets securely (use single quotes to prevent shell expansion)
simple-secrets put api-key '--prod-key-abc123' --token YOUR_ADMIN_TOKEN
simple-secrets put db-url 'postgresql://user:pass@localhost:5432/db' --token YOUR_ADMIN_TOKEN

# Generate secure random secrets directly
simple-secrets put jwt-secret --generate --length 64 --token YOUR_ADMIN_TOKEN
simple-secrets put api-key --generate -l 32 --token YOUR_ADMIN_TOKEN

# ⚠️  SECURITY: Single quotes vs double quotes
simple-secrets put safe-key 'echo $(whoami)'     # ✅ Stores literally: "echo $(whoami)"
simple-secrets put danger "echo $(whoami)"       # ❌ Executes command before storing!

# Retrieve secrets
simple-secrets get api-key --token YOUR_ADMIN_TOKEN

# List and manage secrets
simple-secrets list keys --token YOUR_ADMIN_TOKEN
simple-secrets list users --token YOUR_ADMIN_TOKEN
simple-secrets list backups --token YOUR_ADMIN_TOKEN
simple-secrets list disabled --token YOUR_ADMIN_TOKEN

# Advanced operations
simple-secrets disable secret-name --token YOUR_ADMIN_TOKEN
simple-secrets enable secret-name --token YOUR_ADMIN_TOKEN
simple-secrets delete secret-name --token YOUR_ADMIN_TOKEN
```

## Configuration

### Advanced Features\n\n```bash\n# User Management (Admin only)\nsimple-secrets create-user alice reader --token YOUR_ADMIN_TOKEN\nsimple-secrets create-user bob admin --token YOUR_ADMIN_TOKEN\n\n# Token Management & Rotation\nsimple-secrets rotate token alice --token YOUR_ADMIN_TOKEN\nsimple-secrets rotate token --self --token YOUR_TOKEN  # Self-rotate\n\n# Master Key Rotation (creates automatic backup)\nsimple-secrets rotate master-key --token YOUR_ADMIN_TOKEN\n\n# Backup & Restore Operations\nsimple-secrets restore secret old-secret-name --token YOUR_ADMIN_TOKEN\nsimple-secrets restore database backup-20250105-143022 --token YOUR_ADMIN_TOKEN\n```\n\n## ⚙️ Configuration\n\n### Data Storage Locations\n\n**Linux/macOS**:\n- Config: `~/.config/simple-secrets/`\n- Data: `~/.local/share/simple-secrets/`\n- Backups: `~/.local/share/simple-secrets/backups/`\n\n**Windows**:\n- Config: `%APPDATA%\\simple-secrets\\`\n- Data: `%LOCALAPPDATA%\\simple-secrets\\`\n- Backups: `%LOCALAPPDATA%\\simple-secrets\\backups\\`\n\n### Environment Variables\n\n```bash\n# Authentication token (alternative to --token flag)\nexport SIMPLE_SECRETS_TOKEN=your-token-here\n\n# Custom data directory\nexport SIMPLE_SECRETS_DATA_DIR=/custom/path/to/data\n\n# Custom config directory  \nexport SIMPLE_SECRETS_CONFIG_DIR=/custom/path/to/config\n```\n\n### Configuration File\n\nOptional config file at `~/.config/simple-secrets/config.yaml`:\n\n```yaml\n# Default authentication token\ntoken: your-default-token-here\n\n# Custom data directory\ndata_dir: /custom/path/to/secrets\n\n# Backup retention (days)\nbackup_retention_days: 30\n\n# Enable debug logging\ndebug: false\n```\n\n### Legacy Configuration\n\nFor backwards compatibility, you can still use `~/.simple-secrets/config.json`:\n\n```json
{
  "_comment": "Configuration file for simple-secrets CLI - run 'simple-secrets config' for full documentation",
  "rotation_backup_count": 1
}
```

**Configuration Options:**

- `token`: Personal access token for authentication (optional)
- `rotation_backup_count`: Number of backup copies kept during master key rotation (default: 1)

**Note:** Individual secret backups are always 1 (previous version) by design. The `rotation_backup_count` only affects master key rotation operations.

For complete configuration documentation and examples, run:

```bash
simple-secrets config
```

### Authentication Methods

1. **Command flag**: `--token YOUR_TOKEN`
2. **Environment variable**: `export SIMPLE_SECRETS_TOKEN=YOUR_TOKEN`
3. **Config file**: `~/.simple-secrets/config.json` (optional, you can create manually)

**Configuration Options:**

- `token`: Authentication token for API access
- `rotation_backup_count`: Number of rotation backups to keep (default: 1)
  - Set to 1 for minimal attack surface (recommended for high-security environments)
  - Increase (e.g., 3-5) for more recovery options in development/testing
  - Must be a positive integer

### File Structure

```text
~/.simple-secrets/
├── config.json     # Optional configuration
├── master.key      # Encryption key (protect this!)
├── secrets.json    # Encrypted secrets
├── users.json      # User accounts and roles
├── roles.json      # Permission definitions
└── backups/        # Automatic backups
```

## Version Information

Check the version and build information:

```bash
# Show full version information
simple-secrets version
simple-secrets --version  # Standard flag
simple-secrets -v         # Short flag

# Show short version only
simple-secrets version --short
```

### Version Format

- **Release versions**: `v1.0.0`, `v1.2.3-beta.1`
- **Development builds**: `dev-abc1234` (includes git commit hash)

### Build Targets

```bash
# Development build (includes git commit)
make dev

# Release build with specific version
make release VERSION=v1.0.0
```

## Commands

### Secret Management

```bash
# Store secrets
simple-secrets put KEY VALUE
simple-secrets add KEY VALUE  # alias

# Generate secure secrets automatically
simple-secrets put KEY --generate           # Generate 32-character secret
simple-secrets put KEY -g                   # Short flag variant
simple-secrets put KEY --generate --length 64  # Custom length
simple-secrets put KEY -g -l 32             # Short flags for both

# Retrieve secrets
simple-secrets get KEY

# List secrets
simple-secrets list keys

# Delete secrets
simple-secrets delete KEY

# Disable/enable secrets
simple-secrets disable secret KEY
simple-secrets enable secret KEY
simple-secrets list disabled
```

#### Generate Secure Secrets

The `--generate` flag automatically creates cryptographically secure secrets:

```bash
# Generate a 32-character secret (default)
simple-secrets put api-key --generate
# Output:
# Secret "api-key" stored.
# Kj9#mX2w@pL8vR3qN7z!F5sT4uY6eA1c

# Generate with custom length
simple-secrets put db-password --generate --length 64

# Short flag variant
simple-secrets put jwt-secret -g --length 128

# Both short flags
simple-secrets put api-token -g -l 64
```

**Generated Secret Specifications:**

- Uses `crypto/rand` for cryptographically secure randomness
- Character set: `A-Z`, `a-z`, `0-9`, `!@#$%^&*()-_=+` (URL-safe)
- Default length: 32 characters
- Custom length: Use `--length N` or `-l N` flag
- Cannot combine with manual values (error if both provided)

#### Working with Complex Values

**🛡️ Shell Security Warning**: Always use single quotes for secret values to prevent accidental command execution:

```bash
# ✅ SAFE - Single quotes store values literally
simple-secrets put script 'rm -rf /'           # Stores the literal string
simple-secrets put cmd 'echo $(date)'          # Stores the literal string
simple-secrets put password 'p@$$w0rd!$'       # Stores exactly as written

# ❌ DANGEROUS - Double quotes allow shell command substitution
simple-secrets put dangerous "echo $(whoami)"  # Executes whoami command!
simple-secrets put risky "rm -rf $HOME"        # Could execute rm command!

# The shell processes double quotes BEFORE the app sees them
# This affects ALL CLI tools (git, cp, mv, etc.) - not just simple-secrets
```

#### Complex Value Examples

Values starting with dashes or containing special characters work naturally with quotes:

```bash
# API keys starting with dashes
simple-secrets put api-key "--prod-key-abc123"

# Multi-line content like SSH keys
simple-secrets put ssh-key "-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----"

# JSON configuration
simple-secrets put config '{"database": {"host": "localhost", "port": 5432}}'

# From files
simple-secrets put ssl-cert "$(cat /path/to/certificate.pem)"
```

### User Management

```bash
# Create users
simple-secrets create-user USERNAME ROLE  # admin or reader

# List users
simple-secrets list users

# Disable user tokens (clear, specific commands)
simple-secrets disable user USERNAME      # By username
simple-secrets disable token TOKEN_VALUE  # By token value

# Re-enable disabled users (generate new tokens)
simple-secrets enable user USERNAME       # Generate new token for user
simple-secrets enable user USERNAME       # Same as above (alias)
```

### Token Rotation

```bash
# Rotate your own token
simple-secrets rotate token

# Admin: rotate another user's token
simple-secrets rotate token USERNAME
```

### Master Key Rotation

```bash
# Rotate encryption key (admin only)
simple-secrets rotate master-key
```

### Backup & Restore

Simple Secrets CLI provides two complementary backup systems for different scenarios:

#### Individual Secret Backups

Automatic backup files (`.bak`) created when secrets are overwritten, providing protection against accidental data loss:

```bash
# List available individual backups
simple-secrets list backups

# Restore a deleted or overwritten secret
simple-secrets restore secret KEY
```

**When created**: Only when overwriting existing secrets (not on initial creation)
**Location**: `~/.simple-secrets/backups/[key].bak`
**Cross-rotation compatibility**: Individual backups survive master key rotation via automatic re-encryption

#### Master Key Rotation Backups

Complete database snapshots created during master key rotation, preserving entire system state:

```bash
# List rotation backup directories
simple-secrets list backups

# Restore entire database from rotation backup
simple-secrets restore database BACKUP_ID
```

**When created**: Automatically during `simple-secrets rotate master-key`
**Location**: `~/.simple-secrets/backup-[timestamp]/`
**Contains**: Complete encrypted database snapshot before key rotation

## Database Reset & Recovery

### ⚠️ Safe Database Reset Procedure

If you need to completely reset your simple-secrets installation:

**🔴 CRITICAL: Backup first if you have important data!**

```bash
# 1. BACKUP YOUR DATA FIRST (if you want to keep anything)
simple-secrets list backups --token <your-token>
# Copy any important backup directories from ~/.simple-secrets/backups/

# 2. For complete reset, remove the config directory
rm -rf ~/.simple-secrets/

# Or if using custom config directory:
rm -rf $SIMPLE_SECRETS_CONFIG_DIR

# 3. Run setup to create fresh installation
simple-secrets setup
```

### 🛟 Recovery Options

If something goes wrong and you have backups:

```bash
# Option 1: Restore from automatic backup (recommended)
simple-secrets list backups --token <admin-token>
simple-secrets restore database BACKUP_ID --token <admin-token>

# Option 2: Manual recovery from backup directory
# Check ~/.simple-secrets/backups/ for rotation snapshots
# Each rotation creates a timestamped backup with keys + secrets

# Option 3: Restore individual secrets
simple-secrets restore secret SECRET_KEY --token <admin-token>
```

### 🚨 Emergency Recovery

If the database is corrupted and you can't access normal commands:

1. **Check backup directory manually**: `ls ~/.simple-secrets/backups/`
2. **Look for rotation backups**: Directories named like `rotate-2024-01-15-10-30-45/`
3. **Each contains**: `master.key` and `secrets.json` from that point in time
4. **Emergency contact info**: Check error messages - they guide you to recovery options

**Remember**: The error message "Do not delete ~/.simple-secrets/ - your backups contain recoverable data" means exactly that - your backup directory has your recovery options!

## Secret Lifecycle Management

### Disable Secrets

Temporarily hide secrets from normal operations without deleting them:

```bash
# Disable a secret (hides from list, get, etc.)
simple-secrets disable secret api_key --token <admin-token>

# List disabled secrets
simple-secrets list disabled --token <token>
```

### Enable Secrets

Re-enable previously disabled secrets:

```bash
# Re-enable a disabled secret
simple-secrets enable secret api_key --token <admin-token>
```

### Token Management

Disable user tokens for security purposes:

```bash
# Disable by username
simple-secrets disable user alice --token <admin-token>

# Disable by token value
simple-secrets disable token abc123def456 --token <admin-token>

# Re-enable disabled users (generate new tokens)
simple-secrets enable user alice --token <admin-token>

# Generate new token for user (recovery)
simple-secrets rotate token alice --token <admin-token>
```

**Clear Command Structure**: Use `disable user` when you know the username, and `disable token` when you have the actual token value. This makes it clear what you're disabling and removes ambiguity.

## RBAC Permissions

| Permission | Admin | Reader |
|------------|-------|---------|
| Read secrets | ✅ | ✅ |
| Write secrets | ✅ | ❌ |
| Delete secrets | ✅ | ❌ |
| Manage users | ✅ | ❌ |
| Rotate others' tokens | ✅ | ❌ |
| Rotate own token | ✅ | ✅ |
| Rotate master key | ✅ | ❌ |

## Development

```bash
# Run tests
make test

# Development build
make dev

# Release build
make release VERSION=v1.0.0

# Clean build artifacts
make clean
```

## Shell Completion

Enable shell autocompletion for better CLI experience. Completion includes:

- **Commands and subcommands**: `get`, `put`, `list keys`, `disable secret`, etc.
- **Secret names**: Tab-complete existing secret keys for `get`, `delete`, `put`, etc.
- **User/Token operations**: Context-aware completion for user and token management
- **Disable/Enable**: Complete with appropriate secret lists (active vs disabled)

```bash
# Bash completion
simple-secrets completion bash > /etc/bash_completion.d/simple-secrets

# Zsh completion
simple-secrets completion zsh > "${fpath[1]}/_simple-secrets"

# Fish completion
simple-secrets completion fish > ~/.config/fish/completions/simple-secrets.fish

# PowerShell completion
simple-secrets completion powershell > simple-secrets.ps1
```

**Note**: Secret name completion requires authentication (set `SIMPLE_SECRETS_TOKEN` or use `--token` flag).

## Security Considerations

- **Master key protection**: The `master.key` file contains your encryption key. Protect it like a private key.
- **Token security**: Tokens are hashed with SHA-256 before storage
- **Backup encryption**: All backups maintain encryption with their original keys
- **File permissions**: All files created with 0600 (user read/write only)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
