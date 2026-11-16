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
