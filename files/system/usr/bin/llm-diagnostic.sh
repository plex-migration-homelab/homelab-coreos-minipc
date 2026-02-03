#!/bin/bash
# Homelab LLM Diagnostic Script v20 (SRE Refactored)
# Usage: sudo /usr/bin/llm-diagnostic.sh
# Updates: Exit Codes, OOM Detection, FCOS Staged Updates, Previous Boot Logs, Dynamic Config

set -euo pipefail

# --- CONFIGURATION ---
NAS_IP="192.168.7.179"
WG_INT="wg0"
NVME_DEV="/dev/nvme0n1"
LOG_DIR="/var/log/llm-diagnostic"
LOG_DATE="$(date +%Y%m%d)"
LOG_TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
LOG_FILE_DAILY="${LOG_DIR}/llm-diagnostic-${LOG_DATE}.log"
LOG_FILE_FULL="${LOG_DIR}/minipc-diagnostic-${LOG_TIMESTAMP}.log"
# ---------------------

# 1. ROOT CHECK
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (sudo)."
  exit 1
fi

# 2. SUPPRESS TERMINAL ESCAPE SEQUENCES
export TERM=dumb
unset PROMPT_COMMAND

# 3. SETUP LOGGING
mkdir -p "$LOG_DIR"
exec > >(tee "$LOG_FILE_DAILY" "$LOG_FILE_FULL") 2>&1

# Code block delimiter for markdown
CB='```'

# 4. HELPER FUNCTIONS
run_safe() {
    local cmd="$1"
    local duration="${2:-5s}"
    if ! timeout "$duration" sh -c "$cmd"; then
        echo "Command timed out or failed: $cmd"
    fi
}

# 5. GLOBAL LLM INSTRUCTIONS
cat << 'INSTRUCTIONS'
================================================================================
>>> LLM GLOBAL INSTRUCTIONS <<<
================================================================================
ROLE: You are a Senior Site Reliability Engineer (SRE) diagnosing a Fedora CoreOS
      Homelab node.

SYSTEM CONTEXT:
- OS: Fedora CoreOS (Immutable, UBlue image)
- Hardware: Mini PC (Intel i9-12900HK, 32GB RAM, 1TB NVMe)
- Role: Docker Host (Portainer), WireGuard Client
- Storage Strategy:
  1. Local: NVMe (OS + Docker Appdata)
  2. Remote (Unraid): NFS (Read-Only Media)
  3. Remote (Unraid): SMB (Read-Write Nextcloud)

ANALYSIS GOALS:
1. **BOOT HEALTH:** If "Previous Boot Logs" are present, prioritize analyzing them for crash causes (e.g., kernel panics, shim errors).
2. **DOCKER HYGIENE:** Distinguish between "Restarting" loops (Config errors) and "OOMKilled" (Resource Exhaustion).
3. **STORAGE BLOAT:** Contextualize Appdata size. Do NOT flag <200GB metadata as bloat unless disk >80% full.
4. **NOISE FILTER:** Ignore the following common FCOS/Docker logs unless critical:
   - "overlayfs: missing 'lowerdir'"
   - "libusb_init"
   - "x-systemd.idle-timeout" (This is a feature, not a bug, unless the mount fails to wake).

================================================================================
START OF DIAGNOSTICS
================================================================================
INSTRUCTIONS

echo ""
echo ">>> LLM PROMPT: Analyze System Overview. Check load average vs core count (14 cores)."
echo "## SECTION 1: SYSTEM OVERVIEW"
echo "### Basic System Info"
echo "$CB"
echo "Hostname: $(hostname)"
echo "Kernel: $(uname -r)"
if [ -f /etc/os-release ]; then
    echo "OS: $(grep PRETTY_NAME /etc/os-release | cut -d= -f2 | tr -d '"')"
fi
echo "Current Time: $(date)"
echo "Uptime: $(uptime -p)"
echo "Load Average: $(cat /proc/loadavg)"
echo "$CB"
echo ""

