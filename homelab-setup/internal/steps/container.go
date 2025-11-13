package steps

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/config"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/system"
	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// ContainerSetup handles container stack setup and configuration
type ContainerSetup struct {
	containers *system.ContainerManager
	fs         *system.FileSystem
	config     *config.Config
	ui         *ui.UI
	markers    *config.Markers
}

var (
	stackRequiredConfigKeys = map[string][]string{
		"cloud": {
			"NEXTCLOUD_ADMIN_PASSWORD",
			"NEXTCLOUD_DB_PASSWORD",
			"COLLABORA_PASSWORD",
			"IMMICH_DB_PASSWORD",
			"REDIS_PASSWORD",
		},
	}

	configFlagHints = map[string]string{
		"NEXTCLOUD_ADMIN_PASSWORD": "--nextcloud-admin-password",
		"NEXTCLOUD_DB_PASSWORD":    "--nextcloud-db-password",
		"COLLABORA_PASSWORD":       "--collabora-password",
		"IMMICH_DB_PASSWORD":       "--immich-db-password",
		"REDIS_PASSWORD":           "--redis-password",
	}

	configValueDescriptions = map[string]string{
		"NEXTCLOUD_ADMIN_PASSWORD": "Nextcloud admin password",
		"NEXTCLOUD_DB_PASSWORD":    "Nextcloud database password",
		"COLLABORA_PASSWORD":       "Collabora admin password",
		"IMMICH_DB_PASSWORD":       "Immich database password",
		"REDIS_PASSWORD":           "Redis password",
	}
)

// NewContainerSetup creates a new ContainerSetup instance
func NewContainerSetup(containers *system.ContainerManager, fs *system.FileSystem, cfg *config.Config, ui *ui.UI, markers *config.Markers) *ContainerSetup {
	return &ContainerSetup{
		containers: containers,
		fs:         fs,
		config:     cfg,
		ui:         ui,
		markers:    markers,
	}
}

// FindTemplateDirectory locates compose templates
func (c *ContainerSetup) FindTemplateDirectory() (string, error) {
	c.ui.Step("Locating Compose Templates")

	// Check home setup directory first
	homeDir := os.Getenv("HOME")
	templateDirHome := filepath.Join(homeDir, "setup", "compose-setup")

	if exists, _ := c.fs.DirectoryExists(templateDirHome); exists {
		// Count YAML files
		count, _ := c.countYAMLFiles(templateDirHome)
		if count > 0 {
			c.ui.Successf("Found templates in: %s (%d YAML file(s))", templateDirHome, count)
			return templateDirHome, nil
		}
		c.ui.Warningf("Directory exists but contains no YAML files: %s", templateDirHome)
	}

	// Check /usr/share as fallback
	templateDirUsr := "/usr/share/compose-setup"
	if exists, _ := c.fs.DirectoryExists(templateDirUsr); exists {
		count, _ := c.countYAMLFiles(templateDirUsr)
		if count > 0 {
			c.ui.Successf("Found templates in: %s (%d YAML file(s))", templateDirUsr, count)
			return templateDirUsr, nil
		}
		c.ui.Warningf("Directory exists but contains no YAML files: %s", templateDirUsr)
	}

	c.ui.Error("No compose templates found in any location")
	c.ui.Info("Searched locations:")
	c.ui.Infof("  - %s", templateDirHome)
	c.ui.Infof("  - %s", templateDirUsr)
	c.ui.Print("")
	c.ui.Info("Expected to find .yml or .yaml files in one of these directories")

	return "", fmt.Errorf("no compose templates found")
}

// countYAMLFiles counts YAML files in a directory
func (c *ContainerSetup) countYAMLFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := filepath.Ext(entry.Name())
		if ext == ".yml" || ext == ".yaml" {
			count++
		}
	}

	return count, nil
}

