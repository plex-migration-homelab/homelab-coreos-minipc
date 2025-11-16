package config

// Configuration key constants to prevent typos and enable autocomplete
const (
	// User configuration
	KeyHomelabUser     = "HOMELAB_USER"
	KeyHomelabUID      = "HOMELAB_UID"
	KeyHomelabGID      = "HOMELAB_GID"
	KeyHomelabTimezone = "HOMELAB_TIMEZONE"

	// Directory configuration
	KeyHomelabBaseDir = "HOMELAB_BASE_DIR"
	KeyContainersBase = "CONTAINERS_BASE" // Legacy, prefer HOMELAB_BASE_DIR

	// NFS configuration
	KeyNFSServer     = "NFS_SERVER"
	KeyNFSExport     = "NFS_EXPORT"
	KeyNFSMountPoint = "NFS_MOUNT_POINT"
	KeyNFSFstabPath  = "NFS_FSTAB_PATH"

	// WireGuard configuration
	KeyWGInterface   = "WG_INTERFACE"
	KeyWGInterfaceIP = "WG_INTERFACE_IP"
	KeyWGListenPort  = "WG_LISTEN_PORT"
	KeyWGConfigPath  = "WG_CONFIG_PATH"

	// Container configuration
	KeyContainerRuntime   = "CONTAINER_RUNTIME"
	KeySelectedServices   = "SELECTED_SERVICES"
	KeyComposeProjectName = "COMPOSE_PROJECT_NAME"

	// Network configuration
	KeyNetworkTestRetries = "NETWORK_TEST_RETRIES"
	KeyNetworkTestTimeout = "NETWORK_TEST_TIMEOUT"

	// System configuration
	KeyConfigVersion = "CONFIG_VERSION"
)

// Default values for configuration keys
var Defaults = map[string]string{
	KeyHomelabBaseDir:     "/srv/containers",
	KeyContainerRuntime:   "podman",
	KeyNFSMountPoint:      "/mnt/nas",
	KeyNFSFstabPath:       "/etc/fstab",
	KeyNetworkTestRetries: "5",
	KeyNetworkTestTimeout: "10",
	KeyConfigVersion:      "1",
	KeyWGInterface:        "wg0",
	KeyWGListenPort:       "51820",
}
