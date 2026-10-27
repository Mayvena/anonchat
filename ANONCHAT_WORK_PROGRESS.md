# ANONCHAT Work Progress

## Session: 2026-10-28 - Tauri Feature Config Mismatch

### Task Summary
- Resolve the Tauri build-script failure reporting `protocol-asset` was not enabled in `tauri.conf.json`.

### Scope
- Files:
  - client/src-tauri/Cargo.toml
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - tauri

### Progress Log
- Confirmed `protocol-asset` was enabled only in Cargo, while `app.security.assetProtocol.enable` remained false and the frontend did not use `convertFileSrc` or the asset protocol.
- Removed the unused `protocol-asset` feature rather than widening asset-protocol access.
- Reran `cargo check`; the feature/config mismatch did not recur, and compilation proceeded to GTK system-library detection on Linux.

### Validation Performed
- `cargo check` (from `client/src-tauri`) -> blocked by missing `pkg-config` and GTK/GIO development libraries on this Linux host; no Tauri feature/config mismatch reported.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass.

### Risks / Blockers
- The Windows-native check must still be rerun to validate the full Windows target; Linux cannot validate the MSVC/WebView2 build.
- `security-core/src/device_bootstrap.rs` has a non-fatal `unused_mut` warning unrelated to this failure.

### Next Actions
1. Rerun `cargo check` from `client\src-tauri` on Windows.
2. Address any subsequent Windows compiler or linker errors.

### Final Outcome
- Status: partial
- Summary: Removed the root cause of the reported Tauri feature/config mismatch; local compilation is blocked later by missing Linux GTK prerequisites.

## Session: 2026-10-28 - Tauri Cargo Context Generation

### Task Summary
- Fix `cargo check` failing because `tauri::generate_context!()` had no `OUT_DIR`.

### Scope
- Files:
  - client/src-tauri/Cargo.toml
  - client/src-tauri/build.rs
  - client/src-tauri/src/ipc_bridge.rs
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - tauri

### Progress Log
- Confirmed the Tauri crate had no build script or `tauri-build` dependency, so Cargo did not generate the build-time context required by `generate_context!()`.
- Added `tauri-build` and `build.rs`, including a `check-cfg` declaration for the app's `mobile` cfg; removed the reported unused `Manager` import.
- `cargo check` now passes the original `OUT_DIR` macro failure and proceeds to Linux GTK/GIO native dependency detection, where this host lacks `pkg-config`.

### Validation Performed
- `cd client/src-tauri && cargo check` -> blocked after the fix by missing `pkg-config` and Linux GTK/GIO development libraries; the original `OUT_DIR` error did not recur.
- Editor diagnostics for `build.rs` and `src/lib.rs` -> pass - no errors reported.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass.

### Risks / Blockers
- Full app validation still needs a host with Tauri's native platform dependencies; for the reported Windows case, rerun from `client\src-tauri` on the Windows machine.

### Next Actions
1. On Windows, run `cargo check` in `client\src-tauri` and confirm `generate_context!()` compiles.
2. Address any further Windows-specific compiler or linker errors revealed after the context is generated.

### Final Outcome
- Status: partial
- Summary: Added the standard Tauri build hook; validation passed the reported failure and is now blocked only by this Linux host's missing GTK build prerequisites.

## Session: 2026-10-27 16:15 EET - Windows Desktop Endpoint Setup

### Task Summary
- Begin the Windows desktop client implementation with configurable remote router and proxy-registry endpoints.

### Scope
- Files:
  - client/frontend/src/main.ts
  - client/frontend/index.html
  - client/src-tauri/tauri.conf.json
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend
  - tauri

### Plan
1. Add endpoint validation, a first-run settings dialog, and non-secret endpoint persistence.
2. Connect the Tauri client to the configured remote services and allow HTTPS/WSS in the app CSP.
3. Run focused frontend validation and record Windows-specific remaining work.

