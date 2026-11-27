// SPDX-License-Identifier: GPL-2.0-only
//! Native Move preview and cancel on a selected sketch line (#4040).
use super::super::super::ApplicationWorld;
use crate::HeadSketchWork;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;

const SHIFT: [i32; 2] = [0, -100];

#[given("a sketch modification tool is requesting existing geometry")]
fn move_is_open(world: &mut ApplicationWorld) {
    super::add_isolated_line(world);
    world.remember_edited_sketch(Vec::new());
    open_move(world);
    ensure_single_selection(world);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
}

fn open_move(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "Move") {
        let button = super::super::move_precise::collapsed_modify_button(world);
        super::native_click(world, button);
    }
    let center = world.run("Move").center();
    super::native_click(
        world,
        super::super::super::diagnostics::pointer_pixels(center),
    );
    assert!(world.state().sketch_move.is_some());
}

fn ensure_single_selection(world: &mut ApplicationWorld) {
    if !world.state().sketch_move.unwrap().optimize_single {
        let center = world
            .run_in_window("Move", "OptimizeforSingleSelection")
            .center();
        super::native_click(
            world,
            super::super::super::diagnostics::pointer_pixels(center),
        );
    }
    assert!(world.state().sketch_move.unwrap().optimize_single);
    assert!(!world.state().sketch_move.unwrap().copy);
}

#[given("the tool has not committed its result")]
fn no_move_committed(world: &mut ApplicationWorld) {
    assert!(world.app().owner.sketch_preview().is_none());
    world.expect_edited_sketch_retained();
}

#[when("the user picks the required sketch entities")]
fn select_line_and_base(world: &mut ApplicationWorld) {
    select_line(world);
    let base = choose_base(world);
    preview_target(world, base);
}

fn select_line(world: &mut ApplicationWorld) {
    let point = world.line_point(0, 0.25);
    super::native_click(world, point);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
}

fn choose_base(world: &mut ApplicationWorld) -> [i32; 2] {
    let base = world.line_point(0, 0.5);
    super::native_click(world, base);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Target
    );
    base
}

fn preview_target(world: &mut ApplicationWorld, base: [i32; 2]) {
    let target = [base[0] + SHIFT[0], base[1] + SHIFT[1]];
    #[cfg(target_os = "linux")]
    super::super::super::x11_input::move_pointer(target);
    #[cfg(not(target_os = "linux"))]
    world.hover(target);
    world.pump(Vec::new());
}

#[then("the tool displays a pending preview of the modification")]
fn move_preview_is_visible(world: &mut ApplicationWorld) {
    assert!(world.app().owner.sketch_preview().is_some());
    let pixel = world.line_point(0, 0.5);
    let moved = [pixel[0] + SHIFT[0], pixel[1] + SHIFT[1]];
    world.expect_segment_drawn(moved, moved);
    world.save_capture("sketch-move-preview-4040");
}

#[when("the user cancels before committing the result")]
fn cancel_move(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    super::super::super::x11_input::press_escape();
    #[cfg(not(target_os = "linux"))]
    world.escape();
    world.pump(Vec::new());
}

#[then("the original sketch geometry remains unchanged")]
fn move_keeps_source(world: &mut ApplicationWorld) {
    world.expect_edited_sketch_retained();
    let pixel = world.line_point(0, 0.5);
    let moved = [pixel[0] + SHIFT[0], pixel[1] + SHIFT[1]];
    world.expect_segment_absent(moved, moved);
    world.save_capture("sketch-move-cancel-4040");
}
