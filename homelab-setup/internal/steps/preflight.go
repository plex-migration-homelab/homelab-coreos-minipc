package steps

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

const preflightCompletionMarker = "preflight-complete"

// checkRpmOstree verifies the system is running rpm-ostree
func checkRpmOstree(ui *ui.UI) error {
	ui.Info("Checking for rpm-ostree system...")

	if !system.IsRpmOstreeSystem() {
		ui.Error("This system does not appear to be running rpm-ostree")
		ui.Info("These setup scripts are designed for UBlue uCore (rpm-ostree based)")
		ui.Info("Please use the appropriate setup scripts for your system")
		return fmt.Errorf("not an rpm-ostree system")
	}

	ui.Success("Confirmed: Running on rpm-ostree system")

	// Get and display rpm-ostree status
	status, err := system.GetRpmOstreeStatus()
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not get detailed rpm-ostree status: %v", err))
		return nil
	}

	// Just log that we got the status (parsing JSON would require encoding/json)
	if len(status) > 0 {
		ui.Info("Successfully retrieved rpm-ostree deployment information")
	}

	return nil
}

// checkRequiredPackages verifies all required packages are installed
func checkRequiredPackages(ui *ui.UI) error {
	ui.Info("Checking packages...")

	// Core packages that are always needed
	corePackages := []string{}

	// Optional packages (for optional setup steps)
	optionalPackages := []string{
		"nfs-utils",       // Optional: for NFS setup
		"wireguard-tools", // Optional: for WireGuard VPN setup
	}

	// Check core packages (none currently required)
	if len(corePackages) > 0 {
		results, err := system.CheckMultiplePackages(corePackages)
		if err != nil {
			return fmt.Errorf("failed to check packages: %w", err)
		}

		missingPackages := []string{}
		for _, pkg := range corePackages {
			if results[pkg] {
				ui.Successf("  ✓ %s is installed", pkg)
			} else {
				ui.Errorf("  ✗ %s is NOT installed", pkg)
				missingPackages = append(missingPackages, pkg)
			}
		}

		if len(missingPackages) > 0 {
			ui.Error("Missing required packages")
			ui.Info("To install them, run:")
			for _, pkg := range missingPackages {
				ui.Infof("  sudo rpm-ostree install %s", pkg)
			}
			ui.Info("Then reboot the system:")
			ui.Info("  sudo systemctl reboot")
			return fmt.Errorf("missing required packages: %v", missingPackages)
		}
	}

	// Check optional packages (warnings only)
	if len(optionalPackages) > 0 {
		ui.Info("Checking optional packages...")
		results, err := system.CheckMultiplePackages(optionalPackages)
		if err != nil {
			ui.Warning(fmt.Sprintf("Failed to check optional packages: %v", err))
		} else {
			missingOptional := []string{}
			for _, pkg := range optionalPackages {
				if results[pkg] {
					ui.Successf("  ✓ %s is installed", pkg)
				} else {
					ui.Infof("  - %s is not installed (optional)", pkg)
					missingOptional = append(missingOptional, pkg)
				}
			}

			if len(missingOptional) > 0 {
				ui.Info("Optional packages can be installed later if needed:")
				for _, pkg := range missingOptional {
					ui.Infof("  sudo rpm-ostree install %s", pkg)
				}
			}
		}
	}

	ui.Success("Package check completed")
	return nil
}

