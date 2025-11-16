# Go Codebase Review - Findings and Recommendations
**Date:** November 16, 2025
**Reviewer:** AI Code Audit
**Scope:** homelab-setup Go application

## Executive Summary

Overall code quality is **GOOD** with clean architecture, proper separation of concerns, and comprehensive error handling. However, there are several areas needing improvement regarding user experience, error recovery, configuration management, and operational robustness.

**Priority Issues:** 3 Critical, 7 High, 12 Medium, 8 Low

---

## Critical Issues (Must Fix)

### 1. **No Rollback/Undo Mechanism** 🔴
**Location:** All step files
**Impact:** User cannot recover from failed steps

**Problem:**
- If NFS mount fails after updating `/etc/fstab`, the entry remains
- If service creation fails, partial configuration is left behind
- No way to cleanly undo a half-completed step

**Recommendation:**
```go
// Add rollback tracking to each step
type RollbackAction struct {
    Description string
    Undo func() error
}

type Step interface {
    Execute() error
    Rollback() error  // New method
}

// In each step, track actions for rollback
func (n *NFSConfigurator) Run() error {
    rollbacks := []RollbackAction{}
    defer func() {
        if r := recover(); r != nil {
            n.rollbackAll(rollbacks)
        }
    }()

    // Track each action
    if err := n.CreateMountPoint(mountPoint); err != nil {
        return err
    }
    rollbacks = append(rollbacks, RollbackAction{
        Description: "Remove mount point",
        Undo: func() error { return os.RemoveAll(mountPoint) },
    })
    // ... continue with rollback tracking
}
```

### 2. **Race Condition in Config Save** 🔴
**Location:** `internal/config/config.go` lines 92-145
**Impact:** Potential config corruption in concurrent scenarios

**Problem:**
```go
// Set() calls Save() which does atomic write, but Get() doesn't lock
func (c *Config) Set(key, value string) error {
    c.data[key] = value  // No mutex here
    return c.Save()       // Another goroutine could read during save
}
```

**Recommendation:**
```go
type Config struct {
    filePath string
    data     map[string]string
    loaded   bool
    mu       sync.RWMutex  // Add mutex
}

func (c *Config) Set(key, value string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    // ... rest of implementation
}

func (c *Config) Get(key string) (string, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    // ... rest of implementation
}
```

### 3. **Sudo Command Failures Not User-Friendly** 🔴
**Location:** `internal/steps/nfs.go` line 179, multiple locations
**Impact:** Cryptic errors when sudo fails

**Problem:**
```go
if output, err := n.runner.Run("sudo", "-n", "systemctl", "daemon-reload"); err != nil {
    return fmt.Errorf("failed to reload systemd after fstab update: %w\nOutput: %s", err, output)
}
```
The `-n` flag causes silent failure if password is required.

**Recommendation:**
```go
// Add helper to detect sudo configuration
func (s *System) RequiresPassword() (bool, error) {
    _, err := exec.Command("sudo", "-n", "true").CombinedOutput()
    return err != nil, nil
}

// In preflight check
func (p *PreflightChecker) CheckSudoAccess() error {
    requiresPwd, _ := system.RequiresPassword()
    if requiresPwd {
        p.ui.Warning("sudo requires password authentication")
        p.ui.Info("For unattended operation, configure passwordless sudo:")
        p.ui.Info("  echo '$USER ALL=(ALL) NOPASSWD: ALL' | sudo tee /etc/sudoers.d/homelab")

        // Try interactive sudo
        if err := exec.Command("sudo", "true").Run(); err != nil {
            return fmt.Errorf("sudo authentication failed")
        }
    }
    return nil
}
```

---

## High Priority Issues

### 4. **Config Keys Are Magic Strings** ⚠️
**Location:** Throughout codebase
**Impact:** Typos cause silent failures, no autocomplete

**Problem:**
```go
// Easy to mistype
host := n.config.GetOrDefault("NFS_SERVER", "")
export := n.config.GetOrDefault("NFS_EXPORT", "")  // Could typo as "NSF_EXPORT"
```

