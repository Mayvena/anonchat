# ANONCHAT Work Progress

## Session: 2026-07-23 00:00 UTC - Instruction Files Optimization

### Task Summary
- Added project-specific guidance and prompt frameworks for ANONCHAT instruction files.
- Reworked generic Android prompts into ANONCHAT-specific prompts tied to current runtime contracts.
- Added a documented process for ongoing progress tracking in AGENTS.

### Scope
- Files:
  - AGENTS.md
  - ANDROID_APP_PROMPTS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - tauri
  - frontend
  - infra

### Plan
1. Review and harden AGENTS guidance with ANONCHAT-specific constraints.
2. Generate Android prompt library and then rework it into ANONCHAT-first prompts.
3. Add a progress tracking standard and create the initial tracker file.

### Progress Log
- 00:00 Reworked AGENTS from mixed/generic guidance to ANONCHAT-specific boundaries and contracts.
- 00:00 Added step-by-step frontend prompt templates in AGENTS with event and websocket compatibility constraints.
- 00:00 Created ANDROID_APP_PROMPTS.md and then replaced generic tracks with ANONCHAT-specific playbook.
- 00:00 Added quickstart copy-ready Android prompts in execution order.
- 00:00 Added Work Progress Tracking File section in AGENTS with required format and update rules.
- 00:00 Created ANONCHAT_WORK_PROGRESS.md starter session.

### Validation Performed
- file_search c:/Projects/playground/anonchat/ANONCHAT_WORK_PROGRESS.md -> pass - confirmed tracker file did not exist before creation.
- read_file AGENTS.md (progress-tracking section) -> pass - confirmed required template and rules are present.
- grep_search AGENTS.md and ANDROID_APP_PROMPTS.md anchors -> pass - confirmed inserted sections exist in expected locations.
- build/test commands -> not run - task was documentation-only and did not modify runtime code.

### Risks / Blockers
- None for documentation updates.
- Residual risk: future code changes may drift from docs if tracker is not updated per session.

### Next Actions
1. Update this file at each non-trivial task step going forward.
2. Add backend-focused step-by-step prompt section in AGENTS for proxy/router/security-core if requested.
3. Periodically run a docs-consistency pass against README, DEPLOYMENT, and runtime files.

### Final Outcome
- Status: complete
- Summary: Progress-tracking process is now documented in AGENTS and initialized with a first session in ANONCHAT_WORK_PROGRESS.md.

## Session: 2026-07-24 00:00 UTC - Project Implementation Review

### Task Summary
- Reviewed ANONCHAT implementation status across backend, security core, Tauri bridge, frontend, and local infra.
- Documented completed parts in a single progress entry with file-based evidence.

### Scope
- Files:
  - README.md
  - docker-compose.yml
  - server/proxy/main.go
  - server/router/main.go
  - client/src-tauri/src/ipc_bridge.rs
  - client/frontend/src/main.ts
  - client/frontend/src/chat.ts
  - client/frontend/src/globe.ts
  - client/security-core/src/onion/cipher.rs
  - client/security-core/src/panic_destruction.rs
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - proxy
  - router
  - security-core
  - tauri
  - frontend
  - infra

### Plan
1. Review core runtime files and local-run docs.
2. Compile completed implementation parts by subsystem.
3. Record findings and evidence in the progress tracker.

### Progress Log
- 00:00 Reviewed local stack documentation and compose topology (Redis RAM, router, guard/middle/exit services).
- 00:00 Reviewed proxy service implementation: role-based behavior, TLS listener, SOCKS5 guard handshake, middle relay, exit forwarding, and bi-directional tunnel flow.
- 00:00 Reviewed router implementation: websocket endpoint, session ack envelope, heartbeat timers, online routing, offline Redis TTL buffering, and replay deletion flow.
- 00:00 Reviewed Tauri IPC bridge implementation: broadcast buses, `start_proxy_event_stream`, forwarding of `proxy-layer-changed` and `core-log` events.
- 00:00 Reviewed frontend implementation: event listeners, chain status UI updates, ephemeral chat panel messages with TTL removal, and globe arc/node rendering.
- 00:00 Reviewed security-core implementation: fixed-size 4096-byte onion packet construction with tests, and panic destruction pipeline with zeroization and multi-pass file overwrite.
- 00:00 Appended completed-parts review to ANONCHAT_WORK_PROGRESS.md.

### Validation Performed
- read_file README.md -> pass - confirmed documented local stack startup and websocket message envelope examples.
- read_file docker-compose.yml -> pass - confirmed infra services for redis-ram, router, and three proxy roles are defined.
- read_file server/proxy/main.go -> pass - confirmed guard/middle/exit flow, TLS setup, and tunnel relay logic exist.
- read_file server/router/main.go -> pass - confirmed heartbeat timings, session ack, offline TTL storage, replay delivery, and cleanup logic exist.
- read_file client/src-tauri/src/ipc_bridge.rs -> pass - confirmed Tauri event stream command and event forwarding implementation exists.
- read_file client/frontend/src/main.ts -> pass - confirmed listeners for `proxy-layer-changed` and `core-log` and UI wiring exist.
- read_file client/frontend/src/chat.ts -> pass - confirmed ephemeral TTL message behavior and timed removal logic exist.
- read_file client/frontend/src/globe.ts -> pass - confirmed chain arc and node rendering pipeline exists.
- read_file client/security-core/src/onion/cipher.rs -> pass - confirmed 4096-byte packet invariant implementation and unit tests exist.
- read_file client/security-core/src/panic_destruction.rs -> pass - confirmed zeroization plus three-pass overwrite pipeline exists.
- runtime test/build commands -> not run - this task was a documentation review and did not modify runtime code paths.

