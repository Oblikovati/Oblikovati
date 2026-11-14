// SPDX-License-Identifier: GPL-2.0-only
//! Cancelled preview settings must have changed before they are discarded.
use super::*;

#[test]
fn cancel_discards_actual_checkbox_and_radio_changes() {
    let mut world = ApplicationWorld::new();
    dialog_open(&mut world);
    preview_selected(&mut world);
    clear_preview(&mut world);
    assert!(!world.checked("Save Preview Picture"));
    world.expect_no_text(ICONIC);
    cancel_dialog(&mut world);
    world.press_save_shortcut();
    preview_selected(&mut world);
    assert!(world.radio_selected(ICONIC));
    choose_window(&mut world);
    assert!(world.radio_selected(WINDOW));
    assert!(!world.radio_selected(ICONIC));
    cancel_dialog(&mut world);
    world.press_save_shortcut();
    preview_selected(&mut world);
    assert!(world.radio_selected(ICONIC));
    assert!(!world.radio_selected(WINDOW));
    world.close();
}
