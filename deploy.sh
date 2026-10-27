#!/usr/bin/env bash
# Runs the server deployment setup steps in order.
# Review SSH_PORT and the firewall rules before confirming.

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info() { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
abort() { echo -e "${RED}[ERR]${NC}  $*" >&2; exit 1; }

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$PROJECT_ROOT/scripts"
SSH_PORT="${SSH_PORT:-22}"

for script in install-server-prereqs.sh generate-tls-certs.sh configure-firewall.sh; do
    [[ -r "$SCRIPTS_DIR/$script" ]] || abort "Missing or unreadable script: $SCRIPTS_DIR/$script"
done

[[ "$SSH_PORT" =~ ^[0-9]{1,5}$ ]] && (( SSH_PORT >= 1 && SSH_PORT <= 65535 )) \
    || abort "SSH_PORT must be a port number between 1 and 65535."

[[ "$EUID" -ne 0 ]] || abort "Run deploy.sh as your project user, not root. The script will use sudo only for privileged steps."

if ! command -v sudo >/dev/null 2>&1; then
    abort "sudo is required to run the server prerequisites and firewall setup."
fi

cat <<EOF
This will run server prerequisites, generate local self-signed TLS certificates,
and reset UFW rules before applying the ANONCHAT firewall rules.

Project: $PROJECT_ROOT
SSH port to allow: $SSH_PORT

Ensure this is the correct SSH port and that you have console access before
continuing. The firewall script also exposes TCP ports 9001 and 8080 by default.
EOF

read -r -p "Continue with deployment setup? Type 'yes' to proceed: " confirmation
[[ "$confirmation" == "yes" ]] || abort "Deployment setup cancelled."

info "Installing server prerequisites..."
sudo bash "$SCRIPTS_DIR/install-server-prereqs.sh"

info "Generating TLS certificates as $(id -un)..."
bash "$SCRIPTS_DIR/generate-tls-certs.sh"

info "Configuring UFW (SSH port $SSH_PORT)..."
sudo env SSH_PORT="$SSH_PORT" bash "$SCRIPTS_DIR/configure-firewall.sh"

info "Server setup scripts completed."
info "Review the Compose configuration with: cd '$PROJECT_ROOT' && docker compose config"
info "Build and start the stack with: cd '$PROJECT_ROOT' && docker compose up -d --build"
warn "If your user was just added to the docker group, log out and back in before using Docker without sudo."
