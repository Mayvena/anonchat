#!/usr/bin/env bash
# Start the local ANONCHAT backend and frontend development server.

set -euo pipefail

info() { printf '[INFO] %s\n' "$*"; }
abort() { printf '[ERR] %s\n' "$*" >&2; exit 1; }

PROJECT_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$PROJECT_ROOT/client/frontend"
MODE="${1:-browser}"

if [[ $# -gt 1 ]] || [[ "$MODE" != "browser" && "$MODE" != "--desktop" ]]; then
    abort "Usage: ./start.sh [--desktop]"
fi

command -v docker >/dev/null 2>&1 || abort "Docker is required."
docker compose version >/dev/null 2>&1 || abort "Docker Compose v2 is required."
command -v npm >/dev/null 2>&1 || abort "npm is required to run the frontend."

if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    info "Frontend dependencies are missing; installing with npm ci..."
    (cd "$FRONTEND_DIR" && npm ci)
fi

info "Building and starting backend services with Docker Compose..."
(cd "$PROJECT_ROOT" && docker compose up -d --build)

if [[ "$MODE" == "--desktop" ]]; then
    info "Starting the ANONCHAT desktop app with the encrypted send bridge..."
    (cd "$FRONTEND_DIR" && npm run dev -- --host 127.0.0.1) &
    FRONTEND_PID=$!
    cleanup() {
        kill "$FRONTEND_PID" 2>/dev/null || true
        wait "$FRONTEND_PID" 2>/dev/null || true
    }
    trap cleanup EXIT
    cd "$PROJECT_ROOT/client/src-tauri"
    "$FRONTEND_DIR/node_modules/.bin/tauri" dev
    exit $?
fi

info "Backend is running. Frontend is available at http://<host-lan-ip>:1420/ (and http://127.0.0.1:1420/ on this host)."
info "Browser mode is for preview; use './start.sh --desktop' to send encrypted messages."
info "Press Ctrl+C to stop the frontend. Backend containers remain running."
info "Stop the backend later with: cd '$PROJECT_ROOT' && docker compose down"

cd "$FRONTEND_DIR"
exec npm run dev -- --host 0.0.0.0
