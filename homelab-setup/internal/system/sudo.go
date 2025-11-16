package system

import (
	"fmt"
	"os/exec"
)

// SudoChecker handles sudo access validation
type SudoChecker struct{}

// NewSudoChecker creates a new SudoChecker instance
func NewSudoChecker() *SudoChecker {
	return &SudoChecker{}
}

// RequiresPassword checks if sudo requires password authentication
func (s *SudoChecker) RequiresPassword() (bool, error) {
	// Use -n flag to prevent prompting, and -v to validate cached credentials
	cmd := exec.Command("sudo", "-n", "-v")
	err := cmd.Run()
	if err != nil {
		// If exit code is 1, sudo requires password
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode() == 1, nil
		}
		return false, fmt.Errorf("failed to check sudo status: %w", err)
	}
	// sudo -n succeeded, no password needed (or credentials cached)
	return false, nil
}

// ValidateAccess checks if sudo access is available
// Returns true if sudo works (with or without password)
func (s *SudoChecker) ValidateAccess() error {
	// Try passwordless first
	requiresPwd, err := s.RequiresPassword()
	if err != nil {
		return fmt.Errorf("failed to check sudo access: %w", err)
	}

	if !requiresPwd {
		// Passwordless sudo works
		return nil
	}

	// Try with password prompt
	cmd := exec.Command("sudo", "true")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sudo authentication failed: %w", err)
	}

	return nil
}

// GetSudoConfig returns information about sudo configuration
func (s *SudoChecker) GetSudoConfig() (map[string]string, error) {
	info := make(map[string]string)

	requiresPwd, err := s.RequiresPassword()
	if err != nil {
		return nil, err
	}

	if requiresPwd {
		info["requires_password"] = "yes"
		info["recommendation"] = "Configure passwordless sudo for automation"
	} else {
		info["requires_password"] = "no"
		info["recommendation"] = "Sudo already configured for passwordless access"
	}

	// Check if user is in sudo/wheel group
	cmd := exec.Command("groups")
	output, err := cmd.Output()
	if err == nil {
		info["groups"] = string(output)
	}

	return info, nil
}

// SetupPasswordlessSudo provides instructions for configuring passwordless sudo
func (s *SudoChecker) SetupPasswordlessSudo() string {
	return `To configure passwordless sudo for this user:

1. Create a sudoers file:
   sudo visudo -f /etc/sudoers.d/homelab-setup

2. Add this line (replace USERNAME with your username):
   USERNAME ALL=(ALL) NOPASSWD: ALL

3. Save and exit

For security, you can limit to specific commands:
   USERNAME ALL=(ALL) NOPASSWD: /usr/bin/systemctl, /usr/bin/mount, /usr/bin/mkdir

After configuration, test with:
   sudo -n true

This should succeed without prompting for a password.
`
}
