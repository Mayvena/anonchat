#!/usr/bin/env bash
# Installs prerequisites for building the Tauri desktop client on Ubuntu.

set -euo pipefail

info() { echo "[INFO]  $*"; }

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

if ! command -v rustup &>/dev/null; then
    info "Installing Rust via rustup (stable channel)..."
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \
        | sh -s -- -y --default-toolchain stable --no-modify-path
    source "$HOME/.cargo/env"
else
    info "rustup already present - updating toolchain..."
    rustup update stable
fi

RUST_VERSION=$(rustc --version)
info "Rust: $RUST_VERSION"

RUST_SEMVER=$(rustc --version | grep -oP '\d+\.\d+\.\d+')
RUST_MAJOR=$(echo "$RUST_SEMVER" | cut -d. -f1)
RUST_MINOR=$(echo "$RUST_SEMVER" | cut -d. -f2)
if [[ "$RUST_MAJOR" -lt 1 || ( "$RUST_MAJOR" -eq 1 && "$RUST_MINOR" -lt 75 ) ]]; then
    echo "[ERR]  Rust >= 1.75 required. Got: $RUST_SEMVER" >&2
    exit 1
fi

if ! command -v node &>/dev/null || [[ "$(node --version | tr -d 'v' | cut -d. -f1)" -lt 20 ]]; then
    info "Installing Node.js v20 via NodeSource..."
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nodejs
else
    info "Node.js $(node --version) already installed."
fi

info "Node: $(node --version)   npm: $(npm --version)"

info "Installing Tauri CLI..."
cargo install tauri-cli --version "^2.0" --locked

info "Tauri CLI: $(cargo tauri --version)"
info "All client build prerequisites installed."
echo ""
echo "Next step: run scripts/build-client-linux.sh"