### Progress Log
- 16:12 Confirmed the existing Windows desktop uses Tauri 2, while service defaults and CSP were localhost-only; found existing user edits and preserved them.
- 16:14 Added accessible endpoint settings, validation requiring WSS/HTTPS, local storage of endpoint URLs only, and router/route startup after configuration.
- 16:15 Updated CSP for secure remote connections and built the frontend successfully.

### Validation Performed
- `npm run build` (from `client/frontend`) -> pass - TypeScript check and Vite production build; existing Three.js chunk-size warning remains.
- Frontend diagnostics for `client/frontend/src/main.ts` -> pass - no errors reported.
- Windows-native Tauri build, portable ZIP packaging, and Windows runtime smoke tests -> not run - this workspace runs on Linux; a native Windows host/runner is required.

### Risks / Blockers
- The current CSP permits HTTPS and WSS connections by scheme because endpoint hosts are user-configurable; the app only requests the configured endpoints, but origin-specific policy would require a runtime CSP strategy or a native networking bridge.
- The Tauri native Windows build, WebView2 availability handling, portable archive creation, and Windows 10/11 behavior remain unverified.

### Next Actions
1. Add and exercise the Windows x64 build/portable packaging workflow on a native Windows runner.
2. Verify whether the chosen archive includes or documents the WebView2 runtime requirement.
3. Smoke-test configured remote WSS/HTTPS endpoints and encrypted send on clean Windows 10 and 11 machines.

### Final Outcome
- Status: partial
- Summary: First-run and editable secure endpoint configuration is implemented and frontend validation passes; native Windows packaging and runtime verification remain.

## Session: 2026-10-27 15:18 EET - Initial Globe Route Selection

### Task Summary
- Populate the globe route at startup instead of waiting for the first onion packet build.

### Scope
- Files:
  - client/frontend/src/main.ts
  - client/src-tauri/src/ipc_bridge.rs
  - client/src-tauri/src/lib.rs
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend
  - tauri

### Plan
1. Trace route event publication and distinguish it from static chat node entries.
2. Add startup route selection and publish the existing proxy-chain event.
3. Run focused frontend and Rust validation, then update the bug status.

### Progress Log
- 15:18 Confirmed the globe route event is only published during message packet construction; the chat node entries are placeholders.
- 15:18 Added a Tauri command to select and emit an initial route and invoked it after event listeners start.
- 15:19 Frontend build and standalone security-core check passed; the Tauri check was blocked by missing `pkg-config`/GTK libraries, and `cargo fmt --check` reported existing formatting differences in `ipc_bridge.rs`.
- 15:19 Recorded bug 14 as Blocked pending a host with native Tauri build prerequisites.

### Validation Performed
- `npm run build` (from `client/frontend`) -> pass - TypeScript check and Vite production build; existing Three.js chunk-size warning remains.
- `cd ../src-tauri && cargo check` (from `client/frontend`) -> blocked - stopped in `glib-sys` because `pkg-config` and GTK development libraries are unavailable.
- `cargo check` (from `client/security-core`) -> pass - validates the exported `fetch_and_select_chain` API; two pre-existing Rust warnings remain.
- `cargo fmt --check` (from `client/src-tauri`) -> fail - reports formatting differences in existing sections of `ipc_bridge.rs`; no formatter rewrite was applied to avoid unrelated formatting churn.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass - no whitespace errors.

### Risks / Blockers
- The startup route fetch requires the configured proxy registry to be reachable and to return at least three distinct countries. Without that, the route correctly remains pending.

### Next Actions
1. Run the Tauri check on a host with `pkg-config` and GTK development libraries installed.

### Final Outcome
- Status: blocked
- Summary: Initial route selection is wired to startup, but app-level Rust validation cannot complete without native Tauri build prerequisites.

## Session: 2026-10-27 15:05 EET - Globe Visible Before Route

### Task Summary
- Make the base globe visible while the router is connected and proxy-route data is still pending.

### Scope
- Files:
  - client/frontend/src/globe.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend

### Plan
1. Trace base globe rendering and the route-pending state.
2. Increase the base globe contrast without coupling rendering to route events.
3. Build the frontend and record the bug status.