// DiscoverStacks discovers available container stacks
func (c *ContainerSetup) DiscoverStacks(templateDir string) (map[string]string, error) {
	c.ui.Step("Discovering Available Container Stacks")
	c.ui.Infof("Scanning directory: %s", templateDir)

	// Exclude patterns
	excludePatterns := []string{
		".*",        // Hidden files
		"*.example", // Example files
		"README*",   // Documentation files
		"*.md",      // Markdown files
	}

	entries, err := os.ReadDir(templateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
	}

	stacks := make(map[string]string)
	totalYAML := 0
	excludedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		ext := filepath.Ext(filename)

		// Only process YAML files
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		totalYAML++

		// Check exclude patterns
		shouldExclude := false
		for _, pattern := range excludePatterns {
			matched, _ := filepath.Match(pattern, filename)
			if matched {
				c.ui.Infof("Excluding: %s (matches pattern: %s)", filename, pattern)
				excludedCount++
				shouldExclude = true
				break
			}
		}

		if shouldExclude {
			continue
		}

		// Get service name (filename without extension)
		serviceName := strings.TrimSuffix(filename, ext)
		stacks[serviceName] = filename
		c.ui.Successf("Found stack: %s (%s)", serviceName, filename)
	}

	if len(stacks) == 0 {
		c.ui.Error("No valid compose stack files discovered")
		c.ui.Infof("Directory checked: %s", templateDir)
		c.ui.Infof("Total YAML files found: %d", totalYAML)
		c.ui.Infof("Files excluded by patterns: %d", excludedCount)
		c.ui.Print("")
		c.ui.Info("Exclude patterns:")
		for _, pattern := range excludePatterns {
			c.ui.Infof("  - %s", pattern)
		}
		c.ui.Print("")
		c.ui.Info("Stack files should be named like: media.yml, web.yml, cloud.yml")
		c.ui.Info("Excluded files: .env.example, README.md, .hidden files")
		return nil, fmt.Errorf("no valid stacks found")
	}

	c.ui.Successf("Discovered %d valid container stack(s) (excluded %d file(s))", len(stacks), excludedCount)
	return stacks, nil
}

// SelectStacks allows user to select which stacks to setup
func (c *ContainerSetup) SelectStacks(stacks map[string]string) ([]string, error) {
	c.ui.Step("Container Stack Selection")
	c.ui.Print("")
	c.ui.Info("Available container stacks:")
	c.ui.Print("")

	// Sort stack names for consistent ordering
	var stackNames []string
	for name := range stacks {
		stackNames = append(stackNames, name)
	}
	sort.Strings(stackNames)

	// Display available stacks
	for i, name := range stackNames {
		c.ui.Printf("  %d) %s (%s)", i+1, name, stacks[name])
	}
	c.ui.Printf("  %d) All stacks", len(stackNames)+1)
	c.ui.Print("")

	// Prompt for selection
	c.ui.Info("Select which container stacks to setup:")
	c.ui.Info("  - You can select multiple stacks using Space, then press Enter")
	c.ui.Info("  - Or select 'All stacks' to setup everything")
	c.ui.Print("")

	// Build options for multi-select
	var options []string
	for _, name := range stackNames {
		options = append(options, fmt.Sprintf("%s (%s)", name, stacks[name]))
	}
	options = append(options, "All stacks")

	// Use multi-select prompt
	selectedIndices, err := c.ui.PromptMultiSelect("Select stacks to setup", options)
	if err != nil {
		return nil, fmt.Errorf("failed to prompt for stack selection: %w", err)
	}

	if len(selectedIndices) == 0 {
		return nil, fmt.Errorf("no stacks selected")
	}

	// Check if "All stacks" was selected
	allStacksIndex := len(stackNames)
	for _, idx := range selectedIndices {
		if idx == allStacksIndex {
			c.ui.Success("Selected: All stacks")
			// Save selected services to config before returning
			if err := c.config.Set("SELECTED_SERVICES", strings.Join(stackNames, " ")); err != nil {
				c.ui.Warning(fmt.Sprintf("Failed to save selected services: %v", err))
			}
			return stackNames, nil
		}
	}

	// Get selected stack names
	var selected []string
	for _, idx := range selectedIndices {
		if idx < len(stackNames) {
			selected = append(selected, stackNames[idx])
		}
	}

	c.ui.Success("Selected stacks:")
	for _, name := range selected {
		c.ui.Infof("  - %s", name)
	}
	c.ui.Print("")

	// Save selected services to config
	if err := c.config.Set("SELECTED_SERVICES", strings.Join(selected, " ")); err != nil {
		c.ui.Warning(fmt.Sprintf("Failed to save selected services: %v", err))
	}

	return selected, nil
}

