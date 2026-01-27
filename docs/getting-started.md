# Getting Started (Mini PC)

Walkthrough for rebuilding the NAB9 mini PC from scratch.

## Before you start
- Generate an Ignition file from `ignition/config.bu.template`.
- Ensure you have the Fileserver (192.168.7.179) reachable on the LAN.
- Collect your WireGuard keys and service passwords.

## 1. OS Installation
1. Boot the NAB9 from a Fedora CoreOS USB.
2. Install to NVMe: `sudo coreos-installer install /dev/nvme0n1 --ignition-file config.ign`.
3. Reboot. The system will auto-rebase to the custom UBlue image.

## 2. Run the `homelab-setup` tool
SSH in as `core` and launch the wizard:
```bash
homelab-setup
```

### What the wizard configures:
1. **User Setup**: Creates the service user and configures UID/GID (65001).
2. **Directory Setup**: Scaffolds `/srv/containers` and `/var/lib/containers/appdata`.
3. **WireGuard**: Generates keys and configures the tunnel to the VPS.
4. **NFS Mounts**: Connects to the Fileserver's `Media` and `nextcloud` shares.
5. **Container Setup**: Deploys Portainer and initializes the stack repositories.

## 3. Verification
Check that everything is mounted and running:

### Daily diagnostics
A systemd timer runs `/usr/bin/llm-diagnostic.sh` daily at 07:30, writing logs to:
- `/var/log/llm-diagnostic/llm-diagnostic-YYYYMMDD.log`
- `/var/log/llm-diagnostic/minipc-diagnostic-YYYYMMDD_HHMMSS.log`

Logs older than 30 days are pruned automatically.
```bash
# Check mounts
df -h | grep /mnt/nas

# Check services
docker ps
systemctl status docker-compose-portainer.service
```

## Service URLs
- **Portainer**: `https://<ip>:9443`
- **Plex**: `http://<ip>:32400/web`
- **Jellyfin**: `http://<ip>:8096`
- **Overseerr**: `http://<ip>:5055`
