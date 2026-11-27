// SPDX-License-Identifier: GPL-2.0-only
//! Use the published Parts for Home, the Switch menu, and tab previews.
use super::super::{saved_sketch_parts, saved_tabs};
use super::*;
use cucumber::given;
use oblikovati_api::types::DOCUMENT_PART;
use std::path::PathBuf;

const EMPTY_SKETCH: &str = "fixtures/cad/planar-sketch-empty-xy.opd";
const PROFILE: &str = "fixtures/cad/revolve/revolve-profile.opd";

#[given("the user opens View Switch, chooses More Windows and highlights PartTwo in Select Window")]
fn choose_more_and_highlight_part_two(world: &mut ApplicationWorld) {
    choose_more(world);
    let row = world.run_in_window("Select Window", "PartTwo");
    world.click(row.center().map(|value| value as i32));
    world.expect_text("Select Window");
}

#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/planar-sketch-empty-xy.opd` are open"
)]
fn saved_part_one_and_sketch(world: &mut ApplicationWorld) {
    world.restore_layout_without_asset_tools();
    saved_tabs::open_disposable_part(world, "PartOne.opd");
    let sketch = saved_tabs::open_disposable_tab(world, EMPTY_SKETCH, DOCUMENT_PART);
    world.parts.push(sketch);
}

#[given("the user edits Sketch1 in the second part and activates the first part")]
fn edit_second_sketch(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.parts[1]);
    saved_tabs::edit_active_saved_sketch(world);
    world.activate_through_tab(world.parts[0]);
}

#[given("the second part's empty Sketch1 is visible and the first part is active")]
fn show_second_sketch(world: &mut ApplicationWorld) {
    let [first, second] = [world.parts[0], world.parts[1]];
    world.activate_through_tab(second);
    saved_sketch_parts::expect_sketch_lines(world, 0);
    saved_sketch_parts::second_sketch_is_visible(world);
    world.activate_through_tab(first);
    world.previous_document = Some(first);
}

#[given("`fixtures/cad/revolve/revolve-profile.opd` is open with Sketch1 being edited")]
fn edit_saved_profile(world: &mut ApplicationWorld) {
    world.restore_layout_without_asset_tools();
    let part = saved_tabs::open_disposable_tab(world, PROFILE, DOCUMENT_PART);
    world.parts.push(part);
    world.previous_document = Some(part);
    world.other_document = Some(part);
    world.saved_path = world.state().document_storage[&part]
        .path
        .as_deref()
        .map(PathBuf::from);
    saved_tabs::edit_active_saved_sketch(world);
}

#[given("the second part has been saved to a file")]
fn second_saved(world: &mut ApplicationWorld) {
    let [first, second] = [world.parts[0], world.parts[1]];
    let path = world.state().document_storage[&second]
        .path
        .as_deref()
        .map(PathBuf::from)
        .unwrap_or_else(|| world.save_part_to_file(second, "Saved.opd"));
    world.saved_path = Some(path);
    world.activate_through_tab(second);
    world.pump(Vec::new());
    world.activate_through_tab(first);
}

#[given("Home and the part window are adjacent in the tab strip")]
fn home_and_part_adjacent(world: &mut ApplicationWorld) {
    let part = world.active_id();
    world.previous_document = Some(part);
    let name = world.document_name(part);
    let home = world.expect_tab("Home").center()[0];
    let part_x = world.expect_tab(&name).center()[0];
    assert!(home < part_x && world.state().documents.len() == 1);
}
