# ANONCHAT — Verbose Deployment Guide

> **Target OS:** Ubuntu 22.04 LTS (Jammy) or 24.04 LTS (Noble) — 64-bit
> **Architecture:** The server stack runs entirely in Docker containers. The Tauri desktop client is built and distributed separately.
> **Security reminder:** ANONCHAT is a zero-knowledge, no-log application. Every step below that touches secrets must be run in a RAM-backed temp directory where possible.

---

## Table of Contents

1. [Server Prerequisites](#1-server-prerequisites)
2. [Repository Setup](#2-repository-setup)
3. [TLS Certificate Generation](#3-tls-certificate-generation)
4. [Environment Configuration](#4-environment-configuration)
5. [Docker Compose — Server Stack](#5-docker-compose--server-stack)
6. [Firewall Rules](#6-firewall-rules)
7. [Systemd Service (auto-start)](#7-systemd-service-auto-start)
8. [Verifying the Running Stack](#8-verifying-the-running-stack)
9. [Building the Desktop Client (Linux)](#9-building-the-desktop-client-linux)
10. [Building the Desktop Client (Windows / macOS)](#10-building-the-desktop-client-windows--macos)
11. [Production Hardening Checklist](#11-production-hardening-checklist)
12. [Tear-Down / Clean Restart](#12-tear-down--clean-restart)

---

## 1. Server Prerequisites

Run the script below as `root` (or via `sudo bash`) on a **fresh** Ubuntu 22.04 / 24.04 server.
The script is idempotent — safe to re-run if a step was interrupted.

```bash
#!/usr/bin/env bash
# scripts/install-server-prereqs.sh
# Installs: Docker Engine, Docker Compose v2, git, openssl, curl, ufw
# Tested on: Ubuntu 22.04 LTS, Ubuntu 24.04 LTS

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

DISTRO_ID=$(. /etc/os-release && echo "$ID")
DISTRO_VERSION=$(. /etc/os-release && echo "$VERSION_ID")
info "Detected OS: $DISTRO_ID $DISTRO_VERSION"

[[ "$DISTRO_ID" == "ubuntu" ]] || abort "This script targets Ubuntu only."

#──────────────────────────────────────────────────────────────
# 2. Base system update
#──────────────────────────────────────────────────────────────
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
#    Removes any Ubuntu-bundled docker.io / docker-compose
#    before installing the upstream packages.
#──────────────────────────────────────────────────────────────
info "Removing legacy Docker packages if present..."
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
    apt-get remove -y "$pkg" 2>/dev/null || true
done

info "Adding Docker's official GPG key and APT repository..."
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
   https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) stable" \
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
#    Replace 'ubuntu' with your actual non-root deploy user if different.
#──────────────────────────────────────────────────────────────
DEPLOY_USER="${SUDO_USER:-ubuntu}"
if id "$DEPLOY_USER" &>/dev/null; then
    info "Adding $DEPLOY_USER to the docker group..."
    usermod -aG docker "$DEPLOY_USER"
    warn "You must log out and back in (or run 'newgrp docker') for the group change to take effect."
else
    warn "User '$DEPLOY_USER' not found — skipping docker group assignment."
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
```

Save the script and run it:

```bash
sudo bash scripts/install-server-prereqs.sh
# Re-login or run: newgrp docker
```

---

## 2. Repository Setup

```bash
# Choose a deployment root — /opt is conventional for services
sudo mkdir -p /opt/anonchat
sudo chown "$USER":"$USER" /opt/anonchat

git clone https://github.com/your-org/anonchat.git /opt/anonchat
cd /opt/anonchat
```

---

## 3. TLS Certificate Generation

Each proxy node (Guard, Middle, Exit) and the Chat Router require TLS certificates.
In production use certificates issued by a trusted CA (e.g. Let's Encrypt via `certbot`).
The script below generates **self-signed** certificates suitable for an internal network
or for initial smoke-testing; swap them for real certs before exposing the service publicly.

```bash
#!/usr/bin/env bash
# scripts/generate-tls-certs.sh
# Generates self-signed TLS keypairs for each server component.
# Output: certs/  directory at the project root.

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; NC='\033[0m'
info() { echo -e "${GREEN}[INFO]${NC}  $*"; }

CERT_DIR="$(pwd)/certs"
mkdir -p "$CERT_DIR"

# Key size — 4096 for production, 2048 acceptable for internal dev
KEY_BITS=4096

# Days until expiry
DAYS=365

generate_cert() {
    local name="$1"
    local cn="$2"
    local out_dir="$CERT_DIR/$name"
    mkdir -p "$out_dir"

    info "Generating keypair for: $cn"

    openssl req -x509 -newkey rsa:${KEY_BITS} \
        -keyout "$out_dir/key.pem" \
        -out    "$out_dir/cert.pem" \
        -days   "${DAYS}" \
        -nodes \
        -subj   "/CN=${cn}/O=AnonChat/C=XX" \
        -addext "subjectAltName=DNS:${cn},DNS:localhost,IP:127.0.0.1"

    chmod 600 "$out_dir/key.pem"
    chmod 644 "$out_dir/cert.pem"

    info "  cert: $out_dir/cert.pem"
    info "  key:  $out_dir/key.pem"
}

generate_cert "proxy-guard"  "proxy-guard"
generate_cert "proxy-middle" "proxy-middle"
generate_cert "proxy-exit"   "proxy-exit"
generate_cert "router"       "chat-router"

info "All certificates generated in: $CERT_DIR"
echo ""
echo "Mount them into containers via volumes in docker-compose.override.yml"
echo "and point TLS_CERT_FILE / TLS_KEY_FILE env vars at the mounted paths."
```

```bash
bash scripts/generate-tls-certs.sh
```

---

## 4. Environment Configuration

Create a `.env` file at the project root. This file is **never** committed to version control.

```bash
cat > /opt/anonchat/.env <<'EOF'
# ─── Redis ────────────────────────────────────────────────────────────────────
REDIS_URL=redis://redis-ram:6379

# ─── Chat Router ──────────────────────────────────────────────────────────────
ROUTER_PORT=8080

# ─── Proxy Nodes ──────────────────────────────────────────────────────────────
# NEXT_HOP_ADDR is resolved automatically by role; set explicitly to override.
# GUARD_NEXT_HOP=proxy-middle:9002
# MIDDLE_NEXT_HOP=proxy-exit:9003
# EXIT_NEXT_HOP=chat-router:8080

# ─── TLS (leave blank to use auto-generated ephemeral certs in dev mode) ──────
GUARD_TLS_CERT=/run/secrets/guard_cert
GUARD_TLS_KEY=/run/secrets/guard_key
MIDDLE_TLS_CERT=/run/secrets/middle_cert
MIDDLE_TLS_KEY=/run/secrets/middle_key
EXIT_TLS_CERT=/run/secrets/exit_cert
EXIT_TLS_KEY=/run/secrets/exit_key

# ─── Dial / idle timeouts (seconds) ───────────────────────────────────────────
PROXY_DIAL_TIMEOUT=10
PROXY_IDLE_TIMEOUT=120
EOF

chmod 600 /opt/anonchat/.env
```

Create an override file to mount the generated certs and wire the env vars:

```bash
cat > /opt/anonchat/docker-compose.override.yml <<'EOF'
version: '3.8'

services:

  proxy-guard:
    environment:
      - TLS_CERT_FILE=/certs/cert.pem
      - TLS_KEY_FILE=/certs/key.pem
    volumes:
      - ./certs/proxy-guard:/certs:ro

  proxy-middle:
    environment:
      - TLS_CERT_FILE=/certs/cert.pem
      - TLS_KEY_FILE=/certs/key.pem
    volumes:
      - ./certs/proxy-middle:/certs:ro

  proxy-exit:
    environment:
      - TLS_CERT_FILE=/certs/cert.pem
      - TLS_KEY_FILE=/certs/key.pem
    volumes:
      - ./certs/proxy-exit:/certs:ro

  chat-router:
    volumes:
      - ./certs/router:/certs:ro
EOF
```

---

## 5. Docker Compose — Server Stack

### 5.1 Build all images

```bash
cd /opt/anonchat

# Build without using the cache to get fresh Go module downloads
docker compose build --no-cache --pull

# Verify images were created
docker images | grep anonchat
```

Expected output (image names will be prefixed with the project directory name):

```
anonchat-proxy-guard    latest   ...
anonchat-proxy-middle   latest   ...
anonchat-proxy-exit     latest   ...
anonchat-chat-router    latest   ...
```

### 5.2 Start the stack

```bash
# Start in detached mode; Docker Compose merges docker-compose.yml and
# docker-compose.override.yml automatically.
docker compose up -d

# Tail the logs of all services simultaneously (Ctrl+C to stop tailing;
# containers continue running)
docker compose logs -f
```

### 5.3 Expected startup sequence

1. `anonchat-redis` starts first (no dependencies); should be ready within ~1 s.
2. `anonchat-router` connects to Redis; logs a "listening on :8080" line.
3. `proxy-guard`, `proxy-middle`, and `proxy-exit` start in parallel; each logs
   the role it assumed and the port it bound.

---

## 6. Firewall Rules

```bash
#!/usr/bin/env bash
# scripts/configure-firewall.sh
# Configures UFW to expose only the ports required by ANONCHAT.
# Run as root on the server.

set -euo pipefail

info() { echo "[INFO]  $*"; }

info "Resetting UFW to defaults..."
ufw --force reset

info "Setting default policies..."
ufw default deny incoming
ufw default allow outgoing

info "Allowing SSH (port 22)..."
ufw allow 22/tcp comment "SSH admin access"

info "Allowing Guard Proxy (port 9001 — clients connect here)..."
ufw allow 9001/tcp comment "AnonChat Guard Proxy"

info "Allowing Chat Router WebSocket (port 8080 — only reachable from exit proxy)..."
# In a multi-server deployment, restrict this to the exit node's IP:
# ufw allow from <EXIT_NODE_IP> to any port 8080 proto tcp
# For single-host deployments, Docker handles inter-container networking
# internally; 8080 need not be publicly exposed.  Comment out the next
# line to keep the router private.
ufw allow 8080/tcp comment "AnonChat Chat Router (restrict in multi-server setup)"

info "Ports 9002 and 9003 (middle / exit) are internal Docker network only."
info "They are NOT opened externally."

info "Enabling UFW..."
ufw --force enable
ufw status verbose
```

```bash
sudo bash scripts/configure-firewall.sh
```

> **Multi-server deployment note:** If Guard, Middle, Exit, and Router run on
> separate machines, also open 9002 on the Middle host and 9003 on the Exit
> host — but restrict each to the IP of the preceding node using
> `ufw allow from <IP> to any port <PORT> proto tcp`.

---

## 7. Systemd Service (auto-start)

Make the stack start automatically on boot and restart on failure.

```bash
sudo tee /etc/systemd/system/anonchat.service > /dev/null <<EOF
[Unit]
Description=AnonChat Server Stack (Docker Compose)
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/anonchat
ExecStart=/usr/bin/docker compose up -d --remove-orphans
ExecStop=/usr/bin/docker compose down
ExecReload=/usr/bin/docker compose pull && /usr/bin/docker compose up -d --remove-orphans
TimeoutStartSec=120
TimeoutStopSec=30
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now anonchat.service
sudo systemctl status anonchat.service
```

To trigger a rolling update after pushing new images:

```bash
sudo systemctl reload anonchat.service
# or manually:
cd /opt/anonchat && docker compose pull && docker compose up -d --remove-orphans
```

---

## 8. Verifying the Running Stack

```bash
# All five containers should show "Up"
docker compose ps

# Inspect Redis — confirm persistence is OFF
docker exec anonchat-redis redis-cli CONFIG GET save
# Expected: (empty array — no save rules)
docker exec anonchat-redis redis-cli CONFIG GET appendonly
# Expected: "no"

# Confirm the chat router is accepting WebSocket upgrades
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Sec-WebSocket-Version: 13" \
  -H "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
  http://localhost:8080/ws
# Expected: HTTP/1.1 101 Switching Protocols

# Check guard proxy TLS listener
openssl s_client -connect localhost:9001 -verify_quiet 2>&1 | head -5
# Expected: shows TLS handshake details and the certificate CN

# Live container resource usage
docker stats --no-stream
```

---

## 9. Building the Desktop Client (Linux)

Run this on a **developer workstation** (not the server), or in a CI pipeline.

```bash
#!/usr/bin/env bash
# scripts/install-client-prereqs-linux.sh
# Installs prerequisites for building the Tauri desktop client on Ubuntu.

set -euo pipefail

info() { echo "[INFO]  $*"; }

#──────────────────────────────────────────────────────────────
# 1. System packages required by Tauri on Linux
#──────────────────────────────────────────────────────────────
info "Installing system dependencies for Tauri..."
sudo apt-get update -y
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    build-essential \
    libssl-dev \
    libgtk-3-dev \
    libayatana-appindicator3-dev \
    librsvg2-dev \
    libwebkit2gtk-4.1-dev \
    libxdo-dev \
    pkg-config \
    curl \
    git

#──────────────────────────────────────────────────────────────
# 2. Rust toolchain (rustup)
#    Minimum required version: 1.75
#──────────────────────────────────────────────────────────────
if ! command -v rustup &>/dev/null; then
    info "Installing Rust via rustup (stable channel)..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \
        | sh -s -- -y --default-toolchain stable --no-modify-path
    source "$HOME/.cargo/env"
else
    info "rustup already present — updating toolchain..."
    rustup update stable
fi

RUST_VERSION=$(rustc --version)
info "Rust: $RUST_VERSION"

# Confirm Rust >= 1.75
RUST_SEMVER=$(rustc --version | grep -oP '\d+\.\d+\.\d+')
RUST_MAJOR=$(echo "$RUST_SEMVER" | cut -d. -f1)
RUST_MINOR=$(echo "$RUST_SEMVER" | cut -d. -f2)
if [[ "$RUST_MAJOR" -lt 1 || ( "$RUST_MAJOR" -eq 1 && "$RUST_MINOR" -lt 75 ) ]]; then
    echo "[ERR]  Rust >= 1.75 required. Got: $RUST_SEMVER" >&2
    exit 1
fi

#──────────────────────────────────────────────────────────────
# 3. Node.js v20+ (via NodeSource)
#──────────────────────────────────────────────────────────────
if ! command -v node &>/dev/null || [[ "$(node --version | tr -d 'v' | cut -d. -f1)" -lt 20 ]]; then
    info "Installing Node.js v20 via NodeSource..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
else
    info "Node.js $(node --version) already installed."
fi

info "Node: $(node --version)   npm: $(npm --version)"

#──────────────────────────────────────────────────────────────
# 4. Tauri CLI (installed as a Cargo binary)
#──────────────────────────────────────────────────────────────
info "Installing Tauri CLI..."
cargo install tauri-cli --version "^2.0" --locked

info "Tauri CLI: $(cargo tauri --version)"

info "All client build prerequisites installed."
echo ""
echo "Next step: run scripts/build-client-linux.sh"
```

```bash
bash scripts/install-client-prereqs-linux.sh
source "$HOME/.cargo/env"   # ensure Cargo binaries are on PATH
```

Then build the release binary and installer packages:

```bash
#!/usr/bin/env bash
# scripts/build-client-linux.sh
# Produces a .deb, .rpm, and an AppImage under client/src-tauri/target/release/bundle/

set -euo pipefail

info() { echo "[INFO]  $*"; }

REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
FRONTEND_DIR="$REPO_ROOT/client/frontend"
TAURI_DIR="$REPO_ROOT/client/src-tauri"

#──────────────────────────────────────────────────────────────
# 1. Install frontend JS dependencies
#──────────────────────────────────────────────────────────────
info "Installing Node.js dependencies..."
cd "$FRONTEND_DIR"
npm ci --prefer-offline

#──────────────────────────────────────────────────────────────
# 2. Type-check + build the Vite frontend bundle
#──────────────────────────────────────────────────────────────
info "Building frontend (TypeScript type-check + Vite bundle)..."
npm run build
# Output: client/frontend/dist/

#──────────────────────────────────────────────────────────────
# 3. Build the Tauri application (Rust + bundle)
#    --release applies the [profile.release] settings in Cargo.toml:
#      strip = true, opt-level = "z", lto = true, panic = "abort"
#──────────────────────────────────────────────────────────────
info "Building Tauri application in release mode..."
cd "$TAURI_DIR"
cargo tauri build

BUNDLE_DIR="$TAURI_DIR/target/release/bundle"
info "Build complete. Bundles:"
find "$BUNDLE_DIR" -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.AppImage" \) \
    | while read -r f; do echo "  $f"; done
```

```bash
bash scripts/build-client-linux.sh
```

Distribute the resulting `.deb`, `.rpm`, or `.AppImage` from
`client/src-tauri/target/release/bundle/` to end-users.

---

## 10. Building the Desktop Client (Windows / macOS)

These steps run on the respective platform natively (cross-compilation for
Tauri GUI bundles is not supported).

### Windows (PowerShell, run as user — no admin required for Rustup/Node)

```powershell
# 1. Install Rust
winget install --id Rustlang.Rustup -e
# Restart terminal, then:
rustup update stable

# 2. Install Node.js v20+
winget install --id OpenJS.NodeJS.LTS -e
# Restart terminal

# 3. Install Visual Studio C++ build tools (required by Rust on Windows)
winget install --id Microsoft.VisualStudio.2022.BuildTools -e `
    --override "--passive --wait --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"

# 4. Tauri CLI
cargo install tauri-cli --version "^2.0" --locked

# 5. Build
cd client\frontend
npm ci
npm run build
cd ..\src-tauri
cargo tauri build
# Installer: client\src-tauri\target\release\bundle\msi\AnonChat_*.msi
#            client\src-tauri\target\release\bundle\nsis\AnonChat_*-setup.exe
```

### macOS (zsh / bash)

```bash
# 1. Install Xcode Command Line Tools
xcode-select --install

# 2. Install Homebrew (if not present)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 3. Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustup update stable

# 4. Node.js v20+
brew install node@20
echo 'export PATH="/opt/homebrew/opt/node@20/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# 5. Tauri CLI
cargo install tauri-cli --version "^2.0" --locked

# 6. Build
cd client/frontend && npm ci && npm run build
cd ../src-tauri && cargo tauri build
# Output: client/src-tauri/target/release/bundle/dmg/AnonChat_*.dmg
#         client/src-tauri/target/release/bundle/macos/AnonChat.app
```

---

## 11. Production Hardening Checklist

| # | Item | How |
|---|------|-----|
| 1 | Replace self-signed certs with CA-signed certs | `certbot certonly --standalone -d <your-domain>` then mount `/etc/letsencrypt/live/<domain>/` |
| 2 | Set `INSECURE_SKIP_VERIFY=false` (default) | Never override this in production |
| 3 | Redis `protected-mode yes` + no public port | Remove `ports: - "6379:6379"` from `docker-compose.yml` in production |
| 4 | Rotate TLS certs before expiry | Add a cron job: `certbot renew && docker compose restart` |
| 5 | Run containers as non-root | Already done — Dockerfiles use UID 10001/10002 |
| 6 | No bind-mount of sensitive host paths | Only `./certs/` is mounted; never mount `/etc` or `/home` |
| 7 | Limit container memory | Add `mem_limit: 256m` per service in `docker-compose.override.yml` |
| 8 | Enable Docker Content Trust | `export DOCKER_CONTENT_TRUST=1` before any `docker pull/build` |
| 9 | Disable swap on the server | `swapoff -a && sed -i '/swap/d' /etc/fstab` — prevents key material from being swapped to disk |
| 10 | Audit logs with `auditd` | `apt install auditd && auditctl -w /opt/anonchat/certs -p rwa -k cert-access` |

---

## 12. Tear-Down / Clean Restart

```bash
cd /opt/anonchat

# Stop all containers (does NOT remove volumes)
docker compose down

# Full clean: remove containers, networks, and built images
docker compose down --rmi all --volumes --remove-orphans

# Prune dangling build cache (reclaim disk space)
docker builder prune -f

# Rebuild and restart from scratch
docker compose build --no-cache --pull
docker compose up -d
```

To permanently remove the deployment:

```bash
sudo systemctl disable --now anonchat.service
sudo rm /etc/systemd/system/anonchat.service
sudo systemctl daemon-reload

docker compose -f /opt/anonchat/docker-compose.yml down --rmi all --volumes --remove-orphans
sudo rm -rf /opt/anonchat
```

---

*End of deployment guide.*
