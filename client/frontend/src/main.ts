/**
 * main.ts – Tauri IPC wiring.
 *
 * Listens for `proxy-layer-changed` and `core-log` events emitted by the Rust
 * security core, then delegates to GlobeRenderer and TerminalPanel.
 */

import { listen }  from '@tauri-apps/api/event';
import { invoke, isTauri } from '@tauri-apps/api/core';
import { GlobeRenderer, type ProxyChain, type ProxyNode } from './globe';
import { TerminalPanel, type LogEntry } from './terminal';
import { ChatPanel } from './chat';
import { RouterClient, MAX_BODY_CHARS } from './ws';
import { createId } from './id';

const ROUTER_WS_URL: string =
  (import.meta.env?.VITE_ROUTER_WS_URL as string | undefined) ??
  `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.hostname}:8080/ws`;
const PROXY_REGISTRY_URL: string =
  (import.meta.env?.VITE_PROXY_REGISTRY_URL as string | undefined) ?? '';
const SERVICE_SETTINGS_KEY = 'anonchat.service-endpoints.v1';
const MESSAGE_TTL_SECONDS = 30;

interface ServiceEndpoints {
  routerWebSocketUrl: string;
  proxyRegistryUrl: string;
}

function parseServiceEndpoints(routerValue: string, registryValue: string): ServiceEndpoints | null {
  try {
    const router = new URL(routerValue.trim());
    const registry = new URL(registryValue.trim());
    if (router.protocol !== 'wss:' || registry.protocol !== 'https:') return null;
    if (router.username || router.password || router.hash || registry.username || registry.password || registry.hash) return null;
    return { routerWebSocketUrl: router.toString(), proxyRegistryUrl: registry.toString() };
  } catch {
    return null;
  }
}

function loadServiceEndpoints(): ServiceEndpoints | null {
  try {
    const saved = localStorage.getItem(SERVICE_SETTINGS_KEY);
    if (saved) {
      const value: unknown = JSON.parse(saved);
      if (typeof value === 'object' && value !== null && 'routerWebSocketUrl' in value && 'proxyRegistryUrl' in value) {
        const endpoints = value as ServiceEndpoints;
        return parseServiceEndpoints(endpoints.routerWebSocketUrl, endpoints.proxyRegistryUrl);
      }
    }
  } catch {
    return null;
  }

  return PROXY_REGISTRY_URL
    ? parseServiceEndpoints(import.meta.env?.VITE_ROUTER_WS_URL ?? '', PROXY_REGISTRY_URL)
    : null;
}

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

