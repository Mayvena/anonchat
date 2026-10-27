//! Panic-button destruction pipeline.
//!
//! `trigger_panic_destruction` performs, in order:
//!
//!  1. Cryptographic zeroization of all in-memory secret material via `zeroize`.
//!  2. Three-pass overwrite of the SQLCipher database file with CSPRNG noise.
//!  3. `std::process::exit(0)` – hard termination; no destructors run after
//!     this point, so no secret bytes can be leaked through normal drop paths.
//!
//! A successful return from the overwrite helper means that all three passes
//! were written and synchronised. On failure the process exits with status 1;
//! reporting success after an incomplete wipe would be unsafe.
//!
//! The file API can overwrite the logical database file and request a storage
//! sync, but cannot guarantee physical-sector destruction on SSDs, copy-on-write
//! filesystems, or storage devices with remapping. Such guarantees require
//! device-specific secure-erase support outside this crate.
//!
//! # Memory-safety contract
//! Secret buffers are wrapped in types that implement `Zeroize`.  After
//! calling `zeroize()` the bytes are unconditionally overwritten before the
//! binding goes out of scope.  The compiler is prevented from eliding these
//! writes by the `zeroize` crate's use of `core::ptr::write_volatile` and
//! compiler fences internally.

use std::fs::{File, OpenOptions};
use std::io::{Seek, SeekFrom, Write};
use std::path::Path;

use zeroize::{Zeroize, ZeroizeOnDrop};

/// Number of overwrite passes over the database file.
const OVERWRITE_PASSES: usize = 3;

/// Write buffer size per pass (must be a power-of-two for alignment; 64 KiB).
const WRITE_BUF_SIZE: usize = 64 * 1024;

// ─────────────────────────────────────────────────────────────────────────────
// Secret containers
// ─────────────────────────────────────────────────────────────────────────────

/// Holds the device private key that must be zeroized.
///
/// The `ZeroizeOnDrop` derive ensures the memory is wiped even if the
/// destructor path is taken instead of an explicit `zeroize()` call.
#[derive(Zeroize, ZeroizeOnDrop)]
pub struct PrivateKeyMaterial {
    pub bytes: Vec<u8>,
}

/// Holds an arbitrary number of symmetric session keys.
#[derive(Zeroize, ZeroizeOnDrop)]
pub struct SessionKeys {
    /// Each element is one session key (e.g. 32-byte AES-256 key).
    pub keys: Vec<Vec<u8>>,
}

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

/// Perform the full panic-button destruction pipeline and **exit the process**.
///
/// Steps:
///   1. Zeroize `private_key` and every key in `session_keys` in RAM.
///   2. Overwrite the SQLCipher database file at `db_path` with CSPRNG random
///      bytes three times and synchronise each pass.
///   3. Call `std::process::exit(0)` on success, or `exit(1)` on I/O failure.
///
/// This function never returns.
pub fn trigger_panic_destruction(
    private_key: &mut PrivateKeyMaterial,
    session_keys: &mut SessionKeys,
    db_path: &Path,
) -> ! {
    // ── Step 1: Zeroize in-memory secrets ────────────────────────────────────
    private_key.zeroize();
    session_keys.zeroize();

    // ── Step 2: Overwrite the logical SQLCipher file ─────────────────────────
    if overwrite_db_file(db_path).is_err() {
        // Do not disclose the database path or any secret-bearing details.
        eprintln!("panic destruction failed while overwriting the local database");
        std::process::exit(1);
    }

    // ── Step 3: Hard exit ────────────────────────────────────────────────────
    //
    // `exit(0)` bypasses Rust destructors.  All secret material has already
    // been zeroized above, so this is the desired behaviour: nothing in the
    // remaining drop chain can touch or leak secrets.
    std::process::exit(0)
}

// ─────────────────────────────────────────────────────────────────────────────
// File overwrite
// ─────────────────────────────────────────────────────────────────────────────

