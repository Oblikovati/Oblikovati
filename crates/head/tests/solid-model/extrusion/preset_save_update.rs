// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{direction::preview_depth, replace_extrusion_distance};
use super::super::feature_count;
use super::{
    ApplicationWorld, EMPTY, add_named_preset, assert_distance, click_in_extrude, click_menu_row,
    menu_row, open_preset_menu, rename_preset,
};
use cucumber::{given, then, when};

const ORIGINAL: &str = "Saveddepth";
const RENAMED: &str = "Shortdepth";

#[given("for Revolve the long Y=0 sketch edge is the selected axis")]
fn axis_setup_applies_to_revolve(world: &mut ApplicationWorld) {
    // The Extrude example has an active rectangular preview and needs no axis.
    assert!(world.state().extrude_preview_active);
    world.run_in_window("Extrude", "1Profile");
}

#[when("the user saves the draft using the preset row add button and gives it a unique name")]
fn save_new_preset(world: &mut ApplicationWorld) {
    assert_distance(world, "10.00 mm");
    add_named_preset(world, "Saved depth");
    world.run_in_window("Extrude", ORIGINAL);
}

#[then("the named preset is selectable from the preset list")]
fn select_original_preset(world: &mut ApplicationWorld) {
    click_in_extrude(world, ORIGINAL);
    click_menu_row(world, ORIGINAL);
    assert_distance(world, "10.00 mm");
}

#[when("the user changes the main value and chooses Save Current")]
fn save_edited_distance(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    open_preset_menu(world, ORIGINAL);
    click_menu_row(world, "SaveCurrent");
}

#[when("the user reselects the saved preset")]
fn reselect_updated_preset(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "6");
    click_in_extrude(world, ORIGINAL);
    click_menu_row(world, EMPTY);
    assert_distance(world, "6");
    click_in_extrude(world, EMPTY);
    click_menu_row(world, ORIGINAL);
}

#[then("the saved value is restored into the panel")]
fn assert_eight_millimetres(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "8");
    assert_distance(world, "8");
    assert_eq!(preview_depth(world), [0., 0.8]);
    assert_eq!(feature_count(world), 0);
}

#[when("the user chooses Rename Current and enters a different name")]
fn rename_current_preset(world: &mut ApplicationWorld) {
    rename_preset(world, ORIGINAL, "Short depth");
    world.run_in_window("Extrude", RENAMED);
}

#[then("the list uses the new name for the same stored settings")]
fn check_renamed_preset(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "5");
    click_in_extrude(world, RENAMED);
    menu_row(world, RENAMED);
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| row.text == ORIGINAL && world.frame.is_popup_row(row))
    );
    click_menu_row(world, RENAMED);
    assert_eight_millimetres(world);
}
