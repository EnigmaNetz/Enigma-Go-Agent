#!/bin/bash
#
# Unified Enigma Agent installer for Debian/Ubuntu and CentOS/RHEL/Fedora
#
# Usage:
#   Place the appropriate .deb or .rpm package in the current directory.
#   Run: sudo bash install-enigma-agent.sh
#
# This script auto-detects your OS, installs Zeek and tcpdump,
# installs the Enigma Agent package, writes config if missing,
# and restarts the systemd service if present.
#
# The old CentOS-specific script (installer/centos/install-enigma-agent.sh) has been removed.
#
set -eu

# --- User-provided variables ---
ENIGMA_API_KEY="${ENIGMA_API_KEY:-}"
ENIGMA_API_URL="${ENIGMA_API_URL:-https://api.enigmaai.net}"

if [ -z "$ENIGMA_API_KEY" ]; then
  echo "ENIGMA_API_KEY environment variable not set."
  read -r -s -p "Enter your Enigma API Key: " ENIGMA_API_KEY
  echo
  if [ -z "$ENIGMA_API_KEY" ]; then
    echo "ERROR: API key is required."
    exit 1
  fi
fi

# --- Detect OS ---
if [ -f /etc/os-release ]; then
  . /etc/os-release
  OS_ID=$ID
else
  echo "ERROR: Cannot detect OS type (missing /etc/os-release)."
  exit 1
fi

case "$OS_ID" in
  ubuntu|debian)
    # --- Ensure curl and gpg are installed ---
    apt update
    apt install -y curl gpg
    # --- Add Zeek repository and key if not present ---
    if ! grep -q 'security:/zeek' /etc/apt/sources.list.d/security:zeek.list 2>/dev/null; then
      curl -fsSL https://download.opensuse.org/repositories/security:zeek/xUbuntu_22.04/Release.key | gpg --dearmor | tee /etc/apt/trusted.gpg.d/security_zeek.gpg
      echo 'deb http://download.opensuse.org/repositories/security:/zeek/xUbuntu_22.04/ /' | tee /etc/apt/sources.list.d/security:zeek.list
      apt update
    fi
    # --- Install Zeek, tcpdump, and dependencies ---
    export DEBIAN_FRONTEND=noninteractive
    apt install -y zeek tcpdump
    # --- Find and install Enigma Agent .deb package ---
    PKG=$(ls ./*.deb 2>/dev/null | head -n1)
    if [ -z "$PKG" ]; then
      echo "ERROR: No .deb package found in the current directory."
      exit 1
    fi
    dpkg -i "$PKG" || apt-get install -f -y
    ;;
  rocky)
    # --- Add Zeek repository ---
    dnf install -y https://download.opensuse.org/repositories/security:/zeek/CentOS_8/security:zeek.repo
    # --- Install Zeek, tcpdump, and dependencies ---
    dnf install -y epel-release || true
    dnf install -y zeek tcpdump || dnf install -y zeek tcpdump
    # --- Find and install Enigma Agent .rpm package ---
    PKG=$(ls ./*.rpm 2>/dev/null | head -n1)
    if [ -z "$PKG" ]; then
      echo "ERROR: No .rpm package found in the current directory."
      exit 1
    fi
    dnf install -y "$PKG" || dnf install -y "$PKG"
    ;;
  *)
    echo "ERROR: Unsupported Linux distribution: $OS_ID"
    exit 1
    ;;
esac

# --- Write config file only if it doesn't exist ---
mkdir -p /etc/enigma-agent
if [ ! -f /etc/enigma-agent/config.json ]; then
  cat > /etc/enigma-agent/config.json <<EOF
{
  "enigma_api": {
    "api_key": "$ENIGMA_API_KEY",
    "server": "$ENIGMA_API_URL"
  }
}
EOF
fi

# --- Restart service if systemd is present ---
if command -v systemctl >/dev/null 2>&1; then
  systemctl restart enigma-agent || true
fi

echo "Enigma Agent installed and configured."
