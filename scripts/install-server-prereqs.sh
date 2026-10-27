#!/usr/bin/env bash
# scripts/install-server-prereqs.sh
# Installs: Docker Engine, Docker Compose v2, git, openssl, curl, ufw
# Supported: Ubuntu 22.04/24.04 LTS and Debian 13 (Trixie)

set -euo pipefail

#──────────────────────────────────────────────────────────────
# 0. Colour helpers
#──────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()    { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
abort()   { echo -e "${RED}[ERR]${NC}   $*" >&2; exit 1; }

#──────────────────────────────────────────────────────────────
# 1. Verify we are on a supported platform
#──────────────────────────────────────────────────────────────
ARCH=$(uname -m)
[[ "$ARCH" == "x86_64" ]] || abort "Only x86_64 is supported. Got: $ARCH"

. /etc/os-release
DISTRO_ID=${ID:-unknown}
DISTRO_VERSION=${VERSION_ID:-unknown}
DISTRO_CODENAME=${VERSION_CODENAME:-}
info "Detected OS: $DISTRO_ID $DISTRO_VERSION"

case "$DISTRO_ID" in
    ubuntu)
        DOCKER_REPO_OS=ubuntu
        ;;
    debian)
        [[ "$DISTRO_VERSION" == "13" ]] || abort "Only Debian 13 is supported. Got: $DISTRO_VERSION"
        DOCKER_REPO_OS=debian
        ;;
    *)
        abort "This script supports Ubuntu and Debian 13 only. Got: $DISTRO_ID"
        ;;
esac
[[ -n "$DISTRO_CODENAME" ]] || abort "Could not determine the OS release codename."

#──────────────────────────────────────────────────────────────
# 2. Base system update
#──────────────────────────────────────────────────────────────
if grep -RqsF 'dl.cloudsmith.io/public/caddy/stable/deb' \
    /etc/apt/sources.list /etc/apt/sources.list.d 2>/dev/null \
    && [[ ! -s /usr/share/keyrings/caddy-stable-archive-keyring.gpg ]]; then
    command -v curl >/dev/null 2>&1 || abort "Caddy APT source exists but its keyring is missing; install curl and repair the Caddy repository key first."
    command -v gpg >/dev/null 2>&1 || abort "Caddy APT source exists but its keyring is missing; install gnupg and repair the Caddy repository key first."
    info "Restoring the missing Caddy repository signing key..."
    install -m 0755 -d /usr/share/keyrings
    curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
        | gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    chmod 0644 /usr/share/keyrings/caddy-stable-archive-keyring.gpg
fi

info "Updating package lists and upgrading existing packages..."
apt-get update -y
DEBIAN_FRONTEND=noninteractive apt-get upgrade -y --no-install-recommends

#──────────────────────────────────────────────────────────────
# 3. Essential utilities
#──────────────────────────────────────────────────────────────
info "Installing essential utilities..."
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git \
    openssl \
    ufw \
    htop \
    jq

#──────────────────────────────────────────────────────────────
# 4. Docker Engine (official repo)
#    Removes any distribution-bundled docker.io / docker-compose
#    before installing the upstream packages.
#──────────────────────────────────────────────────────────────
info "Removing legacy Docker packages if present..."
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
    apt-get remove -y "$pkg" 2>/dev/null || true
done

info "Adding Docker's official GPG key and APT repository..."
install -m 0755 -d /etc/apt/keyrings
curl -fsSL "https://download.docker.com/linux/$DOCKER_REPO_OS/gpg" \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
    https://download.docker.com/linux/$DOCKER_REPO_OS \
    $DISTRO_CODENAME stable" \
  > /etc/apt/sources.list.d/docker.list

apt-get update -y

info "Installing Docker Engine and Docker Compose plugin..."
DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

#──────────────────────────────────────────────────────────────
# 5. Enable and start Docker
#──────────────────────────────────────────────────────────────
info "Enabling and starting Docker daemon..."
systemctl enable --now docker

#──────────────────────────────────────────────────────────────
# 6. Add the deploy user to the docker group (optional but convenient)
#    The user who invoked sudo is used when available.
#──────────────────────────────────────────────────────────────
DEPLOY_USER="${SUDO_USER:-}"
if [[ -n "$DEPLOY_USER" ]] && id "$DEPLOY_USER" &>/dev/null; then
    info "Adding $DEPLOY_USER to the docker group..."
    usermod -aG docker "$DEPLOY_USER"
    warn "You must log out and back in (or run 'newgrp docker') for the group change to take effect."
else
    warn "SUDO_USER is unset or the user was not found — skipping docker group assignment."
fi

#──────────────────────────────────────────────────────────────
# 7. Verify Docker installation
#──────────────────────────────────────────────────────────────
DOCKER_VERSION=$(docker --version)
COMPOSE_VERSION=$(docker compose version)
info "Docker installed:         $DOCKER_VERSION"
info "Docker Compose installed: $COMPOSE_VERSION"

#──────────────────────────────────────────────────────────────
# 8. Kernel tuning — increase connection backlog for proxy nodes
#──────────────────────────────────────────────────────────────
info "Applying kernel network tuning..."
cat > /etc/sysctl.d/99-anonchat.conf <<'EOF'
# Allow larger connection queues for the proxy nodes
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
# Increase local port range to avoid exhaustion under load
net.ipv4.ip_local_port_range = 1024 65535
# Keep TIME_WAIT sockets shorter in testing; adjust for production
net.ipv4.tcp_fin_timeout = 15
EOF
sysctl --system

info "All prerequisites installed successfully."
echo ""
echo "Next step: clone the repo and run scripts/generate-tls-certs.sh"