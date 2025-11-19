# CLAUDE.md - AI Assistant Guide

**Last Updated**: 2025-11-17
**Purpose**: Comprehensive guide for AI assistants working with this codebase

---

## Project Overview

This repository contains a declarative homelab management system for NAB9 mini PCs running Fedora CoreOS. It consists of:

1. **Custom CoreOS Image** - BlueBuild-based image layering on UBlue uCore
2. **Setup CLI Tool** - Go-based interactive wizard for post-installation configuration
3. **Infrastructure as Code** - Butane/Ignition configs, systemd units, compose stacks

**Key Services**: Plex, Jellyfin (hardware transcoding), Nextcloud, Immich, Overseerr, Nginx Proxy Manager
**Architecture**: Intel QuickSync GPU, WireGuard VPN tunnel to VPS, NFS-backed media storage

---

## Repository Structure

```
homelab-coreos-minipc/
├── homelab-setup/              # Go CLI tool (10,275+ lines)
│   ├── cmd/homelab-setup/      # CLI commands & entry point
│   ├── internal/               # Unexported packages
│   │   ├── cli/                # Interactive menu system
│   │   ├── config/             # Configuration & state management
│   │   ├── common/             # Validators (IP, port, CIDR, paths)
│   │   ├── system/             # System operations (packages, services, users)
│   │   ├── steps/              # Setup steps (preflight, user, nfs, wireguard, etc.)
│   │   └── ui/                 # User interface (prompts, spinners, colors)
│   ├── pkg/version/            # Version info (injected at build)
│   ├── go.mod, go.sum          # Dependencies (cobra, color, term)
│   └── Makefile                # Build automation
│
├── recipes/                    # BlueBuild image recipes
│   ├── recipe.yml              # Main recipe (base: ucore, modules)
│   ├── packages.yml            # Package installation manifest
│   └── systemd.yml             # Systemd unit definitions
│
├── files/                      # Files bundled into image
│   ├── scripts/                # Build-time scripts (RPM Fusion)
│   ├── system/                 # Filesystem overlay (/etc, /usr)
│   │   ├── etc/systemd/system/ # Compose & WireGuard services
│   │   ├── etc/profile.d/      # Intel VAAPI environment
│   │   └── usr/local/bin/      # Compiled Go binary
│   └── setup_scripts/          # Legacy bash setup scripts
│
├── ignition/                   # CoreOS first-boot provisioning
│   ├── config.bu.template      # Butane config template
│   └── transpile.sh            # Butane→Ignition transpiler
│
├── docs/                       # Documentation
│   ├── getting-started.md      # Quick setup guide
│   ├── reference/              # CLI manual, Ignition docs
│   ├── testing/                # QA checklists
│   └── dev/                    # Devcontainer, CI pipeline docs
│
├── .github/workflows/          # CI/CD
│   ├── build.yml               # BlueBuild image builds (daily @ 06:00 UTC)
│   └── build-homelab-setup.yml # Go binary builds (auto-commit)
│
├── .devcontainer/              # Development container
├── .vscode/                    # VS Code configuration
└── modules/                    # Custom BlueBuild modules
```

---

## Technology Stack

### Primary Language: Go 1.23.3

**Dependencies** (from `homelab-setup/go.mod`):
- `github.com/spf13/cobra@v1.10.1` - CLI framework
- `github.com/fatih/color@v1.18.0` - Terminal colors
- `golang.org/x/term` - Terminal utilities

### Infrastructure

- **Base OS**: Fedora CoreOS (UBlue uCore variant)
- **Image Build**: BlueBuild v1.8 (YAML-based recipes)
- **Provisioning**: Butane/Ignition (FCOS 1.4.0)
- **Container Runtime**: Podman (primary) or Docker
- **VPN**: WireGuard
- **Storage**: NFS client
- **GPU**: Intel VAAPI (media-driver, libva, ffmpeg)

### CI/CD

- **GitHub Actions**: Image builds, binary compilation
- **Cosign**: Image signing (optional, via secrets)
- **GHCR**: GitHub Container Registry (`ghcr.io/plex-migration-homelab/homelab-coreos-minipc`)

---

## Development Workflows

### Setting Up Development Environment

**Option 1: Devcontainer** (Recommended)
```bash
# Open in VS Code with devcontainer extension
# Includes: Go 1.23, golangci-lint v1.60.1, zsh, git, sudo
```

