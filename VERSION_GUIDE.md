# Version Management Quick Reference

## Daily Development
```bash
make dev          # Build with git commit info (dev-abc1234)
```

## When Ready to Release

### 1. Choose Version Number
Ask yourself:
- **Breaking changes?** → Bump MAJOR (v1.x.x → v2.0.0)
- **New features?** → Bump MINOR (v1.0.x → v1.1.0)  
- **Bug fixes only?** → Bump PATCH (v1.0.0 → v1.0.1)

### 2. Build and Test
```bash
make release VERSION=vX.Y.Z
./simple-secrets version              # Verify version
```

### 3. Tag and Push (Optional)
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Examples

### Your First Release
```bash
make release VERSION=v1.0.0
```

### Bug Fix Release  
```bash
make release VERSION=v1.0.1
```

### New Feature Release
```bash
make release VERSION=v1.1.0
```

### Major Update (Breaking Changes)
```bash
make release VERSION=v2.0.0
```

### Beta/Pre-release
```bash
make release VERSION=v1.1.0-beta.1
```

## Current Version Check
```bash
./simple-secrets version --short     # See current version
git tag --sort=-version:refname      # See all your releases
```
# Test comment
