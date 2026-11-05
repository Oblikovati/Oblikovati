// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn destination(peer: &NativeClipboardPeer, name: &[u8]) -> Atom {
    peer.connection
        .intern_atom(false, name)
        .unwrap()
        .reply()
        .unwrap()
        .atom
}
fn read_destination(peer: &NativeClipboardPeer, property: Atom) -> GetPropertyReply {
    peer.connection
        .get_property(true, peer.window, property, AtomEnum::ANY, 0, 1024)
        .unwrap()
        .reply()
        .unwrap()
}
#[test]
fn multiple_request_reports_each_conversion_and_declines_unsupported_targets() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("é").unwrap();
    let utf8 = destination(&fixture.peer, b"_OBLIKOVATI_MULTI_UTF8");
    let stamp = destination(&fixture.peer, b"_OBLIKOVATI_MULTI_TIME");
    let unsupported = destination(&fixture.peer, b"_OBLIKOVATI_MULTI_UNSUPPORTED");
    let pairs = [
        fixture.peer.atoms.utf8,
        utf8,
        fixture.peer.atoms.timestamp,
        stamp,
        u32::from(AtomEnum::PIXMAP),
        unsupported,
    ];
    fixture
        .peer
        .connection
        .change_property32(
            PropMode::REPLACE,
            fixture.peer.window,
            fixture.peer.property,
            fixture.peer.atoms.atom_pair,
            &pairs,
        )
        .unwrap()
        .check()
        .unwrap();
    assert_eq!(
        fixture.transfer(fixture.peer.atoms.multiple).property,
        fixture.peer.property
    );
    let result = fixture
        .peer
        .property()
        .value32()
        .unwrap()
        .collect::<Vec<_>>();
    assert_eq!(result, [pairs[0], utf8, pairs[2], stamp, pairs[4], 0]);
    assert_eq!(read_destination(&fixture.peer, utf8).value, "é".as_bytes());
    assert_eq!(read_destination(&fixture.peer, stamp).format, 32);
}
#[test]
fn multiple_request_cannot_overwrite_its_pair_list_or_alias_destinations() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("text").unwrap();
    let output = destination(&fixture.peer, b"_OBLIKOVATI_MULTI_ALIASED");
    let target = fixture.peer.atoms.utf8;
    let pairs = [
        target,
        fixture.peer.property,
        target,
        output,
        target,
        output,
        fixture.peer.atoms.multiple,
        output,
    ];
    fixture
        .peer
        .connection
        .change_property32(
            PropMode::REPLACE,
            fixture.peer.window,
            fixture.peer.property,
            fixture.peer.atoms.atom_pair,
            &pairs,
        )
        .unwrap()
        .check()
        .unwrap();
    fixture.transfer(fixture.peer.atoms.multiple);
    assert_eq!(
        fixture
            .peer
            .property()
            .value32()
            .unwrap()
            .collect::<Vec<_>>(),
        [target, 0, target, output, target, 0, pairs[6], 0]
    );
    assert_eq!(read_destination(&fixture.peer, output).value, b"text");
}
#[test]
fn latin1_is_advertised_and_encoded_only_when_the_snapshot_is_representable() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("é").unwrap();
    fixture.transfer(AtomEnum::STRING.into());
    assert_eq!(fixture.peer.property().value, [0xe9]);
    fixture.endpoint.publish_text("🙂").unwrap();
    fixture.transfer(fixture.peer.atoms.targets);
    assert!(
        !fixture
            .peer
            .property()
            .value32()
            .unwrap()
            .any(|target| target == u32::from(AtomEnum::STRING))
    );
    assert_eq!(fixture.transfer(AtomEnum::STRING.into()).property, 0);
}
#[test]
fn pending_transfer_budget_declines_new_requests_without_disturbing_the_first() {
    let fixture = NativeClipboardFixture::new();
    let options = NativeClipboardOptions {
        byte_limit: 4096,
        target_limit: 64,
        chunk_limit: 32,
        transfer_limit: 1,
        transfer_timeout: Duration::from_secs(1),
    };
    let endpoint =
        NativeClipboardEndpoint::connect(Some(fixture._server.display()), options).unwrap();
    endpoint.publish_text(&"x".repeat(256)).unwrap();
    fixture.peer.request(fixture.peer.atoms.utf8);
    assert_ne!(fixture.peer.receive(&endpoint).property, 0);
    let second = NativeClipboardPeer::connect(fixture._server.display());
    second.request(second.atoms.utf8);
    assert_eq!(second.receive(&endpoint).property, 0);
    assert!(endpoint.transfer_deadline().is_some());
    fixture.peer.property();
    assert_eq!(fixture.peer.read_incremental(&endpoint), vec![b'x'; 256]);
}

#[test]
fn malformed_multiple_request_is_notified_as_declined_and_keeps_the_owner_usable() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("retained").unwrap();
    fixture
        .peer
        .connection
        .change_property32(
            PropMode::REPLACE,
            fixture.peer.window,
            fixture.peer.property,
            fixture.peer.atoms.atom_pair,
            &[fixture.peer.atoms.utf8],
        )
        .unwrap()
        .check()
        .unwrap();
    fixture.peer.request(fixture.peer.atoms.multiple);
    assert!(
        fixture
            .endpoint
            .service_pending(32)
            .unwrap_err()
            .contains("ATOM_PAIR/32")
    );
    assert_eq!(fixture.peer.receive(&fixture.endpoint).property, 0);
    fixture.transfer(fixture.peer.atoms.utf8);
    assert_eq!(fixture.peer.property().value, b"retained");
}
