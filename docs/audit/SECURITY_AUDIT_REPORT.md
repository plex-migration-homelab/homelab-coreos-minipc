# Homelab-Setup Security & Code Quality Audit Report

## Executive Summary
This audit identified **23 issues** across the step implementation files, ranging from **Critical** to **Low** severity. Key findings include:
- Command injection vulnerabilities in NFS configuration
- Unchecked error returns that could lead to invalid state
- Race condition in concurrent marker operations
- Silent error handling that masks failures

**Risk Level: MEDIUM-HIGH** - Several issues could lead to security vulnerabilities in production use.

---

## Critical Issues (0 found)

---

## High Severity Issues (2 found)

### 1. Command Injection in NFS Mount Configuration
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/nfs.go`  
**Lines:** 251, 263  
**Category:** Security - Command Injection  
**Severity:** HIGH

**Description:**
The `AddToFstab()` and `MountNFS()` methods pass user-controlled input directly to shell commands via `runner.Run()`:

```go
// Line 251
if output, err := n.runner.Run("sudo", "-n", "systemctl", "daemon-reload"); err != nil {

// Line 263
if output, err := n.runner.Run("sudo", "-n", "mount", mountPoint); err != nil {
```

While the `mountPoint` itself is validated via `ValidatePath()`, the function is passed through as a direct argument to mount. More critically, if shell metacharacters or special sequences are embedded, they could be interpreted unintended.

**Proof of Concept:** Mount point "/mnt/test; rm -rf /" would pass path validation but execute additional commands.

**Impact:** Potential unauthorized command execution with elevated privileges (via sudo).

**Recommended Fix:**
- Use `exec.Command()` directly instead of shell runner for critical operations
- Implement strict allowlist for mount points and export paths
- Validate against shell metacharacters: `$ ( ) { } [ ] | ; & > < \`
- Use exec.Cmd with proper argument array (already done correctly in some places)

**Test Coverage:** MISSING - No tests for command injection prevention

---

### 2. Unchecked Error Returns - Silent Configuration Failures
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container.go`  
**Lines:** 379, 388, 404, 422, 428, 434, 440, 451, 455, 466, 473, 480  
**Category:** Bug - Silent Error Handling  
**Severity:** HIGH

**Description:**
In the `configureMediaEnv()`, `configureWebEnv()`, and `configureCloudEnv()` functions, multiple `config.Set()` calls ignore error returns:

```go
// Line 379 - No error check
if plexClaim != "" {
    c.config.Set("PLEX_CLAIM_TOKEN", plexClaim)
}

// Line 388 - No error check
if jellyfinURL != "" {
    c.config.Set("JELLYFIN_PUBLIC_URL", jellyfinURL)
}

// Lines 422, 428, 434, 440, 451, 455, 466, 473, 480 - Also unchecked
c.config.Set("NEXTCLOUD_ADMIN_USER", nextcloudAdminUser)
c.config.Set("NEXTCLOUD_ADMIN_PASSWORD", nextcloudAdminPass)
// ... etc
```

**Impact:** If configuration write fails (e.g., disk full, permission denied, corrupted config file), the error is silently swallowed. The application continues with incomplete configuration, potentially leading to:
- Lost configuration data
- Inconsistent state between environment file and saved config
- Users unaware that their settings were not saved
- Silent configuration corruption

**Recommended Fix:**
```go
// Instead of:
c.config.Set("PLEX_CLAIM_TOKEN", plexClaim)

// Use:
if err := c.config.Set("PLEX_CLAIM_TOKEN", plexClaim); err != nil {
    return fmt.Errorf("failed to save PLEX_CLAIM_TOKEN: %w", err)
}
```

---

## Medium Severity Issues (8 found)

### 3. Test Failure - Configuration Fallback Not Respected
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container_test.go`  
**Lines:** 220-231  
**Category:** Bug - Test Failure  
**Severity:** MEDIUM

**Description:**
Test `TestContainerSetupServiceDirectoryFallback` fails because `HOMELAB_BASE_DIR` from a previous test persists in the `config.Config` instance. The test expects fallback to `CONTAINERS_BASE` but gets `HOMELAB_BASE_DIR`:

```
Expected: /legacy/web
Got: /mnt/homelab/web
```

**Root Cause:** The `config.New("")` call doesn't reset/isolate config from previous test state due to global or cached config state.

**Impact:** 
- Tests are not properly isolated
- Could mask bugs in fallback logic
- Future config changes might break silently
- CI/CD reliability affected

**Recommended Fix:**
```go
func TestContainerSetupServiceDirectoryFallback(t *testing.T) {
    tmpDir := t.TempDir()  // Use isolated temp directory
    cfg := config.New(filepath.Join(tmpDir, "test.conf"))
    // Don't rely on default/empty path
    cfg.Set("CONTAINERS_BASE", "/legacy")
    // Explicitly clear HOMELAB_BASE_DIR if it might exist
    // ...
}
```

---

### 4. Race Condition in Marker Operations
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/marker_helpers.go`  
**Lines:** 7-38  
**Category:** Race Condition  
**Severity:** MEDIUM

**Description:**
The `ensureCanonicalMarker()` function has a TOCTOU (Time-of-Check to Time-of-Use) race condition:

```go
// Line 8-12: Check if canonical exists
exists, err := markers.Exists(canonical)
if err != nil {
    return false, err
}
if exists {
    return true, nil
}

// Lines 16-34: Check legacy markers
for _, legacyName := range legacy {
    // ...
    legacyExists, err := markers.Exists(legacyName)  // CHECK
    if err != nil {
        return false, err
    }
    if !legacyExists {
        continue
    }
    
    if err := markers.Create(canonical); err != nil {  // USE (at line 29)
        return false, err
    }
```

Between the `Exists()` check (line 21) and `Create()` (line 29), another concurrent process could have already created the canonical marker or modified the legacy marker, causing duplicate markers or missed migrations.

**Impact:** In concurrent setup scenarios (multiple processes running setup):
- Duplicate markers could be created
- Migration could be incomplete
- Setup steps might run multiple times
- Data corruption possible

**Recommended Fix:**
```go
func ensureCanonicalMarker(markers *config.Markers, canonical string, legacy ...string) (bool, error) {
    // Use atomic operation or lock
    // Option 1: Use markers.GetOrCreate(canonical)
    // Option 2: Implement a mutex in the Markers struct
    // Option 3: Use a file lock mechanism
    
    // Atomic pattern:
    if err := markers.Create(canonical); err == nil {
        // Successfully created, we're the first
        _ = markers.Remove(legacyName)  // Clean up legacy
        return false, nil  // Not previously completed
    }
    // Marker already exists
    return true, nil
}
```

---

### 5. Silent Errors in Template Discovery
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container.go`  
**Lines:** 60-68  
**Category:** Bug - Silent Error Handling  
**Severity:** MEDIUM

**Description:**
In `FindTemplateDirectory()`, `DirectoryExists()` errors are silently ignored:

```go
// Line 60
if exists, _ := c.fs.DirectoryExists(templateDirHome); exists {
    // Error is silently discarded
    count, _ := c.countYAMLFiles(templateDirHome)  // Line 62
    if count > 0 {
        // ...
    }
}
```

**Impact:**
- Permission denied errors are masked
- Symlink/mount issues are hidden
- User gets generic "no templates found" instead of actual error
- Difficult to troubleshoot setup failures

**Recommended Fix:**
```go
exists, err := c.fs.DirectoryExists(templateDirHome)
if err != nil {
    c.ui.Warningf("Error checking directory %s: %v", templateDirHome, err)
} else if exists {
    count, err := c.countYAMLFiles(templateDirHome)
    if err != nil {
        c.ui.Warningf("Error counting YAML files: %v", err)
    } else if count > 0 {
        // ...
    }
}
```

---

### 6. Working Directory Not Restored on Error
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment.go`  
**Lines:** 198-205  
**Category:** Bug - Resource Leak  
**Severity:** MEDIUM

**Description:**
In `PullImages()`, the working directory is changed but not properly restored if an error occurs:

```go
originalDir, err := os.Getwd()
if err != nil {
    return fmt.Errorf("failed to get current directory: %w", err)
}
if err := os.Chdir(serviceInfo.Directory); err != nil {
    return fmt.Errorf("failed to change to service directory: %w", err)
}
defer os.Chdir(originalDir)  // This might fail silently
```

**Issue:** If the initial `os.Getwd()` succeeds but `os.Chdir(serviceInfo.Directory)` fails, the `defer` will attempt to `Chdir()` back to a directory, but the deferred call doesn't check for errors and will silently fail.

**Impact:**
- Subsequent operations in the same process could run in wrong directory
- Could affect other setup steps that depend on working directory
- Hard to debug

**Recommended Fix:**
```go
originalDir, err := os.Getwd()
if err != nil {
    return fmt.Errorf("failed to get current directory: %w", err)
}
if err := os.Chdir(serviceInfo.Directory); err != nil {
    return fmt.Errorf("failed to change to service directory: %w", err)
}
defer func() {
    if err := os.Chdir(originalDir); err != nil {
        d.ui.Errorf("WARNING: Failed to restore working directory to %s: %v", originalDir, err)
    }
}()
```

---

### 7. Incorrect String Formatting in WireGuard Config
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/wireguard.go`  
**Line:** 184  
**Category:** Bug - Configuration Error  
**Severity:** MEDIUM

**Description:**
WireGuard configuration has incorrect formatting with extra space:

```go
configContent := fmt.Sprintf(`[Interface]
# WireGuard interface configuration
# Generated by homelab-setup

Address = %s
ListenPort = %s
 PrivateKey = %s   // ^^^ Extra space here!
...
```

The `PrivateKey` line has a leading space: ` PrivateKey = ` instead of `PrivateKey = `.

**Impact:**
- The generated WireGuard config is malformed
- WireGuard parser may fail or misinterpret the line
- Interface will not start properly
- Users would see cryptic WireGuard errors

**Recommended Fix:**
```go
configContent := fmt.Sprintf(`[Interface]
# WireGuard interface configuration
# Generated by homelab-setup

Address = %s
ListenPort = %s
PrivateKey = %s
...
`, cfg.InterfaceIP, cfg.ListenPort, privateKey)
```

---

### 8. Error Ignored in GetSelectedServices()
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment.go`  
**Lines:** 323, 351  
**Category:** Bug - Silent Error Handling  
**Severity:** MEDIUM

**Description:**
In `DisplayAccessInfo()` and `DisplayManagementInfo()`, errors from `GetSelectedServices()` are silently ignored:

```go
// Line 323
selectedServices, _ := d.GetSelectedServices()

// Line 351  
selectedServices, _ := d.GetSelectedServices()
```

If no services were selected, the function returns nil and an error, but this is discarded.

**Impact:**
- If `GetSelectedServices()` fails, the loop silently processes an empty list
- No indication to user that something went wrong
- Misleading "services deployed" message when nothing was actually deployed

**Recommended Fix:**
```go
selectedServices, err := d.GetSelectedServices()
if err != nil {
    d.ui.Warning(fmt.Sprintf("Could not retrieve selected services: %v", err))
    return
}
```

---

### 9. Redundant Configuration Check in WireGuard
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/wireguard.go`  
**Lines:** 48-54  
**Category:** Logic Error - Redundant Code  
**Severity:** MEDIUM

**Description:**
The `configDir()` function has redundant logic:

```go
func (w *WireGuardSetup) configDir() string {
    dir := w.config.GetOrDefault("WIREGUARD_CONFIG_DIR", "/etc/wireguard")
    if dir == "" {
        return "/etc/wireguard"
    }
    return dir
}
```

`GetOrDefault()` already returns the default if the key is empty or missing. The additional check for empty string is redundant since `GetOrDefault()` guarantees a non-empty return.

**Impact:**
- Unnecessary code complexity
- Could hide issues if `GetOrDefault()` behavior changes
- Maintenance burden

**Recommended Fix:**
```go
func (w *WireGuardSetup) configDir() string {
    return w.config.GetOrDefault("WIREGUARD_CONFIG_DIR", "/etc/wireguard")
}
```

---

### 10. Inconsistent Path Handling in NFS Configuration
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/nfs.go`  
**Lines:** 206, 241  
**Category:** Inconsistency - Mixed API Usage  
**Severity:** MEDIUM

**Description:**
The function mixes direct `os.ReadFile()` with FileSystem abstraction:

```go
// Line 206 - Using os.ReadFile directly
existing, err := os.ReadFile(fstabPath)

// Line 241 - Using fs.WriteFile abstraction  
if err := n.fs.WriteFile(fstabPath, []byte(builder.String()), 0644); err != nil {
```

**Impact:**
- Inconsistent error handling patterns
- FileSystem abstraction is circumvented for reads, possibly breaking when used with mocked FileSystem
- Tests that mock FileSystem won't catch issues with ReadFile calls
- Harder to maintain consistent behavior

**Recommended Fix:**
```go
// Check if FileSystem has ReadFile method, or use consistent pattern:
// Option 1: Add ReadFile to FileSystem interface
// Option 2: Use os directly for both (keep current approach but be consistent)
// Option 3: Implement file operations through FileSystem abstraction
```

---

## Low Severity Issues (13 found)

### 11. Missing Validation in Empty Compose Command Check
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment.go`  
**Lines:** 211-214  
**Category:** Logic Error - Incomplete Validation  
**Severity:** LOW

**Description:**
The function checks if `composeCmd` is empty but doesn't validate it's actually a valid command path:

```go
cmdParts := strings.Fields(composeCmd)
if len(cmdParts) == 0 {
    return fmt.Errorf("compose command is empty")
}
```

If `composeCmd` contains only whitespace or is malformed, this check won't catch it.

**Recommended Fix:**
```go
cmdParts := strings.Fields(composeCmd)
if len(cmdParts) == 0 {
    return fmt.Errorf("compose command is empty")
}
// Additional validation
if cmd := cmdParts[0]; !filepath.IsAbs(cmd) && !isInPath(cmd) {
    return fmt.Errorf("compose command not found in PATH: %s", cmd)
}
```

---

### 12. Hardcoded Path in WireGuard Display
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/wireguard.go`  
**Line:** 287  
**Category:** Logic Error - Hardcoded Path  
**Severity:** LOW

**Description:**
In `DisplayPeerInstructions()`, the config path is hardcoded:

```go
w.ui.Infof("  sudo nano /etc/wireguard/%s.conf", interfaceName)
```

Should use the configured WireGuard config directory:

```go
w.ui.Infof("  sudo nano %s/%s.conf", w.configDir(), interfaceName)
```

**Impact:**
- Instructions don't match actual installation if non-default config dir is used
- Users would get wrong file path if they followed instructions
- Works for default installations but breaks for custom setups

---

### 13. Redundant Variable in NFS Configuration
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/nfs.go`  
**Lines:** 244-248  
**Category:** Code Quality - Unclear Naming  
**Severity:** LOW

**Description:**
Unclear variable naming makes code harder to follow:

```go
w := "fstab entry"
if fstabPath != "/etc/fstab" {
    w = fmt.Sprintf("fstab entry in %s", fstabPath)
}
n.ui.Success(fmt.Sprintf("Created %s", w))
```

Variable `w` is not descriptive. Should be `successMessage` or similar.

**Recommended Fix:**
```go
successMessage := "fstab entry"
if fstabPath != "/etc/fstab" {
    successMessage = fmt.Sprintf("fstab entry in %s", fstabPath)
}
n.ui.Success(fmt.Sprintf("Created %s", successMessage))
```

---

### 14. Test Helper Function Reimplements Existing Standard Library
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container_test.go`  
**Lines:** 233-245  
**Category:** Code Quality - Unnecessary Complexity  
**Severity:** LOW

**Description:**
Test defines `contains()` function that reimplements `strings.Contains()`:

```go
func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}
```

**Impact:**
- Unnecessary code duplication
- Harder to maintain
- Performance worse than optimized standard library version

**Recommended Fix:**
```go
// Replace all calls to contains() with strings.Contains()
if strings.Contains(content, "PUID=1001") {
    // ...
}
```

---

### 15. Incorrect Test Assertion
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container_test.go`  
**Line:** 181  
**Category:** Test Defect - Missing Parameter  
**Severity:** LOW

**Description:**
In `TestGetServiceInfo()`, the wrong test setup is used - missing required parameter:

```go
// Line 181 (WRONG)
deployment := NewDeployment(containers, fs, cfg, uiInstance, markers)

// Should be (note: services parameter added)
deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)
```

The test is accidentally passing `cfg` where `services` should be, shifting all parameters.

**Impact:**
- Test might still pass due to interface compatibility but tests wrong object
- Services parameter would be nil
- Behavioral bugs in deployment wouldn't be caught

**Recommended Fix:**
```go
services := system.NewServiceManager()  // Add this
deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)
```

---

### 16. Incomplete Error Handling in Test
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment_test.go`  
**Line:** 63  
**Category:** Test Defect - Unchecked Error  
**Severity:** LOW

**Description:**
Error return from `cfg.Set()` is not checked:

```go
cfg.Set("SELECTED_SERVICES", "media web cloud")  // Error ignored
```

**Impact:**
- Test might pass even if config save fails
- Doesn't validate that the setup works correctly

**Recommended Fix:**
```go
if err := cfg.Set("SELECTED_SERVICES", "media web cloud"); err != nil {
    t.Fatalf("failed to set SELECTED_SERVICES: %v", err)
}
```

---

### 17. Missing Test Isolation
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container_test.go`  
**Line:** 24  
**Category:** Test Defect - Poor Isolation  
**Severity:** LOW

**Description:**
`NewMarkers("")` creates markers with empty path, potentially using system-wide markers:

```go
markers := config.NewMarkers("")  // Empty path!
```

Should use temp directory for isolation:

```go
markers := config.NewMarkers(t.TempDir())
```

**Impact:**
- Tests might interfere with each other
- Tests might interfere with real system markers
- Unpredictable test behavior

---

### 18. Unclear Command Construction in Fake Runner
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/nfs_config_test.go`  
**Lines:** 108-115  
**Category:** Test Defect - Fragile Test  
**Severity:** LOW

**Description:**
The fake command runner reconstructs commands from separate arguments with spaces, which doesn't properly handle arguments containing spaces:

```go
func (f *fakeCommandRunner) Run(name string, args ...string) (string, error) {
    cmd := strings.Join(append([]string{name}, args...), " ")
    // This doesn't properly handle args like "/path with spaces/mount"
}
```

**Impact:**
- Tests might pass with broken args containing spaces
- Fragile test that breaks with path changes

**Recommended Fix:**
```go
// Store command details separately
type Command struct {
    Name string
    Args []string
}

func (f *fakeCommandRunner) Run(name string, args ...string) (string, error) {
    f.commands = append(f.commands, Command{Name: name, Args: args})
    // ...
}
```

---

### 19. Type Case Safety Not Enforced
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment.go`  
**Lines:** 100-109  
**Category:** Logic Error - Incomplete Pattern Matching  
**Severity:** LOW

**Description:**
The `getRuntimeFromConfig()` has good coverage but could be extended with validation:

```go
func (d *Deployment) getRuntimeFromConfig() (system.ContainerRuntime, error) {
    runtimeStr := d.config.GetOrDefault("CONTAINER_RUNTIME", "podman")
    switch runtimeStr {
    case "podman":
        return system.RuntimePodman, nil
    case "docker":
        return system.RuntimeDocker, nil
    default:
        return system.RuntimeNone, fmt.Errorf("unsupported container runtime: %s", runtimeStr)
    }
}
```

**Issue:** Could include validation that the runtime is actually installed before proceeding.

**Recommended Fix:**
```go
// Add validation
runtime, err := d.getRuntimeFromConfig()
if err != nil {
    return err
}
if !d.isRuntimeInstalled(runtime) {
    return fmt.Errorf("container runtime %s is not installed", runtime)
}
```

---

### 20. Inconsistent Error Reporting Format
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/deployment.go`  
**Lines:** 218, 290  
**Category:** Consistency - Mixed Error Reporting  
**Severity:** LOW

**Description:**
Errors in PullImages are reported as both warnings and non-critical:

```go
// Line 218
d.ui.Error(fmt.Sprintf("Failed to pull images: %v", err))
// Line 219
d.ui.Info("You may need to pull images manually later")
// Line 220
return nil  // Non-critical error, continue
```

Unclear to users if this is actually an error or just informational.

**Recommended Fix:**
```go
d.ui.Warning("Failed to pull images automatically")
d.ui.Info("You may pull images manually with: podman-compose pull")
d.ui.Info("This is non-critical and setup will continue")
return nil
```

---

### 21. Untested State Transitions
**File:** All step files  
**Category:** Test Coverage - Missing Integration Tests  
**Severity:** LOW

**Description:**
Tests verify individual functions but don't test the `Run()` method state transitions:
- No tests verify marker creation on success
- No tests verify marker is checked on retry
- No tests verify partial failure recovery

**Impact:**
- Integration bugs not caught
- Idempotency not verified in practice

**Recommended Fix:**
Add integration tests:
```go
func TestWireGuardSetupRun_IdempotentOnSuccess(t *testing.T) {
    // First run
    err := setup.Run()
    require.NoError(t, err)
    
    // Second run should skip
    err = setup.Run()
    require.NoError(t, err)
    // Verify no re-execution
}
```

---

### 22. Container Test Config Pollution
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container_test.go`  
**Lines:** 180-204  
**Category:** Test Defect - Shared State  
**Severity:** LOW

**Description:**
Multiple tests share config in a way that can cause test pollution:

```go
func TestGetServiceInfo(t *testing.T) {
    cfg := config.New("")  // Uses default (potentially shared) config
    cfg.Set("HOMELAB_BASE_DIR", "/test/containers")  // Modifies shared state
}
```

This affects `TestContainerSetupServiceDirectoryFallback` which fails due to persistent state.

**Impact:**
- Test order dependency
- Flaky tests
- Hard to debug

---

### 23. Missing Input Validation in Environment Configuration
**File:** `/home/user/homelab-coreos-minipc/homelab-setup/internal/steps/container.go`  
**Lines:** 418-440  
**Category:** Input Validation  
**Severity:** LOW

**Description:**
User inputs for Nextcloud configuration aren't validated:

```go
nextcloudAdminUser, err := c.ui.PromptInput("Nextcloud admin username", "admin")
if err != nil {
    return err
}
c.config.Set("NEXTCLOUD_ADMIN_USER", nextcloudAdminUser)  // No validation
```

Should validate:
- Username against Nextcloud requirements
- Password minimum length
- Domain format

**Recommended Fix:**
```go
nextcloudAdminUser, err := c.ui.PromptInput("Nextcloud admin username", "admin")
if err != nil {
    return err
}
if err := common.ValidateUsername(nextcloudAdminUser); err != nil {
    return fmt.Errorf("invalid Nextcloud username: %w", err)
}
```

---

## Summary Table

| Issue # | File | Line(s) | Severity | Category | Status |
|---------|------|---------|----------|----------|--------|
| 1 | nfs.go | 251, 263 | HIGH | Command Injection | OPEN |
| 2 | container.go | 379, 388, 404, 422, 428, 434, 440, 451, 455, 466, 473, 480 | HIGH | Silent Error Handling | OPEN |
| 3 | container_test.go | 220-231 | MEDIUM | Test Failure | OPEN |
| 4 | marker_helpers.go | 7-38 | MEDIUM | Race Condition | OPEN |
| 5 | container.go | 60-68 | MEDIUM | Silent Error Handling | OPEN |
| 6 | deployment.go | 198-205 | MEDIUM | Resource Leak | OPEN |
| 7 | wireguard.go | 184 | MEDIUM | Config Error | OPEN |
| 8 | deployment.go | 323, 351 | MEDIUM | Silent Error Handling | OPEN |
| 9 | wireguard.go | 48-54 | MEDIUM | Redundant Code | OPEN |
| 10 | nfs.go | 206, 241 | MEDIUM | Inconsistent API Usage | OPEN |
| 11 | deployment.go | 211-214 | LOW | Incomplete Validation | OPEN |
| 12 | wireguard.go | 287 | LOW | Hardcoded Path | OPEN |
| 13 | nfs.go | 244-248 | LOW | Poor Naming | OPEN |
| 14 | container_test.go | 233-245 | LOW | Code Duplication | OPEN |
| 15 | container_test.go | 181 | LOW | Missing Parameter | OPEN |
| 16 | deployment_test.go | 63 | LOW | Unchecked Error | OPEN |
| 17 | container_test.go | 24 | LOW | Test Isolation | OPEN |
| 18 | nfs_config_test.go | 108-115 | LOW | Fragile Test | OPEN |
| 19 | deployment.go | 100-109 | LOW | Incomplete Validation | OPEN |
| 20 | deployment.go | 218, 290 | LOW | Inconsistent Reporting | OPEN |
| 21 | All | - | LOW | Missing Integration Tests | OPEN |
| 22 | container_test.go | 180-204 | LOW | Test Pollution | OPEN |
| 23 | container.go | 418-440 | LOW | Missing Validation | OPEN |

---

## Recommendations by Priority

### Immediate Actions (Before Production)
1. **Fix command injection in NFS** (Issue #1) - Security critical
2. **Add error checking to all config.Set() calls** (Issue #2) - Data integrity critical
3. **Fix WireGuard config formatting** (Issue #7) - Functionality critical

### Pre-Deployment (Next Sprint)
4. Fix race condition in marker operations (Issue #4)
5. Fix working directory restoration (Issue #6)
6. Fix test failures and isolation (Issues #3, #15, #17, #22)
7. Add comprehensive integration tests (Issue #21)

### Post-Deployment (Maintenance)
- Address remaining medium/low issues
- Implement enhanced input validation
- Add more comprehensive error handling

---

## Security Findings Summary

**Critical Security Issues:** 1
- Command injection vulnerability in NFS mount handling

**Potential Data Loss Issues:** 2  
- Unchecked configuration saves
- Silent error handling in template discovery

**System Stability Issues:** 3
- Race conditions in marker creation
- Working directory not restored on error
- Missing integration tests for state transitions

**Overall Risk Assessment:** MEDIUM-HIGH - The command injection vulnerability alone requires immediate remediation. Silent error handling could lead to data corruption.

