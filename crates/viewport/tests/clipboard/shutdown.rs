// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::NativeClipboardHandoff;
use std::{sync::mpsc, thread};
#[derive(Clone, Copy)]
enum RetentionBehavior {
    Save,
    Decline,
    Ignore,
    WrongProperty,
}
struct ClipboardRetentionPeer {
    stop: mpsc::Sender<()>,
    thread: Option<thread::JoinHandle<()>>,
}
impl ClipboardRetentionPeer {
    fn open(display: String, behavior: RetentionBehavior) -> Self {
        let (ready, received) = mpsc::sync_channel(1);
        let (stop, stopped) = mpsc::channel();
        let thread = thread::spawn(move || run_retention_peer(display, behavior, ready, stopped));
        received.recv_timeout(Duration::from_secs(2)).unwrap();
        Self {
            stop,
            thread: Some(thread),
        }
    }
}
impl Drop for ClipboardRetentionPeer {
    fn drop(&mut self) {
        let _ = self.stop.send(());
        self.thread.take().unwrap().join().unwrap();
    }
}
struct RetentionTransfer {
    request: SelectionRequestEvent,
    bytes: Vec<u8>,
    incremental: bool,
}
fn run_retention_peer(
    display: String,
    behavior: RetentionBehavior,
    ready: mpsc::SyncSender<()>,
    stopped: mpsc::Receiver<()>,
) {
    let peer = NativeClipboardPeer::connect(&display);
    peer.connection
        .set_selection_owner(
            peer.window,
            peer.atoms.clipboard_manager,
            x11rb::CURRENT_TIME,
        )
        .unwrap()
        .check()
        .unwrap();
    ready.send(()).unwrap();
    let mut transfer = None;
    let mut retained = None;
    while matches!(stopped.try_recv(), Err(mpsc::TryRecvError::Empty)) {
        if let Some(event) = peer.connection.poll_for_event().unwrap() {
            service_retention_event(&peer, behavior, &mut transfer, &mut retained, event);
            continue;
        }
        properties::wait_selection(&peer.connection, Instant::now() + Duration::from_millis(10))
            .unwrap();
    }
}
fn service_retention_event(
    peer: &NativeClipboardPeer,
    behavior: RetentionBehavior,
    transfer: &mut Option<RetentionTransfer>,
    retained: &mut Option<Vec<u8>>,
    event: Event,
) {
    match event {
        Event::SelectionRequest(request) if request.selection == peer.atoms.clipboard_manager => {
            *transfer = start_retention_request(peer, behavior, request);
        }
        Event::SelectionRequest(request) => reply_retained_text(peer, retained.as_deref(), request),
        Event::SelectionNotify(notification) if notification.selection == peer.atoms.clipboard => {
            read_retention_header(peer, transfer, retained);
        }
        Event::PropertyNotify(property)
            if property.state == Property::NEW_VALUE && property.atom == peer.property =>
        {
            read_retention_chunk(peer, transfer, retained);
        }
        _ => {}
    }
}
fn start_retention_request(
    peer: &NativeClipboardPeer,
    behavior: RetentionBehavior,
    request: SelectionRequestEvent,
) -> Option<RetentionTransfer> {
    assert_eq!(request.target, peer.atoms.save_targets);
    let targets = peer
        .connection
        .get_property(
            false,
            request.requestor,
            request.property,
            AtomEnum::ATOM,
            0,
            64,
        )
        .unwrap()
        .reply()
        .unwrap();
    assert_eq!(
        targets.value32().unwrap().collect::<Vec<_>>(),
        [peer.atoms.utf8]
    );
    match behavior {
        RetentionBehavior::Save => {
            peer.request(peer.atoms.utf8);
            Some(RetentionTransfer {
                request,
                bytes: Vec::new(),
                incremental: false,
            })
        }
        RetentionBehavior::Decline => {
            notify_retention_peer(peer, request, 0);
            None
        }
        RetentionBehavior::WrongProperty => {
            notify_retention_peer(peer, request, peer.property);
            None
        }
        RetentionBehavior::Ignore => None,
    }
}
fn read_retention_header(
    peer: &NativeClipboardPeer,
    transfer: &mut Option<RetentionTransfer>,
    retained: &mut Option<Vec<u8>>,
) {
    let current = transfer.as_mut().unwrap();
    let header = peer.property();
    current.incremental = header.type_ == peer.atoms.incr;
    if current.incremental {
        return;
    }
    assert_eq!(header.type_, peer.atoms.utf8);
    current.bytes = header.value;
    complete_retention(peer, transfer, retained);
}
fn read_retention_chunk(
    peer: &NativeClipboardPeer,
    transfer: &mut Option<RetentionTransfer>,
    retained: &mut Option<Vec<u8>>,
) {
    let Some(current) = transfer.as_mut().filter(|current| current.incremental) else {
        return;
    };
    let chunk = peer.property();
    assert_eq!(chunk.type_, peer.atoms.utf8);
    assert_eq!(chunk.format, 8);
    if chunk.value.is_empty() {
        complete_retention(peer, transfer, retained);
        return;
    }
    current.bytes.extend_from_slice(&chunk.value);
}
fn complete_retention(
    peer: &NativeClipboardPeer,
    transfer: &mut Option<RetentionTransfer>,
    retained: &mut Option<Vec<u8>>,
) {
    let completed = transfer.take().unwrap();
    peer.connection
        .set_selection_owner(peer.window, peer.atoms.clipboard, x11rb::CURRENT_TIME)
        .unwrap()
        .check()
        .unwrap();
    *retained = Some(completed.bytes);
    peer.connection
        .change_property32(
            PropMode::REPLACE,
            completed.request.requestor,
            completed.request.property,
            peer.atoms.null,
            &[],
        )
        .unwrap()
        .check()
        .unwrap();
    notify_retention_peer(peer, completed.request, completed.request.property);
}
fn notify_retention_peer(
    peer: &NativeClipboardPeer,
    request: SelectionRequestEvent,
    property: Atom,
) {
    let notification = SelectionNotifyEvent {
        response_type: SELECTION_NOTIFY_EVENT,
        sequence: 0,
        time: request.time,
        requestor: request.requestor,
        selection: request.selection,
        target: request.target,
        property,
    };
    peer.connection
        .send_event(false, request.requestor, EventMask::NO_EVENT, notification)
        .unwrap()
        .check()
        .unwrap();
}
fn reply_retained_text(
    peer: &NativeClipboardPeer,
    retained: Option<&[u8]>,
    request: SelectionRequestEvent,
) {
    let retained = retained.unwrap();
    assert_eq!(request.target, peer.atoms.utf8);
    peer.connection
        .change_property8(
            PropMode::REPLACE,
            request.requestor,
            request.property,
            peer.atoms.utf8,
            retained,
        )
        .unwrap()
        .check()
        .unwrap();
    notify_retention_peer(peer, request, request.property);
}
fn receive_retained_notification(peer: &NativeClipboardPeer) -> SelectionNotifyEvent {
    let deadline = Instant::now() + Duration::from_secs(1);
    loop {
        assert!(
            Instant::now() < deadline,
            "retained clipboard: expected manager response before deadline"
        );
        if let Some(event) = peer.connection.poll_for_event().unwrap() {
            if let Event::SelectionNotify(notification) = event {
                return notification;
            }
            continue;
        }
        properties::wait_selection(&peer.connection, deadline).unwrap();
    }
}
#[test]
fn shutdown_without_owned_text_or_manager_is_immediate_and_keeps_current_text() {
    let fixture = NativeClipboardFixture::new();
    assert_eq!(
        fixture.endpoint.handoff_on_shutdown().unwrap(),
        NativeClipboardHandoff::NotOwned
    );
    fixture.endpoint.publish_text("without manager").unwrap();
    assert_eq!(
        fixture.endpoint.handoff_on_shutdown().unwrap(),
        NativeClipboardHandoff::NoManager
    );
    assert_eq!(
        crate::ViewportClipboard::read_text(&fixture.endpoint)
            .unwrap()
            .as_deref(),
        Some("without manager")
    );
}
#[test]
fn shutdown_manager_retains_complete_unicode_after_the_source_endpoint_is_dropped() {
    for text in [String::new(), "é🙂".into(), "é🙂".repeat(200)] {
        let fixture = NativeClipboardFixture::new();
        fixture.endpoint.publish_text(&text).unwrap();
        let _retention =
            ClipboardRetentionPeer::open(fixture._server.display().into(), RetentionBehavior::Save);
        assert_eq!(
            fixture.endpoint.handoff_on_shutdown().unwrap(),
            NativeClipboardHandoff::Saved
        );
        drop(fixture.endpoint);
        fixture.peer.request(fixture.peer.atoms.utf8);
        assert_ne!(receive_retained_notification(&fixture.peer).property, 0);
        assert_eq!(fixture.peer.property().value, text.as_bytes());
    }
}
#[test]
fn a_refused_shutdown_transfer_preserves_text_and_reports_the_named_outcome() {
    let fixture = NativeClipboardFixture::new();
    let _retention =
        ClipboardRetentionPeer::open(fixture._server.display().into(), RetentionBehavior::Decline);
    fixture
        .endpoint
        .publish_text("retained on decline")
        .unwrap();
    assert_eq!(
        fixture.endpoint.handoff_on_shutdown().unwrap(),
        NativeClipboardHandoff::Declined
    );
    assert_eq!(
        crate::ViewportClipboard::read_text(&fixture.endpoint)
            .unwrap()
            .as_deref(),
        Some("retained on decline")
    );
}
#[test]
fn a_wrong_shutdown_property_is_a_named_failure_and_keeps_the_endpoint_usable() {
    let fixture = NativeClipboardFixture::new();
    let _retention = ClipboardRetentionPeer::open(
        fixture._server.display().into(),
        RetentionBehavior::WrongProperty,
    );
    fixture.endpoint.publish_text("owned").unwrap();
    assert!(
        fixture
            .endpoint
            .handoff_on_shutdown()
            .unwrap_err()
            .contains("handoff property")
    );
    fixture.endpoint.publish_text("still usable").unwrap();
}
#[test]
fn a_stalled_shutdown_manager_cannot_hold_the_owner_beyond_its_deadline() {
    let fixture = NativeClipboardFixture::new();
    let _retention =
        ClipboardRetentionPeer::open(fixture._server.display().into(), RetentionBehavior::Ignore);
    let options = NativeClipboardOptions {
        transfer_timeout: Duration::from_millis(40),
        ..fixture.endpoint.0.options
    };
    let endpoint =
        NativeClipboardEndpoint::connect(Some(fixture._server.display()), options).unwrap();
    endpoint.publish_text("bounded shutdown").unwrap();
    assert!(
        endpoint
            .handoff_on_shutdown()
            .unwrap_err()
            .contains("deadline")
    );
    endpoint.publish_text("still usable").unwrap();
}
#[test]
fn shutdown_marker_and_target_sizes_are_served_without_changing_ownership_or_text() {
    let fixture = NativeClipboardFixture::new();
    fixture.endpoint.publish_text("é🙂").unwrap();
    fixture.transfer(fixture.peer.atoms.save_targets);
    assert_eq!(fixture.peer.property().type_, fixture.peer.atoms.null);
    fixture.transfer(fixture.peer.atoms.target_sizes);
    let sizes = fixture
        .peer
        .property()
        .value32()
        .unwrap()
        .collect::<Vec<_>>();
    let estimates: std::collections::BTreeMap<_, _> = sizes
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| (pair[0], pair[1]))
        .collect();
    assert_eq!(estimates[&fixture.peer.atoms.save_targets], u32::MAX);
    assert_eq!(estimates[&fixture.peer.atoms.utf8], 6);
    assert_eq!(
        crate::ViewportClipboard::read_text(&fixture.endpoint)
            .unwrap()
            .as_deref(),
        Some("é🙂")
    );
}
