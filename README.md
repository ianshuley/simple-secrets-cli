# Simple Secrets CLI

A secure command-line tool for managing secrets with role-based access control (RBAC), designed for teams that need a simple, file-based secret management solution without complex infrastructure requirements.

## Features

- **🔐 Secure Storage**: AES-256-GCM encryption for all secrets
- **👥 Role-Based Access Control**: Admin and reader roles with granular permissions
- **🔄 Token Management**: Secure token rotation with self-service capabilities
- **💾 Backup & Restore**: Automatic backups with encrypted restore capabilities
- **🎯 Simple CLI**: Intuitive commands for common secret operations
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

When you first run `simple-secrets`, it will automatically detect this and prompt you to create an admin user:

```bash
# Any command requiring authentication will trigger setup
simple-secrets list keys

# Or run setup explicitly
simple-secrets --setup
```

Setup process:

1. **Detects first run** and prompts for admin user creation
2. **Creates admin user** and generates secure authentication token
3. **Displays token once** - save it securely as it won't be shown again
4. **Requires re-running** your original command with the token

### Basic Usage

```bash
# First run: triggers setup and creates admin user
simple-secrets list keys
# (Shows setup prompt, creates admin, displays token)

# Then use the token for actual operations:
simple-secrets list keys --token YOUR_ADMIN_TOKEN

# Store a secret (values with dashes work naturally in quotes)
simple-secrets put api-key "--prod-key-abc123" --token YOUR_ADMIN_TOKEN

# Retrieve a secret
simple-secrets get api-key --token YOUR_ADMIN_TOKEN
```

## Configuration

### Authentication Methods

1. **Command flag**: `--token YOUR_TOKEN`
2. **Environment variable**: `export SIMPLE_SECRETS_TOKEN=YOUR_TOKEN`
3. **Config file**: `~/.simple-secrets/config.json`

```json
{
  "token": "YOUR_TOKEN"
}
```

### File Structure

```text
~/.simple-secrets/
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

#### Working with Complex Values

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

# Disable user tokens
simple-secrets disable token USERNAME
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

```bash
# List available backups
simple-secrets list backups

# Restore a deleted secret
simple-secrets restore secret KEY

# Restore entire database from backup
simple-secrets restore database BACKUP_ID
```

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
# Disable a user's token (admin only)
simple-secrets disable token alice --token <admin-token>

# Generate new token for user (recovery)
simple-secrets rotate token alice --token <admin-token>
```

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

Enable shell autocompletion for better CLI experience:

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

## Security Considerations

- **Master key protection**: The `master.key` file contains your encryption key. Protect it like a private key.
- **Token security**: Tokens are hashed with SHA-256 before storage
- **Backup encryption**: All backups maintain encryption with their original keys
- **File permissions**: All files created with 0600 (user read/write only)

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
