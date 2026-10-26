/**
 * main.ts – Tauri IPC wiring.
 *
 * Listens for `proxy-layer-changed` and `core-log` events emitted by the Rust
 * security core, then delegates to GlobeRenderer and TerminalPanel.
 */

import { listen }  from '@tauri-apps/api/event';
import { invoke }  from '@tauri-apps/api/core';
import { GlobeRenderer, type ProxyChain, type ProxyNode } from './globe';
import { TerminalPanel, type LogEntry } from './terminal';
import { ChatPanel } from './chat';
import { RouterClient, MAX_BODY_CHARS } from './ws';

const ROUTER_WS_URL: string =
  (import.meta.env?.VITE_ROUTER_WS_URL as string | undefined) ?? 'ws://127.0.0.1:8080/ws';
const MESSAGE_TTL_SECONDS = 30;

// ─────────────────────────────────────────────────────────────────────────────
// Payload types (must mirror Rust src-tauri/src/events.rs)
// ─────────────────────────────────────────────────────────────────────────────

interface GeoCoordinates {
  latitude:  number;
  longitude: number;
}

interface ProxyNodeInfo {
  role:        string;
  countryCode: string;
  ipAddress:   string;
  coordinates: GeoCoordinates;
}

interface ProxyChainChangedPayload {
  guard:     ProxyNodeInfo;
  middle:    ProxyNodeInfo;
  exit:      ProxyNodeInfo;
  rotatedAt: number;           // Unix seconds
}

interface CoreLogPayload {
  level:       'info' | 'warn' | 'error';
  message:     string;
  timestampMs: number;
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

function toProxyNode(n: ProxyNodeInfo): ProxyNode {
  return {
    latitude:    n.coordinates.latitude,
    longitude:   n.coordinates.longitude,
    countryCode: n.countryCode,
    role:        n.role,
    ipAddress:   n.ipAddress,
  };
}

function toProxyChain(p: ProxyChainChangedPayload): ProxyChain {
  return {
    guard:  toProxyNode(p.guard),
    middle: toProxyNode(p.middle),
    exit:   toProxyNode(p.exit),
  };
}

// ─────────────────────────────────────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  const canvas     = document.getElementById('globe-canvas') as HTMLCanvasElement;
  const terminalEl = document.getElementById('terminal')     as HTMLElement;
  const statusEl   = document.getElementById('chain-status') as HTMLElement;

  const globe    = new GlobeRenderer(canvas);
  const terminal = new TerminalPanel(terminalEl);
  const chat     = new ChatPanel();

  // Ephemeral identity: lives in memory only, new on every launch.
  const myUserId = crypto.randomUUID().slice(0, 8);

  const router = new RouterClient({
    url:    ROUTER_WS_URL,
    userId: myUserId,
    onState: (state) => {
      chat.setConnectionState(state);
      terminal.append({
        level:       state === 'failed' ? 'error' : state === 'connected' ? 'info' : 'warn',
        message:     `Chat router: ${state}`,
        timestampMs: Date.now(),
      });
    },
    onMessage: (msg) => {
      if (!chat.hasUser(msg.from)) {
        chat.addUser({ id: msg.from, displayName: msg.from });
      }
      chat.addMessage({
        id:         msg.id,
        fromUserId: msg.from,
        text:       msg.body,
        ttlSeconds: MESSAGE_TTL_SECONDS,
      });
    },
  });

  chat.onSend((to, body) => {
    if (body.length > MAX_BODY_CHARS) return false;
    const id = crypto.randomUUID();
    if (!router.send(id, to, body)) return false;
    chat.addMessage({ id, fromUserId: 'me', text: `→ ${to}: ${body}`, ttlSeconds: MESSAGE_TTL_SECONDS });
    return true;
  });
  chat.onRetry(() => router.retry());
  router.connect();

  chat.setUsers([
    { id: 'me', displayName: `You: ${myUserId}` },
    { id: 'guard', displayName: 'Guard Node' },
    { id: 'middle', displayName: 'Middle Node' },
    { id: 'exit', displayName: 'Exit Node' },
    { id: 'system', displayName: 'System' },
  ]);

  terminal.append({ level: 'info', message: 'AnonChat initializing…', timestampMs: Date.now() });

  // ── Proxy chain rotation ────────────────────────────────────────────────
  await listen<ProxyChainChangedPayload>('proxy-layer-changed', ({ payload }) => {
    globe.updateChain(toProxyChain(payload));

    const g = payload.guard.countryCode;
    const m = payload.middle.countryCode;
    const e = payload.exit.countryCode;

    statusEl.textContent = `${g}  ──►  ${m}  ──►  ${e}`;

    terminal.append({
      level:       'info',
      message:     `Proxy rotated → Guard:${g} (${payload.guard.ipAddress})  Middle:${m} (${payload.middle.ipAddress})  Exit:${e} (${payload.exit.ipAddress})`,
      timestampMs: payload.rotatedAt * 1_000,
    });

    chat.addMessage({
      fromUserId: 'system',
      text: `Route updated: ${g} -> ${m} -> ${e}`,
      ttlSeconds: 18,
    });

    chat.addMessage({
      fromUserId: 'guard',
      text: `Connected: ${payload.guard.ipAddress}`,
      ttlSeconds: 11,
    });

    chat.addMessage({
      fromUserId: 'middle',
      text: `Relaying via ${payload.middle.countryCode}`,
      ttlSeconds: 13,
    });

    chat.addMessage({
      fromUserId: 'exit',
      text: `Exit handoff: ${payload.exit.ipAddress}`,
      ttlSeconds: 15,
    });
  });

  // ── Core log stream ─────────────────────────────────────────────────────
  await listen<CoreLogPayload>('core-log', ({ payload }) => {
    const entry: LogEntry = {
      level:       payload.level,
      message:     payload.message,
      timestampMs: payload.timestampMs,
    };
    terminal.append(entry);

    if (payload.level !== 'info') {
      chat.addMessage({
        fromUserId: 'system',
        text: payload.message,
        ttlSeconds: payload.level === 'error' ? 24 : 16,
      });
    }
  });

  // ── Kick the Rust event emitter ─────────────────────────────────────────
  try {
    await invoke('start_proxy_event_stream');
    terminal.append({ level: 'info', message: 'Event streams active.', timestampMs: Date.now() });
  } catch (err: unknown) {
    terminal.append({
      level:       'error',
      message:     `Failed to start event stream: ${String(err)}`,
      timestampMs: Date.now(),
    });
  }
}

main().catch((err: unknown) => {
  // Last-resort error surfacing – the terminal may not be initialised here.
  const termEl = document.getElementById('terminal');
  if (termEl) {
    termEl.textContent = `Fatal: ${String(err)}`;
  }
});