### Risks / Blockers
- Review confirms many core components are implemented, but this entry is static code/document inspection only.
- End-to-end runtime verification is still needed to confirm current behavior in a live environment.

### Next Actions
1. Run subsystem validation commands from AGENTS for proxy, router, security-core, and frontend.
2. Execute README online/offline websocket smoke scenario and record pass/fail results in a new session entry.
3. Add a concise completed-vs-pending matrix if roadmap milestones are provided.

### Final Outcome
- Status: complete
- Summary: Completed implementation parts were reviewed and documented across infra, backend, Tauri bridge, frontend, and security-core modules.

## Session: 2026-07-24 00:00 UTC - Component Checklist Creation

### Task Summary
- Reviewed ANONCHAT project implementation coverage.
- Created a dedicated checklist file capturing completed components, partial components, and missing components.

### Scope
- Files:
  - ANONCHAT_COMPONENT_CHECKLIST.md
  - ANONCHAT_WORK_PROGRESS.md
  - README.md
  - DEPLOYMENT.md
  - docker-compose.yml
  - server/proxy/main.go
  - server/router/main.go
  - client/security-core/src/lib.rs
  - client/security-core/src/device_bootstrap.rs
  - client/security-core/src/onion/cipher.rs
  - client/security-core/src/panic_destruction.rs
  - client/src-tauri/src/lib.rs
  - client/src-tauri/src/events.rs
  - client/src-tauri/src/ipc_bridge.rs
  - client/frontend/src/main.ts
  - client/frontend/src/chat.ts
  - client/frontend/src/globe.ts
  - client/frontend/src/terminal.ts
- Subsystems:
  - proxy
  - router
  - security-core
  - tauri
  - frontend
  - infra

### Plan
1. Inspect code and docs for implemented components and integration gaps.
2. Classify each component as done, partial, or missing with evidence.
3. Create checklist file and record follow-up priorities.

### Progress Log
- 00:00 Reviewed topology and local/dev docs (README, compose, deployment guide).
- 00:00 Reviewed Go proxy and router implementations for core behavior coverage.
- 00:00 Reviewed Rust security-core, Tauri bridge, and frontend event/rendering modules.
- 00:00 Performed gap scan for tests, CI workflows, missing scripts, and missing integration implementations.
- 00:00 Created ANONCHAT_COMPONENT_CHECKLIST.md with status legend, evidence, and priority next build items.

### Validation Performed
- read/list/search commands across anonchat source tree -> pass - evidence collected for each checklist item.
- file_search for Go tests, frontend tests, scripts folder, and CI workflows -> pass - identified current gaps.
- runtime build/test commands -> not run - task was repository review and checklist documentation only.

### Risks / Blockers
- Checklist reflects static repository state at review time and may drift as code changes.

### Next Actions
1. Keep checklist updated after each major implementation session.
2. Execute validation commands and annotate checklist items with runtime verification evidence.
3. Start with priority items listed in ANONCHAT_COMPONENT_CHECKLIST.md.

### Final Outcome
- Status: complete
- Summary: Added a project checklist file that documents implemented, partial, and missing ANONCHAT components with concrete file evidence.

## Session: 2026-10-23 - Frontend Router Chat Integration

### Task Summary
- Added missing frontend chat pieces from AGENTS Prompt 4: router websocket client, composer UI, connection states.

### Scope
- Files:
  - client/frontend/src/ws.ts (new), chat.ts, main.ts, vite-env.d.ts (new), index.html, globe.ts
  - client/src-tauri/tauri.conf.json (CSP connect-src)
- Subsystems:
  - frontend
  - tauri

### Plan
1. Add RouterClient with reconnect and state machine.
2. Add composer and status UI to ChatPanel.
3. Wire in main.ts, allow ws in CSP, build.

### Progress Log
- Implemented ws.ts, composer, states (connecting/connected/degraded/reconnecting/failed), retry button.
- Removed invalid pointLabel call in globe.ts that broke the build.

### Validation Performed
- cd client/frontend && npm run build -> pass - chunk size warning only
- manual online/offline/reconnect checks -> not run
- frontend automated tests -> not added - no test dependency allowed without approval

### Risks / Blockers
- Router URL defaults to ws://127.0.0.1:8080/ws; other hosts need VITE_ROUTER_WS_URL and a CSP update.
- Chat is not yet routed through the onion chain or end-to-end encrypted.

### Next Actions
1. Approve a test runner (e.g. vitest) to cover parseIncoming and TTL behavior.

