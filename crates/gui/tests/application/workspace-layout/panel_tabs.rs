// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn a_new_panel_takes_the_selection_and_the_dock_stays_open() {
    let mut panels = DockPanels::default();
    panels.open_panel(DockPanel::Log);
    assert_eq!(panels.select, Some(DockPanel::Log));
    assert!(panels.model_open && panels.log_open && panels.needs_frame());
}
#[test]
fn closing_the_active_log_returns_to_the_model() {
    let mut panels = DockPanels::default();
    panels.open_panel(DockPanel::Log);
    panels.close_panel(DockPanel::Log);
    assert_eq!(panels.select, Some(DockPanel::Model));
    assert!(panels.dock_visible());
}
#[test]
fn closing_the_last_panel_empties_the_dock() {
    let mut panels = DockPanels::default();
    panels.close_panel(DockPanel::Model);
    assert!(!panels.dock_visible());
    assert_eq!(panels.select, None);
    panels.open_panel(DockPanel::Model);
    assert!(panels.dock_visible());
}
#[test]
fn closing_the_model_leaves_an_open_log_in_the_dock() {
    let mut panels = DockPanels::default();
    panels.open_panel(DockPanel::Log);
    panels.close_panel(DockPanel::Model);
    assert_eq!(panels.select, Some(DockPanel::Log));
    assert!(panels.dock_visible() && !panels.model_open);
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
