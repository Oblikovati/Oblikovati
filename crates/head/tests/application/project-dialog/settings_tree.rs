// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard and selection checks for the project settings tree.
use super::{ApplicationWorld, choose_button, holds_text};
use cucumber::{given, then, when};

fn press_arrow(world: &mut ApplicationWorld, key: oblikovati_viewport::ViewportPhysicalKey) {
    world.key(key, false);
    world.pump(Vec::new());
}
fn select_group(world: &mut ApplicationWorld, group: &str) {
    choose_button(world, group);
}
#[given("an active project with Folder Options and Options groups is displayed")]
#[given("`ActiveProject` displays collapsed Folder Options and Options groups")]
fn groups_shown(world: &mut ApplicationWorld) {
    world.expect_text("Folder Options");
    world.expect_text("Options");
    world.expect_no_text("Presets:");
    world.expect_no_text("Old Versions To Keep");
}
#[when("I select the collapsed Folder Options group")]
#[when("the user selects the collapsed Folder Options group")]
fn select_folder_options(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Presets"),
        "expected the group collapsed"
    );
    select_group(world, "Folder Options");
}
#[when("I press the Right Arrow key")]
fn press_right(world: &mut ApplicationWorld) {
    press_arrow(world, oblikovati_viewport::ViewportPhysicalKey::RightArrow);
}
#[when("I press the Left Arrow key")]
fn press_left(world: &mut ApplicationWorld) {
    press_arrow(world, oblikovati_viewport::ViewportPhysicalKey::LeftArrow);
}
#[then("the group expands to show folder entries including Templates, Design Data and Presets")]
pub(super) fn folder_entries(world: &mut ApplicationWorld) {
    for row in ["Templates:", "Design Data:", "Presets:"] {
        assert!(holds_text(world, row), "expected {row} in the tree");
    }
}
#[then("the Folder Options children are hidden")]
pub(super) fn folder_entries_hidden(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Presets:"),
        "expected the group collapsed"
    );
}
#[when("I select the collapsed Options group")]
#[when("the user selects the collapsed Options group")]
fn select_options(world: &mut ApplicationWorld) {
    select_group(world, "Options");
}
#[then(
    "the group expands to show project options including old versions to keep and unique file names"
)]
fn option_entries(world: &mut ApplicationWorld) {
    for row in ["Old Versions To Keep", "Using Unique File Names"] {
        assert!(holds_text(world, row), "expected {row} in the tree");
    }
}
#[given(
    "the selected project has collapsed Appearance Libraries and Material Libraries groups with entries"
)]
#[given(
    "`ActiveProject` has collapsed Appearance Libraries and Material Libraries groups with entries"
)]
fn library_groups(world: &mut ApplicationWorld) {
    world.expect_text("Appearance Libraries");
    world.expect_text("Material Libraries");
    assert!(
        !holds_text(world, "Default Appearances"),
        "expected the group collapsed"
    );
    material_hidden(world);
}
#[when("I select Appearance Libraries and press the Right Arrow key")]
#[when("the user selects Appearance Libraries and presses the Right Arrow key")]
fn open_appearances(world: &mut ApplicationWorld) {
    select_group(world, "Appearance Libraries");
    press_right(world);
}
#[then("the appearance library entries are visible beneath their group")]
#[then("the appearance library entries remain visible")]
fn appearance_entries(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "Default Appearances"),
        "expected the entries"
    );
}
#[when("I select Material Libraries and press the Right Arrow key")]
#[when("the user selects Material Libraries and presses the Right Arrow key")]
fn open_materials(world: &mut ApplicationWorld) {
    select_group(world, "Material Libraries");
    press_right(world);
}
#[then("the material library entries are visible beneath their group")]
fn material_entries(world: &mut ApplicationWorld) {
    assert!(
        holds_text(world, "Default Materials"),
        "expected the entries"
    );
}
#[then("the material library entries are hidden")]
fn material_hidden(world: &mut ApplicationWorld) {
    assert!(
        !holds_text(world, "Default Materials"),
        "expected the group collapsed"
    );
}
