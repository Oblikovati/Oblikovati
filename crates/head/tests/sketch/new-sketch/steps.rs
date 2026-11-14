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
    assert_eq!(
        world.state().browser.unwrap().sketches[1]
            .inspection
            .as_ref()
            .unwrap()
            .plane,
        "XZ"
    );
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

#[given("the user have created empty Sketch2 and chosen Finish Sketch")]
fn finished_second_sketch(world: &mut ApplicationWorld) {
    one_finished_sketch(world);
    choose_xz(world);
    world.finish_sketch();
    assert_eq!(world.browser_sketch_count(), 2);
}
#[when("the user press Ctrl+Z once")]
#[when("the user press Ctrl+Z again")]
fn undo_sketch_step(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("Sketch2 remains and its editing environment returns")]
fn undo_finish_reenters_sketch(world: &mut ApplicationWorld) {
    second_sketch_active(world);
}
#[then("Sketch2 disappears from the model browser")]
fn undo_creation_removes_sketch(world: &mut ApplicationWorld) {
    only_first_sketch(world);
}
#[when("the user press Ctrl+Y twice in sequence")]
fn redo_creation_and_finish(world: &mut ApplicationWorld) {
    world.redo();
    second_sketch_active(world);
    world.redo();
}
#[then("creation and Finish Sketch are restored in order")]
fn redo_restores_finished_sketch(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 2);
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    world.expect_text("Sketch2");
}

#[given("the model browser exposes YZ Plane")]
fn two_existing_sketches(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
    world.add_finished_sketch();
    viewport_orientation::press_part(world, |layout| layout.home);
    assert_eq!(world.browser_sketch_count(), 2);
    world.expect_text("YZ Plane");
}
#[when("the user select YZ Plane before invoking Start 2D Sketch")]
fn start_with_selected_plane(world: &mut ApplicationWorld) {
    world.click_text("YZ Plane");
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
}
#[then("Sketch3 is created without another support-selection click")]
fn third_sketch_started(world: &mut ApplicationWorld) {
    assert_eq!(world.browser_sketch_count(), 3);
    let state = world.state();
    assert_eq!(state.authoring, GuiAuthoringContext::Sketch);
    let sketch = &state.browser.unwrap().sketches[2];
    assert_eq!(sketch.inspection.as_ref().unwrap().plane, "YZ");
    assert_eq!(state.active_sketch, Some(sketch.id));
    assert!(diagnostics::window_geometry("##plane-selection").is_none());
}
#[then("the ViewCube reads RIGHT")]
fn cube_reads_right(world: &mut ApplicationWorld) {
    assert_eq!(viewport_orientation::control_letters(world), "Rghit");
}

#[test]
fn origin_preselection_is_consumed_and_does_not_cross_documents() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("XZ Plane");
    world.click_text("Start 2D Sketch");
    assert_eq!(world.browser_sketch_count(), 1);
    assert_eq!(
        world.state().browser.unwrap().sketches[0]
            .inspection
            .as_ref()
            .unwrap()
            .plane,
        "XZ"
    );
    world.finish_sketch();
    world.click_text("Start 2D Sketch");
    assert!(diagnostics::window_geometry("##plane-selection").is_some());
    world.escape();
    world.click_text("XY Plane");
    world.add_part();
    world.click_text("Start 2D Sketch");
    assert_eq!(world.browser_sketch_count(), 0);
    assert!(diagnostics::window_geometry("##plane-selection").is_some());
    world.close();
}