### Final Outcome
- Status: partial
- Summary: Live chat UI wired to router; tests pending.

## Session: 2026-10-26 12:56 EET - Local docker-compose.yml review and hardening

### Task Summary
- Verify the local compose file (Redis RAM, one chat router, guard/middle/exit proxies) against AGENTS.md and fix gaps found in review.

### Scope
- Files:
  - docker-compose.yml
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra

### Plan
1. Compare compose env vars with server/proxy/main.go and server/router/main.go.
2. Remove Redis host port, add Redis healthcheck and health-gated router start, drop obsolete version key.
3. Set explicit NEXT_HOP_ADDR and TLS_INSECURE_SKIP_VERIFY for dev proxy chain.
4. Validate with docker compose.

### Progress Log
- 12:56 Confirmed env vars used by code: NODE_ROLE, LISTEN_PORT, NEXT_HOP_ADDR, TLS_CERT_FILE, TLS_KEY_FILE, TLS_INSECURE_SKIP_VERIFY, REDIS_URL, PORT.
- 12:58 Rewrote docker-compose.yml: Redis stays memory only (no save, no AOF, no volume) and is no longer published to the host; healthcheck added; router waits for healthy Redis; proxies have explicit hop addresses and start-order dependencies; guard and middle skip verification of generated self-signed certs (dev only).

### Validation Performed
- docker compose config -q -> not run - docker is not installed in this environment
- docker compose up -d --build -> not run - docker is not installed in this environment
- go test ./... && go build ./... in server/proxy and server/router -> not run - go is not installed; Go sources were not changed

### Risks / Blockers
- Compose file is unverified at runtime until run on a machine with Docker.
- TLS_INSECURE_SKIP_VERIFY=true is for local development only; production must use real certificates (see DEPLOYMENT.md).

### Next Actions
1. Run `docker compose up -d --build` and confirm router connects to Redis and the guard-middle-exit chain accepts connections.

### Final Outcome
- Status: partial
- Summary: Points 1-4 applied; runtime verification pending because Docker is unavailable here.

---

## Session: 2026-10-26 13:10 +02:00 - Proxy hardening

### Task Summary
- Idle (renewing) timeout, silent failure exit codes, accept backoff, connection cap, exit private-destination blocking, tests.

### Scope
- Files: server/proxy/main.go, server/proxy/main_test.go, docker-compose.yml
- Subsystems: proxy, infra

### Plan
1. Edit proxy. 2. Add tests. 3. Allow private dest in dev compose exit.

### Progress Log
- 13:10 Implemented idleConn, MAX_CONNECTIONS, EXIT_ALLOW_PRIVATE (default blocked; dev compose sets true so exit can reach router), exit codes 1/2, accept backoff. No logging added.

### Validation Performed
- cd server/proxy && go test ./... && go build ./... -> not run - go is not installed here

### Risks / Blockers
- Code and tests are uncompiled/unrun.
- Production must leave EXIT_ALLOW_PRIVATE unset.

### Next Actions
1. Run go test and go build in server/proxy.

### Final Outcome
- Status: partial
- Summary: Changes made; validation pending (no Go toolchain).

## Session: 2026-10-26 13:40 EET - Router review fixes

### Task Summary
- Fix router defects found in review: double close of send channel, missing go.sum, no tests, ignored startup errors.

### Scope
- Files:
	- server/router/main.go
	- server/router/main_test.go
	- server/router/Dockerfile
- Subsystems:
	- router

### Plan
1. Replace close(send) with idempotent shutdown and done channel.
2. Log startup/serve errors without sensitive data.
3. Add miniredis based tests (TTL, DEL, replay, slow consumer).
4. Make Docker build generate go.sum via go mod tidy.

### Progress Log
- 13:40 All four steps implemented.

### Validation Performed
- cd server/router && go mod tidy && go test -race ./... && go build ./... -> not run - go and docker are not installed here

### Risks / Blockers
- go.sum not generated locally; Dockerfile runs go mod tidy as a fallback. Code and tests are uncompiled.

### Next Actions
1. Run go mod tidy, go test, go build in server/router and commit go.sum.

### Final Outcome
- Status: partial
- Summary: Fixes applied; validation pending (no Go toolchain).

## Session: 2026-10-26 13:55 +02:00 - Device bootstrap tests

### Task Summary
- Added unit tests for device_bootstrap (Ed25519 nickname signing, X3DH pre-key bundle, ciphertext-only storage).

### Scope
- Files:
  - client/security-core/src/device_bootstrap.rs
- Subsystems:
  - security-core

### Plan
1. Add in-memory store and test protector.
2. Test first run, reload, empty nickname, corrupted blob.
3. Run cargo test and cargo check.

### Progress Log
- 13:55 Added tests module with 4 tests.

### Validation Performed
- cargo test / cargo check -> not run - Rust toolchain not installed on this machine.

### Risks / Blockers
- libsignal-protocol is a git dependency on branch main (unpinned); API usage unverified by compiler.
- No concrete EncryptedDeviceMaterialStore / KeyMaterialProtector in app layer yet.