**Recommendation:**
```go
// Add constants
package config

const (
    KeyNFSServer      = "NFS_SERVER"
    KeyNFSExport      = "NFS_EXPORT"
    KeyNFSMountPoint  = "NFS_MOUNT_POINT"
    KeyHomelabBaseDir = "HOMELAB_BASE_DIR"
    // ... etc
)

// Then use: host := n.config.GetOrDefault(config.KeyNFSServer, "")
```

### 5. **No Dry-Run Mode** ⚠️
**Location:** All steps
**Impact:** Users scared to run unknown commands

**Problem:**
- No way to preview what will be done
- Can't test without actually making changes

**Recommendation:**
```go
// Add to SetupContext
type SetupContext struct {
    // ... existing fields
    DryRun bool
}

// In each step
func (n *NFSConfigurator) CreateMountPoint(mountPoint string) error {
    if n.ctx.DryRun {
        n.ui.Info(fmt.Sprintf("[DRY-RUN] Would create directory: %s", mountPoint))
        return nil
    }
    // actual implementation
}

// Add CLI flag
var dryRunFlag bool
rootCmd.PersistentFlags().BoolVar(&dryRunFlag, "dry-run", false, "Preview actions without executing")
```

### 6. **Network Timeout Too Short** ⚠️
**Location:** `internal/steps/preflight.go` line 166
**Impact:** Fails on slow networks

**Problem:**
```go
reachable, err := p.network.TestConnectivity("8.8.8.8", 3)  // Only 3 retries
```

**Recommendation:**
```go
// Make configurable
const (
    DefaultNetworkTimeout = 10 * time.Second
    DefaultNetworkRetries = 5
)

// Allow override via config
retries := p.config.GetOrDefault("NETWORK_TEST_RETRIES", "5")
timeout := p.config.GetOrDefault("NETWORK_TEST_TIMEOUT", "10")
```

### 7. **Step Completion Markers Can Desync** ⚠️
**Location:** `internal/config/markers.go`
**Impact:** Config and markers can disagree on state

**Problem:**
- Marker says "NFS complete" but config has no NFS_SERVER
- User could manually delete config file but leave markers
- No validation between marker state and actual system state

**Recommendation:**
```go
// Add validation step
func (sm *StepManager) ValidateSetupState() []string {
    issues := []string{}

    if sm.IsStepComplete("nfs-setup-complete") {
        if !sm.config.Exists("NFS_SERVER") {
            issues = append(issues, "NFS marked complete but no server configured")
        }
    }

    if sm.IsStepComplete("user-setup-complete") {
        username := sm.config.GetOrDefault("HOMELAB_USER", "")
        if username != "" {
            if _, err := user.Lookup(username); err != nil {
                issues = append(issues, fmt.Sprintf("User %s marked configured but doesn't exist", username))
            }
        }
    }

    return issues
}

// Call in status command
func (m *Menu) showStatus() error {
    // ... existing code

    issues := m.ctx.Steps.ValidateSetupState()
    if len(issues) > 0 {
        m.ctx.UI.Warning("Configuration inconsistencies detected:")
        for _, issue := range issues {
            m.ctx.UI.Warningf("  - %s", issue)
        }
    }
}
```

### 8. **No Progress Indication for Long Operations** ⚠️
**Location:** Container pulls, NFS mounts, etc.
**Impact:** User thinks program is frozen

**Problem:**
```go
// This can take 5+ minutes with no feedback
if output, err := n.runner.Run("sudo", "-n", "mount", mountPoint); err != nil {
    return fmt.Errorf("failed to mount %s: %w\nOutput: %s", mountPoint, err, output)
}
```