**Option 2: Local Development**
```bash
cd homelab-setup/
make deps        # Download dependencies
make build       # Build binary
make test        # Run tests
make lint        # Run linter
```

### Common Development Tasks

#### Building the Go CLI


```bash
cd homelab-setup/

# Build for current platform
make build
# Output: bin/homelab-setup

# Build for all architectures
make build-all
# Output: bin/homelab-setup-linux-amd64, bin/homelab-setup-linux-arm64

# Run directly
make run

# Format, vet, and tidy
make fmt vet tidy
```

#### Running Tests

```bash
# All tests
make test

# Verbose with coverage
make test-verbose

# Generate HTML coverage report
make test-coverage
# Open: coverage.html
```

#### Building the Custom Image

**Local Testing** (requires BlueBuild CLI):
```bash
# Install BlueBuild CLI first
# See: https://blue-build.org/learn/getting-started/

# Build recipe
bluebuild build recipes/recipe.yml

# Test in VM (virt-manager recommended)
# See: docs/testing/virt-manager-qa.md
```

**CI/CD** (automatic):
- Push to main → triggers build
- Daily @ 06:00 UTC → rebuilds image
- PR → test build (no push)

### Code Review Checklist

**Before Committing**:
1. **Format**: `make fmt` (gofmt)
2. **Lint**: `make lint` (golangci-lint)
3. **Test**: `make test` (all tests pass)
4. **Vet**: `make vet` (go vet clean)
5. **Build**: `make build` (compiles successfully)

**Go Code Quality**:
- No security vulnerabilities (SQL injection, command injection, XSS)
- Error handling with context: `fmt.Errorf("context: %w", err)`
- Input validation using `internal/common` validators
- Sudo operations use `sudo -n` (non-interactive)
- Paths validated with `ValidateSafePath()` to prevent injection

**File Operations**:
- Use `Read` tool for reading files
- Use `Edit` tool for modifying files
- Use `Write` tool only for NEW files (prefer editing existing)
- Never create unnecessary documentation files

---

## Code Conventions

### Naming Conventions

**Files**:
- Go: `snake_case.go`, `*_test.go`
- Shell: `kebab-case.sh`
- YAML: `kebab-case.yml`

**Go Identifiers**:
- Exported: `PascalCase`
- Unexported: `camelCase`
- Constants: `PascalCase` or `SCREAMING_SNAKE_CASE`

**Configuration Keys**:
- All uppercase with underscores: `HOMELAB_USER`, `NFS_SERVER`, `CONTAINER_RUNTIME`

**Shell Variables**:
- Exported/config: `UPPERCASE=value`
- Internal: `lowercase=value`

### Code Style

**EditorConfig** (`.editorconfig`):
```ini
[*.go]
indent_style = tab
indent_size = 4

[*.{yml,yaml,md}]
indent_style = space
indent_size = 2

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
```

**VS Code Settings** (`.vscode/settings.json`):
- Linter: `golangci-lint`
- Format on save: enabled
- Organize imports on save: enabled
- Build flags: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64`
- Rulers: 100, 120 columns

### Error Handling Patterns

**Standard Pattern**:
```go
if err != nil {
    return fmt.Errorf("descriptive context: %w", err)
}
```

**System Operations**:
```go
cmd := exec.Command("sudo", "-n", "mkdir", "-p", path)
if output, err := cmd.CombinedOutput(); err != nil {
    return fmt.Errorf("failed to create directory %s: %w\nOutput: %s",
        path, err, string(output))
}
```

**User-Facing Errors**:
```go
// Use UI methods for colored output
ui.Error("Operation failed: " + err.Error())
ui.Warning("This is a warning")
ui.Success("Operation completed")
ui.Info("Information message")
```

### Validation

**Always validate user input** using `internal/common` validators:

```go
import "github.com/plex-migration-homelab/homelab-coreos-minipc/homelab-setup/internal/common"

// IP addresses (IPv4 only)
if err := common.ValidateIP("192.168.1.1"); err != nil {
    return err
}

// Ports (1-65535)
if err := common.ValidatePort("8080"); err != nil {
    return err
}

// CIDR blocks
if err := common.ValidateCIDR("10.0.0.0/24"); err != nil {
    return err
}

// Paths (absolute only)
if err := common.ValidatePath("/srv/containers"); err != nil {
    return err
}