### Next Actions
1. Install Rust, run cargo test and cargo check, pin libsignal revision.
2. Implement concrete store/protector (keystore-backed AEAD).

### Final Outcome
- Status: partial
- Summary: Tests added; not compiled or executed.

## Session: 2026-10-26 13:56 +02:00 - Completing onion proxy flow

### Task Summary
- Complete the central proxy registry, Rust fetch-to-encrypt flow, focused tests, and documentation.

### Scope
- Files:
  - server/router/main.go
  - server/router/main_test.go
  - client/security-core/src/onion/mod.rs
  - client/security-core/src/onion/node_selector.rs
  - client/security-core/src/lib.rs
  - docker-compose.yml
  - README.md
- Subsystems:
  - router
  - security-core
  - infra

### Plan
1. Add an active proxy registry endpoint backed by explicit runtime configuration.
2. Add a Rust end-to-end fetch/select/encrypt entrypoint and focused tests.
3. Document configuration, endpoint shape, and validation.
4. Run the narrowest available tests and record blockers.

### Progress Log
- 13:56 Inspected existing router, Rust onion modules, compose configuration, and tests.
- 13:58 Added `GET /api/proxies/active` backed by the `ACTIVE_PROXIES` runtime configuration.
- 13:59 Added the Rust `fetch_and_encrypt_chain` end-to-end entrypoint and node-selection tests.
- 14:00 Documented registry configuration and the 4096-byte packet flow in README and compose.
- 14:01 Wired the Rust flow through the Tauri `build_onion_packet_from_registry` command.

### Validation Performed
- `go test ./...` -> not run - Go toolchain is unavailable in the environment.
- `cargo test` -> not run - Cargo toolchain is unavailable in the environment.
- Static symbol/configuration scan -> pass - endpoint, Rust entrypoint, tests, and docs are present.
- Tauri command registration scan -> pass - command is registered in the application invoke handler.

### Risks / Blockers
- Go and Rust toolchains are unavailable in this environment, so compilation and tests remain unverified.
- The registry requires deployment to provide real active-node metadata and X25519 public keys through `ACTIVE_PROXIES`.
- The final transport send path remains in the existing proxy/transport layer; this command produces the encrypted packet.

### Next Actions
1. Run `go test ./...` in `server/router` on a Go-enabled machine.
2. Run `cargo test` and `cargo check` in `client/security-core` on a Rust-enabled machine.
3. Configure `ACTIVE_PROXIES` with at least three countries and expose the router endpoint through HTTPS.

### Final Outcome
- Status: partial
- Summary: missing registry, end-to-end Rust API, focused tests, and documentation were implemented; local validation is blocked by missing toolchains.

## Session: 2026-10-26 15:10 +02:00 - Hardening panic destruction

### Task Summary
- Complete the missing reliability details identified in `trigger_panic_destruction`.

### Scope
- Files:
  - client/security-core/src/panic_destruction.rs
  - USER_MANUAL.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - security-core

### Plan
1. Make database overwrite failures explicit and fail closed.
2. Keep three full CSPRNG overwrite passes with per-pass storage synchronisation.
3. Document the physical-storage limitation and validation status.

### Progress Log
- 15:10 Reworked the overwrite helper to operate on the opened database file, propagate I/O errors, and require all three passes to complete.
- 15:10 Added failure-status handling and non-sensitive operational error output.
- 15:10 Documented that file APIs cannot guarantee physical-sector destruction on SSD or copy-on-write storage.

### Validation Performed
- `cargo test` -> not run - Cargo is not installed in the environment.
- Static review -> pass - zeroization, three passes, CSPRNG generation, sync calls, and exit paths are present.

### Risks / Blockers
- SQLCipher is not currently a dependency or concrete storage implementation in the repository; the function receives the already-openable database path as an input.
- Physical-sector destruction remains impossible to guarantee portably through Rust file I/O alone.

### Next Actions
1. Run `cargo test` and `cargo check` on a machine with the Rust toolchain.
2. Wire the actual SQLCipher store's database path and invoke the panic routine from the application panic action.

### Final Outcome
- Status: partial
- Summary: overwrite error handling and documentation were completed; SQLCipher store wiring and hardware-level secure erase remain application/deployment responsibilities.

## Session: 2026-10-26 15:30 +02:00 - Completing Tauri event producers

### Task Summary
- Complete the missing producer side of the Tauri IPC bridge so successful Rust proxy-chain selection and core errors reach the frontend event stream.

### Scope
- Files:
  - client/security-core/src/onion/node_selector.rs
  - client/src-tauri/src/ipc_bridge.rs
  - server/router/main.go
  - README.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - security-core
  - tauri
  - router

### Plan
1. Extend registry node metadata with renderer coordinates.
2. Publish proxy-chain and structured log payloads from the real Rust onion flow.
3. Validate frontend compatibility and record toolchain blockers.

### Progress Log
- 15:30 Added backward-compatible latitude/longitude fields to Rust proxy nodes.
- 15:31 Converted the selected security-core chain into the existing Tauri proxy event payload.
- 15:31 Published success and failure log entries through the existing log bus.
- 15:32 Added coordinate fields to the router registry response and documented the event behavior.