/// Overwrite every byte of the file at `db_path` with CSPRNG random bytes
/// `OVERWRITE_PASSES` times. Errors are returned so the caller cannot report a
/// successful destruction after an incomplete pass.
fn overwrite_db_file(db_path: &Path) -> std::io::Result<()> {
    let mut file = OpenOptions::new().read(true).write(true).open(db_path)?;
    let file_len = file.metadata()?.len();
    if file_len == 0 {
        return Ok(());
    }

    let mut buf = vec![0u8; WRITE_BUF_SIZE];

    for _ in 0..OVERWRITE_PASSES {
        // Rewind to the beginning of the file for each pass.
        file.seek(SeekFrom::Start(0))?;

        let mut remaining = file_len;
        while remaining > 0 {
            let chunk = remaining.min(WRITE_BUF_SIZE as u64) as usize;
            getrandom::fill(&mut buf[..chunk]).map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, format!("entropy failed: {e}")))?;

            file.write_all(&buf[..chunk])?;

            remaining -= chunk as u64;
        }

        // Flush OS page-cache to storage after each pass.
        file.flush()?;

        // Request the OS/filesystem to sync buffered writes to the physical
        // medium.  This is a best-effort call; some filesystems or storage
        // controllers may still journal or reorder writes.
        sync_file(&file)?;

        // Zeroize the write buffer between passes so transient state is not
        // readable in this function's stack frame.
        buf[..WRITE_BUF_SIZE.min(file_len as usize)].zeroize();
    }

    Ok(())
}

#[cfg(unix)]
fn sync_file(file: &File) -> std::io::Result<()> {
    use std::os::unix::io::AsRawFd;
    // SAFETY: file is a valid open file descriptor for the duration of this call.
    let ret = unsafe { libc_fsync(file.as_raw_fd()) };
    if ret == 0 {
        Ok(())
    } else {
        Err(std::io::Error::last_os_error())
    }
}

#[cfg(unix)]
fn libc_fsync(fd: i32) -> i32 {
    // Call fsync(2) directly via inline assembly or the standard extern "C"
    // block to avoid pulling in the `libc` crate as a runtime dependency.
    extern "C" {
        fn fsync(fd: i32) -> i32;
    }
    // SAFETY: `fsync` is a standard POSIX function; `fd` is a valid descriptor.
    unsafe { fsync(fd) }
}

#[cfg(windows)]
fn sync_file(file: &File) -> std::io::Result<()> {
    file.sync_all()
}

#[cfg(not(any(unix, windows)))]
fn sync_file(file: &File) -> std::io::Result<()> {
    file.sync_all()
}

// ─────────────────────────────────────────────────────────────────────────────
// Unit tests
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use std::io::Read;

    use super::*;

    /// Verify that `overwrite_db_file` fully covers the file with non-trivial
    /// bytes across all passes.  We test only the overwrite helper because
    /// `trigger_panic_destruction` calls `process::exit` which would kill the
    /// test runner.
    #[test]
    fn file_is_overwritten_with_noise() {
        use std::io::Write;

        let mut tmp = tempfile::NamedTempFile::new().expect("tempfile");
        // Write a recognisable sentinel payload.
        let original = vec![0xFFu8; 8192];
        tmp.write_all(&original).unwrap();
        tmp.flush().unwrap();

        overwrite_db_file(tmp.path()).expect("overwrite must succeed");

        // Re-read and confirm the sentinel is gone (i.e. the file is different
        // from the original sentinel content after the overwrite).
        let mut result = Vec::new();
        tmp.reopen()
            .unwrap()
            .read_to_end(&mut result)
            .unwrap();

        assert_eq!(result.len(), original.len(), "file size must be preserved");
        assert_ne!(
            result, original,
            "file content must differ from original sentinel after overwrite"
        );
    }

    #[test]
    fn zeroize_clears_private_key_bytes() {
        let secret = vec![0xABu8; 64];
        let mut key = PrivateKeyMaterial { bytes: secret };
        key.zeroize();
        assert!(key.bytes.iter().all(|&b| b == 0));
    }

    #[test]
    fn zeroize_clears_all_session_keys() {
        let mut sk = SessionKeys {
            keys: vec![vec![0xCDu8; 32], vec![0xEFu8; 32]],
        };
        sk.zeroize();
        for k in &sk.keys {
            assert!(k.iter().all(|&b| b == 0));
        }
    }

    #[test]
    fn missing_file_does_not_panic() {
        // Should complete silently even if the path does not exist.
        assert!(overwrite_db_file(Path::new("/nonexistent/path/db.sqlite")).is_err());
    }
}
