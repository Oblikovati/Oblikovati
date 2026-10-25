// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWindow;
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
