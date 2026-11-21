// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn a_request_proposes_the_document_and_asks_for_a_frame() {
    let mut dialog = ComponentDialog {
        name: "left over".into(),
        ..Default::default()
    };
    assert!(!dialog.needs_frame());
    dialog.request("Part1");
    assert!(dialog.needs_frame());
    assert_eq!(dialog.name, "Part1");
}