echo "### FCOS Update & Boot Status"
echo "$CB"
run_safe "rpm-ostree status -v" "10s"
echo "---"
# Check if update is staged (pending reboot)
if rpm-ostree status | grep -q "Staged: yes"; then
    echo "NOTICE: Update is STAGED. Reboot required to apply."
else
    echo "No pending updates staged."
fi

echo "---"
# Previous Boot Analysis (Critical for crash diagnostics)
uptime_sec=$(cat /proc/uptime | awk '{print int($1)}')
if [ "$uptime_sec" -lt 600 ]; then
    echo "SYSTEM RECENTLY REBOOTED (<10m). DUMPING PREVIOUS BOOT ERRORS:"
    journalctl -b -1 -p err --no-pager | tail -20 || echo "Unable to read previous boot journal."
else
    echo "System uptime > 10m. Skipping previous boot log dump."
fi
echo "$CB"
echo ""

echo ">>> LLM PROMPT: Verify mixed protocol connectivity to NAS ($NAS_IP)."
echo "## SECTION 2: VPN & CONNECTIVITY"
echo "### WireGuard Status"
echo "$CB"
if command -v wg &> /dev/null; then
    wg show "$WG_INT"
    echo "---"

    # Handshake Check
    latest_handshake=$(wg show "$WG_INT" dump | awk '{print $5}')
    current_time=$(date +%s)

    if [ -n "$latest_handshake" ] && [ "$latest_handshake" -ne 0 ]; then
        diff=$((current_time - latest_handshake))
        if [ "$diff" -gt 180 ]; then
            echo "CRITICAL WARNING: Stale WireGuard Handshake! ($diff seconds ago)"
        else
            echo "Handshake Healthy: ${diff}s ago"
        fi
    else
        echo "WARNING: No handshake data found."
    fi

    echo "---"
    # Data Flow Check (Tunnel Ping)
    # Tries to ping Cloudflare DNS forcing traffic through the WG interface
    if ping -c 1 -W 2 -I "$WG_INT" 1.1.1.1 > /dev/null 2>&1; then
        echo "Tunnel Data Flow: CONFIRMED (Ping success via $WG_INT)"
    else
        echo "Tunnel Data Flow: FAILED (Cannot ping via $WG_INT)"
    fi
else
    echo "WireGuard tools not installed"
fi
echo "$CB"
echo ""

echo "### NAS Reachability ($NAS_IP)"
echo "$CB"
if ping -c 3 -W 2 "$NAS_IP" > /dev/null; then
     echo "PING: $NAS_IP is reachable."
else
     echo "PING: Cannot reach $NAS_IP (Unraid NAS)."
fi

# Port Checks
if timeout 2 bash -c "</dev/tcp/$NAS_IP/2049" &>/dev/null; then
    echo "NFS (TCP 2049): Reachable"
else
    echo "NFS (TCP 2049): UNREACHABLE"
fi

if timeout 2 bash -c "</dev/tcp/$NAS_IP/445" &>/dev/null; then
    echo "SMB (TCP 445): Reachable"
else
    echo "SMB (TCP 445): UNREACHABLE"
fi
echo "$CB"
echo ""

echo "## SECTION 3: CONTAINER RUNTIME"
echo "### Docker Service Status"
echo "$CB"
run_safe "systemctl status docker.service --no-pager | head -15"
echo "$CB"
echo ""

echo "## SECTION 4: RUNNING CONTAINERS"
echo "### All Containers"
echo "$CB"
run_safe "docker ps -a --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'"
echo "$CB"
echo ""

echo "### Container Health & Exit Codes"
echo "$CB"
# 1. Restart Loops
run_safe "docker ps -a --format '{{.Names}}\t{{.Status}}' | grep -i restarting || echo 'No containers in restart loop'"
echo "---"
# 2. Abnormal Exits (OOM Detection)
echo "Checking for OOMKilled or Error Exits:"
docker ps -a --filter "status=exited" --format "{{.Names}}" | while read -r name; do
    code=$(docker inspect "$name" --format '{{.State.ExitCode}}')
    oom=$(docker inspect "$name" --format '{{.State.OOMKilled}}')

    if [ "$code" -ne 0 ]; then
        if [ "$oom" == "true" ]; then
             echo "CRITICAL: $name was OOM Killed (Exit Code: 137). RAM Exhaustion."
        else
             echo "WARNING: $name exited with error code: $code"
        fi
    fi