**Recommendation:**
```go
// Add progress callback
type CommandWithProgress struct {
    runner CommandRunner
    ui     *ui.UI
}

func (c *CommandWithProgress) RunWithSpinner(description string, name string, args ...string) (string, error) {
    done := make(chan bool)

    go func() {
        spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
        i := 0
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()

        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                fmt.Printf("\r%s %s", spinner[i%len(spinner)], description)
                i++
            }
        }
    }()

    output, err := c.runner.Run(name, args...)
    done <- true
    fmt.Print("\r" + strings.Repeat(" ", 80) + "\r")  // Clear spinner

    return output, err
}
```

### 9. **Menu Doesn't Show Which Steps Failed** ⚠️
**Location:** `internal/cli/menu.go` displayMenu
**Impact:** User doesn't know what to fix

**Problem:**
```go
// Only shows ✓ or blank, not ✗ for failed
if m.ctx.Steps.IsStepComplete(step.MarkerName) {
    status = green.Sprint("✓")
}
```

**Recommendation:**
```go
// Add failure tracking
type StepStatus int
const (
    StepNotStarted StepStatus = iota
    StepInProgress
    StepCompleted
    StepFailed
)

// Store in markers directory
type Markers struct {
    // ... existing fields
}

func (m *Markers) GetStatus(name string) StepStatus {
    if exists, _ := m.Exists(name + "-failed"); exists {
        return StepFailed
    }
    if exists, _ := m.Exists(name); exists {
        return StepCompleted
    }
    return StepNotStarted
}

func (m *Markers) MarkFailed(name string) error {
    return m.Create(name + "-failed")
}

// In menu display
status := "  "
switch m.ctx.Steps.GetStepStatus(step.MarkerName) {
    case StepCompleted:
        status = green.Sprint("✓")
    case StepFailed:
        status = red.Sprint("✗")
    case StepInProgress:
        status = yellow.Sprint("⋯")
}
```

### 10. **WireGuard Keys Stored in Plain Text** ⚠️
**Location:** `internal/steps/wireguard.go`
**Impact:** Private keys exposed in config file (mode 0600)

**Problem:**
```go
// Keys written to readable config file
cfg.Set("WG_PRIVATE_KEY", privateKey)
cfg.Set("WG_PUBLIC_KEY", publicKey)
```

**Current state:** File has mode 0600, which is acceptable, but keys should ideally only be in WireGuard config.

**Recommendation:**
```go
// Don't store in config.conf, only in /etc/wireguard/wg0.conf (already mode 0600)
// Just store a reference
cfg.Set("WG_CONFIG_PATH", "/etc/wireguard/wg0.conf")
cfg.Set("WG_INTERFACE", "wg0")

// When displaying status, read from actual config
func (w *WireGuardSetup) GetPublicKey() (string, error) {
    configPath := w.config.GetOrDefault("WG_CONFIG_PATH", "/etc/wireguard/wg0.conf")
    content, err := os.ReadFile(configPath)
    // Parse and extract PublicKey line
}
```

---

## Medium Priority Issues

### 11. **No Log File for Debugging** 📋
**Location:** Throughout
**Impact:** Hard to troubleshoot user issues

**Recommendation:**
```go
// Add logging to file and stderr
package ui

import "log"

type UI struct {
    // ... existing fields
    logger *log.Logger
}

func New() *UI {
    logFile, _ := os.OpenFile(
        filepath.Join(os.Getenv("HOME"), ".homelab-setup.log"),
        os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600,
    )

    return &UI{
        // ... existing initialization
        logger: log.New(logFile, "", log.LstdFlags),
    }
}

func (u *UI) Info(msg string) {
    u.logger.Printf("[INFO] %s", msg)  // Log to file
    u.colorInfo.Fprintf(u.output, "[INFO] %s\n", msg)  // Display to user
}
```

### 12. **Preflight Doesn't Check Disk Space** 📋
**Location:** `internal/steps/preflight.go`
**Impact:** Setup fails halfway through

