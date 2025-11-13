package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/cli"
)

var (
	// Flags for non-interactive mode
	nonInteractive     bool
	setupUser          string
	nfsServer          string
	homelabBaseDir     string
	skipWireguard      bool
	plexClaimToken     string
	jellyfinURL        string
	overseerrAPI       string
	nextcloudAdmin     string
	nextcloudAdminPass string
	nextcloudDBPass    string
	nextcloudDomain    string
	collaboraPass      string
	immichDBPass       string
	postgresUser       string
	redisPass          string
)

var runCmd = &cobra.Command{
	Use:   "run [step|all|quick]",
	Short: "Run setup steps",
	Long: `Run one or more setup steps.

Steps:
  all         - Run all setup steps
  quick       - Run all steps except WireGuard
  preflight   - Pre-flight system checks
  user        - User and group configuration
  directory   - Directory structure creation
  wireguard   - WireGuard VPN setup
  nfs         - NFS mount configuration
  container   - Container service setup
  deployment  - Service deployment`,
	Args: cobra.ExactArgs(1),
	RunE: runSetup,
}

func init() {
	runCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode")
	runCmd.Flags().StringVar(&setupUser, "setup-user", "", "Username for homelab setup")
	runCmd.Flags().StringVar(&nfsServer, "nfs-server", "", "NFS server address")
	runCmd.Flags().StringVar(&homelabBaseDir, "homelab-base-dir", "", "Base directory for homelab")
	runCmd.Flags().BoolVar(&skipWireguard, "skip-wireguard", false, "Skip WireGuard setup")
	runCmd.Flags().StringVar(&plexClaimToken, "plex-token", "", "Plex claim token for media stack")
	runCmd.Flags().StringVar(&jellyfinURL, "jellyfin-url", "", "Jellyfin public URL")
	runCmd.Flags().StringVar(&overseerrAPI, "overseerr-api-key", "", "Overseerr API key")
	runCmd.Flags().StringVar(&nextcloudAdmin, "nextcloud-admin-user", "", "Nextcloud admin username")
	runCmd.Flags().StringVar(&nextcloudAdminPass, "nextcloud-admin-password", "", "Nextcloud admin password")
	runCmd.Flags().StringVar(&nextcloudDBPass, "nextcloud-db-password", "", "Nextcloud database password")
	runCmd.Flags().StringVar(&nextcloudDomain, "nextcloud-domain", "", "Nextcloud trusted domain")
	runCmd.Flags().StringVar(&collaboraPass, "collabora-password", "", "Collabora admin password")
	runCmd.Flags().StringVar(&immichDBPass, "immich-db-password", "", "Immich database password")
	runCmd.Flags().StringVar(&postgresUser, "postgres-user", "", "PostgreSQL username for Immich stack")
	runCmd.Flags().StringVar(&redisPass, "redis-password", "", "Redis password for Immich stack")

	rootCmd.AddCommand(runCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Create setup context with non-interactive mode if requested
	ctx, err := cli.NewSetupContextWithOptions(nonInteractive)
	if err != nil {
		return fmt.Errorf("failed to initialize setup context: %w", err)
	}

	// Apply non-interactive config if provided
	if nonInteractive {
		if err := applyNonInteractiveConfig(ctx); err != nil {
			return err
		}
		ctx.UI.Info("Running in non-interactive mode")
	}

	step := args[0]

	switch step {
	case "all":
		return ctx.Steps.RunAll(skipWireguard)
	case "quick":
		return ctx.Steps.RunAll(true)
	case "preflight", "user", "directory", "wireguard", "nfs", "container", "deployment":
		return ctx.Steps.RunStep(step)
	default:
		return fmt.Errorf("unknown step: %s", step)
	}
}

func applyNonInteractiveConfig(ctx *cli.SetupContext) error {
	if setupUser != "" {
		if err := ctx.Config.Set("SETUP_USER", setupUser); err != nil {
			return fmt.Errorf("failed to set SETUP_USER: %w", err)
		}
	}

	if nfsServer != "" {
		if err := ctx.Config.Set("NFS_SERVER", nfsServer); err != nil {
			return fmt.Errorf("failed to set NFS_SERVER: %w", err)
		}
	}

	if homelabBaseDir != "" {
		if err := ctx.Config.Set("HOMELAB_BASE_DIR", homelabBaseDir); err != nil {
			return fmt.Errorf("failed to set HOMELAB_BASE_DIR: %w", err)
		}
	}

	if plexClaimToken != "" {
		if err := ctx.Config.Set("PLEX_CLAIM_TOKEN", plexClaimToken); err != nil {
			return fmt.Errorf("failed to set PLEX_CLAIM_TOKEN: %w", err)
		}
	}

	if jellyfinURL != "" {
		if err := ctx.Config.Set("JELLYFIN_PUBLIC_URL", jellyfinURL); err != nil {
			return fmt.Errorf("failed to set JELLYFIN_PUBLIC_URL: %w", err)
		}
	}

	if overseerrAPI != "" {
		if err := ctx.Config.Set("OVERSEERR_API_KEY", overseerrAPI); err != nil {
			return fmt.Errorf("failed to set OVERSEERR_API_KEY: %w", err)
		}
	}

	if nextcloudAdmin != "" {
		if err := ctx.Config.Set("NEXTCLOUD_ADMIN_USER", nextcloudAdmin); err != nil {
			return fmt.Errorf("failed to set NEXTCLOUD_ADMIN_USER: %w", err)
		}
	}

	if nextcloudAdminPass != "" {
		if err := ctx.Config.Set("NEXTCLOUD_ADMIN_PASSWORD", nextcloudAdminPass); err != nil {
			return fmt.Errorf("failed to set NEXTCLOUD_ADMIN_PASSWORD: %w", err)
		}
	}

	if nextcloudDBPass != "" {
		if err := ctx.Config.Set("NEXTCLOUD_DB_PASSWORD", nextcloudDBPass); err != nil {
			return fmt.Errorf("failed to set NEXTCLOUD_DB_PASSWORD: %w", err)
		}
	}

	if nextcloudDomain != "" {
		if err := ctx.Config.Set("NEXTCLOUD_TRUSTED_DOMAINS", nextcloudDomain); err != nil {
			return fmt.Errorf("failed to set NEXTCLOUD_TRUSTED_DOMAINS: %w", err)
		}
	}

	if collaboraPass != "" {
		if err := ctx.Config.Set("COLLABORA_PASSWORD", collaboraPass); err != nil {
			return fmt.Errorf("failed to set COLLABORA_PASSWORD: %w", err)
		}
	}

	if immichDBPass != "" {
		if err := ctx.Config.Set("IMMICH_DB_PASSWORD", immichDBPass); err != nil {
			return fmt.Errorf("failed to set IMMICH_DB_PASSWORD: %w", err)
		}
	}

	if postgresUser != "" {
		if err := ctx.Config.Set("POSTGRES_USER", postgresUser); err != nil {
			return fmt.Errorf("failed to set POSTGRES_USER: %w", err)
		}
	}

	if redisPass != "" {
		if err := ctx.Config.Set("REDIS_PASSWORD", redisPass); err != nil {
			return fmt.Errorf("failed to set REDIS_PASSWORD: %w", err)
		}
	}

	return nil
}
