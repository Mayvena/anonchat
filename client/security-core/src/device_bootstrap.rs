use ed25519_dalek::{Signature, Signer, SigningKey, VerifyingKey};
use libsignal_client::{
    DeviceId, IdentityKeyPair, KeyPair, KyberPreKeyRecord, PreKeyBundle, PreKeyId,
    SignedPreKeyId, SignedPreKeyRecord, Timestamp, kem,
};
use rand::{RngCore, rngs::OsRng};
use serde::{Deserialize, Serialize};
use thiserror::Error;
use zeroize::{Zeroize, ZeroizeOnDrop, Zeroizing};

const DEFAULT_DEVICE_ID: u32 = 1;
const DEFAULT_PREKEY_ID: u32 = 1001;
const DEFAULT_SIGNED_PREKEY_ID: u32 = 2001;
const DEFAULT_KYBER_PREKEY_ID: u32 = 3001;

pub trait EncryptedDeviceMaterialStore {
    fn load_encrypted(&self) -> Result<Option<Vec<u8>>, BootstrapError>;
    fn save_encrypted(&self, ciphertext: &[u8]) -> Result<(), BootstrapError>;
}

pub trait KeyMaterialProtector {
    fn encrypt(&self, plaintext: &[u8]) -> Result<Vec<u8>, BootstrapError>;
    fn decrypt(&self, ciphertext: &[u8]) -> Result<Vec<u8>, BootstrapError>;
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BootstrapArtifact {
    pub registration_id: u32,
    pub device_id: u32,
    pub signed_temp_nickname: Vec<u8>,
    pub temp_nickname_pubkey: Vec<u8>,
    pub prekey_bundle: Vec<u8>,
}

#[derive(Debug)]
pub struct DeviceBootstrapOutput {
    pub artifact: BootstrapArtifact,
    pub nickname_signature: Signature,
    pub nickname_verify_key: VerifyingKey,
}

#[derive(Debug, Serialize, Deserialize, Zeroize, ZeroizeOnDrop)]
struct SecretEnvelope {
    registration_id: u32,
    device_id: u32,
    nickname_signing_key: Vec<u8>,
    signal_identity_pair: Vec<u8>,
    signed_prekey_record: Vec<u8>,
    kyber_prekey_record: Vec<u8>,
}

#[derive(Debug, Error)]
pub enum BootstrapError {
    #[error("store error: {0}")]
    Store(String),
    #[error("serialization error: {0}")]
    Serialization(String),
    #[error("crypto error: {0}")]
    Crypto(String),
    #[error("invalid data: {0}")]
    InvalidData(String),
}

pub fn bootstrap_or_load(
    temp_nickname: &str,
    store: &dyn EncryptedDeviceMaterialStore,
    protector: &dyn KeyMaterialProtector,
) -> Result<DeviceBootstrapOutput, BootstrapError> {
    let trimmed = temp_nickname.trim();
    if trimmed.is_empty() {
        return Err(BootstrapError::InvalidData(
            "temporary nickname cannot be empty".to_owned(),
        ));
    }

    if let Some(existing_ciphertext) = store.load_encrypted()? {
        return load_existing(trimmed, &existing_ciphertext, protector);
    }

    generate_first_bootstrap(trimmed, store, protector)
}

fn load_existing(
    temp_nickname: &str,
    ciphertext: &[u8],
    protector: &dyn KeyMaterialProtector,
) -> Result<DeviceBootstrapOutput, BootstrapError> {
    let plaintext = protector.decrypt(ciphertext)?;
    let mut secret_blob = Zeroizing::new(plaintext);

    let mut envelope: SecretEnvelope =
        serde_json::from_slice(secret_blob.as_slice()).map_err(|e| {
            BootstrapError::Serialization(format!("failed to decode encrypted envelope: {e}"))
        })?;

    let signing_key_bytes: [u8; 32] = envelope
        .nickname_signing_key
        .as_slice()
        .try_into()
        .map_err(|_| BootstrapError::InvalidData("invalid Ed25519 private key bytes".to_owned()))?;
    let signing_key = SigningKey::from_bytes(&signing_key_bytes);
    let verify_key = signing_key.verifying_key();

    let nickname_signature = signing_key.sign(temp_nickname.as_bytes());

    let identity_pair = IdentityKeyPair::deserialize(&envelope.signal_identity_pair)
        .map_err(|e| BootstrapError::Crypto(format!("failed to deserialize identity pair: {e}")))?;

    let signed_prekey = SignedPreKeyRecord::deserialize(&envelope.signed_prekey_record).map_err(
        |e| BootstrapError::Crypto(format!("failed to deserialize signed pre-key: {e}")),
    )?;

    let kyber_prekey = KyberPreKeyRecord::deserialize(&envelope.kyber_prekey_record).map_err(
        |e| BootstrapError::Crypto(format!("failed to deserialize kyber pre-key: {e}")),
    )?;

    let bundle = build_prekey_bundle(
        envelope.registration_id,
        envelope.device_id,
        &identity_pair,
        &signed_prekey,
        &kyber_prekey,
    )?;

    let serialized_bundle = bundle
        .serialize()
        .map_err(|e| BootstrapError::Crypto(format!("failed to serialize pre-key bundle: {e}")))?;

    envelope.zeroize();

    Ok(DeviceBootstrapOutput {
        artifact: BootstrapArtifact {
            registration_id: bundle
                .registration_id()
                .map_err(|e| BootstrapError::Crypto(format!("registration id error: {e}")))?,
            device_id: u32::from(
                bundle
                    .device_id()
                    .map_err(|e| BootstrapError::Crypto(format!("device id error: {e}")))?,
            ),
            signed_temp_nickname: nickname_signature.to_vec(),
            temp_nickname_pubkey: verify_key.to_bytes().to_vec(),
            prekey_bundle: serialized_bundle,
        },
        nickname_signature,
        nickname_verify_key: verify_key,
    })
}

fn generate_first_bootstrap(
    temp_nickname: &str,
    store: &dyn EncryptedDeviceMaterialStore,
    protector: &dyn KeyMaterialProtector,
) -> Result<DeviceBootstrapOutput, BootstrapError> {
    let mut rng = OsRng;

    let mut registration_id_buf = [0u8; 4];
    rng.fill_bytes(&mut registration_id_buf);
    let registration_id = u32::from_le_bytes(registration_id_buf);

    let nickname_signing = SigningKey::generate(&mut rng);
    let nickname_verify = nickname_signing.verifying_key();
    let nickname_signature = nickname_signing.sign(temp_nickname.as_bytes());

    let identity_pair = IdentityKeyPair::generate(&mut rng);
    let signed_prekey_pair = KeyPair::generate(&mut rng);
    let signed_prekey_public = signed_prekey_pair.public_key.serialize();

    let signed_prekey_signature = identity_pair
        .private_key()
        .calculate_signature(&signed_prekey_public, &mut rng)
        .map_err(|e| BootstrapError::Crypto(format!("signed pre-key signature failed: {e}")))?;

    let signed_prekey = SignedPreKeyRecord::new(
        SignedPreKeyId::from(DEFAULT_SIGNED_PREKEY_ID),
        Timestamp::from_epoch_millis(now_millis()),
        &signed_prekey_pair,
        &signed_prekey_signature,
    );

    let kyber_prekey = KyberPreKeyRecord::generate(
        kem::KeyType::Kyber1024,
        DEFAULT_KYBER_PREKEY_ID.into(),
        identity_pair.private_key(),
    )
    .map_err(|e| BootstrapError::Crypto(format!("kyber pre-key generation failed: {e}")))?;

    let bundle = build_prekey_bundle(
        registration_id,
        DEFAULT_DEVICE_ID,
        &identity_pair,
        &signed_prekey,
        &kyber_prekey,
    )?;

    let serialized_bundle = bundle
        .serialize()
        .map_err(|e| BootstrapError::Crypto(format!("pre-key bundle serialization failed: {e}")))?;

    let signed_prekey_serialized = signed_prekey
        .serialize()
        .map_err(|e| BootstrapError::Crypto(format!("signed pre-key serialization failed: {e}")))?;
    let kyber_prekey_serialized = kyber_prekey
        .serialize()
        .map_err(|e| BootstrapError::Crypto(format!("kyber pre-key serialization failed: {e}")))?;

    let identity_serialized = identity_pair
        .serialize()
        .map_err(|e| BootstrapError::Crypto(format!("identity pair serialization failed: {e}")))?;

    let mut envelope = SecretEnvelope {
        registration_id,
        device_id: DEFAULT_DEVICE_ID,
        nickname_signing_key: nickname_signing.to_bytes().to_vec(),
        signal_identity_pair: identity_serialized,
        signed_prekey_record: signed_prekey_serialized,
        kyber_prekey_record: kyber_prekey_serialized,
    };

    let plaintext = serde_json::to_vec(&envelope)
        .map_err(|e| BootstrapError::Serialization(format!("failed to encode envelope: {e}")))?;

    let encrypted = protector.encrypt(&plaintext)?;
    store.save_encrypted(&encrypted)?;

    envelope.zeroize();
    let mut plaintext = Zeroizing::new(plaintext);
    plaintext.zeroize();

    Ok(DeviceBootstrapOutput {
        artifact: BootstrapArtifact {
            registration_id,
            device_id: DEFAULT_DEVICE_ID,
            signed_temp_nickname: nickname_signature.to_vec(),
            temp_nickname_pubkey: nickname_verify.to_bytes().to_vec(),
            prekey_bundle: serialized_bundle,
        },
        nickname_signature,
        nickname_verify_key: nickname_verify,
    })
}

fn build_prekey_bundle(
    registration_id: u32,
    device_id: u32,
    identity_pair: &IdentityKeyPair,
    signed_prekey: &SignedPreKeyRecord,
    kyber_prekey: &KyberPreKeyRecord,
) -> Result<PreKeyBundle, BootstrapError> {
    let one_time_prekey = KeyPair::generate(&mut OsRng);

    let signed_pub = signed_prekey
        .public_key()
        .map_err(|e| BootstrapError::Crypto(format!("signed pre-key public extraction failed: {e}")))?;
    let signed_sig = signed_prekey
        .signature()
        .map_err(|e| BootstrapError::Crypto(format!("signed pre-key signature extraction failed: {e}")))?;

    let kyber_pub = kyber_prekey
        .public_key()
        .map_err(|e| BootstrapError::Crypto(format!("kyber public extraction failed: {e}")))?;
    let kyber_sig = kyber_prekey
        .signature()
        .map_err(|e| BootstrapError::Crypto(format!("kyber signature extraction failed: {e}")))?;

    PreKeyBundle::new(
        registration_id,
        DeviceId::new(device_id)
            .map_err(|e| BootstrapError::InvalidData(format!("invalid device id: {e}")))?,
        Some((PreKeyId::from(DEFAULT_PREKEY_ID), one_time_prekey.public_key)),
        SignedPreKeyId::from(DEFAULT_SIGNED_PREKEY_ID),
        signed_pub,
        signed_sig,
        DEFAULT_KYBER_PREKEY_ID.into(),
        kyber_pub,
        kyber_sig,
        *identity_pair.identity_key(),
    )
    .map_err(|e| BootstrapError::Crypto(format!("pre-key bundle construction failed: {e}")))
}

fn now_millis() -> u64 {
    let now = std::time::SystemTime::now();
    let ts = now
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_else(|_| std::time::Duration::from_secs(0));
    ts.as_millis() as u64
}

#[cfg(test)]
mod tests {
    use super::*;
    use ed25519_dalek::Verifier;
    use std::cell::RefCell;

