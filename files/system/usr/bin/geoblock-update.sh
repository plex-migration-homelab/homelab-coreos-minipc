#!/bin/bash
# Geoblock Update
# Drops new inbound connections from blocked countries (all ports) using
# an nftables table hooked before Docker's DNAT. Ranges from ipdeny.com.

set -euo pipefail

# --- Configuration ---
COUNTRIES="in ru"
STATE_DIR="/var/lib/geoblock"
RULESET="$STATE_DIR/geoblock.nft"
MIN_ENTRIES=1000

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

mkdir -p "$STATE_DIR"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

fetch() {
    local url=$1 out=$2
    curl -sf --max-time 60 "$url" | grep -E '^[0-9a-fA-F.:]+/[0-9]+$' >> "$out"
}

download_ok=true
for c in $COUNTRIES; do
    fetch "https://www.ipdeny.com/ipblocks/data/aggregated/${c}-aggregated.zone" "$tmp/v4" || download_ok=false
    fetch "https://www.ipdeny.com/ipv6/ipaddresses/aggregated/${c}-aggregated.zone" "$tmp/v6" || download_ok=false
done

if $download_ok && [ "$(cat "$tmp/v4" "$tmp/v6" | wc -l)" -ge "$MIN_ENTRIES" ]; then
    {
        echo "table inet geoblock"
        echo "delete table inet geoblock"
        echo "table inet geoblock {"
        echo "  set geo4 { type ipv4_addr; flags interval; auto-merge; elements = { $(paste -sd, "$tmp/v4") } }"
        echo "  set geo6 { type ipv6_addr; flags interval; auto-merge; elements = { $(paste -sd, "$tmp/v6") } }"
        echo "  chain prerouting {"
        echo "    type filter hook prerouting priority mangle; policy accept;"
        echo "    ct state new ip saddr @geo4 counter drop"
        echo "    ct state new ip6 saddr @geo6 counter drop"
        echo "  }"
        echo "}"
    } > "$tmp/geoblock.nft"
    nft -c -f "$tmp/geoblock.nft"
    install -m 600 "$tmp/geoblock.nft" "$RULESET"
    log "Downloaded $(wc -l < "$tmp/v4") IPv4 / $(wc -l < "$tmp/v6") IPv6 ranges for: $COUNTRIES"
elif [ -f "$RULESET" ]; then
    log "Download failed or incomplete; loading cached ruleset"
else
    log "Download failed and no cached ruleset exists"
    exit 1
fi

nft -f "$RULESET"
log "Geoblock active"
