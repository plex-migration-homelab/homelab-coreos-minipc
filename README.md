# Homelab CoreOS Mini PC &nbsp; [![build](https://github.com/plex-migration-homelab/homelab-coreos-minipc/actions/workflows/build.yml/badge.svg)](https://github.com/plex-migration-homelab/homelab-coreos-minipc/actions/workflows/build.yml)

Declarative image + helper tooling for the NAB9 mini PC. It rebases Fedora CoreOS into a custom UBlue uCore build, tunnels traffic through WireGuard to a VPS, and mounts media from the backend Fileserver over SMB.

## Scope & Assumptions
- Single-node helper for the user-facing side of the homelab.
- Focuses on the interactive `homelab-setup` Go helper for post-install configuration.
- Integrates with the backend Fileserver (Debian 13) for bulk storage.

## What's Running
- **Media**: Plex and Jellyfin with Intel QuickSync for hardware transcodes.
- **Portals**: Jellyseerr and Wizarr.
- **Cloud**: Nextcloud (AIO stack).
- **Management**: Portainer (via systemd) managing GitOps-based Docker stacks.
- **Platform**: Docker, WireGuard, VAAPI drivers for GPU acceleration.

## Getting Started
1. **Install the image**: Build an Ignition file from `ignition/config.bu.template` and install Fedora CoreOS. The first boot rebases into `ghcr.io/plex-migration-homelab/homelab-coreos-minipc`.
2. **Run the helper**: SSH in as `core` and launch `homelab-setup`. The wizard walks through user creation, WireGuard, SMB mounts, and service deployment.
3. **Expose services**: Plex/Jellyfin use direct port forwards. Other services route through the VPS Caddy reverse proxy.

## Documentation
- [In-Depth Documentation Hub](/home/justin/Documents/plex/documentation/README.md): The main source of truth for the entire infrastructure.
- [`docs/getting-started.md`](docs/getting-started.md): Walkthrough for the image install and setup tool.
