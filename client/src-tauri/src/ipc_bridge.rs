//! Tauri IPC bridge between the Rust security core and the JavaScript frontend.
//!
//! ## Events emitted to the frontend
//!
//! | Event name             | Payload type              | Trigger                        |
//! |------------------------|---------------------------|--------------------------------|
//! | `proxy-layer-changed`  | `ProxyChainChangedPayload`| Every proxy chain rotation     |
//! | `core-log`             | `CoreLogPayload`          | Any core diagnostic message    |
//!
//! ## Tauri commands exposed to the frontend
//!
//! | Command                    | Description                                    |
//! |----------------------------|------------------------------------------------|
//! | `start_proxy_event_stream` | Spawns the background emitter task             |
//! | `emit_core_log`            | Lets Rust sub-systems push a log entry         |
//! | `build_onion_packet_from_registry` | Fetches nodes and builds a 4096-byte packet |

use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

use tauri::{AppHandle, Emitter, Manager, State};
use tokio::sync::{Mutex, broadcast};

use crate::events::{
    CoreLogPayload, GeoCoordinates, LogLevel, ProxyChainChangedPayload, ProxyNodeInfo,
};
use security_core::{fetch_and_encrypt_chain, fetch_and_select_chain};

// ─────────────────────────────────────────────────────────────────────────────
// Shared state injected via Tauri's managed state mechanism
// ─────────────────────────────────────────────────────────────────────────────

/// A broadcast channel through which any Rust sub-system can push a
/// `ProxyChainChangedPayload`.  The Tauri command holds a receiver and
/// forwards every message to the frontend via `app_handle.emit`.
pub struct ProxyEventBus {
    pub tx: broadcast::Sender<ProxyChainChangedPayload>,
}

impl ProxyEventBus {
    pub fn new() -> Self {
        let (tx, _) = broadcast::channel(32);
        Self { tx }
    }

    /// Publish a new chain snapshot.  Returns the number of active receivers.
    pub fn publish(&self, payload: ProxyChainChangedPayload) -> usize {
        self.tx.send(payload).unwrap_or(0)
    }
}

/// Separate bus for structured log entries.
pub struct LogEventBus {
    pub tx: broadcast::Sender<CoreLogPayload>,
}

impl LogEventBus {
    pub fn new() -> Self {
        let (tx, _) = broadcast::channel(128);
        Self { tx }
    }

    /// Publish a log entry.
    pub fn publish(&self, payload: CoreLogPayload) -> usize {
        self.tx.send(payload).unwrap_or(0)
    }
}

/// Guard flag: prevents starting more than one stream per session.
pub struct StreamRunning(pub Arc<Mutex<bool>>);

// ─────────────────────────────────────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────────────────────────────────────

fn now_ms() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

fn node_info(node: &security_core::ProxyNode, role: &str) -> ProxyNodeInfo {
    let ip_address = node
        .address
        .rsplit_once(':')
        .map(|(host, _)| host.trim_start_matches('[').trim_end_matches(']'))
        .unwrap_or(node.address.as_str())
        .to_owned();

    ProxyNodeInfo {
        role: role.to_owned(),
        country_code: node.country_code.clone(),
        ip_address,
        coordinates: GeoCoordinates {
            latitude: node.latitude,
            longitude: node.longitude,
        },
    }
}

