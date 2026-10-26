# ANONCHAT Android Prompt Playbook

This file contains ANONCHAT specific prompts for generating Android code that integrates with the existing ANONCHAT architecture.

## Intent

Use these prompts to build or modify Android client behavior for ANONCHAT without drifting into generic app patterns.

## Quickstart: 3 Copy-Ready Prompts

Use these three prompts in order when starting a new ANONCHAT Android task.

### Quickstart 1: Contract And Scope Preflight

```text
Task:
Perform a preflight for ANONCHAT Android integration before any code changes.

Read first:
- AGENTS.md
- README.md
- client/src-tauri/src/events.rs
- client/src-tauri/src/ipc_bridge.rs
- server/router/main.go

Deliverables:
1. List exact event contracts to preserve (proxy-layer-changed, core-log).
2. List websocket envelope/session expectations from server/router.
3. List security constraints that must remain true (no durable chat history, no sensitive logs).
4. Propose target files for the implementation and explain why each file is needed.

Output format:
- Contract table
- Risk list
- File plan
```

### Quickstart 2: Implement Minimal Vertical Slice

```text
Task:
Implement a minimal ANONCHAT Android vertical slice in <TARGET_FILES>.

Requirements:
1. Subscribe to proxy-layer-changed and core-log with strict payload guards.
2. Render connection/session status states: initializing, connecting, connected, degraded, disconnected.
3. Add websocket client compatibility for /ws?user_id flow and session ack handling.
4. Keep message handling ephemeral in memory only; do not persist message bodies.
5. Do not rename events or change envelope compatibility.

Validation:
- Build command(s)
- Manual check list for event reception and status transitions

Output format:
- File-by-file change summary
- Validation output summary
- Residual risk notes
```

### Quickstart 3: End-To-End Behavior Verification

```text
Task:
Validate ANONCHAT Android behavior against local stack after implementation.

Checks:
1. Connect with user_id and receive session ack.
2. Send message while recipient online and confirm delivery behavior.
3. Send message while recipient offline, then reconnect recipient and confirm replay behavior.
4. Confirm replay is not duplicated after successful delivery.
5. Confirm reconnect UX after transient disconnect.
6. Confirm logs contain no sensitive payload fields.

Output format:
- Pass/fail matrix
- Evidence notes per check
- Follow-up fixes by priority
```

## Source Alignment

Before running any prompt, require the agent to align with these project files:

- AGENTS.md
- README.md
- client/src-tauri/src/events.rs
- client/src-tauri/src/ipc_bridge.rs
- client/frontend/src/main.ts
- client/security-core/src/onion/cipher.rs
- client/security-core/src/panic_destruction.rs
- server/router/main.go

## Global ANONCHAT Constraints Block

Append this block to every prompt:

```text
Constraints:
1. Keep edits scoped to target files listed in the prompt.
2. Do not introduce OAuth, social login, or generic token-refresh auth flows unless explicitly requested.
3. Preserve anonymous session model and websocket user_id semantics.
4. No durable local chat history. Message bodies must not be persisted on disk.
5. Do not log plaintext message bodies, key material, destination addresses, or stable user identifiers.
6. Keep offline behavior aligned with server TTL semantics; do not create permanent client-side replay queues.
7. Keep onion packet assumptions compatible with security-core invariants, including fixed packet construction rules.
8. Do not rename or change existing Tauri event names without explicit coordination.
9. Do not add dependencies unless explicitly requested.
10. Return exact validation commands and summarized output.
```

## Track 1: ANONCHAT Android Foundation

### 1.1 Android Client Entry Setup

```text
Create or refine Android client entry flow for ANONCHAT in <TARGET_FILES>.

Requirements:
- Focus on anonymous session UX, not account signup.
- Include startup states: initializing, connecting, connected, degraded, disconnected.
- Keep all message rendering ephemeral unless explicitly asked to cache non-sensitive metadata.

Return:
- State model and transitions
- File-level changes
- Build and run commands
```

### 1.2 Session Lifecycle Prompt

```text
Implement ANONCHAT session lifecycle handling in <TARGET_FILES>.

Requirements:
- Support connect, reconnect with backoff, transient failure, and hard failure.
- Session expiry behavior must be explicit in UI.
- Do not persist sensitive session artifacts.

Return:
- Lifecycle timeline
- Edge cases handled
- Manual verification checklist
```

## Track 2: Tauri Bridge and Event Contracts

### 2.1 Event Contract Sync Prompt

```text
Read event contracts from client/src-tauri/src/events.rs and bridge emission paths from client/src-tauri/src/ipc_bridge.rs.

Implement in <TARGET_FILES>:
- Strict typed handling for proxy-layer-changed
- Strict typed handling for core-log
- Runtime guards for missing or malformed payload fields

Requirements:
- Preserve event names exactly.
- Ensure listener registration and cleanup are lifecycle safe.

Return:
- Contract table (event -> fields -> UI effect)
- Error handling behavior for invalid payloads
- Validation commands
```

### 2.2 Chain Visualization Integration Prompt

```text
Implement UI updates driven by proxy-layer-changed event in <TARGET_FILES>.

Requirements:
- Show guard, middle, and exit transitions clearly.
- Handle missing coordinates safely with a graceful fallback state.
- Avoid heavy per-frame allocations if rendering animation.

Return:
- Update flow from event receipt to render
- Performance safeguards
- Manual validation plan for chain rotation
```

