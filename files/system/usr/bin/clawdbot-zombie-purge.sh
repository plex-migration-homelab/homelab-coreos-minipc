#!/bin/bash
# clawdbot-zombie-purge.sh
# Purges stopped/exited clawdbot sandbox containers every 6 hours
# Run via systemd timer or cron

set -euo pipefail

LOG_FILE="/var/log/clawdbot-zombie-purge.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

# Function to log with timestamp
log() {
    echo "[$TIMESTAMP] $1" | tee -a "$LOG_FILE"
}

# Get list of exited/stopped clawdbot containers
ZOMBIES=$(docker ps -aq --filter name=clawdbot-sbx --filter status=exited --filter status=dead 2>/dev/null || true)
RUNNING=$(docker ps -q --filter name=clawdbot-sbx --filter status=running 2>/dev/null | wc -l)

if [ -n "$ZOMBIES" ]; then
    COUNT=$(echo "$ZOMBIES" | wc -l)
    log "Found $COUNT zombie containers to purge"
    echo "$ZOMBIES" | xargs -r docker rm -f
    log "Purged $COUNT zombie containers"
else
    log "No zombie containers found"
fi

log "Current running containers: $RUNNING"

# Alert if running count exceeds threshold (optional)
if [ "$RUNNING" -gt 8 ]; then
    log "WARNING: $RUNNING containers running, approaching maxConcurrent limit"
fi