// checkContainerRuntime verifies Docker is available and configured
func checkContainerRuntime(cfg *config.Config, ui *ui.UI) error {
	ui.Info("Checking container runtime...")

	// Check if Docker service is active
	if err := system.CheckDockerService(); err != nil {
		ui.Error("  ✗ docker.service is not active")
		ui.Info("Docker must be running. Start it with:")
		ui.Info("  sudo systemctl start docker.service")
		ui.Info("  sudo systemctl enable docker.service")
		return fmt.Errorf("docker.service is not active")
	}
	ui.Success("  ✓ Docker service is available")

	// Check for Docker Compose (prefer V2 plugin, fallback to V1)
	if err := system.CheckDockerComposeV2(); err == nil {
		ui.Success("  ✓ Docker Compose V2 (docker compose) is available")
		if err := cfg.Set(config.KeyComposeCommand, "docker compose"); err != nil {
			ui.Warning("Failed to save compose command to config")
		}
	} else if err := system.CheckDockerComposeV1(); err == nil {
		ui.Success("  ✓ Docker Compose V1 (docker-compose) is available")
		if err := cfg.Set(config.KeyComposeCommand, "docker-compose"); err != nil {
			ui.Warning("Failed to save compose command to config")
		}
	} else {
		ui.Error("  ✗ Docker Compose is not available")
		ui.Info("Install Docker Compose V2 (preferred):")
		ui.Info("  Follow: https://docs.docker.com/compose/install/")
		ui.Info("Or install V1 standalone:")
		ui.Info("  sudo rpm-ostree install docker-compose")
		ui.Info("  sudo systemctl reboot")
		return fmt.Errorf("docker compose not available")
	}

	// Set runtime in config
	if err := cfg.Set(config.KeyContainerRuntime, "docker"); err != nil {
		ui.Warning("Failed to save container runtime to config")
	}

	return nil
}

// checkSudoAccess validates sudo is available and configured
func checkSudoAccess(ui *ui.UI) error {
	ui.Info("Checking sudo access...")

	sudoChecker := system.NewSudoChecker()

	requiresPwd, err := sudoChecker.RequiresPassword()
	if err != nil {
		return fmt.Errorf("failed to check sudo: %w", err)
	}

	if requiresPwd {
		ui.Warning("Sudo requires password authentication")
		ui.Info("For unattended operation, configure passwordless sudo")
		ui.Print("")
		ui.Info("Quick setup:")
		ui.Info("  echo '$USER ALL=(ALL) NOPASSWD: ALL' | sudo tee /etc/sudoers.d/$USER")
		ui.Info("  sudo chmod 440 /etc/sudoers.d/$USER")
		ui.Print("")

		// Try to authenticate once
		ui.Info("Validating sudo access (you may be prompted for password)...")
		if err := sudoChecker.ValidateAccess(); err != nil {
			ui.Error("Failed to authenticate with sudo")
			return fmt.Errorf("sudo authentication failed: %w", err)
		}
		ui.Success("Sudo access validated (credentials cached)")
	} else {
		ui.Success("Passwordless sudo is configured")
	}

	return nil
}

// checkNetworkConnectivity tests basic network connectivity
func checkNetworkConnectivity(ui *ui.UI) error {
	ui.Info("Checking network connectivity...")

	// Test connectivity to a reliable host
	reachable, err := system.TestConnectivity("8.8.8.8", 3)
	if err != nil {
		return fmt.Errorf("failed to test connectivity: %w", err)
	}

	if !reachable {
		ui.Error("No internet connectivity detected")
		ui.Info("Please check:")
		ui.Info("  1. Network cable is connected")
		ui.Info("  2. Network configuration is correct")
		ui.Info("  3. Default gateway is reachable")
		return fmt.Errorf("no internet connectivity")
	}

	ui.Success("Internet connectivity confirmed")

	// Get and display default gateway
	gateway, err := system.GetDefaultGateway()
	if err != nil {
		ui.Warning(fmt.Sprintf("Could not determine default gateway: %v", err))
	} else {
		ui.Infof("Default gateway: %s", gateway)

		// Test gateway connectivity
		gwReachable, _ := system.TestConnectivity(gateway, 2)
		if gwReachable {
			ui.Success("Default gateway is reachable")
		} else {
			ui.Warning("Default gateway is not responding to ping")
		}
	}

	return nil
}

