#!/bin/bash
# Plex Codec Cleanup & Permission Fix Script
# Function: Stops Plex, cleans codecs, fixes ALL permissions, restarts Plex.

set -euo pipefail

# --- Configuration ---
CONTAINER_NAME="plex"
# The root folder for all Plex data
PLEX_ROOT_DIR="/var/lib/containers/appdata/plex"
# The specific Codec folder to delete
CODEC_DIR="${PLEX_ROOT_DIR}/config/Library/Application Support/Plex Media Server/Codecs"

PLEX_UID=1001
PLEX_GID=1001

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 1. Stop Plex
log "Stopping Plex container..."
docker stop "$CONTAINER_NAME" || true

# 2. Delete Codec Folder
if [ -d "$CODEC_DIR" ]; then
    log "Removing codec directory: $CODEC_DIR"
    rm -rf "$CODEC_DIR"
fi

# 3. Fix Permissions (Entire Directory)
# This may take time depending on library size
log "Setting ownership of ${PLEX_ROOT_DIR} to ${PLEX_UID}:${PLEX_GID}..."
find "$PLEX_ROOT_DIR" \( ! -user "$PLEX_UID" -o ! -group "$PLEX_GID" \) -exec chown "${PLEX_UID}:${PLEX_GID}" {} +

log "Setting owner/group read-write on ${PLEX_ROOT_DIR}..."
chmod -R u+rwX,g+rwX "$PLEX_ROOT_DIR"

# 4. Start Plex
log "Starting Plex container..."
docker start "$CONTAINER_NAME"

log "Maintenance complete."
