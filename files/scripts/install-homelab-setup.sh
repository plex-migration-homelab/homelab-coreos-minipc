#!/usr/bin/env bash
set -euo pipefail

BASE_URL="https://github.com/plex-migration-homelab/homelab-coreos-minipc/releases/download/homelab-setup"

TMP_WORKDIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_WORKDIR}"
}
trap cleanup EXIT

cd "${TMP_WORKDIR}"
curl --retry 5 --retry-delay 10 --connect-timeout 10 --max-time 120 -fsSLO "${BASE_URL}/homelab-setup"
curl --retry 5 --retry-delay 10 --connect-timeout 10 --max-time 60 -fsSLO "${BASE_URL}/homelab-setup.sha256"
sha256sum -c homelab-setup.sha256
install -m 0755 homelab-setup /usr/bin/homelab-setup
