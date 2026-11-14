// SPDX-License-Identifier: GPL-2.0-only
//! Finish a sketch from a non-sketch ribbon tab through the native window.
use super::*;

#[test]
fn tools_keeps_finish_until_the_sketch_is_finished() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let document = world.state().document;
    world.click_text("Tools");
    world.pump(Vec::new());
    world.expect_text("Application Options");
    world.expect_text("Finish Sketch");
    assert!(world.run_is_enabled("Finish Sketch"));
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    assert_eq!(world.state().document, document);
    world.expect_text("Application Options");
    world.expect_no_text("Finish Sketch");
    world.close();
}

#[test]
fn view_keeps_finish_until_the_sketch_is_finished() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("View");
    world.pump(Vec::new());
    world.expect_text("User Interface");
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.expect_text("User Interface");
    world.expect_no_text("Finish Sketch");
    world.close();
}

use super::super::workspace_layout::browser_feature::{
    SKETCH, expect_solid_visible, open_visible_extruded_part, right_click_row,
};
use cucumber::{given, then, when};

#[given("an existing sketch is being edited in the MCAD Software")]
fn existing_sketch_under_edit(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the Sketch ribbon is minimized to tabs")]
fn minimized_sketch_ribbon(world: &mut ApplicationWorld) {
    choose_tabs_mode(world);
    world.expect_no_text("Finish Sketch");
}
#[when("the user clicks the Sketch tab")]
fn open_sketch_tab(world: &mut ApplicationWorld) {
    world.click_text("Sketch");
    world.pump(Vec::new());
}
#[then("the Sketch command groups appear temporarily beneath the tabs")]
fn temporary_sketch_groups(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_some());
    world.expect_text("Rectangle");
}
#[then("the temporary ribbon includes the Finish action")]
fn temporary_finish(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Finish Sketch"));
}
#[when("the user selects Finish")]
fn finish_from_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Finish Sketch");
    world.pump(Vec::new());
}
#[then("sketch editing ends and the part model is displayed")]
fn finished_model_visible(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    expect_solid_visible(world);
}
