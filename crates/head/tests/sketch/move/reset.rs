// SPDX-License-Identifier: GPL-2.0-only
//! Reset the Move triad without moving its selected ellipse.
use super::{ApplicationWorld, diagnostics, move_precise};
use cucumber::{given, then, when};

#[given("the ellipse is selected in precise Base Point mode")]
fn select_precise_ellipse(world: &mut ApplicationWorld) {
    move_precise::select_ellipse(world);
    let mode = world.run_in_window("Precise Input", "Mode").center();
    world.click(diagnostics::pointer_pixels(mode));
    world.click_text("Distance and X axis angle");
}

#[given("the triad remains at its prior visible position")]
fn reposition_triad(world: &mut ApplicationWorld) {
    let origin = world.sketch_pixel_of([0., 0.]);
    world.click_text("Reposition Triad");
    world.sketch_click(origin);
    world.sketch_click([1120, 650]);
    let position = world.app().owner.precise_triad_position().unwrap().1;
    assert_ne!(position, [0., 0.]);
    let visible = world.sketch_pixel_of(position);
    world.expect_segment_drawn(visible, [visible[0] + 24, visible[1]]);
}

#[when("the user clicks Reset to Origin")]
fn reset_triad(world: &mut ApplicationWorld) {
    world.click_text("Reset to Origin");
}

#[then("the visible triad moves to the sketch axes intersection")]
fn triad_at_origin(world: &mut ApplicationWorld) {
    assert_eq!(
        world.app().owner.precise_triad_position().unwrap().1,
        [0., 0.]
    );
    let origin = world.sketch_pixel_of([0., 0.]);
    world.expect_segment_drawn(origin, [origin[0] + 24, origin[1]]);
}

#[then("the selected ellipse remains at its original position")]
fn ellipse_unchanged(world: &mut ApplicationWorld) {
    world.expect_edited_sketch_retained();
    world.expect_segment_drawn(move_precise::EDGE, move_precise::EDGE);
}

#[then("the Distance and Angle fields appear blank")]
fn blank_fields(world: &mut ApplicationWorld) {
    let text = world.window_text("Precise Input");
    assert!(
        text.contains("Distance") && text.contains("Angle"),
        "{text:?}"
    );
    assert!(
        !text.chars().any(|value| value.is_ascii_digit()),
        "{text:?}"
    );
}

#[then("Move and its toolbar close without an accepted endpoint")]
fn cancel_without_placement(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_move.is_none());
    assert!(diagnostics::window_geometry("Precise Input").is_none());
    world.expect_ready();
    world.expect_edited_sketch_retained();
}