// CopyTemplates copies selected compose templates to destination
func (c *ContainerSetup) CopyTemplates(templateDir string, stacks map[string]string, selectedStacks []string) error {
	c.ui.Step("Copying Compose Templates")

	setupUser := c.config.GetOrDefault("HOMELAB_USER", "")
	if setupUser == "" {
		return fmt.Errorf("homelab user not configured")
	}

	containersBase := c.config.GetOrDefault("CONTAINERS_BASE", "/srv/containers")

	for _, serviceName := range selectedStacks {
		templateFile := stacks[serviceName]
		srcPath := filepath.Join(templateDir, templateFile)
		dstDir := filepath.Join(containersBase, serviceName)
		dstPath := filepath.Join(dstDir, "compose.yml")

		// Ensure destination directory exists
		if err := c.fs.EnsureDirectory(dstDir, setupUser, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dstDir, err)
		}

		// Copy template
		c.ui.Infof("Copying: %s → %s", templateFile, dstPath)
		if err := c.fs.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", templateFile, err)
		}

		// Set ownership and permissions
		if err := c.fs.Chown(dstPath, fmt.Sprintf("%s:%s", setupUser, setupUser)); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", dstPath, err)
		}

		if err := c.fs.Chmod(dstPath, 0644); err != nil {
			return fmt.Errorf("failed to set permissions on %s: %w", dstPath, err)
		}

		c.ui.Successf("✓ %s/compose.yml", serviceName)

		// Also create docker-compose.yml symlink for compatibility
		altDstPath := filepath.Join(dstDir, "docker-compose.yml")
		if exists, _ := c.fs.FileExists(altDstPath); !exists {
			if err := c.fs.CreateSymlink("compose.yml", altDstPath); err != nil {
				c.ui.Warning(fmt.Sprintf("Failed to create symlink: %v", err))
			}
		}
	}

	c.ui.Successf("Copied %d compose file(s)", len(selectedStacks))
	return nil
}

// CreateBaseEnvConfig creates base environment configuration
func (c *ContainerSetup) CreateBaseEnvConfig() error {
	c.ui.Step("Creating Base Environment Configuration")

	// Load or prompt for configuration values
	puid := c.config.GetOrDefault("PUID", "1000")
	pgid := c.config.GetOrDefault("PGID", "1000")
	tz := c.config.GetOrDefault("TZ", "America/Chicago")
	appdataPath := c.config.GetOrDefault("APPDATA_PATH", "/var/lib/containers/appdata")

	// Save base config
	if err := c.config.Set("ENV_PUID", puid); err != nil {
		return fmt.Errorf("failed to save ENV_PUID: %w", err)
	}
	if err := c.config.Set("ENV_PGID", pgid); err != nil {
		return fmt.Errorf("failed to save ENV_PGID: %w", err)
	}
	if err := c.config.Set("ENV_TZ", tz); err != nil {
		return fmt.Errorf("failed to save ENV_TZ: %w", err)
	}
	if err := c.config.Set("ENV_APPDATA_PATH", appdataPath); err != nil {
		return fmt.Errorf("failed to save ENV_APPDATA_PATH: %w", err)
	}

	c.ui.Success("Base configuration:")
	c.ui.Infof("  PUID=%s", puid)
	c.ui.Infof("  PGID=%s", pgid)
	c.ui.Infof("  TZ=%s", tz)
	c.ui.Infof("  APPDATA_PATH=%s", appdataPath)

	return nil
}

// ConfigureStackEnv configures environment for a specific stack
func (c *ContainerSetup) ConfigureStackEnv(serviceName string) error {
	switch serviceName {
	case "media":
		return c.configureMediaEnv()
	case "web":
		return c.configureWebEnv()
	case "cloud":
		return c.configureCloudEnv()
	default:
		c.ui.Infof("No specific configuration for %s stack", serviceName)
		return nil
	}
}

// ensureNonInteractiveRequirements validates required config exists before continuing
func (c *ContainerSetup) ensureNonInteractiveRequirements(selectedStacks []string) error {
	if !c.ui.IsNonInteractive() {
		return nil
	}

	for _, stack := range selectedStacks {
		keys, ok := stackRequiredConfigKeys[stack]
		if !ok {
			continue
		}
		for _, key := range keys {
			if value := c.config.GetOrDefault(key, ""); value == "" {
				desc := configValueDescriptions[key]
				if desc == "" {
					desc = key
				}
				flag := configFlagHints[key]
				if flag != "" {
					return fmt.Errorf("non-interactive mode requires %s (set via %s or config key %s)", desc, flag, key)
				}
				return fmt.Errorf("non-interactive mode requires config value %s", key)
			}
		}
	}

	return nil
}

