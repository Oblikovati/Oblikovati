// SPDX-License-Identifier: GPL-2.0-only
//! The persisted view is valid only for the binary that produced it.
use sha2::{Digest, Sha256};
use std::io::Read;

pub(crate) fn current_executable_digest() -> Result<String, String> {
    let file = open_executable()?;
    digest_reader(file).map_err(|error| format!("assembly producer executable: {error}"))
}

#[cfg(target_os = "linux")]
fn open_executable() -> Result<std::fs::File, String> {
    std::fs::File::open("/proc/self/exe").map_err(|error| {
        format!("assembly producer /proc/self/exe: expected a readable running image: {error}")
    })
}

#[cfg(not(target_os = "linux"))]
fn open_executable() -> Result<std::fs::File, String> {
    Err(unsupported_platform_error(std::env::consts::OS))
}

#[cfg(any(not(target_os = "linux"), test))]
fn unsupported_platform_error(platform: &str) -> String {
    format!(
        "assembly producer on {}: expected a verified running executable handle; Express cache is unavailable",
        platform
    )
}

fn digest_reader(mut reader: impl Read) -> Result<String, String> {
    let mut digest = Sha256::new();
    let mut buffer = [0_u8; 65536];
    let mut total = 0_u64;
    loop {
        let count = match reader.read(&mut buffer) {
            Ok(count) => count,
            Err(error) if error.kind() == std::io::ErrorKind::Interrupted => continue,
            Err(error) => return Err(error.to_string()),
        };
        if count == 0 {
            break;
        }
        digest.update(&buffer[..count]);
        total += count as u64;
    }
    if total == 0 {
        return Err("expected a nonempty executable".into());
    }
    Ok(format!("{:x}", digest.finalize()))
}

#[cfg(test)]
#[path = "../tests/assembly_producer.rs"]
mod tests;
