package cli

import (
	"fmt"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/steps"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// SetupContext holds all dependencies needed for setup operations
type SetupContext struct {
	Config  *config.Config
	Markers *config.Markers
	UI      *ui.UI
	// SkipWireGuard indicates whether WireGuard should be skipped when running all steps
	SkipWireGuard bool
}

// NewSetupContext creates a new SetupContext with all dependencies initialized
func NewSetupContext() (*SetupContext, error) {
	return NewSetupContextWithOptions(false, false)
}

// NewSetupContextWithOptions creates a new SetupContext with custom options
func NewSetupContextWithOptions(nonInteractive bool, skipWireGuard bool) (*SetupContext, error) {
	// Initialize configuration
	cfg := config.New("")
	if err := cfg.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize markers
	markers := config.NewMarkers("")

	// Initialize UI
	uiInstance := ui.New()
	uiInstance.SetNonInteractive(nonInteractive)

	return &SetupContext{
		Config:        cfg,
		Markers:       markers,
		UI:            uiInstance,
		SkipWireGuard: skipWireGuard,
	}, nil
}

// StepInfo contains metadata about a setup step
type StepInfo struct {
	Name        string
	ShortName   string
	Description string
	MarkerName  string
	Optional    bool
}

// GetAllSteps returns information about all steps in order
func GetAllSteps() []StepInfo {
	return []StepInfo{
		{Name: "Pre-flight Check", ShortName: "preflight", Description: "Verify system requirements", MarkerName: "preflight-complete", Optional: false},
		{Name: "User Setup", ShortName: "user", Description: "Configure user account and permissions", MarkerName: "user-setup-complete", Optional: false},
		{Name: "Directory Setup", ShortName: "directory", Description: "Create directory structure", MarkerName: "directory-setup-complete", Optional: false},
		{Name: "WireGuard Setup", ShortName: "wireguard", Description: "Configure VPN (optional)", MarkerName: "wireguard-setup-complete", Optional: true},
		{Name: "NFS Setup", ShortName: "nfs", Description: "Configure network storage", MarkerName: "nfs-setup-complete", Optional: false},
		{Name: "Container Setup", ShortName: "container", Description: "Configure container services", MarkerName: "container-setup-complete", Optional: false},
		{Name: "Service Deployment", ShortName: "deployment", Description: "Deploy and start services", MarkerName: "service-deployment-complete", Optional: false},
	}
}

// IsStepComplete checks if a step is complete
func IsStepComplete(markers *config.Markers, markerName string) bool {
	exists, err := markers.Exists(markerName)
	if err != nil {
		return false
	}
	return exists
}

// removeMarkerIfRerun removes a marker if the user chooses to rerun the step
func removeMarkerIfRerun(ui *ui.UI, markers *config.Markers, markerName string, rerun bool) {
	if rerun {
		if err := markers.Remove(markerName); err != nil {
			ui.Warning(fmt.Sprintf("Failed to remove marker: %v", err))
		}
	}
}

// RunStep executes a specific step by short name
func RunStep(ctx *SetupContext, shortName string) error {
	ctx.UI.Header(fmt.Sprintf("Running: %s", shortName))

	var err error
	var markerName string

	switch shortName {
	case "preflight":
		markerName = "preflight-complete"
		err = runPreflight(ctx)
	case "user":
		markerName = "user-setup-complete"
		err = runUser(ctx)
	case "directory":
		markerName = "directory-setup-complete"
		err = runDirectory(ctx)
	case "wireguard":
		markerName = "wireguard-setup-complete"
		err = runWireGuard(ctx)
	case "nfs":
		markerName = "nfs-setup-complete"
		err = runNFS(ctx)
	case "container":
		markerName = "container-setup-complete"
		err = runContainer(ctx)
	case "deployment":
		markerName = "service-deployment-complete"
		err = runDeployment(ctx)
	default:
		return fmt.Errorf("unknown step: %s", shortName)
	}

	if err != nil {
		// Mark step as failed
		if markErr := ctx.Markers.MarkFailed(markerName); markErr != nil {
			ctx.UI.Warning(fmt.Sprintf("Failed to create failure marker: %v", markErr))
		}
		return err
	}

	// Clear any previous failure marker
	if clearErr := ctx.Markers.ClearFailure(markerName); clearErr != nil {
		ctx.UI.Warning(fmt.Sprintf("Failed to clear failure marker: %v", clearErr))
	}

	// Mark step as complete
	if err := ctx.Markers.Create(markerName); err != nil {
		ctx.UI.Warning(fmt.Sprintf("Failed to create completion marker: %v", err))
	}

	ctx.UI.Success(fmt.Sprintf("Step '%s' completed successfully!", shortName))
	return nil
}

// AddWireGuardPeer invokes the WireGuard peer workflow helper.
func AddWireGuardPeer(ctx *SetupContext, opts *steps.WireGuardPeerWorkflowOptions) error {
	return steps.NewWireGuardSetup(ctx.Config, ctx.UI, ctx.Markers).AddPeerWorkflow(opts)
}

// Individual step runners
func runPreflight(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "preflight-complete") {
		ctx.UI.Info("Pre-flight check already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "preflight-complete", rerun)
	}

	// Use the RunAll method that exists in PreflightChecker
	return steps.NewPreflightChecker(ctx.UI, ctx.Markers, ctx.Config).RunAll()
}

func runUser(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "user-setup-complete") {
		ctx.UI.Info("User setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "user-setup-complete", rerun)
	}

	// Use the Run method that exists in UserConfigurator
	return steps.NewUserConfigurator(ctx.Config, ctx.UI, ctx.Markers).Run()
}

func runDirectory(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "directory-setup-complete") {
		ctx.UI.Info("Directory setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "directory-setup-complete", rerun)
	}

	// Use the Run method that exists in DirectorySetup
	return steps.NewDirectorySetup(ctx.Config, ctx.UI, ctx.Markers).Run()
}

func runWireGuard(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "wireguard-setup-complete") {
		ctx.UI.Info("WireGuard setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "wireguard-setup-complete", rerun)
	}

	// Use the Run method that exists in WireGuardSetup
	// This handles all the logic including prompting, key generation, config writing, etc.
	return steps.NewWireGuardSetup(ctx.Config, ctx.UI, ctx.Markers).Run()
}

func runNFS(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "nfs-setup-complete") {
		ctx.UI.Info("NFS setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "nfs-setup-complete", rerun)
	}

	// Use the Run method that exists in NFSConfigurator
	return steps.NewNFSConfigurator(ctx.Config, ctx.UI, ctx.Markers).Run()
}

func runContainer(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "container-setup-complete") {
		ctx.UI.Info("Container setup already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "container-setup-complete", rerun)
	}

	// Use the Run method that exists in ContainerSetup
	return steps.NewContainerSetup(ctx.Config, ctx.UI, ctx.Markers).Run()
}

func runDeployment(ctx *SetupContext) error {
	// Check if already completed
	if IsStepComplete(ctx.Markers, "service-deployment-complete") {
		ctx.UI.Info("Service deployment already completed")
		rerun, err := ctx.UI.PromptYesNo("Run again?", false)
		if err != nil || !rerun {
			return nil
		}
		removeMarkerIfRerun(ctx.UI, ctx.Markers, "service-deployment-complete", rerun)
	}

	// Use the Run method that exists in Deployment
	return steps.NewDeployment(ctx.Config, ctx.UI, ctx.Markers).Run()
}

// RunAll runs all setup steps in order

func RunAll(ctx *SetupContext, skipWireGuard bool) error {
	steps := []string{"preflight", "user", "directory"}

	if !skipWireGuard {
		steps = append(steps, "wireguard")
	}

	steps = append(steps, "nfs", "container", "deployment")

	for _, step := range steps {
		if err := RunStep(ctx, step); err != nil {
			return fmt.Errorf("step %s failed: %w", step, err)
		}
	}

	ctx.UI.Success("All steps completed successfully!")
	return nil
}
