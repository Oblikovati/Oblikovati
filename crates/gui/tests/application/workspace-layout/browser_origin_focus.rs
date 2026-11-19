// SPDX-License-Identifier: GPL-2.0-only
//! Origin row focus is separate from viewport selection (#4140).
use super::*;

#[test]
fn choosing_an_origin_row_moves_its_retained_focus() {
    let state = GuiCommandState::default();
    let mut view = BrowserView::default();
    let mut actions = Vec::new();
    choose_origin_row(&state, &mut view, 4, &mut actions);
    assert_eq!(view.selected_origin_row, Some(4));
    assert_eq!(view.focused_origin_row, Some(4));
    choose_origin_row(&state, &mut view, 6, &mut actions);
    assert_eq!(view.selected_origin_row, Some(6));
    assert_eq!(view.focused_origin_row, Some(6));
}
