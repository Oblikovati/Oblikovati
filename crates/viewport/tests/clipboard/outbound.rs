// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeClipboardPeer {
    fn await_property(&self, endpoint: &NativeClipboardEndpoint) {
        let deadline = Instant::now() + Duration::from_secs(1);
        loop {
            assert!(
                Instant::now() < deadline,
                "native INCR peer: expected a property before deadline"
            );
            endpoint.service_pending(32).unwrap();
            if let Some(event) = self.connection.poll_for_event().unwrap() {
                if let Event::PropertyNotify(property) = event
                    && property.window == self.window
                    && property.atom == self.property
                    && property.state == Property::NEW_VALUE
                {
                    return;
                }
                continue;
            }
            self.wait_endpoints(endpoint, deadline);
        }
    }
    pub(super) fn read_incremental(&self, endpoint: &NativeClipboardEndpoint) -> Vec<u8> {
        let mut bytes = Vec::new();
        loop {
            self.await_property(endpoint);
            let chunk = self.property();
            assert_eq!(chunk.type_, self.atoms.utf8);
            assert_eq!(chunk.format, 8);
            if chunk.value.is_empty() {
                return bytes;
            }
            bytes.extend_from_slice(&chunk.value);
        }
    }
}
fn begin_peer_transfer(fixture: &NativeClipboardFixture, text: &str) {
    fixture.endpoint.publish_text(text).unwrap();
    fixture.transfer(fixture.peer.atoms.utf8);
    let header = fixture.peer.property();
    assert_eq!(header.type_, fixture.peer.atoms.incr);
    assert_eq!(header.format, 32);
    assert_eq!(
        header.value32().unwrap().collect::<Vec<_>>(),
        [text.len() as u32]
    );
}
#[test]
fn large_unicode_is_sent_in_incremental_properties_and_ends_with_empty_text() {
    let fixture = NativeClipboardFixture::new();
    let text = "é🙂".repeat(200);
    begin_peer_transfer(&fixture, &text);
    assert_eq!(
        fixture.peer.read_incremental(&fixture.endpoint),
        text.as_bytes()
    );
    assert!(fixture.endpoint.transfer_deadline().is_none());
}
#[test]
fn accepted_transfer_retains_its_snapshot_after_publish_and_ownership_changes() {
    let fixture = NativeClipboardFixture::new();
    let text = "old snapshot 🙂".repeat(100);
    begin_peer_transfer(&fixture, &text);
    fixture.endpoint.publish_text("new snapshot").unwrap();
    fixture
        .peer
        .connection
        .set_selection_owner(
            fixture.peer.window,
            fixture.peer.atoms.clipboard,
            x11rb::CURRENT_TIME,
        )
        .unwrap()
        .check()
        .unwrap();
    assert_eq!(
        fixture.peer.read_incremental(&fixture.endpoint),
        text.as_bytes()
    );
    fixture.endpoint.service_pending(32).unwrap();
    assert_eq!(*fixture.endpoint.0.snapshot.borrow(), None);
    assert!(fixture.endpoint.transfer_deadline().is_none());
}

#[test]
fn stalled_incremental_send_retires_its_snapshot_at_the_transfer_deadline() {
    let fixture = NativeClipboardFixture::new();
    let options = NativeClipboardOptions {
        byte_limit: 4096,
        target_limit: 64,
        chunk_limit: 32,
        transfer_limit: 16,
        transfer_timeout: Duration::from_millis(40),
    };
    let endpoint =
        NativeClipboardEndpoint::connect(Some(fixture._server.display()), options).unwrap();
    endpoint.publish_text(&"x".repeat(256)).unwrap();
    fixture.peer.request(fixture.peer.atoms.utf8);
    fixture.peer.receive(&endpoint);
    assert!(endpoint.transfer_deadline().is_some());
    std::thread::sleep(Duration::from_millis(60));
    assert!(
        endpoint
            .service_pending(32)
            .unwrap_err()
            .contains("deadline")
    );
    assert_eq!(endpoint.transfer_deadline(), None);
    endpoint.publish_text("healthy").unwrap();
    assert_eq!(
        crate::ViewportClipboard::read_text(&endpoint)
            .unwrap()
            .as_deref(),
        Some("healthy")
    );
}
#[test]
fn requestor_destruction_retires_pending_incremental_resources() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text(&"x".repeat(256)).unwrap();
    fixture.transfer(fixture.peer.atoms.utf8);
    assert!(fixture.endpoint.transfer_deadline().is_some());
    fixture
        .peer
        .connection
        .destroy_window(fixture.peer.window)
        .unwrap()
        .check()
        .unwrap();
    fixture.endpoint.service_pending(32).unwrap();
    assert_eq!(fixture.endpoint.transfer_deadline(), None);
}