**Recommendation:**
```go
func (p *PreflightChecker) CheckDiskSpace() error {
    p.ui.Info("Checking available disk space...")

    // Check /var (where containers store data)
    varStat := syscall.Statfs_t{}
    if err := syscall.Statfs("/var", &varStat); err != nil {
        return fmt.Errorf("failed to stat /var: %w", err)
    }

    availableGB := (varStat.Bavail * uint64(varStat.Bsize)) / (1024 * 1024 * 1024)
    requiredGB := uint64(20)  // Minimum 20GB recommended

    if availableGB < requiredGB {
        p.ui.Warning(fmt.Sprintf("Low disk space: %dGB available, %dGB recommended", availableGB, requiredGB))
    } else {
        p.ui.Success(fmt.Sprintf("Sufficient disk space: %dGB available", availableGB))
    }

    return nil
}
```

### 13. **Container Runtime Detection Is Binary** 📋
**Location:** `internal/steps/preflight.go` CheckContainerRuntime
**Impact:** Doesn't detect podman-docker wrapper

**Problem:**
```go
// Could have podman-docker wrapper making "docker" command work
if system.CommandExists("podman") {
    // ...
} else if system.CommandExists("docker") {
    // ...
}
```

**Recommendation:**
```go
func (s *System) DetectContainerRuntime() (runtime, version string, err error) {
    // Check podman first
    if CommandExists("podman") {
        output, err := exec.Command("podman", "--version").CombinedOutput()
        if err == nil {
            return "podman", strings.TrimSpace(string(output)), nil
        }
    }

    // Check docker
    if CommandExists("docker") {
        // Detect if it's actually podman-docker wrapper
        output, err := exec.Command("docker", "version", "--format", "{{.Server.Os}}").CombinedOutput()
        if err == nil && strings.Contains(string(output), "podman") {
            return "podman-docker", strings.TrimSpace(string(output)), nil
        }

        output, err = exec.Command("docker", "--version").CombinedOutput()
        if err == nil {
            return "docker", strings.TrimSpace(string(output)), nil
        }
    }

    return "", "", fmt.Errorf("no container runtime found")
}
```

### 14. **Error Messages Missing Context** 📋
**Location:** Throughout
**Impact:** Hard to diagnose issues

**Problem:**
```go
return fmt.Errorf("failed to mount %s: %w\nOutput: %s", mountPoint, err, output)
```
Doesn't tell user *why* mount failed or what to check.

**Recommendation:**
```go
func (n *NFSConfigurator) MountNFS(mountPoint string) error {
    output, err := n.runner.Run("sudo", "-n", "mount", mountPoint)
    if err != nil {
        // Parse common error patterns
        if strings.Contains(output, "access denied") {
            return fmt.Errorf("NFS mount failed: Access denied. Check NFS export permissions on server")
        }
        if strings.Contains(output, "No route to host") {
            return fmt.Errorf("NFS mount failed: Cannot reach NFS server. Check network and firewall")
        }
        if strings.Contains(output, "Connection refused") {
            return fmt.Errorf("NFS mount failed: NFS service not running on server")
        }

        return fmt.Errorf("failed to mount %s: %w\nOutput: %s\nTry: showmount -e %s",
            mountPoint, err, output, n.config.GetOrDefault("NFS_SERVER", ""))
    }
    return nil
}
```

### 15. **Menu Number Choices Don't Match Step Indices** 📋
**Location:** `internal/cli/menu.go` displayMenu
**Impact:** Confusing - menu shows 0-6 but steps array is 0-indexed

**Current:**
```
  [0] ✓ Pre-flight Check
  [1]   User Setup
```

**Recommendation:**
```
  [1] ✓ Pre-flight Check     (or keep 0-indexed but make it clear)
  [2]   User Setup
```

Better: Use letters for individual steps to avoid confusion with "run all" (A/Q):
```
  [P] ✓ Pre-flight Check
  [U]   User Setup
  [D]   Directory Setup
```

### 16. **No Way to Skip Optional Steps Non-Interactively** 📋
**Location:** CLI commands
**Impact:** Can't automate with `--skip-wireguard` flag

