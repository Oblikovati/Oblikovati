// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn a_request_starts_empty_and_asks_for_a_frame() {
    let mut dialog = ComponentDialog {
        name: "left over".into(),
        browsing: true,
        ..Default::default()
    };
    assert!(!dialog.needs_frame());
    dialog.request();
    assert!(dialog.needs_frame() && !dialog.browsing && dialog.name.is_empty());
}