    #[derive(Default)]
    struct MemStore {
        blob: RefCell<Option<Vec<u8>>>,
        saves: RefCell<u32>,
    }

    impl EncryptedDeviceMaterialStore for MemStore {
        fn load_encrypted(&self) -> Result<Option<Vec<u8>>, BootstrapError> {
            Ok(self.blob.borrow().clone())
        }
        fn save_encrypted(&self, ciphertext: &[u8]) -> Result<(), BootstrapError> {
            *self.blob.borrow_mut() = Some(ciphertext.to_vec());
            *self.saves.borrow_mut() += 1;
            Ok(())
        }
    }

    // Test-only reversible transform; real protectors must use an AEAD.
    struct XorProtector(u8);

    impl KeyMaterialProtector for XorProtector {
        fn encrypt(&self, plaintext: &[u8]) -> Result<Vec<u8>, BootstrapError> {
            Ok(plaintext.iter().map(|b| b ^ self.0).collect())
        }
        fn decrypt(&self, ciphertext: &[u8]) -> Result<Vec<u8>, BootstrapError> {
            Ok(ciphertext.iter().map(|b| b ^ self.0).collect())
        }
    }

    fn contains(haystack: &[u8], needle: &[u8]) -> bool {
        haystack.windows(needle.len()).any(|w| w == needle)
    }

