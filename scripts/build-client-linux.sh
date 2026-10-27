#!/usr/bin/env bash
# Produces a .deb, .rpm, and an AppImage under client/src-tauri/target/release/bundle/

set -euo pipefail

info() { echo "[INFO]  $*"; }

REPO_ROOT="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
FRONTEND_DIR="$REPO_ROOT/client/frontend"
TAURI_DIR="$REPO_ROOT/client/src-tauri"

info "Installing Node.js dependencies..."
cd "$FRONTEND_DIR"
npm ci --prefer-offline

info "Building frontend (TypeScript type-check + Vite bundle)..."
npm run build

info "Building Tauri application in release mode..."
cd "$TAURI_DIR"
cargo tauri build

BUNDLE_DIR="$TAURI_DIR/target/release/bundle"
info "Build complete. Bundles:"
find "$BUNDLE_DIR" -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.AppImage" \) \
    | while read -r bundle_file; do echo "  $bundle_file"; done