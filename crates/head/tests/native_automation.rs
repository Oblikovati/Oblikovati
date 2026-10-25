// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWindow;
use oblikovati_api::wire::{
    CaptureWindowArgs, ClickViewportArgs, METHOD_VIEWPORT_CAPTURE_WINDOW, METHOD_VIEWPORT_CLICK,
    METHOD_VIEWPORT_KEY, METHOD_VIEWPORT_SCROLL, PressKeyArgs, ScrollViewportArgs,
};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey};

#[test]
fn owner_thread_automation_captures_and_queues_viewport_input() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.present().unwrap();
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("window.png");
    let capture = window.capture_window_png(&path).unwrap();
    assert!(!capture.rgba.is_empty());
    assert!(std::fs::metadata(path).unwrap().len() > 8);
    window
        .inject_viewport_input(ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Home,
            pressed: true,
        }))
        .unwrap();
    assert!(matches!(
        window.events(1).unwrap().as_slice(),
        [ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Home,
            pressed: true
        })]
    ));
    window.finish().unwrap();
}

#[test]
fn native_zero_document_uses_the_go_dark_viewport_background() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.present().unwrap();
    let capture = window.capture_window().unwrap();
    assert!(capture.rgba.as_chunks::<4>().0.contains(&[30, 45, 80, 255]));
    window.finish().unwrap();
}

#[test]
fn public_automation_requests_capture_and_queue_each_input_kind() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.present().unwrap();
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("captured.png");
    call(
        &mut window,
        METHOD_VIEWPORT_CAPTURE_WINDOW,
        &CaptureWindowArgs {
            path: path.to_string_lossy().into_owned(),
        },
    );
    assert!(!path.exists());
    window.present().unwrap();
    call(
        &mut window,
        METHOD_VIEWPORT_CLICK,
        &ClickViewportArgs {
            x: 20.,
            y: 30.,
            ..Default::default()
        },
    );
    call(
        &mut window,
        METHOD_VIEWPORT_KEY,
        &PressKeyArgs {
            key: "Enter".into(),
            ..Default::default()
        },
    );
    call(
        &mut window,
        METHOD_VIEWPORT_SCROLL,
        &ScrollViewportArgs {
            dy: 2.,
            x: 20.,
            y: 30.,
            ..Default::default()
        },
    );
    assert!(std::fs::metadata(path).unwrap().len() > 8);
    assert_eq!(window.events(5).unwrap().len(), 5);
    window.finish().unwrap();
}

#[test]
fn public_automation_keys_follow_native_physical_and_text_input() {
    let mut window = NativeHeadWindow::open().unwrap();
    call(
        &mut window,
        METHOD_VIEWPORT_KEY,
        &PressKeyArgs {
            key: "a".into(),
            ..Default::default()
        },
    );
    assert!(matches!(
        window.events(3).unwrap().as_slice(),
        [
            ViewportEvent::Input(ViewportInput::PhysicalKey { pressed: true, .. }),
            ViewportEvent::Input(ViewportInput::PhysicalKey { pressed: false, .. }),
            ViewportEvent::Input(ViewportInput::Text('a')),
        ]
    ));
    window.finish().unwrap();
}

#[test]
fn public_automation_delivers_printable_text_without_a_named_key() {
    let mut window = NativeHeadWindow::open().unwrap();
    call(
        &mut window,
        METHOD_VIEWPORT_KEY,
        &PressKeyArgs {
            key: "5".into(),
            ..Default::default()
        },
    );
    assert_eq!(
        window.events(1).unwrap(),
        [ViewportEvent::Input(ViewportInput::Text('5'))]
    );
    window.finish().unwrap();
}

fn call<T: serde::Serialize>(window: &mut NativeHeadWindow, method: &str, args: &T) {
    let bytes = serde_json::to_vec(args).unwrap();
    crate::viewport_automation::apply(window, method, &bytes)
        .unwrap()
        .unwrap();
}
