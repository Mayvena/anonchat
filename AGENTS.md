# AGENTS.md

This file defines working guidance for AI agents and developers making changes in ANONCHAT.

## Scope

- Apply these rules to the full anonchat project tree.
- Treat this folder as the project root for compose, server, and client tasks.
- Keep edits narrowly scoped to the requested subsystem. Do not rewrite unrelated architecture.

## Project Snapshot

- Stack: Go services for proxy and router, Rust security core, Tauri bridge, TypeScript frontend.
- Runtime path: client -> guard proxy -> middle proxy -> exit proxy -> chat router -> Redis RAM.
- Product posture: anonymous, encrypted chat with no durable message history.
- Current behavior includes short lived offline buffering in Redis with TTL for reconnect delivery.

## Source Of Truth Files

- Local run and smoke tests: README.md
- Deployment and hardening: DEPLOYMENT.md
- Local infra definition: docker-compose.yml
- Proxy runtime: server/proxy/main.go
- Router runtime: server/router/main.go
- Rust security core: client/security-core/src

If this file conflicts with those files, align changes to runtime code and update this file.

## Ownership And Boundaries

- Go proxy handles TLS transport, SOCKS5 entry, relay plumbing, and destination header forwarding.
- Go router handles websocket sessions, heartbeat, online delivery, and Redis backed short TTL offline queue.
- Rust security core owns key material lifecycle, onion packet building, and secret zeroization behavior.
- Tauri and frontend own event wiring and visual chain representation.

Do not move Go networking behavior into Rust unless the task explicitly requests a redesign.

## Security And Privacy Rules

- Never log plaintext message bodies, key material, destination addresses, or user identifiers.
- Do not introduce file based persistence for chat payloads, session keys, or offline queue data.
- Redis must remain memory only in local and deployment configs.
- Keep secret data in zeroizing containers in Rust and clear buffers after use.
- Keep panic destruction behavior explicit, testable, and fail closed.

### Logging Policy

- Allowed: minimal operational logs that do not include sensitive payload fields.
- Forbidden: debug dumps, packet/body prints, IP or key disclosure, and disk persisted sensitive logs.

## Protocol And Data Invariants

- The 4096 byte fixed packet invariant applies to onion packet construction in Rust security core.
- Do not impose a fake 4096 byte requirement on every Go transport frame or stream segment.
- Proxy destination header format must remain compatible across guard, middle, and exit nodes.
- Router offline data is ephemeral only and must expire automatically.

## Websocket And API Rules

- Preserve heartbeat timing guarantees unless a task requests protocol changes.
- Keep message delivery semantics: deliver directly when online, else store short TTL and delete after successful replay.
- Development may use permissive websocket origin checks; production guidance must require explicit allowlist policy.

## Agent Workflow

1. Confirm subsystem ownership before editing.
2. Reuse existing patterns and constants before adding new ones.
3. Add tests or focused checks for behavior changing edits.
4. Validate only the narrowest relevant targets.
5. Update docs when runtime behavior changes.

## Validation Commands

- Server proxy: cd server/proxy && go test ./... && go build ./...
- Server router: cd server/router && go test ./... && go build ./...
- Security core: cd client/security-core && cargo test && cargo check
- Frontend: cd client/frontend && npm run build
- Full stack smoke: docker compose up -d --build

## Work Progress Tracking File

For any non-trivial ANONCHAT task (multi-step changes, cross-subsystem changes, or security-sensitive edits), create and maintain a progress tracker file at project root named `ANONCHAT_WORK_PROGRESS.md`.

When to create:

- Create the file at task start if it does not exist.
- If it exists, append a new dated session section instead of replacing prior history.

Required sections per session:

1. Task Summary
2. Scope (target files and subsystems)
3. Plan (step-by-step)
4. Progress Log (timestamped updates)
5. Validation Performed (commands and outcomes)
6. Risks / Blockers
7. Next Actions
8. Final Outcome

Update rules:

- Update Progress Log after each completed implementation step.
- Record exact validation commands run and whether they passed or failed.
- If a command is not run, explicitly state why.
- Keep entries factual and concise; do not include secrets or sensitive payload data.
- On task completion, mark Final Outcome as `complete`, `partial`, or `blocked`.

Minimum template to generate:

```markdown
# ANONCHAT Work Progress

## Session: <YYYY-MM-DD HH:MM TZ> - <SHORT_TASK_NAME>

### Task Summary
- <what is being changed and why>

### Scope
- Files:
	- <path>
- Subsystems:
	- <proxy|router|security-core|tauri|frontend|infra>

### Plan
1. <step 1>
2. <step 2>
3. <step 3>

### Progress Log
- <HH:MM> <completed step update>

### Validation Performed
- <command> -> <pass|fail|not run> - <short note>

### Risks / Blockers
- <none or details>

### Next Actions
1. <next step>

### Final Outcome
- Status: <complete|partial|blocked>
- Summary: <result>
```

If the user asks for a status update, summarize directly from `ANONCHAT_WORK_PROGRESS.md` first.

## Change Review Checklist

- No sensitive logging introduced.
- No durable storage introduced for messages or secrets.
- Onion packet and proxy compatibility preserved.
- Offline queue remains TTL based and cleanup on successful delivery is preserved.
- Any security sensitive behavior change includes rationale in PR notes.

## Prompting Guidance For AI Users

When requesting code generation, use ANONCHAT runtime contracts and acceptance criteria instead of open ended templates.

Include:

- Target files and subsystem.
- Existing contracts that must remain compatible.
- Required tests or validation commands.
- Security constraints for logging, storage, and key handling.

ANONCHAT contract anchors:

- Tauri events: `proxy-layer-changed`, `core-log`.
- Router websocket endpoint and envelopes: `/ws?user_id=...`, `session`, `message`.
- Security-core packet invariant: fixed 4096-byte onion packet construction.
- Offline message semantics: short TTL replay from Redis, no durable chat history.

Example request shape:

Implement in server/router/main.go only. Keep heartbeat semantics and TTL offline queue behavior unchanged. Do not log user IDs or message bodies. Provide go test and go build validation output.

## Step By Step Frontend Code Generation Prompts

Use these prompts when asking an AI to generate frontend code in ANONCHAT. These prompts are tied to existing frontend and Tauri files.

### Prompt 1: Frontend Change In Existing ANONCHAT UI

Step 1 prompt:
Inspect `client/frontend/src/main.ts`, `client/frontend/src/chat.ts`, `client/frontend/src/terminal.ts`, and `client/frontend/src/globe.ts`. Identify the smallest owning file set for <FEATURE_NAME> and explain why each file is required.

Step 2 prompt:
Implement <FEATURE_NAME> in the selected files only. Preserve current styling conventions from `client/frontend/index.html`. Do not introduce new UI libraries.

Step 3 prompt:
Wire state and handlers with minimal changes. If event-driven, integrate through existing `main.ts` listener wiring patterns.

Step 4 prompt:
Add accessibility checks for keyboard interaction, focus behavior, and readable labels for any new controls.

Step 5 prompt:
Provide validation steps including `cd client/frontend && npm run build` and manual checks for chat send flow, terminal updates, and chain status rendering.

### Prompt 2: Existing Tauri Event To UI Integration

Step 1 prompt:
Inspect `client/src-tauri/src/events.rs` and `client/frontend/src/main.ts`. List payload fields for `proxy-layer-changed` and `core-log` currently consumed by the frontend.

Step 2 prompt:
Implement UI behavior for one existing event (`proxy-layer-changed` or `core-log`) in `client/frontend/src/main.ts` and target view files with strict payload typing and runtime guards for missing fields.

Step 3 prompt:
Render event-driven state in existing UI components without changing unrelated behavior.

Step 4 prompt:
Ensure listener registration and cleanup are lifecycle safe and do not leak subscriptions.

Step 5 prompt:
Return final diff summary plus `npm run build` output for `client/frontend`.

### Prompt 3: Proxy Chain Globe Update

Step 1 prompt:
Read `client/frontend/src/globe.ts` and document how guard, middle, and exit nodes are rendered and updated from chain events.

Step 2 prompt:
Implement <VISUAL_CHANGE> while preserving existing chain data contracts and animation entry points used by `main.ts`.

Step 3 prompt:
Keep rendering performance stable by avoiding heavy per-frame allocations and unnecessary object recreation.

Step 4 prompt:
Add defensive handling for partial, invalid, or missing node coordinates in chain payloads.

Step 5 prompt:
Provide manual verification steps for first load, chain rotation events, and invalid-coordinate fallback behavior.

### Prompt 4: Chat And WebSocket UX Behavior Change

Step 1 prompt:
Inspect `client/frontend/src/chat.ts` and summarize current message lifecycle, ephemeral rendering expectations, and timer behavior.

Step 2 prompt:
Implement <CHAT_CHANGE> with no persistent message storage and no sensitive debug output.

Step 3 prompt:
Keep compatibility with current router envelope shapes from `server/router/main.go` (`session` and `message`).

Step 4 prompt:
Add clear user-facing states for connecting, connected, degraded, reconnecting, and failed states.

Step 5 prompt:
Provide a focused manual test script for online delivery, offline replay after reconnect, and reconnect failure paths.

### Prompt 5: Frontend Refactor Without Contract Drift

Step 1 prompt:
Identify duplication in <TARGET_FILE> and propose a minimal refactor plan that keeps behavior unchanged.

Step 2 prompt:
Apply the refactor in the same file set only. Avoid broad renames and avoid changing event names or payload shapes.

Step 3 prompt:
Confirm no contract changes for exported types, listener signatures, or websocket envelope handling paths.

Step 4 prompt:
Run `cd client/frontend && npm run build` and report results.

Step 5 prompt:
List before/after checks proving no regressions in chain updates, chat rendering, and terminal event display.

### Prompt 6: Frontend Bugfix Prompt

Step 1 prompt:
Reproduce bug <BUG_DESCRIPTION> using current ANONCHAT frontend flow and identify exact failing module and function.

Step 2 prompt:
Implement the smallest safe fix in the owning file.

Step 3 prompt:
Add guards for edge cases that caused the bug, especially malformed event payloads and websocket state transitions.

Step 4 prompt:
Verify no regressions in related flows: chain updates, chat delivery status, and terminal log rendering.

Step 5 prompt:
Return root cause, patch summary, and verification results.

### Prompt Usage Rules

- Always specify exact target files.
- Always include ANONCHAT invariants that must remain unchanged.
- Always require build output from client/frontend.
- Never allow introduction of new dependencies unless explicitly requested.
- Never include sensitive data in console output.

Required invariants to include in frontend prompts:

- Keep existing Tauri event names unchanged (`proxy-layer-changed`, `core-log`).
- Preserve websocket compatibility with router envelopes and `/ws?user_id` flow.
- Preserve ephemeral message behavior and avoid durable local history.

---