// promptOrConfigValue returns a config value or prompts the user when interactive
func (c *ContainerSetup) promptOrConfigValue(key, prompt, defaultValue string, required bool, secret bool) (string, error) {
	if c.ui.IsNonInteractive() {
		value := c.config.GetOrDefault(key, defaultValue)
		if required && value == "" {
			desc := configValueDescriptions[key]
			if desc == "" {
				desc = prompt
			}
			return "", fmt.Errorf("non-interactive mode requires %s (config key %s)", desc, key)
		}
		return value, nil
	}

	var (
		value string
		err   error
	)
	if secret {
		value, err = c.ui.PromptPasswordConfirm(prompt)
	} else {
		value, err = c.ui.PromptInput(prompt, defaultValue)
	}
	if err != nil {
		return "", err
	}
	if required && value == "" {
		return "", fmt.Errorf("%s is required", prompt)
	}
	return value, nil
}

// configureMediaEnv configures media stack environment
func (c *ContainerSetup) configureMediaEnv() error {
	c.ui.Step("Configuring Media Stack Environment")

	// Get Plex claim token
	c.ui.Info("Plex Setup:")
	c.ui.Info("  Get your claim token from: https://plex.tv/claim")
	plexClaim, err := c.promptOrConfigValue("PLEX_CLAIM_TOKEN", "Plex claim token (optional)", "", false, false)
	if err != nil {
		return err
	}
	if plexClaim != "" {
		if err := c.config.Set("PLEX_CLAIM_TOKEN", plexClaim); err != nil {
			return fmt.Errorf("failed to save PLEX_CLAIM_TOKEN: %w", err)
		}
	}

	// Jellyfin public URL
	jellyfinURL, err := c.promptOrConfigValue("JELLYFIN_PUBLIC_URL", "Jellyfin public URL (optional)", "", false, false)
	if err != nil {
		return err
	}
	if jellyfinURL != "" {
		if err := c.config.Set("JELLYFIN_PUBLIC_URL", jellyfinURL); err != nil {
			return fmt.Errorf("failed to save JELLYFIN_PUBLIC_URL: %w", err)
		}
	}

	return nil
}

// configureWebEnv configures web stack environment
func (c *ContainerSetup) configureWebEnv() error {
	c.ui.Step("Configuring Web Stack Environment")

	// Overseerr API key (optional)
	overseerrAPI, err := c.promptOrConfigValue("OVERSEERR_API_KEY", "Overseerr API key (optional, can configure later)", "", false, false)
	if err != nil {
		return err
	}
	if overseerrAPI != "" {
		if err := c.config.Set("OVERSEERR_API_KEY", overseerrAPI); err != nil {
			return fmt.Errorf("failed to save OVERSEERR_API_KEY: %w", err)
		}
	}

	return nil
}

