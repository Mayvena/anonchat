# ANONCHAT Component Checklist

Last reviewed: 2026-07-24
Reviewer basis: repository source and documentation inspection (no runtime execution in this review).

## Status Legend

- [x] Done: implemented in repo and traceable to concrete files.
- [~] Partial: core pieces exist, but integration/validation is still missing.
- [ ] Missing: not present in repo yet.

## 1) Infrastructure and Runtime Topology

- [x] Docker compose stack for Redis RAM + router + 3 proxy roles.
  - Evidence: docker-compose.yml
- [x] Local run guide with startup/shutdown and smoke scenario.
  - Evidence: README.md
- [~] Production deployment runbook is documented.
  - Evidence: DEPLOYMENT.md
  - Gap: referenced scripts under scripts/ are not present in repo.

## 2) Proxy Network Layer (Go)

- [x] Role-based proxy implementation (guard, middle, exit).
  - Evidence: server/proxy/main.go
- [x] TLS listener and optional certificate loading.
  - Evidence: server/proxy/main.go
- [x] SOCKS5 guard ingress and header forwarding to chain.
  - Evidence: server/proxy/main.go
- [x] Middle relay and exit final-destination dial.
  - Evidence: server/proxy/main.go
- [x] Bi-directional stream tunnel logic.
  - Evidence: server/proxy/main.go
- [~] Automated Go tests for proxy behavior.
  - Evidence: no *_test.go files found under server/proxy

## 3) Chat Router Layer (Go)

- [x] WebSocket endpoint with user_id session model.
  - Evidence: server/router/main.go, README.md
- [x] Session ack envelope and message envelope handling.
  - Evidence: server/router/main.go, README.md
- [x] Heartbeat and read/write timeout behavior.
  - Evidence: server/router/main.go
- [x] Online delivery path.
  - Evidence: server/router/main.go
- [x] Offline TTL buffering in Redis and replay cleanup.
  - Evidence: server/router/main.go
- [~] Automated Go tests for router behavior.
  - Evidence: no *_test.go files found under server/router

## 4) Security Core (Rust)

- [x] Public crate exports for onion, bootstrap, and panic destruction APIs.
  - Evidence: client/security-core/src/lib.rs
- [x] 3-layer onion packet builder with fixed 4096-byte output invariant.
  - Evidence: client/security-core/src/onion/cipher.rs
- [x] Onion cipher unit tests for size limits, oversize rejection, and randomness.
  - Evidence: client/security-core/src/onion/cipher.rs
- [x] Panic destruction pipeline: zeroization + 3-pass overwrite + hard exit.
  - Evidence: client/security-core/src/panic_destruction.rs
- [x] Panic destruction unit tests for helper logic and zeroization behavior.
  - Evidence: client/security-core/src/panic_destruction.rs
- [~] Device bootstrap API with encrypted-store abstraction.
  - Evidence: client/security-core/src/device_bootstrap.rs
  - Gap: concrete implementations of EncryptedDeviceMaterialStore and KeyMaterialProtector are not found in repo.

## 5) Tauri Bridge (Rust)

- [x] Tauri app wiring and command registration.
  - Evidence: client/src-tauri/src/lib.rs
- [x] Event payload models for proxy chain and core log.
  - Evidence: client/src-tauri/src/events.rs
- [x] Event bus bridge and stream command (`start_proxy_event_stream`, `emit_core_log`).
  - Evidence: client/src-tauri/src/ipc_bridge.rs
- [~] End-to-end validation of event flow under runtime load.
  - Evidence: implementation exists; runtime verification not recorded in this review.

## 6) Frontend (TypeScript)

- [x] Tauri event listeners for `proxy-layer-changed` and `core-log`.
  - Evidence: client/frontend/src/main.ts
- [x] Globe rendering for guard/middle/exit chain arcs and points.
  - Evidence: client/frontend/src/globe.ts
- [x] Ephemeral chat panel with TTL-based message expiry.
  - Evidence: client/frontend/src/chat.ts
- [x] Terminal panel integration for structured core logs.
  - Evidence: client/frontend/src/main.ts, client/frontend/src/terminal.ts
- [~] Automated frontend tests.
  - Evidence: no test/spec files found under client/frontend

## 7) Android Client Work

- [x] ANONCHAT Android prompt playbook and quickstart prompts.
  - Evidence: ANDROID_APP_PROMPTS.md
- [ ] Actual Android client implementation source code.
  - Evidence: no Android module/project present under anonchat tree.

## 8) CI, Quality Gates, and Release Automation

- [ ] CI pipeline workflows in repository.
  - Evidence: no .github/workflows files found.
- [~] Manual validation commands documented.
  - Evidence: AGENTS.md, README.md
  - Gap: no automated execution pipeline currently present in repo.

## 9) Project Guidance and Tracking

- [x] Project-specific AI development guidance.
  - Evidence: AGENTS.md
- [x] Work progress tracking policy and initial sessions.
  - Evidence: AGENTS.md, ANONCHAT_WORK_PROGRESS.md

## Priority Next Build Items

1. Implement concrete device material storage/protector integration for `bootstrap_or_load` path in app layer.
2. Add Go tests for proxy and router critical paths (session, heartbeat, offline replay cleanup).
3. Add frontend test coverage for event handling and message TTL behavior.
4. Add CI workflows to run Go, Rust, and frontend validation commands automatically.
5. Add real deployment helper scripts referenced by DEPLOYMENT.md, or revise docs to match actual repo assets.