done || echo "No abnormal exits found."
echo "$CB"
echo ""

echo "## SECTION 5: CONTAINER LOGS (Active Errors - Last 2 Hours)"
echo ""
for container in $(docker ps --format '{{.Names}}' 2>/dev/null); do
    echo "### ${container} Logs (Last 2h)"
    echo "$CB"
    run_safe "docker logs --since 2h $container 2>&1 | grep -iE '(error|fail|fatal|exception|critical)' | grep -v 'libusb_init' | tail -10 || echo 'No active errors in last 2h'"
    echo "$CB"
    echo ""
done

echo ">>> LLM PROMPT: CRITICAL STORAGE ANALYSIS. 1) Check NVMe health. 2) Compare Appdata size to Free Space. 3) Analyze Inodes."
echo "## SECTION 6: STORAGE AND MOUNTS"
echo ""

echo "### Local NVMe Health Check ($NVME_DEV)"
echo "$CB"
if command -v smartctl &> /dev/null; then
    run_safe "smartctl -H $NVME_DEV"
elif command -v nvme &> /dev/null; then
    run_safe "nvme smart-log $NVME_DEV | head -10"
else
    dmesg | grep -i "nvme" | grep -iE "(error|fail|critical|warn)" | tail -10 || echo "No critical NVMe errors found in dmesg."
fi
echo "$CB"
echo ""

echo "### Local Storage Usage (Capacity & Inodes)"
echo "$CB"
# Capacity
df -h / /var | grep -v "Filesystem"
echo "---"
# Inodes (Crucial for Docker)
echo "Inode Usage:"
df -hi / /var | grep -v "Filesystem"
echo "---"
echo "TOP 10 CONSUMERS IN /var/lib/containers/appdata:"
if [ -d "/var/lib/containers/appdata" ]; then
    timeout 15s du -h --max-depth=1 /var/lib/containers/appdata 2>/dev/null | sort -hr | head -10
else
    echo "CRITICAL: /var/lib/containers/appdata does not exist."
fi
echo "$CB"
echo ""

echo "### Mount Configuration & Status"
echo "$CB"
grep -E "(nfs|cifs)" /etc/fstab | grep -v "^#"
echo "---"
systemctl list-units --type=mount --type=automount --all | grep -E '(nas|mnt)'
echo "$CB"
echo ""

echo "### Mount Point Accessibility Check"
echo "$CB"
echo "Checking /mnt/nas-media (NFS)..."
if timeout 3s ls -la "/mnt/nas-media" > /dev/null 2>&1; then
    count=$(ls -1 "/mnt/nas-media" 2>/dev/null | wc -l)
    echo "  OK: NFS Accessible ($count items)"
    echo "  NOTE: /mnt/nas-media is expected RO on Mini PC"
else
    echo "  ERROR: NFS Mount is Stale or Unresponsive"
fi

echo "Checking /var/mnt/nas-nextcloud (SMB)..."
if timeout 5s ls -la "/var/mnt/nas-nextcloud" > /dev/null 2>&1; then
    count=$(ls -1 "/var/mnt/nas-nextcloud" 2>/dev/null | wc -l)
    echo "  OK: SMB Accessible ($count items)"
else
    echo "  ERROR: SMB Mount is Unresponsive or Permissions Denied"
fi
echo "$CB"
echo ""

echo "## SECTION 7: SYSTEM RESOURCES"
echo "### Memory & ZRAM Usage"
echo "$CB"
run_safe "free -h"
echo "---"
if command -v zramctl &> /dev/null; then
    zramctl
else
    echo "zramctl not installed"
fi
echo "$CB"
echo ""

echo "### Top Processes by Memory"
echo "$CB"
run_safe "ps aux --sort=-%mem | head -10"
echo "$CB"
echo ""