### Progress Log
- 15:05 Confirmed the renderer draws continuously, but its near-black surface has no land data and blends into the black scene.
- 15:05 Increased the globe surface contrast and added a subtle emissive tint; recorded bug 13 as In Progress.
- 15:06 Frontend TypeScript check and production build passed; marked bug 13 Resolved.

### Validation Performed
- `npm run build` (from repository root) -> fail - the root has no `package.json`; this invocation did not validate the change.
- `cd client/frontend && npm run build` -> pass - TypeScript check and Vite production build; existing Three.js chunk-size warning remains.

### Risks / Blockers
- The globe remains intentionally featureless until route-specific node and arc data arrives; this change improves the visibility of the base sphere only.

### Next Actions
- None.

### Final Outcome
- Status: complete
- Summary: Base globe is now visibly separated from the black background while waiting for route data; frontend build passed.

## Session: 2026-10-27 14:28 EET - LAN Router WebSocket

### Task Summary
- Fix browser clients opened over LAN using a router WebSocket loopback address that points to the viewer device.

### Scope
- Files:
  - client/frontend/src/main.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend

### Plan
1. Confirm the frontend host binding and router port mapping.
2. Derive the default router WebSocket scheme and hostname from the current page while retaining the explicit environment override.
3. Build the frontend and record the result and final bug status.

### Progress Log
- Confirmed Vite binds to all interfaces and Docker publishes router port 8080, while the frontend default uses `127.0.0.1`.
- Changed the fallback router URL to use the page hostname with `ws` or `wss`; retained `VITE_ROUTER_WS_URL` overrides and documented LAN port reachability.
- Marked bug 12 Resolved after the frontend build passed.

### Validation Performed
- `npm run build` (from repository root) -> not run - the frontend package is located in `client/frontend`; a subsequent invocation from that package was used.
- `cd client/frontend && npm run build` -> not run - terminal was already in `client/frontend`, so this relative path was unavailable.
- `npm run build` (from `client/frontend`) -> pass - TypeScript check and Vite production build; existing large Three.js chunk warning remains.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass - no whitespace errors.
- Frontend diagnostics for `client/frontend/src/main.ts` -> pass - no errors reported.

### Risks / Blockers
- The router must remain reachable on port 8080 from the browser device; firewall and LAN isolation settings remain external prerequisites. HTTPS deployments require a TLS-terminating WebSocket endpoint configured through `VITE_ROUTER_WS_URL`.

### Next Actions
- None.

### Final Outcome
- Status: complete
- Summary: Browser clients now connect to the router on the frontend's host using the correct WebSocket scheme; frontend build passed.

## Session: 2026-10-27 14:17 EET - Globe Connection Status

### Task Summary
- Make the globe status reflect router connectivity while no proxy-chain snapshot has been received.

### Scope
- Files:
  - client/frontend/src/main.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend

### Plan
1. Trace router and proxy-chain status ownership.
2. Update the globe status from router state until a route snapshot arrives.
3. Build the frontend and record the result and final bug status.

### Progress Log
- Started investigation; confirmed the router state callback does not currently update the globe status.
- Updated router-state handling to display connecting/connected/degraded/reconnecting/failed status until a chain snapshot arrives; chain events continue to render route countries.
- Recorded bug 11 as Resolved after the focused frontend build passed.

### Validation Performed
- `npm run build` (from repository root) -> fail - the repository root has no `package.json`; this invocation did not validate the change.
- `cd client/frontend && npm run build` -> pass - TypeScript check and Vite production build; existing large Three.js chunk warning remains.

### Risks / Blockers
- The router being connected does not mean a proxy route has been selected; the status explicitly reports `ROUTE PENDING` until a chain event arrives.

### Next Actions
- None.

### Final Outcome
- Status: complete
- Summary: Globe status reflects router connection while awaiting a proxy route, then continues showing the active route; frontend build passed.

## Session: 2026-10-27 13:58 EET - Desktop Chat Launch

