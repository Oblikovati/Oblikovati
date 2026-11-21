// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::VecDeque;
#[derive(Clone)]
enum SelectionScript {
    DirectLatin1(Vec<u8>),
    IncrementalUtf8(Vec<Vec<u8>>),
}
struct ScriptedSelectionOwner {
    stop: mpsc::Sender<()>,
    thread: Option<thread::JoinHandle<()>>,
}
impl ScriptedSelectionOwner {
    fn open(display: String, script: SelectionScript) -> Self {
        let (ready, received) = mpsc::sync_channel(1);
        let (stop, stopped) = mpsc::channel();
        let thread = thread::spawn(move || run_scripted_owner(display, script, ready, stopped));
        received.recv_timeout(Duration::from_secs(2)).unwrap();
        Self {
            stop,
            thread: Some(thread),
        }
    }
}
impl Drop for ScriptedSelectionOwner {
    fn drop(&mut self) {
        let _ = self.stop.send(());
        self.thread.take().unwrap().join().unwrap();
    }
}
struct ScriptedTransfer {
    request: SelectionRequestEvent,
    chunks: VecDeque<Vec<u8>>,
}
fn run_scripted_owner(
    display: String,
    script: SelectionScript,
    ready: mpsc::SyncSender<()>,
    stopped: mpsc::Receiver<()>,
) {
    let peer = NativeClipboardPeer::connect(&display);
    peer.connection
        .set_selection_owner(peer.window, peer.atoms.clipboard, x11rb::CURRENT_TIME)
        .unwrap()
        .check()
        .unwrap();
    ready.send(()).unwrap();
    let mut transfer = None;
    while matches!(stopped.try_recv(), Err(mpsc::TryRecvError::Empty)) {
        if let Some(event) = peer.connection.poll_for_event().unwrap() {
            service_script_event(&peer, &script, &mut transfer, event);
            continue;
        }
        properties::wait_selection(&peer.connection, Instant::now() + Duration::from_millis(10))
            .unwrap();
    }
}
fn service_script_event(
    peer: &NativeClipboardPeer,
    script: &SelectionScript,
    transfer: &mut Option<ScriptedTransfer>,
    event: Event,
) {
    match event {
        Event::SelectionRequest(request) => *transfer = reply_script_request(peer, script, request),
        Event::PropertyNotify(property) if property.state == Property::DELETE => {
            send_script_chunk(peer, transfer, property)
        }
        _ => {}
    }
}
fn reply_script_request(
    peer: &NativeClipboardPeer,
    script: &SelectionScript,
    request: SelectionRequestEvent,
) -> Option<ScriptedTransfer> {
    let target = match script {
        SelectionScript::DirectLatin1(_) => u32::from(AtomEnum::STRING),
        _ => peer.atoms.utf8,
    };
    if request.target == peer.atoms.targets {
        peer.connection
            .change_property32(
                PropMode::REPLACE,
                request.requestor,
                request.property,
                AtomEnum::ATOM,
                &[target],
            )
            .unwrap()
            .check()
            .unwrap();
        notify_script_peer(peer, request);
        return None;
    }
    assert_eq!(request.target, target);
    let transfer = publish_script_property(peer, script, request);
    notify_script_peer(peer, request);
    transfer
}
fn publish_script_property(
    peer: &NativeClipboardPeer,
    script: &SelectionScript,
    request: SelectionRequestEvent,
) -> Option<ScriptedTransfer> {
    match script {
        SelectionScript::DirectLatin1(bytes) => {
            peer.connection
                .change_property8(
                    PropMode::REPLACE,
                    request.requestor,
                    request.property,
                    AtomEnum::STRING,
                    bytes,
                )
                .unwrap()
                .check()
                .unwrap();
            None
        }
        SelectionScript::IncrementalUtf8(chunks) => begin_script_incremental(peer, request, chunks),
    }
}
fn begin_script_incremental(
    peer: &NativeClipboardPeer,
    request: SelectionRequestEvent,
    chunks: &[Vec<u8>],
) -> Option<ScriptedTransfer> {
    peer.connection
        .change_window_attributes(
            request.requestor,
            &ChangeWindowAttributesAux::new().event_mask(EventMask::PROPERTY_CHANGE),
        )
        .unwrap()
        .check()
        .unwrap();
    let total = chunks.iter().map(Vec::len).sum::<usize>() as u32;
    peer.connection
        .change_property32(
            PropMode::REPLACE,
            request.requestor,
            request.property,
            peer.atoms.incr,
            &[total],
        )
        .unwrap()
        .check()
        .unwrap();
    let mut chunks: VecDeque<_> = chunks.iter().cloned().collect();
    chunks.push_back(Vec::new());
    Some(ScriptedTransfer { request, chunks })
}
fn notify_script_peer(peer: &NativeClipboardPeer, request: SelectionRequestEvent) {
    let notification = SelectionNotifyEvent {
        response_type: SELECTION_NOTIFY_EVENT,
        sequence: 0,
        time: request.time,
        requestor: request.requestor,
        selection: request.selection,
        target: request.target,
        property: request.property,
    };
    peer.connection
        .send_event(false, request.requestor, EventMask::NO_EVENT, notification)
        .unwrap()
        .check()
        .unwrap();
}
fn send_script_chunk(
    peer: &NativeClipboardPeer,
    transfer: &mut Option<ScriptedTransfer>,
    property: PropertyNotifyEvent,
) {
    let Some(active) = transfer else {
        return;
    };
    if property.window != active.request.requestor || property.atom != active.request.property {
        return;
    }
    let bytes = active.chunks.pop_front().unwrap();
    peer.connection
        .change_property8(
            PropMode::REPLACE,
            property.window,
            property.atom,
            peer.atoms.utf8,
            &bytes,
        )
        .unwrap()
        .check()
        .unwrap();
    if bytes.is_empty() {
        transfer.take();
    }
}
#[test]
fn incremental_native_read_preserves_unicode_split_across_properties() {
    let fixture = NativeClipboardFixture::new();
    let script = SelectionScript::IncrementalUtf8(vec![vec![0xf0], vec![0x9f, 0x99], vec![0x82]]);
    let _owner = ScriptedSelectionOwner::open(fixture._server.display().into(), script);
    assert_eq!(fixture.endpoint.read_text().unwrap().as_deref(), Some("🙂"));
}
#[test]
fn latin1_native_target_is_decoded_without_utf8_guessing() {
    let fixture = NativeClipboardFixture::new();
    let _owner = ScriptedSelectionOwner::open(
        fixture._server.display().into(),
        SelectionScript::DirectLatin1(vec![0xe9]),
    );
    assert_eq!(fixture.endpoint.read_text().unwrap().as_deref(), Some("é"));
}
