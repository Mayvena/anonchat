mod events;
mod ipc_bridge;

use std::sync::Arc;

use tokio::sync::Mutex;

use ipc_bridge::{LogEventBus, ProxyEventBus, StreamRunning};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .manage(ProxyEventBus::new())
        .manage(LogEventBus::new())
        .manage(StreamRunning(Arc::new(Mutex::new(false))))
        .invoke_handler(tauri::generate_handler![
            ipc_bridge::start_proxy_event_stream,
            ipc_bridge::emit_core_log,
            ipc_bridge::build_onion_packet_from_registry,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
