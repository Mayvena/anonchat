//! Onion routing module: node selection and multi-layer encryption.

pub mod cipher;
pub mod node_selector;

pub use cipher::{
    INNER_SIZE, LAYER_OVERHEAD, MAX_MESSAGE_BYTES, NUM_LAYERS, PACKET_SIZE, OnionError,
    build_onion_packet, build_onion_packet_str,
};
pub use node_selector::{NodeChain, NodeSelectError, ProxyNode, fetch_and_select_chain, select_chain_from};

use x25519_dalek::PublicKey as X25519PublicKey;

/// Convenience: select three proxy nodes from `nodes` and immediately build
/// the encrypted onion packet for `message`.
pub fn select_and_encrypt(
    nodes: Vec<ProxyNode>,
    message: &[u8],
) -> Result<([u8; PACKET_SIZE], NodeChain), OnionOrSelectError> {
    let chain = select_chain_from(nodes).map_err(OnionOrSelectError::Select)?;

    let guard_key: X25519PublicKey = chain.guard.x25519_pubkey().map_err(OnionOrSelectError::Select)?;
    let middle_key: X25519PublicKey = chain.middle.x25519_pubkey().map_err(OnionOrSelectError::Select)?;
    let exit_key: X25519PublicKey = chain.exit.x25519_pubkey().map_err(OnionOrSelectError::Select)?;

    let packet =
        build_onion_packet(message, &guard_key, &middle_key, &exit_key).map_err(OnionOrSelectError::Onion)?;

    Ok((packet, chain))
}

/// Fetch the active proxy registry, select three distinct countries, and
/// build the fixed-size onion packet in one operation.
pub async fn fetch_and_encrypt_chain(
    server_url: &str,
    message: &[u8],
) -> Result<([u8; PACKET_SIZE], NodeChain), OnionOrSelectError> {
    let nodes = node_selector::fetch_proxy_list(server_url)
        .await
        .map_err(OnionOrSelectError::Select)?;
    select_and_encrypt(nodes, message)
}

#[derive(Debug, thiserror::Error)]
pub enum OnionOrSelectError {
    #[error("node selection: {0}")]
    Select(#[from] NodeSelectError),
    #[error("onion encryption: {0}")]
    Onion(OnionError),
}
