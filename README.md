# Installation & Usage (with Makefile)

You can use the provided Makefile for easy build, install, and cleanup:

```sh
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

The binary will be available as `simple-secrets` in your PATH after install.

# simple-secrets

A lightweight secrets manager for Ansible and GitOps workflows. `simple-secrets` securely stores secrets using AES-256-GCM encryption and supports master key rotation, multiple key protection backends, and easy CLI operations.

## Features

- **AES-256-GCM encryption** for all secrets
- **Master key rotation** with automatic backup cleanup (keeps last 5 rotations)
- **Database backup/restore** from rotation snapshots with timestamp tracking
- **Role-based access control (RBAC)**: admin and reader roles
- **User management**: create users via CLI
- **Token-based authentication**: via flag, env var, or config file
- **Individual secret backup/restore** for accidental deletions/overwrites
- **Simple CLI** for storing, retrieving, listing, and deleting secrets
- **Atomic updates** and backup for safe operations

## Installation

### Prerequisites

- Go 1.20 or newer

### Build from source

```sh
# Clone the repo
$ git clone https://github.com/ishuley/simple-secrets-cli.git
$ cd simple-secrets-cli

# Build the CLI
$ go build -o simple-secrets
```

### Install (Linux/macOS)

```sh
# Move binary to your PATH
$ mv simple-secrets /usr/local/bin/
```


## Quick Start

1. **Create your first secret:**
   ```sh
   $ simple-secrets put api_key "my-secret-value"
   User setup required. Creating initial admin user...
   Generated admin token: token_abc123...
   Secret "api_key" stored securely.
   ```

2. **Retrieve it:**
   ```sh
   $ simple-secrets get api_key --token token_abc123...
   my-secret-value
   ```

3. **List all keys:**
   ```sh
   $ simple-secrets list keys --token token_abc123...
   api_key
   ```

4. **Rotate master key (creates backup):**
   ```sh
   $ simple-secrets rotate master-key --token token_abc123...
   ```


## Master Key Rotation

Rotate your master key and re-encrypt all secrets (with backup):

```sh
simple-secrets rotate master-key --token <admin-token>
```

## User Management

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

## Token Rotation

Rotate authentication tokens for security:

```sh
# Rotate your own token (both admin and reader users)
$ simple-secrets rotate token --token <your-current-token>
✅ Your token has been rotated successfully!
New token: <new-token>

# Admins can rotate other users' tokens
$ simple-secrets rotate token alice --token <admin-token>
Token rotated for user "alice" (reader role).
New token: <new-token>
```

**Note**: Old tokens are immediately invalidated after rotation.

## RBAC & Permissions

- **admin**: Can read, write, rotate keys, manage users.
- **reader**: Can only read/list secrets.

## Backup & Restore

### Individual Secret Backups

Previous secret values are backed up automatically on overwrite or delete. Restore a secret from backup:

```sh
simple-secrets restore secret my_api_key --token <admin-token>
```

### Database Backup & Restore

Master key rotations create timestamped database backups. List available backups:

```sh
simple-secrets list backups --token <admin-token>
```

Restore your entire database from a backup:

```sh
# Restore from most recent backup
simple-secrets restore database --token <admin-token>

# Restore from specific backup
simple-secrets restore database rotate-20240901-143022 --token <admin-token>

# Skip confirmation prompt
simple-secrets restore database --yes --token <admin-token>
```

**Note**: Database restore creates a backup of your current state before restoring, and only keeps the last 5 rotation backups automatically.


## First-Run Experience

If no users exist, a default admin user and token are created. The token is printed to the console. Store it securely; it will not be shown again.

## Token Authentication

Authenticate using:

- `--token <token>` flag
- `SIMPLE_SECRETS_TOKEN` environment variable
- `~/.simple-secrets/config.json` with `{ "token": "<token>" }`

## Documentation

See [docs/getting-started.md](docs/getting-started.md) for a full walkthrough and advanced usage.

## 📋 Manual QA Checklist

> See [`docs/testing-framework.md`](docs/testing-framework.md) for comprehensive AI-agent testing procedures

---