### Task Summary
- Provide a documented desktop launch path for encrypted chat after the standard browser launch rejects message sends.

### Scope
- Files:
  - start.sh
  - README.md
  - BUGS.md
  - client/frontend/package.json
  - client/frontend/package-lock.json
- Subsystems:
  - frontend
  - tauri

### Plan
1. Confirm the default launcher and Tauri IPC ownership.
2. Add an opt-in desktop launch mode while retaining browser preview.
3. Validate launcher syntax, frontend build, and desktop CLI configuration; update the bug record.

### Progress Log
- 13:55 Confirmed `./start.sh` runs Vite in a regular browser and the encrypted send path requires the Tauri bridge.
- 13:57 Added the local Tauri CLI dependency; verified `npm exec tauri -- --version` reports tauri-cli 2.12.4.
- 13:58 Added `./start.sh --desktop`, clarified browser preview limits, and recorded bug 10 as In Progress.
- 14:01 Corrected the CLI dependency location and Tauri working directory; the native CLI recognizes the app project.
- 14:02 Restored the existing `protocol-asset` feature after CLI build side effects and recorded the missing Linux build prerequisite.

### Validation Performed
- `npm exec tauri -- --version` -> pass - project-local Tauri CLI is executable.
- `npm exec tauri -- dev --help` -> pass - confirms the development command accepts a config path.
- `bash -n ../../start.sh` (from `client/frontend`) -> pass - desktop launcher parses.
- `npm run build` (from `client/frontend`) -> pass - TypeScript and Vite production build; existing large Three.js chunk warning remains.
- `../frontend/node_modules/.bin/tauri build --no-bundle` (from `client/src-tauri`) -> blocked - CLI recognizes the project, but Rust build stops because `pkg-config` is not installed; Linux GTK/WebKit development prerequisites are therefore unavailable.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass - current patch has no whitespace errors.
- `npm run build` from repository root -> not applicable - frontend has its own package directory; the successful build was run from `client/frontend`.

### Risks / Blockers
- Native desktop startup has not been verified because this host lacks `pkg-config` and likely the Linux GTK/WebKit development packages.

### Next Actions
1. Install Tauri's Linux build prerequisites on the host and run `./start.sh --desktop`.

### Final Outcome
- Status: partial
- Summary: Added a documented native desktop launch path and preserved browser preview mode; frontend build and launcher syntax pass, but native build validation is blocked by missing host packages.

## Session: 2026-10-27 13:39 EET - Optional Chat Recipient

### Task Summary
- Restore an optional recipient user ID field so chat sends privately when selected and to the room when blank.

### Scope
- Files:
  - client/frontend/index.html
  - client/frontend/src/chat.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend

### Plan
1. Confirm room and private-recipient routing contracts.
2. Add the optional recipient field and map blank input to the room destination.
3. Build the frontend and record the final bug status.

### Progress Log
- 13:39 Confirmed the router broadcasts the reserved `room` destination and privately routes other recipient IDs.
- 13:39 Added bug 8 for the missing recipient control and implemented the optional composer field with room fallback.
- 13:40 Frontend production build passed; reviewed generated output and patch formatting.

### Validation Performed
- `npm run build` (from repository root) -> fail - no root `package.json`; this did not validate the change.
- `cd client/frontend && npm run build` -> pass - TypeScript check and Vite production build; Vite reports the existing large Three.js chunk warning.
- Router room broadcast regression test -> not run - the router routing implementation is unchanged; frontend build is the focused validation for this UI change.

### Risks / Blockers
- None identified.

### Next Actions
- None.

### Final Outcome
- Status: complete
- Summary: Restored optional private recipient selection with room delivery as the blank-input fallback; frontend build passed.

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
## Session: 2026-10-27 00:00 UTC - Fix first unresolved bug: private-exit default deny

### Task Summary
- Remove the unsafe exit-proxy deployment default that allowed private/local destinations and align the compose stack with the proxy runtime's default-deny private-IP checks.

