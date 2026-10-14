// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiClipboard;
use oblikovati_viewport::{
    NativeClipboardEndpoint, NativeClipboardOptions, ViewportClipboard, ViewportWake,
};
use std::{
    rc::Rc,
    sync::mpsc,
    thread,
    time::{Duration, Instant},
};
#[path = "../../../../test-utilities/private_x11.rs"]
mod private_server;
enum WidgetPeerCommand {
    Read(mpsc::SyncSender<Result<Option<String>, String>>),
    Stop,
}
struct NativeWidgetClipboardPeer {
    commands: mpsc::Sender<WidgetPeerCommand>,
    wake: ViewportWake,
    thread: Option<thread::JoinHandle<()>>,
}
impl NativeWidgetClipboardPeer {
    fn open(display: String, text: String) -> Self {
        let (commands, received) = mpsc::channel();
        let (ready, startup) = mpsc::sync_channel(1);
        let wake = ViewportWake::new().unwrap();
        let owner_wake = wake.clone();
        let thread =
            thread::spawn(move || run_widget_peer(display, text, received, owner_wake, ready));
        startup.recv_timeout(Duration::from_secs(2)).unwrap();
        Self {
            commands,
            wake,
            thread: Some(thread),
        }
    }
    fn read_from_widget(&self, owner: &NativeClipboardEndpoint) -> Option<String> {
        let (reply, received) = mpsc::sync_channel(1);
        self.commands.send(WidgetPeerCommand::Read(reply)).unwrap();
        self.wake.wake().unwrap();
        let deadline = Instant::now() + Duration::from_secs(2);
        let wake = ViewportWake::new().unwrap();
        loop {
            assert!(
                Instant::now() < deadline,
                "widget clipboard: expected complete peer read before deadline"
            );
            owner.service_pending(32).unwrap();
            if let Ok(reply) = received.try_recv() {
                return reply.unwrap();
            }
            owner
                .wait_events(&wake, Some(Duration::from_millis(10)))
                .unwrap();
        }
    }
}
impl Drop for NativeWidgetClipboardPeer {
    fn drop(&mut self) {
        self.commands.send(WidgetPeerCommand::Stop).unwrap();
        self.wake.wake().unwrap();
        self.thread.take().unwrap().join().unwrap();
    }
}
fn widget_clipboard_options() -> NativeClipboardOptions {
    NativeClipboardOptions {
        byte_limit: 4096,
        target_limit: 64,
        chunk_limit: 64,
        transfer_limit: 16,
        transfer_timeout: Duration::from_secs(1),
    }
}
fn run_widget_peer(
    display: String,
    text: String,
    commands: mpsc::Receiver<WidgetPeerCommand>,
    wake: ViewportWake,
    ready: mpsc::SyncSender<()>,
) {
    let endpoint =
        NativeClipboardEndpoint::connect(Some(&display), widget_clipboard_options()).unwrap();
    endpoint.publish_text(&text).unwrap();
    ready.send(()).unwrap();
    loop {
        match commands.try_recv() {
            Ok(WidgetPeerCommand::Read(reply)) => {
                reply.send(endpoint.read_text()).unwrap();
                continue;
            }
            Ok(WidgetPeerCommand::Stop) | Err(mpsc::TryRecvError::Disconnected) => return,
            Err(mpsc::TryRecvError::Empty) => {}
        }
        endpoint.service_pending(32).unwrap();
        endpoint.wait_events(&wake, None).unwrap();
    }
}
fn widget_shortcut(context: &mut Context, text: &mut String, key: GuiKey) {
    hold_widget_shortcut(context, text, key);
    let released = [
        GuiInputEvent::Key {
            key,
            pressed: false,
        },
        GuiInputEvent::Modifiers(Default::default()),
    ];
    widget_frame(context, &released, text, false);
    widget_frame(context, &[], text, false);
}
fn hold_widget_shortcut(context: &mut Context, text: &mut String, key: GuiKey) {
    // ImGui 1.89.2 routes shortcuts for the next frame; native Control presses precede letter presses.
    let modifiers = GuiModifiers {
        control: true,
        ..Default::default()
    };
    widget_frame(context, &[GuiInputEvent::Modifiers(modifiers)], text, false);
    widget_frame(
        context,
        &[GuiInputEvent::Key { key, pressed: true }],
        text,
        false,
    );
}
#[test]
fn real_text_widget_pastes_copies_and_cuts_native_unicode_without_touching_the_desktop_clipboard() {
    let server = private_server::PrivateX11Server::open_compose_fixture().unwrap();
    let peer = NativeWidgetClipboardPeer::open(server.display().into(), "é🙂".into());
    let endpoint = Rc::new(
        NativeClipboardEndpoint::connect(Some(server.display()), widget_clipboard_options())
            .unwrap(),
    );
    let clipboard = GuiClipboard::new(endpoint.clone(), 4096).unwrap();
    let mut context = input_probe_context();
    context.set_clipboard_backend(clipboard.clone());
    let mut text = String::new();
    focus_text_probe(&mut context, &mut text);
    widget_shortcut(&mut context, &mut text, GuiKey::V);
    assert_eq!(text, "é🙂");
    verify_widget_copy_cut_and_paste(&mut context, &mut text, &peer, &endpoint);
    assert_eq!(clipboard.issue(), None);
}
fn verify_widget_copy_cut_and_paste(
    context: &mut Context,
    text: &mut String,
    peer: &NativeWidgetClipboardPeer,
    endpoint: &NativeClipboardEndpoint,
) {
    widget_shortcut(context, text, GuiKey::A);
    widget_shortcut(context, text, GuiKey::C);
    assert_eq!(peer.read_from_widget(endpoint).as_deref(), Some("é🙂"));
    widget_shortcut(context, text, GuiKey::X);
    assert_eq!(text, "");
    assert_eq!(peer.read_from_widget(endpoint).as_deref(), Some("é🙂"));
    widget_shortcut(context, text, GuiKey::V);
    assert_eq!(text, "é🙂");
}