// configureCloudEnv configures cloud stack environment
func (c *ContainerSetup) configureCloudEnv() error {
	c.ui.Step("Configuring Cloud Stack Environment")

	// Nextcloud configuration
	c.ui.Info("Nextcloud Setup:")
	c.ui.Print("")

	nextcloudAdminUser, err := c.promptOrConfigValue("NEXTCLOUD_ADMIN_USER", "Nextcloud admin username", "admin", false, false)
	if err != nil {
		return err
	}
	if err := c.config.Set("NEXTCLOUD_ADMIN_USER", nextcloudAdminUser); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_ADMIN_USER: %w", err)
	}

	nextcloudAdminPass, err := c.promptOrConfigValue("NEXTCLOUD_ADMIN_PASSWORD", "Nextcloud admin password", "", true, true)
	if err != nil {
		return err
	}
	if err := c.config.Set("NEXTCLOUD_ADMIN_PASSWORD", nextcloudAdminPass); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_ADMIN_PASSWORD: %w", err)
	}

	nextcloudDBPass, err := c.promptOrConfigValue("NEXTCLOUD_DB_PASSWORD", "Nextcloud database password", "", true, true)
	if err != nil {
		return err
	}
	if err := c.config.Set("NEXTCLOUD_DB_PASSWORD", nextcloudDBPass); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_DB_PASSWORD: %w", err)
	}

	nextcloudDomain, err := c.promptOrConfigValue("NEXTCLOUD_TRUSTED_DOMAINS", "Nextcloud trusted domain (e.g., cloud.example.com)", "localhost", false, false)
	if err != nil {
		return err
	}
	if err := c.config.Set("NEXTCLOUD_TRUSTED_DOMAINS", nextcloudDomain); err != nil {
		return fmt.Errorf("failed to save NEXTCLOUD_TRUSTED_DOMAINS: %w", err)
	}

	// Collabora configuration
	c.ui.Print("")
	c.ui.Info("Collabora Setup:")
	c.ui.Print("")

	collaboraPass, err := c.promptOrConfigValue("COLLABORA_PASSWORD", "Collabora admin password", "", true, true)
	if err != nil {
		return err
	}
	if err := c.config.Set("COLLABORA_PASSWORD", collaboraPass); err != nil {
		return fmt.Errorf("failed to save COLLABORA_PASSWORD: %w", err)
	}

	// Escape domain for Collabora (dots need to be escaped)
	collaboraDomain := strings.ReplaceAll(nextcloudDomain, ".", "\\.")
	if err := c.config.Set("COLLABORA_DOMAIN", collaboraDomain); err != nil {
		return fmt.Errorf("failed to save COLLABORA_DOMAIN: %w", err)
	}

	// Immich configuration
	c.ui.Print("")
	c.ui.Info("Immich Setup:")
	c.ui.Print("")

	immichDBPass, err := c.promptOrConfigValue("IMMICH_DB_PASSWORD", "Immich database password", "", true, true)
	if err != nil {
		return err
	}
	if err := c.config.Set("IMMICH_DB_PASSWORD", immichDBPass); err != nil {
		return fmt.Errorf("failed to save IMMICH_DB_PASSWORD: %w", err)
	}

	// Postgres user
	postgresUser, err := c.promptOrConfigValue("POSTGRES_USER", "PostgreSQL username", "homelab", false, false)
	if err != nil {
		return err
	}
	if err := c.config.Set("POSTGRES_USER", postgresUser); err != nil {
		return fmt.Errorf("failed to save POSTGRES_USER: %w", err)
	}

	// Redis password
	redisPass, err := c.promptOrConfigValue("REDIS_PASSWORD", "Redis password", "", true, true)
	if err != nil {
		return err
	}
	if err := c.config.Set("REDIS_PASSWORD", redisPass); err != nil {
		return fmt.Errorf("failed to save REDIS_PASSWORD: %w", err)
	}

	return nil
}

// CreateEnvFiles creates .env files for selected stacks
func (c *ContainerSetup) CreateEnvFiles(selectedStacks []string) error {
	c.ui.Step("Creating Environment Files")

	setupUser := c.config.GetOrDefault("HOMELAB_USER", "")
	containersBase := c.config.GetOrDefault("CONTAINERS_BASE", "/srv/containers")

	for _, serviceName := range selectedStacks {
		envPath := filepath.Join(containersBase, serviceName, ".env")
		c.ui.Infof("Creating environment file: %s", envPath)

		content := c.generateEnvContent(serviceName)

		// Write file
		if err := c.fs.WriteFile(envPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write .env file for %s: %w", serviceName, err)
		}

		// Set ownership
		if err := c.fs.Chown(envPath, fmt.Sprintf("%s:%s", setupUser, setupUser)); err != nil {
			return fmt.Errorf("failed to set ownership on %s: %w", envPath, err)
		}

		c.ui.Successf("Created: %s", envPath)
	}

	return nil
}

