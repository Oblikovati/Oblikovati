// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_dock_tab_windows_open_and_close_independently() {
    let mut panels = DockPanels::default();
    panels.open_panel(DockPanel::Log);
    panels.open_panel(DockPanel::Favorites);
    assert!(panels.model_open && panels.log_open && panels.favorites_open);
    panels.set_open(DockPanel::Log, false);
    assert!(panels.model_open && !panels.log_open && panels.favorites_open);
}

#[test]
fn closing_the_last_native_tab_empties_the_dock() {
    let mut panels = DockPanels::default();
    panels.set_open(DockPanel::Model, false);
    assert!(!panels.model_open && !panels.log_open && !panels.favorites_open);
    panels.open_panel(DockPanel::Model);
    assert!(panels.model_open);
}

#[test]
fn the_log_keeps_each_new_message_once_and_never_the_idle_status() {
    let mut panels = DockPanels::default();
    for status in [
        Some("Ready"),
        Some("Saved"),
        Some("Saved"),
        None,
        Some(""),
        Some("Closed"),
    ] {
        panels.record_status(status);
    }
    assert_eq!(panels.messages, ["Saved", "Closed"]);
}

#[test]
fn the_log_stays_within_its_capacity() {
    let mut panels = DockPanels::default();
    for index in 0..=MESSAGE_CAPACITY {
        panels.record_status(Some(&index.to_string()));
    }
    assert_eq!(panels.messages.len(), MESSAGE_CAPACITY);
    assert_eq!(panels.messages[0], "1");
}

#[test]
fn saved_open_panels_restore_without_overriding_imgui_active_tab() {
    let mut original = DockPanels::default();
    original.open_panel(DockPanel::Log);
    original.model_open = false;
    let saved = original.settings();
    let mut restored = DockPanels::default();
    restored.restore(&saved);
    assert!(!restored.model_open && restored.log_open);
    assert!(restored.log_open);
}

#[test]
fn favorites_visibility_follows_menu_toggles_and_keeps_the_model() {
    let mut panels = DockPanels::default();
    assert!(!panels.favorites_visible());
    panels.toggle_favorites();
    assert!(panels.favorites_visible());
    panels.toggle_favorites();
    assert!(!panels.favorites_visible());
    assert!(panels.model_open && !panels.favorites_open);
}
