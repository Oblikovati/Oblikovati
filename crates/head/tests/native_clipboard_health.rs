// SPDX-License-Identifier: GPL-2.0-only
use crate::native::clipboard::NativeClipboardChannel;
#[test]
fn clipboard_connection_decline_reaches_native_gui_health_without_losing_frames() {
    use crate::{HeadWindow, NativeHeadWindow};
    let mut window = NativeHeadWindow::open().unwrap();
    window.clipboard = NativeClipboardChannel::with_endpoint(Err(
        "clipboard fixture: expected a live connection".into(),
    ));
    let camera = window.view.camera_state().unwrap();
    let update = window.gui_frame(None, &Default::default()).unwrap();
    assert!(update.changed);
    assert!(
        window
            .gui
            .as_ref()
            .unwrap()
            .notice
            .as_ref()
            .unwrap()
            .contains("Clipboard unavailable")
    );
    window.present().unwrap();
    assert!(!window.view.capture().unwrap().rgba.is_empty());
    assert_eq!(window.view.camera_state().unwrap(), camera);
    window.finish().unwrap();
}
