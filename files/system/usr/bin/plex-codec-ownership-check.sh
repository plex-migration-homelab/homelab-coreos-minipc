#!/bin/bash
# Plex Codec Ownership Check

set -euo pipefail

CODEC_DIR="/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs"
CLEANUP_SCRIPT="/usr/bin/plex-codec-cleanup.sh"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

if [ ! -d "$CODEC_DIR" ]; then
    log "Codec directory not found: $CODEC_DIR"
    exit 0
fi

owner_uid=$(stat -c "%u" "$CODEC_DIR")

if [ "$owner_uid" -eq 0 ]; then
    log "Codec directory owned by root; running cleanup."
    "$CLEANUP_SCRIPT"
else
    log "Codec directory ownership OK (uid=$owner_uid)."
fi
