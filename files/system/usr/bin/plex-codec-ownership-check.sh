#!/bin/bash
# Plex Codec Ownership Check
# Ensures the codec directory is owned by the correct user (PLEX_UID)
# and triggers cleanup if ownership is incorrect (including root)

set -euo pipefail

# --- Configuration ---
CODEC_DIR="/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs"
CLEANUP_SCRIPT="/usr/bin/plex-codec-cleanup.sh"
PLEX_UID=1001

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Ensure codec directory exists
if [ ! -d "$CODEC_DIR" ]; then
    log "Codec directory not found: $CODEC_DIR"
    log "Creating directory with correct ownership..."
    mkdir -p "$CODEC_DIR"
    chown "${PLEX_UID}:$(stat -c '%g' "$(dirname "$CODEC_DIR")" 2>/dev/null || echo '1001')" "$CODEC_DIR"
    exit 0
fi

owner_uid=$(stat -c "%u" "$CODEC_DIR")

if [ "$owner_uid" -ne "$PLEX_UID" ]; then
    log "Codec directory has incorrect ownership (uid=$owner_uid, expected=$PLEX_UID)."
    log "Running cleanup to fix permissions..."
    "$CLEANUP_SCRIPT"
else
    log "Codec directory ownership OK (uid=$owner_uid)."
fi
