//! Three-layer (onion) encryption.
//!
//! Packet layout guarantee:
//!   The outermost packet sent to the Guard node is **exactly** 4096 bytes.
//!
//! Each layer uses:
//!   X25519 ECDH  →  HKDF-SHA256  →  AES-256-GCM
//!
//! Per-layer byte overhead (LAYER_OVERHEAD = 60):
//!   32  ephemeral X25519 public key
//!   12  random AES-GCM nonce
//!   16  GCM authentication tag
//!
//! Inner payload size = 4096 - 3 × 60 = 3916 bytes.
//! The message is prefixed with a 4-byte big-endian length field; everything
//! after the message up to 3916 bytes is filled with random noise.
//! Messages longer than 3912 bytes are rejected.

use aes_gcm::{
    Aes256Gcm, Key, Nonce,
    aead::{Aead, KeyInit},
};
use hkdf::Hkdf;
use rand::RngCore;
use rand::rngs::OsRng;
use sha2::Sha256;
use thiserror::Error;
use x25519_dalek::{EphemeralSecret, PublicKey as X25519PublicKey};
use zeroize::Zeroizing;

/// Total fixed packet size (bytes).
pub const PACKET_SIZE: usize = 4096;
/// Number of onion layers.
pub const NUM_LAYERS: usize = 3;
/// Overhead added by each encryption layer (bytes).
pub const LAYER_OVERHEAD: usize = 32 + 12 + 16; // 60
/// Size of the innermost plaintext payload (bytes).
pub const INNER_SIZE: usize = PACKET_SIZE - NUM_LAYERS * LAYER_OVERHEAD; // 3916
/// Maximum raw message length that fits inside one packet (bytes).
pub const MAX_MESSAGE_BYTES: usize = INNER_SIZE - 4; // 3912 (4-byte length header)

#[derive(Debug, Error)]
pub enum OnionError {
    #[error("message too large: max {max} bytes, got {got}")]
    MessageTooLarge { max: usize, got: usize },
    #[error("encryption failed: {0}")]
    Encryption(String),
}

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

/// Pad `message` with cryptographically-random noise to exactly `INNER_SIZE`
/// bytes and then encrypt in three onion layers.
///
/// Encryption order (innermost first):
///   1. Exit public key   (layer 3, innermost)
///   2. Middle public key (layer 2)
///   3. Guard public key  (layer 1, outermost)
///
/// The returned packet is exactly `PACKET_SIZE` (4096) bytes.
pub fn build_onion_packet(
    message: &[u8],
    guard_pubkey: &X25519PublicKey,
    middle_pubkey: &X25519PublicKey,
    exit_pubkey: &X25519PublicKey,
) -> Result<[u8; PACKET_SIZE], OnionError> {
    if message.len() > MAX_MESSAGE_BYTES {
        return Err(OnionError::MessageTooLarge {
            max: MAX_MESSAGE_BYTES,
            got: message.len(),
        });
    }

    let inner = pad_to_inner(message)?;

    // Encrypt innermost → outermost.
    let exit_layer = encrypt_layer(&inner, exit_pubkey)?;
    let middle_layer = encrypt_layer(&exit_layer, middle_pubkey)?;
    let guard_layer = encrypt_layer(&middle_layer, guard_pubkey)?;

    // Compile-time size assertion; checked at runtime as a safety net.
    debug_assert_eq!(
        guard_layer.len(),
        PACKET_SIZE,
        "final onion packet size mismatch (this is a bug)"
    );

    let mut packet = [0u8; PACKET_SIZE];
    packet.copy_from_slice(&guard_layer);
    Ok(packet)
}

