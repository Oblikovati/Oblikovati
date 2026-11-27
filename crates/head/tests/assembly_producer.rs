// SPDX-License-Identifier: GPL-2.0-only
use super::*;

struct FailingExecutable;
impl Read for FailingExecutable {
    fn read(&mut self, _: &mut [u8]) -> std::io::Result<usize> {
        Err(std::io::Error::other("unavailable executable"))
    }
}

struct InterruptedExecutable {
    interrupted: bool,
    content: &'static [u8],
}
impl Read for InterruptedExecutable {
    fn read(&mut self, buffer: &mut [u8]) -> std::io::Result<usize> {
        if !self.interrupted {
            self.interrupted = true;
            return Err(std::io::ErrorKind::Interrupted.into());
        }
        self.content.read(buffer)
    }
}

#[test]
fn producer_digest_changes_when_executable_changes() {
    let first = digest_reader(&b"first binary"[..]).unwrap();
    assert_eq!(first, digest_reader(&b"first binary"[..]).unwrap());
    assert_ne!(first, digest_reader(&b"second binary"[..]).unwrap());
    assert_eq!(first.len(), 64);
    let interrupted = InterruptedExecutable {
        interrupted: false,
        content: b"binary",
    };
    assert_eq!(
        digest_reader(interrupted).unwrap(),
        digest_reader(&b"binary"[..]).unwrap()
    );
}

#[test]
fn producer_digest_refuses_missing_or_empty_executable() {
    assert!(
        digest_reader(FailingExecutable)
            .unwrap_err()
            .contains("unavailable executable")
    );
    assert!(
        digest_reader(&b""[..])
            .unwrap_err()
            .contains("nonempty executable")
    );
}

#[test]
fn unsupported_platform_diagnostic_names_the_platform_and_reason() {
    for platform in ["windows", "macos"] {
        let error = unsupported_platform_error(platform);
        assert!(error.contains(platform), "{error}");
        assert!(
            error.contains("verified running executable handle"),
            "{error}"
        );
    }
}

#[test]
#[cfg(target_os = "linux")]
fn current_process_has_a_usable_producer_digest() {
    assert_eq!(current_executable_digest().unwrap().len(), 64);
}

#[test]
#[cfg(not(target_os = "linux"))]
fn unverifiable_running_image_is_refused() {
    let error = current_executable_digest().unwrap_err();
    assert!(
        error.contains("verified running executable handle"),
        "{error}"
    );
}
