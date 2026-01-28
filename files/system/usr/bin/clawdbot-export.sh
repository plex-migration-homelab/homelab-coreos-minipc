#!/usr/bin/env bash
set -euo pipefail

EXPORT_DIR="/var/log/clawdbot-export"
MOUNT_FILTER='(nas|nfs|cifs|mergerfs|mnt)'

umask 022
mkdir -p "$EXPORT_DIR"
chmod 0755 "$EXPORT_DIR"

write_output() {
    local file="$1"
    shift
    local cmd="$*"

    if ! bash -c "$cmd" > "${EXPORT_DIR}/${file}" 2>&1; then
        echo "Command failed: ${cmd}" > "${EXPORT_DIR}/${file}"
    fi
    chmod 0644 "${EXPORT_DIR}/${file}"
}

write_output "kernel.tail.txt" "dmesg -T | tail -n 500"
write_output "systemd.failed.txt" "systemctl --failed --no-pager"
write_output "mounts.txt" "findmnt -r -o TARGET,SOURCE,FSTYPE,OPTIONS; echo ''; mount"

systemctl list-units --type=mount --no-pager --no-legend \
    | awk '{print $1}' \
    | grep -E "${MOUNT_FILTER}" > "${EXPORT_DIR}/systemd-mount-units.txt" || true
chmod 0644 "${EXPORT_DIR}/systemd-mount-units.txt"

if [ -s "${EXPORT_DIR}/systemd-mount-units.txt" ]; then
    while read -r unit; do
        [ -z "$unit" ] && continue
        journal_file="${EXPORT_DIR}/journal.${unit}.txt"
        if ! journalctl -u "$unit" --no-pager -n 50 > "$journal_file" 2>&1; then
            echo "journalctl failed for ${unit}" > "$journal_file"
        fi
        chmod 0644 "$journal_file"
    done < "${EXPORT_DIR}/systemd-mount-units.txt"
fi

if command -v lspci >/dev/null 2>&1; then
    write_output "pcie.link.txt" "lspci -vv | grep -E '(^[0-9a-f]{2}:|LnkCap|LnkSta)'"
else
    write_output "pcie.link.txt" "echo 'lspci not available'"
fi

latest_log="$(ls -1t /var/log/llm-diagnostic/minipc-diagnostic-*.log 2>/dev/null | head -1 || true)"

echo "${latest_log}" > "${EXPORT_DIR}/latest-diagnostic.path"
chmod 0644 "${EXPORT_DIR}/latest-diagnostic.path"

if [ -n "${latest_log}" ] && [ -f "${latest_log}" ]; then
    head -n 15 "${latest_log}" > "${EXPORT_DIR}/latest-diagnostic.head.txt"
    tail -n 15 "${latest_log}" > "${EXPORT_DIR}/latest-diagnostic.tail.txt"
else
    echo "No diagnostic log found" > "${EXPORT_DIR}/latest-diagnostic.head.txt"
    echo "No diagnostic log found" > "${EXPORT_DIR}/latest-diagnostic.tail.txt"
fi

chmod 0644 "${EXPORT_DIR}/latest-diagnostic.head.txt"
chmod 0644 "${EXPORT_DIR}/latest-diagnostic.tail.txt"
