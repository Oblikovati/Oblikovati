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
fn the_keyboard_moves_to_the_next_tab_and_wraps() {
    let rows = [(1, true), (2, false), (3, true)];
    assert_eq!(next_document(&state(1, &rows)), Some(activate(3)));
    assert_eq!(next_document(&state(3, &rows)), Some(activate(1)));
}
#[test]
fn one_open_document_has_nowhere_to_switch() {
    assert_eq!(next_document(&state(1, &[(1, true)])), None);
}
#[test]
fn close_tabs_keeps_its_own_tab_and_close_all_keeps_none() {
    let state = state(1, &[(1, true), (2, true), (3, true)]);
    assert_eq!(tab_closes(&state, Some(2)), [close(1), close(3)]);
    assert_eq!(tab_closes(&state, None), [close(1), close(2), close(3)]);
}
