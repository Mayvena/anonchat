# ANONCHAT Bug Log

This document captures the confirmed issues found during the project review. It is intentionally brief and written to be actionable for follow-up fixes.

## 1. Chat traffic is sent in plain JSON instead of through the onion transport

- Status: Fixed
- Severity: High
- Affected area: client/frontend/src/main.ts, client/frontend/src/ws.ts, client/src-tauri/src/ipc_bridge.rs
- Issue:
  The normal chat flow sends message bodies through the router WebSocket client (`RouterClient.send`) and never calls the Tauri onion-packet command (`build_onion_packet_from_registry`). That means user messages are exchanged as plaintext JSON over the router path instead of using the designed encrypted proxy flow.
- Evidence:
  - The UI sends directly to the router: [client/frontend/src/main.ts](client/frontend/src/main.ts#L110-L115)
  - The WebSocket client emits raw JSON `{ id, to, body }`: [client/frontend/src/ws.ts](client/frontend/src/ws.ts#L121-L131)
  - The Tauri bridge exposes the onion packet command, but it is not used in the chat path: [client/src-tauri/src/ipc_bridge.rs](client/src-tauri/src/ipc_bridge.rs#L220-L247)
- Fix:
  The client send path now invokes `build_onion_packet_from_registry`, encodes the resulting fixed-size onion packet, and only sends the encrypted payload through the router websocket. The UI no longer falls back to plaintext JSON on transport failure.
- Impact:
  This restores the intended anonymous relay design and ensures the built-in onion transport and proxy chain flow are used for chat payloads.

## 2. Router accepts arbitrary user IDs and allows connection takeover

- Status: Fixed
- Severity: High
- Affected area: server/router/main.go, server/router/main_test.go, client/frontend/src/ws.ts, client/frontend/src/main.ts
- Issue:
  The WebSocket endpoint trusts the `user_id` query parameter and registers the connection under that identity without authentication or ownership validation. A client can impersonate another user by connecting with the same user ID, replacing the legitimate session, and receiving queued messages intended for that account.
- Evidence:
  - The server accepts any `user_id` and generates one only when missing: [server/router/main.go](server/router/main.go#L326-L330)
  - The connection is registered under that value immediately: [server/router/main.go](server/router/main.go#L337-L345)
- Impact:
  This is an authentication and identity-confusion issue that can cause message hijacking or denial of service against a valid session.
- Solution: The router now ignores caller-selected query IDs, issues UUID identities with random credentials, and requires the credential over a WebSocket subprotocol to reconnect; only a credential hash is retained in Redis with a TTL. Regression tests verify caller IDs and missing or wrong credentials cannot reclaim a session, while a valid credential reconnects to the same identity.

## 3. Exit proxy allows private-network destinations by default in the compose deployment

- Status: Resolved
- Severity: High
- Affected area: docker-compose.yml, server/proxy/main.go
- Issue:
  The compose stack sets `EXIT_ALLOW_PRIVATE=true`. With the current configuration, the exit node allows private/local addresses, which is unsafe for a deployment that is meant to be a privacy-preserving relay. This is especially risky when the public guard endpoint is reachable from external clients.
- Evidence:
  - Compose config sets the flag: [docker-compose.yml](docker-compose.yml#L56-L62)
  - The proxy enforces private-address filtering only when `EXIT_ALLOW_PRIVATE` is false (default deny): [server/proxy/main.go](server/proxy/main.go#L214-L240)
- Impact:
  This expands the reachable destination set beyond intended public internet addresses and can expose internal or local services.
- Solution:
  The exit proxy now defaults to `EXIT_ALLOW_PRIVATE=false` in the local compose deployment, matching the proxy runtime’s default-deny behavior and requiring an explicit opt-in for private destinations.

## 4. offline message TTL is removed before write completion

- Status: Resolved
- Severity: Medium
- Affected area: server/router/main.go
- Issue:
  Redis offline keys are deleted as soon as the message is placed into the in-memory channel, even though the socket may fail to flush that message. That creates a gap where the message is lost after being considered delivered.
- Evidence:
  - `routeMessage` deletes the Redis key immediately after `h.deliver(...)` succeeds: [server/router/main.go](server/router/main.go#L127-L143)
  - The delivery path only checks whether the payload was accepted onto the channel, not whether `writePump` actually wrote it to the socket: [server/router/main.go](server/router/main.go#L105-L123)
- Impact:
  This violates the offline replay contract and can silently lose messages that were queued but not successfully transmitted.
- Solution: Router messages are now stored with the existing TTL before they are queued, and their Redis keys are deleted only after a successful WebSocket write. Regression tests verify successful writes clear the key and failed writes retain it.

## 5. Pre-key bundle advertises a one-time pre-key that is not retained or recoverable

- Status: Resolved
- Severity: Medium
- Affected area: client/security-core/src/device_bootstrap.rs
- Issue:
  The generated pre-key bundle includes a fresh one-time pre-key, but that secret material is never persisted alongside the encrypted device envelope. A peer that receives that bundle can be handed a key it cannot actually use for a session.
- Evidence:
  - The bundle is constructed using a fresh one-time key: [client/security-core/src/device_bootstrap.rs](client/security-core/src/device_bootstrap.rs#L243-L279)
  - The corresponding serialized key material is stored only in the immediate local temporary variables, not in the persisted envelope: [client/security-core/src/device_bootstrap.rs](client/security-core/src/device_bootstrap.rs#L200-L228)
- Impact:
  The advertised bundle is structurally inconsistent with the device’s actual retrievable key material.
- Solution: The bootstrap flow now persists the one-time pre-key protobuf in the encrypted envelope and reconstructs the bundle from that retained record so the bundle remains recoverable across reloads.

## 6. Security-core does not compile against the current Signal API

- Status: Resolved
- Severity: Medium
- Affected area: client/security-core/src/device_bootstrap.rs, client/security-core/Cargo.toml
- Issue:
  The Rust crate fails to compile with the currently pinned Signal dependency. The module calls APIs that do not exist in the version currently selected by the project, and it also mixes `rand_core` requirements incompatible with the chosen crates.
- Evidence:
  - The dependency points at the Signal lib, but the API used here is not available in the current version: [client/security-core/Cargo.toml](client/security-core/Cargo.toml#L12-L20)
  - Build fails with `IdentityKeyPair::deserialize`, `SignedPreKeyRecord::deserialize`, `KyberPreKeyRecord::deserialize`, and `PreKeyBundle::serialize` errors, plus `OsRng` not satisfying `CryptoRng`: the cargo check output was run and failed for these reasons.
- Impact:
  The security-core cannot be built or tested in its current state; the feature is effectively blocked.
- Solution: The bootstrap now uses the current libsignal record APIs and the project’s compatible rand/rand_core setup, and the crate passes its focused Rust test suite with the retained one-time pre-key persistence regression covered.

## 7. Repository project metadata contains trailing whitespace in the new progress log

- Severity: Low
- Affected area: ANONCHAT_WORK_PROGRESS.md
- Issue:
  The appended work-progress session contains trailing whitespace lines, which is a small repo hygiene issue and is still flagged by `git diff --check`.
- Evidence:
  - `git diff --check` reported trailing whitespace in the new session block inside [ANONCHAT_WORK_PROGRESS.md](ANONCHAT_WORK_PROGRESS.md)
- Impact:
  This is minor but it does reduce patch cleanliness and may fail strict CI checks that enforce whitespace hygiene.

## Validation Summary

The following checks were run during review:

- `cd server/proxy && go test ./... && go build ./...` -> pass
- `cd server/router && go test ./... && go build ./...` -> pass
- `cd /home/mayvena/anonchat/server/router && go test ./... && go build ./...` -> pass - includes session identity ownership and reconnect regression coverage
- `cd /home/mayvena/anonchat/server/router && go test ./... && go build ./...` -> pass - verifies offline message keys survive failed writes and clear after successful writes
- `cd client/frontend && npm run build` -> pass (with a large chunk-size warning)
- `cd /home/mayvena/anonchat/client/frontend && npm run build` -> pass - updated server-issued session and authenticated reconnect protocol; existing large chunk-size warning remains
- `cd client/security-core && cargo test -- --nocapture` -> pass
- `git diff --check` -> reports trailing whitespace in the progress log

## Notes

This list reflects confirmed issues supported by code inspection and validation, not speculative concerns. The most important follow-ups are the chat-transport bypass, router identity trust, and the Rust build blocker.
