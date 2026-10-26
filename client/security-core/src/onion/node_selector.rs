//! Node selector – fetches the active proxy list from the central server,
//! filters by distinct countries, and picks Guard / Middle / Exit nodes
//! using a cryptographically-secure OsRng.

use std::collections::HashMap;

use base64::{Engine as _, engine::general_purpose::STANDARD as BASE64};
use rand::RngCore;
use rand::rngs::OsRng;
use rand::seq::SliceRandom;
use serde::{Deserialize, Serialize};
use thiserror::Error;
use x25519_dalek::PublicKey as X25519PublicKey;

/// Description of one active proxy node as returned by the central server.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProxyNode {
    /// Unique identifier for this node.
    pub id: String,
    /// `host:port` to connect to.
    pub address: String,
    /// ISO 3166-1 alpha-2 country code (e.g. `"DE"`, `"SE"`, `"NL"`).
    pub country_code: String,
    /// Base-64-encoded X25519 public key (32 bytes).
    pub public_key_b64: String,
    /// Latitude used by the desktop route renderer.
    #[serde(default)]
    pub latitude: f64,
    /// Longitude used by the desktop route renderer.
    #[serde(default)]
    pub longitude: f64,
}

impl ProxyNode {
    /// Decode and return the X25519 public key.
    pub fn x25519_pubkey(&self) -> Result<X25519PublicKey, NodeSelectError> {
        let bytes = BASE64
            .decode(&self.public_key_b64)
            .map_err(|e| NodeSelectError::Key(format!("base64 decode failed: {e}")))?;

        let arr: [u8; 32] = bytes
            .try_into()
            .map_err(|_| NodeSelectError::Key("X25519 key must be 32 bytes".to_owned()))?;

        Ok(X25519PublicKey::from(arr))
    }
}

/// The selected three-node onion chain.
#[derive(Debug, Clone)]
pub struct NodeChain {
    pub guard: ProxyNode,
    pub middle: ProxyNode,
    pub exit: ProxyNode,
}

#[derive(Debug, Error)]
pub enum NodeSelectError {
    #[error("http fetch error: {0}")]
    Fetch(String),
    #[error("response decode error: {0}")]
    Decode(String),
    #[error("key error: {0}")]
    Key(String),
    #[error("not enough nodes with three distinct country codes (need ≥ 3, got {got})")]
    InsufficientDiversity { got: usize },
}

/// Fetch the active proxy list from `server_url` and randomly select three
/// nodes from distinct countries.
///
/// `server_url` must point to a JSON endpoint that returns `Vec<ProxyNode>`.
pub async fn fetch_and_select_chain(server_url: &str) -> Result<NodeChain, NodeSelectError> {
    let nodes = fetch_proxy_list(server_url).await?;
    select_chain_from(nodes)
}

pub(crate) async fn fetch_proxy_list(server_url: &str) -> Result<Vec<ProxyNode>, NodeSelectError> {
    let client = reqwest::Client::builder()
        .https_only(true)
        .build()
        .map_err(|e| NodeSelectError::Fetch(format!("client build error: {e}")))?;

    let response = client
        .get(server_url)
        .send()
        .await
        .map_err(|e| NodeSelectError::Fetch(format!("GET error: {e}")))?;

    if !response.status().is_success() {
        return Err(NodeSelectError::Fetch(format!(
            "server returned HTTP {}",
            response.status().as_u16()
        )));
    }

    response
        .json::<Vec<ProxyNode>>()
        .await
        .map_err(|e| NodeSelectError::Decode(format!("JSON decode error: {e}")))
}

/// Select Guard, Middle and Exit nodes from an already-fetched list so that
/// all three originate from **different** country codes.
///
/// Selection is uniform-random via `OsRng`; each call may return a different
/// combination.
pub fn select_chain_from(nodes: Vec<ProxyNode>) -> Result<NodeChain, NodeSelectError> {
    let mut rng = OsRng;

    // Group nodes by country code. Keys are sorted for deterministic shuffling
    // reproducibility in tests (the shuffle itself is still random).
    let mut by_country: HashMap<String, Vec<ProxyNode>> = HashMap::new();
    for node in nodes {
        by_country
            .entry(node.country_code.clone())
            .or_default()
            .push(node);
    }

    let country_count = by_country.len();
    if country_count < 3 {
        return Err(NodeSelectError::InsufficientDiversity { got: country_count });
    }

    // Shuffle country list with a fresh random permutation.
    let mut countries: Vec<String> = by_country.keys().cloned().collect();
    countries.shuffle(&mut rng);

    // Randomly pick one node from each of the first three countries.
    let pick = |country: &str| -> ProxyNode {
        let pool = by_country.get(country).expect("country must exist");
        // pool is non-empty by construction; unwrap is safe.
        pool.choose(&mut OsRng).expect("pool is non-empty").clone()
    };

    // Consume 3 separate OsRng draws to ensure independence.
    let guard = pick(&countries[0]);
    let middle = pick(&countries[1]);
    let exit = pick(&countries[2]);

    Ok(NodeChain { guard, middle, exit })
}

/// Generate a random `u64` via OsRng – useful for request IDs without
/// introducing any external dependency.
pub fn random_request_id() -> u64 {
    let mut buf = [0u8; 8];
    OsRng.fill_bytes(&mut buf);
    u64::from_le_bytes(buf)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn node(id: &str, country: &str) -> ProxyNode {
    	ProxyNode {
    		id: id.to_owned(),
    		address: format!("{id}:9001"),
    		country_code: country.to_owned(),
    		public_key_b64: BASE64.encode([0u8; 32]),
			latitude: 0.0,
			longitude: 0.0,
    	}
    }

    #[test]
    fn selects_three_distinct_countries() {
    	let chain = select_chain_from(vec![
    		node("de", "DE"),
    		node("se", "SE"),
    		node("nl", "NL"),
    		node("de-2", "DE"),
    	])
    	.expect("three countries should be selectable");

    	let countries = [
    		chain.guard.country_code,
    		chain.middle.country_code,
    		chain.exit.country_code,
    	];
    	assert_ne!(countries[0], countries[1]);
    	assert_ne!(countries[0], countries[2]);
    	assert_ne!(countries[1], countries[2]);
    }

    #[test]
    fn rejects_insufficient_country_diversity() {
    	let error = select_chain_from(vec![node("de", "DE"), node("de-2", "DE")])
    		.expect_err("two countries cannot form a three-node chain");
    	assert!(matches!(
    		error,
    		NodeSelectError::InsufficientDiversity { got: 1 }
    	));
    }
}