function encodeOnionPacket(packet: number[]): string {
  let binary = '';
  for (const byte of packet) {
    binary += String.fromCharCode(byte & 0xff);
  }
  return btoa(binary);
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
  const settingsButton = document.getElementById('service-settings') as HTMLButtonElement;
  const settingsDialog = document.getElementById('service-settings-dialog') as HTMLDialogElement;
  const settingsForm = document.getElementById('service-settings-form') as HTMLFormElement;
  const routerInput = document.getElementById('router-endpoint') as HTMLInputElement;
  const registryInput = document.getElementById('registry-endpoint') as HTMLInputElement;
  const settingsError = document.getElementById('service-settings-error') as HTMLElement;

  let myUserId = 'connecting';
  let hasProxyChain = false;
  let endpoints = loadServiceEndpoints();
  let router: RouterClient | null = null;
  let eventStreamReady = false;

  const connectRouter = (url: string): void => {
    router?.close();
    chat.setConnectionState('connecting');
    router = new RouterClient({
      url,
      onState: (state) => {
        chat.setConnectionState(state);
        if (!hasProxyChain) {
          statusEl.textContent = state === 'connected'
            ? 'ROUTER CONNECTED · ROUTE PENDING'
            : `ROUTER ${state.toUpperCase()}`;
        }
        terminal.append({
          level:       state === 'failed' ? 'error' : state === 'connected' ? 'info' : 'warn',
          message:     `Chat router: ${state}`,
          timestampMs: Date.now(),
        });
      },
      onSession: (userId) => {
        myUserId = userId;
        chat.setUsers([
          { id: 'me', displayName: `You: ${myUserId}` },
          { id: 'guard', displayName: 'Guard Node' },
          { id: 'middle', displayName: 'Middle Node' },
          { id: 'exit', displayName: 'Exit Node' },
          { id: 'system', displayName: 'System' },
        ]);
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
    router.connect();
  };

  const selectInitialRoute = (): void => {
    if (!endpoints) return;
    void invoke('select_proxy_chain_from_registry', { registryUrl: endpoints.proxyRegistryUrl }).catch(() => {
      terminal.append({
        level: 'warn',
        message: 'Initial proxy route unavailable; route remains pending.',
        timestampMs: Date.now(),
      });
    });
  };

  chat.onSend(async (to, body) => {
    if (!isTauri()) {
      terminal.append({
        level: 'error',
        message: 'Message sending requires the ANONCHAT desktop app; browser mode cannot build onion packets.',
        timestampMs: Date.now(),
      });
      return false;
    }

    if (!endpoints || !router) return false;
    if (body.length > MAX_BODY_CHARS) return false;
    const id = createId();

    try {
      const packet = await invoke<number[]>('build_onion_packet_from_registry', {
        registryUrl: endpoints.proxyRegistryUrl,
        message: body,
      });
      const encryptedBody = encodeOnionPacket(packet);
      if (!router.send(id, to, encryptedBody)) return false;
      chat.addMessage({ id, fromUserId: 'me', text: `→ ${to}: ${body}`, ttlSeconds: MESSAGE_TTL_SECONDS });
      return true;
    } catch (err: unknown) {
      terminal.append({
        level: 'error',
        message: `Onion packet build failed: ${String(err)}`,
        timestampMs: Date.now(),
      });
      return false;
    }
  });
  chat.onRetry(() => router?.retry());

  settingsButton.addEventListener('click', () => {
    routerInput.value = endpoints?.routerWebSocketUrl ?? '';
    registryInput.value = endpoints?.proxyRegistryUrl ?? '';
    settingsError.textContent = '';
    settingsDialog.showModal();
  });
  document.getElementById('service-settings-cancel')?.addEventListener('click', () => settingsDialog.close());
  settingsForm.addEventListener('submit', (event) => {
    event.preventDefault();
    const nextEndpoints = parseServiceEndpoints(routerInput.value, registryInput.value);
    if (!nextEndpoints) {
      settingsError.textContent = 'Enter a valid wss:// router URL and https:// registry URL without embedded credentials.';
      return;
    }
    try {
      localStorage.setItem(SERVICE_SETTINGS_KEY, JSON.stringify(nextEndpoints));
    } catch {
      settingsError.textContent = 'Settings could not be saved in this profile.';
      return;
    }
    endpoints = nextEndpoints;
    settingsDialog.close();
    if (isTauri() && eventStreamReady) {
      connectRouter(endpoints.routerWebSocketUrl);
      selectInitialRoute();
    }
  });

  chat.setUsers([
    { id: 'me', displayName: 'You: connecting' },
    { id: 'guard', displayName: 'Guard Node' },
    { id: 'middle', displayName: 'Middle Node' },
    { id: 'exit', displayName: 'Exit Node' },
    { id: 'system', displayName: 'System' },
  ]);

  terminal.append({ level: 'info', message: 'AnonChat initializing…', timestampMs: Date.now() });

  if (!isTauri()) {
    terminal.append({
      level: 'info',
      message: 'Tauri event streams are unavailable in browser mode.',
      timestampMs: Date.now(),
    });
    connectRouter(ROUTER_WS_URL);
    return;
  }

  // ── Proxy chain rotation ────────────────────────────────────────────────
  await listen<ProxyChainChangedPayload>('proxy-layer-changed', ({ payload }) => {
    hasProxyChain = true;
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
    eventStreamReady = true;
    terminal.append({ level: 'info', message: 'Event streams active.', timestampMs: Date.now() });
  } catch (err: unknown) {
    terminal.append({
      level:       'error',
      message:     `Failed to start event stream: ${String(err)}`,
      timestampMs: Date.now(),
    });
    return;
  }

  if (endpoints) {
    routerInput.value = endpoints.routerWebSocketUrl;
    registryInput.value = endpoints.proxyRegistryUrl;
    connectRouter(endpoints.routerWebSocketUrl);
    selectInitialRoute();
  } else {
    statusEl.textContent = 'SERVICE ENDPOINTS REQUIRED';
    terminal.append({ level: 'warn', message: 'Configure remote service endpoints to connect.', timestampMs: Date.now() });
    settingsDialog.showModal();
  }
}

main().catch((err: unknown) => {
  // Last-resort error surfacing – the terminal may not be initialised here.
  const termEl = document.getElementById('terminal');
  if (termEl) {
    termEl.textContent = `Fatal: ${String(err)}`;
  }
});