### Validation Performed
- `npm run build` in `client/frontend` -> pass - frontend event contracts remain type-correct.
- `cargo check` in `client/src-tauri` -> not run - Cargo is not installed in the environment.
- `go test ./...` in `server/router` -> not run - Go toolchain was not available in the environment.
- Static symbol scan -> pass - the producer now calls both event bus `publish` methods from the real onion packet command.

### Risks / Blockers
- Existing registry entries without coordinates deserialize as `(0, 0)` for compatibility; deployments should provide real coordinates.
- Runtime end-to-end event delivery remains unverified until Rust and Go toolchains are available.

### Next Actions
1. Run Rust and Go tests/builds on a toolchain-enabled machine.
2. Configure `ACTIVE_PROXIES` with real coordinates and invoke `build_onion_packet_from_registry` while the event stream is active.

### Final Outcome
- Status: partial
- Summary: The missing Rust producer integration is implemented; compilation and live event delivery remain blocked by unavailable local toolchains.

## Session: 2026-10-27 11:45 +02:00 - Go proxy registry refresh

### Task Summary
- Add the example proxy-discovery logic in the project runtime using Go rather than Python so the router can populate and refresh a live proxy registry in Redis.

### Scope
- Files:
	- server/router/main.go
	- server/router/main_test.go
	- ANONCHAT_WORK_PROGRESS.md
- Subsystems:
	- router
	- infra

### Plan
1. Add source parsing and validation for live proxy lists in the router.
2. Probe reachable SOCKS endpoints, attach metadata, and store the result in Redis.
3. Reuse the stored registry through the existing active-proxy API.
4. Run the focused router test suite to verify behavior.

### Progress Log
- 11:45 Inspected the router registry endpoint and confirmed the insert path existed without any live refresh mechanism.
- 11:46 Implemented Go proxy source fetching, normalization, runtime validation, and Redis writeback using the same operational pattern as the provided script.
- 11:47 Added the router refresh loop and Redis-backed `anonchat:proxies` key so `/api/proxies/active` can serve live data.
- 11:48 Added a regression test covering source fetch, successful validation, and Redis persistence.
- 11:49 Verified the router suite passes with `cd server/router && go test ./...`.

### Validation Performed
- `cd /home/mayvena/anonchat/server/router && go test ./...` -> pass - router tests succeeded, including the new proxy registry refresh regression.

### Risks / Blockers
- Public proxy sources can be flaky or rate-limited; the refresh is best-effort and will degrade gracefully to an empty registry when no live nodes are reachable.
- The implementation does not persist proxy secrets; it only stores generated metadata and a random public key placeholder for registry use.

### Next Actions
1. Wire a production-safe proxy source list and real node metadata into deployment configuration.
2. Replace placeholder key material with the actual node public keys in deployment-managed `ACTIVE_PROXIES` data when the live network is configured.

### Final Outcome
- Status: complete
- Summary: The router now refreshes a live proxy registry in Go, stores it in Redis memory, and exposes it through the existing active-proxy API without using Python.

## Session: 2026-10-26 - Deployment readiness and local validation

### Task Summary
- Proceed with the local deployment plan, validate available automated checks, and identify blockers without making unapproved privileged changes.

### Scope
- Files:
  - docker-compose.yml (inspection only)
  - .gitignore
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra
  - frontend
  - proxy
  - router
  - security-core

### Plan
1. Check local runtime permissions and repository deployment safeguards.
2. Run frontend install/build validation and determine remaining test blockers.
3. Start the stack only if Docker access and deployment settings permit it.

### Progress Log
- 16:00 Confirmed Debian 13 x86_64, Docker/Compose installed, but Docker daemon access denied and non-interactive sudo unavailable.
- 16:00 Confirmed Go and Rust toolchains are missing; Node/npm are available.
- 16:05 Added ignore rules for deployment secrets/overrides and generated frontend dependencies/build output.
- 16:06 Installed frontend dependencies with `npm ci`; `npm run build` passed with a large-chunk warning.
- 16:07 Production dependency audit found no vulnerabilities; full install reported two advisories (moderate/high).
- 16:15 Installed Go 1.27.3 and Rust/Cargo 1.99.1 under the user's home directory after verifying official installer checksums.
- 16:16 Updated `~/.profile` so user-local Go and Cargo binaries are on login-shell PATH; fresh login-shell verification passed.
- 16:17 Go proxy tests and build passed.
- 16:19 Installed checksum-verified `protoc` 36.2 under the user's home directory after Rust dependency compilation identified it as required.
- 16:20 Router tests and build passed after `go mod tidy` added the existing miniredis test dependency and generated `go.sum`.
- 16:22 Rust dependency build passed its protoc step but security-core compilation failed with 17 existing source/API type errors in `device_bootstrap.rs`.
- 19:40 Verified Docker group membership; current terminal required `sg docker` because its supplementary groups were stale.
- 19:41 Compose config validation and image build passed; all five services started successfully.
- 19:42 Verified Redis persistence disabled, router WebSocket returned HTTP 101, and guard completed a TLS 1.3 handshake with the development self-signed certificate.
- 19:43 Recent service logs were empty; all services report running and Redis healthy.