/// Convenience wrapper: accepts `&str` message text.
pub fn build_onion_packet_str(
    message: &str,
    guard_pubkey: &X25519PublicKey,
    middle_pubkey: &X25519PublicKey,
    exit_pubkey: &X25519PublicKey,
) -> Result<[u8; PACKET_SIZE], OnionError> {
    build_onion_packet(message.as_bytes(), guard_pubkey, middle_pubkey, exit_pubkey)
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

/// Encode `message` into a fixed `INNER_SIZE`-byte buffer.
///
/// Layout:
///   [0..4]               – message length as u32 big-endian
///   [4 .. 4+msg_len]     – message bytes
///   [4+msg_len .. INNER_SIZE] – cryptographically-random noise padding
fn pad_to_inner(message: &[u8]) -> Result<[u8; INNER_SIZE], OnionError> {
    let mut buf = [0u8; INNER_SIZE];

    let len_header = (message.len() as u32).to_be_bytes();
    buf[..4].copy_from_slice(&len_header);
    buf[4..4 + message.len()].copy_from_slice(message);

    // Fill the padding region with random noise (not zeros).
    OsRng.fill_bytes(&mut buf[4 + message.len()..]);

    Ok(buf)
}

/// Encrypt `payload` for `recipient_pubkey` using:
///   X25519 ECDH  →  HKDF-SHA256 (key only)  →  AES-256-GCM (random nonce)
///
/// Output layout:
///   [0..32]      – ephemeral X25519 public key
///   [32..44]     – random 12-byte AES-GCM nonce
///   [44..]       – AES-256-GCM ciphertext + 16-byte authentication tag
fn encrypt_layer(payload: &[u8], recipient_pubkey: &X25519PublicKey) -> Result<Vec<u8>, OnionError> {
    let mut rng = OsRng;

    // Ephemeral X25519 key pair for this layer.
    let ephemeral_secret = EphemeralSecret::random_from_rng(&mut rng);
    let ephemeral_public = X25519PublicKey::from(&ephemeral_secret);

    // Zeroize-on-drop shared secret.
    let shared = Zeroizing::new(ephemeral_secret.diffie_hellman(recipient_pubkey));

    // Derive 32-byte AES-256 key via HKDF-SHA256.
    let hkdf = Hkdf::<Sha256>::new(None, shared.as_bytes());
    let mut aes_key_bytes = Zeroizing::new([0u8; 32]);
    hkdf.expand(b"anonchat-onion-v1-key", aes_key_bytes.as_mut())
        .map_err(|e| OnionError::Encryption(format!("HKDF expand failed: {e}")))?;

    // Random 12-byte nonce (independent of key derivation).
    let mut nonce_bytes = [0u8; 12];
    rng.fill_bytes(&mut nonce_bytes);

    let aes_key = Key::<Aes256Gcm>::from_slice(aes_key_bytes.as_ref());
    let nonce = Nonce::from_slice(&nonce_bytes);
    let cipher = Aes256Gcm::new(aes_key);

    let ciphertext = cipher
        .encrypt(nonce, payload)
        .map_err(|e| OnionError::Encryption(format!("AES-256-GCM failed: {e}")))?;

    // Assemble output: ephemeral_pubkey ‖ nonce ‖ ciphertext(+tag)
    let mut output = Vec::with_capacity(LAYER_OVERHEAD + payload.len());
    output.extend_from_slice(ephemeral_public.as_bytes());
    output.extend_from_slice(&nonce_bytes);
    output.extend_from_slice(&ciphertext);

    Ok(output)
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use x25519_dalek::StaticSecret;

    fn dummy_keypair() -> (StaticSecret, X25519PublicKey) {
        let secret = StaticSecret::random_from_rng(OsRng);
        let public = X25519PublicKey::from(&secret);
        (secret, public)
    }

    #[test]
    fn constants_are_consistent() {
        assert_eq!(INNER_SIZE + NUM_LAYERS * LAYER_OVERHEAD, PACKET_SIZE);
        assert_eq!(INNER_SIZE, 3916);
        assert_eq!(LAYER_OVERHEAD, 60);
    }

    #[test]
    fn packet_is_exactly_4096_bytes() {
        let (_gs, gp) = dummy_keypair();
        let (_ms, mp) = dummy_keypair();
        let (_es, ep) = dummy_keypair();

        let msg = b"hello anonchat";
        let packet = build_onion_packet(msg, &gp, &mp, &ep).expect("encryption must succeed");
        assert_eq!(packet.len(), 4096);
    }

    #[test]
    fn maximum_length_message_succeeds() {
        let (_gs, gp) = dummy_keypair();
        let (_ms, mp) = dummy_keypair();
        let (_es, ep) = dummy_keypair();

        let msg = vec![0xABu8; MAX_MESSAGE_BYTES];
        let packet = build_onion_packet(&msg, &gp, &mp, &ep).expect("max-size message must succeed");
        assert_eq!(packet.len(), 4096);
    }

    #[test]
    fn oversized_message_is_rejected() {
        let (_gs, gp) = dummy_keypair();
        let (_ms, mp) = dummy_keypair();
        let (_es, ep) = dummy_keypair();

        let msg = vec![0u8; MAX_MESSAGE_BYTES + 1];
        assert!(matches!(
            build_onion_packet(&msg, &gp, &mp, &ep),
            Err(OnionError::MessageTooLarge { .. })
        ));
    }

    #[test]
    fn two_packets_of_same_message_differ() {
        let (_gs, gp) = dummy_keypair();
        let (_ms, mp) = dummy_keypair();
        let (_es, ep) = dummy_keypair();

        let msg = b"same message";
        let p1 = build_onion_packet(msg, &gp, &mp, &ep).unwrap();
        let p2 = build_onion_packet(msg, &gp, &mp, &ep).unwrap();
        // Random ephemeral keys + random nonces + random noise padding ensure
        // no two packets are identical.
        assert_ne!(p1.to_vec(), p2.to_vec());
    }
}
