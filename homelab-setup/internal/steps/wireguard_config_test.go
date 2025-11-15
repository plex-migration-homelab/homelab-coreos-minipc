package steps

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestWireGuardWriteConfigCreatesSecureFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.conf")
	cfg := config.New(cfgPath)
	if err := cfg.Set("WIREGUARD_CONFIG_DIR", tmpDir); err != nil {
		t.Fatalf("failed to set config dir: %v", err)
	}

	packages := system.NewPackageManager()
	services := system.NewServiceManager()
	fs := system.NewFileSystem()
	network := system.NewNetwork()
	markers := config.NewMarkers(tmpDir)
	buf := &bytes.Buffer{}
	testUI := ui.NewWithWriter(buf)

	setup := NewWireGuardSetup(packages, services, fs, network, cfg, testUI, markers)
	wgCfg := &WireGuardConfig{
		InterfaceName: "wgtest",
		InterfaceIP:   "10.1.0.1/24",
		ListenPort:    "51820",
	}

	// WriteConfig creates the file with sudo, owned by root:root with 0600
	if err := setup.WriteConfig(wgCfg, "test-private-key"); err != nil {
		t.Fatalf("WriteConfig failed: %v", err)
	}

	configFile := filepath.Join(tmpDir, "wgtest.conf")

	// File is owned by root:root with 0600, so we need sudo to read it
	output, err := exec.Command("sudo", "cat", configFile).Output()
	if err != nil {
		t.Fatalf("failed to read config file with sudo: %v", err)
	}

	if !strings.Contains(string(output), "PrivateKey = test-private-key") {
		t.Fatalf("config file missing private key, content: %s", string(output))
	}

	// Check permissions using fs.GetPermissions which handles sudo
	info, err := fs.GetPermissions(configFile)
	if err != nil {
		t.Fatalf("failed to get permissions: %v", err)
	}

	if info.Perm() != 0600 {
		t.Fatalf("expected permissions 0600, got %v", info.Perm())
	}
}
