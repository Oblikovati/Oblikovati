// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{acceptance::feature_distance, replace_extrusion_distance};
use super::super::{apply_next::click_settled, feature_count};
use super::{
    ApplicationWorld, EMPTY, add_named_preset, assert_distance, cancel_extrude, click_in_extrude,
    click_menu_row, menu_row, open_extrude_with_presets, open_preset_menu, open_settings,
    restore_hidden, reveal_presets,
};
use cucumber::{given, then, when};

const PRESET: &str = "Saveddepth";

#[given("the rectangular sketch profile is selected for Extrude")]
fn prepare_profile_for_default(world: &mut ApplicationWorld) {
    open_extrude_with_presets(world);
    assert_eq!(feature_count(world), 0);
}

#[given("the user saves a Extrude preset with Distance A set to 8 mm")]
fn save_eight_millimetre_default(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    add_named_preset(world, EMPTY, "Saved depth");
    world.run_in_window("Extrude", PRESET);
    assert_distance(world, "8");
}

#[given("the user accepts Extrude with Distance A set to 12 mm")]
fn accept_twelve_millimetre_last_use(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "12");
    restore_hidden(world);
    click_settled(world, "OK");
    assert_eq!(feature_count(world), 1);
    assert!((feature_distance(world) - 12.).abs() < 1e-10);
}

#[given("the user reopens Extrude, reveals the preset row and selects the saved preset")]
fn reselect_saved_default(world: &mut ApplicationWorld) {
    open_new_extrude(world);
    assert!(!world.window_text("Extrude").contains(EMPTY));
    open_settings(world);
    reveal_presets(world);
    click_in_extrude(world, EMPTY);
    click_menu_row(world, PRESET);
    assert_distance(world, "8");
}

#[when(
    regex = r"^the user chooses (No Preset|Last Used|Use Current Preset) as the new-command default$"
)]
fn choose_extrude_default(world: &mut ApplicationWorld, default: String) {
    let label = default.replace(' ', "");
    open_preset_menu(world, PRESET);
    click_menu_row(world, &label);
    open_preset_menu(world, PRESET);
    assert!(world.checked_run(menu_row(world, &label)));
    world.escape();
    world.pump(Vec::new());
}

#[when("the user cancels the uncommitted command draft")]
fn cancel_preset_draft(world: &mut ApplicationWorld) {
    cancel_extrude(world);
}

#[when("the user opens Extrude for a new feature")]
fn open_new_extrude(world: &mut ApplicationWorld) {
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "1Profile");
}

#[then("the panel starts with the command's built-in settings")]
fn starts_with_builtin_distance(world: &mut ApplicationWorld) {
    assert_default_distance(world, "10.00 mm");
    world.run_in_window("Extrude", EMPTY);
}

#[then("the panel starts with the last-used command settings")]
fn starts_with_last_used_distance(world: &mut ApplicationWorld) {
    assert_default_distance(world, "12");
    world.run_in_window("Extrude", EMPTY);
}

#[then("the panel starts with the chosen named preset's settings")]
fn starts_with_named_distance(world: &mut ApplicationWorld) {
    assert_default_distance(world, "8");
    world.run_in_window("Extrude", PRESET);
}

fn assert_default_distance(world: &mut ApplicationWorld, distance: &str) {
    assert_distance(world, distance);
    assert_eq!(feature_count(world), 1);
    assert!((feature_distance(world) - 12.).abs() < 1e-10);
    assert_eq!(world.state().undo_labels.len(), 1);
    assert!(world.run_is_enabled("OK"));
}