### Validation Performed
- `docker compose config --quiet` -> pass - Compose syntax validated without starting services.
- `docker info` -> blocked - current account cannot access `/var/run/docker.sock`.
- `cd client/frontend && npm ci && npm run build` -> pass - TypeScript and Vite production build succeeded; bundle-size warning remains.
- `cd client/frontend && npm audit --omit=dev` -> pass - no production dependency vulnerabilities found.
- `docker compose ps` -> blocked - current account cannot access Docker daemon.
- `cd server/proxy && go test ./... && go build ./...` -> pass - proxy tests and build succeeded.
- `cd server/router && go mod tidy` -> pass - resolved missing checksums and included existing miniredis test import.
- `cd server/router && go test ./... && go build ./...` -> pass.
- `cd client/security-core && cargo test && cargo check` -> fail - after installing protoc, compilation reports 17 type/API errors in `src/device_bootstrap.rs`; tests did not run.
- `bash -lc 'go version && rustc --version && cargo --version && protoc --version'` -> pass - all user-local toolchains are visible in a fresh login shell.
- `sg docker -c 'cd /home/mayvena/anonchat && docker compose config --quiet'` -> pass.
- `sg docker -c 'cd /home/mayvena/anonchat && docker compose build --pull'` -> pass - four application images built.
- `sg docker -c 'cd /home/mayvena/anonchat && docker compose up -d'` -> pass - all five services started; Redis healthy.
- `sg docker -c 'cd /home/mayvena/anonchat && docker compose ps'` -> pass - all five services up.
- Redis `CONFIG GET save` / `CONFIG GET appendonly` -> pass - save rules empty, appendonly `no`.
- Router WebSocket upgrade smoke test -> pass - HTTP 101.
- `timeout 8 openssl s_client -connect localhost:9001 -brief` -> pass - TLS 1.3 handshake; certificate is self-signed as expected for local development.
- Recent container logs -> pass - no output/errors observed.

### Risks / Blockers
- Compose has the five-service development stack running. The current terminal still needs a new login or `newgrp docker` to inherit Docker group membership; `sg docker -c ...` works immediately.
- Compose enables insecure TLS verification skipping and private-address exit access and publishes router port 8080; do not expose this development configuration as production.
- Guard TLS uses an ephemeral self-signed certificate; public deployment requires reviewed trusted certificates and production hardening.
- Security-core Rust build remains blocked by source/API type errors in `device_bootstrap.rs`; deployment container build does not compile that client subsystem.

### Next Actions
1. Start a fresh login session so Docker group membership applies without `sg`.
2. Resolve security-core Signal API and `rand_core` type errors before Rust tests can pass.
3. Review production TLS, private egress, router exposure, origin allowlist, firewall, and autostart before any public deployment.

### Final Outcome
- Status: partial
- Summary: Development Compose stack is built, running, and smoke-verified. Go proxy/router and frontend validations pass; security-core Rust source errors remain. Production hardening is not performed.

## Session: 2026-10-27 11:00 +02:00 - Fix plain-JSON chat transport bypass

### Task Summary
- Fix the first unresolved bug by making the chat send path invoke the Tauri onion packet builder instead of sending raw JSON through the router websocket.

### Scope
- Files:
  - client/frontend/src/main.ts
  - client/frontend/src/chat.ts
  - client/frontend/src/ws.ts
- Subsystems:
  - frontend
  - tauri

### Plan
1. Confirm the user-send path bypasses the onion packet builder and inspect the contract it is expected to honor.
2. Route outbound message sends through `build_onion_packet_from_registry` and encode the resulting packet into the websocket payload.
3. Preserve the existing UI connection flow and validate the frontend still builds cleanly.

### Progress Log
- 11:00 Confirmed the root cause: `chat.onSend` used `RouterClient.send` directly and never invoked the Tauri onion command.
- 11:02 Updated the send handler to build the onion packet before dispatching via the router websocket.
- 11:03 Made the chat panel support async send handlers without falling back to plaintext on transport errors.
- 11:04 Increased the websocket payload cap to cover the fixed-size encrypted packet flow while keeping the router contract unchanged.
- 11:05 Re-ran the focused frontend build and it succeeded with the onion-send changes.

### Validation Performed
- `cd /home/mayvena/anonchat/client/frontend && npm run build` -> pass - TypeScript and Vite production build succeeded after the fix.

### Risks / Blockers
- Full end-to-end runtime verification remains pending because the Tauri desktop runtime and downstream router endpoint were not exercised in this session.
- The browser path still depends on a valid proxy registry endpoint at `VITE_PROXY_REGISTRY_URL` or the default local HTTPS endpoint.

### Next Actions
1. Exercise the Tauri desktop app against a live router and proxy registry to confirm the encrypted payload is accepted end to end.
2. Revisit the remaining bug list starting with the router identity takeover issue after this fix is validated in the runtime path.