**Recommendation:**
```go
// Add flags to run command
var skipSteps []string
runCmd.Flags().StringSliceVar(&skipSteps, "skip", []string{}, "Steps to skip (wireguard,nfs)")

// In RunAll
func (sm *StepManager) RunAll(skipSteps []string) error {
    skipMap := make(map[string]bool)
    for _, s := range skipSteps {
        skipMap[s] = true
    }

    for _, step := range allSteps {
        if skipMap[step.ShortName] {
            sm.ui.Infof("Skipping %s (--skip flag)", step.Name)
            continue
        }
        // ... run step
    }
}
```

### 17. **Validation Happens Too Late** 📋
**Location:** Each step validates individually
**Impact:** Fails after already making changes

**Recommendation:**
```go
// Add validation phase before execution
type Step interface {
    Validate() error
    Execute() error
    Rollback() error
}

func (sm *StepManager) RunAll(skipWireGuard bool) error {
    // Phase 1: Validate all steps
    sm.ui.Header("Validating Setup Configuration")
    for _, step := range steps {
        if err := step.Validate(); err != nil {
            return fmt.Errorf("validation failed for %s: %w", step.Name, err)
        }
    }

    // Phase 2: Execute (only after all validate)
    sm.ui.Header("Executing Setup Steps")
    for _, step := range steps {
        if err := step.Execute(); err != nil {
            return err
        }
    }
}
```

### 18. **No Service Health Checks** 📋
**Location:** `internal/steps/deployment.go`
**Impact:** Services marked "deployed" but actually failing

**Recommendation:**
```go
func (d *Deployment) VerifyServiceHealth(serviceInfo *ServiceInfo) error {
    d.ui.Info("Verifying service health...")

    // Wait up to 60 seconds for service to stabilize
    for i := 0; i < 60; i++ {
        active, err := d.services.IsServiceActive(serviceInfo.UnitName)
        if err != nil {
            return fmt.Errorf("failed to check service status: %w", err)
        }

        if active {
            d.ui.Success("Service is running")

            // Check container status
            runtime, _ := d.getRuntimeFromConfig()
            if runtime == system.RuntimePodman {
                // Verify containers are actually up
                output, _ := exec.Command("podman", "ps", "--filter",
                    fmt.Sprintf("label=io.podman.compose.project=%s", serviceInfo.Name)).Output()

                if len(output) == 0 {
                    return fmt.Errorf("service active but no containers running")
                }
            }

            return nil
        }

        time.Sleep(1 * time.Second)
    }

    return fmt.Errorf("service failed to become active within 60 seconds")
}
```

### 19. **Config Migration Not Handled** 📋
**Location:** `internal/config/config.go`
**Impact:** Breaking changes require manual migration

**Recommendation:**
```go
const ConfigVersion = 2

func (c *Config) Load() error {
    // ... existing load logic

    version := c.GetOrDefault("CONFIG_VERSION", "1")
    if version != fmt.Sprint(ConfigVersion) {
        if err := c.Migrate(version); err != nil {
            return fmt.Errorf("config migration failed: %w", err)
        }
    }

    return nil
}

func (c *Config) Migrate(fromVersion string) error {
    switch fromVersion {
    case "1":
        // Migrate v1 -> v2
        if oldKey := c.GetOrDefault("OLD_KEY_NAME", ""); oldKey != "" {
            c.Set("NEW_KEY_NAME", oldKey)
            c.Delete("OLD_KEY_NAME")
        }
        c.Set("CONFIG_VERSION", "2")
        return c.Save()
    }
    return nil
}
```

### 20. **Default Values Scattered** 📋
**Location:** Throughout codebase
**Impact:** Hard to find what defaults are

**Recommendation:**
```go
// Create central defaults
package config

var Defaults = map[string]string{
    KeyHomelabBaseDir:  "/srv/containers",
    KeyContainerRuntime: "podman",
    KeyNFSMountPoint:   "/mnt/nas",
    KeyNetworkRetries:  "5",
    KeyNetworkTimeout:  "10",
}

func (c *Config) GetOrDefault(key, fallback string) string {
    if err := c.ensureLoaded(); err != nil {
        return fallback
    }
    if value, exists := c.data[key]; exists {
        return value
    }
    // Check defaults table
    if defaultValue, exists := Defaults[key]; exists {
        return defaultValue
    }
    return fallback
}
```

