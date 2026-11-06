// SPDX-License-Identifier: GPL-2.0-only
//! A second click beside the start: the recording's "same coordinate" made a short segment. Its
//! capture coordinates were scaled, so here the click lands three real pixels on, past what the
//! command treats as a double-click and as near the recorded 0.678 mm as the view's pixels allow.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};

const START: [i32; 2] = [620, 400];
const BESIDE: [i32; 2] = [623, 400];
/// Off every seeded direction, so the connected segment infers no relation and adds two.
const END: [i32; 2] = [840, 367];
const RECORDED_LENGTH: f64 = 0.678;
/// One view pixel is 0.222 mm here, so the display can come this close and no closer.
const DISPLAY_TOLERANCE: f64 = 0.02;

fn displayed_length(text: &str) -> f64 {
    let field = text
        .split(' ')
        .find(|field| field.ends_with("mm"))
        .unwrap_or_else(|| panic!("expected a length in the fields {text:?}"));
    field
        .trim_end_matches("mm")
        .parse()
        .unwrap_or_else(|_| panic!("expected a number in the field {field:?}"))
}
fn undo_and_redo_counts(world: &mut ApplicationWorld, step: fn(&mut ApplicationWorld)) {
    world.observed_counts.clear();
    for _ in 0..2 {
        step(world);
        world.observed_counts.push(world.status_dimensions_needed());
    }
}
#[given("an editable sketch with the captured geometry is active")]
fn captured_geometry(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(215);
}
#[when("I start Line and select a start point")]
fn start_line(world: &mut ApplicationWorld) {
    world.select_line_tool();
    world.sketch_click(START);
}
#[when("I click the same screen coordinate again")]
fn click_again(world: &mut ApplicationWorld) {
    world.hover(BESIDE);
    world.observed_rows = Some(world.window_text("##sketch-hud"));
    world.sketch_click(BESIDE);
}
#[then("the dynamic length display shows approximately 0.678 mm")]
fn length_shown(world: &mut ApplicationWorld) {
    let shown = displayed_length(world.observed_rows.as_deref().unwrap_or_default());
    assert!(
        (shown - RECORDED_LENGTH).abs() <= DISPLAY_TOLERANCE,
        "expected about {RECORDED_LENGTH} mm, shown {shown} mm"
    );
}
#[when("I select a distinct endpoint and exit Line")]
fn endpoint_and_exit(world: &mut ApplicationWorld) {
    world.sketch_click(END);
    world.escape();
}
#[then("the longer connected segment remains visible")]
#[then("the longer upper segment returns")]
fn longer_segment(world: &mut ApplicationWorld) {
    world.expect_segment_drawn(BESIDE, END);
}
// line-07
#[given(
    "the recorded sketch fixture is active after the Line start point was clicked twice at the same screen coordinate"
)]
fn clicked_twice(world: &mut ApplicationWorld) {
    captured_geometry(world);
    start_line(world);
    click_again(world);
}
#[given("a distinct endpoint was then selected")]
fn endpoint_selected(world: &mut ApplicationWorld) {
    world.sketch_click(END);
}
#[when("I exit creation and dismiss the transient point selector")]
fn exit_creation(world: &mut ApplicationWorld) {
    world.escape();
}
#[then("the status reports Ready and 220 dimensions needed")]
fn ready_220(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(220);
}
#[when("I press Undo twice, observing the state after each key press")]
fn undo_twice(world: &mut ApplicationWorld) {
    undo_and_redo_counts(world, ApplicationWorld::undo);
}
#[then("the reported counts are 218 and 215 respectively")]
fn counts_after_undo(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_counts, [218, 215], "counts after each undo");
}
#[when("I press Redo twice, observing the state after each key press")]
fn redo_twice(world: &mut ApplicationWorld) {
    undo_and_redo_counts(world, ApplicationWorld::redo);
}
#[then("the reported counts are 218 and 220 respectively")]
fn counts_after_redo(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_counts, [218, 220], "counts after each redo");
}
