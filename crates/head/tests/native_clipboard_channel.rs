// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn unavailable_clipboard_remains_a_local_named_decline() {
    let channel = NativeClipboardChannel::with_endpoint(Err(
        "clipboard fixture: expected a live connection".into(),
    ));
    assert!(channel.endpoint().is_none());
    assert_eq!(
        channel.issue().as_deref(),
        Some("clipboard fixture: expected a live connection")
    );
    assert!(channel.read_text().unwrap_err().contains("live connection"));
    assert!(
        channel
            .write_text("text")
            .unwrap_err()
            .contains("live connection")
    );
    channel.service_pending(32);
    assert!(channel.issue().is_some());
    channel.handoff_on_shutdown().unwrap();
}
