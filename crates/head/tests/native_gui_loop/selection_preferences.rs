// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::{GuiSelectionKind, GuiSelectionMode};

fn preference_window(directory: &std::path::Path) -> (ScriptedGuiWindow, NativeDesktopOwner) {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory).unwrap();
    owner.create_part_document().unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    (window, owner)
}
fn finish_window(mut window: ScriptedGuiWindow, mut owner: NativeDesktopOwner) {
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn native_selection_preferences_survive_restart_and_reset_from_the_panel() {
    let directory = tempfile::tempdir().unwrap();
    let (window, mut owner) = preference_window(directory.path());
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    let face = filter
        .entries
        .iter()
        .position(|entry| entry.kind == GuiSelectionKind::Face)
        .unwrap();
    owner.move_selection_kind_from_gui(face, 0).unwrap();
    owner
        .set_selection_enabled_from_gui(GuiSelectionKind::Body, false)
        .unwrap();
    finish_window(window, owner);

    let (mut window, mut owner) = preference_window(directory.path());
    let state = owner.gui_command_state().unwrap();
    let filter = state.selection_filter.unwrap();
    assert_eq!(filter.entries[0].kind, GuiSelectionKind::Face);
    assert!(
        !filter
            .entries
            .iter()
            .find(|row| row.kind == GuiSelectionKind::Body)
            .unwrap()
            .enabled
    );
    assert_eq!(state.selection_mode, GuiSelectionMode::Custom);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Select"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_start("Selection Filter"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Reset"),
    );
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    assert_eq!(filter.entries[0].kind, GuiSelectionKind::WorkPoint);
    assert!(filter.entries.iter().all(|row| row.enabled));
    finish_window(window, owner);

    let (window, mut owner) = preference_window(directory.path());
    let state = owner.gui_command_state().unwrap();
    let filter = state.selection_filter.unwrap();
    assert_eq!(filter.entries[0].kind, GuiSelectionKind::WorkPoint);
    assert!(filter.entries.iter().all(|row| row.enabled));
    assert_eq!(state.selection_mode, GuiSelectionMode::All);
    finish_window(window, owner);
}

#[test]
fn panel_priority_and_checkbox_changes_survive_a_native_restart() {
    let directory = tempfile::tempdir().unwrap();
    let (mut window, mut owner) = preference_window(directory.path());
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Select"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_start("Selection Filter"),
    );
    let label = application_bdd::rendered_label_start("Work points");
    let (position, size) = application_bdd::window_geometry("Selection Filter").unwrap();
    let down = application_bdd::rendered_label_in_rect(
        "Down",
        [position[0], label[1] as f32 - 8.],
        [position[0] + size[0], label[1] as f32 + 8.],
    );
    click_shell_position(&mut window, &mut owner, down);
    let label = application_bdd::rendered_label_start("Work points");
    click_shell_position(&mut window, &mut owner, [label[0] - 15, label[1]]);
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    assert_eq!(filter.entries[1].kind, GuiSelectionKind::WorkPoint);
    assert!(!filter.entries[1].enabled);
    finish_window(window, owner);

    let (window, mut owner) = preference_window(directory.path());
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    assert_eq!(filter.entries[1].kind, GuiSelectionKind::WorkPoint);
    assert!(!filter.entries[1].enabled);
    finish_window(window, owner);
}
