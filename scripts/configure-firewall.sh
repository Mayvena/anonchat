#!/usr/bin/env bash
# Configures UFW to expose the ports required by ANONCHAT.
# Run as root on the server. Set SSH_PORT if SSH does not use port 22.

set -euo pipefail

info() { echo "[INFO]  $*"; }
SSH_PORT="${SSH_PORT:-22}"

info "Resetting UFW rules. Existing firewall rules will be removed."
ufw --force reset

info "Setting default policies..."
ufw default deny incoming
ufw default allow outgoing

info "Allowing SSH on port $SSH_PORT..."
ufw allow "$SSH_PORT/tcp" comment "SSH admin access"

info "Allowing Guard Proxy (port 9001 - clients connect here)..."
ufw allow 9001/tcp comment "AnonChat Guard Proxy"

info "Allowing Chat Router WebSocket (port 8080)..."
# In a multi-server deployment, restrict this to the exit node's IP:
# ufw allow from <EXIT_NODE_IP> to any port 8080 proto tcp
# For a single-host deployment, Docker handles inter-container networking
# internally; 8080 need not be publicly exposed. Remove the next rule to keep
# the router private.
ufw allow 8080/tcp comment "AnonChat Chat Router (restrict in multi-server setup)"

info "Ports 9002 and 9003 (middle / exit) are internal Docker network only."
info "They are NOT opened externally."

info "Enabling UFW..."
ufw --force enable
ufw status verbose