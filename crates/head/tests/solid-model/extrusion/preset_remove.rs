// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{direction::preview_depth, replace_extrusion_distance};
use super::super::feature_count;
use super::{
    ApplicationWorld, EMPTY, add_named_preset, assert_distance, click_in_extrude, click_menu_row,
    menu_row, open_preset_menu,
};
use cucumber::{given, then, when};

const FIRST: &str = "Firstdepth";
const SECOND: &str = "Seconddepth";

#[given(
    "the user saves two uniquely named presets and selects the first in the command's Properties panel"
)]
fn save_two_presets(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    add_named_preset(world, EMPTY, "First depth");
    replace_extrusion_distance(world, "6");
    add_named_preset(world, FIRST, "Second depth");
    click_in_extrude(world, SECOND);
    click_menu_row(world, FIRST);
    assert_distance(world, "8");
    menu_is_closed_and_preview_is_valid(world, 0.8);
}

#[when("the user chooses Delete Current and confirms removal if prompted")]
fn remove_first_preset(world: &mut ApplicationWorld) {
    open_preset_menu(world, FIRST);
    click_menu_row(world, "DeleteCurrent");
    world.run_in_window("Extrude", EMPTY);
}

#[then("that name is absent from the preset list")]
fn first_preset_is_absent(world: &mut ApplicationWorld) {
    click_in_extrude(world, EMPTY);
    menu_row(world, EMPTY);
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| row.text == FIRST && world.frame.is_popup_row(row))
    );
}

#[then("other presets remain available")]
fn second_preset_keeps_its_settings(world: &mut ApplicationWorld) {
    click_menu_row(world, SECOND);
    world.run_in_window("Extrude", SECOND);
    assert_distance(world, "6");
    menu_is_closed_and_preview_is_valid(world, 0.6);
}

fn menu_is_closed_and_preview_is_valid(world: &mut ApplicationWorld, depth: f32) {
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| world.frame.is_popup_row(row))
    );
    assert_eq!(preview_depth(world), [0., depth]);
    assert_eq!(feature_count(world), 0);
    assert!(world.state().undo_labels.is_empty());
    assert!(world.run_is_enabled("OK"));
}