### 21. **Reset Doesn't Warn About Side Effects** 📋
**Location:** `internal/cli/menu.go` resetSetup
**Impact:** User loses work without full understanding

**Recommendation:**
```go
func (m *Menu) resetSetup() error {
    m.ctx.UI.Warning("This will clear all completion markers")
    m.ctx.UI.Warning("Configuration file will NOT be deleted")

    // NEW: Show what will happen
    m.ctx.UI.Warning("")
    m.ctx.UI.Warning("After reset, you will need to:")
    m.ctx.UI.Warning("  - Re-run all setup steps")
    m.ctx.UI.Warning("  - Services will NOT be stopped")
    m.ctx.UI.Warning("  - Created files will NOT be deleted")
    m.ctx.UI.Warning("  - System configuration (fstab, systemd) will remain")
    m.ctx.UI.Warning("")
    m.ctx.UI.Info("To completely remove the homelab setup, use: homelab-setup uninstall")
}
```

### 22. **No Uninstall Command** 📋
**Location:** Missing
**Impact:** Users can't cleanly remove setup

**Recommendation:**
```go
// Add cmd_uninstall.go
var uninstallCmd = &cobra.Command{
    Use:   "uninstall",
    Short: "Remove homelab setup and revert changes",
    Long:  `Removes services, unmounts NFS, and reverts system configuration.`,
    RunE:  runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
    ctx, _ := cli.NewSetupContext()

    ctx.UI.Warning("This will remove all homelab setup")
    confirm, _ := ctx.UI.PromptYesNo("Are you sure?", false)
    if !confirm {
        return nil
    }

    // 1. Stop and disable services
    // 2. Remove systemd units
    // 3. Unmount NFS
    // 4. Remove fstab entries
    // 5. Remove directories (ask first)
    // 6. Remove config and markers
}
```

---

## Low Priority Issues

### 23. **Menu Clears Screen Aggressively** 💡
**Location:** `internal/cli/menu.go`
**Impact:** Can't review previous output

**Recommendation:** Add `--no-clear` flag or keep last N lines visible

### 24. **No Shell Completion** 💡
**Impact:** Poor CLI UX

**Recommendation:**
```go
// Cobra has built-in support
var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate shell completion scripts",
    Long:  `Generate shell completion scripts for homelab-setup.`,
    // ... cobra.GenBashCompletion, etc.
}
```

### 25. **Version Command Missing Build Info** 💡
**Location:** `pkg/version/version.go`
**Current:** Shows version, commit, date
**Missing:** Go version, OS/arch, build tags

**Recommendation:**
```go
func Info() string {
    return fmt.Sprintf(`homelab-setup %s
Commit: %s
Built: %s
Go: %s
Platform: %s/%s`,
        Version, GitCommit, BuildDate,
        runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
```

### 26. **UI Methods Don't Return Errors** 💡
**Location:** `internal/ui/output.go`
**Impact:** Can't detect if output fails

Currently all UI methods are `void`. Should return errors:
```go
func (u *UI) Info(msg string) error {
    u.logger.Printf("[INFO] %s", msg)
    _, err := u.colorInfo.Fprintf(u.output, "[INFO] %s\n", msg)
    return err
}
```

### 27. **Container Deployment Doesn't Check Compose File Syntax** 💡
**Location:** `internal/steps/deployment.go`
**Impact:** Fails at runtime instead of validation

**Recommendation:**
```go
func (d *Deployment) ValidateComposeFile(path string) error {
    composeCmd, _ := d.containers.GetComposeCommand(runtime)
    output, err := exec.Command(composeCmd, "-f", path, "config", "--quiet").CombinedOutput()
    if err != nil {
        return fmt.Errorf("compose file validation failed: %w\nOutput: %s", err, output)
    }
    return nil
}
```