### Final Outcome
- Status: complete
- Summary: The frontend no longer sends message bodies as raw JSON through the router path; it now invokes the onion packet builder and only transmits the encrypted payload through the existing websocket flow.

## Session: 2026-10-27 09:29 EET - Debian 13 installer support

### Task Summary
- Extend the server prerequisite installer to support Debian 13 while retaining the existing Ubuntu support.

### Scope
- Files:
  - install-server-prereqs.sh
  - DEPLOYMENT.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra

### Plan
1. Select the matching Docker APT repository for supported distributions.
2. Keep the deployment guide's embedded installer in sync.
3. Validate shell syntax and installer-copy consistency.

### Progress Log
- 09:30 Added Debian 13 (Trixie) detection and Docker's Debian repository selection; retained Ubuntu repository selection.
- 09:30 Removed the Ubuntu-specific fallback deploy username so group assignment uses the invoking sudo user when available.
- 09:31 Updated the deployment guide's supported OS list, embedded installer, and invocation path.

### Validation Performed
- `bash -n install-server-prereqs.sh` -> pass - standalone installer parses successfully.
- `awk '!inside && /^```bash$/ { inside=1; next } inside && /^```$/ { exit } inside { print }' DEPLOYMENT.md | bash -n` -> pass - embedded guide installer parses successfully.
- `diff -u -w install-server-prereqs.sh <(sed -n '32,171p' DEPLOYMENT.md)` -> pass - installer logic matches the embedded guide copy, ignoring Markdown indentation.
- `git -c core.whitespace=cr-at-eol diff --check -- DEPLOYMENT.md ANONCHAT_WORK_PROGRESS.md` -> pass - recognizes the tracked CRLF line-ending convention.
- Privileged installer execution -> not run - the script upgrades system packages and changes Docker/sysctl configuration; syntax and documentation checks were used instead.

### Risks / Blockers
- The installer has not been run on a clean Debian 13 host in this session.

### Next Actions
1. Run the installer on a disposable Debian 13 VM before production deployment.

### Final Outcome
- Status: complete
- Summary: Installer and deployment documentation now support Debian 13 and preserve Ubuntu 22.04/24.04 support.

## Session: 2026-10-27 09:43 EET - Repair missing Caddy APT keyring

### Task Summary
- Prevent a configured Caddy Cloudsmith source with a missing signing key from blocking the prerequisite installer at its first APT update.

### Scope
- Files:
  - install-server-prereqs.sh
  - DEPLOYMENT.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra

### Progress Log
- 09:43 Added a conditional repair that fetches the official Caddy signing key only when the matching source is configured and its expected keyring is absent.
- 09:43 Synchronized the guide's embedded installer with the standalone script.

### Validation Performed
- `bash -n install-server-prereqs.sh` -> pass.
- Embedded guide installer `bash -n` -> pass.
- `diff -u -w` between standalone and embedded installer -> pass.
- Live installer execution -> not run - privileged package upgrades and system configuration changes are outside safe validation here.

### Risks / Blockers
- The Caddy repair path was not exercised against a live configured Caddy APT source in this workspace.

### Final Outcome
- Status: complete
- Summary: A missing Caddy keyring is repaired from Caddy's Cloudsmith signing-key endpoint before APT metadata refresh; unrelated APT sources are not modified.

## Session: 2026-10-27 10:12 EET - TLS Compose override automation

### Task Summary
- Automate creation of the local Docker Compose TLS override during certificate generation.

### Scope
- Files:
  - scripts/generate-tls-certs.sh
  - DEPLOYMENT.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra

### Plan
1. Generate a safe, repeatable Compose override for TLS-enabled proxy services.
2. Replace manual override instructions with the script workflow and correct router TLS guidance.
3. Validate shell behavior and merged Compose configuration in an isolated copy.

### Progress Log
- 10:12 Updated the generator to create a project-root Compose override, mount proxy certificates read-only, set proxy TLS paths, and preserve non-generated overrides.
- 10:12 Updated the deployment guide to use the generator and clarify that the current router does not load TLS certificates.

### Validation Performed
- `bash -n scripts/generate-tls-certs.sh` -> pass - generator shell syntax is valid.
- Isolated generation plus `docker compose config --format yaml` -> pass - all three proxy TLS paths and read-only mounts merge correctly; no unused router certificate is generated.
- Isolated second generation -> pass - the generated override is accepted on rerun.
- Isolated custom override check -> pass - non-generated content is left intact and generation stops with an error.

### Risks / Blockers
- The generated override is intended for the repository's local single-host Compose stack; a custom override must be merged manually.

### Next Actions
1. Replace self-signed certificates with trusted CA-issued files before public production exposure.

### Final Outcome
- Status: complete
- Summary: Certificate generation now creates the proxy TLS Compose override automatically, while preserving user-created overrides and matching the deployment guide to actual router behavior.

## Session: 2026-10-27 10:30 EET - Extract deployment scripts

### Task Summary
- Move standalone deployment and Linux client Bash scripts from DEPLOYMENT.md into scripts/ and review adjacent deployment instructions.

