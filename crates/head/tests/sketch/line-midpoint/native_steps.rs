// SPDX-License-Identifier: GPL-2.0-only
//! Midpoint Line observations driven through the sketch viewport.
use super::ApplicationWorld;
use cucumber::{then, when};
#[path = "precise_steps.rs"]
mod precise_steps;

#[when("the user select the sketch origin as the midpoint")]
#[when("the user select the sketch origin as its midpoint")]
#[when("the user select the sketch origin again for the endpoint")]
fn click_sketch_origin(world: &mut ApplicationWorld) {
    let origin = world.sketch_pixel_of([0., 0.]);
    world.sketch_click(origin);
}

#[then("the prompt requests the end of the line")]
#[then("Select end of line remains active")]
fn asks_for_endpoint(world: &mut ApplicationWorld) {
    world.expect_status("Select end of line");
}

#[then("no segment is added")]
fn no_segment_added(world: &mut ApplicationWorld) {
    assert!(world.edited_geometry().lines().is_empty());
}

#[then("the displayed length is 0.000 mm")]
fn zero_length_displayed(world: &mut ApplicationWorld) {
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("0.000mm"), "expected zero length in {text:?}");
}

#[then("no error dialog or new segment appears")]
fn no_error_or_segment(world: &mut ApplicationWorld) {
    assert!(super::super::super::diagnostics::window_geometry("Warning").is_none());
    no_segment_added(world);
}