### Scope
- Files:
	- docker-compose.yml
	- BUGS.md
	- ANONCHAT_WORK_PROGRESS.md
- Subsystems:
	- infra
	- proxy

### Plan
1. Confirm the first unresolved bug and the owning config path.
2. Change the exit proxy deployment default to deny private-network destinations by default.
3. Validate the compose file and proxy test suite.
4. Update the bug log and progress tracker with the remediation evidence.

### Progress Log
- 00:00 Confirmed the first unresolved bug is the exit node's `EXIT_ALLOW_PRIVATE=true` compose setting, which overrides the proxy code's safe default deny behavior.
- 00:01 Updated the local deployment config to `EXIT_ALLOW_PRIVATE=false` so the exit node only allows public destinations unless the deployment explicitly opts in.
- 00:03 Ran the narrow validation commands for the proxy package and for Docker-compose parsing.

### Validation Performed
- cd /home/mayvena/anonchat/server/proxy && go test ./... -> pass - proxy package tests pass with the default-deny semantics unchanged.
- docker compose config -q -> pass - compose file parses cleanly with the exit proxy explicitly set to `EXIT_ALLOW_PRIVATE=false`.

### Risks / Blockers
- None. This fix is limited to the deployment default and preserves the proxy runtime contract.

### Next Actions
1. Review the next unresolved bug in BUGS.md in priority order.
2. Keep the deployment default explicit in future compose or environment overrides.

### Final Outcome
- Status: complete
- Summary: The first unresolved bug is fixed by enforcing default-deny private destination filtering in the exit proxy deployment configuration, and the bug record reflects the resolved state.
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

## Session: 2026-10-27 13:30 +00:00 - Security-core bootstrap fix and validation

### Task Summary
- Fix the first unresolved Rust bug by persisting and reusing the one-time pre-key in the device bootstrap envelope and by bringing the code into line with the current libsignal and rand compatibility contract.

### Scope
- Files:
	- client/security-core/src/device_bootstrap.rs
	- client/security-core/src/onion/cipher.rs
	- client/security-core/src/onion/node_selector.rs
	- client/security-core/src/panic_destruction.rs
	- client/security-core/Cargo.toml
	- BUGS.md
	- ANONCHAT_WORK_PROGRESS.md
- Subsystems:
	- security-core

### Plan
1. Confirm the root cause behind the missing one-time pre-key record and the Signal/rand API drift.
2. Persist the one-time key in the encrypted bootstrap envelope and rebuild the bundle from that stored record.
3. Replace incompatible direct `OsRng` calls with OS-entropy generation that matches the active library versions.
4. Validate the focused crate test suite and update the bug log with the final verified status.

### Progress Log
- 13:30 Confirmed the root cause: the bootstrap bundle was built from a fresh in-memory one-time key that was never saved, and the crate was also still calling legacy `rand`/`rand_core` APIs from older assumptions.
- 13:31 Persisted the serialized one-time pre-key into the encrypted bootstrap envelope and reloaded that key when constructing the bundle.
- 13:32 Switched the crate’s direct entropy calls to `getrandom`/matching APIs so they do not cross incompatible `rand_core` versions between libsignal and x25519.
- 13:33 Fixed the regression assertion to check the actual persisted protobuf record instead of looking for a raw numeric string inside the serialized JSON blob.
- 13:34 Re-ran the security-core test suite and it passed.

### Validation Performed
- `cd /home/mayvena/anonchat/client/security-core && cargo test -- --nocapture` -> pass - 15 unit tests passed, 1 regression test for persisted one-time pre-key reuse passed.

### Risks / Blockers
- None for the targeted bootstrap fix.
- Remaining warnings are non-blocking (`unused_mut` and an unnecessary unsafe block), but they are not required for this bug fix.

### Next Actions
1. Keep the bootstrap path and record serialization contract under regression coverage.
2. Decide whether to clean the remaining warnings in a follow-up pass if the project wants stricter `cargo clippy` or warning-only CI.

