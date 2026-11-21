// SPDX-License-Identifier: GPL-2.0-only
//! Exercise cancellation after choosing a base point with the native Move panel.
use super::{ApplicationWorld, diagnostics, move_precise};
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;

#[given("Move reopens with Optimize for Single Selection checked")]
fn reopen_move_with_single_selection(world: &mut ApplicationWorld) {
    world.remember_edited_sketch(vec![move_precise::EDGE]);
    move_precise::open_move(world);
    if !world.state().sketch_move.unwrap().optimize_single {
        let toggle = world
            .run_in_window("Move", "OptimizeforSingleSelection")
            .center();
        world.click(diagnostics::pointer_pixels(toggle));
    }
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    move_precise::open_move(world);
    assert!(world.state().sketch_move.unwrap().optimize_single);
}

#[given("Copy and Precise Input are unchecked")]
fn move_options_are_off(world: &mut ApplicationWorld) {
    assert!(!world.state().sketch_move.unwrap().copy);
    assert!(diagnostics::window_geometry("Precise Input").is_none());
}

#[when("the user selects the ellipse and clicks its center as base point")]
fn select_ellipse_and_base(world: &mut ApplicationWorld) {
    world.sketch_click(move_precise::EDGE);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    assert!(diagnostics::window_geometry("Precise Input").is_none());
    world.sketch_click(move_precise::CENTER);
}

#[then("the status requests an endpoint for Move")]
fn endpoint_prompt(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Target
    );
    world.expect_status("Move: Select end point");
}

#[when("the user presses Escape before accepting an endpoint")]
fn cancel_before_endpoint(world: &mut ApplicationWorld) {
    world.escape();
}

#[then("Move closes and the ellipse remains in its original position")]
fn original_ellipse_remains(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_move.is_none());
    assert!(crate::HeadSketchWork::sketch_preview(&world.app().owner).is_none());
    world.expect_ready();
    world.expect_edited_sketch_retained();
}

#[given("Move has Copy and Precise Input off")]
fn move_without_copy_or_precise_input(world: &mut ApplicationWorld) {
    world.remember_edited_sketch(vec![move_precise::EDGE]);
    move_precise::open_move(world);
    assert!(!world.state().sketch_move.unwrap().copy);
    if world.state().sketch_move.unwrap().optimize_single {
        toggle_single_selection(world);
    }
}

fn toggle_single_selection(world: &mut ApplicationWorld) {
    let toggle = world
        .run_in_window("Move", "OptimizeforSingleSelection")
        .center();
    world.click(diagnostics::pointer_pixels(toggle));
}

#[when("the user enables Optimize for Single Selection and clicks the ellipse perimeter")]
fn select_one_ellipse(world: &mut ApplicationWorld) {
    toggle_single_selection(world);
    assert!(world.state().sketch_move.unwrap().optimize_single);
    world.sketch_click(move_precise::EDGE);
}

#[then("Base Point activates without a separate button click")]
fn single_selection_advances(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    assert!(diagnostics::window_geometry("Precise Input").is_none());
}

#[then("the status reads Select base point")]
fn request_base_point(world: &mut ApplicationWorld) {
    world.expect_status("Move: Select base point");
}

#[when("the user presses Escape before choosing a base point")]
fn cancel_before_base(world: &mut ApplicationWorld) {
    world.escape();
}

#[cfg(target_os = "linux")]
#[test]
fn rejected_move_can_be_cancelled_with_native_mouse() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    move_precise::open_move(&mut world);
    reject_same_endpoint(&mut world);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::ZeroDisplacement
    );
    world.expect_text("The base point and end point must be different.");
    world.save_capture("move-rejected-endpoint-4020");
    let cancel = world
        .run_in_window("Move could not be completed", "Cancel")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(cancel));
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    assert!(world.state().sketch_selection.entities.is_empty());
    world.expect_edited_sketch_retained();
    reject_same_endpoint(&mut world);
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    world.expect_edited_sketch_retained();
    reject_same_endpoint(&mut world);
    crate::HeadSketchWork::sketch_move_from_gui(
        &mut world.app().owner,
        oblikovati_gui::GuiSketchMoveAction::Done,
    )
    .unwrap();
    world.pump(Vec::new());
    assert!(world.state().sketch_move.is_none());
    assert!(diagnostics::window_geometry("Move could not be completed").is_none());
    world.expect_edited_sketch_retained();
    world.close();
}

#[cfg(target_os = "linux")]
fn reject_same_endpoint(world: &mut ApplicationWorld) {
    for point in [
        move_precise::EDGE,
        move_precise::CENTER,
        move_precise::CENTER,
    ] {
        super::super::x11_input::click(point);
        world.pump(Vec::new());
    }
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::ZeroDisplacement
    );
}