## Track 3: Networking and WebSocket Behavior

### 3.1 WebSocket Protocol Compatibility Prompt

```text
Implement websocket client compatibility with server/router/main.go behavior in <TARGET_FILES>.

Requirements:
- Use /ws?user_id session shape.
- Handle session ack envelope and message envelope shapes.
- Surface heartbeat/reconnect status in UI.
- Keep online vs offline delivery expectations aligned with server behavior.

Return:
- Envelope parsing strategy
- Failure and reconnect strategy
- Verification script using local stack
```

### 3.2 Offline Replay UX Prompt

```text
Implement user-facing UX for short-lived offline replay semantics in <TARGET_FILES>.

Requirements:
- Explain transient delivery to users without implying durable message history.
- Do not store message bodies for long-term replay on device.
- Reflect replay completion and expiry clearly.

Return:
- UX state matrix
- Copy text recommendations
- End-to-end checks
```

## Track 4: Security and Privacy Enforcement

### 4.1 Log Hygiene Prompt

```text
Audit and patch Android-side logs in <TARGET_FILES> to enforce ANONCHAT privacy policy.

Requirements:
- Remove or redact sensitive fields.
- Keep only operationally necessary diagnostics.
- Include before/after log examples with redacted placeholders.

Return:
- Files changed
- Redaction rules
- Validation command output
```

### 4.2 In-Memory Handling Prompt

```text
Implement secure in-memory handling for sensitive UI/runtime data in <TARGET_FILES>.

Requirements:
- Minimize retention window of plaintext message content and session secrets.
- Clear sensitive in-memory state on explicit logout/teardown flows.
- Document trade-offs when full memory wiping is not guaranteed at framework level.

Return:
- Data lifecycle notes
- Clear points where cleanup occurs
- Residual risk notes
```

### 4.3 Storage Policy Prompt

```text
Implement storage policy boundaries in <TARGET_FILES> for ANONCHAT Android client.

Allow list:
- Non-sensitive UI preferences
- Non-sensitive feature flags

Deny list:
- Message bodies
- Decrypted payloads
- Private keys and raw session key material
- Stable identity maps beyond active session needs

Return:
- Storage map (what is stored, where, why)
- Policy enforcement checks
- Regression checks
```

## Track 5: Reliability, Performance, and UX

### 5.1 Connectivity Resilience Prompt

```text
Improve connection resilience in <TARGET_FILES>.

Requirements:
- Use bounded exponential backoff with jitter.
- Avoid busy-loop reconnects.
- Show clear user status for retry attempts and failure states.

Return:
- Backoff algorithm details
- UX status behavior
- Test scenarios
```

### 5.2 Compose Performance Prompt

```text
Run a Compose performance pass for <SCREEN_NAME> in <TARGET_FILES>.

Requirements:
- Reduce unnecessary recompositions.
- Keep rendering and state updates predictable.
- Avoid heavy work on the main thread.

Return:
- Optimizations made
- Measurement approach
- Validation results
```

### 5.3 Accessibility Prompt

```text
Improve accessibility in <TARGET_FILES> for ANONCHAT Android UI.

Requirements:
- Ensure screen reader labels for key controls.
- Ensure keyboard/focus traversal works for all critical actions.
- Ensure color and status indicators are not color-only.

Return:
- Accessibility checklist
- Remaining gaps
- Manual verification steps
```

## Track 6: ANONCHAT End-to-End Validation Prompts

### 6.1 Local Stack Validation Prompt

```text
Validate Android client behavior against local ANONCHAT stack using docker compose and ws flow in README.md.

Checks:
1. Connect with user_id and receive session ack.
2. Send message while recipient online.
3. Send message while recipient offline.
4. Reconnect recipient and verify replay delivery.
5. Verify replay cleanup semantics and no duplicate replay.
6. Verify heartbeat timeout and reconnect UX behavior.
7. Confirm no sensitive logs were emitted.

Return:
- Pass/fail matrix
- Evidence for each check
- Follow-up fixes needed
```

### 6.2 Contract Regression Prompt

```text
Perform contract regression checks after Android changes.

Requirements:
- Confirm event names unchanged.
- Confirm websocket envelope compatibility unchanged.
- Confirm no changes to security-core packet assumptions.

Return:
- Regression checklist
- Any breaking changes detected
- Required mitigation actions
```

## What Not To Generate By Default

- OAuth or social auth flows
- Persistent chat history features
- Background sync queues for sensitive message bodies
- Telemetry payloads containing message content or identifiers
- Event contract renames across Tauri bridge
- Dependency-heavy architecture rewrites

## ANONCHAT Prompt Template

```text
Task:
<ONE_SENTENCE_GOAL>

Project context:
- This is ANONCHAT. Preserve anonymous session model, ephemeral message policy, and current Tauri event contracts.

Target files:
<EXACT_FILE_LIST>

Implementation requirements:
1. <REQUIREMENT_1>
2. <REQUIREMENT_2>
3. <REQUIREMENT_3>

Invariants (must not change):
1. websocket envelope compatibility with server/router/main.go
2. event name compatibility with client/src-tauri/src/events.rs
3. no durable local message history

Validation:
- <COMMAND_1>
- <COMMAND_2>
- <MANUAL_CHECK_1>

Output format:
- Summary of changes
- File-by-file diff explanation
- Validation output summary
- Security/privacy check results
- Remaining risks
```