echo "### Time Synchronization (NTP)"
echo "$CB"
if command -v timedatectl &> /dev/null; then
    timedatectl status | grep -E "NTP service:|System clock synchronized:|RTC in local TZ:"
    echo "---"
    if command -v chronyd &> /dev/null; then
        chronyd -Q 'server pool.ntp.org iburst' 2>/dev/null | tail -1 || echo "Could not query external NTP pool."
    fi
fi
echo "$CB"
echo ""

echo "### Transcoding Activity Check"
echo "$CB"
ps aux | grep -iE "(Plex Transcoder|ffmpeg)" | grep -v grep || echo "No active transcoding detected."
echo "$CB"
echo ""

echo "## SECTION 8: HARDWARE TRANSCODING"
echo "### DRI Devices & VA-API"
echo "$CB"
if [ -d /dev/dri ]; then
    ls -la /dev/dri/
else
    echo "ERROR: /dev/dri does not exist"
fi
echo "---"
if command -v vainfo &> /dev/null; then
    run_safe "vainfo 2>&1 | head -25"
else
    echo "vainfo not installed"
fi
echo "$CB"
echo ""

echo "## SECTION 9: SECURITY & LOGS"
echo "### Fail2Ban Health & Activity"
echo "$CB"
run_safe "systemctl status fail2ban.service --no-pager -l | head -15"
echo "---"
echo "Recent Bans (Last 24h):"
journalctl -u fail2ban --since "24 hours ago" --no-pager | grep -i "Ban " | tail -10 || echo "No active bans in last 24h."
echo "$CB"
echo ""

echo "### Caddy & Nextcloud Logs"
echo "$CB"
echo "--- Caddy (Reverse Proxy) ---"
if [ -f /var/log/caddy/access.log ]; then
    tail -10 /var/log/caddy/access.log
else
    run_safe "docker logs --tail 20 caddy 2>&1 || echo 'Caddy container not found'"
fi

echo "--- Nextcloud ---"
run_safe "docker logs --tail 20 nextcloud 2>&1 || echo 'Nextcloud container not found'"
echo "$CB"
echo ""

echo "### System Journal (Errors - Last 2 Hours)"
echo "$CB"
run_safe "journalctl --since '2 hours ago' -p err --no-pager | tail -50"
echo "$CB"
echo ""

echo "## SECTION 10: FAILED SERVICES"
echo "$CB"
run_safe "systemctl --failed --no-pager"
echo "$CB"
echo ""

cat << 'REPORT_TEMPLATE'
================================================================================
>>> LLM REPORTING INSTRUCTION <<<
================================================================================
Based on the logs above, generate a report using the markdown format below.

**CRITICAL INSTRUCTION: REMEDIATION PRIORITIES**
1. **Immediate Action:** Any "Failed" service, "OOMKilled" container, or "Stale" WireGuard handshake.
2. **Maintenance:** Staged updates, High (but not full) disk usage, Old "Transient" errors.

# Diagnostic Report: [Date]

## 1. Executive Summary
**System Health:** [GREEN / YELLOW / RED]
**Primary Issue:** [One concise sentence describing the main *active* problem]

## 2. Critical Findings
* **Boot Health:** [Mention if system recently crashed/rebooted and why]
* **Storage:** [Status of NVMe, Inodes, and Mounts]
* **Containers:** [Highlight OOMKilled containers or Restart loops]
* **Network:** [WireGuard Tunnel Health & NAS Connectivity]

## 3. Remediation Plan
### Immediate Actions (High Priority)
1. **[Action]**: [Exact command]
   * *Reasoning:* [Why this is critical]

### Maintenance Tasks (Medium Priority)
1. **[Task]**: [e.g., Reboot to apply update, Clean Docker builder cache]

## 4. Technical Deep Dive
* **Log Evidence:** [Quote specific lines]
* **Root Cause Analysis:** [Explain *why*]

================================================================================
END OF DIAGNOSTIC REPORT INSTRUCTIONS
================================================================================
REPORT_TEMPLATE

echo "Pruning diagnostic logs older than 30 days..."
find "$LOG_DIR" -type f \( -name "llm-diagnostic-*.log" -o -name "minipc-diagnostic-*.log" \) -mtime +30 -delete
