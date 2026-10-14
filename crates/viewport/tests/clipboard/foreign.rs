// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportClipboard;
use std::{sync::mpsc, thread};
struct IndependentClipboardOwner {
    stop: mpsc::Sender<()>,
    thread: Option<thread::JoinHandle<()>>,
}
impl IndependentClipboardOwner {
    fn open(display: String, text: String) -> Self {
        let (ready, received) = mpsc::sync_channel(1);
        let (stop, stopped) = mpsc::channel();
        let thread = thread::spawn(move || run_independent_owner(display, text, ready, stopped));
        received.recv_timeout(Duration::from_secs(2)).unwrap();
        Self {
            stop,
            thread: Some(thread),
        }
    }
}
impl Drop for IndependentClipboardOwner {
    fn drop(&mut self) {
        let _ = self.stop.send(());
        self.thread.take().unwrap().join().unwrap();
    }
}
fn run_independent_owner(
    display: String,
    text: String,
    ready: mpsc::SyncSender<()>,
    stopped: mpsc::Receiver<()>,
) {
    let options = NativeClipboardOptions {
        byte_limit: 4096,
        target_limit: 64,
        chunk_limit: 64,
        transfer_limit: 16,
        transfer_timeout: Duration::from_secs(1),
    };
    let owner = NativeClipboardEndpoint::connect(Some(&display), options).unwrap();
    owner.publish_text(&text).unwrap();
    ready.send(()).unwrap();
    while stopped.try_recv().is_err() {
        owner.service_pending(32).unwrap();
        let deadline = Instant::now() + Duration::from_millis(10);
        properties::wait_selection(&owner.0.connection, deadline).unwrap();
    }
}
#[test]
fn a_foreign_owner_is_negotiated_and_read_as_complete_unicode_or_empty_text() {
    let fixture = NativeClipboardFixture::new();
    for text in ["é界🙂", ""] {
        let _owner = IndependentClipboardOwner::open(fixture._server.display().into(), text.into());
        assert_eq!(fixture.endpoint.read_text().unwrap().as_deref(), Some(text));
    }
}
#[test]
fn no_owner_and_same_owner_reads_do_not_wait_for_a_gui_callback() {
    let fixture = NativeClipboardFixture::new();
    assert_eq!(fixture.endpoint.read_text().unwrap(), None);
    fixture.endpoint.write_text("owned").unwrap();
    assert_eq!(
        fixture.endpoint.read_text().unwrap().as_deref(),
        Some("owned")
    );
}
#[test]
fn a_nonresponsive_owner_has_one_transfer_deadline_and_keeps_the_endpoint_usable() {
    let fixture = NativeClipboardFixture::new();
    let options = NativeClipboardOptions {
        byte_limit: 4096,
        target_limit: 64,
        chunk_limit: 64,
        transfer_limit: 16,
        transfer_timeout: Duration::from_millis(40),
    };
    let reader =
        NativeClipboardEndpoint::connect(Some(fixture._server.display()), options).unwrap();
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
    let elapsed = Instant::now();
    assert!(reader.read_text().unwrap_err().contains("deadline"));
    assert!(elapsed.elapsed() < Duration::from_secs(1));
    reader.publish_text("after timeout").unwrap();
    assert_eq!(
        reader.read_text().unwrap().as_deref(),
        Some("after timeout")
    );
}

#[test]
fn target_metadata_does_not_consume_the_text_byte_budget() {
    let fixture = NativeClipboardFixture::new();
    let _owner = IndependentClipboardOwner::open(fixture._server.display().into(), "🙂".into());
    let options = NativeClipboardOptions {
        byte_limit: 4,
        target_limit: 8,
        chunk_limit: 64,
        transfer_limit: 16,
        transfer_timeout: Duration::from_secs(1),
    };
    let reader =
        NativeClipboardEndpoint::connect(Some(fixture._server.display()), options).unwrap();
    assert_eq!(reader.read_text().unwrap().as_deref(), Some("🙂"));
}

#[path = "scripted_owner.rs"]
mod scripted_tests;