// generateEnvContent generates .env file content for a service
func (c *ContainerSetup) generateEnvContent(serviceName string) string {
	puid := c.config.GetOrDefault("ENV_PUID", "1000")
	pgid := c.config.GetOrDefault("ENV_PGID", "1000")
	tz := c.config.GetOrDefault("ENV_TZ", "America/Chicago")
	appdataPath := c.config.GetOrDefault("ENV_APPDATA_PATH", "/var/lib/containers/appdata")

	// Use cases.Title instead of deprecated strings.Title
	caser := cases.Title(language.English)

	content := fmt.Sprintf(`# UBlue uCore Homelab - %s Stack Environment
# Generated by homelab-setup

# User/Group Configuration
PUID=%s
PGID=%s
TZ=%s

# Paths
APPDATA_PATH=%s

`, caser.String(serviceName), puid, pgid, tz, appdataPath)

	// Add service-specific variables
	switch serviceName {
	case "media":
		content += fmt.Sprintf(`# Plex Configuration
PLEX_CLAIM_TOKEN=%s

# Jellyfin Configuration
JELLYFIN_PUBLIC_URL=%s

# Hardware Transcoding
# Intel QuickSync device for hardware transcoding
TRANSCODE_DEVICE=/dev/dri

`, c.config.GetOrDefault("PLEX_CLAIM_TOKEN", ""),
			c.config.GetOrDefault("JELLYFIN_PUBLIC_URL", ""))

	case "web":
		content += fmt.Sprintf(`# Overseerr Configuration
OVERSEERR_API_KEY=%s

# Web Service Ports
OVERSEERR_PORT=5055
WIZARR_PORT=5690
ORGANIZR_PORT=9983
HOMEPAGE_PORT=3000

`, c.config.GetOrDefault("OVERSEERR_API_KEY", ""))

	case "cloud":
		content += fmt.Sprintf(`# Nextcloud Configuration
NEXTCLOUD_ADMIN_USER=%s
NEXTCLOUD_ADMIN_PASSWORD=%s
NEXTCLOUD_DB_PASSWORD=%s
NEXTCLOUD_TRUSTED_DOMAINS=%s

# Collabora Configuration
COLLABORA_PASSWORD=%s
COLLABORA_DOMAIN=%s

# Immich Configuration
IMMICH_DB_PASSWORD=%s

# Database Configuration
POSTGRES_USER=%s
REDIS_PASSWORD=%s

`, c.config.GetOrDefault("NEXTCLOUD_ADMIN_USER", "admin"),
			c.config.GetOrDefault("NEXTCLOUD_ADMIN_PASSWORD", ""),
			c.config.GetOrDefault("NEXTCLOUD_DB_PASSWORD", ""),
			c.config.GetOrDefault("NEXTCLOUD_TRUSTED_DOMAINS", "localhost"),
			c.config.GetOrDefault("COLLABORA_PASSWORD", ""),
			c.config.GetOrDefault("COLLABORA_DOMAIN", "localhost"),
			c.config.GetOrDefault("IMMICH_DB_PASSWORD", ""),
			c.config.GetOrDefault("POSTGRES_USER", "homelab"),
			c.config.GetOrDefault("REDIS_PASSWORD", ""))
	}

	return content
}

// Run executes the container setup step
func (c *ContainerSetup) Run() error {
	// Check if already completed
	exists, err := c.markers.Exists("container-setup-complete")
	if err != nil {
		return fmt.Errorf("failed to check marker: %w", err)
	}
	if exists {
		c.ui.Info("Container setup already completed (marker found)")
		c.ui.Info("To re-run, remove marker: ~/.local/homelab-setup/container-setup-complete")
		return nil
	}

	c.ui.Header("Container Stack Setup")
	c.ui.Info("Configuring container services for homelab...")
	c.ui.Print("")

	// Check homelab user
	homelabUser := c.config.GetOrDefault("HOMELAB_USER", "")
	if homelabUser == "" {
		return fmt.Errorf("homelab user not configured (run user setup first)")
	}

	// Find template directory
	templateDir, err := c.FindTemplateDirectory()
	if err != nil {
		return fmt.Errorf("failed to find templates: %w", err)
	}

	// Discover available stacks
	stacks, err := c.DiscoverStacks(templateDir)
	if err != nil {
		return fmt.Errorf("failed to discover stacks: %w", err)
	}

	// Select stacks to setup
	selectedStacks, err := c.SelectStacks(stacks)
	if err != nil {
		return fmt.Errorf("failed to select stacks: %w", err)
	}

	if err := c.ensureNonInteractiveRequirements(selectedStacks); err != nil {
		return err
	}

	// Copy templates
	if err := c.CopyTemplates(templateDir, stacks, selectedStacks); err != nil {
		return fmt.Errorf("failed to copy templates: %w", err)
	}

	// Create base environment configuration
	if err := c.CreateBaseEnvConfig(); err != nil {
		return fmt.Errorf("failed to create base config: %w", err)
	}

	// Configure each selected stack
	for _, serviceName := range selectedStacks {
		if err := c.ConfigureStackEnv(serviceName); err != nil {
			c.ui.Warningf("Failed to configure %s: %v", serviceName, err)
			// Continue with other stacks
		}
	}

	// Create .env files
	if err := c.CreateEnvFiles(selectedStacks); err != nil {
		return fmt.Errorf("failed to create .env files: %w", err)
	}

	c.ui.Print("")
	c.ui.Separator()
	c.ui.Success("✓ Container stack setup completed")
	c.ui.Infof("Configured %d stack(s): %s", len(selectedStacks), strings.Join(selectedStacks, ", "))

	// Create completion marker
	if err := c.markers.Create("container-setup-complete"); err != nil {
		return fmt.Errorf("failed to create completion marker: %w", err)
	}

	return nil
}
