/// Geographical coordinates for rendering on the Three.js globe.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GeoCoordinates {
    pub latitude: f64,
    pub longitude: f64,
}

/// Description of one active proxy node as known to the frontend.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ProxyNodeInfo {
    /// Guard / Middle / Exit
    pub role: String,
    /// ISO 3166-1 alpha-2 country code.
    pub country_code: String,
    /// Public IP address of the node (never the private / WireGuard address).
    pub ip_address: String,
    /// Position on the globe.
    pub coordinates: GeoCoordinates,
}

/// Full proxy chain snapshot sent to the frontend on every rotation.
///
/// Tauri event name: `"proxy-layer-changed"`
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ProxyChainChangedPayload {
    /// Guard node information.
    pub guard: ProxyNodeInfo,
    /// Middle node information.
    pub middle: ProxyNodeInfo,
    /// Exit node information.
    pub exit: ProxyNodeInfo,
    /// Unix timestamp (seconds) when this rotation happened.
    pub rotated_at: i64,
}

/// Severity level for a system log entry.
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum LogLevel {
    Info,
    Warn,
    Error,
}

/// A single structured log line emitted from the Rust core to the frontend.
///
/// Tauri event name: `"core-log"`
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CoreLogPayload {
    pub level: LogLevel,
    /// Human-readable message (never contains PII, IPs, or key material).
    pub message: String,
    /// Unix timestamp (milliseconds).
    pub timestamp_ms: i64,
}
