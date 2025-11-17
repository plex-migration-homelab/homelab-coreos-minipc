package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.conf")
	return config.New(configPath)
}

func TestNewContainerSetup(t *testing.T) {
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	cfg := newTestConfig(t)
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	if setup == nil {
		t.Fatal("NewContainerSetup returned nil")
	}

	if setup.containers == nil {
		t.Error("ContainerSetup.containers is nil")
	}
	if setup.fs == nil {
		t.Error("ContainerSetup.fs is nil")
	}
	if setup.config == nil {
		t.Error("ContainerSetup.config is nil")
	}
	if setup.ui == nil {
		t.Error("ContainerSetup.ui is nil")
	}
	if setup.markers == nil {
		t.Error("ContainerSetup.markers is nil")
	}
}

func TestCountYAMLFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "container-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	files := []string{
		"test1.yml",
		"test2.yaml",
		"test3.txt",
		".hidden.yml",
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test counting
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	cfg := newTestConfig(t)
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	count, err := setup.countYAMLFiles(tmpDir)
	if err != nil {
		t.Fatalf("countYAMLFiles failed: %v", err)
	}

	// Should count 3 YAML files (test1.yml, test2.yaml, .hidden.yml)
	expected := 3
	if count != expected {
		t.Errorf("Expected %d YAML files, got %d", expected, count)
	}
}

func TestGenerateEnvContent(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Set("ENV_PUID", "1001"); err != nil {
		t.Fatalf("failed to set ENV_PUID: %v", err)
	}
	if err := cfg.Set("ENV_PGID", "1002"); err != nil {
		t.Fatalf("failed to set ENV_PGID: %v", err)
	}
	if err := cfg.Set("ENV_TZ", "America/New_York"); err != nil {
		t.Fatalf("failed to set ENV_TZ: %v", err)
	}
	if err := cfg.Set("ENV_APPDATA_PATH", "/custom/path"); err != nil {
		t.Fatalf("failed to set ENV_APPDATA_PATH: %v", err)
	}

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	// Test generic service
	content := setup.generateEnvContent("generic")

	if content == "" {
		t.Error("generateEnvContent returned empty string")
	}

	// Check that base variables are present
	if !strings.Contains(content, "PUID=1001") {
		t.Error("Content missing PUID=1001")
	}
	if !strings.Contains(content, "PGID=1002") {
		t.Error("Content missing PGID=1002")
	}
	if !strings.Contains(content, "TZ=America/New_York") {
		t.Error("Content missing TZ=America/New_York")
	}
	if !strings.Contains(content, "APPDATA_PATH=/custom/path") {
		t.Error("Content missing APPDATA_PATH=/custom/path")
	}
}

func TestGenerateEnvContent_Media(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Set("ENV_PUID", "1000"); err != nil {
		t.Fatalf("failed to set ENV_PUID: %v", err)
	}
	if err := cfg.Set("ENV_PGID", "1000"); err != nil {
		t.Fatalf("failed to set ENV_PGID: %v", err)
	}
	if err := cfg.Set("ENV_TZ", "UTC"); err != nil {
		t.Fatalf("failed to set ENV_TZ: %v", err)
	}
	if err := cfg.Set("ENV_APPDATA_PATH", "/data"); err != nil {
		t.Fatalf("failed to set ENV_APPDATA_PATH: %v", err)
	}
	if err := cfg.Set("PLEX_CLAIM_TOKEN", "claim-test-token"); err != nil {
		t.Fatalf("failed to set PLEX_CLAIM_TOKEN: %v", err)
	}
	if err := cfg.Set("JELLYFIN_PUBLIC_URL", "https://jellyfin.example.com"); err != nil {
		t.Fatalf("failed to set JELLYFIN_PUBLIC_URL: %v", err)
	}

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	setup := NewContainerSetup(containers, fs, cfg, uiInstance, markers)

	content := setup.generateEnvContent("media")

	// Check media-specific variables
	if !strings.Contains(content, "PLEX_CLAIM_TOKEN=claim-test-token") {
		t.Error("Content missing PLEX_CLAIM_TOKEN")
	}
	if !strings.Contains(content, "JELLYFIN_PUBLIC_URL=https://jellyfin.example.com") {
		t.Error("Content missing JELLYFIN_PUBLIC_URL")
	}
	if !strings.Contains(content, "TRANSCODE_DEVICE=/dev/dri") {
		t.Error("Content missing TRANSCODE_DEVICE")
	}
}

func TestGetServiceInfo(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Set("CONTAINERS_BASE", "/test/containers"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	services := system.NewServiceManager()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)

	info := deployment.GetServiceInfo("media")

	if info.Name != "media" {
		t.Errorf("Expected Name=media, got %s", info.Name)
	}
	if info.DisplayName != "Media" {
		t.Errorf("Expected DisplayName=Media, got %s", info.DisplayName)
	}
	if info.Directory != "/test/containers/media" {
		t.Errorf("Expected Directory=/test/containers/media, got %s", info.Directory)
	}
	if info.UnitName != "podman-compose-media.service" {
		t.Errorf("Expected UnitName=podman-compose-media.service, got %s", info.UnitName)
	}
}

func TestContainerSetupServiceDirectory(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Set("CONTAINERS_BASE", "/srv/containers"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	setup := NewContainerSetup(system.NewContainerManager(), system.NewFileSystem(), cfg, ui.New(), config.NewMarkers(""))

	dir := setup.serviceDirectory("media")
	expected := filepath.Join("/srv/containers", "media")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}

func TestContainerSetupServiceDirectoryCustomPath(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Set("CONTAINERS_BASE", "/custom/path"); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	setup := NewContainerSetup(system.NewContainerManager(), system.NewFileSystem(), cfg, ui.New(), config.NewMarkers(""))

	dir := setup.serviceDirectory("web")
	expected := filepath.Join("/custom/path", "web")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}
