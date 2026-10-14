// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustix::fd::AsFd;
use std::os::unix::net::UnixStream;
#[test]
fn worker_wakes_coalesce_and_empty_reads_do_not_block() {
    let wake = ViewportWake::new().unwrap();
    let worker = wake.clone();
    assert!(!wake.consume().unwrap());
    for _ in 0..1000 {
        worker.wake().unwrap();
    }
    assert!(wake.consume().unwrap());
    assert!(!wake.consume().unwrap());
}
#[test]
fn wait_distinguishes_worker_native_timeout_and_broken_sources() {
    let (native, peer) = UnixStream::pair().unwrap();
    let wake = ViewportWake::new().unwrap();
    assert_eq!(
        wake.wait(native.as_fd(), Some(Duration::ZERO)).unwrap(),
        ViewportReady::default()
    );
    wake.wake().unwrap();
    assert_eq!(
        wake.wait(native.as_fd(), None).unwrap(),
        ViewportReady {
            native_events: false,
            worker_wake: true
        }
    );
    verify_native_and_failed_sources(&wake, &native, peer);
}
fn verify_native_and_failed_sources(wake: &ViewportWake, native: &UnixStream, peer: UnixStream) {
    rustix::io::write(&peer, b"event").unwrap();
    wake.wake().unwrap();
    assert_eq!(
        wake.wait(native.as_fd(), None).unwrap(),
        ViewportReady {
            native_events: true,
            worker_wake: true
        }
    );
    assert!(wake.wait(native.as_fd(), Some(Duration::MAX)).is_err());
    drop(peer);
    assert!(
        wake.wait(native.as_fd(), Some(Duration::ZERO))
            .unwrap_err()
            .contains("live event source")
    );
}
#[test]
fn multiple_native_sources_share_one_wait_with_worker_readiness() {
    let (window, _window_peer) = UnixStream::pair().unwrap();
    let (clipboard, clipboard_peer) = UnixStream::pair().unwrap();
    let wake = ViewportWake::new().unwrap();
    let sources = [window.as_fd(), clipboard.as_fd()];
    assert_eq!(
        wake.wait_sources(&sources, Some(Duration::ZERO)).unwrap(),
        ViewportReady::default()
    );
    rustix::io::write(&clipboard_peer, b"selection").unwrap();
    wake.wake().unwrap();
    assert_eq!(
        wake.wait_sources(&sources, None).unwrap(),
        ViewportReady {
            native_events: true,
            worker_wake: true
        }
    );
    drop(clipboard_peer);
    assert!(wake.wait_sources(&sources, Some(Duration::ZERO)).is_err());
}
#[test]
fn worker_only_wait_preserves_timeout_and_consumes_signal() {
    let wake = ViewportWake::new().unwrap();
    assert_eq!(
        wake.wait_sources(&[], Some(Duration::ZERO)).unwrap(),
        ViewportReady::default()
    );
    wake.wake().unwrap();
    assert_eq!(
        wake.wait_sources(&[], None).unwrap(),
        ViewportReady {
            native_events: false,
            worker_wake: true
        }
    );
}
