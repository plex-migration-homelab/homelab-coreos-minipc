#!/bin/bash
# Plex Codec Cleanup Script
# Function: Stops Plex, nukes codecs, resets permissions, restarts Plex.

set -euo pipefail

# --- Configuration ---
CONTAINER_NAME="plex"
# Exact path based on your logs
CODEC_DIR="/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs"
PLEX_UID=1001
PLEX_GID=1001

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 1. Stop Plex
log "Stopping Plex container..."
# Using '|| true' ensures the script continues even if the container is already stopped
docker stop "$CONTAINER_NAME" || true

# 2. Delete Codec Folder
if [ -d "$CODEC_DIR" ]; then
    log "Removing codec directory..."
    rm -rf "$CODEC_DIR"
fi

# 3. Recreate Directory
# Necessary to exist before applying permissions
log "Recreating codec directory..."
mkdir -p "$CODEC_DIR"

# 4. Set Ownership (1001:1001)
log "Setting ownership to ${PLEX_UID}:${PLEX_GID}..."
chown -R "${PLEX_UID}:${PLEX_GID}" "$CODEC_DIR"

# 5. Set Permissions (775)
log "Setting permissions to 775..."
chmod -R 775 "$CODEC_DIR"

# 6. Start Plex
log "Starting Plex container..."
docker start "$CONTAINER_NAME"

log "Maintenance complete."
