// SPDX-License-Identifier: GPL-2.0-only
//! Named support selection and cancellation in `features/sketch/new-sketch`.
use super::super::{ApplicationWorld, diagnostics, viewport_orientation};
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;

fn one_finished_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
    assert_eq!(world.browser_sketch_count(), 1);
    // Go keeps the viewer's side on sketch entry; the recorded TOP case starts above XZ.
    viewport_orientation::press_part(world, |layout| layout.home);
}
fn choose_xz(world: &mut ApplicationWorld) {
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.click_text("XZ Plane");
    world.pump(Vec::new());
}
#[given("Part2 has Sketch1 and the Origin browser is expanded")]
#[given("Part2 has only Sketch1")]
fn existing_sketch(world: &mut ApplicationWorld) {
    one_finished_sketch(world);
    world.expect_text("XZ Plane");
}
#[when("the user invoke Start 2D Sketch and choose XZ Plane")]
fn start_on_xz(world: &mut ApplicationWorld) {
    choose_xz(world);
}
#[then("Sketch2 is added in the Sketch editing environment")]
fn second_sketch_active(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 2);
    let state = world.state();
    assert_eq!(state.authoring, GuiAuthoringContext::Sketch);
    let sketch = &state.browser.unwrap().sketches[1];
    assert_eq!(sketch.inspection.as_ref().unwrap().plane, "XZ");
    assert_eq!(state.active_sketch, Some(sketch.id));
    world.expect_text("Sketch2");
    world.expect_text("Finish Sketch");
}
#[then("the ViewCube reads TOP")]
fn cube_reads_top(world: &mut ApplicationWorld) {
    assert_eq!(viewport_orientation::control_letters(world), "Top");
}
#[then("the empty sketch status reads Fully Constrained")]
fn empty_fully_constrained(world: &mut ApplicationWorld) {
    assert_eq!(world.dimensions_needed(), 0);
    world.expect_status("Fully Constrained");
}
#[when("the user invoke Start 2D Sketch and press Escape before choosing a plane")]
fn cancel_support_selection(world: &mut ApplicationWorld) {
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##plane-selection").is_some());
    world.escape();
    world.pump(Vec::new());
}
#[then("the reference-plane previews disappear")]
fn no_support_previews(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##plane-selection").is_none());
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
}
#[then("the browser still contains only Sketch1")]
fn only_first_sketch(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 1);
    world.expect_text("Sketch1");
    world.expect_no_text("Sketch2");
}
#[given("selecting XZ Plane has created empty Sketch2")]
fn second_empty_sketch(world: &mut ApplicationWorld) {
    one_finished_sketch(world);
    choose_xz(world);
    second_sketch_active(world);
    empty_fully_constrained(world);
}
#[then("Sketch2 remains active in the Sketch editing environment")]
#[then("creation is not rolled back")]
fn still_editing_second_sketch(world: &mut ApplicationWorld) {
    second_sketch_active(world);
}
