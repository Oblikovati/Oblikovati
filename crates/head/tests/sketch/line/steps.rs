// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/line`: a segment commits when accepted; the command keeps the chain going.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

/// Two viewport points that lie clear of the origin axes and every other fixture.
const SEGMENT: [[i32; 2]; 2] = [[620, 300], [840, 420]];

#[given("the MCAD Software is running")]
fn software_running(world: &mut ApplicationWorld) {
    world.expect_text("Ready");
}
#[given("an empty editable planar sketch is active")]
fn empty_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.expect_ready();
}
#[when("I select Line and click two distinct points")]
fn line_two_points(world: &mut ApplicationWorld) {
    world.select_line_tool();
    for point in SEGMENT {
        world.sketch_click(point);
    }
}
#[then("one segment joins the points and the command offers a connected next segment")]
fn one_segment_and_next(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(SEGMENT[0], SEGMENT[1]);
    world.expect_status("Select end of line");
    assert!(
        !world.window_text("##sketch-hud").is_empty(),
        "expected the fields of the next segment"
    );
}
#[then("the command ends and the visible segment remains")]
fn command_ended(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_segment_drawn(SEGMENT[0], SEGMENT[1]);
}

#[path = "chain.rs"]
mod chain;
#[path = "inspect.rs"]
mod inspect;
#[path = "numeric.rs"]
mod numeric;
#[path = "recorded.rs"]
mod recorded;
#[path = "repeated_click.rs"]
mod repeated_click;
#[path = "retained_arc.rs"]
mod retained_arc;