### Scope
- Files:
  - scripts/configure-firewall.sh
  - scripts/install-client-prereqs-linux.sh
  - scripts/build-client-linux.sh
  - scripts/generate-tls-certs.sh
  - DEPLOYMENT.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra
  - client build

### Plan
1. Extract embedded Bash scripts into standalone files and update guide invocations.
2. Review deployment safety details and correct material mismatches.
3. Validate shell syntax, Compose certificate access, guide structure, and whitespace.

### Progress Log
- 10:30 Extracted the firewall, Linux client prerequisite, and Linux client build scripts; existing server prerequisite and certificate generation scripts remain standalone.
- 10:30 Replaced duplicate script bodies in DEPLOYMENT.md and restored the Windows/macOS instructions and production hardening table.
- 10:30 Made SSH port configurable in the UFW script and documented that it resets existing rules.
- 10:30 Adjusted generated TLS key permissions to 0640 and Compose supplementary groups so unprivileged proxy containers can read keys without world access.
- 10:30 Corrected the Systemd reload shell invocation and documented the actual TLS verification setting.

### Validation Performed
- `bash -n scripts/*.sh` -> pass - all five scripts parse.
- Isolated `docker compose config --format yaml` and key permission checks -> pass - all three certificate mounts are read-only and the key is group-accessible only to the owner's primary group.
- Guide structure checks -> pass - all 12 numbered sections remain and no standalone Bash script bodies are embedded.
- `git -c core.whitespace=cr-at-eol diff --check -- DEPLOYMENT.md scripts` -> pass.

### Risks / Blockers
- The firewall script still intentionally resets UFW rules; deployers must confirm SSH access/port and review exposed ports before execution.
- Linux prerequisite installation and client builds were not run because they alter the host or require full build dependencies.

### Next Actions
1. Review the firewall port policy for the target single-host or multi-server deployment before running it.

### Final Outcome
- Status: complete
- Summary: Deployment script bodies are extracted into scripts/ and the guide now points to them; script syntax and generated TLS Compose access were validated.

## Session: 2026-10-27 10:56 EET - Master deployment script

### Task Summary
- Add a project-root orchestrator for sequential server deployment setup.

### Scope
- Files:
  - deploy.sh
  - DEPLOYMENT.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra

### Plan
1. Order the prerequisite, certificate, and firewall scripts with correct privilege boundaries.
2. Require confirmation before host changes and allow SSH port configuration.
3. Validate cancellation and accepted paths in an isolated project copy.

### Progress Log
- 10:56 Added executable `deploy.sh` at the project root.
- 10:56 The master script requires an explicit confirmation, runs prerequisites and firewall setup via sudo, generates keys unprivileged, validates `SSH_PORT`, and prints next Compose actions.
- 10:56 Documented the master command in DEPLOYMENT.md.

### Validation Performed
- `bash -n deploy.sh` -> pass.
- `printf 'no\\n' | ./deploy.sh` -> pass - cancellation exits before running component scripts.
- Isolated accepted run with stubbed sudo -> pass - prerequisite, certificate, and firewall steps execute sequentially; `SSH_PORT` is forwarded and TLS key mode remains 0640.
- Host package installation and firewall configuration -> not run - those actions modify system state; the isolated stub test validated orchestration without host changes.

### Risks / Blockers
- The firewall component intentionally resets UFW rules; deployers must confirm SSH port and console access before confirming the master script.
- The master script does not build or start Docker services; it prints those follow-up Compose commands.

### Next Actions
1. Review the firewall exposure policy for the target deployment before running `./deploy.sh` on a host.

### Final Outcome
- Status: complete
- Summary: Added a confirmed, ordered server setup orchestrator with unprivileged TLS key generation and configurable SSH firewall access.

## Session: 2026-10-27 11:10 EET - Combined local startup script

### Task Summary
- Add one root command to start the local Compose backend and frontend dev server.

### Scope
- Files:
  - start.sh
  - README.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - infra
  - frontend

### Plan
1. Verify backend and frontend startup commands and prerequisites.
2. Add a root-level startup script with ordered startup and clear shutdown guidance.
3. Validate shell, Compose configuration, and command order without leaving services running.

### Progress Log
- 11:10 Added executable `start.sh`; it checks Docker Compose/npm, installs frontend dependencies when absent, starts the backend stack, then runs Vite on port 1420.
- 11:10 Documented the combined command and backend stop command in README.md.

### Validation Performed
- `bash -n start.sh` -> pass.
- `docker compose config -q` -> pass.
- Isolated docker/npm stub test -> pass - backend Compose startup precedes frontend Vite, using the expected frontend working directory.
- Live stack startup -> not run - this would leave Docker services running in the user's environment.

### Risks / Blockers
- The script intentionally leaves backend containers running after Ctrl+C; README documents `docker compose down` as the separate cleanup command.

### Next Actions
1. Run `./start.sh` locally to use the combined development workflow.

### Final Outcome
- Status: complete
- Summary: Added and documented a root startup script for the local backend and frontend dev server.