### Final Outcome
- Status: complete
- Summary: The security-core bootstrap now retains and reuses the one-time pre-key record, the crate compiles against the current Signal/rand-compatible API surface, and the targeted regression test passes.

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

## Session: 2026-10-27 - Router session identity takeover mitigation

### Task Summary
- Prevent websocket callers from selecting or reclaiming another user's router identity without its unguessable owner credential.

### Scope
- Files:
  - server/router/main.go
  - server/router/main_test.go
  - client/frontend/src/ws.ts
  - client/frontend/src/main.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - router
  - frontend

### Plan
1. Issue server-generated UUID identities and validate reconnect credentials before websocket upgrade.
2. Keep the credential in frontend memory and send it in a websocket subprotocol, not the URL.
3. Add takeover regression tests, run router/frontend validation, and update the bug record.

### Progress Log
- Confirmed `/ws` accepted `user_id` directly and `hub.register` replaced any existing connection under that ID.
- Added Redis-TTL ownership records containing only a SHA-256 digest; new sessions receive a random 256-bit credential and server-generated UUID.
- Added authenticated reconnect using `Sec-WebSocket-Protocol`; the frontend retains the returned ID and credential only in `RouterClient` memory.
- Added handler-level coverage for server-issued IDs, ignored caller IDs, missing/wrong credential rejection from an existing identity, and valid credential reconnect.
- Updated bug 2 to Fixed after the focused behavior checks passed.

### Validation Performed
- `cd /home/mayvena/anonchat/server/router && go test ./... && go build ./...` -> pass - router regression tests and build succeeded.
- `cd /home/mayvena/anonchat/client/frontend && npm run build` -> pass - TypeScript and Vite production build succeeded; existing large Three.js chunk warning remains.
- `git -c core.whitespace=cr-at-eol diff --check -- client/frontend/src/main.ts client/frontend/src/ws.ts server/router/main.go server/router/main_test.go` -> pass - touched source files pass whitespace validation with the repository's CRLF convention.
- Plain `git diff --check` -> reports CRLF carriage returns on newly edited frontend lines; no line-ending normalization was performed.

### Risks / Blockers
- Session credentials are bearer secrets: a party that obtains a credential can reconnect as that identity until its Redis ownership record expires. Credentials are not persisted by the frontend and are not logged.
- Redis session ownership lasts 30 days without renewal by the client; a later connection after expiry receives a new identity rather than reclaiming the expired one.
- No live browser/router deployment smoke test was run; router handler tests and production frontend build were used.

### Next Actions
1. Consider secure OS-backed persistence only if identity must survive a full client restart; the current contract intentionally keeps credentials in memory only.

### Final Outcome
- Status: complete
- Summary: New router sessions now receive unguessable server-issued identities, and reconnect requires the matching credential; arbitrary query IDs cannot claim or replace an existing session.

## Session: 2026-10-27 12:26 EET - Router Offline Delivery Acknowledgement

### Task Summary
- Keep TTL-backed offline messages available when a WebSocket write fails, and delete them only after a successful write.

### Scope
- Files:
  - server/router/main.go
  - server/router/main_test.go
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - router

### Plan
1. Trace Redis deletion and WebSocket write paths for the first unresolved bug.
2. Carry offline keys through queued sends and acknowledge them after successful writes.
3. Add focused regression tests, run router validation, and resolve the bug entry.

### Progress Log
- 12:26 Confirmed live and replay delivery deleted Redis keys when a message entered the buffered channel, before socket write completion.
- 12:26 Changed queued messages to retain their Redis key; successful writes now delete the key, while failed writes leave its TTL-backed record available for replay.
- 12:26 Added router coverage for successful-write deletion and failed-write retention; marked bug 4 Resolved.

### Validation Performed
- `cd /home/mayvena/anonchat/server/router && go test ./...` -> pass - focused router suite, including failed socket write retention.
- `cd /home/mayvena/anonchat/server/router && go test ./... && go build ./...` -> pass - router tests and build succeeded.
- `git -c core.whitespace=cr-at-eol diff --check -- server/router/main.go server/router/main_test.go` -> pass - Go source checked with the repository's CRLF convention.

