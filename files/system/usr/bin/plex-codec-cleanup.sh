#!/bin/bash
# /usr/bin/plex-codec-cleanup.sh
# Cleans Plex codec cache to fix transcoding bugs
# Ownership: 1001:1001 (Plex container user)

set -euo pipefail

CONTAINER_NAME="plex"
CODEC_DIR="/var/lib/containers/appdata/plex/config/Library/Application Support/Plex Media Server/Codecs"
MAX_TEMP=75000  # 75°C in millidegrees
PLEX_UID=1001
PLEX_GID=1001

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
    logger -t "plex-cleanup" "$1" 2>/dev/null || true
}

# 1. Thermal Safety Check
CURRENT_TEMP=$(cat /sys/class/thermal/thermal_zone*/temp 2>/dev/null | sort -nr | head -n1 || echo "0")
if [ "$CURRENT_TEMP" -gt "$MAX_TEMP" ]; then
    log "CRITICAL: System too hot ($((CURRENT_TEMP/1000))°C). Aborting."
    exit 1
fi
log "Starting cleanup. Temp: $((CURRENT_TEMP/1000))°C"

# 2. Stop Plex gracefully (45s timeout for transcoder cleanup)
log "Stopping Plex..."
if ! /usr/bin/podman stop -t 45 "$CONTAINER_NAME" 2>/dev/null; then
    log "WARNING: Stop command returned non-zero (container may already be stopped)"
fi

# 3. Verify container is actually stopped
sleep 2
RUNNING=$(/usr/bin/podman inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null || echo "false")
if [ "$RUNNING" == "true" ]; then
    log "CRITICAL: Plex refused to stop. Aborting to prevent corruption."
    exit 1
fi
log "Container confirmed stopped."

# 4. Cleanup and recreate with proper permissions
if [ -d "$CODEC_DIR" ]; then
    log "Removing codec directory..."
    rm -rf "$CODEC_DIR"
else
    log "Codec directory not found (first run or already cleaned)."
fi

log "Recreating codec directory with correct permissions..."
mkdir -p "$CODEC_DIR"
chown "${PLEX_UID}:${PLEX_GID}" "$CODEC_DIR"
chmod 775 "$CODEC_DIR"

# 5. Restart Plex
log "Starting Plex..."
/usr/bin/podman start "$CONTAINER_NAME"

# 6. Verify it came back up
sleep 5
RUNNING=$(/usr/bin/podman inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null || echo "false")
if [ "$RUNNING" != "true" ]; then
    log "WARNING: Plex may have failed to start. Check container logs."
    exit 1
fi

log "Maintenance complete. Codec dir recreated with ${PLEX_UID}:${PLEX_GID} / 775"