// checkNFSServer validates NFS server is accessible if configured
func checkNFSServer(host string, ui *ui.UI) error {
	if host == "" {
		ui.Info("NFS server not configured yet, skipping NFS check")
		return nil
	}

	ui.Infof("Checking NFS server: %s", host)

	// First check basic connectivity
	reachable, err := system.TestConnectivity(host, 5)
	if err != nil {
		return fmt.Errorf("failed to test NFS server connectivity: %w", err)
	}

	if !reachable {
		ui.Error(fmt.Sprintf("NFS server %s is not reachable", host))
		ui.Info("Please check:")
		ui.Info("  1. NFS server is powered on")
		ui.Info("  2. Network connectivity to the server")
		ui.Info("  3. Firewall rules allow NFS traffic")
		return fmt.Errorf("NFS server %s is unreachable", host)
	}

	ui.Success(fmt.Sprintf("NFS server %s is reachable", host))

	// Check if NFS exports are available
	hasExports, err := system.CheckNFSServer(host)
	if err != nil {
		return fmt.Errorf("failed to check NFS exports: %w", err)
	}

	if !hasExports {
		ui.Warning("NFS server is reachable but showmount failed")
		ui.Info("This might indicate:")
		ui.Info("  1. NFS service is not running on the server")
		ui.Info("  2. No exports are configured")
		ui.Info("  3. Firewall is blocking NFS RPC calls")
		return fmt.Errorf("NFS server has no accessible exports")
	}

	ui.Success("NFS server has accessible exports")

	// Try to get and display exports
	exports, err := system.GetNFSExports(host)
	if err == nil && exports != "" {
		ui.Info("Available NFS exports:")
		ui.Print(exports)
	}

	return nil
}

// RunPreflightChecks executes all preflight checks
func RunPreflightChecks(cfg *config.Config, ui *ui.UI) error {
	// Check if already completed
	if cfg.IsComplete(preflightCompletionMarker) {
		ui.Info("Preflight checks already completed (marker found)")
		ui.Info("To re-run, remove marker: ~/.local/homelab-setup/" + preflightCompletionMarker)
		return nil
	}

	ui.Header("Pre-flight System Validation")
	ui.Info("Verifying system requirements before setup...")
	ui.Print("")

	hasErrors := false
	errorMessages := []string{}

	// Run rpm-ostree check
	ui.Step("Checking Operating System")
	if err := checkRpmOstree(ui); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run package checks
	ui.Step("Checking Required Packages")
	if err := checkRequiredPackages(ui); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run container runtime check
	ui.Step("Checking Container Runtime")
	if err := checkContainerRuntime(cfg, ui); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run sudo access check
	ui.Step("Checking Sudo Access")
	if err := checkSudoAccess(ui); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Run network connectivity check
	ui.Step("Checking Network Connectivity")
	if err := checkNetworkConnectivity(ui); err != nil {
		hasErrors = true
		errorMessages = append(errorMessages, err.Error())
	}

	// Check NFS server if configured
	nfsServer := cfg.GetOrDefault("NFS_SERVER", "")
	if nfsServer != "" {
		ui.Step("Checking NFS Server")
		if err := checkNFSServer(nfsServer, ui); err != nil {
			// NFS errors are warnings, not critical errors
			ui.Warning(err.Error())
		}
	}

	ui.Print("")
	ui.Separator()

	if hasErrors {
		ui.Error("Pre-flight checks FAILED")
		ui.Info("Please resolve the issues above before continuing")
		ui.Print("")
		for i, msg := range errorMessages {
			ui.Errorf("%d. %s", i+1, msg)
		}
		return fmt.Errorf("preflight checks failed with %d error(s)", len(errorMessages))
	}

	ui.Success("✓ All pre-flight checks PASSED")
	ui.Info("System is ready for homelab setup")

	// Create completion marker
	if err := cfg.MarkComplete(preflightCompletionMarker); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