### Risks / Blockers
- If Redis cannot store a message, the existing online best-effort delivery path remains available, but a failed socket write cannot be buffered during that Redis outage.

### Next Actions
1. None.

### Final Outcome
- Status: complete
- Summary: Offline message keys are retained until the router successfully writes their payload to the WebSocket; failed writes remain eligible for TTL replay.

## Session: 2026-10-27 - Shared room messaging

### Task Summary
- Let the frontend send messages to a shared live room without requiring a recipient user ID.

### Scope
- Files:
  - client/frontend/index.html
  - client/frontend/src/chat.ts
  - client/frontend/src/main.ts
  - server/router/main.go
  - server/router/main_test.go
  - README.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend
  - router

### Plan
1. Remove the per-user recipient requirement from the frontend and route sends to the shared room.
2. Fan room messages out to currently connected clients, excluding the sender, without offline storage.
3. Add router regression coverage, document the contract, and run router/frontend validation.

### Progress Log
- Confirmed the popup comes from the required recipient input; the current router only supports direct user delivery and has no room route.
- Agreed that room sends should reach connected clients only; room messages will not be buffered for offline users.
- Removed recipient ID from the chat form and routed frontend sends to the reserved `room` destination.
- Added router fan-out to connected clients, excluding the sender, with no Redis storage for room messages; documented the contract and added regression coverage.

### Validation Performed
- `cd /home/mayvena/anonchat/server/router && go test ./...` -> pass - room fan-out regression and existing router tests pass.
- `cd /home/mayvena/anonchat/server/router && go build ./...` -> pass - router builds with shared room routing.
- `cd /home/mayvena/anonchat/client/frontend && npm run build` -> pass - TypeScript and Vite production build succeeded; existing Three.js chunk-size warning remains.
- `git -c core.whitespace=cr-at-eol diff --check -- ANONCHAT_WORK_PROGRESS.md README.md client/frontend/index.html client/frontend/src/chat.ts server/router/main.go server/router/main_test.go` -> not confirmed - terminal output did not clearly show command completion.

### Risks / Blockers
- The shared room has no persistent membership or history; only clients connected at send time receive room messages.

### Next Actions
1. None.

### Final Outcome
- Status: complete
- Summary: The frontend no longer requires a recipient user ID and sends to a shared online-only room; direct-message TTL delivery remains unchanged.

## Session: 2026-10-27 13:45 EET - Browser-Mode Onion Send Error

### Task Summary
- Prevent browser-mode chat sends from calling Tauri IPC without an available Tauri runtime.

### Scope
- Files:
  - client/frontend/src/main.ts
  - BUGS.md
  - ANONCHAT_WORK_PROGRESS.md
- Subsystems:
  - frontend

### Plan
1. Confirm the onion-send path and browser/Tauri runtime boundary.
2. Add a runtime guard and clear user-facing log entry.
3. Build the frontend and record the bug's final status.

### Progress Log
- 13:45 Confirmed the send handler calls Tauri `invoke` without checking runtime, while browser mode is supported for initialization.
- 13:45 Added bug 9 as In Progress and guarded packet construction with a browser-mode error message.
- 13:46 Frontend TypeScript check and production build passed; marked bug 9 Resolved.

### Validation Performed
- `npm run build` (from repository root) -> fail - no root `package.json`; this did not validate the change.
- `cd client/frontend && npm run build` -> pass - TypeScript check and Vite production build; existing Three.js chunk-size warning remains.
- `git -c core.whitespace=cr-at-eol diff --check` -> pass - whitespace check with existing CRLF line endings recognized.

### Risks / Blockers
- Browser mode remains available for UI preview, but encrypted message sending requires the Tauri desktop runtime.

### Next Actions
1. None.

### Final Outcome
- Status: complete
- Summary: Browser-mode sends now fail before invoking Tauri IPC with an explicit explanation; frontend build passed.
