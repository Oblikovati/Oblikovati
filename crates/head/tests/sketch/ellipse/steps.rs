// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/ellipse`: the three defining clicks, cancel, and native visibility.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[path = "alias.rs"]
mod alias;
#[path = "measure.rs"]
mod measure;

const CENTER: [i32; 2] = [850, 550];
const FIRST_AXIS: [i32; 2] = [1010, 470];
const MINOR_AXIS: [i32; 2] = [880, 610];
const OPPOSITE_AXIS: [i32; 2] = [690, 630];
const OPPOSITE_MINOR: [i32; 2] = [820, 490];
const CLEAR_POINTER: [i32; 2] = [1200, 760];
const BOUNDARY: [[i32; 2]; 4] = [FIRST_AXIS, MINOR_AXIS, OPPOSITE_AXIS, OPPOSITE_MINOR];

fn choose_ellipse(world: &mut ApplicationWorld) {
    world.choose_from_flyout("Circle", "Ellipse");
}
fn count_ellipses(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().ellipses().len()
}
fn expect_oval_visible(world: &mut ApplicationWorld) {
    assert_eq!(count_ellipses(world), 1, "ellipse count");
    for position in BOUNDARY {
        world.expect_segment_drawn(position, position);
    }
}
#[when("I select Ellipse and specify a center, first-axis point, and point on the ellipse")]
#[when("the user select Ellipse and specify a center, first-axis point, and point on the ellipse")]
fn create_ellipse(world: &mut ApplicationWorld) {
    choose_ellipse(world);
    for position in [CENTER, FIRST_AXIS, MINOR_AXIS] {
        world.sketch_click(position);
    }
    world.hover(CLEAR_POINTER);
}
#[then("a closed oval appears and the command requests another ellipse center")]
#[then("a tilted closed oval appears and the command requests another center")]
fn created_oval(world: &mut ApplicationWorld) {
    expect_oval_visible(world);
    world.expect_status("Select center");
}
#[then("the oval remains and the sketch returns to Ready")]
fn ellipse_at_ready(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_oval_visible(world);
}
#[given("Ellipse is active in an editable sketch")]
fn ellipse_active(world: &mut ApplicationWorld) {
    choose_ellipse(world);
}
#[when("I select only a center and press Escape")]
#[when("the user select only a center and press Escape")]
fn cancel_after_center(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
    world.escape();
}
#[then("Ready returns without an additional ellipse")]
#[then("Ready returns with no new ellipse")]
#[then("the MCAD Software returns to Ready with no new ellipse visible")]
#[then("the MCAD Software is Ready with no additional ellipse visible")]
fn cancelled_ellipse(world: &mut ApplicationWorld) {
    world.expect_ready();
    expect_no_oval_visible(world);
}
#[when("I invoke Ellipse again, select a center and rotated first axis, and press Escape")]
#[when("the user invoke Ellipse again, select a center and rotated first axis, and press Escape")]
fn cancel_after_axis(world: &mut ApplicationWorld) {
    choose_ellipse(world);
    world.sketch_click(CENTER);
    world.sketch_click(FIRST_AXIS);
    world.escape();
}
#[when("I choose Ellipse from the Circle dropdown")]
#[when("the user choose Ellipse from the Circle dropdown")]
#[when("I invoke Ellipse")]
#[when("the user invoke Ellipse")]
#[when("I open Circle dropdown and choose Ellipse")]
#[when("the user open Circle dropdown and choose Ellipse")]
fn invoke_ellipse(world: &mut ApplicationWorld) {
    choose_ellipse(world);
}
#[when("I select the fixture center")]
#[when("the user select the fixture center")]
fn select_center(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("the prompt requests the first axis point")]
fn asks_first_axis(world: &mut ApplicationWorld) {
    world.expect_status("Select first axis point");
}
#[when("I select the fixture first-axis point")]
#[when("the user select the fixture first-axis point")]
#[when("I select a distinct first-axis point")]
#[when("the user select a distinct first-axis point")]
fn select_first_axis(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST_AXIS);
}
#[then("the prompt requests a point on the ellipse")]
#[then("the prompt advances to Select point on ellipse")]
fn asks_perimeter(world: &mut ApplicationWorld) {
    world.expect_status("Select point on ellipse");
}
#[given("the editable recorded fixture has clear canvas for a rotated ellipse")]
fn empty_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[when("I select the fixture point on the ellipse")]
#[when("the user select the fixture point on the ellipse")]
fn select_perimeter(world: &mut ApplicationWorld) {
    world.sketch_click(MINOR_AXIS);
    world.hover(CLEAR_POINTER);
}
#[then("the MCAD Software is Ready")]
fn ready(world: &mut ApplicationWorld) {
    world.expect_ready();
}
#[then("the oval disappears")]
fn oval_removed(world: &mut ApplicationWorld) {
    expect_no_oval_visible(world);
}
fn expect_no_oval_visible(world: &mut ApplicationWorld) {
    assert_eq!(count_ellipses(world), 0, "ellipse count");
    for position in BOUNDARY {
        world.expect_segment_absent(position, position);
    }
}
#[then("the rotated oval returns")]
fn oval_restored(world: &mut ApplicationWorld) {
    expect_oval_visible(world);
}
#[when("I invoke Ellipse and select the fixture center")]
#[when("the user invoke Ellipse and select the fixture center")]
fn invoke_and_center(world: &mut ApplicationWorld) {
    choose_ellipse(world);
    world.sketch_click(CENTER);
}
#[when("I select the same location again as its first-axis point")]
#[when("the user select the same location again as its first-axis point")]
fn coincident_first_axis(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}
#[then("the prompt remains Select first axis point without an error dialog")]
fn first_axis_not_advanced(world: &mut ApplicationWorld) {
    world.expect_status("Select first axis point");
    assert_eq!(count_ellipses(world), 0, "ellipse count");
    world.expect_no_text("Error");
}
#[path = "dimension_lifecycle.rs"]
mod dimension_lifecycle;