// Safe paths (no shell metacharacters) - USE THIS for system commands
if err := common.ValidateSafePath(userInput); err != nil {
    return err
}
```

**Available Validators** (`homelab-setup/internal/common/validation.go`):
- `ValidateIP(ip string)` - IPv4 addresses
- `ValidatePort(port string)` - 1-65535
- `ValidateCIDR(cidr string)` - IPv4 CIDR blocks
- `ValidatePath(path string)` - Absolute paths
- `ValidateSafePath(path string)` - Paths safe for system commands (no metacharacters)
- `ValidateUsername(username string)` - Alphanumeric + dash
- `ValidateDomain(domain string)` - FQDN validation
- `ValidateTimezone(tz string)` - Timezone strings

---

## Testing Practices

### Test Organization

**File Location**: Co-located with source (`*_test.go`)

**Test Structure** (table-driven):
```go
func TestValidateIP(t *testing.T) {
    tests := []struct {
        name    string
        ip      string
        wantErr bool
    }{
        {"valid IPv4", "192.168.1.1", false},
        {"invalid - empty", "", true},
        {"invalid - too high", "256.1.1.1", true},
        {"invalid - IPv6", "::1", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateIP(tt.ip)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateIP() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Coverage

**Current Coverage** (as of 2025-11-17):
- `internal/common`: 100+ test cases (IP, port, CIDR, path, username, domain, timezone)
- `internal/config`: Config file operations, markers
- `internal/steps`: Individual setup steps
- `internal/system`: System operations

**Running Coverage**:
```bash
make test-coverage
# Opens: coverage.html
```

### Security Testing

**Always test for**:
1. Command injection via paths (use `ValidateSafePath`)
2. Shell metacharacter filtering
3. Invalid IP addresses/CIDR blocks
4. Port ranges (1-65535)
5. Empty/nil inputs
6. Overly long inputs

---

## Build & Deployment

### Go Binary Build

**Makefile Targets**:
```bash
make build       # Single architecture (current)
make build-all   # Multi-arch (amd64, arm64)
make install     # Install to GOPATH/bin
```

**Version Injection**:
```makefile
# Automatically set via ldflags
VERSION=0.1.0-dev
GIT_COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
```

**CI/CD Binary Build** (`.github/workflows/build-homelab-setup.yml`):
1. Triggered on push to `homelab-setup/**`
2. Runs tests: `go test ./... -v`
3. Builds binary: `make build`
4. Copies to: `/files/system/usr/bin/homelab-setup`
5. Auto-commits if changed (git-actions bot)
6. Uploads artifact (30-day retention)

### Image Build

**BlueBuild Recipe** (`recipes/recipe.yml`):
```yaml
base-image: ghcr.io/ublue-os/ucore
image-version: stable

modules:
  - type: script
    scripts: [install-rpmfusion-release.sh]
  - from-file: packages.yml       # Package installation
  - type: files                   # Filesystem overlay
    files:
      - source: system
        destination: /
  - from-file: systemd.yml        # Service units
  - type: signing                 # Cosign signing
```

**Build Schedule**:
- **Daily**: 06:00 UTC (20 min after UBlue upstream builds)
- **On Push**: Automatic (ignores `*.md` changes)
- **On PR**: Test build (no push)
- **Manual**: `workflow_dispatch`

**Build Output**:
- Registry: `ghcr.io/plex-migration-homelab/homelab-coreos-minipc:latest`
- Tags: `latest`, git commit SHA
- Signing: Cosign (if `SIGNING_SECRET` set)

### First-Boot Deployment

**Process**:
1. **Install FCOS** with Ignition from `ignition/config.bu.template`
2. **First Boot**: Auto-rebase to custom image via systemd units
3. **Post-Rebase**: SSH as `core`, run `~/setup/homelab-setup`
4. **Interactive Setup**: Wizard configures user, WireGuard, NFS, containers

**Ignition Template** (`ignition/config.bu.template`):
- Creates `core` user with SSH keys
- Sets password hash
- Enables auto-rebase service
- Creates directories (`~/setup/`)

---

## Configuration Management

### Config File Format

**Location**: `~/.homelab-setup.conf`
**Format**: INI-style (simple `key=value`)

**Example**:
```ini
CONTAINER_RUNTIME=podman
HOMELAB_USER=containeruser
PUID=1001
PGID=1001
TZ=America/Chicago
NFS_SERVER=192.168.7.10
NFS_MEDIA_PATH=/volume1/media
WG_SERVER_ENDPOINT=vpn.example.com:51820
```

### Completion Markers

**Location**: `~/.local/homelab-setup/`
**Files**:
- `preflight-complete`
- `user-setup-complete`
- `directory-setup-complete`
- `wireguard-setup-complete`
- `nfs-setup-complete`
- `container-setup-complete`
- `service-deployment-complete`

**Usage**: Touch files to mark steps complete, remove to re-run

### Preseed Support

**Environment Variables** (auto-seeds config):
- `HOMELAB_USER` - Service account username
- `SETUP_USER` - Legacy key (maps to `HOMELAB_USER`)

**Example**:
```bash
HOMELAB_USER=containeruser ./homelab-setup
# Skips user input prompt for username
```

---

## Key Files Reference

### Must-Read Files for New Contributors

| File | Purpose | Lines |
|------|---------|-------|
| `homelab-setup/cmd/homelab-setup/main.go` | CLI entry point | ~100 |
| `homelab-setup/internal/cli/menu.go` | Interactive menu | ~400 |
| `homelab-setup/internal/common/validation.go` | Input validators | ~200 |
| `homelab-setup/internal/config/config.go` | Configuration management | ~300 |
| `homelab-setup/internal/steps/preflight.go` | Preflight checks | ~150 |
| `recipes/recipe.yml` | Image build recipe | ~25 |
| `.github/workflows/build.yml` | Image CI/CD | ~45 |
| `docs/getting-started.md` | User quickstart | ~200 |

### Critical Security Files

| File | Security Concern | Pattern |
|------|------------------|---------|
| `homelab-setup/internal/common/validation.go` | Command injection prevention | `ValidateSafePath()` |
| `homelab-setup/internal/system/filesystem.go` | Sudo operations | `sudo -n` only |
| `homelab-setup/internal/system/commandrunner.go` | Command execution | `exec.Command()` (no shell) |

### Build Configuration

| File | Purpose |
|------|---------|
| `homelab-setup/Makefile` | Build automation |
| `homelab-setup/go.mod` | Go dependencies |
| `.editorconfig` | Cross-editor formatting |
| `.vscode/settings.json` | VS Code Go configuration |
| `.github/workflows/build-homelab-setup.yml` | Binary CI/CD |

---

## Important Patterns to Follow

### 1. User Interaction Pattern

```go
// Always use UI methods from internal/ui
ui := ui.NewUI()

// Prompts with validation
username, err := ui.PromptWithValidation(
    "Enter username",
    "containeruser",
    common.ValidateUsername,
)

// Confirmation
confirmed, err := ui.PromptConfirm("Proceed?", true)

// Password (hidden input)
password, err := ui.PromptPassword("Enter password")

// Selection
choice, err := ui.PromptSelect("Choose runtime", []string{"podman", "docker"})

// Output
ui.Success("Operation completed")
ui.Error("Operation failed: " + err.Error())
ui.Warning("This is a warning")
ui.Info("Information message")

// Spinner for long operations
spinner := ui.NewSpinner("Processing...")
spinner.Start()
// ... do work ...
spinner.Stop()
ui.Success("Done!")
```

### 2. Configuration Pattern

```go
import "github.com/plex-migration-homelab/homelab-coreos-minipc/homelab-setup/internal/config"

// Load config
cfg, err := config.NewConfig("~/.homelab-setup.conf")

// Read values
user := cfg.Get(config.KeyHomelabUser, "defaultuser")
runtime := cfg.Get(config.KeyContainerRuntime, "podman")

// Write values
if err := cfg.Set(config.KeyPUID, "1001"); err != nil {
    return err
}

// Markers
markers := config.NewMarkers("~/.local/homelab-setup")
if markers.IsComplete("preflight") {
    fmt.Println("Preflight already completed")
}
markers.MarkComplete("preflight")
```

### 3. System Operations Pattern

```go
import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"

// Check sudo
if !system.HasSudo() {
    return fmt.Errorf("sudo access required")
}

// Create directory (with sudo)
if err := system.CreateDirectory("/srv/containers/media", "containeruser", "containeruser", 0755); err != nil {
    return err
}

// Install packages (rpm-ostree)
packages := []string{"wireguard-tools", "nfs-utils"}
if err := system.InstallPackages(packages); err != nil {
    return err
}

// Enable systemd service
if err := system.EnableService("wg-quick@wg0"); err != nil {
    return err
}
```

### 4. Setup Step Pattern

```go
import "github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/steps"

// Each step implements this interface
type Step interface {
    Name() string
    Description() string
    Run() error
    IsComplete() bool
    Skip() bool
}

// Example step structure
type PreflightStep struct {
    ui      *ui.UI
    config  *config.Config
    markers *config.Markers
}

func (s *PreflightStep) Run() error {
    s.ui.Info("Running preflight checks...")

    // Check rpm-ostree
    if !system.IsRpmOstree() {
        return fmt.Errorf("not running rpm-ostree")
    }

    // Check packages
    required := []string{"podman", "git", "wireguard-tools"}
    for _, pkg := range required {
        if !system.IsPackageInstalled(pkg) {
            return fmt.Errorf("required package not installed: %s", pkg)
        }
    }

    // Mark complete
    s.markers.MarkComplete("preflight")
    s.ui.Success("Preflight checks passed!")
    return nil
}
```

---

## Common Pitfalls to Avoid

### Security

1. **Never execute shell commands with user input**
   - ❌ `exec.Command("sh", "-c", "mkdir " + userPath)`
   - ✅ `exec.Command("sudo", "-n", "mkdir", "-p", userPath)`

2. **Always validate paths before system operations**
   - ❌ `system.CreateDirectory(userInput, ...)`
   - ✅ `ValidateSafePath(userInput); system.CreateDirectory(userInput, ...)`

3. **Never use interactive sudo**
   - ❌ `sudo mkdir /srv/containers`
   - ✅ `sudo -n mkdir /srv/containers` (fail if password required)

### File Operations

1. **Prefer editing existing files over creating new ones**
   - ❌ Creating new documentation files
   - ✅ Updating existing README.md, docs/

2. **Don't use bash for file operations**
   - ❌ `bash cat file.txt`
   - ✅ `Read` tool

3. **Preserve exact indentation when editing**
   - Read file first, note indentation style (tabs vs spaces)
   - Match existing style in edits

### Testing

1. **Always write table-driven tests**
   - ❌ One test per case
   - ✅ Table-driven with multiple cases

2. **Test edge cases**
   - Empty strings
   - Nil values
   - Maximum values (ports: 65535, IPs: 255.255.255.255)
   - Invalid formats

3. **Run tests before committing**
   - `make test` must pass
   - `make lint` must be clean

---

## Quick Reference Commands

### Development

```bash
# Setup
cd homelab-setup/
make deps

# Build & Test
make build test lint

# Coverage
make test-coverage
open coverage.html

# Format & Clean
make fmt vet tidy clean

# Multi-arch build
make build-all
```

### Git Workflow

```bash
# Create feature branch (must start with claude/)
git checkout -b claude/feature-name-<session-id>

# Commit changes
git add .
git commit -m "feat: Add feature description"

# Push (with retry on network errors)
git push -u origin claude/feature-name-<session-id>
```

### Testing Image Locally

```bash
# Build image (requires BlueBuild CLI)
bluebuild build recipes/recipe.yml

# Test in virt-manager
# See: docs/testing/virt-manager-qa.md
```

### Debugging

```bash
# Check Go binary version
./bin/homelab-setup version

# Check setup status
./bin/homelab-setup status

# Run troubleshooting
./bin/homelab-setup troubleshoot

# Check markers
ls -la ~/.local/homelab-setup/

# Check config
cat ~/.homelab-setup.conf
```

---

## Getting Help

### Documentation

- **Getting Started**: `docs/getting-started.md`
- **CLI Manual**: `docs/reference/homelab-setup-cli.md`
- **Ignition Guide**: `docs/reference/ignition.md`
- **Testing**: `docs/testing/virt-manager-qa.md`
- **Go Implementation**: `homelab-setup/README.md`

### External Resources

- **BlueBuild Docs**: https://blue-build.org/
- **Fedora CoreOS**: https://docs.fedoraproject.org/en-US/fedora-coreos/
- **Butane Configs**: https://coreos.github.io/butane/
- **UBlue**: https://universal-blue.org/

### Project Structure

- **Main Branch**: Production-ready code
- **Feature Branches**: `claude/<description>-<session-id>`
- **PR Process**: Test build runs automatically, requires passing tests

---

## Change Log

| Date | Change |
|------|--------|
| 2025-11-17 | Initial CLAUDE.md creation |

---

**For AI Assistants**: This document is your primary reference. Follow the patterns, conventions, and security practices outlined here. When in doubt, examine existing code in similar files for examples. Always prioritize security (validate inputs, use safe paths, no shell injection) and maintainability (tests, documentation, clear error messages).