### 28. **Timezone Detection Could Be More Robust** 💡
**Location:** Likely in user setup
**Recommendation:** Check `/etc/timezone`, `timedatectl`, and `TZ` env var as fallbacks

### 29. **Service Names Hardcoded as podman-compose-*.service** 💡
**Location:** `internal/steps/deployment.go` line 77
**Impact:** Won't work if user renames services

**Recommendation:** Make service name pattern configurable

### 30. **No Metrics or Telemetry (Even Opt-In)** 💡
**Impact:** Can't see which features are used/failing

**Recommendation:** Add optional anonymous usage stats with clear opt-in

---

## Architecture & Design Observations

### Strengths ✅
1. **Clean separation of concerns** - Steps, System, UI, Config are well isolated
2. **Testable design** - Interfaces allow mocking (CommandRunner, FileSystem)
3. **Good error wrapping** - Uses `fmt.Errorf` with `%w` consistently
4. **Immutable OS awareness** - Understands rpm-ostree semantics
5. **Idempotency via markers** - Can re-run safely

### Weaknesses ⚠️
1. **No transaction/rollback model** - Partial failures leave system in bad state
2. **Config is flat key-value** - No nested structures (could use TOML/YAML)
3. **Heavy reliance on shell commands** - Parsing output is fragile
4. **No plugin architecture** - Can't add custom steps without code changes
5. **Tight coupling to systemd** - Won't work on non-systemd systems

---

## Testing Observations

**Missing Test Coverage:**
- Integration tests for full workflow
- Error path testing (what happens when commands fail)
- Concurrent access tests for Config
- Filesystem permission tests
- Network failure scenarios

**Recommendation:** Add test suite with:
```go
// tests/integration_test.go
func TestFullSetupWorkflow(t *testing.T) {
    // Use testcontainers or VM
    // Run full setup end-to-end
    // Verify all services start
}

func TestSetupRollbackOnFailure(t *testing.T) {
    // Inject failure at each step
    // Verify clean rollback
}
```

---

## Performance Considerations

1. **Sequential execution** - Could parallelize independent steps (e.g., user + directory)
2. **No caching** - Re-detects system state every run
3. **Spawns many processes** - Each `sudo` call is a fork+exec

**Not critical** for a setup tool that runs once, but could improve UX.

---

## Security Observations

### Potential Issues:
1. **Path injection** - Mitigated by `ValidateSafePath()` ✅
2. **Command injection** - Using exec.Command with args (safe) ✅
3. **Config file permissions** - Mode 0600 ✅
4. **WireGuard keys in config** - Mode 0600 but should only be in wg config ⚠️
5. **No input sanitization on domain names** - Uses validation functions ✅

### Missing:
- No audit logging of privileged operations
- No verification of downloaded files (if any)
- No signing/verification of config files

---

## Documentation Gaps

1. **No godoc for exported functions** - Add package comments
2. **No architecture diagram** - Hard to onboard new contributors
3. **No troubleshooting guide** - Users will struggle with failures
4. **No FAQ** - Common questions not answered

---

## Recommendations Priority

**Implement Immediately:**
1. Add rollback mechanism (#1)
2. Fix config race condition (#2)
3. Improve sudo error messages (#3)
4. Add config key constants (#4)

**Implement Soon:**
5. Add dry-run mode (#5)
6. Step failure tracking in menu (#9)
7. Progress indicators (#8)
8. Setup state validation (#7)

**Nice to Have:**
- All remaining items
- Plugin architecture
- Better test coverage

---

## Conclusion

The codebase is well-structured and functional, but lacks operational robustness. The main gaps are:
- **No recovery from failures** (critical)
- **Poor visibility into what's happening** (UX issue)
- **Configuration fragility** (maintenance burden)

Recommended action: Fix critical issues (#1-3), then improve UX (#4-10), then polish (#11-30).

**Overall Grade: B+** (Good foundation, needs production hardening)
