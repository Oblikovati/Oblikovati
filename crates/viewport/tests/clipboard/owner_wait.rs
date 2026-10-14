// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportReady, ViewportWake};
use rustix::fd::AsFd;
use std::os::unix::net::UnixStream;
#[test]
fn idle_owner_wakes_for_an_external_request_and_retains_it_until_service() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("idle Unicode 🙂").unwrap();
    fixture.endpoint.service_pending(32).unwrap();
    let wake = ViewportWake::new().unwrap();
    assert_eq!(
        fixture
            .endpoint
            .wait_events(&wake, Some(Duration::ZERO))
            .unwrap(),
        ViewportReady::default()
    );
    fixture.peer.request(fixture.peer.atoms.utf8);
    wake.wake().unwrap();
    assert_eq!(
        fixture
            .endpoint
            .wait_events(&wake, Some(Duration::from_secs(1)))
            .unwrap(),
        ViewportReady {
            native_events: true,
            worker_wake: true
        }
    );
    assert!(
        fixture
            .endpoint
            .wait_events(&wake, Some(Duration::ZERO))
            .unwrap()
            .native_events
    );
    assert_ne!(fixture.peer.receive(&fixture.endpoint).property, 0);
    assert_eq!(fixture.peer.property().value, "idle Unicode 🙂".as_bytes());
}
#[test]
fn clipboard_owner_wait_preserves_window_and_worker_readiness() {
    let fixture = NativeClipboardFixture::new();
    let wake = ViewportWake::new().unwrap();
    let (window, window_peer) = UnixStream::pair().unwrap();
    rustix::io::write(&window_peer, b"window").unwrap();
    wake.wake().unwrap();
    assert_eq!(
        fixture
            .endpoint
            .wait_with_sources(&wake, &[window.as_fd()], None)
            .unwrap(),
        ViewportReady {
            native_events: true,
            worker_wake: true
        }
    );
}
#[test]
fn incremental_expiry_wakes_the_owner_without_a_window_event() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text(&"x".repeat(256)).unwrap();
    fixture.transfer(fixture.peer.atoms.utf8);
    fixture.endpoint.service_pending(32).unwrap();
    let wake = ViewportWake::new().unwrap();
    assert!(
        fixture
            .endpoint
            .wait_events(&wake, None)
            .unwrap()
            .native_events
    );
    assert!(
        fixture
            .endpoint
            .service_pending(32)
            .unwrap_err()
            .contains("deadline")
    );
    assert_eq!(fixture.endpoint.transfer_deadline(), None);
}
