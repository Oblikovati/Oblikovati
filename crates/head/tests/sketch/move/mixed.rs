// SPDX-License-Identifier: GPL-2.0-only
//! Mixed coordinate and angle entry through the Move window.
use super::{ApplicationWorld, diagnostics, move_precise};
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_viewport::ViewportPhysicalKey;

#[given("Move has Copy off and Precise Input and Optimize for Single Selection on")]
fn prepare_move(world: &mut ApplicationWorld) {
    move_precise::open_move(world);
    let state = world.state().sketch_move.unwrap();
    assert!(!state.copy && state.optimize_single);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
}

#[given("the ellipse is selected with X coordinate and X axis angle mode")]
fn select_x_angle(world: &mut ApplicationWorld) {
    world.sketch_click(move_precise::EDGE);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    set_relative(world);
    choose_mode(world, "X coordinate and X axis angle");
}

fn set_relative(world: &mut ApplicationWorld) {
    let button = world.run_in_window("Precise Input", "Abs").center();
    world.click(diagnostics::pointer_pixels(button));
    world.click_text("Relative");
}

fn choose_mode(world: &mut ApplicationWorld, mode: &str) {
    let button = world.run_in_window("Precise Input", "Mode").center();
    world.click(diagnostics::pointer_pixels(button));
    world.click_text(mode);
}

#[when("its recorded center is chosen as the base point")]
#[when("the ellipse center is chosen as the base point")]
fn choose_center(world: &mut ApplicationWorld) {
    world.sketch_click(move_precise::CENTER);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Target
    );
}

#[when("individual keys enter X=10 and Angle=45")]
fn enter_x_angle(world: &mut ApplicationWorld) {
    enter_values(world, "X", "10");
    world.observed_counts = vec![10];
}

#[when("individual keys replace Y with 5 and Angle with 45")]
fn enter_y_angle(world: &mut ApplicationWorld) {
    enter_values(world, "Y", "5");
    world.observed_counts = vec![5];
}

fn enter_values(world: &mut ApplicationWorld, field: &str, value: &str) {
    world.click_precise_field(field);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("45");
}

#[when("Return is pressed")]
fn accept_endpoint(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
}

#[then("Move returns to geometry selection without an error or question")]
fn geometry_selection(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    world.expect_status("Move: Select geometry");
    world.expect_no_text("Constraint Settings");
}

#[then("one ellipse remains at the up-left position")]
fn check_displacement(world: &mut ApplicationWorld) {
    let previous = &world.observed_sketch.as_ref().unwrap().geometry;
    let id = previous.ellipses()[0].center();
    let origin = previous.point(id).unwrap().position;
    let current = world.edited_geometry();
    assert_eq!(current.ellipses().len(), 1);
    let moved = current.point(id).unwrap().position;
    let expected = world.observed_counts[0] as f64 / 10.;
    assert!((moved[0] - origin[0] - expected).abs() < 1e-8);
    assert!((moved[1] - origin[1] - expected).abs() < 1e-8);
    let before = world.sketch_pixel_of(origin);
    let after = world.sketch_pixel_of(moved);
    assert!(
        after[0] < before[0] && after[1] < before[1],
        "{before:?} -> {after:?}"
    );
    let edge = [
        move_precise::EDGE[0] + after[0] - before[0],
        move_precise::EDGE[1] + after[1] - before[1],
    ];
    world.expect_segment_drawn(edge, edge);
}

#[when("Done and undoes are used")]
fn finish_and_undo(world: &mut ApplicationWorld) {
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    world.undo();
}

#[then("the ellipse returns to its original visible position")]
#[then("the original visible ellipse position is restored")]
fn original_restored(world: &mut ApplicationWorld) {
    let original = world.observed_sketch.as_ref().unwrap().geometry.clone();
    assert_eq!(world.edited_geometry(), original);
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap();
    assert_eq!(session.redo_labels().last(), Some(&"Move sketch entities"));
    world.expect_segment_drawn(move_precise::EDGE, move_precise::EDGE);
}

#[given("reopening Move retains the previously selected mixed X/angle mode")]
fn reopen_x_angle(world: &mut ApplicationWorld) {
    move_precise::select_ellipse(world);
    choose_mode(world, "X coordinate and X axis angle");
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    move_precise::open_move(world);
    world.sketch_click(move_precise::EDGE);
    let text = world.window_text("Precise Input");
    assert!(text.contains("X") && text.contains("Angle"), "{text:?}");
}

#[when("Y coordinate and X axis angle is selected")]
fn select_y_angle(world: &mut ApplicationWorld) {
    choose_mode(world, "Y coordinate and X axis angle");
}
