// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn document(id: u64, name: &str) -> DocumentInfo {
    DocumentInfo {
        id,
        name: name.into(),
        visible: true,
        ..Default::default()
    }
}
fn state(windows: Vec<GuiDocumentWindow>, active: Option<u64>) -> GuiCommandState {
    GuiCommandState {
        documents: vec![document(1, "Part1"), document(2, "Part2")],
        document_windows: windows,
        document: Some(1),
        active_window: active,
        ..Default::default()
    }
}
const FIRST: GuiDocumentWindow = GuiDocumentWindow {
    id: 1,
    document: 1,
    number: 1,
};
const SECOND: GuiDocumentWindow = GuiDocumentWindow {
    id: 90,
    document: 1,
    number: 2,
};
#[test]
fn a_document_with_one_window_has_no_number_and_activates_as_a_document() {
    let state = state(Vec::new(), None);
    let rows = tab_rows(&state);
    assert_eq!(
        rows.iter().map(|tab| tab.name.as_str()).collect::<Vec<_>>(),
        ["Part1", "Part2"]
    );
    assert_eq!(activation(&state, &rows[0]), None);
    assert_eq!(
        activation(&state, &rows[1]),
        Some(GuiShellAction::ActivateDocument(ActivateDocumentArgs {
            id: 2
        }))
    );
}
#[test]
fn two_windows_read_as_numbered_tabs_and_only_the_other_one_activates() {
    let state = state(vec![FIRST, SECOND], Some(90));
    let rows = tab_rows(&state);
    assert_eq!(rows[0].name, "Part1:1");
    assert_eq!(rows[1].name, "Part1:2");
    assert_eq!(
        activation(&state, &rows[0]),
        Some(GuiShellAction::ActivateWindow(1))
    );
    assert_eq!(activation(&state, &rows[1]), None);
}
