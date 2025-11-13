package steps

import (
	"testing"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

func TestNewDeployment(t *testing.T) {
	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	services := system.NewServiceManager()
	cfg := config.New("")
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)

	if deployment == nil {
		t.Fatal("NewDeployment returned nil")
	}

	if deployment.containers == nil {
		t.Error("Deployment.containers is nil")
	}
	if deployment.fs == nil {
		t.Error("Deployment.fs is nil")
	}
	if deployment.services == nil {
		t.Error("Deployment.services is nil")
	}
	if deployment.config == nil {
		t.Error("Deployment.config is nil")
	}
	if deployment.ui == nil {
		t.Error("Deployment.ui is nil")
	}
	if deployment.markers == nil {
		t.Error("Deployment.markers is nil")
	}
}

func TestGetSelectedServices(t *testing.T) {
	cfg := config.New("")

	containers := system.NewContainerManager()
	fs := system.NewFileSystem()
	services := system.NewServiceManager()
	uiInstance := ui.New()
	markers := config.NewMarkers("")

	deployment := NewDeployment(containers, fs, services, cfg, uiInstance, markers)

	// Test with no services selected
	_, err := deployment.GetSelectedServices()
	if err == nil {
		t.Error("Expected error when no services selected, got nil")
	}

	// Test with services selected
	cfg.Set("SELECTED_SERVICES", "media web cloud")
	selectedServices, err := deployment.GetSelectedServices()
	if err != nil {
		t.Fatalf("GetSelectedServices failed: %v", err)
	}

	expected := []string{"media", "web", "cloud"}
	if len(selectedServices) != len(expected) {
		t.Errorf("Expected %d services, got %d", len(expected), len(selectedServices))
	}

	for i, service := range selectedServices {
		if service != expected[i] {
			t.Errorf("Expected service %s at index %d, got %s", expected[i], i, service)
		}
	}
}
