# Getting Started

Quick guide for setting up the homelab CoreOS mini PC. This reflects the NAB9 mini PC personal setup using the automated Go CLI tool.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Automated Setup (Recommended)](#automated-setup-recommended)
  - [Quick Start](#quick-start)
  - [What Gets Configured](#what-gets-configured)
  - [Interactive Prompts](#interactive-prompts)
  - [Configuration Storage](#configuration-storage)
  - [Post-Setup](#post-setup)
- [Manual Setup](#manual-setup)
  - [1. User Setup](#1-user-setup)
  - [2. Directory Structure](#2-directory-structure)
  - [3. WireGuard Configuration](#3-wireguard-configuration)
  - [4. NFS Mounts Setup](#4-nfs-mounts-setup)
  - [5. Container Setup](#5-container-setup)
  - [6. Service Deployment](#6-service-deployment)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before you start, make sure you have:

- System installed with the Ignition config (see [`docs/reference/ignition.md`](reference/ignition.md))
- SSH access as the `core` user
- Network connection to your file server (if using NFS)
- WireGuard endpoint details (if using VPN)
- These packages in your image:
  `podman` or `docker`, `nfs-utils`, `wireguard-tools`

**Note:** The `homelab-setup` CLI binary is automatically baked into the CoreOS image at `/usr/bin/homelab-setup` and is available immediately after first boot.

---

## Automated Setup (Recommended)

The `homelab-setup` Go CLI provides an interactive wizard that handles the full setup: users, directories, WireGuard, NFS, and containers. This binary is pre-installed in your custom CoreOS image.

### Quick Start

```bash
# The binary is in your PATH
homelab-setup
```

Menu options:

```
[A] Run All Steps (Complete Setup)
[Q] Quick Setup (Skip WireGuard)

Individual Steps:
[0] Pre-flight Check
[1] User Setup
[2] Directory Setup
[3] WireGuard Setup
[4] NFS Setup
[5] Container Setup
[6] Service Deployment

[T] Troubleshooting Tool
[S] Show Setup Status
[P] Add WireGuard Peer (post-setup)
```

For a first-time setup:

* `[A]` → full setup (includes WireGuard)
* `[Q]` → skip WireGuard

### What Gets Configured

1. **Container Runtime**
   * Choose Docker or Podman
   * Auto-detect installed runtimes
   * Uses correct compose commands

2. **User Account**
   * Dedicated user for containers (recommended)
   * Groups, subuid/subgid for rootless containers
   * Uses that user's UID/GID for container ownership
   * You keep using `core` to run the CLI

3. **Directory Structure**
   * `/srv/containers/{media,web,cloud}/` for compose files
   * `/var/lib/containers/appdata/` for persistent data
   * `/mnt/nas-*` for NFS mounts

4. **WireGuard VPN** (optional)
   * Generates server + peer keys
   * Auto-detects WAN interface
   * Builds config from templates
   * Exports peer configs with QR codes (install `qrencode` for QR support)

5. **NFS Mounts**
   * Uses systemd mount units
   * Tests server connectivity
   * Enables mounts

6. **Container Services**
   * Copies compose templates
   * Creates `.env` from your inputs
   * Applies passwords, tokens, and ownership

7. **Service Deployment**
   * Enables systemd services
   * Pulls images
   * Starts services and checks basic health

### Interactive Prompts

Examples of what you'll be asked:

**Container runtime:**

```
Multiple container runtimes detected:
  1. Podman (rootless, recommended for UBlue uCore)
  2. Docker

Select container runtime [1]:
```

**User configuration:**

```
SECURITY BEST PRACTICE:
Create a dedicated user for container management separate from your admin user.

Options:
  1. Create a new dedicated user (RECOMMENDED)
  2. Use current user (core) - not recommended for production

Choose option [1]: 1
Enter new username for container management [containeruser]: myhomelabuser
```

**NFS server:**

```
NFS server IP address [192.168.7.10]: 192.168.1.50
```

**Service passwords:**

```
Nextcloud admin password: ********
Nextcloud database password: ********
Immich database password: ********
```

### Configuration Storage

Settings are saved to `~/.homelab-setup.conf`:

```bash
CONTAINER_RUNTIME=podman
HOMELAB_USER=myhomelabuser
PUID=1001
PGID=1001
TZ=America/Chicago
APPDATA_PATH=/var/lib/containers/appdata
NFS_SERVER=192.168.1.50
```

Completion markers are stored in `~/.local/homelab-setup/`:

```
preflight-complete
user-setup-complete
directory-setup-complete
wireguard-setup-complete
nfs-setup-complete
container-setup-complete
service-deployment-complete
```

### Post-Setup

After automated setup:

1. Access services at the URLs printed by the CLI
2. Configure each service through its web UI
3. Use the troubleshooting helper if needed:
   ```bash
   homelab-setup troubleshoot
   ```

### Adding WireGuard Peers

After initial setup, add additional peers without re-running the entire wizard:

```bash
homelab-setup wireguard add-peer
```

Or from the main menu: `[P] Add WireGuard Peer`

This workflow:
- Reads `/etc/wireguard/wg0.conf`, finds the next free IP, and appends a `[Peer]` block
- Generates a full client config with private key, address, DNS, server details
- Saves to `~/setup/export/wireguard-peers/<peer>.conf` with `0600` permissions
- Prints the config and renders an ASCII QR code (if `qrencode` is installed)
- Offers to restart `wg-quick@wg0`

---

## Manual Setup

Manual setup is useful if you want to customize behavior or troubleshoot specific parts. The automated CLI performs all of these steps, so only follow this section if you need full control.

---

### 1. User Setup

Recommended: a dedicated non-admin user for containers.

> **Automated setup:** The CLI lets you choose current user or a new user and configures everything, including subuid/subgid.

#### Create a Container Management User

Example usernames: `containeruser`, `homelabuser`, `dockeruser`

```bash
# Replace USERNAME with your chosen user
sudo useradd -m -s /bin/bash USERNAME

sudo usermod -aG wheel USERNAME      # sudo access
sudo usermod -aG podman USERNAME     # if using Podman
sudo usermod -aG docker USERNAME     # if using Docker

sudo passwd USERNAME

# Rootless container support
echo "USERNAME:100000:65536" | sudo tee -a /etc/subuid
echo "USERNAME:100000:65536" | sudo tee -a /etc/subgid
```

Switch to that user when needed:

```bash
sudo su - USERNAME
```

---

### 2. Directory Structure

Create a standard layout for compose files and app data.

> **Automated setup:** Creates `/srv/containers/{media,web,cloud}/` and `/var/lib/containers/appdata/` with proper ownership.

#### Compose Directories

```bash
# Recommended
sudo mkdir -p /srv/containers/media
sudo mkdir -p /srv/containers/web
sudo mkdir -p /srv/containers/cloud
sudo chown -R USERNAME:USERNAME /srv/containers

# Alternative: in home
mkdir -p ~/compose/{media,web,cloud}
```

#### Application Data

```bash
# Recommended
sudo mkdir -p /var/lib/containers/appdata
sudo mkdir -p /var/lib/containers/appdata/{plex,jellyfin,tautulli,overseerr,wizarr,organizr,homepage,nextcloud,nextcloud-db,nextcloud-redis,collabora,immich,immich-db,immich-redis,immich-ml}
sudo chown -R USERNAME:USERNAME /var/lib/containers/appdata

# Alternative: in home
mkdir -p ~/appdata/{plex,jellyfin,overseerr,wizarr,nextcloud,immich,postgres,redis}
```

#### Recommended Layout

```
/srv/containers/
├── media/
│   ├── compose.yml
│   └── .env
├── web/
│   ├── compose.yml
│   └── .env
└── cloud/
    ├── compose.yml
    └── .env

/var/lib/containers/appdata/
├── plex/
├── jellyfin/
├── overseerr/
├── wizarr/
├── nextcloud/
├── immich/
├── postgres/
└── redis/
```

---

### 3. WireGuard Configuration

WireGuard provides secure remote access and VPS tunneling.

> **Automated setup:** The CLI generates keys, detects your WAN interface, builds config from templates, and exports peer configs with QR codes.

#### Network Details

* Server IP: `10.253.0.1/24`
* Port: `51820`
* Range: `10.253.0.0/24`
* Default peers:
  * Desktop: `10.253.0.6/32`
  * VPS: `10.253.0.8/32`
  * iPhone: `10.253.0.9/32`
  * Laptop: `10.253.0.11/32`

#### Manual Configuration

If not using the automated CLI:

```bash
# Generate server keys
wg genkey | sudo tee /etc/wireguard/server_private.key
sudo cat /etc/wireguard/server_private.key | wg pubkey | sudo tee /etc/wireguard/server_public.key

# Generate peer keys
wg genkey | tee peer_private.key | wg pubkey > peer_public.key

# Create config
sudo nano /etc/wireguard/wg0.conf
```

Example `wg0.conf`:

```ini
[Interface]
Address = 10.253.0.1/24
ListenPort = 51820
PrivateKey = <server-private-key>

# Desktop peer
[Peer]
PublicKey = <desktop-public-key>
AllowedIPs = 10.253.0.6/32

# VPS peer
[Peer]
PublicKey = <vps-public-key>
AllowedIPs = 10.253.0.8/32
PersistentKeepalive = 25
```

Deploy:

```bash
sudo chmod 600 /etc/wireguard/wg0.conf
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0
sudo wg show
```

---

### 4. NFS Mounts Setup

Mount NFS shares so containers can access NAS data.

> **Automated setup:** The CLI detects pre-existing systemd mount units, tests NFS server connectivity, and enables mounts.

#### Mount Points

```bash
sudo mkdir -p /mnt/nas-media
sudo mkdir -p /mnt/nas-nextcloud
sudo mkdir -p /mnt/nas-immich
sudo mkdir -p /mnt/nas-photos
```

#### Systemd Mount Units

Systemd mount files live in `/etc/systemd/system/`. Adjust the NFS server IP as needed.

Example: `/etc/systemd/system/mnt-nas-media.mount`

```ini
[Unit]
Description=NFS mount for media storage (Plex/Jellyfin)
After=network-online.target
Wants=network-online.target
Before=docker.service

[Mount]
What=192.168.7.10:/mnt/storage/Media
Where=/mnt/nas-media
Type=nfs
Options=ro,hard,intr,rsize=131072,wsize=131072,tcp,timeo=600,retrans=2,_netdev

TimeoutSec=60

[Install]
WantedBy=multi-user.target
WantedBy=remote-fs.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable mnt-nas-media.mount
sudo systemctl enable mnt-nas-nextcloud.mount
sudo systemctl enable mnt-nas-immich.mount

sudo systemctl start mnt-nas-media.mount
sudo systemctl start mnt-nas-nextcloud.mount
sudo systemctl start mnt-nas-immich.mount

df -h | grep nas
mount | grep nfs
```

#### `/etc/fstab` Alternative

```bash
sudo nano /etc/fstab
```

Example entries:

```bash
192.168.7.10:/mnt/storage/Media      /mnt/nas-media      nfs  defaults,ro,_netdev  0 0
192.168.7.10:/mnt/storage/Nextcloud  /mnt/nas-nextcloud  nfs  defaults,rw,_netdev  0 0
192.168.7.10:/mnt/storage/Photos     /mnt/nas-photos     nfs  defaults,ro,_netdev  0 0
```

Apply:

```bash
sudo mount -a
```

Key options:

* `_netdev`: delay mount until network is up
* `ro` / `rw`: read-only / read-write
* `hard`, `intr`, `rsize`, `wsize`, `tcp`, `timeo`, `retrans`: performance and reliability tuning

---

### 5. Container Setup

You can use Podman or Docker. Podman is recommended for uCore.

> **Automated setup:** The CLI detects your runtime, sets the correct compose command, and populates `.env` files.

#### Runtime Choice

**Podman (recommended on uCore):**

* Daemonless
* Rootless containers
* Good systemd integration
* Pre-installed on uCore

**Docker:**

* Well-known ecosystem
* Requires daemon
* Must be layered onto uCore

#### Environment Variables

Configure in `/srv/containers/.env`:

* `PUID`, `PGID` → from `id` for your container user
* `TZ` → timezone, e.g., `America/Chicago`
* `APPDATA_PATH` → usually `/var/lib/containers/appdata`
* Service-specific secrets:
  * `PLEX_CLAIM_TOKEN`
  * `NEXTCLOUD_DB_PASSWORD`
  * `NEXTCLOUD_TRUSTED_DOMAINS`
  * `IMMICH_DB_PASSWORD`
  * `COLLABORA_PASSWORD`

#### Start Services (Podman Compose)

```bash
cd /srv/containers

podman-compose -f media.yml up -d
podman-compose -f web.yml up -d
podman-compose -f cloud.yml up -d

podman-compose -f media.yml logs -f
```

#### Systemd Services (Podman)

Example for media stack:

```bash
sudo nano /etc/systemd/system/podman-compose-media.service
```

```ini
[Unit]
Description=Podman Compose - Media Stack
Requires=network-online.target
After=network-online.target mnt-nas-media.mount

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/srv/containers
ExecStart=/usr/bin/podman-compose -f media.yml up -d
ExecStop=/usr/bin/podman-compose -f media.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable podman-compose-media podman-compose-web podman-compose-cloud
sudo systemctl start podman-compose-media podman-compose-web podman-compose-cloud
```

---

### 6. Service Deployment

#### Deploy

```bash
cd /srv/containers

# Podman
podman-compose -f media.yml up -d
podman-compose -f web.yml up -d
podman-compose -f cloud.yml up -d

# Or via systemd
sudo systemctl start podman-compose-media podman-compose-web podman-compose-cloud
```

#### Verify

```bash
podman ps   # or docker ps

curl http://localhost:8096   # Jellyfin
curl http://localhost:32400  # Plex (host mode)
curl http://localhost:5055   # Overseerr
curl http://localhost:8080   # Nextcloud
curl http://localhost:2283   # Immich
```

#### Access URLs

Media (direct):

* Plex: `http://your-ip:32400/web`
* Jellyfin: `http://your-ip:8096`

Web (typically via VPS proxy):

* Overseerr: `http://your-ip:5055`
* Wizarr: `http://your-ip:5690`
* Organizr: `http://your-ip:9983`
* Homepage: `http://your-ip:3000`

Cloud (typically via VPS proxy):

* Nextcloud: `http://your-ip:8080`
* Collabora: `http://your-ip:9980`
* Immich: `http://your-ip:2283`

---

## Troubleshooting

Use the built-in troubleshooting tool:

```bash
homelab-setup troubleshoot
```

Or check the status:

```bash
homelab-setup status
```

### NFS Issues

```bash
ping 192.168.7.10
sudo mount -t nfs 192.168.7.10:/mnt/storage/Media /mnt/nas-media
systemctl status mnt-nas-media.mount
journalctl -u mnt-nas-media.mount -f
```

### WireGuard Issues

```bash
sudo wg show
journalctl -u wg-quick@wg0 -f
sudo systemctl restart wg-quick@wg0
ping 10.253.0.8  # VPS
```

### Container Issues

```bash
podman logs <container-name>
docker logs <container-name>

podman restart <container-name>
docker restart <container-name>

cd /srv/containers
podman-compose -f media.yml down && podman-compose -f media.yml up -d
```

### Permission Issues

```bash
sudo chown -R 1000:1000 /var/lib/containers/appdata
ls -Z /var/lib/containers/appdata

sudo semanage fcontext -a -t container_file_t "/var/lib/containers/appdata(/.*)?"
sudo restorecon -Rv /var/lib/containers/appdata
```

### System Updates

```bash
sudo rpm-ostree rollback
sudo systemctl reboot

sudo rpm-ostree status
sudo ostree admin pin 0
```

---

## Next Steps

1. Configure each service via web UI
2. Set up reverse proxy (e.g., Nginx Proxy Manager on VPS over WireGuard)
3. Configure SSL certificates
4. Set up backups for `/var/lib/containers/appdata`
5. Optional: add monitoring and Fail2ban

See the main `README.md` for more detail on architecture and design choices.

---

## CLI Reference

For complete CLI documentation, see [`docs/reference/homelab-setup-cli.md`](reference/homelab-setup-cli.md).

Common commands:

```bash
homelab-setup                 # Interactive menu
homelab-setup status          # Show setup status
homelab-setup troubleshoot    # Run diagnostics
homelab-setup reset           # Reset setup (backs up config)
homelab-setup wireguard add-peer  # Add WireGuard peer
```