fn chain_payload(chain: &security_core::NodeChain) -> ProxyChainChangedPayload {
    ProxyChainChangedPayload {
        guard: node_info(&chain.guard, "guard"),
        middle: node_info(&chain.middle, "middle"),
        exit: node_info(&chain.exit, "exit"),
        rotated_at: now_ms() / 1_000,
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Tauri commands
// ─────────────────────────────────────────────────────────────────────────────

/// Start the background task that listens on both broadcast buses and forwards
/// every event to the frontend via `app_handle.emit(...)`.
///
/// Safe to call multiple times: subsequent calls are no-ops while a stream is
/// already running.
///
/// ### Frontend usage (TypeScript)
/// ```ts
/// import { invoke } from "@tauri-apps/api/core";
/// await invoke("start_proxy_event_stream");
/// ```
#[tauri::command]
pub async fn start_proxy_event_stream(
    app: AppHandle,
    proxy_bus: State<'_, ProxyEventBus>,
    log_bus: State<'_, LogEventBus>,
    running: State<'_, StreamRunning>,
) -> Result<(), String> {
    let mut guard = running.0.lock().await;
    if *guard {
        return Ok(()); // already running
    }
    *guard = true;
    drop(guard);

    // Subscribe before spawning so no events are missed.
    let mut proxy_rx = proxy_bus.tx.subscribe();
    let mut log_rx = log_bus.tx.subscribe();

    let app_proxy = app.clone();
    let app_log = app.clone();

    // ── Proxy chain rotation forwarder ────────────────────────────────────
    tokio::spawn(async move {
        loop {
            match proxy_rx.recv().await {
                Ok(payload) => {
                    // Errors here mean the window was closed; exit gracefully.
                    if app_proxy
                        .emit("proxy-layer-changed", &payload)
                        .is_err()
                    {
                        break;
                    }
                }
                Err(broadcast::error::RecvError::Closed) => break,
                Err(broadcast::error::RecvError::Lagged(_)) => {
                    // Slow consumer: skip missed messages, keep running.
                    continue;
                }
            }
        }
    });

    // ── Core log forwarder ────────────────────────────────────────────────
    tokio::spawn(async move {
        loop {
            match log_rx.recv().await {
                Ok(payload) => {
                    if app_log.emit("core-log", &payload).is_err() {
                        break;
                    }
                }
                Err(broadcast::error::RecvError::Closed) => break,
                Err(broadcast::error::RecvError::Lagged(_)) => continue,
            }
        }
    });

    Ok(())
}

/// Push a structured log entry from any Tauri command into the event bus so
/// the frontend receives it in real-time.
///
/// ### Frontend usage (TypeScript)
/// ```ts
/// import { invoke } from "@tauri-apps/api/core";
/// await invoke("emit_core_log", { level: "info", message: "Session started" });
/// ```
#[tauri::command]
pub async fn emit_core_log(
    level: String,
    message: String,
    log_bus: State<'_, LogEventBus>,
) -> Result<(), String> {
    let level = match level.to_lowercase().as_str() {
        "warn"  => LogLevel::Warn,
        "error" => LogLevel::Error,
        _       => LogLevel::Info,
    };

    log_bus.publish(CoreLogPayload {
        level,
        message,
        timestamp_ms: now_ms(),
    });

    Ok(())
}

/// Fetch the active proxy registry and build one fixed-size onion packet.
///
/// The packet is returned as bytes for the transport layer. Message contents,
/// node addresses, and key material are never logged.
#[tauri::command]
pub async fn build_onion_packet_from_registry(
    registry_url: String,
    message: String,
    proxy_bus: State<'_, ProxyEventBus>,
    log_bus: State<'_, LogEventBus>,
) -> Result<Vec<u8>, String> {
    let (packet, chain) = fetch_and_encrypt_chain(&registry_url, message.as_bytes())
        .await
        .map_err(|error| {
            log_bus.publish(CoreLogPayload {
                level: LogLevel::Error,
                message: format!("Proxy chain selection failed: {error}"),
                timestamp_ms: now_ms(),
            });
            error.to_string()
        })?;

    let payload = chain_payload(&chain);
    proxy_bus.publish(payload);
    log_bus.publish(CoreLogPayload {
        level: LogLevel::Info,
        message: "Proxy chain selected and onion packet built".to_owned(),
        timestamp_ms: now_ms(),
    });

    Ok(packet.to_vec())
}

/// Select and publish an initial proxy route so the frontend can render the
/// active chain before the user sends a message.
#[tauri::command]
pub async fn select_proxy_chain_from_registry(
    registry_url: String,
    proxy_bus: State<'_, ProxyEventBus>,
    log_bus: State<'_, LogEventBus>,
) -> Result<(), String> {
    let chain = fetch_and_select_chain(&registry_url).await.map_err(|_| {
        log_bus.publish(CoreLogPayload {
            level: LogLevel::Error,
            message: "Proxy chain selection failed".to_owned(),
            timestamp_ms: now_ms(),
        });
        "Proxy chain selection failed".to_owned()
    })?;

    proxy_bus.publish(chain_payload(&chain));
    Ok(())
}

// ─────────────────────────────────────────────────────────────────────────────
// Convenience: emit helpers callable from Rust (not exposed to frontend)
// ─────────────────────────────────────────────────────────────────────────────

/// Emit a proxy chain rotation from any Rust sub-system.
///
/// # Example
/// ```rust,ignore
/// emit_proxy_rotation(&app_handle, ProxyChainChangedPayload { … });
/// ```
pub fn emit_proxy_rotation(app: &AppHandle, payload: ProxyChainChangedPayload) {
    let _ = app.emit("proxy-layer-changed", payload);
}

/// Emit a structured info log from any Rust sub-system.
pub fn emit_info(app: &AppHandle, message: impl Into<String>) {
    let _ = app.emit(
        "core-log",
        CoreLogPayload {
            level: LogLevel::Info,
            message: message.into(),
            timestamp_ms: now_ms(),
        },
    );
}

/// Emit a structured warning from any Rust sub-system.
pub fn emit_warn(app: &AppHandle, message: impl Into<String>) {
    let _ = app.emit(
        "core-log",
        CoreLogPayload {
            level: LogLevel::Warn,
            message: message.into(),
            timestamp_ms: now_ms(),
        },
    );
}

/// Emit a structured error from any Rust sub-system.
pub fn emit_error(app: &AppHandle, message: impl Into<String>) {
    let _ = app.emit(
        "core-log",
        CoreLogPayload {
            level: LogLevel::Error,
            message: message.into(),
            timestamp_ms: now_ms(),
        },
    );
}
