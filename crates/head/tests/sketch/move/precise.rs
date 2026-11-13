// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard and mouse coverage for the precise ellipse Move scenario.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_viewport::ViewportPhysicalKey;

const CENTER: [i32; 2] = [850, 550];
const EDGE: [i32; 2] = [1010, 470];

#[given("an editable sketch with eligible geometry")]
fn editable_ellipse(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.choose_from_flyout("Circle", "Ellipse");
    for point in [CENTER, EDGE, [880, 610]] {
        world.sketch_click(point);
    }
    world.escape();
    world.remember_edited_sketch(Vec::new());
}

#[given("the ellipse is selected in Move with Copy off and Precise Input on")]
fn select_ellipse(world: &mut ApplicationWorld) {
    world.click_text("Move");
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    world.sketch_click(EDGE);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    let reference = world.run_in_window("Precise Input", "Abs").center();
    world.click(diagnostics::pointer_pixels(reference));
    world.click_text("Relative");
}

#[when("I choose Base Point and click its center")]
fn choose_base(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}

#[then("the coordinate toolbar appears and the status requests an endpoint")]
fn endpoint_requested(world: &mut ApplicationWorld) {
    world.run_in_window("Precise Input", "Rel");
    world.expect_status("Move: Select end point");
}

#[when("I type 1 then 0 into X and Tab to Y")]
fn type_x(world: &mut ApplicationWorld) {
    world.click_precise_field("X");
    world.type_text("10");
    world.key(ViewportPhysicalKey::Tab, false);
}

#[when("I type 0 and press Enter")]
fn type_y(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("Move returns to geometry selection without an error dialog")]
fn move_ready(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    world.expect_status("Move: Select geometry");
}

#[then("the ellipse appears above its original position")]
fn ellipse_moved(world: &mut ApplicationWorld) {
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    let center = before.ellipses()[0].center();
    let original = before.point(center).unwrap().position;
    let moved = world.edited_geometry().point(center).unwrap().position;
    let old_pixel = world.sketch_pixel_of(original);
    let new_pixel = world.sketch_pixel_of(moved);
    assert!(
        new_pixel[1] < old_pixel[1],
        "ellipse center {old_pixel:?} -> {new_pixel:?}"
    );
    world.observed_view = Some(new_pixel);
}

#[then("Move closes and the displaced ellipse remains")]
fn finished(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_move.is_none());
    let geometry = world.edited_geometry();
    let center = geometry.ellipses()[0].center();
    assert_eq!(
        world.sketch_pixel_of(geometry.point(center).unwrap().position),
        world.observed_view.unwrap()
    );
}
