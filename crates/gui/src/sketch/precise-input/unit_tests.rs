// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn only_a_finite_number_is_a_coordinate() {
    assert!(valid_number("5"));
    assert!(valid_number(" -6.25 "));
    assert!(!valid_number("a"));
    assert!(!valid_number(""));
    assert!(!valid_number("inf"));
}
#[test]
fn a_field_follows_the_pointer_until_a_value_is_typed() {
    let mut field = PreciseField::default();
    field.follow(Some(&"5.000".to_owned()));
    assert_eq!(field.text, "5.000");
    field.typed = true;
    field.follow(Some(&"9.000".to_owned()));
    assert_eq!(field.text, "5.000");
}
#[test]
fn changing_reference_discards_values_typed_for_the_other_origin() {
    let mut bar = PreciseInputBar::default();
    bar.first.text = "5".into();
    bar.first.typed = true;
    bar.second.text = "6".into();
    bar.second.typed = true;
    bar.set_reference(true);
    assert!(bar.relative);
    assert!(bar.first.text.is_empty() && bar.second.text.is_empty());
    assert!(!bar.first.typed && !bar.second.typed);
    bar.first.text = "3".into();
    bar.set_reference(true);
    assert_eq!(bar.first.text, "3");
}
#[test]
fn reference_mode_selects_the_matching_visible_triad() {
    let reference = [[30, 40], [58, 40], [30, 12]];
    let origin = [[0, 0], [28, 0], [0, -28]];
    let target = GuiPreciseTarget {
        triad: Some(reference),
        origin_triad: Some(origin),
        ..Default::default()
    };
    assert_eq!(target.displayed_triad(true), Some(reference));
    assert_eq!(target.displayed_triad(false), Some(origin));
}
#[test]
fn auto_fade_keeps_active_editing_and_open_menus_expanded() {
    let mut bar = PreciseInputBar {
        auto_fade: true,
        ..Default::default()
    };
    assert!(!bar.show_full_controls(false));
    assert!(bar.show_full_controls(true));
    bar.first.active = true;
    assert!(bar.show_full_controls(false));
    bar.first.active = false;
    bar.second.active = true;
    assert!(bar.show_full_controls(false));
    bar.second.active = false;
    bar.popup_open = true;
    assert!(bar.show_full_controls(false));
    bar.popup_open = false;
    bar.auto_fade = false;
    assert!(bar.show_full_controls(false));
}
#[test]
fn toggling_a_suspended_toolbar_closes_the_saved_request() {
    let mut bar = PreciseInputBar::default();
    bar.open.set_open(true);
    bar.sync_before_docking(&GuiCommandState::default());
    bar.toggle_visibility();
    let state = GuiCommandState {
        precise_target: Some(GuiPreciseTarget::default()),
        ..Default::default()
    };
    bar.sync_before_docking(&state);
    assert!(!bar.open.is_open());
}
#[test]
fn missing_target_suspends_and_restored_target_resumes() {
    let bar = PreciseInputBar::default();
    bar.open.set_open(true);
    let mut state = GuiCommandState::default();
    bar.sync_before_docking(&state);
    assert!(!bar.open.is_open());
    state.precise_target = Some(GuiPreciseTarget::default());
    bar.sync_before_docking(&state);
    assert!(bar.open.is_open());
}
#[test]
fn closing_toolbar_preserves_move_option_until_explicitly_changed() {
    let mut bar = PreciseInputBar::default();
    bar.open.set_open(true);
    bar.sync_move_context(true);
    bar.open.set_open(false);
    assert!(bar.move_option());
    bar.set_move_option(false);
    assert!(!bar.open.is_open() && !bar.move_option());
    bar.set_move_option(true);
    assert!(bar.open.is_open() && bar.move_option());
    bar.toggle_visibility();
    assert!(!bar.open.is_open() && !bar.move_option());
    bar.sync_move_context(false);
    bar.open.set_open(true);
    bar.sync_move_context(true);
    assert!(bar.move_option());
}
#[test]
fn pin_locks_only_the_toolbar_position() {
    let mut bar = PreciseInputBar::default();
    assert!(!bar.window_flags().contains(WindowFlags::NO_MOVE));
    bar.pinned = true;
    assert!(bar.window_flags().contains(WindowFlags::NO_MOVE));
    assert!(!bar.window_flags().contains(WindowFlags::NO_INPUTS));
    bar.pinned = false;
    assert!(!bar.window_flags().contains(WindowFlags::NO_MOVE));
}
#[test]
fn each_mode_names_its_two_values() {
    assert_eq!(GuiPreciseMode::XAndY.field_labels(), ["X", "Y"]);
    assert_eq!(
        GuiPreciseMode::DistanceAndAngle.field_labels(),
        ["Distance", "Angle"]
    );
}
