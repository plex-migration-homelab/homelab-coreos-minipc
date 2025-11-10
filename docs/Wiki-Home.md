# Homelab CoreOS Mini PC Wiki

Welcome to the documentation wiki for the homelab CoreOS mini PC setup!

## Overview

This wiki provides detailed documentation for setting up and maintaining a declarative **uCore (Ublue CoreOS)** frontend application node that runs all user-facing services on a NAB9 mini PC.

## Documentation

### Getting Started

- **[Installation Guide](https://github.com/zoro11031/homelab-coreos-minipc?tab=readme-ov-file#ignition-setup-first-time-installation)** - Initial system installation with Ignition
- **[Setup Guide](https://github.com/zoro11031/homelab-coreos-minipc/wiki/Setup)** - Post-installation configuration and service deployment

## Quick Links

### Common Tasks

1. **[Setting up WireGuard](https://github.com/zoro11031/homelab-coreos-minipc/wiki/Setup#3-wireguard-configuration)** - Configure VPN connectivity
2. **[Configuring NFS Mounts](https://github.com/zoro11031/homelab-coreos-minipc/wiki/Setup#4-nfs-mounts-setup)** - Set up network storage
3. **[Deploying Containers](https://github.com/zoro11031/homelab-coreos-minipc/wiki/Setup#5-container-setup)** - Run your services
4. **[Troubleshooting](https://github.com/zoro11031/homelab-coreos-minipc/wiki/Testing)** - Fix common issues

### System Information

- **Base Image**: [uCore](https://github.com/ublue-os/ucore) (Ublue CoreOS)
- **Custom Image**: `ghcr.io/zoro11031/homelab-coreos-minipc:latest`
- **Build System**: [BlueBuild](https://blue-build.org/)

### Architecture

```
Internet → VPS (Nginx Proxy Manager) → WireGuard Tunnel → NAB9 Mini PC
                                                               ↓
                                                         Podman Stack
                                                    (Plex, Jellyfin, etc.)
                                                               ↓
                                                          NFS Mounts
                                                               ↓
                                                         File Server
```

## Services

This system runs the following services:

### Media Services
- **Plex** - Media server with hardware transcoding
- **Jellyfin** - Open-source media server

### Web Services
- **Overseerr** - Media request management
- **Wizarr** - Automated invitation system

### Cloud Services
- **Nextcloud** - Personal cloud and groupware
- **Immich** - Photo and video backup platform

## Contributing

This is a personal homelab setup, but contributions and suggestions are welcome! Please open an issue or pull request on the [GitHub repository](https://github.com/zoro11031/homelab-coreos-minipc).

## Support

For issues or questions:
- Check the [Troubleshooting](Setup.md#troubleshooting) section
- Review existing [GitHub Issues](https://github.com/zoro11031/homelab-coreos-minipc/issues)
- Open a new issue if you find a bug or have a suggestion

## License

See [LICENSE](../LICENSE) for details.