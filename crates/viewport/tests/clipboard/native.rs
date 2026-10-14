// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::x11::private_server::PrivateX11Server;
struct NativeClipboardPeer {
    connection: RustConnection,
    window: Window,
    atoms: ClipboardAtoms,
    property: Atom,
}
impl NativeClipboardPeer {
    fn connect(display: &str) -> Self {
        let (connection, screen) = RustConnection::connect(Some(display)).unwrap();
        let window = create_selection_window(&connection, screen).unwrap();
        let atoms = ClipboardAtoms::load(&connection).unwrap();
        let property = connection
            .intern_atom(false, b"_OBLIKOVATI_CLIPBOARD_PEER_RECEIPT")
            .unwrap()
            .reply()
            .unwrap()
            .atom;
        Self {
            connection,
            window,
            atoms,
            property,
        }
    }
    fn request(&self, target: Atom) {
        self.connection
            .convert_selection(
                self.window,
                self.atoms.clipboard,
                target,
                self.property,
                x11rb::CURRENT_TIME,
            )
            .unwrap()
            .check()
            .unwrap();
    }
    fn receive(&self, endpoint: &NativeClipboardEndpoint) -> SelectionNotifyEvent {
        let deadline = Instant::now() + Duration::from_secs(1);
        loop {
            assert!(
                Instant::now() < deadline,
                "native clipboard peer: expected notification before deadline"
            );
            endpoint.service_pending(32).unwrap();
            if let Some(event) = self.connection.poll_for_event().unwrap() {
                if let Event::SelectionNotify(notification) = event {
                    return notification;
                }
                continue;
            }
            self.wait_endpoints(endpoint, deadline);
        }
    }
    fn wait_endpoints(&self, endpoint: &NativeClipboardEndpoint, deadline: Instant) {
        use rustix::event::{PollFd, PollFlags, Timespec, poll};
        let timeout =
            Timespec::try_from(deadline.saturating_duration_since(Instant::now())).unwrap();
        let mut descriptors = [
            PollFd::new(self.connection.stream(), PollFlags::IN),
            PollFd::new(endpoint.0.connection.stream(), PollFlags::IN),
        ];
        poll(&mut descriptors, Some(&timeout)).unwrap();
    }
    fn property(&self) -> GetPropertyReply {
        self.connection
            .get_property(true, self.window, self.property, AtomEnum::ANY, 0, 1024)
            .unwrap()
            .reply()
            .unwrap()
    }
}
struct NativeClipboardFixture {
    endpoint: NativeClipboardEndpoint,
    peer: NativeClipboardPeer,
    _server: PrivateX11Server,
}
impl NativeClipboardFixture {
    fn new() -> Self {
        let server = PrivateX11Server::open_compose_fixture().unwrap();
        let options = NativeClipboardOptions {
            byte_limit: 4096,
            target_limit: 64,
            chunk_limit: 64,
            transfer_limit: 16,
            transfer_timeout: Duration::from_secs(1),
        };
        let endpoint = NativeClipboardEndpoint::connect(Some(server.display()), options).unwrap();
        let peer = NativeClipboardPeer::connect(server.display());
        Self {
            endpoint,
            peer,
            _server: server,
        }
    }
    fn transfer(&self, target: Atom) -> SelectionNotifyEvent {
        self.peer.request(target);
        self.peer.receive(&self.endpoint)
    }
}
#[test]
fn independent_native_client_reads_unicode_empty_text_and_advertised_targets() {
    let fixture = NativeClipboardFixture::new();
    for text in ["é界🙂", ""] {
        fixture.endpoint.publish_text(text).unwrap();
        assert_eq!(
            fixture.transfer(fixture.peer.atoms.utf8).property,
            fixture.peer.property
        );
        let property = fixture.peer.property();
        assert_eq!(property.type_, fixture.peer.atoms.utf8);
        assert_eq!(property.format, 8);
        assert_eq!(property.value, text.as_bytes());
    }
    fixture.transfer(fixture.peer.atoms.targets);
    let targets = fixture.peer.property();
    let atoms: Vec<u32> = targets.value32().unwrap().collect();
    assert_eq!(
        atoms,
        [
            fixture.peer.atoms.targets,
            fixture.peer.atoms.timestamp,
            fixture.peer.atoms.multiple,
            fixture.peer.atoms.utf8,
            fixture.peer.atoms.save_targets,
            fixture.peer.atoms.target_sizes,
            u32::from(AtomEnum::STRING)
        ]
    );
}
#[test]
fn native_owner_reports_acquisition_time_and_refuses_unknown_targets() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("owned").unwrap();
    fixture.transfer(fixture.peer.atoms.timestamp);
    let timestamp = fixture.peer.property();
    assert_eq!(timestamp.format, 32);
    assert_eq!(timestamp.value32().unwrap().collect::<Vec<_>>().len(), 1);
    assert_eq!(fixture.transfer(AtomEnum::PIXMAP.into()).property, 0);
}
#[test]
fn clipboard_options_and_text_are_validated_before_ownership_changes() {
    let options = NativeClipboardOptions {
        byte_limit: 0,
        target_limit: 64,
        chunk_limit: 64,
        transfer_limit: 16,
        transfer_timeout: Duration::from_secs(1),
    };
    assert!(options.validate().is_err());
    assert!(
        NativeClipboardOptions {
            byte_limit: 1,
            target_limit: 64,
            chunk_limit: 64,
            transfer_limit: 16,
            transfer_timeout: Duration::ZERO
        }
        .validate()
        .is_err()
    );
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("retained").unwrap();
    assert!(fixture.endpoint.publish_text("a\0b").is_err());
    assert!(fixture.endpoint.publish_text(&"x".repeat(4097)).is_err());
    fixture.transfer(fixture.peer.atoms.utf8);
    assert_eq!(fixture.peer.property().value, b"retained");
}

#[test]
fn native_selection_clear_releases_only_the_current_owned_snapshot() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("owned").unwrap();
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
    fixture.endpoint.service_pending(32).unwrap();
    assert_eq!(*fixture.endpoint.0.snapshot.borrow(), None);
}
#[test]
fn a_destroyed_requestor_returns_a_checked_error_without_killing_the_endpoint() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("retained").unwrap();
    fixture.peer.request(fixture.peer.atoms.utf8);
    fixture
        .peer
        .connection
        .destroy_window(fixture.peer.window)
        .unwrap()
        .check()
        .unwrap();
    assert!(
        fixture
            .endpoint
            .service_pending(32)
            .unwrap_err()
            .contains("clipboard")
    );
    let next = NativeClipboardPeer::connect(fixture._server.display());
    next.request(next.atoms.utf8);
    assert_ne!(next.receive(&fixture.endpoint).property, 0);
    assert_eq!(next.property().value, b"retained");
}

#[path = "foreign.rs"]
mod foreign_tests;

#[path = "outbound.rs"]
mod outbound_tests;

#[path = "multiple.rs"]
mod multiple_tests;

#[path = "owner_wait.rs"]
mod owner_wait_tests;

#[path = "shutdown.rs"]
mod shutdown_tests;
