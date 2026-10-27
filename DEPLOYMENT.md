# ANONCHAT — Verbose Deployment Guide

> **Target OS:** Ubuntu 22.04 LTS (Jammy), Ubuntu 24.04 LTS (Noble), or Debian 13 (Trixie) — 64-bit
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

For a guided run of the server setup steps, use the project-root master script
after cloning the repository and reviewing the firewall warning in section 6:

```bash
./deploy.sh
```

It runs server prerequisites, TLS certificate generation, and firewall setup in
that order. It prompts before changing the host; set `SSH_PORT` for nonstandard
SSH ports. Continue with the build and startup steps below after it completes.

The standalone script supports Ubuntu 22.04/24.04 and Debian 13. It updates
system packages, installs Docker Engine and Compose, and applies kernel tuning.
Run it as `root` (or via `sudo bash`) on a fresh server; it is safe to rerun
after an interrupted setup.

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

git clone https://github.com/Mayvena/anonchat.git /opt/anonchat
cd /opt/anonchat
```

---

## 3. TLS Certificate Generation

Each proxy node (Guard, Middle, Exit) uses TLS certificates. The Chat Router currently serves plain HTTP/WebSocket and does not load a certificate.
In production use certificates issued by a trusted CA (e.g. Let's Encrypt via `certbot`).
Run the generator as the project owner, not with `sudo`; it makes private keys
group-readable only to that user's primary group and grants the proxy containers
that supplementary group.

The script generates **self-signed** certificates suitable for an internal network
or initial smoke-testing, and writes `docker-compose.override.yml` with read-only
mounts and TLS paths for all three proxies. Replace the certificates with trusted
CA-issued files before exposing the service publicly.

```bash
bash scripts/generate-tls-certs.sh
```

The generated override is automatically merged by `docker compose`. The script
will refuse to replace an existing override unless it exactly matches its own
generated file; move a custom override aside and merge these TLS settings yourself
if you need additional Compose customization.

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

# ─── Dial / idle timeouts (seconds) ───────────────────────────────────────────
PROXY_DIAL_TIMEOUT=10
PROXY_IDLE_TIMEOUT=120
EOF

chmod 600 /opt/anonchat/.env
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
sudo bash scripts/configure-firewall.sh
```

This resets existing UFW rules before applying its defaults and ANONCHAT ports.
Confirm your SSH port first; for a nonstandard port, run with `SSH_PORT=<port>`.
For single-host deployments, remove or restrict the router's port 8080 rule if
it should not be exposed publicly.

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
ExecReload=/bin/sh -c '/usr/bin/docker compose pull && /usr/bin/docker compose up -d --remove-orphans'
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

# Inspect Redis - confirm persistence is OFF
docker exec anonchat-redis redis-cli CONFIG GET save
# Expected: (empty array - no save rules)
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

Run these scripts on a **developer workstation** (not the server), or in a CI pipeline.

```bash
bash scripts/install-client-prereqs-linux.sh
source "$HOME/.cargo/env"   # ensure Cargo binaries are on PATH
```

Then build the release binary and installer packages:

```bash
bash scripts/build-client-linux.sh
```

Distribute the resulting `.deb`, `.rpm`, or `.AppImage` from
`client/src-tauri/target/release/bundle/` to end-users.

---

## 10. Building the Desktop Client (Windows / macOS)

These steps run on the respective platform natively (cross-compilation for Tauri GUI bundles is not supported).

### Windows (PowerShell, run as user - no admin required for Rustup/Node)

```powershell
winget install --id Rustlang.Rustup -e
rustup update stable
winget install --id OpenJS.NodeJS.LTS -e
winget install --id Microsoft.VisualStudio.2022.BuildTools -e `
    --override "--passive --wait --add Microsoft.VisualStudio.Workload.VCTools --includeRecommended"
cargo install tauri-cli --version "^2.0" --locked
cd client\frontend
npm ci
npm run build
cd ..\src-tauri
cargo tauri build
```

### macOS (zsh / bash)

```bash
xcode-select --install
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source "$HOME/.cargo/env"
rustup update stable
brew install node@20
echo 'export PATH="/opt/homebrew/opt/node@20/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
cargo install tauri-cli --version "^2.0" --locked
cd client/frontend && npm ci && npm run build
cd ../src-tauri && cargo tauri build
```

---

## 11. Production Hardening Checklist

| # | Item | How |
|---|------|-----|
| 1 | Replace self-signed certs with CA-signed certs | Issue certificates for proxy node names and replace the files under `certs/` |
| 2 | Disable insecure upstream verification | Set `TLS_INSECURE_SKIP_VERIFY=false` for guard and middle; do not use self-signed upstream certificates in production |
| 3 | Keep Redis private and memory-only | Do not publish port 6379; retain `--save "" --appendonly no` |
| 4 | Rotate TLS certs before expiry | Renew certificates and restart proxy services |
| 5 | Run containers as non-root | Proxy and router Dockerfiles use dedicated unprivileged UIDs |
| 6 | No bind-mount of sensitive host paths | Mount only the required certificate directories |
| 7 | Limit container memory | Add `mem_limit: 256m` per service in a custom Compose override |
| 8 | Enable Docker Content Trust | `export DOCKER_CONTENT_TRUST=1` before any `docker pull/build` |
| 9 | Disable swap on the server | Disable swap and configure persistent swap policy according to the host OS |
| 10 | Audit certificate access | Install and configure `auditd` for the deployment certificate directory |

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
