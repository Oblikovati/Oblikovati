// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn state(active: u64, ids: &[(u64, bool)]) -> GuiCommandState {
    GuiCommandState {
        document: Some(active),
        documents: ids
            .iter()
            .map(|(id, visible)| DocumentInfo {
                id: *id,
                visible: *visible,
                ..Default::default()
            })
            .collect(),
        ..Default::default()
    }
}
#[test]
fn only_visible_documents_are_offered() {
    let state = state(1, &[(1, true), (2, false)]);
    let ids: Vec<u64> = visible(&state).map(|row| row.id).collect();
    assert_eq!(ids, [1]);
}
#[test]
fn the_keyboard_walks_the_tabs_with_home_first_and_wraps() {
    let rows = [(1, true), (2, false), (3, true)];
    let next = DocumentStep::Next;
    let previous = DocumentStep::Previous;
    assert_eq!(
        neighbor_tab(&state(1, &rows), next, false),
        Some(activate(3))
    );
    assert_eq!(
        neighbor_tab(&state(3, &rows), next, false),
        Some(GuiShellAction::ShowHome)
    );
    assert_eq!(
        neighbor_tab(&state(1, &rows), previous, false),
        Some(GuiShellAction::ShowHome)
    );
    assert_eq!(
        neighbor_tab(&state(1, &rows), next, true),
        Some(activate(1))
    );
    assert_eq!(
        neighbor_tab(&state(1, &rows), previous, true),
        Some(activate(3))
    );
}
#[test]
fn with_no_open_document_home_has_nowhere_to_switch() {
    assert_eq!(neighbor_tab(&state(1, &[]), DocumentStep::Next, true), None);
}
#[test]
fn close_tabs_keeps_its_own_tab_and_close_all_keeps_none() {
    let state = state(1, &[(1, true), (2, true), (3, true)]);
    assert_eq!(tab_closes(&state, Some(2)), [close(1), close(3)]);
    assert_eq!(tab_closes(&state, None), [close(1), close(2), close(3)]);
}
