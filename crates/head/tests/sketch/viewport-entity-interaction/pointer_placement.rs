// SPDX-License-Identifier: GPL-2.0-only
//! Pointer-driven line creation and cancellation through the native window.
use super::super::super::ApplicationWorld;
use crate::HeadSketchWork;
use cucumber::{given, then, when};

const START: [i32; 2] = [620, 300];
const END: [i32; 2] = [840, 420];
const NEXT: [i32; 2] = [980, 340];

#[given("a sketch geometry tool is active and requesting viewport input")]
fn line_tool_active(world: &mut ApplicationWorld) {
    let center = world.run("Line").center();
    let point = super::super::super::diagnostics::pointer_pixels(center);
    super::native_click(world, point);
    world.expect_status("Select start of line");
}

#[when("the user selects the positions required by that tool")]
fn place_line(world: &mut ApplicationWorld) {
    super::native_click(world, START);
    super::native_click(world, END);
}

#[then("accepted geometry appears in the sketch")]
fn line_is_accepted(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 1);
    world.expect_segment_drawn(START, END);
    world.record_initial_sketch();
}

#[then("the tool advances to its next input or returns to Ready")]
fn line_advances(world: &mut ApplicationWorld) {
    world.expect_status("Select end of line");
}

#[when("the user presses Escape while a later placement preview is pending")]
fn cancel_pending_line(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    super::super::super::x11_input::move_pointer(NEXT);
    #[cfg(not(target_os = "linux"))]
    world.hover(NEXT);
    world.pump(Vec::new());
    assert!(world.app().owner.sketch_preview().is_some());
    world.expect_segment_drawn(END, NEXT);
    world.save_capture("sketch-pointer-line-preview-4037");
    #[cfg(target_os = "linux")]
    super::super::super::x11_input::press_escape();
    #[cfg(not(target_os = "linux"))]
    world.escape();
    world.pump(Vec::new());
}

#[then("the pending preview is removed")]
fn preview_is_removed(world: &mut ApplicationWorld) {
    assert!(world.app().owner.sketch_preview().is_none());
    world.expect_segment_absent(END, NEXT);
    assert!(world.state().sketch_at_ready);
}

#[then("previously accepted geometry remains in the sketch")]
fn accepted_line_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 1);
    let after = world.edited_geometry();
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    assert!(after.same_recipe(before));
    world.expect_segment_drawn(START, END);
    world.save_capture("sketch-pointer-line-cancel-4037");
}