    #[test]
    fn first_run_signs_nickname_and_stores_only_ciphertext() {
        let store = MemStore::default();
        let out = bootstrap_or_load("temp-nick", &store, &XorProtector(0xA5)).unwrap();

        out.nickname_verify_key
            .verify(b"temp-nick", &out.nickname_signature)
            .expect("signature must verify");
        assert!(!out.artifact.prekey_bundle.is_empty());
        assert_eq!(*store.saves.borrow(), 1);

        let stored = store.blob.borrow().clone().unwrap();
        assert!(!contains(&stored, b"nickname_signing_key"));
        assert!(!contains(&stored, b"signal_identity_pair"));
    }

    #[test]
    fn second_run_loads_same_identity_without_resaving() {
        let store = MemStore::default();
        let protector = XorProtector(0x5A);
        let first = bootstrap_or_load("temp-nick", &store, &protector).unwrap();
        let second = bootstrap_or_load("temp-nick", &store, &protector).unwrap();

        assert_eq!(*store.saves.borrow(), 1);
        assert_eq!(
            first.artifact.temp_nickname_pubkey,
            second.artifact.temp_nickname_pubkey
        );
        assert_eq!(first.artifact.registration_id, second.artifact.registration_id);
        second
            .nickname_verify_key
            .verify(b"temp-nick", &second.nickname_signature)
            .unwrap();
    }

    #[test]
    fn empty_nickname_is_rejected_and_nothing_is_stored() {
        let store = MemStore::default();
        let err = bootstrap_or_load("   ", &store, &XorProtector(1)).unwrap_err();
        assert!(matches!(err, BootstrapError::InvalidData(_)));
        assert!(store.blob.borrow().is_none());
    }

    #[test]
    fn corrupted_ciphertext_fails_closed() {
        let store = MemStore::default();
        *store.blob.borrow_mut() = Some(vec![1, 2, 3, 4]);
        assert!(bootstrap_or_load("temp-nick", &store, &XorProtector(0x11)).is_err());
    }
}
